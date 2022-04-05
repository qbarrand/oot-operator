package job

import (
	"fmt"

	ootov1beta1 "github.com/qbarrand/oot-operator/api/v1beta1"
	batchv1 "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

type Maker interface {
	MakeJob(mod ootov1beta1.Module, m ootov1beta1.KernelMapping, targetKernel string) (*batchv1.Job, error)
}

type maker struct {
	namespace string
	scheme    *runtime.Scheme
}

func NewMaker(namespace string, scheme *runtime.Scheme) Maker {
	return &maker{
		namespace: namespace,
		scheme:    scheme,
	}
}

func (m *maker) MakeJob(mod ootov1beta1.Module, km ootov1beta1.KernelMapping, targetKernel string) (*batchv1.Job, error) {
	args := []string{
		"--destination", km.ContainerImage,
		"--build-arg", "KERNEL_VERSION=" + targetKernel,
	}

	if km.Build.Pull.Insecure {
		args = append(args, "--insecure-pull")
	}

	if km.Build.Push.Insecure {
		args = append(args, "--insecure")
	}

	var one int32 = 1

	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: mod.Name + "-build-",
			Namespace:    m.namespace,
			Labels:       Labels(mod, targetKernel),
		},
		Spec: batchv1.JobSpec{
			Completions: &one,
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{"Dockerfile": km.Build.Dockerfile},
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

	if err := controllerutil.SetControllerReference(&mod, job, m.scheme); err != nil {
		return nil, fmt.Errorf("could not set the owner reference: %v", err)
	}

	return job, nil
}
