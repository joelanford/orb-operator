package v1alpha1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster,shortName=cos
// +kubebuilder:printcolumn:name="Availability",type=string,JSONPath=`.status.conditions[?(@.type=="Available")].reason`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
type ClusterObjectSet struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ClusterObjectSetSpec   `json:"spec,omitempty"`
	Status ClusterObjectSetStatus `json:"status,omitempty"`
}

type ClusterObjectSetSpec struct {
	// +kubebuilder:validation:Minimum=0
	RevisionHistoryLimit *int32                   `json:"revisionHistoryLimit,omitempty"`
	Template             ClusterObjectSetTemplate `json:"template"`
}

type ClusterObjectSetTemplate struct {
	Metadata ClusterObjectSetTemplateMetadata `json:"metadata,omitempty"`
	Spec     ClusterObjectSetTemplateSpec     `json:"spec"`
}

type ClusterObjectSetTemplateMetadata struct {
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
}

type ClusterObjectSetStatus struct {
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

type ClusterObjectSetTemplateSpec struct {
	CollisionProtection *CollisionProtection `json:"collisionProtection,omitempty"`
	// +kubebuilder:validation:MinItems=1
	// +kubebuilder:validation:MaxItems=20
	Phases []Phase `json:"phases"`
}

// +kubebuilder:object:root=true
type ClusterObjectSetList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []ClusterObjectSet `json:"items"`
}
