package job


import (
	ootov1alpha1 "github.com/qbarrand/oot-operator/api/v1alpha1"
	batchv1 "k8s.io/api/batch/v1"
)


type Job interface {
	MakeJob(mod ootov1alpha1.Module, m *ootov1alpha1.KernelMapping, targetKernel string) (*batchv1.Job, error)
	PullOptions(km ootov1alpha1.KernelMapping) ootov1alpha1.PullOptions
	ShouldRun(mod *ootov1alpha1.Module, km *ootov1alpha1.KernelMapping) bool
	GetName() string
	GetOutputImage(mod ootov1alpha1.Module, km *ootov1alpha1.KernelMapping) (string,error)
}
