package validation_test

import (
	"encoding/base64"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/svetlyopet/heimdallr/internal/validation"
)

func TestValidateBase64Output(t *testing.T) {
	t.Parallel()

	exact := base64.StdEncoding.EncodeToString([]byte("1234"))
	require.NoError(t, validation.ValidateBase64Output(exact, 4))

	oversized := base64.StdEncoding.EncodeToString([]byte("12345"))
	err := validation.ValidateBase64Output(oversized, 4)
	require.ErrorIs(t, err, validation.ErrOutputTooLarge)

	err = validation.ValidateBase64Output("not-base64!", 4)
	require.ErrorIs(t, err, validation.ErrInvalidBase64)

	err = validation.ValidateBase64Output(
		base64.StdEncoding.EncodeToString([]byte(strings.Repeat("x", 5))),
		4,
	)
	require.True(t, errors.Is(err, validation.ErrOutputTooLarge))
}
