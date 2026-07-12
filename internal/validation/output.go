package validation

import (
	"encoding/base64"
	"errors"
	"io"
	"strings"
)

var (
	ErrInvalidBase64  = errors.New("invalid base64 output")
	ErrOutputTooLarge = errors.New("decoded output exceeds maximum size")
)

func ValidateBase64Output(value string, maxDecodedBytes int64) error {
	if value == "" {
		return nil
	}
	if maxDecodedBytes <= 0 {
		return ErrOutputTooLarge
	}

	decoder := base64.NewDecoder(base64.StdEncoding.Strict(), strings.NewReader(value))
	decodedBytes, err := io.Copy(io.Discard, io.LimitReader(decoder, maxDecodedBytes+1))
	if decodedBytes > maxDecodedBytes {
		return ErrOutputTooLarge
	}
	if err != nil {
		return ErrInvalidBase64
	}

	return nil
}
