package controller

import (
	"testing"

	"github.com/stretchr/testify/assert"

	orbv1alpha1 "github.com/joelanford/orb-operator/api/v1alpha1"
)

func TestHandleResolutionError_FalseWhenNoHash(t *testing.T) {
	cos := &orbv1alpha1.ClusterObjectSet{}
	cos.Generation = 1

	r := &COSReconciler{}
	_ = r.handleResolutionError(cos, assert.AnError)

	cond := cos.Status.Conditions[0]
	assert.Equal(t, orbv1alpha1.ConditionTypeAvailable, cond.Type)
	assert.Equal(t, "False", string(cond.Status))
	assert.Equal(t, orbv1alpha1.ReasonInvalidRevision, cond.Reason)
}

func TestHandleResolutionError_UnknownWhenHashSet(t *testing.T) {
	cos := &orbv1alpha1.ClusterObjectSet{}
	cos.Generation = 1
	cos.Status.ResolvedContentHash = "previously-resolved"

	r := &COSReconciler{}
	_ = r.handleResolutionError(cos, assert.AnError)

	cond := cos.Status.Conditions[0]
	assert.Equal(t, orbv1alpha1.ConditionTypeAvailable, cond.Type)
	assert.Equal(t, "Unknown", string(cond.Status))
	assert.Equal(t, orbv1alpha1.ReasonInvalidRevision, cond.Reason)
}
