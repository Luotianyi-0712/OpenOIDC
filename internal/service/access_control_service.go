package service

import (
	"context"
	"fmt"
	"net"
	"regexp"
	"strings"

	"github.com/anthropic/oidc-platform/internal/domain"
	"github.com/anthropic/oidc-platform/internal/port"
)

type AccessControlService struct {
	accessRuleRepo port.ClientAccessRuleRepository
	aliasRepo      port.AliasRestrictionRepository
}

func NewAccessControlService(
	accessRuleRepo port.ClientAccessRuleRepository,
	aliasRepo port.AliasRestrictionRepository,
) *AccessControlService {
	return &AccessControlService{
		accessRuleRepo: accessRuleRepo,
		aliasRepo:      aliasRepo,
	}
}

func (s *AccessControlService) CheckAccess(ctx context.Context, client *domain.OIDCClient, user *domain.User, ip string) (bool, string) {
	if !client.IsActive {
		return false, "client_inactive"
	}
	if user.Status != domain.UserStatusActive {
		return false, "user_inactive"
	}
	if user.SecurityLevel < client.MinSecurityLevel {
		return false, "security_level_insufficient"
	}
	if client.RequireEmailVerified && !user.EmailVerified {
		return false, "email_not_verified"
	}

	rules, err := s.accessRuleRepo.ListByClient(ctx, client.ID)
	if err != nil {
		return false, "rule_lookup_failed"
	}

	var emailDomainAllow, emailAllow, emailDeny, ipAllow, ipDeny []string
	for _, r := range rules {
		switch domain.AccessRuleType(r.RuleType) {
		case domain.AccessRuleEmailDomainAllow:
			emailDomainAllow = append(emailDomainAllow, r.Value)
		case domain.AccessRuleEmailAllow:
			emailAllow = append(emailAllow, r.Value)
		case domain.AccessRuleEmailDeny:
			emailDeny = append(emailDeny, r.Value)
		case domain.AccessRuleIPAllow:
			ipAllow = append(ipAllow, r.Value)
		case domain.AccessRuleIPDeny:
			ipDeny = append(ipDeny, r.Value)
		}
	}

	email := strings.ToLower(user.Email)
	for _, e := range emailDeny {
		if strings.EqualFold(e, email) {
			return false, "email_denied"
		}
	}
	if len(emailDomainAllow) > 0 || len(emailAllow) > 0 {
		matched := false
		for _, e := range emailAllow {
			if strings.EqualFold(e, email) {
				matched = true
				break
			}
		}
		if !matched {
			parts := strings.SplitN(email, "@", 2)
			if len(parts) == 2 {
				for _, d := range emailDomainAllow {
					if strings.EqualFold(strings.TrimPrefix(d, "@"), parts[1]) {
						matched = true
						break
					}
				}
			}
		}
		if !matched {
			return false, "email_not_in_allowlist"
		}
	}

	if ip != "" {
		for _, v := range ipDeny {
			if matchIP(v, ip) {
				return false, "ip_denied"
			}
		}
		if len(ipAllow) > 0 {
			matched := false
			for _, v := range ipAllow {
				if matchIP(v, ip) {
					matched = true
					break
				}
			}
			if !matched {
				return false, "ip_not_in_allowlist"
			}
		}
	}

	return true, ""
}

func matchIP(rule, ip string) bool {
	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return false
	}
	if strings.Contains(rule, "/") {
		_, ipNet, err := net.ParseCIDR(rule)
		if err != nil {
			return false
		}
		return ipNet.Contains(parsedIP)
	}
	ruleIP := net.ParseIP(rule)
	if ruleIP == nil {
		return false
	}
	return ruleIP.Equal(parsedIP)
}

func (s *AccessControlService) ValidateAlias(ctx context.Context, alias string) error {
	alias = strings.TrimSpace(alias)
	if alias == "" {
		return fmt.Errorf("%w: alias empty", ErrInvalidAlias)
	}
	restrictions, err := s.aliasRepo.List(ctx)
	if err != nil {
		return fmt.Errorf("list restrictions: %w", err)
	}
	lower := strings.ToLower(alias)
	for _, r := range restrictions {
		switch r.RestrictionType {
		case "reserved", "blocked":
			if strings.EqualFold(r.Pattern, alias) {
				return fmt.Errorf("%w: %s", ErrInvalidAlias, r.Reason)
			}
		case "regex_blocked":
			re, err := regexp.Compile(r.Pattern)
			if err != nil {
				continue
			}
			if re.MatchString(lower) {
				return fmt.Errorf("%w: %s", ErrInvalidAlias, r.Reason)
			}
		}
	}
	return nil
}
