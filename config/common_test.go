package config

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNamedError(t *testing.T) {
	got := namedValidationError{name: "someName", errMsg: "someErr"}
	require.EqualError(t, got, "name = someName: someErr")
}
