package service

import (
	"context"
	"strconv"

	"github.com/anthropic/oidc-platform/internal/config"
	"github.com/anthropic/oidc-platform/internal/port"
)

type PasswordPolicy struct {
	MinLength     int  `json:"min_length"`
	RequireUpper  bool `json:"require_upper"`
	RequireLower  bool `json:"require_lower"`
	RequireDigit  bool `json:"require_digit"`
	RequireSymbol bool `json:"require_symbol"`
}

func ResolvePasswordPolicy(ctx context.Context, settingsRepo port.SettingsRepository, security config.SecurityConfig) PasswordPolicy {
	policy := PasswordPolicy{
		MinLength:     security.PasswordMinLength,
		RequireUpper:  security.PasswordRequireUpper,
		RequireLower:  security.PasswordRequireLower,
		RequireDigit:  security.PasswordRequireDigit,
		RequireSymbol: security.PasswordRequireSymbol,
	}
	if policy.MinLength <= 0 {
		policy.MinLength = 8
	}
	if settingsRepo == nil {
		return policy
	}
	if s, err := settingsRepo.Get(ctx, "password_min_length"); err == nil && s != nil && s.Value != "" {
		if v, e := strconv.Atoi(s.Value); e == nil && v > 0 {
			policy.MinLength = v
		}
	}
	if s, err := settingsRepo.Get(ctx, "password_require_upper"); err == nil && s != nil && s.Value != "" {
		policy.RequireUpper = s.Value == "true"
	}
	if s, err := settingsRepo.Get(ctx, "password_require_lower"); err == nil && s != nil && s.Value != "" {
		policy.RequireLower = s.Value == "true"
	}
	if s, err := settingsRepo.Get(ctx, "password_require_digit"); err == nil && s != nil && s.Value != "" {
		policy.RequireDigit = s.Value == "true"
	}
	if s, err := settingsRepo.Get(ctx, "password_require_symbol"); err == nil && s != nil && s.Value != "" {
		policy.RequireSymbol = s.Value == "true"
	}
	return policy
}

func ValidatePasswordByPolicy(password string, policy PasswordPolicy) error {
	if len(password) < policy.MinLength {
		return ErrPasswordTooWeak
	}
	if policy.RequireUpper && !containsAny(password, "ABCDEFGHIJKLMNOPQRSTUVWXYZ") {
		return ErrPasswordTooWeak
	}
	if policy.RequireLower && !containsAny(password, "abcdefghijklmnopqrstuvwxyz") {
		return ErrPasswordTooWeak
	}
	if policy.RequireDigit && !containsAny(password, "0123456789") {
		return ErrPasswordTooWeak
	}
	if policy.RequireSymbol && !containsAny(password, "!@#$%^&*()-_=+[]{};:,.<>/?\\|`~'\"") {
		return ErrPasswordTooWeak
	}
	return nil
}
