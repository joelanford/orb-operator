package v1alpha1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// ClusterObjectSet represents an immutable snapshot of a set of
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
// +kubebuilder:resource:scope=Cluster,shortName=cos
// +kubebuilder:printcolumn:name="Group",type=string,JSONPath=`.spec.group`
// +kubebuilder:printcolumn:name="Rev",type=integer,JSONPath=`.spec.revision`
// +kubebuilder:printcolumn:name="Available",type=integer,JSONPath=`.status.objectCounts.available`
// +kubebuilder:printcolumn:name="Synced",type=integer,JSONPath=`.status.objectCounts.synced`
// +kubebuilder:printcolumn:name="Present",type=integer,JSONPath=`.status.objectCounts.present`
// +kubebuilder:printcolumn:name="Total",type=integer,JSONPath=`.status.objectCounts.total`
// +kubebuilder:printcolumn:name="Lifecycle",type=string,JSONPath=`.spec.lifecycleState`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
type ClusterObjectSet struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// spec defines the desired state of this revision, including the objects to
	// manage and their lifecycle configuration.
	// +required
	Spec ClusterObjectSetSpec `json:"spec"`

	// status reports the observed state of this revision, including availability
	// conditions.
	// +optional
	Status ClusterObjectSetStatus `json:"status,omitempty"`
}

// ClusterObjectSetList is a list of ClusterObjectSet resources.
//
// +kubebuilder:object:root=true
type ClusterObjectSetList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	// items is the list of ClusterObjectSet resources.
	// +required
	Items []ClusterObjectSet `json:"items"`
}

// ClusterObjectSetSpec defines the desired state of a
// ClusterObjectSet. All fields except lifecycleState are immutable
// after creation.
//
// +kubebuilder:validation:XValidation:rule="self.group == oldSelf.group",message="group is immutable"
// +kubebuilder:validation:XValidation:rule="self.revision == oldSelf.revision",message="revision is immutable"
// +kubebuilder:validation:XValidation:rule="self.phases == oldSelf.phases",message="phases is immutable"
// +kubebuilder:validation:XValidation:rule="!has(oldSelf.collisionProtection) && !has(self.collisionProtection) || has(oldSelf.collisionProtection) && has(self.collisionProtection) && self.collisionProtection == oldSelf.collisionProtection",message="collisionProtection is immutable"
// +kubebuilder:validation:XValidation:rule="!has(oldSelf.lifecycleState) || oldSelf.lifecycleState != 'Archived' || self.lifecycleState == 'Archived'",message="lifecycleState cannot transition from Archived"
type ClusterObjectSetSpec struct {
	// group is a label-safe identifier that links related revisions together.
	// All revisions sharing the same group form an ordered sequence. The value
	// must be at most 52 characters long.
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=52
	// +kubebuilder:validation:XValidation:rule="self.matches('^[a-z]([a-z0-9-]*[a-z0-9])?$')",message="group must be lowercase alphanumeric or '-', starting with a letter, ending with an alphanumeric"
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

	ClusterObjectDeploymentTemplateSpec `json:",inline"`
}

// ClusterObjectSetStatus reports the observed state of a
// ClusterObjectSet.
//
// +kubebuilder:validation:XValidation:rule="!has(oldSelf.completedAt) || (has(self.completedAt) && self.completedAt == oldSelf.completedAt)",message="completedAt is immutable once set"
// +kubebuilder:validation:XValidation:rule="!has(oldSelf.resolvedContentHash) || (has(self.resolvedContentHash) && self.resolvedContentHash == oldSelf.resolvedContentHash)",message="resolvedContentHash is immutable once set"
// +kubebuilder:validation:XValidation:rule="has(self.observedPhases) == has(self.objectCounts)",message="observedPhases and objectCounts must both be present or both be absent"
// +kubebuilder:validation:XValidation:rule="!has(self.observedPhases) || self.objectCounts.total == self.observedPhases.map(p, p.objectCounts.total).sum()",message="objectCounts.total must equal sum of per-phase totals"
// +kubebuilder:validation:XValidation:rule="!has(self.observedPhases) || self.objectCounts.synced == self.observedPhases.map(p, p.objectCounts.synced).sum()",message="objectCounts.synced must equal sum of per-phase synced"
// +kubebuilder:validation:XValidation:rule="!has(self.observedPhases) || self.objectCounts.present == self.observedPhases.map(p, p.objectCounts.present).sum()",message="objectCounts.present must equal sum of per-phase present"
// +kubebuilder:validation:XValidation:rule="!has(self.observedPhases) || self.objectCounts.available == self.observedPhases.map(p, p.objectCounts.available).sum()",message="objectCounts.available must equal sum of per-phase available"
type ClusterObjectSetStatus struct {
	// conditions represent the latest available observations of the revision's
	// state. The "Available" condition indicates whether all managed objects in
	// this revision satisfy their assertions.
	// +listType=map
	// +listMapKey=type
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// resolvedContentHash is a SHA-256 hash of all resolved phase object
	// content (inline objects and objectRef-resolved objects, in phase order).
	// Set once on the first successful resolution and never changed. Used to
	// detect content substitution (e.g. a ClusterObjectSlice deleted and
	// recreated with different content).
	// +optional
	ResolvedContentHash string `json:"resolvedContentHash,omitempty"`

	// completedAt is the timestamp when all phases first completed
	// successfully. Set once and never cleared. Nil means the revision
	// has never been fully available. When set and Available is False,
	// the revision has regressed after a successful rollout.
	// +optional
	CompletedAt *metav1.Time `json:"completedAt,omitempty"`

	// objectCounts reports aggregate object counts across all phases.
	// Values are sums of the per-phase counts in observedPhases.
	// +optional
	ObjectCounts *ObjectCounts `json:"objectCounts,omitempty"`

	// observedPhases reports the observed state of each phase in the
	// revision. All phases from the spec are always listed, in spec
	// order. Each phase's status indicates whether the controller has
	// evaluated it and whether it has completed. The list may contain
	// at most 20 entries, matching the maximum number of spec phases.
	// +kubebuilder:validation:MaxItems=20
	// +listType=map
	// +listMapKey=name
	// +optional
	ObservedPhases []ObservedPhase `json:"observedPhases,omitempty"`
}

// PhaseStatus describes the current state of a phase in the rollout.
//
// +kubebuilder:validation:Enum=Invalid;Pending;Reconciling;WaitingForAssertions;Available;Unknown;Superseded;TearingDown;TeardownComplete
type PhaseStatus string

const (
	// PhaseStatusInvalid indicates the phase failed preflight validation.
	// The error and objectDetails fields describe what went wrong. Some
	// errors are permanent (e.g. cross-phase duplication), while others
	// may resolve on a future reconcile (e.g. a missing CRD is installed).
	PhaseStatusInvalid PhaseStatus = "Invalid"

	// PhaseStatusPending indicates the phase is not being actively
	// reconciled, but the controller has checked and some objects do not
	// match the desired state. The objectDetails field lists what would
	// need to change.
	PhaseStatusPending PhaseStatus = "Pending"

	// PhaseStatusReconciling indicates the controller is actively applying
	// objects in this phase and not all objects are synced yet.
	PhaseStatusReconciling PhaseStatus = "Reconciling"

	// PhaseStatusWaitingForAssertions indicates all objects in this phase are
	// synced but some assertions are not yet passing. The controller is
	// not actively writing; it is waiting for other controllers or
	// external actions to bring the objects into the expected state.
	PhaseStatusWaitingForAssertions PhaseStatus = "WaitingForAssertions"

	// PhaseStatusAvailable indicates all objects in this phase are synced
	// and pass their assertions.
	PhaseStatusAvailable PhaseStatus = "Available"

	// PhaseStatusUnknown indicates this phase was not evaluated during
	// the most recent reconcile.
	PhaseStatusUnknown PhaseStatus = "Unknown"

	// PhaseStatusSuperseded indicates all objects in this phase have been
	// adopted by a newer revision.
	PhaseStatusSuperseded PhaseStatus = "Superseded"

	// PhaseStatusTearingDown indicates the controller is actively deleting
	// objects in this phase. Objects still awaiting deletion are listed
	// in objectDetails.
	PhaseStatusTearingDown PhaseStatus = "TearingDown"

	// PhaseStatusTeardownComplete indicates all objects in this phase have
	// been deleted from the cluster.
	PhaseStatusTeardownComplete PhaseStatus = "TeardownComplete"
)

// ObservedPhase reports the observed state of a single phase in the rollout.
//
// +kubebuilder:validation:XValidation:rule="!has(oldSelf.completedAt) || (has(self.completedAt) && self.completedAt == oldSelf.completedAt)",message="completedAt is immutable once set"
type ObservedPhase struct {
	// name is the name of the phase from the spec. Must be a valid DNS-1035
	// label: lowercase alphanumeric characters or '-', must start with a
	// letter and end with an alphanumeric character (e.g. "my-phase",
	// "phase1"), matching the Phase name constraints.
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=63
	// +kubebuilder:validation:XValidation:rule="self.matches('^[a-z]([a-z0-9-]*[a-z0-9])?$')",message="name must be a valid DNS-1035 label: lowercase alphanumeric or '-', starting with a letter, ending with an alphanumeric"
	// +required
	Name string `json:"name"`

	// status is the current state of this phase in the rollout. Must be
	// one of Invalid, Pending, Reconciling, WaitingForAssertions,
	// Available, Unknown, Superseded, TearingDown, or TeardownComplete.
	// +required
	Status PhaseStatus `json:"status"`

	// completedAt is the timestamp when this phase first became Available.
	// Set once and never cleared. Nil means the phase has never been
	// Available.
	// +optional
	CompletedAt *metav1.Time `json:"completedAt,omitempty"`

	// message is a phase-level message providing context about the
	// current status (e.g. a validation error for Invalid phases, or
	// a summary for Pending phases). At most 1024 characters; longer
	// messages are truncated by the controller.
	// +kubebuilder:validation:MaxLength=1024
	// +optional
	Message string `json:"message,omitempty"`

	// objectCounts reports the number of objects in this phase by state.
	ObjectCounts ObjectCounts `json:"objectCounts"`

	// objectDetails lists objects in this phase that are not yet
	// complete. For Reconciling and Pending phases, this includes probe
	// failures, collisions, creation/update errors, and any other
	// condition preventing completion. For TearingDown phases, this
	// lists objects still awaiting deletion. Each entry identifies the
	// object and carries failure messages. Empty when status is
	// Available, TeardownComplete, or Unknown. The list may contain at
	// most 50 entries, matching the maximum number of objects per phase.
	// +kubebuilder:validation:MaxItems=50
	// +optional
	ObjectDetails []ObjectStatus `json:"objectDetails,omitempty"`
}

// ObjectCounts reports the number of objects in a phase by state.
type ObjectCounts struct {
	// total is the number of objects in this phase.
	Total int64 `json:"total"`

	// present is the number of objects in this phase that exist on the
	// cluster. During reconcile, this counts objects that the controller
	// has found or created. During teardown, this decrements toward zero
	// as objects are deleted.
	Present int64 `json:"present"`

	// synced is the number of objects in this phase whose cluster state
	// matches the desired state.
	Synced int64 `json:"synced"`

	// available is the number of objects in this phase that are synced
	// and pass their assertions.
	Available int64 `json:"available"`
}

// ObjectStatus identifies a managed object and its failure messages.
type ObjectStatus struct {
	// group is the API group of the object (empty string for core
	// resources). At most 253 characters (DNS subdomain max).
	// +kubebuilder:validation:MaxLength=253
	// +optional
	Group string `json:"group,omitempty"`

	// version is the API version of the object. Must be between 1 and
	// 63 characters.
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=63
	// +required
	Version string `json:"version"`

	// kind is the kind of the object. Must be between 1 and 63
	// characters.
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=63
	// +required
	Kind string `json:"kind"`

	// namespace is the namespace of the object. Empty for cluster-scoped
	// resources. At most 253 characters (DNS subdomain max).
	// +kubebuilder:validation:MaxLength=253
	// +optional
	Namespace string `json:"namespace,omitempty"`

	// name is the name of the object. Must be between 1 and 253
	// characters (DNS subdomain max).
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=253
	// +required
	Name string `json:"name"`

	// messages lists the failure reasons for this object. The maximum
	// of 17 entries accounts for up to 16 assertion probe failures
	// plus one collision message. Each message is at most 1024
	// characters; longer messages are truncated by the controller.
	// +kubebuilder:validation:MaxItems=17
	// +kubebuilder:validation:items:MaxLength=1024
	// +optional
	Messages []string `json:"messages,omitempty"`
}

// LifecycleState describes whether a ClusterObjectSet is actively
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
