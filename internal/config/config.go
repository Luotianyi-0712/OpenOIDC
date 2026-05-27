package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	Server         ServerConfig
	Database       DatabaseConfig
	Redis          RedisConfig
	JWT            JWTConfig
	Session        SessionConfig
	Admin          AdminConfig
	Log            LogConfig
	Security       SecurityConfig
	OAuth2         OAuth2Config
	SocialAuthSync SocialAuthSyncConfig `mapstructure:"social_auth_sync"`
	SMS            SMSConfig
	SMTP           SMTPConfig
	Secrets        SecretsConfig
	WebAuthn       WebAuthnConfig `mapstructure:"webauthn"`
}

type ServerConfig struct {
	Host            string
	Port            int
	Issuer          string
	BaseURL         string        `mapstructure:"public_url"`
	ReadTimeout     time.Duration `mapstructure:"read_timeout"`
	WriteTimeout    time.Duration `mapstructure:"write_timeout"`
	ShutdownTimeout time.Duration `mapstructure:"shutdown_timeout"`
}

type DatabaseConfig struct {
	Driver          string `mapstructure:"driver"`
	DSN             string `mapstructure:"dsn"`
	Host            string
	Port            int
	User            string
	Password        string
	DBName          string        `mapstructure:"name"`
	SSLMode         string        `mapstructure:"sslmode"`
	MaxOpenConns    int           `mapstructure:"max_conns"`
	MaxIdleConns    int           `mapstructure:"min_conns"`
	MaxConnLifetime time.Duration `mapstructure:"max_conn_lifetime"`
	MaxConnIdleTime time.Duration `mapstructure:"max_conn_idle_time"`
}

type RedisConfig struct {
	Host         string
	Port         int
	Password     string
	DB           int
	PoolSize     int `mapstructure:"pool_size"`
	MinIdleConns int `mapstructure:"min_idle_conns"`
}

type JWTConfig struct {
	SigningMethod   string        `mapstructure:"algorithm"`
	AccessTokenTTL  time.Duration `mapstructure:"access_token_ttl"`
	RefreshTokenTTL time.Duration `mapstructure:"refresh_token_ttl"`
	IDTokenTTL      time.Duration `mapstructure:"id_token_ttl"`
	AuthCodeTTL     time.Duration `mapstructure:"auth_code_ttl"`
	KeyRotationDays int           `mapstructure:"key_rotation_days"`
}

type SessionConfig struct {
	CookieName     string        `mapstructure:"cookie_name"`
	CookieDomain   string        `mapstructure:"cookie_domain"`
	CookieSecure   bool          `mapstructure:"cookie_secure"`
	CookieHTTPOnly bool          `mapstructure:"cookie_http_only"`
	CookieSameSite string        `mapstructure:"cookie_same_site"`
	TTL            time.Duration `mapstructure:"ttl"`
}

type AdminConfig struct {
	Email    string
	Password string
}

type LogConfig struct {
	Level  string
	Format string
}

type SecurityConfig struct {
	PasswordMinLength     int           `mapstructure:"password_min_length"`
	PasswordRequireUpper  bool          `mapstructure:"password_require_upper"`
	PasswordRequireLower  bool          `mapstructure:"password_require_lower"`
	PasswordRequireDigit  bool          `mapstructure:"password_require_digit"`
	PasswordRequireSymbol bool          `mapstructure:"password_require_symbol"`
	MaxLoginAttempts      int           `mapstructure:"max_login_attempts"`
	LockoutDuration       time.Duration `mapstructure:"lockout_duration"`
	BcryptCost            int           `mapstructure:"bcrypt_cost"`
}

type OAuth2Config struct {
	Providers map[string]ProviderOAuth2Config `mapstructure:",remain"`
}

type SocialAuthSyncConfig struct {
	Enabled   bool          `mapstructure:"enabled"`
	Interval  time.Duration `mapstructure:"interval"`
	BatchSize int           `mapstructure:"batch_size"`
}

type ProviderOAuth2Config struct {
	Enabled      bool     `mapstructure:"enabled"`
	ClientID     string   `mapstructure:"client_id"`
	ClientSecret string   `mapstructure:"client_secret"`
	AppID        string   `mapstructure:"app_id"`
	AppSecret    string   `mapstructure:"app_secret"`
	AppKey       string   `mapstructure:"app_key"`
	TeamID       string   `mapstructure:"team_id"`
	KeyID        string   `mapstructure:"key_id"`
	PrivateKey   string   `mapstructure:"private_key"`
	Tenant       string   `mapstructure:"tenant"`
	Scopes       []string `mapstructure:"scopes"`
	RedirectPath string   `mapstructure:"redirect_path"`
}

type SMSConfig struct {
	Provider     string        `mapstructure:"provider"`
	AccessKey    string        `mapstructure:"access_key"`
	AccessSecret string        `mapstructure:"access_secret"`
	SignName     string        `mapstructure:"sign_name"`
	TemplateCode string        `mapstructure:"template_code"`
	CodeTTL      time.Duration `mapstructure:"code_ttl"`
	SendInterval time.Duration `mapstructure:"send_interval"`
	DailyLimit   int           `mapstructure:"daily_limit"`
}

type SMTPConfig struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string
}

type SecretsConfig struct {
	ClientSecretEncryptionKey string `mapstructure:"client_secret_encryption_key"`
}

type WebAuthnConfig struct {
	RPID          string `mapstructure:"rp_id"`
	RPOrigin      string `mapstructure:"rp_origin"`
	RPDisplayName string `mapstructure:"rp_display_name"`
}

func Load() (*Config, error) {
	v := viper.New()
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath("./configs")
	v.AddConfigPath(".")
	v.AddConfigPath("/etc/oidc")

	v.SetEnvPrefix("OIDC")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	setDefaults(v)
	bindEnv(v)

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("read config: %w", err)
		}
	}

	cfg := &Config{}
	if err := v.Unmarshal(cfg); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}
	applyEnvOverrides(cfg)

	providers := map[string]ProviderOAuth2Config{}
	if oauth2 := v.GetStringMap("oauth2"); oauth2 != nil {
		for name := range oauth2 {
			sub := v.Sub("oauth2." + name)
			if sub == nil {
				continue
			}
			var pc ProviderOAuth2Config
			if err := sub.Unmarshal(&pc); err != nil {
				return nil, fmt.Errorf("unmarshal oauth2.%s: %w", name, err)
			}
			providers[name] = pc
		}
	}
	cfg.OAuth2.Providers = providers

	return cfg, nil
}

func applyEnvOverrides(cfg *Config) {
	setString := func(name string, target *string) {
		if value, ok := os.LookupEnv(name); ok {
			*target = value
		}
	}
	setInt := func(name string, target *int) {
		if value, ok := os.LookupEnv(name); ok {
			parsed, err := strconv.Atoi(strings.TrimSpace(value))
			if err == nil {
				*target = parsed
			}
		}
	}
	setBool := func(name string, target *bool) {
		if value, ok := os.LookupEnv(name); ok {
			parsed, err := strconv.ParseBool(strings.TrimSpace(value))
			if err == nil {
				*target = parsed
			}
		}
	}
	setDuration := func(name string, target *time.Duration) {
		if value, ok := os.LookupEnv(name); ok {
			parsed, err := time.ParseDuration(strings.TrimSpace(value))
			if err == nil {
				*target = parsed
			}
		}
	}

	setString("OIDC_SERVER_HOST", &cfg.Server.Host)
	setInt("OIDC_SERVER_PORT", &cfg.Server.Port)
	setString("OIDC_SERVER_ISSUER", &cfg.Server.Issuer)
	setString("OIDC_SERVER_PUBLIC_URL", &cfg.Server.BaseURL)
	setDuration("OIDC_SERVER_READ_TIMEOUT", &cfg.Server.ReadTimeout)
	setDuration("OIDC_SERVER_WRITE_TIMEOUT", &cfg.Server.WriteTimeout)
	setDuration("OIDC_SERVER_SHUTDOWN_TIMEOUT", &cfg.Server.ShutdownTimeout)

	setString("OIDC_DATABASE_DRIVER", &cfg.Database.Driver)
	setString("OIDC_DATABASE_DSN", &cfg.Database.DSN)
	setString("OIDC_DATABASE_HOST", &cfg.Database.Host)
	setInt("OIDC_DATABASE_PORT", &cfg.Database.Port)
	setString("OIDC_DATABASE_USER", &cfg.Database.User)
	setString("OIDC_DATABASE_PASSWORD", &cfg.Database.Password)
	setString("OIDC_DATABASE_NAME", &cfg.Database.DBName)
	setString("OIDC_DATABASE_SSLMODE", &cfg.Database.SSLMode)
	setInt("OIDC_DATABASE_MAX_CONNS", &cfg.Database.MaxOpenConns)
	setInt("OIDC_DATABASE_MIN_CONNS", &cfg.Database.MaxIdleConns)
	setDuration("OIDC_DATABASE_MAX_CONN_LIFETIME", &cfg.Database.MaxConnLifetime)
	setDuration("OIDC_DATABASE_MAX_CONN_IDLE_TIME", &cfg.Database.MaxConnIdleTime)

	setString("OIDC_REDIS_HOST", &cfg.Redis.Host)
	setInt("OIDC_REDIS_PORT", &cfg.Redis.Port)
	setString("OIDC_REDIS_PASSWORD", &cfg.Redis.Password)
	setInt("OIDC_REDIS_DB", &cfg.Redis.DB)
	setInt("OIDC_REDIS_POOL_SIZE", &cfg.Redis.PoolSize)
	setInt("OIDC_REDIS_MIN_IDLE_CONNS", &cfg.Redis.MinIdleConns)

	setString("OIDC_JWT_ALGORITHM", &cfg.JWT.SigningMethod)
	setDuration("OIDC_JWT_ACCESS_TOKEN_TTL", &cfg.JWT.AccessTokenTTL)
	setDuration("OIDC_JWT_REFRESH_TOKEN_TTL", &cfg.JWT.RefreshTokenTTL)
	setDuration("OIDC_JWT_ID_TOKEN_TTL", &cfg.JWT.IDTokenTTL)
	setDuration("OIDC_JWT_AUTH_CODE_TTL", &cfg.JWT.AuthCodeTTL)
	setInt("OIDC_JWT_KEY_ROTATION_DAYS", &cfg.JWT.KeyRotationDays)

	setString("OIDC_SESSION_COOKIE_NAME", &cfg.Session.CookieName)
	setString("OIDC_SESSION_COOKIE_DOMAIN", &cfg.Session.CookieDomain)
	setBool("OIDC_SESSION_COOKIE_SECURE", &cfg.Session.CookieSecure)
	setBool("OIDC_SESSION_COOKIE_HTTP_ONLY", &cfg.Session.CookieHTTPOnly)
	setString("OIDC_SESSION_COOKIE_SAME_SITE", &cfg.Session.CookieSameSite)
	setDuration("OIDC_SESSION_TTL", &cfg.Session.TTL)

	setString("OIDC_ADMIN_EMAIL", &cfg.Admin.Email)
	setString("OIDC_ADMIN_PASSWORD", &cfg.Admin.Password)
	setString("OIDC_LOG_LEVEL", &cfg.Log.Level)
	setString("OIDC_LOG_FORMAT", &cfg.Log.Format)

	setInt("OIDC_SECURITY_PASSWORD_MIN_LENGTH", &cfg.Security.PasswordMinLength)
	setBool("OIDC_SECURITY_PASSWORD_REQUIRE_UPPER", &cfg.Security.PasswordRequireUpper)
	setBool("OIDC_SECURITY_PASSWORD_REQUIRE_LOWER", &cfg.Security.PasswordRequireLower)
	setBool("OIDC_SECURITY_PASSWORD_REQUIRE_DIGIT", &cfg.Security.PasswordRequireDigit)
	setBool("OIDC_SECURITY_PASSWORD_REQUIRE_SYMBOL", &cfg.Security.PasswordRequireSymbol)
	setInt("OIDC_SECURITY_MAX_LOGIN_ATTEMPTS", &cfg.Security.MaxLoginAttempts)
	setDuration("OIDC_SECURITY_LOCKOUT_DURATION", &cfg.Security.LockoutDuration)
	setInt("OIDC_SECURITY_BCRYPT_COST", &cfg.Security.BcryptCost)

	setBool("OIDC_SOCIAL_AUTH_SYNC_ENABLED", &cfg.SocialAuthSync.Enabled)
	setDuration("OIDC_SOCIAL_AUTH_SYNC_INTERVAL", &cfg.SocialAuthSync.Interval)
	setInt("OIDC_SOCIAL_AUTH_SYNC_BATCH_SIZE", &cfg.SocialAuthSync.BatchSize)

	setString("OIDC_SMS_PROVIDER", &cfg.SMS.Provider)
	setString("OIDC_SMS_ACCESS_KEY", &cfg.SMS.AccessKey)
	setString("OIDC_SMS_ACCESS_SECRET", &cfg.SMS.AccessSecret)
	setString("OIDC_SMS_SIGN_NAME", &cfg.SMS.SignName)
	setString("OIDC_SMS_TEMPLATE_CODE", &cfg.SMS.TemplateCode)
	setDuration("OIDC_SMS_CODE_TTL", &cfg.SMS.CodeTTL)
	setDuration("OIDC_SMS_SEND_INTERVAL", &cfg.SMS.SendInterval)
	setInt("OIDC_SMS_DAILY_LIMIT", &cfg.SMS.DailyLimit)

	setString("OIDC_SMTP_HOST", &cfg.SMTP.Host)
	setInt("OIDC_SMTP_PORT", &cfg.SMTP.Port)
	setString("OIDC_SMTP_USERNAME", &cfg.SMTP.Username)
	setString("OIDC_SMTP_PASSWORD", &cfg.SMTP.Password)
	setString("OIDC_SMTP_FROM", &cfg.SMTP.From)

	setString("OIDC_SECRETS_CLIENT_SECRET_ENCRYPTION_KEY", &cfg.Secrets.ClientSecretEncryptionKey)
	setString("OIDC_WEBAUTHN_RP_ID", &cfg.WebAuthn.RPID)
	setString("OIDC_WEBAUTHN_RP_ORIGIN", &cfg.WebAuthn.RPOrigin)
	setString("OIDC_WEBAUTHN_RP_DISPLAY_NAME", &cfg.WebAuthn.RPDisplayName)
}

func bindEnv(v *viper.Viper) {
	keys := []string{
		"server.host",
		"server.port",
		"server.issuer",
		"server.public_url",
		"server.read_timeout",
		"server.write_timeout",
		"server.shutdown_timeout",
		"database.driver",
		"database.dsn",
		"database.host",
		"database.port",
		"database.user",
		"database.password",
		"database.name",
		"database.sslmode",
		"database.max_conns",
		"database.min_conns",
		"database.max_conn_lifetime",
		"database.max_conn_idle_time",
		"redis.host",
		"redis.port",
		"redis.password",
		"redis.db",
		"redis.pool_size",
		"redis.min_idle_conns",
		"jwt.algorithm",
		"jwt.access_token_ttl",
		"jwt.refresh_token_ttl",
		"jwt.id_token_ttl",
		"jwt.auth_code_ttl",
		"jwt.key_rotation_days",
		"session.cookie_name",
		"session.cookie_domain",
		"session.cookie_secure",
		"session.cookie_http_only",
		"session.cookie_same_site",
		"session.ttl",
		"admin.email",
		"admin.password",
		"log.level",
		"log.format",
		"security.password_min_length",
		"security.password_require_upper",
		"security.password_require_lower",
		"security.password_require_digit",
		"security.password_require_symbol",
		"security.max_login_attempts",
		"security.lockout_duration",
		"security.bcrypt_cost",
		"social_auth_sync.enabled",
		"social_auth_sync.interval",
		"social_auth_sync.batch_size",
		"sms.provider",
		"sms.access_key",
		"sms.access_secret",
		"sms.sign_name",
		"sms.template_code",
		"sms.code_ttl",
		"sms.send_interval",
		"sms.daily_limit",
		"smtp.host",
		"smtp.port",
		"smtp.username",
		"smtp.password",
		"smtp.from",
		"secrets.client_secret_encryption_key",
		"webauthn.rp_id",
		"webauthn.rp_origin",
		"webauthn.rp_display_name",
	}
	providers := []string{"google", "github", "gitlab", "gitee", "linuxdo", "microsoft", "apple", "facebook", "twitter", "linkedin", "wechat", "qq", "weibo", "dingtalk", "discord", "telegram"}
	providerFields := []string{"enabled", "client_id", "client_secret", "app_id", "app_secret", "app_key", "team_id", "key_id", "private_key", "tenant", "redirect_path"}
	for _, provider := range providers {
		for _, field := range providerFields {
			keys = append(keys, "oauth2."+provider+"."+field)
		}
	}
	for _, key := range keys {
		_ = v.BindEnv(key)
	}
}

func setDefaults(v *viper.Viper) {
	v.SetDefault("server.host", "0.0.0.0")
	v.SetDefault("server.port", 8080)
	v.SetDefault("server.read_timeout", "15s")
	v.SetDefault("server.write_timeout", "15s")
	v.SetDefault("server.shutdown_timeout", "30s")

	v.SetDefault("database.driver", "postgres")
	v.SetDefault("database.sslmode", "disable")
	v.SetDefault("database.max_conns", 20)
	v.SetDefault("database.min_conns", 2)

	v.SetDefault("redis.db", 0)
	v.SetDefault("redis.pool_size", 10)

	v.SetDefault("jwt.algorithm", "RS256")
	v.SetDefault("jwt.access_token_ttl", "1h")
	v.SetDefault("jwt.refresh_token_ttl", "720h")
	v.SetDefault("jwt.id_token_ttl", "1h")
	v.SetDefault("jwt.auth_code_ttl", "10m")

	v.SetDefault("log.level", "info")
	v.SetDefault("log.format", "json")
	v.SetDefault("social_auth_sync.enabled", true)
	v.SetDefault("social_auth_sync.interval", "1h")
	v.SetDefault("social_auth_sync.batch_size", 100)
	v.SetDefault("secrets.client_secret_encryption_key", "dev-client-secret-encryption-key")
	v.SetDefault("webauthn.rp_display_name", "OIDC Platform")
}
