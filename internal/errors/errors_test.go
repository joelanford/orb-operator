package errors

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestObjectResolutionError(t *testing.T) {
	inner := fmt.Errorf("slice not found")
	err := &ObjectResolutionError{Err: inner}

	assert.Equal(t, "slice not found", err.Error())
	assert.Equal(t, inner, err.Unwrap())

	var target *ObjectResolutionError
	require.ErrorAs(t, err, &target)
	assert.Equal(t, inner, target.Err)
}

func TestInternalError(t *testing.T) {
	inner := fmt.Errorf("engine setup failed")
	err := &InternalError{Err: inner}

	assert.Equal(t, "engine setup failed", err.Error())
	assert.Equal(t, inner, err.Unwrap())

	var target *InternalError
	require.ErrorAs(t, err, &target)
	assert.Equal(t, inner, target.Err)
}

func TestErrorTypeDisambiguation(t *testing.T) {
	resolution := &ObjectResolutionError{Err: fmt.Errorf("res")}
	internal := &InternalError{Err: fmt.Errorf("int")}

	var resTarget *ObjectResolutionError
	var intTarget *InternalError

	require.ErrorAs(t, resolution, &resTarget)
	require.NotErrorAs(t, resolution, &intTarget)

	require.NotErrorAs(t, internal, &resTarget)
	require.ErrorAs(t, internal, &intTarget)
}
