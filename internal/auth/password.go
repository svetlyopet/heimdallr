package auth

import (
	"crypto/sha256"
	"crypto/subtle"
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

	return string(hashed), nil
}

func legacyHashPassword(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

func verifyPassword(password, storedHash string) (valid bool, needsRehash bool) {
	if strings.HasPrefix(storedHash, "$2a$") || strings.HasPrefix(storedHash, "$2b$") || strings.HasPrefix(storedHash, "$2y$") {
		err := bcrypt.CompareHashAndPassword([]byte(storedHash), []byte(password))
		return err == nil, false
	}

	if len(storedHash) == 64 {
		legacy := legacyHashPassword(password)
		if subtle.ConstantTimeCompare([]byte(storedHash), []byte(legacy)) == 1 {
			return true, true
		}
	}

	return false, false
}
