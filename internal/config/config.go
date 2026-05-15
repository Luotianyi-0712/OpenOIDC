package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Redis    RedisConfig
	JWT      JWTConfig
	Session  SessionConfig
	Admin    AdminConfig
	Log      LogConfig
	Security SecurityConfig
	OAuth2   OAuth2Config
	SMS      SMSConfig
	SMTP     SMTPConfig
}

type ServerConfig struct {
	Host            string
	Port            int
	Issuer          string
	BaseURL         string `mapstructure:"public_url"`
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
	DBName          string `mapstructure:"name"`
	SSLMode         string `mapstructure:"sslmode"`
	MaxOpenConns    int    `mapstructure:"max_conns"`
	MaxIdleConns    int    `mapstructure:"min_conns"`
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
	PasswordMinLength    int           `mapstructure:"password_min_length"`
	PasswordRequireUpper bool          `mapstructure:"password_require_upper"`
	PasswordRequireLower bool          `mapstructure:"password_require_lower"`
	PasswordRequireDigit bool          `mapstructure:"password_require_digit"`
	PasswordRequireSymbol bool         `mapstructure:"password_require_symbol"`
	MaxLoginAttempts     int           `mapstructure:"max_login_attempts"`
	LockoutDuration      time.Duration `mapstructure:"lockout_duration"`
	BcryptCost           int           `mapstructure:"bcrypt_cost"`
}

type OAuth2Config struct {
	Providers map[string]ProviderOAuth2Config `mapstructure:",remain"`
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

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("read config: %w", err)
		}
	}

	cfg := &Config{}
	if err := v.Unmarshal(cfg); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}

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
}
