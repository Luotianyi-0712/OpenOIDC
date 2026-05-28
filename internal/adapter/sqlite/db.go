package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
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
		uid INTEGER NOT NULL UNIQUE,
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

	CREATE TABLE IF NOT EXISTS user_uid_sequence (
		id INTEGER PRIMARY KEY CHECK (id = 1),
		next_uid INTEGER NOT NULL
	);

	CREATE TABLE IF NOT EXISTS social_bindings (
		id TEXT PRIMARY KEY,
		user_id TEXT NOT NULL REFERENCES users(id),
		provider TEXT NOT NULL,
		provider_uid TEXT NOT NULL,
		provider_email TEXT,
		provider_name TEXT,
		provider_avatar TEXT,
		status TEXT NOT NULL DEFAULT 'active',
		access_token TEXT,
		refresh_token TEXT,
		token_expiry DATETIME,
		token_type TEXT,
		token_scopes TEXT,
		raw_profile TEXT,
		bound_at DATETIME NOT NULL,
		verified_at DATETIME,
		unbound_at DATETIME,
		unbind_reason TEXT,
		last_auth_check_at DATETIME,
		last_auth_status TEXT NOT NULL DEFAULT 'unknown',
		last_auth_error TEXT,
		created_at DATETIME NOT NULL,
		updated_at DATETIME NOT NULL
	);

	CREATE TABLE IF NOT EXISTS oidc_clients (
		id TEXT PRIMARY KEY,
		client_id TEXT UNIQUE NOT NULL,
		client_secret_encrypted TEXT NOT NULL DEFAULT '',
		client_name TEXT NOT NULL DEFAULT '',
		description TEXT NOT NULL DEFAULT '',
		logo_url TEXT NOT NULL DEFAULT '',
		homepage_url TEXT NOT NULL DEFAULT '',
		owner_user_id TEXT,
		redirect_uris TEXT NOT NULL DEFAULT '[]',
		post_logout_redirect_uris TEXT NOT NULL DEFAULT '[]',
		grant_types TEXT NOT NULL DEFAULT '[]',
		response_types TEXT NOT NULL DEFAULT '[]',
		scopes TEXT NOT NULL DEFAULT '[]',
		token_endpoint_auth_method TEXT NOT NULL DEFAULT 'client_secret_basic',
		min_security_level INTEGER NOT NULL DEFAULT 0,
		require_email_verified BOOLEAN NOT NULL DEFAULT 1,
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
		scopes TEXT NOT NULL DEFAULT '[]',
		redirect_path TEXT NOT NULL DEFAULT '',
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
		signature TEXT,
		subject TEXT NOT NULL DEFAULT '',
		data BLOB,
		created_at DATETIME NOT NULL,
		expires_at DATETIME,
		active BOOLEAN NOT NULL DEFAULT 1,
		UNIQUE(session_type, signature)
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

	CREATE TABLE IF NOT EXISTS passkey_credentials (
		id TEXT PRIMARY KEY,
		user_id TEXT NOT NULL REFERENCES users(id),
		credential_id BLOB NOT NULL UNIQUE,
		public_key BLOB NOT NULL,
		attestation_type TEXT NOT NULL DEFAULT 'none',
		transport TEXT NOT NULL DEFAULT '[]',
		sign_count INTEGER NOT NULL DEFAULT 0,
		aaguid BLOB,
		name TEXT NOT NULL DEFAULT '',
		last_used_at DATETIME,
		created_at DATETIME NOT NULL
	);
	`

	_, err := db.Exec(schema)
	if err != nil {
		return fmt.Errorf("run migrations: %w", err)
	}

	// Add columns for existing databases (idempotent).
	alterStmts := []string{
		`ALTER TABLE users ADD COLUMN uid INTEGER`,
		`ALTER TABLE oidc_clients ADD COLUMN client_secret_encrypted TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE oidc_clients ADD COLUMN homepage_url TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE oidc_clients ADD COLUMN post_logout_redirect_uris TEXT NOT NULL DEFAULT '[]'`,
		`ALTER TABLE oidc_clients ADD COLUMN require_email_verified BOOLEAN NOT NULL DEFAULT 1`,
		`ALTER TABLE oidc_clients ADD COLUMN is_confidential BOOLEAN NOT NULL DEFAULT 1`,
		`ALTER TABLE oauth2_sessions ADD COLUMN subject TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE social_bindings ADD COLUMN provider_avatar TEXT`,
		`ALTER TABLE social_bindings ADD COLUMN status TEXT NOT NULL DEFAULT 'active'`,
		`ALTER TABLE social_bindings ADD COLUMN token_type TEXT`,
		`ALTER TABLE social_bindings ADD COLUMN token_scopes TEXT NOT NULL DEFAULT '[]'`,
		`ALTER TABLE social_bindings ADD COLUMN unbound_at DATETIME`,
		`ALTER TABLE social_bindings ADD COLUMN unbind_reason TEXT`,
		`ALTER TABLE social_bindings ADD COLUMN last_auth_check_at DATETIME`,
		`ALTER TABLE social_bindings ADD COLUMN last_auth_status TEXT NOT NULL DEFAULT 'unknown'`,
		`ALTER TABLE social_bindings ADD COLUMN last_auth_error TEXT`,
		`ALTER TABLE provider_configs ADD COLUMN scopes TEXT NOT NULL DEFAULT '[]'`,
		`ALTER TABLE provider_configs ADD COLUMN redirect_path TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE provider_configs ADD COLUMN sort_order INTEGER NOT NULL DEFAULT 0`,
		`ALTER TABLE users ADD COLUMN risk_report_email_enabled BOOLEAN NOT NULL DEFAULT 1`,
	}
	for _, stmt := range alterStmts {
		db.Exec(stmt) // ignore "duplicate column" errors
	}

	if err := migrateUserUIDs(db); err != nil {
		return err
	}
	if err := syncUserUIDSequence(db); err != nil {
		return err
	}

	if err := migrateSocialBindingsLifecycle(db); err != nil {
		return err
	}
	if err := migrateOAuth2SessionsSignatureScope(db); err != nil {
		return err
	}
	if _, err := db.Exec(`UPDATE users SET role = 'user' WHERE role IS NULL OR TRIM(role) = ''`); err != nil {
		return fmt.Errorf("repair user roles: %w", err)
	}

	// Insert default settings if not exists
	_, _ = db.Exec(`INSERT OR IGNORE INTO global_settings (key, value, description, updated_at) VALUES ('risk_report_email_notification_enabled', 'true', '控制是否允许用户接收举报处理结果的邮件通知。关闭后，用户个人资料页将不显示邮件通知开关。', datetime('now'))`)

	indexStmts := []string{
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_users_uid ON users(uid)`,
		`CREATE INDEX IF NOT EXISTS idx_oauth2_sessions_subject ON oauth2_sessions(subject)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_oauth2_sessions_type_signature ON oauth2_sessions(session_type, signature)`,
		`CREATE INDEX IF NOT EXISTS idx_oauth2_sessions_type_active ON oauth2_sessions(session_type, active)`,
		`CREATE INDEX IF NOT EXISTS idx_oauth2_sessions_client_active ON oauth2_sessions(client_id, active)`,
		`CREATE INDEX IF NOT EXISTS idx_social_bindings_user_id ON social_bindings(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_social_bindings_provider ON social_bindings(provider)`,
		`CREATE INDEX IF NOT EXISTS idx_social_bindings_auth_due ON social_bindings(last_auth_check_at) WHERE status = 'active'`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_social_bindings_active_provider_uid ON social_bindings(provider, provider_uid) WHERE status = 'active'`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_social_bindings_active_user_provider ON social_bindings(user_id, provider) WHERE status = 'active'`,
		`CREATE INDEX IF NOT EXISTS idx_passkey_credentials_user_id ON passkey_credentials(user_id)`,
	}
	for _, stmt := range indexStmts {
		db.Exec(stmt)
	}

	return nil
}

func migrateUserUIDs(db *sql.DB) error {
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("begin user uid migration: %w", err)
	}
	defer tx.Rollback()

	var nextUID int64
	if err := tx.QueryRow(`SELECT COALESCE(MAX(uid), 0) + 1 FROM users WHERE uid IS NOT NULL`).Scan(&nextUID); err != nil {
		return fmt.Errorf("get next user uid: %w", err)
	}

	rows, err := tx.Query(`SELECT id FROM users WHERE uid IS NULL ORDER BY created_at, id`)
	if err != nil {
		return fmt.Errorf("list users missing uid: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return fmt.Errorf("scan user missing uid: %w", err)
		}
		if _, err := tx.Exec(`UPDATE users SET uid = ? WHERE id = ?`, nextUID, id); err != nil {
			return fmt.Errorf("assign user uid: %w", err)
		}
		nextUID++
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate users missing uid: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit user uid migration: %w", err)
	}
	return nil
}

func syncUserUIDSequence(db *sql.DB) error {
	_, err := db.Exec(`
		INSERT INTO user_uid_sequence (id, next_uid)
		VALUES (1, (SELECT COALESCE(MAX(uid), 0) + 1 FROM users))
		ON CONFLICT(id) DO UPDATE SET
			next_uid = MAX(user_uid_sequence.next_uid, excluded.next_uid)
	`)
	if err != nil {
		return fmt.Errorf("sync user uid sequence: %w", err)
	}
	return nil
}

func migrateSocialBindingsLifecycle(db *sql.DB) error {
	var createSQL string
	if err := db.QueryRow(`SELECT sql FROM sqlite_master WHERE type = 'table' AND name = 'social_bindings'`).Scan(&createSQL); err != nil {
		return fmt.Errorf("inspect social_bindings schema: %w", err)
	}
	if createSQL == "" || !containsLegacySocialBindingUnique(createSQL) {
		return nil
	}

	migration := `
		PRAGMA foreign_keys=OFF;
		CREATE TABLE IF NOT EXISTS social_bindings_v2 (
			id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL REFERENCES users(id),
			provider TEXT NOT NULL,
			provider_uid TEXT NOT NULL,
			provider_email TEXT,
			provider_name TEXT,
			provider_avatar TEXT,
			status TEXT NOT NULL DEFAULT 'active',
			access_token TEXT,
			refresh_token TEXT,
			token_expiry DATETIME,
			token_type TEXT,
			token_scopes TEXT,
			raw_profile TEXT,
			bound_at DATETIME NOT NULL,
			verified_at DATETIME,
			unbound_at DATETIME,
			unbind_reason TEXT,
			last_auth_check_at DATETIME,
			last_auth_status TEXT NOT NULL DEFAULT 'unknown',
			last_auth_error TEXT,
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL
		);
		INSERT OR IGNORE INTO social_bindings_v2 (
			id, user_id, provider, provider_uid, provider_email, provider_name, provider_avatar,
			status, access_token, refresh_token, token_expiry, token_type, token_scopes, raw_profile,
			bound_at, verified_at, unbound_at, unbind_reason, last_auth_check_at, last_auth_status, last_auth_error,
			created_at, updated_at
		)
		SELECT
			id, user_id, provider, provider_uid, provider_email, provider_name, NULL,
			COALESCE(status, 'active'), access_token, refresh_token, token_expiry, token_type, token_scopes, raw_profile,
			bound_at, verified_at, unbound_at, unbind_reason, last_auth_check_at, COALESCE(last_auth_status, 'unknown'), last_auth_error,
			created_at, updated_at
		FROM social_bindings;
		DROP TABLE social_bindings;
		ALTER TABLE social_bindings_v2 RENAME TO social_bindings;
		PRAGMA foreign_keys=ON;
	`
	if _, err := db.Exec(migration); err != nil {
		return fmt.Errorf("migrate social_bindings lifecycle: %w", err)
	}
	return nil
}

func migrateOAuth2SessionsSignatureScope(db *sql.DB) error {
	var createSQL string
	if err := db.QueryRow(`SELECT sql FROM sqlite_master WHERE type = 'table' AND name = 'oauth2_sessions'`).Scan(&createSQL); err != nil {
		return fmt.Errorf("inspect oauth2_sessions schema: %w", err)
	}
	compact := compactSQL(createSQL)
	if !strings.Contains(compact, "signaturetextunique") && strings.Contains(compact, "unique(session_type,signature)") {
		return nil
	}

	migration := `
		PRAGMA foreign_keys=OFF;
		CREATE TABLE IF NOT EXISTS oauth2_sessions_v2 (
			id TEXT PRIMARY KEY,
			request_id TEXT,
			session_type TEXT,
			client_id TEXT,
			signature TEXT,
			subject TEXT NOT NULL DEFAULT '',
			data BLOB,
			created_at DATETIME NOT NULL,
			expires_at DATETIME,
			active BOOLEAN NOT NULL DEFAULT 1,
			UNIQUE(session_type, signature)
		);
		INSERT OR IGNORE INTO oauth2_sessions_v2 (
			id, request_id, session_type, client_id, signature, subject, data, created_at, expires_at, active
		)
		SELECT
			id, request_id, session_type, client_id, signature, COALESCE(subject, ''), data, created_at, expires_at, active
		FROM oauth2_sessions;
		DROP TABLE oauth2_sessions;
		ALTER TABLE oauth2_sessions_v2 RENAME TO oauth2_sessions;
		PRAGMA foreign_keys=ON;
	`
	if _, err := db.Exec(migration); err != nil {
		return fmt.Errorf("migrate oauth2_sessions signature scope: %w", err)
	}
	return nil
}

func containsLegacySocialBindingUnique(createSQL string) bool {
	return strings.Contains(compactSQL(createSQL), "unique(provider,provider_uid)")
}

func compactSQL(sql string) string {
	compact := strings.ToLower(strings.ReplaceAll(sql, " ", ""))
	compact = strings.ReplaceAll(compact, "\n", "")
	compact = strings.ReplaceAll(compact, "\t", "")
	return compact
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
