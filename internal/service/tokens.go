package service

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"math/big"
)

func generateSessionToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate session token: %w", err)
	}
	return hex.EncodeToString(b), nil
}

func generateRandomToken(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate random token: %w", err)
	}
	return hex.EncodeToString(b), nil
}

func generateNumericCode(length int) (string, error) {
	if length <= 0 {
		return "", nil
	}
	code := make([]byte, length)
	for i := range code {
		n, err := rand.Int(rand.Reader, big.NewInt(10))
		if err != nil {
			return "", fmt.Errorf("generate numeric code: %w", err)
		}
		code[i] = byte('0' + n.Int64())
	}
	return string(code), nil
}

// GenerateRandomTokenExported is an exported wrapper around generateRandomToken.
func GenerateRandomTokenExported(n int) (string, error) {
	return generateRandomToken(n)
}

func generateClientID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate client id: %w", err)
	}
	return hex.EncodeToString(b), nil
}

func generateClientSecret() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate client secret: %w", err)
	}
	return hex.EncodeToString(b), nil
}
