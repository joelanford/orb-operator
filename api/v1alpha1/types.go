package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

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

	Spec   ClusterObjectSetRevisionSpec   `json:"spec,omitempty"`
	Status ClusterObjectSetRevisionStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
type ClusterObjectSetRevisionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ClusterObjectSetRevision `json:"items"`
}

// +kubebuilder:validation:XValidation:rule="self.group == oldSelf.group",message="group is immutable"
// +kubebuilder:validation:XValidation:rule="self.revision == oldSelf.revision",message="revision is immutable"
// +kubebuilder:validation:XValidation:rule="self.phases == oldSelf.phases",message="phases is immutable"
// +kubebuilder:validation:XValidation:rule="!has(oldSelf.collisionProtection) && !has(self.collisionProtection) || has(oldSelf.collisionProtection) && has(self.collisionProtection) && self.collisionProtection == oldSelf.collisionProtection",message="collisionProtection is immutable"
// +kubebuilder:validation:XValidation:rule="!has(oldSelf.lifecycleState) || oldSelf.lifecycleState != 'Archived' || self.lifecycleState == 'Archived'",message="lifecycleState cannot transition from Archived"
type ClusterObjectSetRevisionSpec struct {
	// +kubebuilder:validation:MaxLength=52
	Group string `json:"group"`
	// +kubebuilder:validation:Minimum=1
	Revision                     uint32         `json:"revision"`
	LifecycleState               LifecycleState `json:"lifecycleState,omitempty"`
	ClusterObjectSetTemplateSpec `json:",inline"`
}

type ClusterObjectSetRevisionStatus struct {
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:validation:Enum=Active;Archived
type LifecycleState string

const (
	LifecycleStateActive   LifecycleState = "Active"
	LifecycleStateArchived LifecycleState = "Archived"
)

// +kubebuilder:validation:Enum=Prevent;IfNoController;None
type CollisionProtection string

const (
	CollisionProtectionPrevent        CollisionProtection = "Prevent"
	CollisionProtectionIfNoController CollisionProtection = "IfNoController"
	CollisionProtectionNone           CollisionProtection = "None"
)

const (
	ConditionTypeAvailable = "Available"

	ReasonAvailable   = "Available"
	ReasonUnavailable = "Unavailable"
	ReasonArchived    = "Archived"
	ReasonSuperseded  = "Superseded"
)

type Phase struct {
	Name                string               `json:"name"`
	CollisionProtection *CollisionProtection `json:"collisionProtection,omitempty"`
	// +kubebuilder:validation:MinItems=1
	// +kubebuilder:validation:MaxItems=50
	Objects []PhaseObject `json:"objects"`
}

type PhaseObject struct {
	Object              runtime.RawExtension `json:"object"`
	CollisionProtection *CollisionProtection `json:"collisionProtection,omitempty"`
	// +kubebuilder:validation:MaxItems=16
	Assertions []Assertion `json:"assertions,omitempty"`
}

type Assertion struct {
	ConditionEqual *ConditionEqualAssertion `json:"conditionEqual,omitempty"`
	FieldsEqual    *FieldsEqualAssertion    `json:"fieldsEqual,omitempty"`
	FieldValue     *FieldValueAssertion     `json:"fieldValue,omitempty"`
	CELExpression  *CELExpressionAssertion  `json:"celExpression,omitempty"`
}

type ConditionEqualAssertion struct {
	Type   string `json:"type"`
	Status string `json:"status"`
}

type FieldsEqualAssertion struct {
	FieldA string `json:"fieldA"`
	FieldB string `json:"fieldB"`
}

type FieldValueAssertion struct {
	FieldPath string `json:"fieldPath"`
	Value     string `json:"value"`
}

type CELExpressionAssertion struct {
	Expression string `json:"expression"`
	Message    string `json:"message,omitempty"`
}
