/*
Copyright 2022.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// BuildArg represents a build argument used when building a container image.
type BuildArg struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type PullOptions struct {
	// If Insecure is true, images can be pulled from an insecure (plain HTTP) registry.
	// +optional
	Insecure bool `json:"insecure,omitempty"`

	// If InsecureSkipTLSVerify, the operator will accept any certificate provided by the registry.
	// +optional
	InsecureSkipTLSVerify bool `json:"insecureSkipTLSVerify,omitempty"`
}

type PushOptions struct {
	// If Insecure is true, built images can be pushed to an insecure (plain HTTP) registry.
	// +optional
	Insecure bool `json:"insecure,omitempty"`

	// If InsecureSkipTLSVerify, the operator will accept any certificate provided by the registry.
	// +optional
	InsecureSkipTLSVerify bool `json:"insecureSkipTLSVerify,omitempty"`
}

type Build struct {
	// BuildArgs is an array of build variables that are provided to the image building backend.
	// +optional
	BuildArgs []BuildArg `json:"buildArgs,omitempty"`

	Dockerfile string `json:"dockerfile"`

	// Pull contains settings determining how to check if the DriverContainer image already exists.
	// +optional
	Pull PullOptions `json:"pull"`

	// Push contains settings determining how to push a built DriverContainer image.
	// +optional
	Push PushOptions `json:"push,omitempty"`

	// Secrets is an optional list of secrets to be made available to the build system.
	// +optional
	Secrets []v1.LocalObjectReference `json:"secrets"`
}

// KernelMapping pairs kernel versions with a DriverContainer image.
// Kernel versions can be matched literally or using a regular expression.
type KernelMapping struct {
	// Build enables in-cluster builds for this mapping and allows overriding the Module's build settings.
	// +optional
	Build *Build `json:"build,omitempty"`

	// ContainerImage is the name of the DriverContainer image that should be used to deploy the module.
	// +optional
	ContainerImage string `json:"containerImage,omitempty"`

	// Literal defines a literal target kernel version to be matched exactly against node kernels.
	// +optional
	Literal string `json:"literal,omitempty"`

	// Regexp is a regular expression to be match against node kernels.
	// +optional
	Regexp string `json:"regexp,omitempty"`
}

type DevicePluginSpec struct {
	Container v1.Container `json:"container"`

	// ServiceAccountName is the name of the ServiceAccount to use to run this pod.
	// More info: https://kubernetes.io/docs/tasks/configure-pod-container/configure-service-account/
	// +optional
	ServiceAccountName string `json:"serviceAccountName,omitempty"`
}

type DriverContainerContainerSpec struct {
	// Build are top-level instructions to build a DriverContainer image.
	// +optional
	Build *Build `json:"build,omitempty"`

	// Entrypoint array. Not executed within a shell.
	// The container image's ENTRYPOINT is used if this is not provided.
	// Variable references $(VAR_NAME) are expanded using the container's environment. If a variable
	// cannot be resolved, the reference in the input string will be unchanged. Double $$ are reduced
	// to a single $, which allows for escaping the $(VAR_NAME) syntax: i.e. "$$(VAR_NAME)" will
	// produce the string literal "$(VAR_NAME)". Escaped references will never be expanded, regardless
	// of whether the variable exists or not. Cannot be updated.
	// More info: https://kubernetes.io/docs/tasks/inject-data-application/define-command-argument-container/#running-a-command-in-a-shell
	// +optional
	// +kubebuilder:default={"sleep", "infinity"}
	Command []string `json:"command,omitempty"`

	// ModprobeArgs is a list of arguments to be passed to modprobe when loading the container.
	// +optional
	ModprobeArgs []string `json:"modprobeArgs,omitempty"`

	// SecurityContext defines the security options the container should be run with.
	// If set, the fields of SecurityContext override the equivalent fields of PodSecurityContext.
	// More info: https://kubernetes.io/docs/tasks/configure-pod-container/security-context/
	// +optional
	SecurityContext *v1.SecurityContext `json:"securityContext,omitempty"`
}

type DriverContainerSpec struct {
	Container DriverContainerContainerSpec `json:"container"`

	// ServiceAccountName is the name of the ServiceAccount to use to run this pod.
	// More info: https://kubernetes.io/docs/tasks/configure-pod-container/configure-service-account/
	// +optional
	ServiceAccountName string `json:"serviceAccountName,omitempty"`
}

// ModuleSpec describes how the OOT operator should deploy a Module on those nodes that need it.
type ModuleSpec struct {
	// +optional

	// AdditionalVolumes is a list of volumes that will be attached to the DriverContainer / DevicePlugin pod,
	// in addition to the default ones.
	AdditionalVolumes []v1.Volume `json:"additionalVolumes"`

	// DevicePlugin allows overriding some properties of the container that deploys the device plugin on the node.
	// Name is ignored and is set automatically by the OOT Operator.
	// +optional
	DevicePlugin *DevicePluginSpec `json:"devicePlugin"`

	// DriverContainer allows overriding some properties of the container that deploys the driver on the node.
	// Name and image are ignored and are set automatically by the OOT Operator.
	DriverContainer DriverContainerSpec `json:"driverContainer"`

	ImageRepoSecret v1.LocalObjectReference `json:"imageRepoSecret,omitempty"`

	// KernelMappings is a list of kernel mappings.
	// When a node's labels match Selector, then the OOT Operator will look for the first mapping that matches its
	// kernel version, and use the corresponding container image to run the DriverContainer.
	// +kubebuilder:validation:MinItems=1
	KernelMappings []KernelMapping `json:"kernelMappings"`

	// Selector describes on which nodes the Module should be loaded.
	Selector map[string]string `json:"selector"`
}

// ModuleStatus defines the observed state of Module.
type ModuleStatus struct {
	// Conditions is a list of conditions representing the Module's current state.
	Conditions []metav1.Condition `json:"conditions"`
}

//+kubebuilder:object:root=true
//+kubebuilder:resource:scope=Namespaced
//+kubebuilder:subresource:status

// Module is the Schema for the modules API
type Module struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ModuleSpec   `json:"spec,omitempty"`
	Status ModuleStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// ModuleList contains a list of Module
type ModuleList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Module `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Module{}, &ModuleList{})
}
