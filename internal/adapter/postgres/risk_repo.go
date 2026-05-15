package postgres

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/anthropic/oidc-platform/internal/domain"
	"github.com/anthropic/oidc-platform/internal/port"
)

// ---- RiskReportRepo ----

type RiskReportRepo struct {
	db *pgxpool.Pool
}

func NewRiskReportRepo(db *pgxpool.Pool) *RiskReportRepo {
	return &RiskReportRepo{db: db}
}

func (r *RiskReportRepo) Create(ctx context.Context, report *domain.RiskReport) error {
	_, err := r.db.Exec(ctx,
		`INSERT INTO risk_reports (id, client_id, reporter_id, target_id, reason, category, status, admin_note, resolved_at, resolved_by, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
		report.ID, report.ClientID, report.ReporterID, report.TargetID,
		report.Reason, report.Category, report.Status, report.AdminNote,
		report.ResolvedAt, report.ResolvedBy, report.CreatedAt,
	)
	return err
}

func (r *RiskReportRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.RiskReport, error) {
	row := r.db.QueryRow(ctx,
		`SELECT id, client_id, reporter_id, target_id, reason, category, status, admin_note, resolved_at, resolved_by, created_at
		 FROM risk_reports WHERE id = $1`, id)
	return scanPgReport(row)
}

func (r *RiskReportRepo) ListByTarget(ctx context.Context, targetID uuid.UUID) ([]*domain.RiskReport, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, client_id, reporter_id, target_id, reason, category, status, admin_note, resolved_at, resolved_by, created_at
		 FROM risk_reports WHERE target_id = $1 ORDER BY created_at DESC`, targetID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanPgReports(rows)
}

func (r *RiskReportRepo) ListByClient(ctx context.Context, clientID uuid.UUID, offset, limit int) ([]*domain.RiskReport, int64, error) {
	var total int64
	_ = r.db.QueryRow(ctx, `SELECT COUNT(*) FROM risk_reports WHERE client_id = $1`, clientID).Scan(&total)

	rows, err := r.db.Query(ctx,
		`SELECT id, client_id, reporter_id, target_id, reason, category, status, admin_note, resolved_at, resolved_by, created_at
		 FROM risk_reports WHERE client_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`,
		clientID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	reports, err := scanPgReports(rows)
	return reports, total, err
}

func (r *RiskReportRepo) ListPending(ctx context.Context, offset, limit int) ([]*domain.RiskReport, int64, error) {
	var total int64
	_ = r.db.QueryRow(ctx, `SELECT COUNT(*) FROM risk_reports WHERE status = 'pending'`).Scan(&total)

	rows, err := r.db.Query(ctx,
		`SELECT id, client_id, reporter_id, target_id, reason, category, status, admin_note, resolved_at, resolved_by, created_at
		 FROM risk_reports WHERE status = 'pending' ORDER BY created_at ASC LIMIT $1 OFFSET $2`,
		limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	reports, err := scanPgReports(rows)
	return reports, total, err
}

func (r *RiskReportRepo) Update(ctx context.Context, report *domain.RiskReport) error {
	_, err := r.db.Exec(ctx,
		`UPDATE risk_reports SET status = $1, admin_note = $2, resolved_at = $3, resolved_by = $4 WHERE id = $5`,
		report.Status, report.AdminNote, report.ResolvedAt, report.ResolvedBy, report.ID,
	)
	return err
}

func (r *RiskReportRepo) CountConfirmedByTarget(ctx context.Context, targetID uuid.UUID) (int64, error) {
	var count int64
	err := r.db.QueryRow(ctx,
		`SELECT COUNT(*) FROM risk_reports WHERE target_id = $1 AND status = 'confirmed'`, targetID).Scan(&count)
	return count, err
}

func scanPgReport(row pgx.Row) (*domain.RiskReport, error) {
	var report domain.RiskReport
	err := row.Scan(&report.ID, &report.ClientID, &report.ReporterID, &report.TargetID,
		&report.Reason, &report.Category, &report.Status, &report.AdminNote,
		&report.ResolvedAt, &report.ResolvedBy, &report.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, port.ErrNotFound
		}
		return nil, err
	}
	return &report, nil
}

func scanPgReports(rows pgx.Rows) ([]*domain.RiskReport, error) {
	var reports []*domain.RiskReport
	for rows.Next() {
		var report domain.RiskReport
		err := rows.Scan(&report.ID, &report.ClientID, &report.ReporterID, &report.TargetID,
			&report.Reason, &report.Category, &report.Status, &report.AdminNote,
			&report.ResolvedAt, &report.ResolvedBy, &report.CreatedAt)
		if err != nil {
			return nil, err
		}
		reports = append(reports, &report)
	}
	return reports, nil
}

// ---- RiskListRepo ----

type RiskListRepo struct {
	db *pgxpool.Pool
}

func NewRiskListRepo(db *pgxpool.Pool) *RiskListRepo {
	return &RiskListRepo{db: db}
}

func (r *RiskListRepo) Add(ctx context.Context, entry *domain.RiskListEntry) error {
	_, err := r.db.Exec(ctx,
		`INSERT INTO risk_list (id, provider, provider_uid, user_id, reason, report_id, added_by, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		 ON CONFLICT (provider, provider_uid) DO NOTHING`,
		entry.ID, entry.Provider, entry.ProviderUID,
		entry.UserID, entry.Reason, entry.ReportID, entry.AddedBy, entry.CreatedAt,
	)
	return err
}

func (r *RiskListRepo) Check(ctx context.Context, provider, providerUID string) (*domain.RiskListEntry, error) {
	row := r.db.QueryRow(ctx,
		`SELECT id, provider, provider_uid, user_id, reason, report_id, added_by, created_at
		 FROM risk_list WHERE provider = $1 AND provider_uid = $2`, provider, providerUID)

	var entry domain.RiskListEntry
	err := row.Scan(&entry.ID, &entry.Provider, &entry.ProviderUID,
		&entry.UserID, &entry.Reason, &entry.ReportID, &entry.AddedBy, &entry.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, port.ErrNotFound
		}
		return nil, err
	}
	return &entry, nil
}

func (r *RiskListRepo) List(ctx context.Context, offset, limit int) ([]*domain.RiskListEntry, int64, error) {
	var total int64
	_ = r.db.QueryRow(ctx, `SELECT COUNT(*) FROM risk_list`).Scan(&total)

	rows, err := r.db.Query(ctx,
		`SELECT id, provider, provider_uid, user_id, reason, report_id, added_by, created_at
		 FROM risk_list ORDER BY created_at DESC LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var entries []*domain.RiskListEntry
	for rows.Next() {
		var entry domain.RiskListEntry
		if err := rows.Scan(&entry.ID, &entry.Provider, &entry.ProviderUID,
			&entry.UserID, &entry.Reason, &entry.ReportID, &entry.AddedBy, &entry.CreatedAt); err != nil {
			return nil, 0, err
		}
		entries = append(entries, &entry)
	}
	return entries, total, nil
}

func (r *RiskListRepo) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `DELETE FROM risk_list WHERE id = $1`, id)
	return err
}

// Ensure compile-time interface compliance.
var _ port.RiskReportRepository = (*RiskReportRepo)(nil)
var _ port.RiskListRepository = (*RiskListRepo)(nil)
