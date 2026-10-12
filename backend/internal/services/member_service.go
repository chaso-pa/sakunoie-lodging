package services

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"

	"github.com/chaso-pa/sakunoie-lodging-backend/internal/models"
	"github.com/chaso-pa/sakunoie-lodging-backend/internal/storage"
	"gorm.io/gorm"
)

type MemberService struct {
	db     *gorm.DB
	images storage.ImageUploader
}

func NewMemberService(db *gorm.DB, images storage.ImageUploader) *MemberService {
	return &MemberService{db: db, images: images}
}

// UpdateProfile changes memberID's own nickname, language and avatar. An
// empty nickname or language and a nil avatar leave that field unchanged.
// There is no way to name another member here: the id always comes from the
// session.
func (s *MemberService) UpdateProfile(ctx context.Context, memberID, nickname, language string, avatar io.Reader) (*models.Member, error) {
	var member models.Member
	if err := s.db.WithContext(ctx).Where("id = ?", memberID).First(&member).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrMemberNotFound
		}
		return nil, fmt.Errorf("get member: %w", err)
	}

	updates := map[string]any{}
	if nickname != "" {
		valid, err := ValidateNickname(nickname)
		if err != nil {
			return nil, err
		}
		updates["nickname"] = valid
	}
	if language != "" {
		valid, err := ValidateLanguage(language)
		if err != nil {
			return nil, err
		}
		updates["preferred_language"] = valid
	}

	var newAvatar string
	if avatar != nil {
		url, err := s.images.UploadImage(ctx, storage.KindAvatar, avatar)
		if err != nil {
			return nil, fmt.Errorf("upload avatar: %w", err)
		}
		newAvatar = url
		updates["avatar_url"] = url
	}
	if len(updates) == 0 {
		return &member, nil
	}

	// Copied by value: Updates writes the new URL through member.AvatarURL.
	var oldAvatar string
	if member.AvatarURL != nil {
		oldAvatar = *member.AvatarURL
	}
	if err := s.db.WithContext(ctx).Model(&models.Member{}).Where("id = ?", memberID).Updates(updates).Error; err != nil {
		if newAvatar != "" {
			s.removeImage(ctx, newAvatar)
		}
		return nil, fmt.Errorf("update member: %w", err)
	}
	if newAvatar != "" && oldAvatar != "" && oldAvatar != newAvatar {
		s.removeImage(ctx, oldAvatar)
	}
	if err := s.db.WithContext(ctx).Where("id = ?", memberID).First(&member).Error; err != nil {
		return nil, fmt.Errorf("reload member: %w", err)
	}
	return &member, nil
}

func (s *MemberService) removeImage(ctx context.Context, url string) {
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), cleanupTimeout)
	defer cancel()
	if err := s.images.DeleteImage(ctx, url); err != nil {
		slog.WarnContext(ctx, "stored image was not deleted", "url", url, "error", err)
	}
}
