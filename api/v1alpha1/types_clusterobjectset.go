package v1alpha1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster,shortName=cos
type ClusterObjectSet struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ClusterObjectSetSpec   `json:"spec,omitempty"`
	Status ClusterObjectSetStatus `json:"status,omitempty"`
}

type ClusterObjectSetSpec struct{}

type ClusterObjectSetStatus struct{}

// +kubebuilder:object:root=true
type ClusterObjectSetList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []ClusterObjectSet `json:"items"`
}
