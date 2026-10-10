package usecase

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/google/uuid"
	"github.com/meta-super-app/backend/internal/domain"
)

var (
	ErrInvalidReplySetCode = errors.New("reply set code must be 2-60 characters and use only A-Z, 0-9, hyphen or underscore")
	ErrReplySetCodeExists  = errors.New("reply set code already exists in this workspace")
	ErrReplySetNameExists  = errors.New("reply set name already exists in this workspace")
)

var replySetCodePattern = regexp.MustCompile(`^[A-Z0-9][A-Z0-9_-]{1,59}$`)

type Reply struct {
	repo domain.ReplyRepository
}

func NewReply(repo domain.ReplyRepository) *Reply {
	return &Reply{repo: repo}
}

func (u *Reply) CreateSet(ctx context.Context, accountID, code, name string) (*domain.ReplySet, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("set name cannot be empty")
	}
	if err := u.ensureNameAvailable(ctx, accountID, name, ""); err != nil {
		return nil, err
	}
	code = normalizeReplySetCode(code)
	if code == "" {
		code = generatedReplySetCode(name)
	}
	if !replySetCodePattern.MatchString(code) {
		return nil, ErrInvalidReplySetCode
	}
	if err := u.ensureCodeAvailable(ctx, accountID, code, ""); err != nil {
		return nil, err
	}

	set := &domain.ReplySet{
		AccountID: accountID,
		Code:      code,
		Name:      name,
	}

	if err := u.repo.CreateSet(ctx, set); err != nil {
		return nil, err
	}

	return set, nil
}

func (u *Reply) GetSets(ctx context.Context, accountID string) ([]domain.ReplySet, error) {
	return u.repo.GetSetsByAccountID(ctx, accountID)
}

// FindSetByTrigger resolves a customer-entered Reply Set name. Matching is
// exact after trimming whitespace and is case-insensitive, so a name never
// fires on an unrelated sentence or partial word. Code remains a backwards-
// compatible alias for Reply Sets created by earlier releases.
func (u *Reply) FindSetByTrigger(ctx context.Context, accountID, trigger string) (*domain.ReplySet, error) {
	trigger = strings.TrimSpace(trigger)
	if trigger == "" {
		return nil, nil
	}
	sets, err := u.repo.GetSetsByAccountID(ctx, accountID)
	if err != nil {
		return nil, err
	}
	var matched *domain.ReplySet
	for index := range sets {
		if strings.EqualFold(strings.TrimSpace(sets[index].Name), trigger) {
			if matched != nil {
				return nil, ErrReplySetNameExists
			}
			set := sets[index]
			matched = &set
		}
	}
	if matched != nil {
		return matched, nil
	}
	// Backwards compatibility for customers who already received an old code.
	for index := range sets {
		if strings.EqualFold(sets[index].Code, trigger) {
			set := sets[index]
			return &set, nil
		}
	}
	return nil, nil
}

func (u *Reply) GetSet(ctx context.Context, setID string, accountID string) (*domain.ReplySet, error) {
	set, err := u.repo.GetSetByID(ctx, setID)
	if err != nil {
		return nil, err
	}

	// Ensure the set belongs to this account
	if set.AccountID != accountID {
		return nil, fmt.Errorf("unauthorized to access this set")
	}

	return set, nil
}

func (u *Reply) UpdateSet(ctx context.Context, setID string, accountID string, code string, name string) error {
	set, err := u.GetSet(ctx, setID, accountID)
	if err != nil {
		return err
	}

	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("set name cannot be empty")
	}
	if err := u.ensureNameAvailable(ctx, accountID, name, setID); err != nil {
		return err
	}
	code = normalizeReplySetCode(code)
	if code == "" {
		code = set.Code
	}
	if code == "" {
		code = generatedReplySetCode(name)
	}
	if !replySetCodePattern.MatchString(code) {
		return ErrInvalidReplySetCode
	}
	if err := u.ensureCodeAvailable(ctx, accountID, code, setID); err != nil {
		return err
	}

	set.Code = code
	set.Name = name
	return u.repo.UpdateSet(ctx, set)
}

func normalizeReplySetCode(code string) string {
	return strings.ToUpper(strings.TrimSpace(code))
}

func generatedReplySetCode(name string) string {
	var builder strings.Builder
	lastSeparator := false
	for _, char := range strings.ToUpper(name) {
		if (char >= 'A' && char <= 'Z') || (char >= '0' && char <= '9') {
			builder.WriteRune(char)
			lastSeparator = false
		} else if builder.Len() > 0 && !lastSeparator {
			builder.WriteByte('-')
			lastSeparator = true
		}
	}
	base := strings.Trim(builder.String(), "-")
	if len(base) > 46 {
		base = strings.TrimRight(base[:46], "-")
	}
	if len(base) < 2 {
		base = "REPLY"
	}
	return base + "-" + strings.ToUpper(uuid.NewString()[:8])
}

func (u *Reply) ensureCodeAvailable(ctx context.Context, accountID, code, excludeID string) error {
	sets, err := u.repo.GetSetsByAccountID(ctx, accountID)
	if err != nil {
		return err
	}
	for _, set := range sets {
		if set.ID != excludeID && strings.EqualFold(set.Code, code) {
			return ErrReplySetCodeExists
		}
	}
	return nil
}

func (u *Reply) ensureNameAvailable(ctx context.Context, accountID, name, excludeID string) error {
	sets, err := u.repo.GetSetsByAccountID(ctx, accountID)
	if err != nil {
		return err
	}
	for _, set := range sets {
		if set.ID != excludeID && strings.EqualFold(strings.TrimSpace(set.Name), name) {
			return ErrReplySetNameExists
		}
	}
	return nil
}

func (u *Reply) DeleteSet(ctx context.Context, setID string, accountID string) error {
	_, err := u.GetSet(ctx, setID, accountID) // check permission
	if err != nil {
		return err
	}

	return u.repo.DeleteSet(ctx, setID)
}

func (u *Reply) UpdateItems(ctx context.Context, setID string, accountID string, items []domain.ReplyItem) error {
	_, err := u.GetSet(ctx, setID, accountID) // check permission
	if err != nil {
		return err
	}

	// Validate items
	for i, item := range items {
		if item.Type != "text" && item.Type != "image" && item.Type != "video" && item.Type != "audio" {
			return fmt.Errorf("invalid item type at index %d", i)
		}
		if item.IsEnabled && strings.TrimSpace(item.Content) == "" {
			return fmt.Errorf("empty content at index %d", i)
		}
	}

	return u.repo.UpdateItems(ctx, setID, items)
}

func (u *Reply) GetReplySetItems(ctx context.Context, setID string, accountID string) ([]domain.ReplyItem, error) {
	set, err := u.GetSet(ctx, setID, accountID)
	if err != nil {
		return nil, err
	}
	return set.Items, nil
}
