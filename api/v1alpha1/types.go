package v1alpha1

import "k8s.io/apimachinery/pkg/runtime"

// CollisionProtection determines how the controller handles objects that
// already exist on the cluster when a revision attempts to manage them.
//
// +kubebuilder:validation:Enum=Prevent;IfNoController;None
type CollisionProtection string

const (
	// CollisionProtectionPrevent reports a collision if the target object
	// already exists on the cluster, regardless of whether it has a controller
	// owner.
	CollisionProtectionPrevent CollisionProtection = "Prevent"

	// CollisionProtectionIfNoController adopts and updates objects that exist
	// but have no controller owner. Objects that are already owned by another
	// controller cause a collision.
	CollisionProtectionIfNoController CollisionProtection = "IfNoController"

	// CollisionProtectionNone adopts and updates objects unconditionally, even
	// if they are owned by another controller. Use with caution: multiple
	// controllers managing the same object may cause unnecessary API server and
	// etcd load.
	CollisionProtectionNone CollisionProtection = "None"
)

const (
	// ConditionTypeAvailable is the condition type that reports whether all
	// managed objects satisfy their assertions.
	ConditionTypeAvailable = "Available"

	// ReasonAvailable indicates all assertions are satisfied.
	ReasonAvailable = "Available"

	// ReasonUnavailable indicates one or more assertions are not satisfied.
	ReasonUnavailable = "Unavailable"

	// ReasonProgressing indicates reconciliation is in progress and assertions
	// have not yet been evaluated.
	ReasonProgressing = "Progressing"

	// ReasonArchived indicates the revision is archived and no longer
	// reconciling.
	ReasonArchived = "Archived"

	// ReasonSuperseded indicates a higher-numbered active revision exists in
	// the same group. A superseded revision relinquishes control of any objects
	// that the highest-numbered revision also claims.
	ReasonSuperseded = "Superseded"
)

// Phase is an ordered group of objects that are applied together. Phases within
// a revision are applied sequentially: the controller does not advance to the
// next phase until all objects in the current phase satisfy their assertions.
type Phase struct {
	// name is a human-readable identifier for this phase. It is used in log
	// messages and status reporting.
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=253
	// +required
	Name string `json:"name"`

	// collisionProtection overrides the revision-level collision protection
	// setting for all objects in this phase. When omitted, the revision-level
	// setting applies.
	// +optional
	CollisionProtection *CollisionProtection `json:"collisionProtection,omitempty"`

	// objects is the list of Kubernetes objects to manage in this phase. The
	// list must contain between 1 and 50 objects.
	// +kubebuilder:validation:MinItems=1
	// +kubebuilder:validation:MaxItems=50
	// +required
	Objects []PhaseObject `json:"objects"`
}

// PhaseObject wraps a single Kubernetes object with optional collision
// protection and availability assertions.
type PhaseObject struct {
	// object is the Kubernetes resource to create or update. The controller
	// applies this object to the cluster and manages it for the lifetime of the
	// owning revision.
	// +required
	Object runtime.RawExtension `json:"object"`

	// collisionProtection overrides the phase-level collision protection
	// setting for this specific object. When omitted, the phase-level setting
	// applies.
	// +optional
	CollisionProtection *CollisionProtection `json:"collisionProtection,omitempty"`

	// assertions define conditions that must be met before this object is
	// considered available. A maximum of 16 assertions may be specified. When
	// omitted, the object is considered available immediately after successful
	// apply.
	// +kubebuilder:validation:MaxItems=16
	// +optional
	Assertions []Assertion `json:"assertions,omitempty"`
}

// Assertion defines a single availability check that must pass before the
// containing object is considered available. Exactly one of conditionEqual,
// fieldsEqual, fieldValue, or celExpression must be set.
//
// +kubebuilder:validation:ExactlyOneOf=conditionEqual;fieldsEqual;fieldValue;celExpression
type Assertion struct {
	// conditionEqual checks that a status condition with the given type has the
	// expected status value.
	// +optional
	ConditionEqual *ConditionEqualAssertion `json:"conditionEqual,omitempty"`

	// fieldsEqual checks that two fields on the object have equal values.
	// +optional
	FieldsEqual *FieldsEqualAssertion `json:"fieldsEqual,omitempty"`

	// fieldValue checks that a single field on the object matches an expected
	// value.
	// +optional
	FieldValue *FieldValueAssertion `json:"fieldValue,omitempty"`

	// celExpression evaluates a CEL expression against the object. The
	// expression must evaluate to true for the assertion to pass.
	// +optional
	CELExpression *CELExpressionAssertion `json:"celExpression,omitempty"`
}

// ConditionEqualAssertion asserts that the object has a status condition
// matching the specified type and status value.
type ConditionEqualAssertion struct {
	// type is the condition type to match, for example "Available" or "Ready".
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=316
	// +required
	Type string `json:"type"`

	// status is the expected value of the condition's status field. Allowed
	// values are "True", "False", and "Unknown".
	// +kubebuilder:validation:Enum=True;False;Unknown
	// +required
	Status string `json:"status"`
}

// FieldsEqualAssertion asserts that two fields on the object have equal values.
// Both fields are specified as JSON path expressions.
type FieldsEqualAssertion struct {
	// fieldA is the JSON path to the first field to compare.
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=512
	// +required
	FieldA string `json:"fieldA"`

	// fieldB is the JSON path to the second field to compare.
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=512
	// +required
	FieldB string `json:"fieldB"`
}

// FieldValueAssertion asserts that a single field on the object matches an
// expected value.
type FieldValueAssertion struct {
	// fieldPath is the JSON path to the field to check.
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=512
	// +required
	FieldPath string `json:"fieldPath"`

	// value is the expected value of the field. An empty string is a valid
	// expected value.
	// +kubebuilder:validation:MaxLength=1024
	// +required
	Value string `json:"value"`
}

// CELExpressionAssertion evaluates a CEL expression against the managed object.
// The expression has access to the full object and must evaluate to a boolean.
type CELExpressionAssertion struct {
	// expression is a CEL expression that must evaluate to true for the
	// assertion to pass. The managed object is available as "self" in the
	// expression scope.
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=4096
	// +required
	Expression string `json:"expression"`

	// message is an optional human-readable explanation shown when the assertion
	// fails. When omitted, the raw expression text is used in error messages.
	// +kubebuilder:validation:MaxLength=1024
	// +optional
	Message string `json:"message,omitempty"`
}
