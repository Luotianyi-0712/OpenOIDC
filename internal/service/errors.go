package service

import "errors"

var (
	ErrNotFound                  = errors.New("not found")
	ErrAlreadyExists             = errors.New("already exists")
	ErrInvalidCredentials        = errors.New("invalid credentials")
	ErrAccountSuspended          = errors.New("account suspended")
	ErrAccountDeleted            = errors.New("account deleted")
	ErrAccountLockedOut          = errors.New("account temporarily locked due to too many failed login attempts")
	ErrEmailNotVerified          = errors.New("email not verified")
	ErrProviderDisabled          = errors.New("provider disabled")
	ErrAlreadyBound              = errors.New("already bound")
	ErrBindingNotFound           = errors.New("binding not found")
	ErrInvalidAlias              = errors.New("invalid alias")
	ErrSecurityLevelInsufficient = errors.New("security level insufficient")
	ErrAccessDenied              = errors.New("access denied")
	ErrInvalidToken              = errors.New("invalid token")
	ErrInvalidEmail              = errors.New("invalid email")
	ErrPasswordTooWeak           = errors.New("password too weak")
	ErrSessionExpired            = errors.New("session expired")
	ErrSessionNotFound           = errors.New("session not found")
	ErrPermissionDenied          = errors.New("permission denied")
	ErrInvalidInput              = errors.New("invalid input")
	ErrRegistrationDisabled      = errors.New("registration disabled")
)
