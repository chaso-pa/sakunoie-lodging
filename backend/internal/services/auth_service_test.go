package services

import (
	"context"
	"errors"
	"io"
	"regexp"
	"strings"
	"testing"

	"github.com/chaso-pa/sakunoie-lodging-backend/internal/models"
	"github.com/chaso-pa/sakunoie-lodging-backend/internal/storage"
	"github.com/chaso-pa/sakunoie-lodging-backend/internal/testutil"
)

func TestVerifyInvite(t *testing.T) {
	s := NewAuthService(nil, "SAKU-26-ABCD", storage.NotConfigured{})

	cases := []struct {
		code string
		want bool
	}{
		{"SAKU-26-ABCD", true},
		{"saku-26-abcd", true},
		{"  Saku-26-Abcd \n", true},
		{"SAKU-26-ABCE", false},
		{"SAKU26ABCD", false},
		{"", false},
	}
	for _, c := range cases {
		if got := s.VerifyInvite(c.code); got != c.want {
			t.Errorf("VerifyInvite(%q) = %v, want %v", c.code, got, c.want)
		}
	}
}

func TestVerifyInviteRejectsEverythingWhenUnconfigured(t *testing.T) {
	s := NewAuthService(nil, "", storage.NotConfigured{})
	if s.VerifyInvite("") || s.VerifyInvite("anything") {
		t.Fatal("an empty configured invite code must not match any input")
	}
}

func TestGenerateTransferCode(t *testing.T) {
	pattern := regexp.MustCompile(`^[A-Z0-9]{12}$`)
	seen := map[string]bool{}
	for range 200 {
		code, err := GenerateTransferCode()
		if err != nil {
			t.Fatal(err)
		}
		if !pattern.MatchString(code) {
			t.Fatalf("code %q is not 12 characters of A-Z0-9", code)
		}
		if seen[code] {
			t.Fatalf("code %q generated twice", code)
		}
		seen[code] = true
	}
}

func TestFormatTransferCode(t *testing.T) {
	if got := FormatTransferCode("ABCD1234WXYZ"); got != "ABCD-1234-WXYZ" {
		t.Fatalf("got %q", got)
	}
}

func TestTransferCodeNormalizationAndHash(t *testing.T) {
	const raw = "ABCD1234WXYZ"
	for _, input := range []string{"ABCD-1234-WXYZ", "abcd-1234-wxyz", " abcd 1234 wxyz ", "AbCd1234wXyZ", "ABCD-1234-\tWXYZ\n"} {
		if got := NormalizeTransferCode(input); got != raw {
			t.Errorf("NormalizeTransferCode(%q) = %q, want %q", input, got, raw)
		}
		if HashTransferCode(input) != HashTransferCode(raw) {
			t.Errorf("HashTransferCode(%q) differs from the hash of the raw code", input)
		}
	}

	hash := HashTransferCode(raw)
	if len(hash) != 64 || strings.ToLower(hash) != hash {
		t.Errorf("hash %q is not lowercase SHA-256 hex", hash)
	}
	if HashTransferCode("ABCD1234WXYA") == hash {
		t.Error("different codes must hash differently")
	}
}

func TestValidateNickname(t *testing.T) {
	if got, err := ValidateNickname("  旅するMika  "); err != nil || got != "旅するMika" {
		t.Errorf("got %q, %v", got, err)
	}
	// The limit counts characters, not bytes.
	if _, err := ValidateNickname(strings.Repeat("あ", 30)); err != nil {
		t.Errorf("30 characters should be accepted: %v", err)
	}
	for _, bad := range []string{"", "   ", strings.Repeat("あ", 31)} {
		if _, err := ValidateNickname(bad); !errors.Is(err, ErrInvalidNickname) {
			t.Errorf("ValidateNickname(%q) error = %v, want ErrInvalidNickname", bad, err)
		}
	}
}

func TestValidateLanguage(t *testing.T) {
	if got, _ := ValidateLanguage(""); got != "ja" {
		t.Errorf("empty language should default to ja, got %q", got)
	}
	for _, lang := range []string{"ja", "en", "ko", "zh"} {
		if got, err := ValidateLanguage(lang); err != nil || got != lang {
			t.Errorf("ValidateLanguage(%q) = %q, %v", lang, got, err)
		}
	}
	if _, err := ValidateLanguage("fr"); !errors.Is(err, ErrInvalidLanguage) {
		t.Errorf("error = %v, want ErrInvalidLanguage", err)
	}
}

type fakeUploader struct{ calls int }

func (f *fakeUploader) UploadImage(_ context.Context, kind string, r io.Reader) (string, error) {
	f.calls++
	io.Copy(io.Discard, r)
	return "http://example.test/" + kind + "/a.jpg", nil
}

func (f *fakeUploader) UploadImages(ctx context.Context, kind string, images []io.Reader) ([]string, error) {
	urls := make([]string, len(images))
	for i, r := range images {
		urls[i], _ = f.UploadImage(ctx, kind, r)
	}
	return urls, nil
}

func (f *fakeUploader) DeleteImage(context.Context, string) error { return nil }

func TestSignupAndRestoreIntegration(t *testing.T) {
	db := testutil.NewDB(t, "services").Gorm
	ctx := context.Background()
	uploader := &fakeUploader{}
	s := NewAuthService(db, "SAKU-26-ABCD", uploader)

	member, code, err := s.Signup(ctx, " Mika ", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	if member.ID == "" || member.Nickname != "Mika" || member.PreferredLanguage != "ja" || member.SessionVersion != 1 || member.AvatarURL != nil {
		t.Fatalf("unexpected member: %+v", member)
	}
	if !regexp.MustCompile(`^[A-Z0-9]{4}-[A-Z0-9]{4}-[A-Z0-9]{4}$`).MatchString(code) {
		t.Fatalf("transfer code %q is not XXXX-XXXX-XXXX", code)
	}
	if uploader.calls != 0 {
		t.Fatal("uploader must not be called without an avatar")
	}

	// Only the hash is stored.
	var stored models.Member
	if err := db.First(&stored, "id = ?", member.ID).Error; err != nil {
		t.Fatal(err)
	}
	if stored.TransferCodeHash != HashTransferCode(code) || strings.Contains(stored.TransferCodeHash, NormalizeTransferCode(code)) {
		t.Fatalf("stored hash %q does not match", stored.TransferCodeHash)
	}

	// Restoring accepts the lowercase, unhyphenated form and bumps the version.
	restored, err := s.RestoreByTransferCode(ctx, strings.ToLower(strings.ReplaceAll(code, "-", " ")))
	if err != nil {
		t.Fatal(err)
	}
	if restored.ID != member.ID || restored.SessionVersion != 2 {
		t.Fatalf("restored id %q version %d, want %q version 2", restored.ID, restored.SessionVersion, member.ID)
	}
	if restored, err = s.RestoreByTransferCode(ctx, code); err != nil || restored.SessionVersion != 3 {
		t.Fatalf("second restore: version %v, err %v", restored, err)
	}

	for _, bad := range []string{"AAAA-BBBB-CCCC", "", "short", code + "X", "ＡＢＣＤ-1234-WXYZ"} {
		if _, err := s.RestoreByTransferCode(ctx, bad); !errors.Is(err, ErrInvalidTransferCode) {
			t.Errorf("RestoreByTransferCode(%q) error = %v, want ErrInvalidTransferCode", bad, err)
		}
	}

	withAvatar, _, err := s.Signup(ctx, "Ken", "en", strings.NewReader("image bytes"))
	if err != nil {
		t.Fatal(err)
	}
	if withAvatar.AvatarURL == nil || *withAvatar.AvatarURL != "http://example.test/avatars/a.jpg" || uploader.calls != 1 {
		t.Fatalf("avatar url %v, uploader calls %d", withAvatar.AvatarURL, uploader.calls)
	}

	if _, err := s.GetMember(ctx, "no-such-id"); !errors.Is(err, ErrMemberNotFound) {
		t.Errorf("GetMember error = %v, want ErrMemberNotFound", err)
	}
}

func TestSignupWithAvatarFailsWithoutStorageIntegration(t *testing.T) {
	db := testutil.NewDB(t, "services_storage").Gorm
	s := NewAuthService(db, "SAKU-26-ABCD", storage.NotConfigured{})

	_, _, err := s.Signup(context.Background(), "Mika", "ja", strings.NewReader("image bytes"))
	if !errors.Is(err, storage.ErrNotConfigured) {
		t.Fatalf("error = %v, want storage.ErrNotConfigured", err)
	}
	var count int64
	db.Model(&models.Member{}).Count(&count)
	if count != 0 {
		t.Fatalf("%d members created, want none when the avatar cannot be stored", count)
	}
}

func TestSignupRejectsInvalidInput(t *testing.T) {
	s := NewAuthService(nil, "SAKU-26-ABCD", storage.NotConfigured{})
	if _, _, err := s.Signup(context.Background(), "  ", "ja", nil); !errors.Is(err, ErrInvalidNickname) {
		t.Errorf("error = %v, want ErrInvalidNickname", err)
	}
	if _, _, err := s.Signup(context.Background(), "Mika", "fr", nil); !errors.Is(err, ErrInvalidLanguage) {
		t.Errorf("error = %v, want ErrInvalidLanguage", err)
	}
}
