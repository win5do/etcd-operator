/*
Copyright 2021.

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

package v1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// EtcdSpec defines the desired state of Etcd
type EtcdSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	Members      int    `json:"members,omitempty"`
	ExternalHost string `json:"externalHost,omitempty"`

	// quota 配额
	Cpu              string `json:"cpu,omitempty"`
	Memory           string `json:"memory,omitempty"`
	Storage          string `json:"storage,omitempty"`
	StorageClassName string `json:"storageClassName,omitempty"`

	Image            string                        `json:"image,omitempty"`
	ImagePullPolicy  corev1.PullPolicy             `json:"imagePullPolicy,omitempty" protobuf:"bytes,14,opt,name=imagePullPolicy,casttype=PullPolicy"`
	ImagePullSecrets []corev1.LocalObjectReference `json:"imagePullSecrets,omitempty" patchStrategy:"merge" patchMergeKey:"name" protobuf:"bytes,15,rep,name=imagePullSecrets"`

	ServiceAccountName string `json:"serviceAccountName,omitempty" protobuf:"bytes,8,opt,name=serviceAccountName"`

	Env []corev1.EnvVar `json:"env,omitempty"`

	PodSpec PodSpec `json:"podSpec,omitempty"`
}

// EtcdStatus defines the observed state of Etcd
type EtcdStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	Status      NodeStatus         `json:"status"`
	ConnectAddr string             `json:"connectAddr,omitempty"`
	Conditions  []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="status",type=string,JSONPath=`.status.status`
// +kubebuilder:printcolumn:name="connectAddr",type=string,JSONPath=`.status.connectAddr`

// Etcd is the Schema for the etcds API
type Etcd struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   EtcdSpec   `json:"spec,omitempty"`
	Status EtcdStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// EtcdList contains a list of Etcd
type EtcdList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Etcd `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Etcd{}, &EtcdList{})
}

// copy from corev1.PodSpec
type PodSpec struct {
	HostAliases     []corev1.HostAlias         `json:"hostAliases,omitempty" patchStrategy:"merge" patchMergeKey:"ip" protobuf:"bytes,23,rep,name=hostAliases"`
	RestartPolicy   corev1.RestartPolicy       `json:"restartPolicy,omitempty" protobuf:"bytes,3,opt,name=restartPolicy,casttype=RestartPolicy"`
	NodeSelector    map[string]string          `json:"nodeSelector,omitempty" protobuf:"bytes,7,rep,name=nodeSelector"`
	SecurityContext *corev1.PodSecurityContext `json:"securityContext,omitempty" protobuf:"bytes,14,opt,name=securityContext"`
	Affinity        *corev1.Affinity           `json:"affinity,omitempty" protobuf:"bytes,18,opt,name=affinity"`
	Tolerations     []corev1.Toleration        `json:"tolerations,omitempty" protobuf:"bytes,22,opt,name=tolerations"`
}

type NodeStatus string

const (
	StatusReady        NodeStatus = "Ready"
	StatusPartialReady NodeStatus = "PartialReady"
	StatusFailed       NodeStatus = "Failed"
	StatusUnknown      NodeStatus = "Unknown"
)
