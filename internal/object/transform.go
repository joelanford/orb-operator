package object

import (
	"fmt"

	orbv1alpha1 "github.com/joelanford/orb-operator/api/v1alpha1"
)

func TransformClusterObjectSlice(obj interface{}) (interface{}, error) {
	slice, ok := obj.(*orbv1alpha1.ClusterObjectSlice)
	if !ok {
		return nil, fmt.Errorf("unexpected type %T, expected *ClusterObjectSlice", obj)
	}
	objectMap := make(map[orbv1alpha1.ObjectKey][]byte, len(slice.Objects))
	for _, so := range slice.Objects {
		objectMap[so.ObjectKey] = so.Content
	}
	slice.ObjectMap = objectMap
	slice.Objects = nil
	return slice, nil
}
