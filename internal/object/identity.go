package object

import (
	"encoding/json"
	"fmt"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	orbv1alpha1 "github.com/joelanford/orb-operator/api/v1alpha1"
)

type objectIdentity struct {
	APIVersion string `json:"apiVersion"`
	Kind       string `json:"kind"`
	Metadata   struct {
		Name      string `json:"name"`
		Namespace string `json:"namespace"`
	} `json:"metadata"`
}

func ResolveIdentities(phases []orbv1alpha1.Phase) (*Result, error) {
	var result []Phase
	for _, p := range phases {
		rp := Phase{
			Name:                p.Name,
			CollisionProtection: p.CollisionProtection,
		}
		for _, po := range p.Objects {
			id, err := extractIdentity(p.Name, po)
			if err != nil {
				return nil, err
			}
			obj := &unstructured.Unstructured{}
			obj.SetAPIVersion(id.APIVersion)
			obj.SetKind(id.Kind)
			obj.SetName(id.Metadata.Name)
			obj.SetNamespace(id.Metadata.Namespace)

			rp.Objects = append(rp.Objects, Object{
				Obj:                 obj,
				CollisionProtection: po.CollisionProtection,
			})
		}
		result = append(result, rp)
	}
	return &Result{Phases: result}, nil
}

func extractIdentity(phaseName string, po orbv1alpha1.PhaseObject) (objectIdentity, error) {
	if po.ObjectRef != nil {
		return objectIdentity{
			APIVersion: po.ObjectRef.APIVersion,
			Kind:       po.ObjectRef.Kind,
			Metadata: struct {
				Name      string `json:"name"`
				Namespace string `json:"namespace"`
			}{
				Name:      po.ObjectRef.Name,
				Namespace: po.ObjectRef.Namespace,
			},
		}, nil
	}
	if len(po.Object.Raw) == 0 {
		return objectIdentity{}, fmt.Errorf("phase %q: object has neither inline content nor objectRef", phaseName)
	}
	var id objectIdentity
	if err := json.Unmarshal(po.Object.Raw, &id); err != nil {
		return objectIdentity{}, fmt.Errorf("phase %q: extracting identity: %w", phaseName, err)
	}
	return id, nil
}
