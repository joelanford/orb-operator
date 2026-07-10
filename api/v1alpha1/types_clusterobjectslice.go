package v1alpha1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// ClusterObjectSlice is a cluster-scoped resource that holds Kubernetes
// object manifests for use by ClusterObjectSet phases via objectRef.
// It has no spec or status — it is a pure content store analogous to
// ConfigMap or Secret.
//
// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Cluster,shortName=cosl
// +kubebuilder:validation:XValidation:rule="self.objects == oldSelf.objects",message="objects is immutable"
type ClusterObjectSlice struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// objects is the list of Kubernetes object manifests stored in this slice.
	// Each entry is keyed by its Kubernetes identity (apiVersion, kind, name,
	// namespace). The list must contain between 1 and 256 entries. Duplicate
	// keys are rejected at admission time.
	// +kubebuilder:validation:MinItems=1
	// +kubebuilder:validation:MaxItems=256
	// +listType=map
	// +listMapKey=apiVersion
	// +listMapKey=kind
	// +listMapKey=name
	// +listMapKey=namespace
	// +required
	Objects []SliceObject `json:"objects"`

	// ObjectMap is an optional in-memory lookup representation of Objects,
	// keyed by ObjectKey. It is not serialized to the wire. Callers that
	// need keyed access should populate this field from Objects after
	// deserialization.
	ObjectMap map[ObjectKey][]byte `json:"-"`
}

// ObjectKey uniquely identifies a Kubernetes object by its API identity.
// It is embedded inline in SliceObject and ObjectRef to provide a single
// source of truth for identity fields and their validation.
type ObjectKey struct {
	// apiVersion is the API version of the object (e.g. "v1",
	// "apps/v1", "apiextensions.k8s.io/v1"). The format is an optional
	// DNS-1123 subdomain group followed by "/" and a DNS-1035 version.
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=253
	// +kubebuilder:validation:XValidation:rule="self.matches('^([a-z0-9]([-a-z0-9]*[a-z0-9])?(\\\\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*\\\\/)?[a-z]([-a-z0-9]*[a-z0-9])?$')",message="must be a valid API version: optional dns-subdomain group, '/', dns-1035 version"
	// +required
	APIVersion string `json:"apiVersion"`

	// kind is the kind of the object (e.g. "ConfigMap", "Deployment").
	// Must be a DNS-1035 label with mixed case allowed.
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=63
	// +kubebuilder:validation:XValidation:rule="self.matches('^[a-zA-Z]([-a-zA-Z0-9]*[a-zA-Z0-9])?$')",message="must be a DNS-1035 label (mixed case allowed)"
	// +required
	Kind string `json:"kind"`

	// name is the metadata.name of the object. Must be a valid DNS-1123
	// subdomain (lowercase alphanumeric, '-', or '.').
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=253
	// +kubebuilder:validation:XValidation:rule="self.matches('^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\\\\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$')",message="must be a valid DNS-1123 subdomain"
	// +required
	Name string `json:"name"`

	// namespace is the metadata.namespace of the object. Defaults to empty
	// string for cluster-scoped resources. Must be a valid DNS-1123 label
	// when non-empty.
	// +kubebuilder:validation:MaxLength=63
	// +kubebuilder:default=""
	// +kubebuilder:validation:XValidation:rule="self == '' || self.matches('^[a-z0-9]([-a-z0-9]*[a-z0-9])?$')",message="must be empty or a valid DNS-1123 label"
	// +optional
	Namespace string `json:"namespace,omitempty"`
}

// SliceObject holds a single Kubernetes object manifest with explicit identity
// fields for keyed lookup and admission-time uniqueness validation.
type SliceObject struct {
	ObjectKey `json:",inline"`

	// content is the Kubernetes object manifest as raw JSON, or as
	// gzip-compressed JSON. The format is auto-detected by checking for
	// the gzip magic number (0x1f 0x8b) in the first two bytes.
	// Decompression happens lazily when the object is referenced by a
	// ClusterObjectSet. The []byte type is automatically base64-encoded
	// during JSON serialization by the Kubernetes API machinery — no
	// user-space encoding is needed.
	// +kubebuilder:validation:MinLength=1
	// +required
	Content []byte `json:"content"`
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
