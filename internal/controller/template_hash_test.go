package controller

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/runtime"

	orbv1alpha1 "github.com/joelanford/orb-operator/api/v1alpha1"
)

func TestTemplateHash_Stability(t *testing.T) {
	tmpl := orbv1alpha1.ClusterObjectSetTemplate{
		Metadata: orbv1alpha1.ClusterObjectSetTemplateMetadata{
			Labels: map[string]string{"app": "test"},
		},
		Spec: orbv1alpha1.ClusterObjectSetTemplateSpec{
			Phases: []orbv1alpha1.Phase{{
				Name: "install",
				Objects: []orbv1alpha1.PhaseObject{{
					Object: runtime.RawExtension{Raw: []byte(`{"apiVersion":"v1","kind":"ConfigMap","metadata":{"name":"cm1"}}`)},
				}},
			}},
		},
	}

	h1 := templateHash(tmpl)
	h2 := templateHash(tmpl)
	assert.Equal(t, h1, h2, "same input must produce the same hash")
	assert.Len(t, h1, 8, "hash should be 8 hex characters")
}

func TestTemplateHash_Sensitivity(t *testing.T) {
	base := orbv1alpha1.ClusterObjectSetTemplate{
		Metadata: orbv1alpha1.ClusterObjectSetTemplateMetadata{
			Labels: map[string]string{"app": "test"},
		},
		Spec: orbv1alpha1.ClusterObjectSetTemplateSpec{
			Phases: []orbv1alpha1.Phase{{
				Name: "install",
				Objects: []orbv1alpha1.PhaseObject{{
					Object: runtime.RawExtension{Raw: []byte(`{"apiVersion":"v1","kind":"ConfigMap","metadata":{"name":"cm1"}}`)},
				}},
			}},
		},
	}

	changedLabel := orbv1alpha1.ClusterObjectSetTemplate{
		Metadata: orbv1alpha1.ClusterObjectSetTemplateMetadata{
			Labels: map[string]string{"app": "changed"},
		},
		Spec: base.Spec,
	}

	changedSpec := orbv1alpha1.ClusterObjectSetTemplate{
		Metadata: base.Metadata,
		Spec: orbv1alpha1.ClusterObjectSetTemplateSpec{
			Phases: []orbv1alpha1.Phase{{
				Name: "install",
				Objects: []orbv1alpha1.PhaseObject{{
					Object: runtime.RawExtension{Raw: []byte(`{"apiVersion":"v1","kind":"ConfigMap","metadata":{"name":"cm2"}}`)},
				}},
			}},
		},
	}

	baseHash := templateHash(base)
	assert.NotEqual(t, baseHash, templateHash(changedLabel), "different labels must produce a different hash")
	assert.NotEqual(t, baseHash, templateHash(changedSpec), "different spec must produce a different hash")
}
