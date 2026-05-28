package domain

import (
	"time"

	"github.com/google/uuid"
)

type UserStatus string

const (
	UserStatusActive    UserStatus = "active"
	UserStatusSuspended UserStatus = "suspended"
	UserStatusDeleted   UserStatus = "deleted"
)

// Role constants for the 3-tier permission model.
const (
	RoleSuperAdmin = "super_admin"
	RoleAdmin      = "admin"
	RoleUser       = "user"
)

type User struct {
	ID                      uuid.UUID  `json:"id"`
	UID                     int64      `json:"uid"`
	Email                   string     `json:"email"`
	EmailVerified           bool       `json:"email_verified"`
	PasswordHash            string     `json:"-"`
	DisplayName             string     `json:"display_name"`
	Alias                   *string    `json:"alias,omitempty"`
	AvatarURL               string     `json:"avatar_url"`
	SecurityLevel           int        `json:"security_level"`
	Role                    string     `json:"role"`
	Status                  UserStatus `json:"status"`
	RiskReportEmailEnabled  bool       `json:"risk_report_email_enabled"`
	LastLoginAt             *time.Time `json:"last_login_at,omitempty"`
	CreatedAt               time.Time  `json:"created_at"`
	UpdatedAt               time.Time  `json:"updated_at"`
}

// IsSuperAdmin returns true if the user has the super_admin role.
func (u *User) IsSuperAdmin() bool {
	return u.Role == RoleSuperAdmin
}

// IsAdmin returns true if the user has admin or super_admin role.
func (u *User) IsAdmin() bool {
	return u.Role == RoleAdmin || u.Role == RoleSuperAdmin
}

// IsUser returns true if the user has the user role.
func (u *User) IsUser() bool {
	return u.Role == RoleUser
}

type SocialBindingStatus string

const (
	SocialBindingStatusActive          SocialBindingStatus = "active"
	SocialBindingStatusUserUnbound     SocialBindingStatus = "user_unbound"
	SocialBindingStatusProviderRevoked SocialBindingStatus = "provider_revoked"
	SocialBindingStatusTokenExpired    SocialBindingStatus = "token_expired"
	SocialBindingStatusDisabled        SocialBindingStatus = "disabled"
)

type SocialAuthStatus string

const (
	SocialAuthStatusActive      SocialAuthStatus = "active"
	SocialAuthStatusRevoked     SocialAuthStatus = "revoked"
	SocialAuthStatusExpired     SocialAuthStatus = "expired"
	SocialAuthStatusUnknown     SocialAuthStatus = "unknown"
	SocialAuthStatusUnsupported SocialAuthStatus = "unsupported"
)

type SocialBinding struct {
	ID              uuid.UUID           `json:"id"`
	UserID          uuid.UUID           `json:"user_id"`
	Provider        string              `json:"provider"`
	ProviderUID     string              `json:"provider_uid"`
	ProviderEmail   *string             `json:"provider_email,omitempty"`
	ProviderName    *string             `json:"provider_name,omitempty"`
	ProviderAvatar  *string             `json:"provider_avatar,omitempty"`
	Status          SocialBindingStatus `json:"status"`
	AccessToken     *string             `json:"-"`
	RefreshToken    *string             `json:"-"`
	TokenExpiry     *time.Time          `json:"-"`
	TokenType       *string             `json:"-"`
	TokenScopes     []string            `json:"-"`
	RawProfile      map[string]any      `json:"-"`
	BoundAt         time.Time           `json:"bound_at"`
	VerifiedAt      *time.Time          `json:"verified_at,omitempty"`
	UnboundAt       *time.Time          `json:"unbound_at,omitempty"`
	UnbindReason    *string             `json:"unbind_reason,omitempty"`
	LastAuthCheckAt *time.Time          `json:"last_auth_check_at,omitempty"`
	LastAuthStatus  SocialAuthStatus    `json:"last_auth_status"`
	LastAuthError   *string             `json:"last_auth_error,omitempty"`
	CreatedAt       time.Time           `json:"created_at"`
	UpdatedAt       time.Time           `json:"updated_at"`
}
