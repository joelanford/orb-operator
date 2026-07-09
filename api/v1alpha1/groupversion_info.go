package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var (
	GroupVersion       = schema.GroupVersion{Group: "orb.operatorframework.io", Version: "v1alpha1"}
	SchemeGroupVersion = GroupVersion
	SchemeBuilder      = runtime.NewSchemeBuilder(addKnownTypes)
	AddToScheme        = SchemeBuilder.AddToScheme
)

func addKnownTypes(s *runtime.Scheme) error {
	s.AddKnownTypes(GroupVersion,
		&ClusterObjectDeployment{},
		&ClusterObjectDeploymentList{},
		&ClusterObjectSetRevision{},
		&ClusterObjectSetRevisionList{},
		&ClusterObjectSlice{},
		&ClusterObjectSliceList{},
	)
	metav1.AddToGroupVersion(s, GroupVersion)
	return nil
}
