package services

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"math/big"
	"strings"
	"unicode/utf8"

	"github.com/chaso-pa/sakunoie-lodging-backend/internal/models"
	"github.com/chaso-pa/sakunoie-lodging-backend/internal/storage"
	"gorm.io/gorm"
)

const (
	transferCodeAlphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	transferCodeLength   = 12
	maxNicknameLength    = 30
	defaultLanguage      = "ja"
)

var supportedLanguages = map[string]bool{"ja": true, "en": true, "ko": true, "zh": true}

var (
	ErrInvalidNickname     = errors.New("nickname must be 1 to 30 characters")
	ErrInvalidLanguage     = errors.New("language must be one of ja, en, ko, zh")
	ErrInvalidTransferCode = errors.New("invalid transfer code")
	ErrMemberNotFound      = errors.New("member not found")
)

type AuthService struct {
	db         *gorm.DB
	inviteCode string
	images     storage.ImageUploader
}

func NewAuthService(db *gorm.DB, inviteCode string, images storage.ImageUploader) *AuthService {
	return &AuthService{db: db, inviteCode: strings.TrimSpace(inviteCode), images: images}
}

// VerifyInvite reports whether code matches the configured invite code,
// ignoring case and surrounding whitespace.
func (s *AuthService) VerifyInvite(code string) bool {
	if s.inviteCode == "" {
		return false
	}
	got := strings.ToUpper(strings.TrimSpace(code))
	want := strings.ToUpper(s.inviteCode)
	return subtle.ConstantTimeCompare([]byte(got), []byte(want)) == 1
}

// GenerateTransferCode returns 12 random characters from A-Z0-9.
func GenerateTransferCode() (string, error) {
	n := big.NewInt(int64(len(transferCodeAlphabet)))
	b := make([]byte, transferCodeLength)
	for i := range b {
		idx, err := rand.Int(rand.Reader, n)
		if err != nil {
			return "", fmt.Errorf("generate transfer code: %w", err)
		}
		b[i] = transferCodeAlphabet[idx.Int64()]
	}
	return string(b), nil
}

// FormatTransferCode renders a raw code as XXXX-XXXX-XXXX.
func FormatTransferCode(raw string) string {
	var parts []string
	for len(raw) > 4 {
		parts = append(parts, raw[:4])
		raw = raw[4:]
	}
	parts = append(parts, raw)
	return strings.Join(parts, "-")
}

// NormalizeTransferCode strips hyphens and whitespace and upper-cases the
// input, so a code can be typed in any of those forms.
func NormalizeTransferCode(input string) string {
	var b strings.Builder
	for _, r := range input {
		switch r {
		case '-', ' ', '\t', '\n', '\r', '　':
			continue
		}
		b.WriteRune(r)
	}
	return strings.ToUpper(b.String())
}

// HashTransferCode returns the SHA-256 hex digest stored in
// members.transfer_code_hash. The input is normalized first.
func HashTransferCode(input string) string {
	sum := sha256.Sum256([]byte(NormalizeTransferCode(input)))
	return hex.EncodeToString(sum[:])
}

func isValidTransferCode(normalized string) bool {
	if len(normalized) != transferCodeLength {
		return false
	}
	for _, r := range normalized {
		if !strings.ContainsRune(transferCodeAlphabet, r) {
			return false
		}
	}
	return true
}

// ValidateNickname trims the nickname and checks its length in characters.
func ValidateNickname(nickname string) (string, error) {
	trimmed := strings.TrimSpace(nickname)
	if trimmed == "" || utf8.RuneCountInString(trimmed) > maxNicknameLength {
		return "", ErrInvalidNickname
	}
	return trimmed, nil
}

// ValidateLanguage returns the language, defaulting to ja when empty.
func ValidateLanguage(language string) (string, error) {
	if language == "" {
		return defaultLanguage, nil
	}
	if !supportedLanguages[language] {
		return "", ErrInvalidLanguage
	}
	return language, nil
}

// Signup creates a member and returns it with its transfer code, formatted
// for display. The code is not stored and cannot be shown again. avatar may
// be nil. The caller must have checked the invite_verified session flag.
func (s *AuthService) Signup(ctx context.Context, nickname, language string, avatar io.Reader) (*models.Member, string, error) {
	nickname, err := ValidateNickname(nickname)
	if err != nil {
		return nil, "", err
	}
	language, err = ValidateLanguage(language)
	if err != nil {
		return nil, "", err
	}

	var avatarURL *string
	if avatar != nil {
		url, err := s.images.UploadImage(ctx, storage.KindAvatar, avatar)
		if err != nil {
			return nil, "", fmt.Errorf("upload avatar: %w", err)
		}
		avatarURL = &url
	}

	code, err := GenerateTransferCode()
	if err != nil {
		return nil, "", err
	}

	member := &models.Member{
		Nickname:          nickname,
		AvatarURL:         avatarURL,
		PreferredLanguage: language,
		TransferCodeHash:  HashTransferCode(code),
		SessionVersion:    1,
	}
	if err := s.db.WithContext(ctx).Create(member).Error; err != nil {
		return nil, "", fmt.Errorf("create member: %w", err)
	}
	return member, FormatTransferCode(code), nil
}

// RestoreByTransferCode finds the member owning the code and increments its
// session_version, which invalidates every session issued before this call.
// The returned member carries the new version.
func (s *AuthService) RestoreByTransferCode(ctx context.Context, input string) (*models.Member, error) {
	if !isValidTransferCode(NormalizeTransferCode(input)) {
		return nil, ErrInvalidTransferCode
	}
	hash := HashTransferCode(input)

	var member models.Member
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		res := tx.Model(&models.Member{}).
			Where("transfer_code_hash = ?", hash).
			UpdateColumn("session_version", gorm.Expr("session_version + 1"))
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return ErrInvalidTransferCode
		}
		return tx.Where("transfer_code_hash = ?", hash).First(&member).Error
	})
	if err != nil {
		if errors.Is(err, ErrInvalidTransferCode) {
			return nil, err
		}
		return nil, fmt.Errorf("restore member: %w", err)
	}
	return &member, nil
}

// GetMember loads a member by id.
func (s *AuthService) GetMember(ctx context.Context, id string) (*models.Member, error) {
	var member models.Member
	if err := s.db.WithContext(ctx).Where("id = ?", id).First(&member).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrMemberNotFound
		}
		return nil, fmt.Errorf("get member: %w", err)
	}
	return &member, nil
}
