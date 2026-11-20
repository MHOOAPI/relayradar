// Package auth implements password hashing without external dependencies.
package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
)

const iterations = 120000

// HashPassword returns pbkdf2$<iter>$<salt>$<hash>.
func HashPassword(password string) string {
	salt := make([]byte, 16)
	_, _ = rand.Read(salt)
	sum := pbkdf2([]byte(password), salt, iterations, 32)
	return fmt.Sprintf("pbkdf2$%d$%s$%s", iterations, hex.EncodeToString(salt), hex.EncodeToString(sum))
}

// VerifyPassword checks a password against a stored hash in constant time.
func VerifyPassword(password, stored string) bool {
	parts := strings.Split(stored, "$")
	if len(parts) != 4 || parts[0] != "pbkdf2" {
		return false
	}
	iter, err := strconv.Atoi(parts[1])
	if err != nil || iter <= 0 {
		return false
	}
	salt, err := hex.DecodeString(parts[2])
	if err != nil {
		return false
	}
	want, err := hex.DecodeString(parts[3])
	if err != nil {
		return false
	}
	got := pbkdf2([]byte(password), salt, iter, len(want))
	return subtle.ConstantTimeCompare(got, want) == 1
}

// pbkdf2 is PBKDF2-HMAC-SHA256 (RFC 8018) for one output block; 32 bytes fits
// exactly in one block, which is all this service needs.
func pbkdf2(password, salt []byte, iter, keyLen int) []byte {
	mac := hmac.New(sha256.New, password)
	mac.Write(salt)
	mac.Write([]byte{0, 0, 0, 1})
	u := mac.Sum(nil)
	out := make([]byte, len(u))
	copy(out, u)
	for i := 1; i < iter; i++ {
		mac.Reset()
		mac.Write(u)
		u = mac.Sum(nil)
		for j := range out {
			out[j] ^= u[j]
		}
	}
	if keyLen < len(out) {
		return out[:keyLen]
	}
	return out
}
