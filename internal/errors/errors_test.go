package errors

import (
	"errors"
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
	require.True(t, errors.As(err, &target))
	assert.Equal(t, inner, target.Err)
}

func TestInternalError(t *testing.T) {
	inner := fmt.Errorf("engine setup failed")
	err := &InternalError{Err: inner}

	assert.Equal(t, "engine setup failed", err.Error())
	assert.Equal(t, inner, err.Unwrap())

	var target *InternalError
	require.True(t, errors.As(err, &target))
	assert.Equal(t, inner, target.Err)
}

func TestErrorTypeDisambiguation(t *testing.T) {
	resolution := &ObjectResolutionError{Err: fmt.Errorf("res")}
	internal := &InternalError{Err: fmt.Errorf("int")}

	var resTarget *ObjectResolutionError
	var intTarget *InternalError

	assert.True(t, errors.As(resolution, &resTarget))
	assert.False(t, errors.As(resolution, &intTarget))

	assert.False(t, errors.As(internal, &resTarget))
	assert.True(t, errors.As(internal, &intTarget))
}
