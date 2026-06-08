/*
Copyright 2026.

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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// BackupPolicySpec defines the desired state of BackupPolicy
type BackupPolicySpec struct {
	// Schedule is a cron expression, e.g. "0 2 * * *"
	Schedule string `json:"schedule"`

	// RetentionDays specifies how many days to keep backup files
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:default=7
	RetentionDays int32 `json:"retentionDays,omitempty"`

	// EtcdEndpoint is the etcd client URL, e.g. "https://127.0.0.1:2379"
	EtcdEndpoint string `json:"etcdEndpoint"`

	// EtcdSecretRef points to a Secret containing etcd client cert files (ca.crt, client.crt, client.key)
	EtcdSecretRef corev1.LocalObjectReference `json:"etcdSecretRef"`

	// StoragePVC is the name of an existing PVC where backup files will be stored
	StoragePVC string `json:"storagePVC"`

	// BackupImage is the container image that provides etcdctl; defaults to bitnami/etcd:latest
	// +optional
	BackupImage string `json:"backupImage,omitempty"`

	// NodeSelector controls which nodes the backup Pod can run on (typically master nodes)
	// +optional
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`

	// Tolerations to apply to the backup Pod
	// +optional
	Tolerations []corev1.Toleration `json:"tolerations,omitempty"`
}

// BackupPolicyStatus defines the observed state of BackupPolicy
type BackupPolicyStatus struct {
	// LastScheduleTime is the last time the CronJob was scheduled
	LastScheduleTime *metav1.Time `json:"lastScheduleTime,omitempty"`
	// ActiveCronJob points to the CronJob managed by this policy
	ActiveCronJob string `json:"activeCronJob,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

type BackupPolicy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              BackupPolicySpec   `json:"spec,omitempty"`
	Status            BackupPolicyStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
type BackupPolicyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []BackupPolicy `json:"items"`
}

func init() {
	SchemeBuilder.Register(&BackupPolicy{}, &BackupPolicyList{})
}
