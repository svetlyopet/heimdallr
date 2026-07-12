package auth

import (
	"encoding/hex"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

const bcryptCost = 12

func hashPassword(value string) (string, error) {
	hashed, err := bcrypt.GenerateFromPassword([]byte(value), bcryptCost)
	if err != nil {
		return "", err
	}

	result := string(hashed)
	if !isBcryptHash(result) {
		return "", ErrInvalidPasswordValue
	}

	return result, nil
}

func isBcryptHash(hash string) bool {
	return strings.HasPrefix(hash, "$2a$") ||
		strings.HasPrefix(hash, "$2b$") ||
		strings.HasPrefix(hash, "$2y$")
}

func IsLegacyPasswordHash(hash string) bool {
	if len(hash) != 64 {
		return false
	}

	decoded, err := hex.DecodeString(hash)
	return err == nil && len(decoded) == 32
}

func verifyPassword(password, storedHash string) (valid bool, needsRehash bool) {
	if !isBcryptHash(storedHash) {
		return false, false
	}

	err := bcrypt.CompareHashAndPassword([]byte(storedHash), []byte(password))
	return err == nil, false
}
