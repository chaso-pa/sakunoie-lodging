package utils

import (
	"bytes"
	"encoding/base64"
	"strings"
	"testing"
)

func testKey(b byte) []byte { return bytes.Repeat([]byte{b}, 32) }

func TestPassportRoundTrip(t *testing.T) {
	c, err := NewPassportCipher(map[byte][]byte{1: testKey(1)}, 1)
	if err != nil {
		t.Fatal(err)
	}
	aad := []byte("guest-public-id")
	ct, err := c.Encrypt("TESTPP0001", aad)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(ct, []byte("TESTPP0001")) {
		t.Fatal("ciphertext contains plaintext")
	}
	if len(ct) < 28 || ct[0] != 1 {
		t.Fatalf("unexpected format len=%d id=%d", len(ct), ct[0])
	}
	pt, err := c.Decrypt(ct, aad)
	if err != nil || pt != "TESTPP0001" {
		t.Fatalf("got %q %v", pt, err)
	}
}

func TestPassportTamper(t *testing.T) {
	c, _ := NewPassportCipher(map[byte][]byte{1: testKey(1)}, 1)
	aad := []byte("guest-a")
	ct, _ := c.Encrypt("TESTPP0001", aad)

	for name, mutate := range map[string]func([]byte){
		"nonce":      func(b []byte) { b[2] ^= 1 },
		"ciphertext": func(b []byte) { b[14] ^= 1 },
		"tag":        func(b []byte) { b[len(b)-1] ^= 1 },
		"key_id":     func(b []byte) { b[0] = 9 },
	} {
		b := bytes.Clone(ct)
		mutate(b)
		if _, err := c.Decrypt(b, aad); err == nil {
			t.Errorf("%s: tampered ciphertext decrypted", name)
		}
	}
	if _, err := c.Decrypt(ct, []byte("guest-b")); err == nil {
		t.Error("wrong AAD decrypted")
	}
	if _, err := c.Decrypt(ct[:10], aad); err == nil {
		t.Error("short blob decrypted")
	}
	other, _ := NewPassportCipher(map[byte][]byte{1: testKey(2)}, 1)
	if _, err := other.Decrypt(ct, aad); err == nil {
		t.Error("wrong key decrypted")
	}
}

func TestPassportKeyRotation(t *testing.T) {
	v1, _ := NewPassportCipher(map[byte][]byte{1: testKey(1)}, 1)
	ct1, _ := v1.Encrypt("TESTPP0001", nil)

	v2, err := NewPassportCipher(map[byte][]byte{1: testKey(1), 2: testKey(2)}, 2)
	if err != nil {
		t.Fatal(err)
	}
	if pt, err := v2.Decrypt(ct1, nil); err != nil || pt != "TESTPP0001" {
		t.Fatalf("v1 ciphertext with v2 active: %q %v", pt, err)
	}
	ct2, _ := v2.Encrypt("TESTPP0002", nil)
	if ct2[0] != 2 {
		t.Fatalf("new ciphertext must use active key 2, got %d", ct2[0])
	}
	if _, err := NewPassportCipher(map[byte][]byte{1: testKey(1)}, 3); err == nil {
		t.Fatal("missing active key accepted")
	}
}

func TestParsePassportKeys(t *testing.T) {
	k := base64.StdEncoding.EncodeToString(testKey(7))
	keys, err := ParsePassportKeys("1:" + k + ", 2:" + k)
	if err != nil || len(keys) != 2 {
		t.Fatalf("got %d keys, %v", len(keys), err)
	}
	short := base64.StdEncoding.EncodeToString([]byte("short"))
	for _, bad := range []string{"", "1", "x:" + k, "0:" + k, "1:" + short, "1:!!", "1:" + k + ",1:" + k} {
		if _, err := ParsePassportKeys(bad); err == nil {
			t.Errorf("accepted %q", bad)
		} else if strings.Contains(err.Error(), k) {
			t.Errorf("error leaks key material")
		}
	}
}

func TestNormalizePassport(t *testing.T) {
	if got := NormalizePassport(" tk 12 34567 "); got != "TK1234567" {
		t.Fatalf("got %q", got)
	}
	if got := PassportLast4("TK1234567"); got != "4567" {
		t.Fatalf("got %q", got)
	}
	if got := PassportLast4("AB1"); got != "AB1" {
		t.Fatalf("got %q", got)
	}
}
