package password

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"golang.org/x/crypto/argon2"
)

const (
	memory      uint32 = 64 * 1024
	iterations  uint32 = 3
	parallelism uint8  = 2
	keyLength   uint32 = 32
	saltLength         = 16
)

var ErrInvalidHash = errors.New("password: invalid hash format")

func Hash(plainText string) (string, error) {
	if plainText == "" {
		return "", fmt.Errorf("password.Hash: password is required")
	}

	salt := make([]byte, saltLength)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("password.Hash: generate salt: %w", err)
	}

	derivedKey := argon2.IDKey([]byte(plainText), salt, iterations, memory, parallelism, keyLength)
	return fmt.Sprintf(
		"$argon2id$v=19$m=%d,t=%d,p=%d$%s$%s",
		memory,
		iterations,
		parallelism,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(derivedKey),
	), nil
}

func Compare(plainText, encodedHash string) (bool, error) {
	parameters, salt, expectedKey, err := parse(encodedHash)
	if err != nil {
		return false, err
	}

	actualKey := argon2.IDKey(
		[]byte(plainText),
		salt,
		parameters.iterations,
		parameters.memory,
		parameters.parallelism,
		parameters.keyLength,
	)
	return subtle.ConstantTimeCompare(actualKey, expectedKey) == 1, nil
}

type parameters struct {
	memory      uint32
	iterations  uint32
	parallelism uint8
	keyLength   uint32
}

func parse(encodedHash string) (parameters, []byte, []byte, error) {
	parts := strings.Split(encodedHash, "$")
	if len(parts) != 6 || parts[1] != "argon2id" || parts[2] != "v=19" {
		return parameters{}, nil, nil, ErrInvalidHash
	}

	parsed := parameters{keyLength: keyLength}
	for _, parameter := range strings.Split(parts[3], ",") {
		keyValue := strings.SplitN(parameter, "=", 2)
		if len(keyValue) != 2 {
			return parameters{}, nil, nil, ErrInvalidHash
		}

		switch keyValue[0] {
		case "m":
			value, err := strconv.ParseUint(keyValue[1], 10, 32)
			if err != nil {
				return parameters{}, nil, nil, ErrInvalidHash
			}
			parsed.memory = uint32(value)
		case "t":
			value, err := strconv.ParseUint(keyValue[1], 10, 32)
			if err != nil {
				return parameters{}, nil, nil, ErrInvalidHash
			}
			parsed.iterations = uint32(value)
		case "p":
			value, err := strconv.ParseUint(keyValue[1], 10, 8)
			if err != nil {
				return parameters{}, nil, nil, ErrInvalidHash
			}
			parsed.parallelism = uint8(value)
		default:
			return parameters{}, nil, nil, ErrInvalidHash
		}
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil || len(salt) == 0 {
		return parameters{}, nil, nil, ErrInvalidHash
	}
	expectedKey, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil || len(expectedKey) == 0 {
		return parameters{}, nil, nil, ErrInvalidHash
	}
	if parsed.memory == 0 || parsed.iterations == 0 || parsed.parallelism == 0 {
		return parameters{}, nil, nil, ErrInvalidHash
	}
	parsed.keyLength = uint32(len(expectedKey))

	return parsed, salt, expectedKey, nil
}
