// SPDX-License-Identifier: Apache-2.0

package store

import (
	"context"
	"fmt"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/google/uuid"
)

// PolicyStore defines read/write operations for policy artifacts.
type PolicyStore interface {
	InsertPolicy(ctx context.Context, p Policy) error
	ListPolicies(ctx context.Context) ([]Policy, error)
	GetPolicy(ctx context.Context, policyID string) (*Policy, error)
}

// MappingStore defines read/write operations for crosswalk mappings.
type MappingStore interface {
	InsertMapping(ctx context.Context, m MappingDocument) error
	ListMappings(ctx context.Context, policyID string) ([]MappingDocument, error)
}

// EvidenceStore defines read/write operations for evidence records.
type EvidenceStore interface {
	InsertEvidence(ctx context.Context, records []EvidenceRecord) (int, error)
	QueryEvidence(ctx context.Context, f EvidenceFilter) ([]EvidenceRecord, error)
}

// AuditLogStore defines read/write operations for audit log artifacts.
type AuditLogStore interface {
	InsertAuditLog(ctx context.Context, a AuditLog) error
	ListAuditLogs(ctx context.Context, policyID string, start, end time.Time) ([]AuditLog, error)
	GetAuditLog(ctx context.Context, auditID string) (*AuditLog, error)
}

// Store provides typed access to ClickHouse tables for policies,
// mapping documents, evidence, and audit logs. Implements all
// domain store interfaces.
type Store struct {
	conn driver.Conn
}

// Compile-time interface satisfaction checks.
var (
	_ PolicyStore   = (*Store)(nil)
	_ MappingStore  = (*Store)(nil)
	_ EvidenceStore = (*Store)(nil)
	_ AuditLogStore = (*Store)(nil)
)

// New wraps an existing ClickHouse connection.
func New(conn driver.Conn) *Store {
	return &Store{conn: conn}
}

// Policy represents a stored policy artifact.
type Policy struct {
	PolicyID     string    `json:"policy_id"`
	Title        string    `json:"title"`
	Version      string    `json:"version,omitempty"`
	OCIReference string    `json:"oci_reference"`
	Content      string    `json:"content"`
	ImportedAt   time.Time `json:"imported_at"`
	ImportedBy   string    `json:"imported_by,omitempty"`
}

// InsertPolicy stores a policy artifact.
func (s *Store) InsertPolicy(ctx context.Context, p Policy) error {
	if p.PolicyID == "" {
		p.PolicyID = uuid.New().String()
	}
	return s.conn.Exec(ctx,
		`INSERT INTO policies (policy_id, title, version, oci_reference, content, imported_by) VALUES (?, ?, ?, ?, ?, ?)`,
		p.PolicyID, p.Title, p.Version, p.OCIReference, p.Content, p.ImportedBy,
	)
}

// ListPolicies returns all stored policies ordered by import date.
func (s *Store) ListPolicies(ctx context.Context) ([]Policy, error) {
	rows, err := s.conn.Query(ctx,
		`SELECT policy_id, title, version, oci_reference, imported_at, imported_by FROM policies FINAL ORDER BY imported_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("list policies: %w", err)
	}
	defer rows.Close()

	var out []Policy
	for rows.Next() {
		var p Policy
		if err := rows.Scan(&p.PolicyID, &p.Title, &p.Version, &p.OCIReference, &p.ImportedAt, &p.ImportedBy); err != nil {
			return nil, fmt.Errorf("scan policy: %w", err)
		}
		out = append(out, p)
	}
	return out, nil
}

// GetPolicy returns a single policy with full content.
func (s *Store) GetPolicy(ctx context.Context, policyID string) (*Policy, error) {
	row := s.conn.QueryRow(ctx,
		`SELECT policy_id, title, version, oci_reference, content, imported_at, imported_by FROM policies FINAL WHERE policy_id = ?`, policyID)
	var p Policy
	if err := row.Scan(&p.PolicyID, &p.Title, &p.Version, &p.OCIReference, &p.Content, &p.ImportedAt, &p.ImportedBy); err != nil {
		return nil, fmt.Errorf("get policy: %w", err)
	}
	return &p, nil
}

// MappingDocument represents a crosswalk mapping artifact.
type MappingDocument struct {
	MappingID  string    `json:"mapping_id"`
	PolicyID   string    `json:"policy_id"`
	Framework  string    `json:"framework"`
	Content    string    `json:"content"`
	ImportedAt time.Time `json:"imported_at"`
}

// InsertMapping stores a mapping document.
func (s *Store) InsertMapping(ctx context.Context, m MappingDocument) error {
	if m.MappingID == "" {
		m.MappingID = uuid.New().String()
	}
	return s.conn.Exec(ctx,
		`INSERT INTO mapping_documents (mapping_id, policy_id, framework, content) VALUES (?, ?, ?, ?)`,
		m.MappingID, m.PolicyID, m.Framework, m.Content,
	)
}

// ListMappings returns mapping documents for a given policy.
func (s *Store) ListMappings(ctx context.Context, policyID string) ([]MappingDocument, error) {
	rows, err := s.conn.Query(ctx,
		`SELECT mapping_id, policy_id, framework, imported_at FROM mapping_documents FINAL WHERE policy_id = ? ORDER BY imported_at DESC`, policyID)
	if err != nil {
		return nil, fmt.Errorf("list mappings: %w", err)
	}
	defer rows.Close()

	var out []MappingDocument
	for rows.Next() {
		var m MappingDocument
		if err := rows.Scan(&m.MappingID, &m.PolicyID, &m.Framework, &m.ImportedAt); err != nil {
			return nil, fmt.Errorf("scan mapping: %w", err)
		}
		out = append(out, m)
	}
	return out, nil
}

// EvidenceRecord represents a single evidence row for the REST API.
type EvidenceRecord struct {
	EvidenceID    string    `json:"evidence_id"`
	PolicyID      string    `json:"policy_id"`
	TargetID      string    `json:"target_id"`
	TargetName    string    `json:"target_name,omitempty"`
	TargetType    string    `json:"target_type,omitempty"`
	TargetEnv     string    `json:"target_env,omitempty"`
	ControlID     string    `json:"control_id"`
	RuleID        string    `json:"rule_id"`
	EvalResult    string    `json:"eval_result"`
	EngineVersion string    `json:"engine_version,omitempty"`
	Frameworks    []string  `json:"frameworks,omitempty"`
	Requirements  []string  `json:"requirements,omitempty"`
	Owner         string    `json:"owner,omitempty"`
	CollectedAt   time.Time `json:"collected_at"`
}

// InsertEvidence batch-inserts evidence records.
func (s *Store) InsertEvidence(ctx context.Context, records []EvidenceRecord) (int, error) {
	batch, err := s.conn.PrepareBatch(ctx,
		`INSERT INTO evidence (evidence_id, policy_id, target_id, control_id, rule_id, eval_result, collected_at)`)
	if err != nil {
		return 0, fmt.Errorf("prepare batch: %w", err)
	}
	count := 0
	for _, r := range records {
		if r.EvidenceID == "" {
			r.EvidenceID = uuid.New().String()
		}
		if err := batch.Append(r.EvidenceID, r.PolicyID, r.TargetID, r.ControlID, r.RuleID, r.EvalResult, r.CollectedAt); err != nil {
			return count, fmt.Errorf("append row: %w", err)
		}
		count++
	}
	if err := batch.Send(); err != nil {
		return 0, fmt.Errorf("send batch: %w", err)
	}
	return count, nil
}

// EvidenceFilter holds query parameters for evidence queries.
type EvidenceFilter struct {
	PolicyID      string
	ControlID     string
	TargetName    string
	TargetType    string
	TargetEnv     string
	Framework     string
	EngineVersion string
	Owner         string
	Start         time.Time
	End           time.Time
	Limit         int
	Offset        int
}

// QueryEvidence returns evidence rows matching the filter.
func (s *Store) QueryEvidence(ctx context.Context, f EvidenceFilter) ([]EvidenceRecord, error) {
	query := `SELECT evidence_id, policy_id, target_id,
		coalesce(target_name, '') AS target_name,
		coalesce(target_type, '') AS target_type,
		coalesce(target_env, '') AS target_env,
		control_id, rule_id, eval_result,
		coalesce(engine_version, '') AS engine_version,
		frameworks, requirements,
		collected_at
		FROM evidence WHERE 1=1`
	var args []any

	if f.PolicyID != "" {
		query += ` AND policy_id = ?`
		args = append(args, f.PolicyID)
	}
	if f.ControlID != "" {
		query += ` AND control_id = ?`
		args = append(args, f.ControlID)
	}
	if f.TargetName != "" {
		query += ` AND target_name = ?`
		args = append(args, f.TargetName)
	}
	if f.TargetType != "" {
		query += ` AND target_type = ?`
		args = append(args, f.TargetType)
	}
	if f.TargetEnv != "" {
		query += ` AND target_env = ?`
		args = append(args, f.TargetEnv)
	}
	if f.Framework != "" {
		query += ` AND has(frameworks, ?)`
		args = append(args, f.Framework)
	}
	if f.EngineVersion != "" {
		query += ` AND engine_version = ?`
		args = append(args, f.EngineVersion)
	}
	if f.Owner != "" {
		query += ` AND target_id IN (SELECT DISTINCT target_id FROM evidence WHERE 1=1)`
		// Owner is stored on the Resource (target) in Gemara, not in the evidence row.
		// For now, filter is a placeholder until owner column is added to the schema.
	}
	if !f.Start.IsZero() {
		query += ` AND collected_at >= ?`
		args = append(args, f.Start)
	}
	if !f.End.IsZero() {
		query += ` AND collected_at <= ?`
		args = append(args, f.End)
	}
	query += ` ORDER BY collected_at DESC`
	if f.Limit > 0 {
		query += fmt.Sprintf(` LIMIT %d`, f.Limit)
	}
	if f.Offset > 0 {
		query += fmt.Sprintf(` OFFSET %d`, f.Offset)
	}

	rows, err := s.conn.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query evidence: %w", err)
	}
	defer rows.Close()

	var out []EvidenceRecord
	for rows.Next() {
		var r EvidenceRecord
		if err := rows.Scan(
			&r.EvidenceID, &r.PolicyID, &r.TargetID,
			&r.TargetName, &r.TargetType, &r.TargetEnv,
			&r.ControlID, &r.RuleID, &r.EvalResult,
			&r.EngineVersion, &r.Frameworks, &r.Requirements,
			&r.CollectedAt,
		); err != nil {
			return nil, fmt.Errorf("scan evidence: %w", err)
		}
		out = append(out, r)
	}
	return out, nil
}

// AuditLog represents a stored audit log artifact.
type AuditLog struct {
	AuditID    string    `json:"audit_id"`
	PolicyID   string    `json:"policy_id"`
	AuditStart time.Time `json:"audit_start"`
	AuditEnd   time.Time `json:"audit_end"`
	Framework  string    `json:"framework,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
	CreatedBy  string    `json:"created_by,omitempty"`
	Content    string    `json:"content"`
	Summary    string    `json:"summary"`
}

// InsertAuditLog stores an AuditLog artifact.
func (s *Store) InsertAuditLog(ctx context.Context, a AuditLog) error {
	if a.AuditID == "" {
		a.AuditID = uuid.New().String()
	}
	return s.conn.Exec(ctx,
		`INSERT INTO audit_logs (audit_id, policy_id, audit_start, audit_end, framework, created_by, content, summary) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		a.AuditID, a.PolicyID, a.AuditStart, a.AuditEnd, a.Framework, a.CreatedBy, a.Content, a.Summary,
	)
}

// ListAuditLogs returns audit logs for a given policy, optionally filtered by time range.
func (s *Store) ListAuditLogs(ctx context.Context, policyID string, start, end time.Time) ([]AuditLog, error) {
	query := `SELECT audit_id, policy_id, audit_start, audit_end, framework, created_at, created_by, summary FROM audit_logs FINAL WHERE policy_id = ?`
	args := []any{policyID}

	if !start.IsZero() {
		query += ` AND audit_start >= ?`
		args = append(args, start)
	}
	if !end.IsZero() {
		query += ` AND audit_end <= ?`
		args = append(args, end)
	}
	query += ` ORDER BY audit_start DESC`

	rows, err := s.conn.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list audit logs: %w", err)
	}
	defer rows.Close()

	var out []AuditLog
	for rows.Next() {
		var a AuditLog
		if err := rows.Scan(&a.AuditID, &a.PolicyID, &a.AuditStart, &a.AuditEnd, &a.Framework, &a.CreatedAt, &a.CreatedBy, &a.Summary); err != nil {
			return nil, fmt.Errorf("scan audit log: %w", err)
		}
		out = append(out, a)
	}
	return out, nil
}

// GetAuditLog returns a single audit log with full content.
func (s *Store) GetAuditLog(ctx context.Context, auditID string) (*AuditLog, error) {
	row := s.conn.QueryRow(ctx,
		`SELECT audit_id, policy_id, audit_start, audit_end, framework, created_at, created_by, content, summary FROM audit_logs FINAL WHERE audit_id = ?`, auditID)
	var a AuditLog
	if err := row.Scan(&a.AuditID, &a.PolicyID, &a.AuditStart, &a.AuditEnd, &a.Framework, &a.CreatedAt, &a.CreatedBy, &a.Content, &a.Summary); err != nil {
		return nil, fmt.Errorf("get audit log: %w", err)
	}
	return &a, nil
}
