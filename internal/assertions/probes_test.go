package assertions

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"pkg.package-operator.run/boxcutter/probing"

	orbv1alpha1 "github.com/joelanford/orb-operator/api/v1alpha1"
)

func TestProbeForAssertions_ConditionEqual(t *testing.T) {
	p, err := ProbeForAssertions([]orbv1alpha1.Assertion{{
		ConditionEqual: &orbv1alpha1.ConditionEqualAssertion{
			Type:   "Ready",
			Status: "True",
		},
	}})
	require.NoError(t, err)
	cp, ok := p.(*probing.ConditionProbe)
	require.True(t, ok)
	assert.Equal(t, "Ready", cp.Type)
	assert.Equal(t, "True", cp.Status)
}

func TestProbeForAssertions_FieldsEqual(t *testing.T) {
	p, err := ProbeForAssertions([]orbv1alpha1.Assertion{{
		FieldsEqual: &orbv1alpha1.FieldsEqualAssertion{
			FieldA: ".data.a",
			FieldB: ".data.b",
		},
	}})
	require.NoError(t, err)
	fe, ok := p.(*probing.FieldsEqualProbe)
	require.True(t, ok)
	assert.Equal(t, ".data.a", fe.FieldA)
	assert.Equal(t, ".data.b", fe.FieldB)
}

func TestProbeForAssertions_FieldValue(t *testing.T) {
	p, err := ProbeForAssertions([]orbv1alpha1.Assertion{{
		FieldValue: &orbv1alpha1.FieldValueAssertion{
			FieldPath: ".status.phase",
			Value:     "Running",
		},
	}})
	require.NoError(t, err)
	fv, ok := p.(*probing.FieldValueProbe)
	require.True(t, ok)
	assert.Equal(t, ".status.phase", fv.FieldPath)
	assert.Equal(t, "Running", fv.Value)
}

func TestProbeForAssertions_CELExpression(t *testing.T) {
	p, err := ProbeForAssertions([]orbv1alpha1.Assertion{{
		CELExpression: &orbv1alpha1.CELExpressionAssertion{
			Expression: "self.metadata.name == 'test'",
			Message:    "name must be test",
		},
	}})
	require.NoError(t, err)
	cp, ok := p.(*probing.CELProbe)
	require.True(t, ok)
	assert.Equal(t, "name must be test", cp.Message)
}

func TestProbeForAssertions_Multiple(t *testing.T) {
	p, err := ProbeForAssertions([]orbv1alpha1.Assertion{
		{ConditionEqual: &orbv1alpha1.ConditionEqualAssertion{Type: "Ready", Status: "True"}},
		{FieldValue: &orbv1alpha1.FieldValueAssertion{FieldPath: ".data.x", Value: "y"}},
	})
	require.NoError(t, err)
	and, ok := p.(probing.And)
	require.True(t, ok)
	assert.Len(t, and, 2)
}

func TestProbeForAssertions_Empty(t *testing.T) {
	p, err := ProbeForAssertions(nil)
	require.NoError(t, err)
	assert.Nil(t, p)
}

func TestProbeForAssertions_NoType(t *testing.T) {
	_, err := ProbeForAssertions([]orbv1alpha1.Assertion{{}})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no recognized type")
}
