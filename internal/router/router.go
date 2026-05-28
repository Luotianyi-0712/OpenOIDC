package router

import (
	"io/fs"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"

	"github.com/anthropic/oidc-platform/internal/handler"
	mw "github.com/anthropic/oidc-platform/internal/handler/middleware"
	"github.com/anthropic/oidc-platform/internal/port"
	"github.com/anthropic/oidc-platform/internal/service"
)

type Deps struct {
	AuthHandler      *handler.AuthHandler
	SocialHandler    *handler.SocialHandler
	OIDCHandler      *handler.OIDCHandler
	UserInfoHandler  *handler.UserInfoHandler
	AdminHandler     *handler.AdminHandler
	DeveloperHandler *handler.DeveloperHandler
	WellKnownHandler *handler.WellKnownHandler
	HealthHandler    *handler.HealthHandler
	PasskeyHandler   *handler.PasskeyHandler
	SessionService   *service.SessionService
	UserRepo         port.UserRepository
	SettingsRepo     port.SettingsRepository
	Cache            port.Cache
	AllowedOrigins   []string
	CookieName       string
	SPAFS            fs.FS
}

func NewRouter(d Deps) *chi.Mux {
	r := chi.NewRouter()

	r.Use(mw.RequestID)
	r.Use(chimw.Recoverer)
	r.Use(mw.RequestLogger)
	r.Use(mw.CORS(d.AllowedOrigins))

	// Public health and discovery.
	r.Get("/healthz", d.HealthHandler.Healthz)
	r.Get("/readyz", d.HealthHandler.Readyz)
	r.Get("/.well-known/openid-configuration", d.WellKnownHandler.Discovery)
	r.Get("/jwks.json", d.WellKnownHandler.JWKS)

	// OAuth2 / OIDC core endpoints.
	r.Route("/oauth2", func(r chi.Router) {
		if d.Cache != nil {
			r.Use(mw.RateLimit(d.Cache, 120, time.Minute))
		}
		r.With(mw.OptionalSessionAuth(d.SessionService, d.CookieName)).Get("/authorize", d.OIDCHandler.Authorize)
		if d.Cache != nil {
			r.With(mw.RateLimit(d.Cache, 60, time.Minute)).Post("/token", d.OIDCHandler.Token)
			r.With(mw.RateLimit(d.Cache, 60, time.Minute)).Post("/introspect", d.OIDCHandler.Introspect)
		} else {
			r.Post("/token", d.OIDCHandler.Token)
			r.Post("/introspect", d.OIDCHandler.Introspect)
		}
		r.Post("/revoke", d.OIDCHandler.Revoke)
		r.Get("/userinfo", d.OIDCHandler.UserInfo)
		r.Post("/userinfo", d.OIDCHandler.UserInfo)
	})

	r.Route("/api/v1", func(r chi.Router) {
		// Public settings (login/registration config).
		r.Get("/settings/public", d.AdminHandler.PublicSettings)
		r.Get("/settings/password-policy", d.AdminHandler.PasswordPolicy)

		// Auth (public, but rate limited).
		r.Route("/auth", func(r chi.Router) {
			if d.Cache != nil {
				r.Use(mw.DynamicRateLimit(d.Cache, d.SettingsRepo, 30, time.Minute))
			}
			r.With(mw.Turnstile(d.SettingsRepo)).Post("/register/code", d.AuthHandler.SendRegisterCode)
			r.With(mw.Turnstile(d.SettingsRepo)).Post("/register", d.AuthHandler.Register)
			r.With(mw.Turnstile(d.SettingsRepo)).Post("/login", d.AuthHandler.Login)
			r.Get("/status", d.AuthHandler.Status)
			r.Post("/logout", d.AuthHandler.Logout)
			r.Post("/verify-email", d.AuthHandler.VerifyEmail)
			r.With(mw.Turnstile(d.SettingsRepo)).Post("/forgot-password", d.AuthHandler.ForgotPassword)
			r.Post("/reset-password", d.AuthHandler.ResetPassword)
			r.Post("/passkey/begin", d.PasskeyHandler.BeginLogin)
			r.Post("/passkey/finish", d.PasskeyHandler.FinishLogin)
		})

		// Public: list enabled social providers (no auth needed).
		r.Get("/social/providers", d.SocialHandler.ListEnabled)

		// Social - login/binding callbacks. Optional session so callback can be either.
		r.Route("/social", func(r chi.Router) {
			if d.Cache != nil {
				r.Use(mw.RateLimit(d.Cache, 60, time.Minute))
			}
			r.Use(mw.OptionalSessionAuth(d.SessionService, d.CookieName))
			r.Get("/{provider}/begin", d.SocialHandler.Begin)
			r.Get("/{provider}/callback", d.SocialHandler.Callback)
		})

		// Authenticated user routes.
		r.Group(func(r chi.Router) {
			r.Use(mw.SessionAuth(d.SessionService, d.CookieName))

			r.Get("/me", d.UserInfoHandler.Me)
			r.Put("/me", d.UserInfoHandler.UpdateMe)
			r.Put("/me/alias", d.UserInfoHandler.SetAlias)
			r.Put("/me/password", d.UserInfoHandler.ChangePassword)
			r.Get("/me/bindings", d.UserInfoHandler.ListBindings)
			r.Delete("/me/bindings/{provider}", d.UserInfoHandler.Unbind)
			r.Get("/me/security-level", d.UserInfoHandler.SecurityLevel)
			r.Get("/me/activity", d.UserInfoHandler.ListActivity)
			r.Post("/me/resend-verification", d.UserInfoHandler.ResendVerification)
			r.Get("/me/sessions", d.UserInfoHandler.ListSessions)
			r.Delete("/me/sessions/{id}", d.UserInfoHandler.RevokeSession)
			r.Get("/me/authorized-apps", d.UserInfoHandler.ListAuthorizedApps)
			r.Delete("/me/authorized-apps/{clientId}", d.UserInfoHandler.RevokeAuthorizedApp)
			r.Post("/me/authorized-apps/{clientId}/report", d.UserInfoHandler.ReportApp)

			r.Post("/me/passkeys/register/begin", d.PasskeyHandler.BeginRegister)
			r.Post("/me/passkeys/register/finish", d.PasskeyHandler.FinishRegister)
			r.Get("/me/passkeys", d.PasskeyHandler.ListPasskeys)
			r.Delete("/me/passkeys/{id}", d.PasskeyHandler.DeletePasskey)
			r.Put("/me/passkeys/{id}", d.PasskeyHandler.RenamePasskey)

			r.Get("/consent/context", d.OIDCHandler.ConsentContext)
			r.Post("/consent/accept", d.OIDCHandler.ConsentAccept)
			r.Post("/consent/reject", d.OIDCHandler.ConsentReject)
		})

		// Developer routes. Existing app management remains available to owners;
		// creating new apps is checked by the developer status in the handler.
		r.Route("/developer", func(r chi.Router) {
			r.Use(mw.SessionAuth(d.SessionService, d.CookieName))
			if d.Cache != nil {
				r.Use(mw.RateLimit(d.Cache, 120, time.Minute))
			}
			r.Get("/status", d.DeveloperHandler.Status)
			r.Get("/apps", d.DeveloperHandler.ListApps)
			r.Post("/apps", d.DeveloperHandler.CreateApp)
			r.Get("/apps/{id}", d.DeveloperHandler.GetApp)
			r.Put("/apps/{id}", d.DeveloperHandler.UpdateApp)
			r.Delete("/apps/{id}", d.DeveloperHandler.DeleteApp)
			r.Post("/apps/{id}/rotate-secret", d.DeveloperHandler.RotateSecret)
			r.Get("/apps/{id}/users", d.DeveloperHandler.ListAppUsers)
			r.Post("/apps/{id}/users/{uid}/block", d.DeveloperHandler.BlockAppUser)
			r.Delete("/apps/{id}/users/{uid}/block", d.DeveloperHandler.UnblockAppUser)
			r.Post("/apps/{id}/users/{uid}/report", d.DeveloperHandler.ReportAppUser)
			r.Post("/apps/{id}/report-user", d.DeveloperHandler.ReportUser)
		})

		// Admin routes.
		r.Route("/admin", func(r chi.Router) {
			r.Use(mw.SessionAuth(d.SessionService, d.CookieName))
			r.Use(mw.AdminOnly(d.UserRepo))

			r.Get("/users", d.AdminHandler.ListUsers)
			r.Post("/users", d.AdminHandler.CreateUser)
			r.Get("/users/{id}", d.AdminHandler.GetUser)
			r.Get("/users/{id}/detail", d.AdminHandler.GetUserDetail)
			r.Put("/users/{id}", d.AdminHandler.UpdateUser)
			r.Delete("/users/{id}", d.AdminHandler.DeleteUser)
			r.Get("/users/{id}/clients", d.AdminHandler.ListUserClients)
			r.Get("/users/{id}/passkeys", d.AdminHandler.ListUserPasskeys)
			r.Delete("/users/{id}/passkeys/{passkey_id}", d.AdminHandler.DeleteUserPasskey)
			r.Put("/users/{id}/security-level", d.AdminHandler.OverrideSecurityLevel)
			r.Post("/users/{id}/reset-password", d.AdminHandler.ResetUserPassword)
			r.Delete("/users/{id}/sessions/{session_id}", d.AdminHandler.RevokeUserSession)
			r.Delete("/users/{id}/bindings/{provider}", d.AdminHandler.UnbindUserSocial)

			r.Get("/clients", d.AdminHandler.ListClients)
			r.Post("/clients", d.AdminHandler.CreateClient)
			r.Get("/clients/{id}", d.AdminHandler.GetClient)
			r.Put("/clients/{id}", d.AdminHandler.UpdateClient)
			r.Delete("/clients/{id}", d.AdminHandler.DeleteClient)
			r.Post("/clients/{id}/rotate-secret", d.AdminHandler.RotateClientSecret)
			r.Get("/clients/{id}/users", d.AdminHandler.ListClientUsers)
			r.Post("/clients/{id}/users/{uid}/block", d.AdminHandler.BlockClientUser)
			r.Delete("/clients/{id}/users/{uid}/block", d.AdminHandler.UnblockClientUser)
			r.Delete("/clients/{id}/users/{uid}/authorization", d.AdminHandler.RevokeClientUserAuthorization)
			r.Get("/clients/{id}/access-rules", d.AdminHandler.ListClientAccessRules)
			r.Post("/clients/{id}/access-rules", d.AdminHandler.AddClientAccessRule)
			r.Delete("/clients/{id}/access-rules/{rid}", d.AdminHandler.RemoveClientAccessRule)

			r.Get("/security-rules", d.AdminHandler.ListSecurityRules)
			r.Post("/security-rules", d.AdminHandler.CreateSecurityRule)
			r.Get("/security-rules/{id}", d.AdminHandler.GetSecurityRule)
			r.Put("/security-rules/{id}", d.AdminHandler.UpdateSecurityRule)
			r.Delete("/security-rules/{id}", d.AdminHandler.DeleteSecurityRule)
			r.Post("/security-rules/recompute", d.AdminHandler.RecomputeSecurityLevels)

			r.Get("/providers", d.AdminHandler.ListProviders)
			r.Post("/providers", d.AdminHandler.CreateProvider)
			r.Get("/providers/{provider}", d.AdminHandler.GetProvider)
			r.Put("/providers/{provider}", d.AdminHandler.UpdateProvider)
			r.Delete("/providers/{provider}", d.AdminHandler.DeleteProvider)

			r.Get("/alias-restrictions", d.AdminHandler.ListAliasRestrictions)
			r.Post("/alias-restrictions", d.AdminHandler.CreateAliasRestriction)
			r.Delete("/alias-restrictions/{id}", d.AdminHandler.DeleteAliasRestriction)

			r.Get("/settings", d.AdminHandler.ListSettings)
			r.Put("/settings/{key}", d.AdminHandler.UpdateSetting)
			r.Get("/version/check", d.AdminHandler.CheckVersionUpdate)

			r.Get("/audit-log", d.AdminHandler.ListAuditLogs)

			r.Get("/stats", d.AdminHandler.Stats)

			r.Get("/keys", d.AdminHandler.ListKeys)
			r.Post("/keys/rotate", d.AdminHandler.RotateKey)

			r.Get("/risk/policy", d.AdminHandler.GetRiskPolicy)
			r.Put("/risk/policy", d.AdminHandler.UpdateRiskPolicy)
			r.Get("/risk/reports", d.AdminHandler.ListRiskReports)
			r.Put("/risk/reports/{id}/confirm", d.AdminHandler.ConfirmRiskReport)
			r.Put("/risk/reports/{id}/dismiss", d.AdminHandler.DismissRiskReport)
			r.Get("/risk/list", d.AdminHandler.ListRiskList)
			r.Post("/risk/list", d.AdminHandler.AddRiskEntry)
			r.Delete("/risk/list/{id}", d.AdminHandler.RemoveRiskEntry)
		})
	})

	// SPA: serve Vue frontend for all non-API routes.
	if d.SPAFS != nil {
		spa := handler.NewSPAHandler(d.SPAFS)
		r.NotFound(spa.ServeHTTP)
	} else {
		r.NotFound(func(w http.ResponseWriter, r *http.Request) {
			handler.Error(w, http.StatusNotFound, "not_found", "resource not found")
		})
	}
	r.MethodNotAllowed(func(w http.ResponseWriter, r *http.Request) {
		handler.Error(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
	})

	return r
}
