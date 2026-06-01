package v1alpha1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster,shortName=cosr
type ClusterObjectSetRevision struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ClusterObjectSetRevisionSpec   `json:"spec,omitempty"`
	Status ClusterObjectSetRevisionStatus `json:"status,omitempty"`
}

type ClusterObjectSetRevisionSpec struct{}

type ClusterObjectSetRevisionStatus struct{}

// +kubebuilder:object:root=true
type ClusterObjectSetRevisionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []ClusterObjectSetRevision `json:"items"`
}
