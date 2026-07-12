package template

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/runtime"

	orbv1alpha1 "github.com/joelanford/orb-operator/api/v1alpha1"
)

func TestHash_Stability(t *testing.T) {
	tmpl := orbv1alpha1.ClusterObjectDeploymentTemplate{
		Metadata: orbv1alpha1.ClusterObjectDeploymentTemplateMetadata{
			Labels: map[string]string{"app": "test"},
		},
		Spec: orbv1alpha1.ClusterObjectDeploymentTemplateSpec{
			Phases: []orbv1alpha1.Phase{{
				Name: "install",
				Objects: []orbv1alpha1.PhaseObject{{
					Object: runtime.RawExtension{Raw: []byte(`{"apiVersion":"v1","kind":"ConfigMap","metadata":{"name":"cm1"}}`)},
				}},
			}},
		},
	}

	h1, err := Hash(tmpl)
	require.NoError(t, err)
	h2, err := Hash(tmpl)
	require.NoError(t, err)
	assert.Equal(t, h1, h2)
	assert.Len(t, h1, 8)
}

func TestHash_Sensitivity(t *testing.T) {
	base := orbv1alpha1.ClusterObjectDeploymentTemplate{
		Metadata: orbv1alpha1.ClusterObjectDeploymentTemplateMetadata{
			Labels: map[string]string{"app": "test"},
		},
		Spec: orbv1alpha1.ClusterObjectDeploymentTemplateSpec{
			Phases: []orbv1alpha1.Phase{{
				Name: "install",
				Objects: []orbv1alpha1.PhaseObject{{
					Object: runtime.RawExtension{Raw: []byte(`{"apiVersion":"v1","kind":"ConfigMap","metadata":{"name":"cm1"}}`)},
				}},
			}},
		},
	}

	changedLabel := orbv1alpha1.ClusterObjectDeploymentTemplate{
		Metadata: orbv1alpha1.ClusterObjectDeploymentTemplateMetadata{
			Labels: map[string]string{"app": "changed"},
		},
		Spec: base.Spec,
	}

	baseHash, err := Hash(base)
	require.NoError(t, err)
	changedHash, err := Hash(changedLabel)
	require.NoError(t, err)
	assert.NotEqual(t, baseHash, changedHash)
}

func TestBuildCOS(t *testing.T) {
	cod := &orbv1alpha1.ClusterObjectDeployment{}
	cod.Name = "my-cod"
	cod.UID = "test-uid"
	cod.Spec.Template = orbv1alpha1.ClusterObjectDeploymentTemplate{
		Metadata: orbv1alpha1.ClusterObjectDeploymentTemplateMetadata{
			Labels:      map[string]string{"app": "test"},
			Annotations: map[string]string{"note": "hello"},
		},
		Spec: orbv1alpha1.ClusterObjectDeploymentTemplateSpec{
			Phases: []orbv1alpha1.Phase{{
				Name: "install",
				Objects: []orbv1alpha1.PhaseObject{{
					Object: runtime.RawExtension{Raw: []byte(`{"apiVersion":"v1","kind":"ConfigMap","metadata":{"name":"cm1"}}`)},
				}},
			}},
		},
	}

	cos, err := BuildCOS(cod, 3, "abcd1234")
	require.NoError(t, err)
	assert.Equal(t, "my-cod-3", *cos.Name)
	assert.Equal(t, "abcd1234", cos.Labels[LabelTemplateHash])
	assert.Equal(t, "test", cos.Labels["app"])
	assert.Equal(t, "hello", cos.Annotations["note"])
	require.NotNil(t, cos.Spec)
	assert.Equal(t, "my-cod", *cos.Spec.Group)
	assert.Equal(t, uint32(3), *cos.Spec.Revision)
	require.Len(t, cos.OwnerReferences, 1)
	assert.Equal(t, "my-cod", *cos.OwnerReferences[0].Name)
	assert.True(t, *cos.OwnerReferences[0].Controller)
}
