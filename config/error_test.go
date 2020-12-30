package config

import (
	"errors"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestFieldErr_Constructor(t *testing.T) {
	err := newFieldErr("myField", errors.New("something went wrong!"))

	wantErr := &fieldErr{
		field: "myField",
		cause: errors.New("something went wrong!"),
	}

	require.Equal(t, wantErr, err)
}

func TestFieldErr(t *testing.T) {
	err := newFieldErr("myField", errors.New("something went wrong!"))

	wantErrMsg := "myField something went wrong!"

	require.EqualError(t, err, wantErrMsg)
}

func TestFieldErr_NestedFieldErr(t *testing.T) {
	baseErr := newFieldErr("myField", errors.New("something went wrong!"))
	topErr := newFieldErr("otherField", baseErr)

	wantErrMsg := "otherField.myField something went wrong!"

	require.EqualError(t, topErr, wantErrMsg)
}

func TestFieldErr_NestedArrFieldErr(t *testing.T) {
	baseErr := newArrFieldErr("myField", 2, errors.New("something went wrong!"))
	topErr := newFieldErr("otherField", baseErr)

	wantErrMsg := "otherField.myField[2] something went wrong!"

	require.EqualError(t, topErr, wantErrMsg)
}

func TestArrFieldErr_Constructor(t *testing.T) {
	err := newArrFieldErr("myField", 2, errors.New("something went wrong!"))

	wantErr := &arrFieldErr{
		field: "myField",
		i:     2,
		cause: errors.New("something went wrong!"),
	}

	require.Equal(t, wantErr, err)
}

func TestArrFieldErr(t *testing.T) {
	err := newArrFieldErr("myField", 2, errors.New("something went wrong!"))

	wantErrMsg := "myField[2] something went wrong!"

	require.EqualError(t, err, wantErrMsg)
}

func TestArrFieldErr_NestedFieldErr(t *testing.T) {
	baseErr := newFieldErr("myField", errors.New("something went wrong!"))
	topErr := newArrFieldErr("otherField", 3, baseErr)

	wantErrMsg := "otherField[3].myField something went wrong!"

	require.EqualError(t, topErr, wantErrMsg)
}

func TestArrFieldErr_NestedArrFieldErr(t *testing.T) {
	baseErr := newArrFieldErr("myField", 2, errors.New("something went wrong!"))
	topErr := newArrFieldErr("otherField", 3, baseErr)

	wantErrMsg := "otherField[3].myField[2] something went wrong!"

	require.EqualError(t, topErr, wantErrMsg)
}
