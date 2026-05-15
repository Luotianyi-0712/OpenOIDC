package domain

import (
	"time"

	"github.com/google/uuid"
)

type OIDCClient struct {
	ID                      uuid.UUID  `json:"id"`
	ClientID                string     `json:"client_id"`
	ClientSecretHash        string     `json:"-"`
	ClientSecretPlain       string     `json:"client_secret,omitempty"`
	ClientName              string     `json:"client_name"`
	Description             string     `json:"description"`
	LogoURL                 string     `json:"logo_url"`
	OwnerUserID             *uuid.UUID `json:"owner_user_id,omitempty"`
	RedirectURIs            []string   `json:"redirect_uris"`
	GrantTypes              []string   `json:"grant_types"`
	ResponseTypes           []string   `json:"response_types"`
	Scopes                  []string   `json:"scopes"`
	TokenEndpointAuthMethod string     `json:"token_endpoint_auth_method"`
	MinSecurityLevel        int        `json:"min_security_level"`
	RequireEmailVerified    bool       `json:"require_email_verified"`
	ProtocolType            string     `json:"protocol_type"`
	IsActive                bool       `json:"is_active"`
	IsConfidential          bool       `json:"is_confidential"`
	CreatedAt               time.Time  `json:"created_at"`
	UpdatedAt               time.Time  `json:"updated_at"`
}

type AccessRuleType string

const (
	AccessRuleEmailDomainAllow AccessRuleType = "email_domain_allow"
	AccessRuleEmailAllow       AccessRuleType = "email_allow"
	AccessRuleEmailDeny        AccessRuleType = "email_deny"
	AccessRuleIPAllow          AccessRuleType = "ip_allow"
	AccessRuleIPDeny           AccessRuleType = "ip_deny"
)

type ClientAccessRule struct {
	ID        uuid.UUID `json:"id"`
	ClientID  uuid.UUID `json:"client_id"`
	RuleType  string    `json:"rule_type"`
	Value     string    `json:"value"`
	CreatedAt time.Time `json:"created_at"`
}
