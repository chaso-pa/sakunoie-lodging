package utils

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"unicode"
)

// Passport numbers are encrypted with AES-256-GCM. Stored format:
//
//	keyID (1 byte) || nonce (12 bytes) || ciphertext || tag (16 bytes)
//
// The key ID lets old rows decrypt after the active key rotates. The caller
// passes the guest's public_id as additional authenticated data, so a
// ciphertext copied to another row fails to decrypt.

const (
	passportKeyLen   = 32
	passportNonceLen = 12
)

var ErrPassportDecrypt = errors.New("passport: decryption failed")

// PassportCipher encrypts and decrypts passport numbers.
type PassportCipher struct {
	aeads  map[byte]cipher.AEAD
	active byte
}

// ParsePassportKeys parses "1:<base64>,2:<base64>". Each key must decode to
// 32 bytes. Error messages never include key material.
func ParsePassportKeys(s string) (map[byte][]byte, error) {
	keys := map[byte][]byte{}
	for part := range strings.SplitSeq(s, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		idStr, b64, ok := strings.Cut(part, ":")
		if !ok {
			return nil, errors.New("expected <id>:<base64> entries")
		}
		id, err := strconv.ParseUint(strings.TrimSpace(idStr), 10, 8)
		if err != nil || id == 0 {
			return nil, errors.New("key id must be an integer from 1 to 255")
		}
		key, err := base64.StdEncoding.DecodeString(strings.TrimSpace(b64))
		if err != nil {
			return nil, fmt.Errorf("key %d: invalid base64", id)
		}
		if len(key) != passportKeyLen {
			return nil, fmt.Errorf("key %d: must be %d bytes after base64 decoding", id, passportKeyLen)
		}
		if _, dup := keys[byte(id)]; dup {
			return nil, fmt.Errorf("key %d: duplicate id", id)
		}
		keys[byte(id)] = key
	}
	if len(keys) == 0 {
		return nil, errors.New("no keys")
	}
	return keys, nil
}

// NewPassportCipher builds a cipher; active must be one of the key IDs.
func NewPassportCipher(keys map[byte][]byte, active byte) (*PassportCipher, error) {
	if _, ok := keys[active]; !ok {
		return nil, fmt.Errorf("active key id %d not found", active)
	}
	c := &PassportCipher{aeads: map[byte]cipher.AEAD{}, active: active}
	for id, key := range keys {
		block, err := aes.NewCipher(key)
		if err != nil {
			return nil, fmt.Errorf("key %d: %w", id, err)
		}
		aead, err := cipher.NewGCM(block)
		if err != nil {
			return nil, fmt.Errorf("key %d: %w", id, err)
		}
		c.aeads[id] = aead
	}
	return c, nil
}

// Encrypt encrypts the plaintext with the active key.
func (c *PassportCipher) Encrypt(plaintext string, aad []byte) ([]byte, error) {
	aead := c.aeads[c.active]
	out := make([]byte, 1+passportNonceLen, 1+passportNonceLen+len(plaintext)+aead.Overhead())
	out[0] = c.active
	if _, err := rand.Read(out[1 : 1+passportNonceLen]); err != nil {
		return nil, err
	}
	return aead.Seal(out, out[1:1+passportNonceLen], []byte(plaintext), aad), nil
}

// Decrypt decrypts a value produced by Encrypt with any configured key.
func (c *PassportCipher) Decrypt(blob []byte, aad []byte) (string, error) {
	if len(blob) < 1+passportNonceLen+16 {
		return "", ErrPassportDecrypt
	}
	aead, ok := c.aeads[blob[0]]
	if !ok {
		return "", ErrPassportDecrypt
	}
	pt, err := aead.Open(nil, blob[1:1+passportNonceLen], blob[1+passportNonceLen:], aad)
	if err != nil {
		return "", ErrPassportDecrypt
	}
	return string(pt), nil
}

// NormalizePassport uppercases and removes all whitespace.
func NormalizePassport(s string) string {
	var b strings.Builder
	for _, r := range s {
		if unicode.IsSpace(r) {
			continue
		}
		b.WriteRune(unicode.ToUpper(r))
	}
	return b.String()
}

// PassportLast4 returns the last four characters of a normalized number.
func PassportLast4(normalized string) string {
	r := []rune(normalized)
	if len(r) <= 4 {
		return string(r)
	}
	return string(r[len(r)-4:])
}
