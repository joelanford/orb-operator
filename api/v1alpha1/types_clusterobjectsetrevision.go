package v1alpha1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// ClusterObjectSetRevision represents an immutable snapshot of a set of
// Kubernetes objects to apply and manage on the cluster. Revisions within the
// same group form an ordered sequence.
//
// When a new revision is created, multiple revisions may be active
// simultaneously. During this transition the highest-numbered active revision
// reconciles its own objects and takes over ownership of objects shared with
// predecessor revisions. Objects that exist only in a predecessor remain under
// that predecessor's ownership until it is archived, at which point they are
// deleted.
//
// Deleting a revision without the orphan finalizer triggers a reverse-order
// teardown of its phases before the resource is removed.
//
// The group, revision number, phases, and collisionProtection fields are
// immutable after creation. Only lifecycleState may be updated.
//
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster,shortName=cosr
// +kubebuilder:printcolumn:name="Group",type=string,JSONPath=`.spec.group`
// +kubebuilder:printcolumn:name="Revision",type=integer,JSONPath=`.spec.revision`
// +kubebuilder:printcolumn:name="Available",type=string,JSONPath=`.status.conditions[?(@.type=="Available")].status`
// +kubebuilder:printcolumn:name="Lifecycle",type=string,JSONPath=`.spec.lifecycleState`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
type ClusterObjectSetRevision struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// spec defines the desired state of this revision, including the objects to
	// manage and their lifecycle configuration.
	// +required
	Spec ClusterObjectSetRevisionSpec `json:"spec"`

	// status reports the observed state of this revision, including availability
	// conditions.
	// +optional
	Status ClusterObjectSetRevisionStatus `json:"status,omitempty"`
}

// ClusterObjectSetRevisionList is a list of ClusterObjectSetRevision resources.
//
// +kubebuilder:object:root=true
type ClusterObjectSetRevisionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	// items is the list of ClusterObjectSetRevision resources.
	// +required
	Items []ClusterObjectSetRevision `json:"items"`
}

// ClusterObjectSetRevisionSpec defines the desired state of a
// ClusterObjectSetRevision. All fields except lifecycleState are immutable
// after creation.
//
// +kubebuilder:validation:XValidation:rule="self.group == oldSelf.group",message="group is immutable"
// +kubebuilder:validation:XValidation:rule="self.revision == oldSelf.revision",message="revision is immutable"
// +kubebuilder:validation:XValidation:rule="self.phases == oldSelf.phases",message="phases is immutable"
// +kubebuilder:validation:XValidation:rule="!has(oldSelf.collisionProtection) && !has(self.collisionProtection) || has(oldSelf.collisionProtection) && has(self.collisionProtection) && self.collisionProtection == oldSelf.collisionProtection",message="collisionProtection is immutable"
// +kubebuilder:validation:XValidation:rule="!has(oldSelf.lifecycleState) || oldSelf.lifecycleState != 'Archived' || self.lifecycleState == 'Archived'",message="lifecycleState cannot transition from Archived"
type ClusterObjectSetRevisionSpec struct {
	// group is a label-safe identifier that links related revisions together.
	// All revisions sharing the same group form an ordered sequence. The value
	// must be at most 52 characters long.
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=52
	// +required
	Group string `json:"group"`

	// revision is the monotonically increasing sequence number within the
	// group. The first revision is 1.
	// +kubebuilder:validation:Minimum=1
	// +required
	Revision uint32 `json:"revision"`

	// lifecycleState controls whether this revision is actively reconciling its
	// managed objects. Allowed values are "Active" and "Archived". An Active
	// revision reconciles and reports availability. Transitioning to Archived
	// triggers teardown (deletion) of any objects still owned by this revision;
	// phases are torn down in reverse order. Once a revision is archived, it
	// cannot be unarchived.
	// +kubebuilder:validation:Required
	// +required
	LifecycleState LifecycleState `json:"lifecycleState"`

	ClusterObjectSetTemplateSpec `json:",inline"`
}

// ClusterObjectSetRevisionStatus reports the observed state of a
// ClusterObjectSetRevision.
type ClusterObjectSetRevisionStatus struct {
	// conditions represent the latest available observations of the revision's
	// state. The "Available" condition indicates whether all managed objects in
	// this revision satisfy their assertions.
	// +listType=map
	// +listMapKey=type
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// LifecycleState describes whether a ClusterObjectSetRevision is actively
// reconciling its managed objects.
//
// +kubebuilder:validation:Enum=Active;Archived
type LifecycleState string

const (
	// LifecycleStateActive indicates the revision is reconciling its managed
	// objects and reporting availability.
	LifecycleStateActive LifecycleState = "Active"

	// LifecycleStateArchived indicates the revision is deleting or has deleted
	// its managed objects.
	LifecycleStateArchived LifecycleState = "Archived"
)
