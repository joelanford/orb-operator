package v1alpha1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// ClusterObjectDeployment declares a set of Kubernetes objects that should be applied
// to the cluster and kept in the desired state. The controller creates
// ClusterObjectSetRevision resources to track each unique template snapshot and
// manages their lifecycle automatically.
//
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster,shortName=cod
// +kubebuilder:printcolumn:name="Availability",type=string,JSONPath=`.status.conditions[?(@.type=="Available")].reason`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
type ClusterObjectDeployment struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// spec defines the desired set of objects and their lifecycle configuration.
	// +required
	Spec ClusterObjectDeploymentSpec `json:"spec"`

	// status reports the observed state of the ClusterObjectDeployment, including
	// aggregate availability and the state of active revisions.
	// +optional
	Status ClusterObjectDeploymentStatus `json:"status,omitempty"`
}

// ClusterObjectDeploymentSpec defines the desired state of a ClusterObjectDeployment.
type ClusterObjectDeploymentSpec struct {
	// revisionHistoryLimit is the maximum number of archived
	// ClusterObjectSetRevision resources to retain. Older archived revisions
	// beyond this limit are garbage collected by the controller. When omitted,
	// the platform chooses a reasonable default, which is subject to change over
	// time. The current default is 10. Set to 0 to disable revision history
	// entirely.
	// +kubebuilder:validation:Minimum=0
	// +optional
	RevisionHistoryLimit *int32 `json:"revisionHistoryLimit,omitempty"`

	// template defines the ClusterObjectSetRevision that the controller will
	// create whenever the template content changes.
	// +required
	Template ClusterObjectDeploymentTemplate `json:"template"`
}

// ClusterObjectDeploymentTemplate defines the template used to stamp out
// ClusterObjectSetRevision resources.
type ClusterObjectDeploymentTemplate struct {
	// metadata contains labels and annotations that are propagated to each
	// ClusterObjectSetRevision created from this template.
	// +optional
	Metadata ClusterObjectDeploymentTemplateMetadata `json:"metadata,omitempty"`

	// spec defines the phases, objects, and configuration for each revision
	// created from this template.
	// +required
	Spec ClusterObjectDeploymentTemplateSpec `json:"spec"`
}

// ClusterObjectDeploymentTemplateMetadata contains labels and annotations propagated
// to revisions created from the template. Labels and annotations must conform
// to the standard Kubernetes metadata format. Annotation values are bounded to
// 256 KiB each.
//
// +kubebuilder:validation:XValidation:rule="!has(self.annotations) || self.annotations.all(k, self.annotations[k].size() <= 262144)",message="annotation values must be 256 KiB or less"
type ClusterObjectDeploymentTemplateMetadata struct {
	// labels is a set of key/value pairs propagated to each revision's metadata.
	// Keys and values must conform to the Kubernetes label format. A maximum of
	// 32 labels may be specified.
	// +kubebuilder:validation:MaxProperties=32
	// +optional
	Labels map[string]string `json:"labels,omitempty"`

	// annotations is a set of key/value pairs propagated to each revision's
	// metadata. Keys must conform to the Kubernetes annotation key format.
	// Values are free-form strings up to 256 KiB each. A maximum of 32
	// annotations may be specified.
	// +kubebuilder:validation:MaxProperties=32
	// +optional
	Annotations map[string]string `json:"annotations,omitempty"`
}

// ClusterObjectDeploymentStatus reports the observed state of a ClusterObjectDeployment.
type ClusterObjectDeploymentStatus struct {
	// conditions represent the latest available observations of the
	// ClusterObjectDeployment's state. The "Available" condition indicates whether the
	// active revision's managed objects satisfy their assertions.
	// +listType=map
	// +listMapKey=type
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// activeRevisions holds the currently active (non-archived)
	// ClusterObjectSetRevision resources, including any revision just created
	// but not yet visible in the informer cache.
	// +listType=map
	// +listMapKey=name
	// +optional
	ActiveRevisions []ClusterObjectSetRevisionStatusSummary `json:"activeRevisions,omitempty"`
}

// ClusterObjectSetRevisionStatusSummary summarizes the state of a single active
// ClusterObjectSetRevision.
type ClusterObjectSetRevisionStatusSummary struct {
	// name is the metadata.name of the ClusterObjectSetRevision resource.
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:Required
	// +required
	Name string `json:"name"`

	// conditions reflects the Available condition of the revision.
	// +listType=map
	// +listMapKey=type
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// ClusterObjectDeploymentTemplateSpec defines the phases and collision protection
// settings that are embedded in each revision created from the template.
type ClusterObjectDeploymentTemplateSpec struct {
	// collisionProtection sets the default collision protection for all phases
	// and objects in the revision. Individual phases and objects may override
	// this setting. When omitted, the platform chooses a reasonable default,
	// which is subject to change over time. The current default is "Prevent".
	// Allowed values are "Prevent", "IfNoController", and "None".
	// +optional
	CollisionProtection *CollisionProtection `json:"collisionProtection,omitempty"`

	// phases is the ordered list of phases that define the managed objects. The
	// list must contain between 1 and 20 phases. Phases are applied
	// sequentially: the controller does not advance to the next phase until all
	// objects in the current phase satisfy their assertions.
	// +kubebuilder:validation:MinItems=1
	// +kubebuilder:validation:MaxItems=20
	// +required
	Phases []Phase `json:"phases"`
}

// ClusterObjectDeploymentList is a list of ClusterObjectDeployment resources.
//
// +kubebuilder:object:root=true
type ClusterObjectDeploymentList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	// items is the list of ClusterObjectDeployment resources.
	// +required
	Items []ClusterObjectDeployment `json:"items"`
}
