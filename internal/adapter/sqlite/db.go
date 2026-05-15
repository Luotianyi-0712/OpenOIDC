package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "modernc.org/sqlite"
)

// NewDB opens a SQLite database with WAL mode and foreign keys enabled.
// If dsn is empty or ":memory:", an in-memory database is used.
func NewDB(ctx context.Context, dsn string) (*sql.DB, error) {
	if dsn == "" || dsn == ":memory:" {
		dsn = ":memory:"
	}

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	// Enable WAL mode and foreign keys
	pragmas := []string{
		"PRAGMA journal_mode=WAL",
		"PRAGMA foreign_keys=ON",
		"PRAGMA busy_timeout=5000",
	}
	for _, p := range pragmas {
		if _, err := db.ExecContext(ctx, p); err != nil {
			db.Close()
			return nil, fmt.Errorf("exec %q: %w", p, err)
		}
	}

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := db.PingContext(pingCtx); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping sqlite: %w", err)
	}

	return db, nil
}

// RunMigrations executes the full schema DDL directly against the database.
func RunMigrations(db *sql.DB) error {
	schema := `
	CREATE TABLE IF NOT EXISTS users (
		id TEXT PRIMARY KEY,
		email TEXT UNIQUE NOT NULL,
		email_verified BOOLEAN NOT NULL DEFAULT 0,
		password_hash TEXT NOT NULL DEFAULT '',
		display_name TEXT NOT NULL DEFAULT '',
		alias TEXT,
		avatar_url TEXT NOT NULL DEFAULT '',
		security_level INTEGER NOT NULL DEFAULT 0,
		role TEXT NOT NULL DEFAULT 'user',
		status TEXT NOT NULL DEFAULT 'active',
		last_login_at DATETIME,
		deleted_at DATETIME,
		created_at DATETIME NOT NULL,
		updated_at DATETIME NOT NULL
	);

	CREATE TABLE IF NOT EXISTS social_bindings (
		id TEXT PRIMARY KEY,
		user_id TEXT NOT NULL REFERENCES users(id),
		provider TEXT NOT NULL,
		provider_uid TEXT NOT NULL,
		provider_email TEXT,
		provider_name TEXT,
		access_token TEXT,
		refresh_token TEXT,
		token_expiry DATETIME,
		raw_profile TEXT,
		bound_at DATETIME NOT NULL,
		verified_at DATETIME,
		created_at DATETIME NOT NULL,
		updated_at DATETIME NOT NULL,
		UNIQUE(provider, provider_uid)
	);

	CREATE TABLE IF NOT EXISTS oidc_clients (
		id TEXT PRIMARY KEY,
		client_id TEXT UNIQUE NOT NULL,
		client_secret_hash TEXT NOT NULL DEFAULT '',
		client_secret_plain TEXT NOT NULL DEFAULT '',
		client_name TEXT NOT NULL DEFAULT '',
		description TEXT NOT NULL DEFAULT '',
		logo_url TEXT NOT NULL DEFAULT '',
		owner_user_id TEXT,
		redirect_uris TEXT NOT NULL DEFAULT '[]',
		grant_types TEXT NOT NULL DEFAULT '[]',
		response_types TEXT NOT NULL DEFAULT '[]',
		scopes TEXT NOT NULL DEFAULT '[]',
		token_endpoint_auth_method TEXT NOT NULL DEFAULT 'client_secret_basic',
		min_security_level INTEGER NOT NULL DEFAULT 0,
		require_email_verified BOOLEAN NOT NULL DEFAULT 0,
		protocol_type TEXT NOT NULL DEFAULT 'oidc',
		is_active BOOLEAN NOT NULL DEFAULT 1,
		is_confidential BOOLEAN NOT NULL DEFAULT 1,
		created_at DATETIME NOT NULL,
		updated_at DATETIME NOT NULL
	);

	CREATE TABLE IF NOT EXISTS client_access_rules (
		id TEXT PRIMARY KEY,
		client_id TEXT NOT NULL REFERENCES oidc_clients(id),
		rule_type TEXT NOT NULL,
		value TEXT NOT NULL,
		created_at DATETIME NOT NULL
	);

	CREATE TABLE IF NOT EXISTS security_level_rules (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		description TEXT NOT NULL DEFAULT '',
		level INTEGER NOT NULL,
		priority INTEGER NOT NULL DEFAULT 0,
		conditions TEXT NOT NULL DEFAULT '{}',
		is_active BOOLEAN NOT NULL DEFAULT 1,
		created_at DATETIME NOT NULL,
		updated_at DATETIME NOT NULL
	);

	CREATE TABLE IF NOT EXISTS user_sessions (
		id TEXT PRIMARY KEY,
		user_id TEXT NOT NULL REFERENCES users(id),
		session_token TEXT UNIQUE NOT NULL,
		ip_address TEXT,
		user_agent TEXT,
		expires_at DATETIME NOT NULL,
		revoked_at DATETIME,
		created_at DATETIME NOT NULL
	);

	CREATE TABLE IF NOT EXISTS provider_configs (
		id TEXT PRIMARY KEY,
		provider TEXT UNIQUE NOT NULL,
		display_name TEXT NOT NULL DEFAULT '',
		is_enabled BOOLEAN NOT NULL DEFAULT 0,
		client_id TEXT,
		client_secret TEXT,
		extra_config TEXT,
		sort_order INTEGER NOT NULL DEFAULT 0,
		created_at DATETIME NOT NULL,
		updated_at DATETIME NOT NULL
	);

	CREATE TABLE IF NOT EXISTS audit_log (
		id TEXT PRIMARY KEY,
		user_id TEXT,
		action TEXT NOT NULL,
		resource_type TEXT,
		resource_id TEXT,
		ip_address TEXT,
		user_agent TEXT,
		details TEXT,
		created_at DATETIME NOT NULL
	);

	CREATE TABLE IF NOT EXISTS security_level_changes (
		id TEXT PRIMARY KEY,
		user_id TEXT NOT NULL,
		old_level INTEGER NOT NULL,
		new_level INTEGER NOT NULL,
		reason TEXT NOT NULL DEFAULT '',
		matched_rule_id TEXT,
		created_at DATETIME NOT NULL
	);

	CREATE TABLE IF NOT EXISTS global_settings (
		key TEXT PRIMARY KEY,
		value TEXT NOT NULL DEFAULT '',
		description TEXT NOT NULL DEFAULT '',
		updated_at DATETIME NOT NULL
	);

	CREATE TABLE IF NOT EXISTS alias_restrictions (
		id TEXT PRIMARY KEY,
		pattern TEXT NOT NULL,
		restriction_type TEXT NOT NULL,
		reason TEXT NOT NULL DEFAULT '',
		created_at DATETIME NOT NULL
	);

	CREATE TABLE IF NOT EXISTS signing_keys (
		id TEXT PRIMARY KEY,
		key_id TEXT UNIQUE NOT NULL,
		algorithm TEXT NOT NULL,
		private_key BLOB NOT NULL,
		public_key BLOB NOT NULL,
		is_current BOOLEAN NOT NULL DEFAULT 0,
		created_at DATETIME NOT NULL,
		rotated_at DATETIME
	);

	CREATE TABLE IF NOT EXISTS oauth2_sessions (
		id TEXT PRIMARY KEY,
		request_id TEXT,
		session_type TEXT,
		client_id TEXT,
		signature TEXT UNIQUE,
		subject TEXT NOT NULL DEFAULT '',
		data BLOB,
		created_at DATETIME NOT NULL,
		expires_at DATETIME,
		active BOOLEAN NOT NULL DEFAULT 1
	);

	CREATE TABLE IF NOT EXISTS risk_reports (
		id TEXT PRIMARY KEY,
		client_id TEXT NOT NULL,
		reporter_id TEXT NOT NULL,
		target_id TEXT NOT NULL,
		reason TEXT NOT NULL DEFAULT '',
		category TEXT NOT NULL DEFAULT 'other',
		status TEXT NOT NULL DEFAULT 'pending',
		admin_note TEXT NOT NULL DEFAULT '',
		resolved_at DATETIME,
		resolved_by TEXT,
		created_at DATETIME NOT NULL
	);

	CREATE TABLE IF NOT EXISTS risk_list (
		id TEXT PRIMARY KEY,
		provider TEXT NOT NULL,
		provider_uid TEXT NOT NULL,
		user_id TEXT,
		reason TEXT NOT NULL DEFAULT '',
		report_id TEXT,
		added_by TEXT,
		created_at DATETIME NOT NULL,
		UNIQUE(provider, provider_uid)
	);
	`

	_, err := db.Exec(schema)
	if err != nil {
		return fmt.Errorf("run migrations: %w", err)
	}

	// Add columns for existing databases (idempotent).
	alterStmts := []string{
		`ALTER TABLE oidc_clients ADD COLUMN client_secret_plain TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE oidc_clients ADD COLUMN require_email_verified BOOLEAN NOT NULL DEFAULT 0`,
		`ALTER TABLE oidc_clients ADD COLUMN is_confidential BOOLEAN NOT NULL DEFAULT 1`,
		`ALTER TABLE oauth2_sessions ADD COLUMN subject TEXT NOT NULL DEFAULT ''`,
	}
	for _, stmt := range alterStmts {
		db.Exec(stmt) // ignore "duplicate column" errors
	}

	indexStmts := []string{
		`CREATE INDEX IF NOT EXISTS idx_oauth2_sessions_subject ON oauth2_sessions(subject)`,
		`CREATE INDEX IF NOT EXISTS idx_oauth2_sessions_type_active ON oauth2_sessions(session_type, active)`,
		`CREATE INDEX IF NOT EXISTS idx_oauth2_sessions_client_active ON oauth2_sessions(client_id, active)`,
	}
	for _, stmt := range indexStmts {
		db.Exec(stmt)
	}

	return nil
}

// toNullString converts a *string to sql.NullString.
func toNullString(s *string) sql.NullString {
	if s == nil {
		return sql.NullString{}
	}
	return sql.NullString{String: *s, Valid: true}
}

// fromNullString converts a sql.NullString to *string.
func fromNullString(n sql.NullString) *string {
	if !n.Valid {
		return nil
	}
	return &n.String
}

// toNullTime converts a *time.Time to sql.NullTime.
func toNullTime(t *time.Time) sql.NullTime {
	if t == nil {
		return sql.NullTime{}
	}
	return sql.NullTime{Time: *t, Valid: true}
}

// fromNullTime converts a sql.NullTime to *time.Time.
func fromNullTime(n sql.NullTime) *time.Time {
	if !n.Valid {
		return nil
	}
	return &n.Time
}
