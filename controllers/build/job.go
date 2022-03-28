package build

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/remote/transport"
	ootov1beta1 "github.com/qbarrand/oot-operator/api/v1beta1"
	"github.com/qbarrand/oot-operator/controllers/constants"
	batchv1 "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

var ErrNoMatchingBuild = errors.New("no matching build")

type jobManager struct {
	client    client.Client
	namespace string
	scheme    *runtime.Scheme
}

func NewJobManager(client client.Client, namespace string, scheme *runtime.Scheme) Manager {
	return &jobManager{
		client:    client,
		namespace: namespace,
		scheme:    scheme,
	}
}

func (jbm *jobManager) GetJobLabels(mod ootov1beta1.Module, targetKernel string) map[string]string {
	return map[string]string{
		constants.ModuleNameLabel:    mod.Name,
		constants.TargetKernelTarget: targetKernel,
	}
}

func (jbm *jobManager) GetJob(ctx context.Context, mod ootov1beta1.Module, targetKernel string) (*batchv1.Job, error) {
	jobList := batchv1.JobList{}

	opts := []client.ListOption{
		client.MatchingLabels(jbm.GetJobLabels(mod, targetKernel)),
		client.InNamespace(jbm.namespace),
	}

	if err := jbm.client.List(ctx, &jobList, opts...); err != nil {
		return nil, fmt.Errorf("could not list jobs: %v", err)
	}

	if n := len(jobList.Items); n == 0 {
		return nil, ErrNoMatchingBuild
	} else if n > 1 {
		return nil, fmt.Errorf("expected 0 or 1 job, got %d", n)
	}

	return &jobList.Items[0], nil
}

func (jbm *jobManager) MakeJob(mod ootov1beta1.Module, m ootov1beta1.KernelMapping, targetKernel string) (*batchv1.Job, error) {
	args := []string{
		"--destination", m.ContainerImage,
		"--build-arg", "KERNEL_VERSION=" + targetKernel,
	}

	if m.Build.Pull.Insecure {
		args = append(args, "--insecure-pull")
	}

	if m.Build.Push.Insecure {
		args = append(args, "--insecure")
	}

	var one int32 = 1

	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: mod.Name + "-build-",
			Namespace:    jbm.namespace,
			Labels:       jbm.GetJobLabels(mod, targetKernel),
		},
		Spec: batchv1.JobSpec{
			Completions: &one,
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{"Dockerfile": m.Build.Dockerfile},
				},
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						{
							Args:  args,
							Name:  "kaniko",
							Image: "gcr.io/kaniko-project/executor:latest",
							VolumeMounts: []v1.VolumeMount{
								{
									Name:      "dockerfile",
									ReadOnly:  true,
									MountPath: "/workspace",
								},
							},
						},
					},
					RestartPolicy: v1.RestartPolicyOnFailure,
					Volumes: []v1.Volume{
						{
							Name: "dockerfile",
							VolumeSource: v1.VolumeSource{
								DownwardAPI: &v1.DownwardAPIVolumeSource{
									Items: []v1.DownwardAPIVolumeFile{
										{
											Path:     "Dockerfile",
											FieldRef: &v1.ObjectFieldSelector{FieldPath: "metadata.annotations['Dockerfile']"},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	if err := controllerutil.SetControllerReference(&mod, job, jbm.scheme); err != nil {
		return nil, fmt.Errorf("could not set the owner reference: %v", err)
	}

	return job, nil
}

func (jbm *jobManager) IsImageAvailable(ctx context.Context, containerImage string, po ootov1beta1.PullOptions) (bool, error) {
	opts := make([]name.Option, 0)

	if po.Insecure {
		opts = append(opts, name.Insecure)
	}

	ref, err := name.ParseReference(containerImage, opts...)
	if err != nil {
		return false, fmt.Errorf("could not parse the container image name: %v", err)
	}

	if _, err = remote.Get(ref); err != nil {
		te := &transport.Error{}

		if errors.As(err, &te) && te.StatusCode == http.StatusNotFound {
			return false, nil
		}

		return false, fmt.Errorf("could not get image: %v", err)
	}

	return true, nil
}

func (jbm *jobManager) Sync(ctx context.Context, mod ootov1beta1.Module, m ootov1beta1.KernelMapping, targetKernel string) (Result, error) {
	logger := log.FromContext(ctx)

	imageAvailable, err := jbm.IsImageAvailable(ctx, m.ContainerImage, m.Build.Pull)
	if err != nil {
		return Result{}, fmt.Errorf("could not check if the image is available: %v", err)
	}

	if imageAvailable {
		return Result{Status: StatusCompleted, Requeue: false}, nil
	}

	logger.Info("Image not pull-able; building in-cluster")

	job, err := jbm.GetJob(ctx, mod, targetKernel)
	if err != nil {
		if !errors.Is(err, ErrNoMatchingBuild) {
			return Result{}, fmt.Errorf("error getting the build: %v", err)
		}

		logger.Info("Creating job")

		job, err = jbm.MakeJob(mod, m, targetKernel)
		if err != nil {
			return Result{}, fmt.Errorf("could not make Job: %v", err)
		}

		if err = jbm.client.Create(ctx, job); err != nil {
			return Result{}, fmt.Errorf("could not create Job: %v", err)
		}

		return Result{Status: StatusCreated, Requeue: true}, nil
	}

	logger.Info("Returning job status", "name", job.Name, "namespace", job.Namespace)

	switch {
	case job.Status.Succeeded == 1:
		return Result{Status: StatusCompleted, Requeue: true}, nil
	case job.Status.Active == 1:
		return Result{Status: StatusInProgress, Requeue: true}, nil
	case job.Status.Failed == 1:
		return Result{}, fmt.Errorf("job failed: %v", err)
	default:
		return Result{}, fmt.Errorf("unknown status: %v", job.Status)
	}
}
