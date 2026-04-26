package identity

import (
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

type Identity struct {
	ID         uuid.UUID `json:"id"`
	FullName   string    `json:"full_name"`
	Phone      string    `json:"phone,omitempty"`
	Address    string    `json:"address,omitempty"`
	Email      string    `json:"email,omitempty"`
	PassportNo string    `json:"passport_no,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

var (
	ErrFullNameRequired = errors.New("full_name is required")
	ErrEmailInvalid     = errors.New("email is invalid")
	ErrNotFound         = errors.New("identity not found")
)

func (i Identity) Validate() error {
	if strings.TrimSpace(i.FullName) == "" {
		return ErrFullNameRequired
	}
	if i.Email != "" && !strings.Contains(i.Email, "@") {
		return ErrEmailInvalid
	}
	return nil
}
