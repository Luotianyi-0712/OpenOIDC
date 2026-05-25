package service

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/anthropic/oidc-platform/internal/domain"
	"github.com/anthropic/oidc-platform/internal/port"
)

type ClientService struct {
	clientRepo     port.ClientRepository
	accessRuleRepo port.ClientAccessRuleRepository
	auditRepo      port.AuditRepository
}

func NewClientService(
	clientRepo port.ClientRepository,
	accessRuleRepo port.ClientAccessRuleRepository,
	auditRepo port.AuditRepository,
) *ClientService {
	return &ClientService{
		clientRepo:     clientRepo,
		accessRuleRepo: accessRuleRepo,
		auditRepo:      auditRepo,
	}
}

type CreateClientInput struct {
	ClientName           string
	Description          string
	LogoURL              string
	HomepageURL          string
	OwnerUserID          *uuid.UUID
	RedirectURIs         []string
	GrantTypes           []string
	ResponseTypes        []string
	Scopes               []string
	MinSecurityLevel     int
	RequireEmailVerified *bool
	ProtocolType         string
	IsConfidential       bool
}

func (s *ClientService) CreateClient(ctx context.Context, input CreateClientInput) (*domain.OIDCClient, string, error) {
	if input.ClientName == "" {
		return nil, "", fmt.Errorf("%w: client_name required", ErrInvalidInput)
	}
	if input.ProtocolType == "" {
		input.ProtocolType = "oidc"
	}
	if len(input.GrantTypes) == 0 {
		input.GrantTypes = []string{"authorization_code", "refresh_token"}
	}
	if len(input.ResponseTypes) == 0 {
		input.ResponseTypes = []string{"code"}
	}
	if len(input.Scopes) == 0 {
		input.Scopes = []string{"openid", "profile", "email"}
	}
	if err := validateClientConfig(input.RedirectURIs, input.GrantTypes, input.ResponseTypes, input.Scopes, input.HomepageURL, input.MinSecurityLevel, input.IsConfidential); err != nil {
		return nil, "", err
	}

	clientID, err := generateClientID()
	if err != nil {
		return nil, "", err
	}
	plainSecret, err := generateClientSecret()
	if err != nil {
		return nil, "", err
	}
	secretHash, err := hashPassword(plainSecret)
	if err != nil {
		return nil, "", fmt.Errorf("hash secret: %w", err)
	}

	tokenAuth := "client_secret_basic"
	if !input.IsConfidential {
		tokenAuth = "none"
	}

	requireEmailVerified := true
	if input.RequireEmailVerified != nil {
		requireEmailVerified = *input.RequireEmailVerified
	}

	now := time.Now().UTC()
	client := &domain.OIDCClient{
		ID:                      uuid.New(),
		ClientID:                clientID,
		ClientSecretHash:        secretHash,
		ClientSecretPlain:       plainSecret,
		ClientName:              input.ClientName,
		Description:             input.Description,
		LogoURL:                 input.LogoURL,
		HomepageURL:             strings.TrimSpace(input.HomepageURL),
		OwnerUserID:             input.OwnerUserID,
		RedirectURIs:            input.RedirectURIs,
		GrantTypes:              input.GrantTypes,
		ResponseTypes:           input.ResponseTypes,
		Scopes:                  input.Scopes,
		TokenEndpointAuthMethod: tokenAuth,
		MinSecurityLevel:        input.MinSecurityLevel,
		RequireEmailVerified:    requireEmailVerified,
		ProtocolType:            input.ProtocolType,
		IsActive:                true,
		IsConfidential:          input.IsConfidential,
		CreatedAt:               now,
		UpdatedAt:               now,
	}
	if err := s.clientRepo.Create(ctx, client); err != nil {
		return nil, "", fmt.Errorf("create client: %w", err)
	}

	rt := "oidc_client"
	rid := client.ID.String()
	_ = s.auditRepo.CreateLog(ctx, &domain.AuditLog{
		ID:           uuid.New(),
		UserID:       input.OwnerUserID,
		Action:       "client.created",
		ResourceType: &rt,
		ResourceID:   &rid,
		Details:      map[string]any{"client_id": clientID, "name": input.ClientName},
		CreatedAt:    now,
	})

	return client, plainSecret, nil
}

func (s *ClientService) GetClient(ctx context.Context, id uuid.UUID) (*domain.OIDCClient, error) {
	c, err := s.clientRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, port.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return c, nil
}

func (s *ClientService) GetClientByClientID(ctx context.Context, clientID string) (*domain.OIDCClient, error) {
	c, err := s.clientRepo.GetByClientID(ctx, clientID)
	if err != nil {
		if errors.Is(err, port.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return c, nil
}

func (s *ClientService) ListClients(ctx context.Context, offset, limit int) ([]*domain.OIDCClient, int64, error) {
	if limit <= 0 {
		limit = 20
	}
	return s.clientRepo.List(ctx, offset, limit)
}

func (s *ClientService) ListClientsByOwner(ctx context.Context, ownerID uuid.UUID, offset, limit int) ([]*domain.OIDCClient, int64, error) {
	if limit <= 0 {
		limit = 20
	}
	return s.clientRepo.ListByOwner(ctx, ownerID, offset, limit)
}

func (s *ClientService) UpdateClient(ctx context.Context, c *domain.OIDCClient) error {
	if c.ClientName == "" {
		return fmt.Errorf("%w: client_name required", ErrInvalidInput)
	}
	if err := validateClientConfig(c.RedirectURIs, c.GrantTypes, c.ResponseTypes, c.Scopes, c.HomepageURL, c.MinSecurityLevel, c.IsConfidential); err != nil {
		return err
	}
	c.UpdatedAt = time.Now().UTC()
	if err := s.clientRepo.Update(ctx, c); err != nil {
		return fmt.Errorf("update client: %w", err)
	}
	rt := "oidc_client"
	rid := c.ID.String()
	_ = s.auditRepo.CreateLog(ctx, &domain.AuditLog{
		ID:           uuid.New(),
		UserID:       c.OwnerUserID,
		Action:       "client.updated",
		ResourceType: &rt,
		ResourceID:   &rid,
		CreatedAt:    time.Now().UTC(),
	})
	return nil
}

func (s *ClientService) DeleteClient(ctx context.Context, id uuid.UUID) error {
	if err := s.clientRepo.Delete(ctx, id); err != nil {
		return fmt.Errorf("delete client: %w", err)
	}
	rt := "oidc_client"
	rid := id.String()
	_ = s.auditRepo.CreateLog(ctx, &domain.AuditLog{
		ID:           uuid.New(),
		Action:       "client.deleted",
		ResourceType: &rt,
		ResourceID:   &rid,
		CreatedAt:    time.Now().UTC(),
	})
	return nil
}

func (s *ClientService) RotateSecret(ctx context.Context, id uuid.UUID) (string, error) {
	plain, err := generateClientSecret()
	if err != nil {
		return "", err
	}
	hash, err := hashPassword(plain)
	if err != nil {
		return "", fmt.Errorf("hash secret: %w", err)
	}
	if err := s.clientRepo.UpdateSecret(ctx, id, hash, plain); err != nil {
		return "", fmt.Errorf("update secret: %w", err)
	}
	rt := "oidc_client"
	rid := id.String()
	_ = s.auditRepo.CreateLog(ctx, &domain.AuditLog{
		ID:           uuid.New(),
		Action:       "client.secret_rotated",
		ResourceType: &rt,
		ResourceID:   &rid,
		CreatedAt:    time.Now().UTC(),
	})
	return plain, nil
}

func (s *ClientService) VerifySecret(ctx context.Context, clientID, secret string) (*domain.OIDCClient, error) {
	c, err := s.clientRepo.GetByClientID(ctx, clientID)
	if err != nil {
		if errors.Is(err, port.ErrNotFound) {
			return nil, ErrInvalidCredentials
		}
		return nil, err
	}
	ok, err := verifyPassword(c.ClientSecretHash, secret)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrInvalidCredentials
	}
	return c, nil
}

func (s *ClientService) AddAccessRule(ctx context.Context, clientID uuid.UUID, ruleType, value string) (*domain.ClientAccessRule, error) {
	if !isValidAccessRuleType(ruleType) {
		return nil, fmt.Errorf("%w: invalid rule_type", ErrInvalidInput)
	}
	if value == "" {
		return nil, fmt.Errorf("%w: value required", ErrInvalidInput)
	}
	rule := &domain.ClientAccessRule{
		ID:        uuid.New(),
		ClientID:  clientID,
		RuleType:  ruleType,
		Value:     value,
		CreatedAt: time.Now().UTC(),
	}
	if err := s.accessRuleRepo.Create(ctx, rule); err != nil {
		return nil, fmt.Errorf("create rule: %w", err)
	}
	return rule, nil
}

func (s *ClientService) ListAccessRules(ctx context.Context, clientID uuid.UUID) ([]*domain.ClientAccessRule, error) {
	return s.accessRuleRepo.ListByClient(ctx, clientID)
}

func (s *ClientService) RemoveAccessRule(ctx context.Context, ruleID uuid.UUID) error {
	return s.accessRuleRepo.Delete(ctx, ruleID)
}

func validateClientConfig(redirectURIs, grantTypes, responseTypes, scopes []string, homepageURL string, minSecurityLevel int, isConfidential bool) error {
	if err := validateRedirectURIs(redirectURIs); err != nil {
		return err
	}
	if err := validateGrantTypes(grantTypes); err != nil {
		return err
	}
	if err := validateResponseTypes(responseTypes); err != nil {
		return err
	}
	if err := validateScopes(scopes); err != nil {
		return err
	}
	if err := validateHomepageURL(homepageURL); err != nil {
		return err
	}
	if minSecurityLevel < 0 {
		return fmt.Errorf("%w: min_security_level must be non-negative", ErrInvalidInput)
	}
	if !isConfidential && containsString(grantTypes, "client_credentials") {
		return fmt.Errorf("%w: public clients cannot use client_credentials", ErrInvalidInput)
	}
	return nil
}

func validateRedirectURIs(redirectURIs []string) error {
	if len(redirectURIs) == 0 {
		return fmt.Errorf("%w: at least one redirect_uri required", ErrInvalidInput)
	}
	seen := make(map[string]struct{}, len(redirectURIs))
	for _, raw := range redirectURIs {
		value := strings.TrimSpace(raw)
		if value == "" {
			return fmt.Errorf("%w: redirect_uri cannot be empty", ErrInvalidInput)
		}
		if _, exists := seen[value]; exists {
			return fmt.Errorf("%w: duplicate redirect_uri", ErrInvalidInput)
		}
		seen[value] = struct{}{}

		u, err := url.Parse(value)
		if err != nil || u.Scheme == "" || u.Host == "" || u.Fragment != "" {
			return fmt.Errorf("%w: invalid redirect_uri", ErrInvalidInput)
		}
		switch u.Scheme {
		case "http":
			if u.Hostname() != "localhost" && u.Hostname() != "127.0.0.1" && u.Hostname() != "::1" {
				return fmt.Errorf("%w: http redirect_uri only allowed for localhost", ErrInvalidInput)
			}
		case "https":
		default:
			return fmt.Errorf("%w: redirect_uri must use https", ErrInvalidInput)
		}
	}
	return nil
}

func validateHomepageURL(raw string) error {
	value := strings.TrimSpace(raw)
	if value == "" {
		return nil
	}
	u, err := url.Parse(value)
	if err != nil || u.Scheme == "" || u.Host == "" || u.Fragment != "" {
		return fmt.Errorf("%w: invalid homepage_url", ErrInvalidInput)
	}
	switch u.Scheme {
	case "http", "https":
		return nil
	default:
		return fmt.Errorf("%w: homepage_url must use http or https", ErrInvalidInput)
	}
}

func validateGrantTypes(grantTypes []string) error {
	if len(grantTypes) == 0 {
		return fmt.Errorf("%w: at least one grant_type required", ErrInvalidInput)
	}
	seen := make(map[string]struct{}, len(grantTypes))
	for _, grantType := range grantTypes {
		grantType = strings.TrimSpace(grantType)
		if _, exists := seen[grantType]; exists {
			return fmt.Errorf("%w: duplicate grant_type", ErrInvalidInput)
		}
		seen[grantType] = struct{}{}
		switch grantType {
		case "authorization_code", "refresh_token", "client_credentials":
		default:
			return fmt.Errorf("%w: unsupported grant_type", ErrInvalidInput)
		}
	}
	return nil
}

func validateResponseTypes(responseTypes []string) error {
	if len(responseTypes) == 0 {
		return fmt.Errorf("%w: at least one response_type required", ErrInvalidInput)
	}
	for _, responseType := range responseTypes {
		if strings.TrimSpace(responseType) != "code" {
			return fmt.Errorf("%w: unsupported response_type", ErrInvalidInput)
		}
	}
	return nil
}

func validateScopes(scopes []string) error {
	if len(scopes) == 0 {
		return fmt.Errorf("%w: at least one scope required", ErrInvalidInput)
	}
	seen := make(map[string]struct{}, len(scopes))
	for _, scope := range scopes {
		scope = strings.TrimSpace(scope)
		if _, exists := seen[scope]; exists {
			return fmt.Errorf("%w: duplicate scope", ErrInvalidInput)
		}
		seen[scope] = struct{}{}
		switch scope {
		case "openid", "profile", "email", "security_level", "offline_access":
		default:
			return fmt.Errorf("%w: unsupported scope", ErrInvalidInput)
		}
	}
	return nil
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if strings.TrimSpace(value) == target {
			return true
		}
	}
	return false
}

func isValidAccessRuleType(t string) bool {
	switch domain.AccessRuleType(t) {
	case domain.AccessRuleEmailDomainAllow,
		domain.AccessRuleEmailAllow,
		domain.AccessRuleEmailDeny,
		domain.AccessRuleIPAllow,
		domain.AccessRuleIPDeny:
		return true
	}
	return false
}
