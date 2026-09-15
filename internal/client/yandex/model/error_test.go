package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestErrorResponse_IsNotFound_NoLyrics(t *testing.T) {
	t.Parallel()

	err := &ErrorResponse{}
	err.APIError.Name = "No lyrics found for track"

	assert.True(t, err.IsNotFound())
	assert.Equal(t, "No lyrics found for track", err.ErrorName())
	assert.Equal(t, "No lyrics found for track", err.Error())
}

func TestErrorResponse_IsNotFound_FalseForValidate(t *testing.T) {
	t.Parallel()

	err := &ErrorResponse{}
	err.APIError.Name = "validate"
	err.APIError.Message = "Parameters requirements are not met."

	assert.False(t, err.IsNotFound())
}
