// Package encoding provides collision-safe Base62 encoding for short code generation.
package encoding

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"strings"
)

const (
	// alphabet is the Base62 character set: [0-9a-zA-Z].
	alphabet = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	base     = int64(len(alphabet))

	// DefaultCodeLength is the default short code length.
	DefaultCodeLength = 7
)

// Encoder generates and decodes Base62 short codes.
type Encoder struct {
	codeLength int
}

// NewEncoder creates an Encoder with the specified code length.
func NewEncoder(codeLength int) *Encoder {
	if codeLength <= 0 {
		codeLength = DefaultCodeLength
	}
	return &Encoder{codeLength: codeLength}
}

// Encode converts a numeric ID into a Base62 string.
// The result is left-padded to ensure consistent code length.
func (e *Encoder) Encode(id int64) string {
	if id == 0 {
		return strings.Repeat(string(alphabet[0]), e.codeLength)
	}

	var encoded strings.Builder
	n := id
	if n < 0 {
		n = -n
	}

	for n > 0 {
		remainder := n % base
		encoded.WriteByte(alphabet[remainder])
		n /= base
	}

	// Reverse the string.
	result := []byte(encoded.String())
	for i, j := 0, len(result)-1; i < j; i, j = i+1, j-1 {
		result[i], result[j] = result[j], result[i]
	}

	// Left-pad to code length.
	code := string(result)
	if len(code) < e.codeLength {
		code = strings.Repeat(string(alphabet[0]), e.codeLength-len(code)) + code
	}

	return code
}

// Decode converts a Base62 string back to a numeric value.
func (e *Encoder) Decode(code string) (int64, error) {
	var result int64
	for _, c := range code {
		idx := strings.IndexRune(alphabet, c)
		if idx < 0 {
			return 0, fmt.Errorf("invalid character '%c' in short code", c)
		}
		result = result*base + int64(idx)
	}
	return result, nil
}

// GenerateRandom creates a cryptographically random Base62 code.
// Used as a fallback when sequential IDs are undesirable.
func (e *Encoder) GenerateRandom() (string, error) {
	result := make([]byte, e.codeLength)
	max := big.NewInt(base)

	for i := range result {
		n, err := rand.Int(rand.Reader, max)
		if err != nil {
			return "", fmt.Errorf("crypto/rand failed: %w", err)
		}
		result[i] = alphabet[n.Int64()]
	}

	return string(result), nil
}

// IsValidCode checks whether a string is a valid Base62 code of the expected length.
func (e *Encoder) IsValidCode(code string) bool {
	if len(code) != e.codeLength {
		return false
	}
	for _, c := range code {
		if strings.IndexRune(alphabet, c) < 0 {
			return false
		}
	}
	return true
}
