package auth

import (
	"crypto/rand"
	"encoding/hex"
	"io"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

const bcryptCost = 12

const passwordAlphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*()-_=+[]{}<>?"

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

func generateSecurePassword(length int) (string, error) {
	return generateSecurePasswordWithReader(rand.Reader, length)
}

func generateSecurePasswordWithReader(reader io.Reader, length int) (string, error) {
	if length <= 0 {
		return "", ErrInvalidPasswordValue
	}

	alphabetSize := len(passwordAlphabet)
	limit := 256 - (256 % alphabetSize)

	result := make([]byte, length)
	for i := 0; i < length; {
		var randomByte [1]byte
		if _, err := io.ReadFull(reader, randomByte[:]); err != nil {
			return "", err
		}

		if int(randomByte[0]) >= limit {
			continue
		}

		result[i] = passwordAlphabet[int(randomByte[0])%alphabetSize]
		i++
	}

	return string(result), nil
}
