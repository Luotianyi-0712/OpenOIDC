package sqlite

import (
	"context"
	"database/sql"

	"github.com/google/uuid"

	"github.com/anthropic/oidc-platform/internal/domain"
	"github.com/anthropic/oidc-platform/internal/port"
)

type RiskReportRepo struct {
	db *sql.DB
}

func NewRiskReportRepo(db *sql.DB) *RiskReportRepo {
	return &RiskReportRepo{db: db}
}

func (r *RiskReportRepo) Create(ctx context.Context, report *domain.RiskReport) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO risk_reports (id, client_id, reporter_id, target_id, reason, category, status, admin_note, resolved_at, resolved_by, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		report.ID.String(), report.ClientID.String(), report.ReporterID.String(), report.TargetID.String(),
		report.Reason, report.Category, report.Status, report.AdminNote,
		toNullTime(report.ResolvedAt), toNullString(uuidPtrToStringPtr(report.ResolvedBy)), report.CreatedAt,
	)
	return err
}

func (r *RiskReportRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.RiskReport, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, client_id, reporter_id, target_id, reason, category, status, admin_note, resolved_at, resolved_by, created_at
		 FROM risk_reports WHERE id = ?`, id.String())
	return scanReport(row)
}

func (r *RiskReportRepo) ListByTarget(ctx context.Context, targetID uuid.UUID) ([]*domain.RiskReport, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, client_id, reporter_id, target_id, reason, category, status, admin_note, resolved_at, resolved_by, created_at
		 FROM risk_reports WHERE target_id = ? ORDER BY created_at DESC`, targetID.String())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanReports(rows)
}

func (r *RiskReportRepo) ListByClient(ctx context.Context, clientID uuid.UUID, offset, limit int) ([]*domain.RiskReport, int64, error) {
	var total int64
	r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM risk_reports WHERE client_id = ?`, clientID.String()).Scan(&total)

	rows, err := r.db.QueryContext(ctx,
		`SELECT id, client_id, reporter_id, target_id, reason, category, status, admin_note, resolved_at, resolved_by, created_at
		 FROM risk_reports WHERE client_id = ? ORDER BY created_at DESC LIMIT ? OFFSET ?`,
		clientID.String(), limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	reports, err := scanReports(rows)
	return reports, total, err
}

func (r *RiskReportRepo) ListPending(ctx context.Context, offset, limit int) ([]*domain.RiskReport, int64, error) {
	var total int64
	r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM risk_reports WHERE status = 'pending'`).Scan(&total)

	rows, err := r.db.QueryContext(ctx,
		`SELECT id, client_id, reporter_id, target_id, reason, category, status, admin_note, resolved_at, resolved_by, created_at
		 FROM risk_reports WHERE status = 'pending' ORDER BY created_at ASC LIMIT ? OFFSET ?`,
		limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	reports, err := scanReports(rows)
	return reports, total, err
}

func (r *RiskReportRepo) Update(ctx context.Context, report *domain.RiskReport) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE risk_reports SET status = ?, admin_note = ?, resolved_at = ?, resolved_by = ? WHERE id = ?`,
		report.Status, report.AdminNote, toNullTime(report.ResolvedAt), toNullString(uuidPtrToStringPtr(report.ResolvedBy)), report.ID.String(),
	)
	return err
}

func (r *RiskReportRepo) CountConfirmedByTarget(ctx context.Context, targetID uuid.UUID) (int64, error) {
	var count int64
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM risk_reports WHERE target_id = ? AND status = 'confirmed'`, targetID.String()).Scan(&count)
	return count, err
}

func scanReport(row *sql.Row) (*domain.RiskReport, error) {
	var report domain.RiskReport
	var id, clientID, reporterID, targetID string
	var resolvedAt sql.NullTime
	var resolvedBy sql.NullString

	err := row.Scan(&id, &clientID, &reporterID, &targetID, &report.Reason, &report.Category,
		&report.Status, &report.AdminNote, &resolvedAt, &resolvedBy, &report.CreatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, port.ErrNotFound
		}
		return nil, err
	}
	report.ID = uuid.MustParse(id)
	report.ClientID = uuid.MustParse(clientID)
	report.ReporterID = uuid.MustParse(reporterID)
	report.TargetID = uuid.MustParse(targetID)
	report.ResolvedAt = fromNullTime(resolvedAt)
	report.ResolvedBy = stringPtrToUUIDPtr(fromNullString(resolvedBy))
	return &report, nil
}

func scanReports(rows *sql.Rows) ([]*domain.RiskReport, error) {
	var reports []*domain.RiskReport
	for rows.Next() {
		var report domain.RiskReport
		var id, clientID, reporterID, targetID string
		var resolvedAt sql.NullTime
		var resolvedBy sql.NullString

		err := rows.Scan(&id, &clientID, &reporterID, &targetID, &report.Reason, &report.Category,
			&report.Status, &report.AdminNote, &resolvedAt, &resolvedBy, &report.CreatedAt)
		if err != nil {
			return nil, err
		}
		report.ID = uuid.MustParse(id)
		report.ClientID = uuid.MustParse(clientID)
		report.ReporterID = uuid.MustParse(reporterID)
		report.TargetID = uuid.MustParse(targetID)
		report.ResolvedAt = fromNullTime(resolvedAt)
		report.ResolvedBy = stringPtrToUUIDPtr(fromNullString(resolvedBy))
		reports = append(reports, &report)
	}
	return reports, nil
}

// ---- Risk List ----

type RiskListRepo struct {
	db *sql.DB
}

func NewRiskListRepo(db *sql.DB) *RiskListRepo {
	return &RiskListRepo{db: db}
}

func (r *RiskListRepo) Add(ctx context.Context, entry *domain.RiskListEntry) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT OR IGNORE INTO risk_list (id, provider, provider_uid, user_id, reason, report_id, added_by, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		entry.ID.String(), entry.Provider, entry.ProviderUID,
		toNullString(uuidPtrToStringPtr(entry.UserID)), entry.Reason,
		toNullString(uuidPtrToStringPtr(entry.ReportID)),
		toNullString(uuidPtrToStringPtr(entry.AddedBy)),
		entry.CreatedAt,
	)
	return err
}

func (r *RiskListRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.RiskListEntry, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, provider, provider_uid, user_id, reason, report_id, added_by, created_at
			 FROM risk_list WHERE id = ?`, id.String())

	var entry domain.RiskListEntry
	var rawID string
	var userID, reportID, addedBy sql.NullString
	if err := row.Scan(&rawID, &entry.Provider, &entry.ProviderUID, &userID, &entry.Reason, &reportID, &addedBy, &entry.CreatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, port.ErrNotFound
		}
		return nil, err
	}
	entry.ID = uuid.MustParse(rawID)
	entry.UserID = stringPtrToUUIDPtr(fromNullString(userID))
	entry.ReportID = stringPtrToUUIDPtr(fromNullString(reportID))
	entry.AddedBy = stringPtrToUUIDPtr(fromNullString(addedBy))
	return &entry, nil
}

func (r *RiskListRepo) Check(ctx context.Context, provider, providerUID string) (*domain.RiskListEntry, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, provider, provider_uid, user_id, reason, report_id, added_by, created_at
		 FROM risk_list WHERE provider = ? AND provider_uid = ?`, provider, providerUID)

	var entry domain.RiskListEntry
	var id string
	var userID, reportID, addedBy sql.NullString

	err := row.Scan(&id, &entry.Provider, &entry.ProviderUID, &userID, &entry.Reason, &reportID, &addedBy, &entry.CreatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, port.ErrNotFound
		}
		return nil, err
	}
	entry.ID = uuid.MustParse(id)
	entry.UserID = stringPtrToUUIDPtr(fromNullString(userID))
	entry.ReportID = stringPtrToUUIDPtr(fromNullString(reportID))
	entry.AddedBy = stringPtrToUUIDPtr(fromNullString(addedBy))
	return &entry, nil
}

func (r *RiskListRepo) List(ctx context.Context, offset, limit int) ([]*domain.RiskListEntry, int64, error) {
	var total int64
	r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM risk_list`).Scan(&total)

	rows, err := r.db.QueryContext(ctx,
		`SELECT id, provider, provider_uid, user_id, reason, report_id, added_by, created_at
		 FROM risk_list ORDER BY created_at DESC LIMIT ? OFFSET ?`, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var entries []*domain.RiskListEntry
	for rows.Next() {
		var entry domain.RiskListEntry
		var id string
		var userID, reportID, addedBy sql.NullString
		if err := rows.Scan(&id, &entry.Provider, &entry.ProviderUID, &userID, &entry.Reason, &reportID, &addedBy, &entry.CreatedAt); err != nil {
			return nil, 0, err
		}
		entry.ID = uuid.MustParse(id)
		entry.UserID = stringPtrToUUIDPtr(fromNullString(userID))
		entry.ReportID = stringPtrToUUIDPtr(fromNullString(reportID))
		entry.AddedBy = stringPtrToUUIDPtr(fromNullString(addedBy))
		entries = append(entries, &entry)
	}
	return entries, total, nil
}

func (r *RiskListRepo) ListByUser(ctx context.Context, userID uuid.UUID) ([]*domain.RiskListEntry, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, provider, provider_uid, user_id, reason, report_id, added_by, created_at
			 FROM risk_list WHERE user_id = ? ORDER BY created_at DESC`, userID.String())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	entries := make([]*domain.RiskListEntry, 0)
	for rows.Next() {
		var entry domain.RiskListEntry
		var id string
		var rowUserID, reportID, addedBy sql.NullString
		if err := rows.Scan(&id, &entry.Provider, &entry.ProviderUID, &rowUserID, &entry.Reason, &reportID, &addedBy, &entry.CreatedAt); err != nil {
			return nil, err
		}
		entry.ID = uuid.MustParse(id)
		entry.UserID = stringPtrToUUIDPtr(fromNullString(rowUserID))
		entry.ReportID = stringPtrToUUIDPtr(fromNullString(reportID))
		entry.AddedBy = stringPtrToUUIDPtr(fromNullString(addedBy))
		entries = append(entries, &entry)
	}
	return entries, rows.Err()
}

func (r *RiskListRepo) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM risk_list WHERE id = ?`, id.String())
	return err
}

func (r *RiskListRepo) DeleteByReport(ctx context.Context, reportID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM risk_list WHERE report_id = ?`, reportID.String())
	return err
}

// helpers

func uuidPtrToStringPtr(u *uuid.UUID) *string {
	if u == nil {
		return nil
	}
	s := u.String()
	return &s
}

func stringPtrToUUIDPtr(s *string) *uuid.UUID {
	if s == nil || *s == "" {
		return nil
	}
	id, err := uuid.Parse(*s)
	if err != nil {
		return nil
	}
	return &id
}

// Ensure compile-time interface compliance.
var (
	_ port.RiskReportRepository = (*RiskReportRepo)(nil)
	_ port.RiskListRepository   = (*RiskListRepo)(nil)
)
