package job

import (
	"context"
	"errors"
	"fmt"

	ootov1alpha1 "github.com/qbarrand/oot-operator/api/v1alpha1"
	"github.com/qbarrand/oot-operator/internal/auth"
	"github.com/qbarrand/oot-operator/internal/jobmanager"
	"github.com/qbarrand/oot-operator/internal/constants"
	"github.com/qbarrand/oot-operator/internal/registry"
	batchv1 "k8s.io/api/batch/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

var errNoMatchingJob = errors.New("no matching job")

type jobManager struct {
	client   client.Client
	registry registry.Registry
	maker    Job
	helper   build.Helper
}

func NewJobManager(client client.Client, registry registry.Registry, maker Job, helper build.Helper) *jobManager {
	return &jobManager{
		client:   client,
		registry: registry,
		maker:    maker,
		helper:   helper,
	}
}

func labels(mod ootov1alpha1.Module, targetKernel string, jobname string) map[string]string {
	return map[string]string{
		constants.ModuleNameLabel:    mod.Name,
		constants.TargetKernelTarget: targetKernel,
		"ooto.sigs.k8s.io/build-stage": jobname,
	}
}

func (jbm *jobManager) GetName() string {
	return jbm.maker.GetName()
}

func (jbm *jobManager) ShouldRun(mod *ootov1alpha1.Module, km *ootov1alpha1.KernelMapping) bool{
	return jbm.maker.ShouldRun(mod, km)
}


func (jbm *jobManager) getJob(ctx context.Context, mod ootov1alpha1.Module, targetKernel string) (*batchv1.Job, error) {
	jobList := batchv1.JobList{}

	opts := []client.ListOption{
		client.MatchingLabels(labels(mod, targetKernel, jbm.GetName())),
		client.InNamespace(mod.Namespace),
	}

	if err := jbm.client.List(ctx, &jobList, opts...); err != nil {
		return nil, fmt.Errorf("could not list jobs: %v", err)
	}

	if n := len(jobList.Items); n == 0 {
		return nil, errNoMatchingJob
	} else if n > 1 {
		return nil, fmt.Errorf("expected 0 or 1 job, got %d", n)
	}

	return &jobList.Items[0], nil
}

func (jbm *jobManager) Sync(ctx context.Context, mod ootov1alpha1.Module, m ootov1alpha1.KernelMapping, targetKernel string) (build.Result, error) {
	logger := log.FromContext(ctx)

	var registryAuthGetter auth.RegistryAuthGetter

	if mod.Spec.ImagePullSecret != nil {
		namespacedName := types.NamespacedName{
			Name:      mod.Spec.ImagePullSecret.Name,
			Namespace: mod.Namespace,
		}
		registryAuthGetter = auth.NewRegistryAuthGetter(jbm.client, namespacedName)
	}

	containerimage, err := jbm.maker.GetOutputImage(mod, &m)
        if err != nil {
                return build.Result{}, err
        }

	pulloptions := jbm.maker.PullOptions(m)

	logger.Info("try to pull image", "img", containerimage, "for job", jbm.GetName() )
	imageAvailable, err := jbm.registry.ImageExists(ctx, containerimage, pulloptions, registryAuthGetter)
	if err != nil {
		return build.Result{}, fmt.Errorf("could not check if the image is available: %v", err)
	}

	if imageAvailable {
		return build.Result{Status: build.StatusCompleted, Requeue: false}, nil
	}

	logger.Info("Image not pull-able; building in-cluster")

	job, err := jbm.getJob(ctx, mod, targetKernel)
	if err != nil {
		if !errors.Is(err, errNoMatchingJob) {
			return build.Result{}, fmt.Errorf("error getting the job: %v", err)
		}

		logger.Info("Creating job")

		//job, err = jbm.maker.MakeJob(mod, buildConfig, targetKernel, m.ContainerImage)
		job, err = jbm.maker.MakeJob(mod, &m, targetKernel)
		if err != nil {
			return build.Result{}, fmt.Errorf("could not make Job: %v", err)
		}

		if err = jbm.client.Create(ctx, job); err != nil {
			return build.Result{}, fmt.Errorf("could not create Job: %v", err)
		}

		return build.Result{Status: build.StatusCreated, Requeue: true}, nil
	}

	logger.Info("Returning job status", "name", job.Name, "namespace", job.Namespace)

	switch {
	case job.Status.Succeeded == 1:
		return build.Result{Status: build.StatusCompleted}, nil
	case job.Status.Active == 1:
		return build.Result{Status: build.StatusInProgress, Requeue: true}, nil
	case job.Status.Failed == 1:
		return build.Result{}, fmt.Errorf("job failed: %v", err)
	default:
		return build.Result{}, fmt.Errorf("unknown status: %v", job.Status)
	}
}
