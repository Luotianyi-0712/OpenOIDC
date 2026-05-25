package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/google/uuid"

	"github.com/anthropic/oidc-platform/internal/config"
	"github.com/anthropic/oidc-platform/internal/domain"
	"github.com/anthropic/oidc-platform/internal/port"
)

const (
	passkeyChallengePrefix = "passkey:challenge:"
	passkeyChallengeExpiry = 60 * time.Second
)

type PasskeyService struct {
	wa           *webauthn.WebAuthn
	passkeyRepo  port.PasskeyRepository
	userRepo     port.UserRepository
	sessionRepo  port.SessionRepository
	cache        port.Cache
	settingsRepo port.SettingsRepository
	cfg          *config.Config
}

func NewPasskeyService(
	passkeyRepo port.PasskeyRepository,
	userRepo port.UserRepository,
	sessionRepo port.SessionRepository,
	cache port.Cache,
	settingsRepo port.SettingsRepository,
	cfg *config.Config,
) (*PasskeyService, error) {
	rpID := cfg.WebAuthn.RPID
	rpOrigin := cfg.WebAuthn.RPOrigin
	rpDisplayName := cfg.WebAuthn.RPDisplayName

	if rpID == "" {
		if cfg.Server.BaseURL != "" {
			u, err := url.Parse(cfg.Server.BaseURL)
			if err == nil {
				rpID = u.Hostname()
			}
		}
		if rpID == "" {
			rpID = "localhost"
		}
	}
	if rpOrigin == "" {
		rpOrigin = cfg.Server.BaseURL
		if rpOrigin == "" {
			rpOrigin = fmt.Sprintf("http://localhost:%d", cfg.Server.Port)
		}
	}
	if rpDisplayName == "" {
		rpDisplayName = "OIDC Platform"
	}

	wa, err := webauthn.New(&webauthn.Config{
		RPID:                  rpID,
		RPDisplayName:         rpDisplayName,
		RPOrigins:             []string{rpOrigin},
		AttestationPreference: protocol.PreferNoAttestation,
		AuthenticatorSelection: protocol.AuthenticatorSelection{
			ResidentKey:      protocol.ResidentKeyRequirementPreferred,
			UserVerification: protocol.VerificationPreferred,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("init webauthn: %w", err)
	}

	return &PasskeyService{
		wa:           wa,
		passkeyRepo:  passkeyRepo,
		userRepo:     userRepo,
		sessionRepo:  sessionRepo,
		cache:        cache,
		settingsRepo: settingsRepo,
		cfg:          cfg,
	}, nil
}

// BeginRegistration starts the passkey registration ceremony for an authenticated user.
func (s *PasskeyService) BeginRegistration(ctx context.Context, userID uuid.UUID) (*protocol.CredentialCreation, string, error) {
	if !s.isSettingEnabled(ctx, "passkey_enabled") {
		return nil, "", ErrPasskeyDisabled
	}

	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, "", fmt.Errorf("get user: %w", err)
	}

	creds, err := s.passkeyRepo.ListByUser(ctx, userID)
	if err != nil {
		return nil, "", fmt.Errorf("list credentials: %w", err)
	}

	waUser := &domain.WebAuthnUser{User: user, Credentials: creds}

	excludeList := make([]protocol.CredentialDescriptor, 0, len(creds))
	for _, c := range creds {
		excludeList = append(excludeList, protocol.CredentialDescriptor{
			Type:         protocol.PublicKeyCredentialType,
			CredentialID: c.CredentialID,
		})
	}

	creation, sessionData, err := s.wa.BeginRegistration(waUser,
		webauthn.WithExclusions(excludeList),
	)
	if err != nil {
		return nil, "", fmt.Errorf("begin registration: %w", err)
	}

	sessionID := uuid.New().String()
	sessionBytes, _ := json.Marshal(sessionData)
	if err := s.cache.Set(ctx, passkeyChallengePrefix+sessionID, sessionBytes, passkeyChallengeExpiry); err != nil {
		return nil, "", fmt.Errorf("store session: %w", err)
	}

	return creation, sessionID, nil
}

// FinishRegistration completes the passkey registration ceremony.
func (s *PasskeyService) FinishRegistration(ctx context.Context, userID uuid.UUID, sessionID string, r *http.Request) (*domain.PasskeyCredential, error) {
	if !s.isSettingEnabled(ctx, "passkey_enabled") {
		return nil, ErrPasskeyDisabled
	}

	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}

	creds, err := s.passkeyRepo.ListByUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list credentials: %w", err)
	}

	waUser := &domain.WebAuthnUser{User: user, Credentials: creds}

	sessionBytes, err := s.cache.Get(ctx, passkeyChallengePrefix+sessionID)
	if err != nil || len(sessionBytes) == 0 {
		return nil, ErrInvalidToken
	}
	_ = s.cache.Delete(ctx, passkeyChallengePrefix+sessionID)

	var sessionData webauthn.SessionData
	if err := json.Unmarshal(sessionBytes, &sessionData); err != nil {
		return nil, ErrInvalidToken
	}

	credential, err := s.wa.FinishRegistration(waUser, sessionData, r)
	if err != nil {
		return nil, fmt.Errorf("finish registration: %w", err)
	}

	transport := make([]string, 0, len(credential.Transport))
	for _, t := range credential.Transport {
		transport = append(transport, string(t))
	}

	pc := &domain.PasskeyCredential{
		ID:              uuid.New(),
		UserID:          userID,
		CredentialID:    credential.ID,
		PublicKey:       credential.PublicKey,
		AttestationType: credential.AttestationType,
		Transport:       transport,
		SignCount:       credential.Authenticator.SignCount,
		AAGUID:          credential.Authenticator.AAGUID,
		CreatedAt:       time.Now().UTC(),
	}

	if err := s.passkeyRepo.Create(ctx, pc); err != nil {
		return nil, fmt.Errorf("save credential: %w", err)
	}

	return pc, nil
}

// BeginLogin starts the passkey login ceremony (discoverable credentials).
func (s *PasskeyService) BeginLogin(ctx context.Context) (*protocol.CredentialAssertion, string, error) {
	if !s.isSettingEnabled(ctx, "passkey_enabled") {
		return nil, "", ErrPasskeyDisabled
	}

	assertion, sessionData, err := s.wa.BeginDiscoverableLogin()
	if err != nil {
		return nil, "", fmt.Errorf("begin login: %w", err)
	}

	sessionID := uuid.New().String()
	sessionBytes, _ := json.Marshal(sessionData)
	if err := s.cache.Set(ctx, passkeyChallengePrefix+sessionID, sessionBytes, passkeyChallengeExpiry); err != nil {
		return nil, "", fmt.Errorf("store session: %w", err)
	}

	return assertion, sessionID, nil
}

// FinishLogin completes the passkey login ceremony and returns a session.
func (s *PasskeyService) FinishLogin(ctx context.Context, sessionID, ip, userAgent string, r *http.Request) (*domain.UserSession, error) {
	if !s.isSettingEnabled(ctx, "passkey_enabled") {
		return nil, ErrPasskeyDisabled
	}

	sessionBytes, err := s.cache.Get(ctx, passkeyChallengePrefix+sessionID)
	if err != nil || len(sessionBytes) == 0 {
		return nil, ErrInvalidToken
	}
	_ = s.cache.Delete(ctx, passkeyChallengePrefix+sessionID)

	var sessionData webauthn.SessionData
	if err := json.Unmarshal(sessionBytes, &sessionData); err != nil {
		return nil, ErrInvalidToken
	}

	// Discoverable login handler: resolve user from credential owner.
	handler := func(rawID, userHandle []byte) (webauthn.User, error) {
		uid, err := uuid.FromBytes(userHandle)
		if err != nil {
			return nil, fmt.Errorf("parse user handle: %w", err)
		}
		user, err := s.userRepo.GetByID(ctx, uid)
		if err != nil {
			return nil, fmt.Errorf("get user: %w", err)
		}
		creds, err := s.passkeyRepo.ListByUser(ctx, uid)
		if err != nil {
			return nil, fmt.Errorf("list credentials: %w", err)
		}
		return &domain.WebAuthnUser{User: user, Credentials: creds}, nil
	}

	credential, err := s.wa.FinishDiscoverableLogin(handler, sessionData, r)
	if err != nil {
		return nil, fmt.Errorf("finish login: %w", err)
	}

	// Find the credential in DB to get user_id and update sign count.
	dbCred, err := s.passkeyRepo.GetByCredentialID(ctx, credential.ID)
	if err != nil {
		return nil, fmt.Errorf("lookup credential: %w", err)
	}

	_ = s.passkeyRepo.UpdateSignCount(ctx, dbCred.ID, credential.Authenticator.SignCount)
	_ = s.passkeyRepo.UpdateLastUsed(ctx, dbCred.ID)

	// Check user status.
	user, err := s.userRepo.GetByID(ctx, dbCred.UserID)
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}
	switch user.Status {
	case domain.UserStatusSuspended:
		return nil, ErrAccountSuspended
	case domain.UserStatusDeleted:
		return nil, ErrAccountDeleted
	}

	// Create session.
	session, err := s.createSession(ctx, user.ID, ip, userAgent)
	if err != nil {
		return nil, err
	}

	_ = s.userRepo.UpdateLastLogin(ctx, user.ID)

	return session, nil
}

func (s *PasskeyService) createSession(ctx context.Context, userID uuid.UUID, ip, userAgent string) (*domain.UserSession, error) {
	token, err := generateSessionToken()
	if err != nil {
		return nil, fmt.Errorf("generate token: %w", err)
	}
	sess := &domain.UserSession{
		ID:           uuid.New(),
		UserID:       userID,
		SessionToken: token,
		ExpiresAt:    time.Now().Add(s.cfg.Session.TTL),
		CreatedAt:    time.Now().UTC(),
	}
	if ip != "" {
		sess.IPAddress = &ip
	}
	if userAgent != "" {
		sess.UserAgent = &userAgent
	}
	if err := s.sessionRepo.Create(ctx, sess); err != nil {
		return nil, fmt.Errorf("create session: %w", err)
	}
	return sess, nil
}

// ListCredentials returns all passkey credentials for a user.
func (s *PasskeyService) ListCredentials(ctx context.Context, userID uuid.UUID) ([]*domain.PasskeyCredential, error) {
	return s.passkeyRepo.ListByUser(ctx, userID)
}

// DeleteCredential removes a passkey credential.
func (s *PasskeyService) DeleteCredential(ctx context.Context, userID, credID uuid.UUID) error {
	creds, err := s.passkeyRepo.ListByUser(ctx, userID)
	if err != nil {
		return err
	}
	found := false
	for _, c := range creds {
		if c.ID == credID {
			found = true
			break
		}
	}
	if !found {
		return ErrNotFound
	}
	return s.passkeyRepo.Delete(ctx, credID)
}

// RenameCredential renames a passkey credential.
func (s *PasskeyService) RenameCredential(ctx context.Context, userID, credID uuid.UUID, name string) error {
	creds, err := s.passkeyRepo.ListByUser(ctx, userID)
	if err != nil {
		return err
	}
	found := false
	for _, c := range creds {
		if c.ID == credID {
			found = true
			break
		}
	}
	if !found {
		return ErrNotFound
	}
	return s.passkeyRepo.Rename(ctx, credID, name)
}

// HasPasskeys returns true if the user has at least one registered passkey.
func (s *PasskeyService) HasPasskeys(ctx context.Context, userID uuid.UUID) (bool, error) {
	creds, err := s.passkeyRepo.ListByUser(ctx, userID)
	if err != nil {
		return false, err
	}
	return len(creds) > 0, nil
}

func (s *PasskeyService) isSettingEnabled(ctx context.Context, key string) bool {
	if s.settingsRepo == nil {
		return true
	}
	setting, err := s.settingsRepo.Get(ctx, key)
	if err != nil || setting == nil || setting.Value == "" {
		return true
	}
	return !strings.EqualFold(strings.TrimSpace(setting.Value), "false")
}
