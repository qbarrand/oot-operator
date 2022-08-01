package job

import (
	"fmt"
	"strings"

	ootov1alpha1 "github.com/qbarrand/oot-operator/api/v1alpha1"
	"github.com/qbarrand/oot-operator/internal/build"
	batchv1 "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

//go:generate mockgen -source=maker.go -package=job -destination=mock_maker.go

type signer struct {
	helper build.Helper
	scheme *runtime.Scheme
}

func NewSigner(helper build.Helper, scheme *runtime.Scheme) Maker {
	return &signer{helper: helper, scheme: scheme}
}

func (m *signer) PullOptions(km ootov1alpha1.KernelMapping) ootov1alpha1.PullOptions{
	return km.Sign.Pull
}

func (m *signer) MakeJob(mod ootov1alpha1.Module,  km *ootov1alpha1.KernelMapping, targetKernel, containerImage string) (*batchv1.Job, error) {
	 //m *ootov1alpha1.KernelMapping 
	 signConfig := km.Sign
	 //*ootov1alpha1.Sign

	args := []string{"-signedimage", containerImage}

	args = append(args, "-unsignedimage", signConfig.UnsignedImage)
	args = append(args, "-pullsecret", "/docker_config/config.json") 
	args = append(args, "-key", "/signingkey/key.priv") 
	args = append(args, "-cert", "/signingcert/public.der")
	args = append(args, "-filestosign", strings.Join(signConfig.FilesToSign, ":"))


	volumes := []v1.Volume{}
	volumeMounts := []v1.VolumeMount{}

	if signConfig.ImagePullSecret != nil {
		volumes = append(volumes, m.makeImagePullSecretVolume(signConfig.ImagePullSecret))
		volumeMounts = append(volumeMounts, m.makeImagePullSecretVolumeMount(signConfig.ImagePullSecret))
	}

	volumes = append(volumes, m.makeImageSigningSecretVolume( signConfig.KeySecret, "key", "key.priv"))
	volumes = append(volumes,  m.makeImageSigningSecretVolume( signConfig.CertSecret, "cert", "public.der"))


	volumeMounts = append(volumeMounts, m.makeImageSigningSecretVolumeMount(signConfig.CertSecret, "/signingcert"))
	volumeMounts = append(volumeMounts, m.makeImageSigningSecretVolumeMount(signConfig.KeySecret, "/signingkey"))

	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: mod.Name + "-sign-",
			Namespace:    mod.Namespace,
			Labels:       labels(mod, targetKernel),
		},
		Spec: batchv1.JobSpec{
			Completions: pointer.Int32(1),
			Template: v1.PodTemplateSpec{
				//ObjectMeta: metav1.ObjectMeta{
				//	Annotations: map[string]string{"Dockerfile": buildConfig.Dockerfile},
				//},
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						{
							Name:         "signimage",
							Image:        "quay.io/chrisp262/kmod-signer:latest",
							Args:         args,	
							VolumeMounts: volumeMounts,
						},
					},
					RestartPolicy: v1.RestartPolicyOnFailure,
					Volumes:       volumes,
				},
			},
		},
	}

	if err := controllerutil.SetControllerReference(&mod, job, m.scheme); err != nil {
		return nil, fmt.Errorf("could not set the owner reference: %v", err)
	}

	return job, nil
}


func (m *signer) makeImageSigningSecretVolume(secretRef *v1.LocalObjectReference, key string, path string) v1.Volume {

	return v1.Volume{
		Name: volumeNameFromSecretRef(*secretRef),
		VolumeSource: v1.VolumeSource{
			Secret: &v1.SecretVolumeSource{
				SecretName: secretRef.Name,
				Items: []v1.KeyToPath{
					{
						Key:  key,
						Path:  path,
					},
				},
			},
		},
	}
}

func (m *signer) makeImageSigningSecretVolumeMount(secretRef *v1.LocalObjectReference, mountpoint string) v1.VolumeMount {

	return v1.VolumeMount{
		Name:      volumeNameFromSecretRef(*secretRef),
		ReadOnly:  true,
		MountPath: mountpoint,
	}
}



func (m *signer) makeImagePullSecretVolume(secretRef *v1.LocalObjectReference) v1.Volume {

	if secretRef == nil {
		return v1.Volume{}
	}

	return v1.Volume{
		Name: volumeNameFromSecretRef(*secretRef),
		VolumeSource: v1.VolumeSource{
			Secret: &v1.SecretVolumeSource{
				SecretName: secretRef.Name,
				Items: []v1.KeyToPath{
					{
						Key:  v1.DockerConfigJsonKey,
						Path: "config.json",
					},
				},
			},
		},
	}
}

func (m *signer) makeImagePullSecretVolumeMount(secretRef *v1.LocalObjectReference) v1.VolumeMount {

	if secretRef == nil {
		return v1.VolumeMount{}
	}

	return v1.VolumeMount{
		Name:      volumeNameFromSecretRef(*secretRef),
		ReadOnly:  true,
		MountPath: "/docker_config",
	}
}


