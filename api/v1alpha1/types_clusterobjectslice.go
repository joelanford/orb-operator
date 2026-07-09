package v1alpha1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// ClusterObjectSlice is a placeholder resource reserved for future use as a
// mechanism to split large ClusterObjectSet phase content across
// multiple objects.
//
// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Cluster,shortName=cosl
type ClusterObjectSlice struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
}

// ClusterObjectSliceList is a list of ClusterObjectSlice resources.
//
// +kubebuilder:object:root=true
type ClusterObjectSliceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	// items is the list of ClusterObjectSlice resources.
	// +required
	Items []ClusterObjectSlice `json:"items"`
}
