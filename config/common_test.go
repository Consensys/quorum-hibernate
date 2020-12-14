package config

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestNamedError(t *testing.T) {
	got := namedValidationError{name: "someName", errMsg: "someErr"}
	require.EqualError(t, got, "name = someName: someErr")
}
