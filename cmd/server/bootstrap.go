package main

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/ory/fosite"
	goredis "github.com/redis/go-redis/v9"
	"golang.org/x/crypto/argon2"

	"github.com/anthropic/oidc-platform/internal/adapter/memcache"
	"github.com/anthropic/oidc-platform/internal/adapter/postgres"
	"github.com/anthropic/oidc-platform/internal/adapter/redis"
	"github.com/anthropic/oidc-platform/internal/adapter/smtp"
	"github.com/anthropic/oidc-platform/internal/adapter/social"
	sqliteAdapter "github.com/anthropic/oidc-platform/internal/adapter/sqlite"
	"github.com/anthropic/oidc-platform/internal/config"
	"github.com/anthropic/oidc-platform/internal/domain"
	"github.com/anthropic/oidc-platform/internal/handler"
	"github.com/anthropic/oidc-platform/internal/oidcprovider"
	"github.com/anthropic/oidc-platform/internal/port"
	"github.com/anthropic/oidc-platform/internal/router"
	"github.com/anthropic/oidc-platform/internal/service"
)

// bootstrap initializes all infrastructure and returns assembled router deps and a cleanup func.
func bootstrap(ctx context.Context, cfg *config.Config) (router.Deps, func(), error) {
	cleanup := func() {}

	var (
		userRepo        port.UserRepository
		bindingRepo     port.BindingRepository
		clientRepo      port.ClientRepository
		accessRuleRepo  port.ClientAccessRuleRepository
		ruleRepo        port.RuleRepository
		sessionRepo     port.SessionRepository
		providerCfgRepo port.ProviderConfigRepository
		auditRepo       port.AuditRepository
		settingsRepo    port.SettingsRepository
		aliasRepo       port.AliasRestrictionRepository
		signingKeyRepo  port.SigningKeyRepository
		riskReportRepo  port.RiskReportRepository
		riskListRepo    port.RiskListRepository
		consentRepo     port.ConsentRepository
		cache           port.Cache
		fositeStore     fosite.Storage
	)

	// Variables for health handler (nil when using sqlite).
	var pgPool *pgxpool.Pool
	var redisClient *goredis.Client

	switch cfg.Database.Driver {
	case "sqlite":
		sqliteDB, err := sqliteAdapter.NewDB(ctx, cfg.Database.DSN)
		if err != nil {
			return router.Deps{}, cleanup, fmt.Errorf("sqlite: %w", err)
		}
		cleanup = chainCleanup(cleanup, func() { sqliteDB.Close() })

		if err := sqliteAdapter.RunMigrations(sqliteDB); err != nil {
			return router.Deps{}, cleanup, fmt.Errorf("sqlite migrations: %w", err)
		}

		userRepo = sqliteAdapter.NewUserRepo(sqliteDB)
		bindingRepo = sqliteAdapter.NewBindingRepo(sqliteDB)
		clientRepo = sqliteAdapter.NewClientRepo(sqliteDB)
		accessRuleRepo = sqliteAdapter.NewClientAccessRuleRepo(sqliteDB)
		ruleRepo = sqliteAdapter.NewRuleRepo(sqliteDB)
		sessionRepo = sqliteAdapter.NewSessionRepo(sqliteDB)
		providerCfgRepo = sqliteAdapter.NewProviderConfigRepo(sqliteDB)
		auditRepo = sqliteAdapter.NewAuditRepo(sqliteDB)
		settingsRepo = sqliteAdapter.NewSettingsRepo(sqliteDB)
		aliasRepo = sqliteAdapter.NewAliasRestrictionRepo(sqliteDB)
		signingKeyRepo = sqliteAdapter.NewSigningKeyRepo(sqliteDB)
		riskReportRepo = sqliteAdapter.NewRiskReportRepo(sqliteDB)
		riskListRepo = sqliteAdapter.NewRiskListRepo(sqliteDB)
		consentRepo = sqliteAdapter.NewConsentRepo(sqliteDB)
		fositeStore = sqliteAdapter.NewFositeStore(sqliteDB)

		mc := memcache.NewMemCache()
		cleanup = chainCleanup(cleanup, func() { _ = mc.Close() })
		cache = mc

	default: // "postgres"
		db, err := postgres.NewDB(ctx, cfg.Database)
		if err != nil {
			return router.Deps{}, cleanup, fmt.Errorf("postgres: %w", err)
		}
		cleanup = chainCleanup(cleanup, func() { db.Close() })
		pgPool = db

		if migrationsPath := os.Getenv("OIDC_MIGRATIONS_PATH"); migrationsPath != "" {
			if err := postgres.RunMigrations(cfg.Database, migrationsPath); err != nil {
				slog.Warn("run migrations", "error", err)
			}
		}

		redisCache, err := redis.NewCache(ctx, cfg.Redis)
		if err != nil {
			return router.Deps{}, cleanup, fmt.Errorf("redis: %w", err)
		}
		cleanup = chainCleanup(cleanup, func() { _ = redisCache.Close() })
		redisClient = redisCache.Client()
		cache = redisCache

		userRepo = postgres.NewUserRepo(db)
		bindingRepo = postgres.NewBindingRepo(db)
		clientRepo = postgres.NewClientRepo(db)
		accessRuleRepo = postgres.NewClientAccessRuleRepo(db)
		ruleRepo = postgres.NewRuleRepo(db)
		sessionRepo = postgres.NewSessionRepo(db)
		providerCfgRepo = postgres.NewProviderConfigRepo(db)
		auditRepo = postgres.NewAuditRepo(db)
		settingsRepo = postgres.NewSettingsRepo(db)
		aliasRepo = postgres.NewAliasRestrictionRepo(db)
		signingKeyRepo = postgres.NewSigningKeyRepo(db)
		riskReportRepo = postgres.NewRiskReportRepo(db)
		riskListRepo = postgres.NewRiskListRepo(db)
		consentRepo = postgres.NewConsentRepo(db)
		fositeStore = postgres.NewFositeStore(db)
	}
	// settingsRepo is used by devHandler above.

	// Social provider registry.
	socialRegistry := buildSocialRegistry(cfg, providerCfgRepo)

	// Email sender.
	baseURL := cfg.Server.BaseURL
	if baseURL == "" {
		baseURL = fmt.Sprintf("http://localhost:%d", cfg.Server.Port)
	}
	emailSender := smtp.NewSender(cfg.SMTP, baseURL, settingsRepo)

	// Services.
	securitySvc := service.NewSecurityLevelService(ruleRepo, bindingRepo, userRepo, auditRepo)
	authSvc := service.NewAuthService(userRepo, sessionRepo, cache, auditRepo, emailSender, settingsRepo, cfg)
	sessionSvc := service.NewSessionService(sessionRepo, cfg)
	socialSvc := service.NewSocialService(bindingRepo, userRepo, socialRegistry, cache, securitySvc, sessionRepo, auditRepo, settingsRepo, riskListRepo, cfg)
	clientSvc := service.NewClientService(clientRepo, accessRuleRepo, auditRepo)
	adminSvc := service.NewAdminService(userRepo, providerCfgRepo, settingsRepo, aliasRepo, signingKeyRepo, auditRepo)
	accessCtrl := service.NewAccessControlService(accessRuleRepo, aliasRepo)
	riskSvc := service.NewRiskService(riskReportRepo, riskListRepo, bindingRepo, userRepo, auditRepo, securitySvc)

	// OIDC provider.
	privKey, err := loadOrCreateSigningKey(ctx, signingKeyRepo, adminSvc)
	if err != nil {
		return router.Deps{}, cleanup, fmt.Errorf("signing key: %w", err)
	}
	oauth2Secret := []byte(os.Getenv("OIDC_OAUTH2_SECRET"))
	if len(oauth2Secret) < 32 {
		slog.Warn("OIDC_OAUTH2_SECRET is not set or too short (min 32 chars), using random secret (sessions will not survive restart)")
		randomBytes := make([]byte, 32)
		if _, err := rand.Read(randomBytes); err != nil {
			return router.Deps{}, cleanup, fmt.Errorf("generate random secret: %w", err)
		}
		oauth2Secret = randomBytes
	}
	provider := oidcprovider.NewOAuth2Provider(fositeStore, oauth2Secret, privKey, cfg.Server.Issuer)

	// Handlers.
	authHandler := handler.NewAuthHandler(authSvc, sessionSvc, cfg.Session)
	socialHandler := handler.NewSocialHandler(socialSvc, socialRegistry, sessionSvc, cfg.Session)
	userInfoHandler := handler.NewUserInfoHandler(userRepo, socialSvc, securitySvc, accessCtrl, authSvc, sessionSvc, consentRepo)
	adminHandler := handler.NewAdminHandler(adminSvc, clientSvc, securitySvc, userRepo, socialRegistry, riskSvc, sessionRepo, bindingRepo, consentRepo)
	loginURL := cfg.Server.BaseURL + "/login"
	oidcHandler := handler.NewOIDCHandler(provider, userRepo, clientSvc, accessCtrl, sessionSvc, cfg.Server, loginURL)
	oidcHandler.SetCache(cache)
	wellKnownHandler := handler.NewWellKnownHandler(cfg.Server.BaseURL, signingKeyRepo)
	devHandler := handler.NewDeveloperHandler(clientSvc, riskSvc, settingsRepo, consentRepo)
	healthHandler := handler.NewHealthHandler(pgPool, redisClient)

	allowedOrigins := []string{"*"}
	if v := os.Getenv("OIDC_ALLOWED_ORIGINS"); v != "" {
		allowedOrigins = strings.Split(v, ",")
	}

	startBackgroundJobs(ctx, sessionRepo, securitySvc)

	if cfg.Admin.Email != "" && cfg.Admin.Password != "" {
		if err := seedAdmin(ctx, userRepo, cfg.Admin); err != nil {
			slog.Warn("seed admin", "error", err)
		}
	}

	seedProviders(ctx, providerCfgRepo)
	seedSettings(ctx, settingsRepo)

	return router.Deps{
		AuthHandler:      authHandler,
		SocialHandler:    socialHandler,
		OIDCHandler:      oidcHandler,
		UserInfoHandler:  userInfoHandler,
		AdminHandler:     adminHandler,
		DeveloperHandler: devHandler,
		WellKnownHandler: wellKnownHandler,
		HealthHandler:    healthHandler,
		SessionService:   sessionSvc,
		UserRepo:         userRepo,
		SettingsRepo:     settingsRepo,
		Cache:            cache,
		AllowedOrigins:   allowedOrigins,
		CookieName:       cfg.Session.CookieName,
		SPAFS:            locateSPAFS(),
	}, cleanup, nil
}

func locateSPAFS() fs.FS {
	for _, dir := range []string{"frontend/dist", "../frontend/dist", "dist"} {
		if info, err := os.Stat(dir); err == nil && info.IsDir() {
			slog.Info("serving SPA from", "dir", dir)
			return os.DirFS(dir)
		}
	}
	slog.Warn("no frontend dist found, SPA disabled")
	return nil
}

func chainCleanup(a, b func()) func() {
	return func() {
		b()
		a()
	}
}

func buildSocialRegistry(cfg *config.Config, providerCfgRepo port.ProviderConfigRepository) port.SocialProviderRegistry {
	reg := social.NewRegistry(providerCfgRepo)

	// Load from config.yaml first (static config).
	for _, name := range domain.AllProviders() {
		pcfg, ok := cfg.OAuth2.Providers[name]
		if !ok || !pcfg.Enabled {
			continue
		}
		var p port.SocialProvider
		switch name {
		case domain.ProviderGitHub:
			p = social.NewGitHubProvider(pcfg.ClientID, pcfg.ClientSecret)
		case domain.ProviderGoogle:
			p = social.NewGoogleProvider(pcfg.ClientID, pcfg.ClientSecret)
		case domain.ProviderGitLab:
			p = social.NewGitLabProvider(pcfg.ClientID, pcfg.ClientSecret, "")
		case domain.ProviderGitee:
			p = social.NewGiteeProvider(pcfg.ClientID, pcfg.ClientSecret)
		case domain.ProviderDiscord:
			p = social.NewDiscordProvider(pcfg.ClientID, pcfg.ClientSecret)
		case domain.ProviderMicrosoft:
			p = social.NewMicrosoftProvider(pcfg.ClientID, pcfg.ClientSecret, "")
		case domain.ProviderQQ:
			p = social.NewQQProvider(pcfg.ClientID, pcfg.ClientSecret)
		case domain.ProviderWeChat:
			p = social.NewWeChatProvider(pcfg.AppID, pcfg.AppSecret)
		case domain.ProviderTelegram:
			p = social.NewTelegramProvider(pcfg.AppSecret)
		default:
			slog.Warn("provider not wired", "provider", name)
			continue
		}
		if p != nil {
			reg.Register(p)
		}
	}

	// Then overlay with DB-stored configs (admin-configured providers).
	if err := reg.Reload(context.Background()); err != nil {
		slog.Warn("reload social providers from DB", "error", err)
	}

	return reg
}

func loadOrCreateSigningKey(ctx context.Context, repo port.SigningKeyRepository, adminSvc *service.AdminService) (*rsa.PrivateKey, error) {
	k, err := repo.GetCurrent(ctx)
	if err == nil && k != nil {
		key, err := parsePrivateKey(k.PrivateKey)
		if err != nil {
			return nil, err
		}
		rk, ok := key.(*rsa.PrivateKey)
		if !ok {
			return nil, fmt.Errorf("not an RSA private key")
		}
		return rk, nil
	}
	// Bootstrap: create the first signing key via admin service.
	newKey, err := adminSvc.RotateSigningKey(ctx)
	if err != nil {
		return nil, fmt.Errorf("create initial signing key: %w", err)
	}
	key, err := parsePrivateKey(newKey.PrivateKey)
	if err != nil {
		return nil, err
	}
	rk, ok := key.(*rsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("not an RSA private key")
	}
	return rk, nil
}

func seedAdmin(ctx context.Context, userRepo port.UserRepository, cfg config.AdminConfig) error {
	existing, err := userRepo.GetByEmail(ctx, cfg.Email)
	if err != nil && !errors.Is(err, port.ErrNotFound) {
		return err
	}
	if existing != nil {
		return nil
	}

	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return err
	}
	key := argon2.IDKey([]byte(cfg.Password), salt, 1, 64*1024, 4, 32)
	hash := fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version, 64*1024, 1, 4,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(key))

	now := time.Now().UTC()
	admin := &domain.User{
		ID:            uuid.New(),
		Email:         cfg.Email,
		EmailVerified: true,
		PasswordHash:  hash,
		DisplayName:   "Admin",
		Role:          domain.RoleSuperAdmin,
		Status:        domain.UserStatusActive,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if err := userRepo.Create(ctx, admin); err != nil {
		return err
	}
	slog.Info("admin account created", "email", cfg.Email)
	return nil
}

func seedProviders(ctx context.Context, repo port.ProviderConfigRepository) {
	displayNames := map[string]string{
		domain.ProviderGitHub:    "GitHub",
		domain.ProviderGoogle:    "Google",
		domain.ProviderGitLab:    "GitLab",
		domain.ProviderGitee:     "Gitee",
		domain.ProviderDiscord:   "Discord",
		domain.ProviderTelegram:  "Telegram",
		domain.ProviderMicrosoft: "Microsoft",
		domain.ProviderApple:     "Apple",
		domain.ProviderQQ:        "QQ",
		domain.ProviderWeChat:    "WeChat",
		domain.ProviderPhone:     "Phone",
	}

	existing, _ := repo.List(ctx)
	have := make(map[string]bool, len(existing))
	for _, pc := range existing {
		have[pc.Provider] = true
	}

	for i, name := range domain.AllProviders() {
		if have[name] {
			continue
		}
		pc := &domain.ProviderConfig{
			ID:          uuid.New(),
			Provider:    name,
			DisplayName: displayNames[name],
			IsEnabled:   false,
			SortOrder:   i,
			CreatedAt:   time.Now().UTC(),
			UpdatedAt:   time.Now().UTC(),
		}
		if err := repo.Upsert(ctx, pc); err != nil {
			slog.Warn("seed provider", "provider", name, "error", err)
		}
	}
}

func seedSettings(ctx context.Context, repo port.SettingsRepository) {
	defaults := map[string]string{
		"registration_enabled":    "true",
		"password_login_enabled":  "true",
		"social_login_enabled":    "true",
		"social_register_enabled": "true",
	}
	for key, value := range defaults {
		if _, err := repo.Get(ctx, key); err != nil {
			// Setting does not exist yet; seed the default.
			if err := repo.Upsert(ctx, key, value, ""); err != nil {
				slog.Warn("seed setting", "key", key, "error", err)
			}
		}
	}
}

func startBackgroundJobs(ctx context.Context, sessionRepo port.SessionRepository, securitySvc *service.SecurityLevelService) {
	go func() {
		ticker := time.NewTicker(15 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := sessionRepo.DeleteExpired(ctx); err != nil {
					slog.Warn("cleanup expired sessions", "error", err)
				}
			}
		}
	}()

	go func() {
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := securitySvc.RecomputeAll(ctx); err != nil {
					slog.Warn("recompute security levels", "error", err)
				}
			}
		}
	}()
}
