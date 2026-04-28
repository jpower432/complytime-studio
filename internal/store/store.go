// SPDX-License-Identifier: Apache-2.0

package store

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/complytime/complytime-studio/internal/auth"
	"github.com/complytime/complytime-studio/internal/consts"
	"github.com/complytime/complytime-studio/internal/gemara"
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
	ListAllMappings(ctx context.Context) ([]MappingDocument, error)
	InsertMappingEntries(ctx context.Context, entries []gemara.MappingEntry) error
	CountMappingEntries(ctx context.Context, mappingID string) (int, error)
}

// ControlStore defines read/write operations for parsed control catalog entries.
type ControlStore interface {
	InsertControls(ctx context.Context, rows []gemara.ControlRow) error
	InsertAssessmentRequirements(ctx context.Context, rows []gemara.AssessmentRequirementRow) error
	InsertControlThreats(ctx context.Context, rows []gemara.ControlThreatRow) error
	CountControls(ctx context.Context, catalogID string) (int, error)
}

// ThreatStore defines read/write operations for parsed threat catalog entries.
type ThreatStore interface {
	InsertThreats(ctx context.Context, rows []gemara.ThreatRow) error
	CountThreats(ctx context.Context, catalogID string) (int, error)
	QueryThreats(ctx context.Context, catalogID, policyID string, limit int) ([]gemara.ThreatRow, error)
	QueryControlThreats(ctx context.Context, catalogID, controlID string, limit int) ([]gemara.ControlThreatRow, error)
}

// RiskStore defines read/write operations for parsed risk catalog entries.
type RiskStore interface {
	InsertRisks(ctx context.Context, rows []gemara.RiskRow) error
	InsertRiskThreats(ctx context.Context, rows []gemara.RiskThreatRow) error
	CountRisks(ctx context.Context, catalogID string) (int, error)
	GetPolicyRiskSeverity(ctx context.Context, policyID string) ([]RiskSeverityRow, error)
	QueryRisks(ctx context.Context, catalogID, policyID string, limit int) ([]gemara.RiskRow, error)
	QueryRiskThreats(ctx context.Context, catalogID, riskID string, limit int) ([]gemara.RiskThreatRow, error)
}

// CatalogStore defines read/write operations for raw catalog artifacts.
type CatalogStore interface {
	InsertCatalog(ctx context.Context, c Catalog) error
	ListCatalogs(ctx context.Context) ([]Catalog, error)
	GetCatalog(ctx context.Context, catalogID string) (*Catalog, error)
}

// EvidenceStore defines read/write operations for evidence records.
type EvidenceStore interface {
	InsertEvidence(ctx context.Context, records []EvidenceRecord) (int, error)
	QueryEvidence(ctx context.Context, f EvidenceFilter) ([]EvidenceRecord, error)
}

// AuditLogStore defines read/write operations for audit log artifacts.
type AuditLogStore interface {
	InsertAuditLog(ctx context.Context, a AuditLog) error
	ListAuditLogs(ctx context.Context, policyID string, start, end time.Time, limit int) ([]AuditLog, error)
	GetAuditLog(ctx context.Context, auditID string) (*AuditLog, error)
}

// EvidenceAssessmentStore defines write operations for agent-produced classifications.
type EvidenceAssessmentStore interface {
	InsertEvidenceAssessments(ctx context.Context, assessments []EvidenceAssessment) error
}

// CertificationStore defines read/write operations for evidence certifications.
type CertificationStore interface {
	InsertCertifications(ctx context.Context, rows []CertificationRow) error
	UpdateEvidenceCertified(ctx context.Context, evidenceID string, certified bool) error
	QueryCertifications(ctx context.Context, evidenceID string) ([]CertificationRow, error)
	QueryRecentEvidence(
		ctx context.Context, policyID string, since time.Time,
	) ([]EvidenceRowLite, error)
}

// PostureStore defines read operations for compliance posture aggregates.
type PostureStore interface {
	ListPosture(ctx context.Context, start, end time.Time) ([]PostureRow, error)
	QueryPolicyPosture(ctx context.Context, policyID string) (total, passed, failed uint64, err error)
}

// RequirementStore defines read operations for the requirement matrix.
type RequirementStore interface {
	ListRequirementMatrix(ctx context.Context, f RequirementFilter) ([]RequirementRow, error)
	ListRequirementEvidence(ctx context.Context, requirementID string, f RequirementFilter) ([]RequirementEvidenceRow, error)
}

// DraftAuditLogStore defines operations for agent-produced draft audit logs
// that require human review before promotion to the official audit_logs table.
type DraftAuditLogStore interface {
	InsertDraftAuditLog(ctx context.Context, d DraftAuditLog) error
	ListDraftAuditLogs(ctx context.Context, status string, limit int) ([]DraftAuditLog, error)
	GetDraftAuditLog(ctx context.Context, draftID string) (*DraftAuditLog, error)
	UpdateDraftEdits(ctx context.Context, draftID string, reviewerEdits string) error
	PromoteDraftAuditLog(ctx context.Context, draftID string, reviewedBy string) error
}

// NotificationStore defines operations for inbox notifications.
type NotificationStore interface {
	InsertNotification(ctx context.Context, n Notification) error
	ListNotifications(ctx context.Context, limit int) ([]Notification, error)
	MarkRead(ctx context.Context, notificationID string) error
	UnreadCount(ctx context.Context) (int, error)
}

// Store provides typed access to ClickHouse tables for policies,
// mapping documents, evidence, and audit logs. Implements all
// domain store interfaces.
type Store struct {
	conn driver.Conn
}

// Compile-time interface satisfaction checks.
var (
	_ PolicyStore             = (*Store)(nil)
	_ MappingStore            = (*Store)(nil)
	_ EvidenceStore           = (*Store)(nil)
	_ AuditLogStore           = (*Store)(nil)
	_ ControlStore            = (*Store)(nil)
	_ ThreatStore             = (*Store)(nil)
	_ RiskStore               = (*Store)(nil)
	_ CatalogStore            = (*Store)(nil)
	_ EvidenceAssessmentStore = (*Store)(nil)
	_ DraftAuditLogStore      = (*Store)(nil)
	_ RequirementStore        = (*Store)(nil)
	_ PostureStore            = (*Store)(nil)
	_ NotificationStore       = (*Store)(nil)
	_ CertificationStore      = (*Store)(nil)
	_ auth.UserStore          = (*Store)(nil)
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
	defer func() { _ = rows.Close() }()

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
		`SELECT mapping_id, policy_id, framework, content, imported_at FROM mapping_documents FINAL WHERE policy_id = ? ORDER BY imported_at DESC`, policyID)
	if err != nil {
		return nil, fmt.Errorf("list mappings: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var out []MappingDocument
	for rows.Next() {
		var m MappingDocument
		if err := rows.Scan(&m.MappingID, &m.PolicyID, &m.Framework, &m.Content, &m.ImportedAt); err != nil {
			return nil, fmt.Errorf("scan mapping: %w", err)
		}
		out = append(out, m)
	}
	return out, nil
}

// ListAllMappings returns all mapping documents across all policies.
func (s *Store) ListAllMappings(ctx context.Context) ([]MappingDocument, error) {
	rows, err := s.conn.Query(ctx,
		`SELECT mapping_id, policy_id, framework, content, imported_at FROM mapping_documents FINAL ORDER BY imported_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("list all mappings: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var out []MappingDocument
	for rows.Next() {
		var m MappingDocument
		if err := rows.Scan(&m.MappingID, &m.PolicyID, &m.Framework, &m.Content, &m.ImportedAt); err != nil {
			return nil, fmt.Errorf("scan mapping: %w", err)
		}
		out = append(out, m)
	}
	return out, nil
}

// InsertMappingEntries batch-inserts structured mapping entries.
func (s *Store) InsertMappingEntries(ctx context.Context, entries []gemara.MappingEntry) error {
	if len(entries) == 0 {
		return nil
	}
	batch, err := s.conn.PrepareBatch(ctx,
		`INSERT INTO mapping_entries (mapping_id, policy_id, control_id, requirement_id, framework, reference, strength, confidence)`)
	if err != nil {
		return fmt.Errorf("prepare mapping entries batch: %w", err)
	}
	for _, e := range entries {
		if err := batch.Append(e.MappingID, e.PolicyID, e.ControlID, e.RequirementID, e.Framework, e.Reference, e.Strength, e.Confidence); err != nil {
			return fmt.Errorf("append mapping entry: %w", err)
		}
	}
	return batch.Send()
}

// CountMappingEntries returns the number of structured entries for a given mapping document.
func (s *Store) CountMappingEntries(ctx context.Context, mappingID string) (int, error) {
	row := s.conn.QueryRow(ctx,
		`SELECT count() FROM mapping_entries WHERE mapping_id = ?`, mappingID)
	var count uint64
	if err := row.Scan(&count); err != nil {
		return 0, fmt.Errorf("count mapping entries: %w", err)
	}
	return int(count), nil
}

func (s *Store) InsertControls(ctx context.Context, rows []gemara.ControlRow) error {
	if len(rows) == 0 {
		return nil
	}
	batch, err := s.conn.PrepareBatch(ctx,
		`INSERT INTO controls (catalog_id, control_id, title, objective, group_id, state, policy_id)`)
	if err != nil {
		return fmt.Errorf("prepare controls batch: %w", err)
	}
	for _, r := range rows {
		if err := batch.Append(
			r.CatalogID, r.ControlID, r.Title, r.Objective, r.GroupID, r.State, r.PolicyID,
		); err != nil {
			return fmt.Errorf("append control: %w", err)
		}
	}
	return batch.Send()
}

func (s *Store) InsertAssessmentRequirements(ctx context.Context, rows []gemara.AssessmentRequirementRow) error {
	if len(rows) == 0 {
		return nil
	}
	batch, err := s.conn.PrepareBatch(ctx,
		`INSERT INTO assessment_requirements (catalog_id, control_id, requirement_id, text, applicability, recommendation, state)`)
	if err != nil {
		return fmt.Errorf("prepare assessment requirements batch: %w", err)
	}
	for _, r := range rows {
		if err := batch.Append(
			r.CatalogID, r.ControlID, r.RequirementID, r.Text, r.Applicability, r.Recommendation, r.State,
		); err != nil {
			return fmt.Errorf("append assessment requirement: %w", err)
		}
	}
	return batch.Send()
}

func (s *Store) InsertControlThreats(ctx context.Context, rows []gemara.ControlThreatRow) error {
	if len(rows) == 0 {
		return nil
	}
	batch, err := s.conn.PrepareBatch(ctx,
		`INSERT INTO control_threats (catalog_id, control_id, threat_reference_id, threat_entry_id)`)
	if err != nil {
		return fmt.Errorf("prepare control threats batch: %w", err)
	}
	for _, r := range rows {
		if err := batch.Append(
			r.CatalogID, r.ControlID, r.ThreatReferenceID, r.ThreatEntryID,
		); err != nil {
			return fmt.Errorf("append control threat: %w", err)
		}
	}
	return batch.Send()
}

func (s *Store) CountControls(ctx context.Context, catalogID string) (int, error) {
	row := s.conn.QueryRow(ctx,
		`SELECT count() FROM controls WHERE catalog_id = ?`, catalogID)
	var count uint64
	if err := row.Scan(&count); err != nil {
		return 0, fmt.Errorf("count controls: %w", err)
	}
	return int(count), nil
}

func (s *Store) InsertThreats(ctx context.Context, rows []gemara.ThreatRow) error {
	if len(rows) == 0 {
		return nil
	}
	batch, err := s.conn.PrepareBatch(ctx,
		`INSERT INTO threats (catalog_id, threat_id, title, description, group_id, policy_id)`)
	if err != nil {
		return fmt.Errorf("prepare threats batch: %w", err)
	}
	for _, r := range rows {
		if err := batch.Append(
			r.CatalogID, r.ThreatID, r.Title, r.Description, r.GroupID, r.PolicyID,
		); err != nil {
			return fmt.Errorf("append threat: %w", err)
		}
	}
	return batch.Send()
}

func (s *Store) CountThreats(ctx context.Context, catalogID string) (int, error) {
	row := s.conn.QueryRow(ctx,
		`SELECT count() FROM threats WHERE catalog_id = ?`, catalogID)
	var count uint64
	if err := row.Scan(&count); err != nil {
		return 0, fmt.Errorf("count threats: %w", err)
	}
	return int(count), nil
}

func (s *Store) QueryThreats(ctx context.Context, catalogID, policyID string, limit int) ([]gemara.ThreatRow, error) {
	where, args := buildCatalogPolicyFilter(catalogID, policyID)
	limit = consts.ClampLimit(limit)
	query := fmt.Sprintf(`SELECT catalog_id, threat_id, title, description, group_id, policy_id FROM threats`+where+` ORDER BY catalog_id, threat_id LIMIT %d`, limit)

	rows, err := s.conn.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query threats: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var out []gemara.ThreatRow
	for rows.Next() {
		var r gemara.ThreatRow
		if err := rows.Scan(&r.CatalogID, &r.ThreatID, &r.Title, &r.Description, &r.GroupID, &r.PolicyID); err != nil {
			return nil, fmt.Errorf("scan threat: %w", err)
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (s *Store) QueryControlThreats(ctx context.Context, catalogID, controlID string, limit int) ([]gemara.ControlThreatRow, error) {
	var clauses []string
	var args []any
	if catalogID != "" {
		clauses = append(clauses, "catalog_id = ?")
		args = append(args, catalogID)
	}
	if controlID != "" {
		clauses = append(clauses, "control_id = ?")
		args = append(args, controlID)
	}
	where := ""
	if len(clauses) > 0 {
		where = " WHERE " + strings.Join(clauses, " AND ")
	}
	limit = consts.ClampLimit(limit)
	query := fmt.Sprintf(`SELECT catalog_id, control_id, threat_reference_id, threat_entry_id FROM control_threats`+where+` ORDER BY catalog_id, control_id, threat_reference_id LIMIT %d`, limit)

	rows, err := s.conn.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query control threats: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var out []gemara.ControlThreatRow
	for rows.Next() {
		var r gemara.ControlThreatRow
		if err := rows.Scan(&r.CatalogID, &r.ControlID, &r.ThreatReferenceID, &r.ThreatEntryID); err != nil {
			return nil, fmt.Errorf("scan control threat: %w", err)
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (s *Store) InsertRisks(ctx context.Context, rows []gemara.RiskRow) error {
	if len(rows) == 0 {
		return nil
	}
	batch, err := s.conn.PrepareBatch(ctx,
		`INSERT INTO risks (catalog_id, risk_id, title, description, severity, group_id, impact, policy_id)`)
	if err != nil {
		return fmt.Errorf("prepare risks batch: %w", err)
	}
	for _, r := range rows {
		if err := batch.Append(
			r.CatalogID, r.RiskID, r.Title, r.Description, r.Severity, r.GroupID, r.Impact, r.PolicyID,
		); err != nil {
			return fmt.Errorf("append risk: %w", err)
		}
	}
	return batch.Send()
}

func (s *Store) InsertRiskThreats(ctx context.Context, rows []gemara.RiskThreatRow) error {
	if len(rows) == 0 {
		return nil
	}
	batch, err := s.conn.PrepareBatch(ctx,
		`INSERT INTO risk_threats (catalog_id, risk_id, threat_reference_id, threat_entry_id)`)
	if err != nil {
		return fmt.Errorf("prepare risk threats batch: %w", err)
	}
	for _, r := range rows {
		if err := batch.Append(
			r.CatalogID, r.RiskID, r.ThreatReferenceID, r.ThreatEntryID,
		); err != nil {
			return fmt.Errorf("append risk threat: %w", err)
		}
	}
	return batch.Send()
}

func (s *Store) CountRisks(ctx context.Context, catalogID string) (int, error) {
	row := s.conn.QueryRow(ctx,
		`SELECT count() FROM risks WHERE catalog_id = ?`, catalogID)
	var count uint64
	if err := row.Scan(&count); err != nil {
		return 0, fmt.Errorf("count risks: %w", err)
	}
	return int(count), nil
}

func (s *Store) QueryRisks(ctx context.Context, catalogID, policyID string, limit int) ([]gemara.RiskRow, error) {
	where, args := buildCatalogPolicyFilter(catalogID, policyID)
	limit = consts.ClampLimit(limit)
	query := fmt.Sprintf(`SELECT catalog_id, risk_id, title, description, severity, group_id, impact, policy_id FROM risks`+where+` ORDER BY catalog_id, risk_id LIMIT %d`, limit)

	rows, err := s.conn.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query risks: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var out []gemara.RiskRow
	for rows.Next() {
		var r gemara.RiskRow
		if err := rows.Scan(&r.CatalogID, &r.RiskID, &r.Title, &r.Description, &r.Severity, &r.GroupID, &r.Impact, &r.PolicyID); err != nil {
			return nil, fmt.Errorf("scan risk: %w", err)
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (s *Store) QueryRiskThreats(ctx context.Context, catalogID, riskID string, limit int) ([]gemara.RiskThreatRow, error) {
	var clauses []string
	var args []any
	if catalogID != "" {
		clauses = append(clauses, "catalog_id = ?")
		args = append(args, catalogID)
	}
	if riskID != "" {
		clauses = append(clauses, "risk_id = ?")
		args = append(args, riskID)
	}
	where := ""
	if len(clauses) > 0 {
		where = " WHERE " + strings.Join(clauses, " AND ")
	}
	limit = consts.ClampLimit(limit)
	query := fmt.Sprintf(`SELECT catalog_id, risk_id, threat_reference_id, threat_entry_id FROM risk_threats`+where+` ORDER BY catalog_id, risk_id, threat_reference_id LIMIT %d`, limit)

	rows, err := s.conn.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query risk threats: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var out []gemara.RiskThreatRow
	for rows.Next() {
		var r gemara.RiskThreatRow
		if err := rows.Scan(&r.CatalogID, &r.RiskID, &r.ThreatReferenceID, &r.ThreatEntryID); err != nil {
			return nil, fmt.Errorf("scan risk threat: %w", err)
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// buildCatalogPolicyFilter builds a WHERE clause for optional catalog_id and policy_id filters.
func buildCatalogPolicyFilter(catalogID, policyID string) (string, []any) {
	var clauses []string
	var args []any
	if catalogID != "" {
		clauses = append(clauses, "catalog_id = ?")
		args = append(args, catalogID)
	}
	if policyID != "" {
		clauses = append(clauses, "policy_id = ?")
		args = append(args, policyID)
	}
	if len(clauses) == 0 {
		return "", nil
	}
	return " WHERE " + strings.Join(clauses, " AND "), args
}

// RiskSeverityRow maps a control to its highest-severity risk for a given policy.
type RiskSeverityRow struct {
	ControlID   string `json:"control_id"`
	MaxSeverity string `json:"max_severity"`
	RiskCount   uint64 `json:"risk_count"`
}

// GetPolicyRiskSeverity returns per-control max risk severity by joining
// risks -> risk_threats -> control_threats -> controls filtered by policy.
func (s *Store) GetPolicyRiskSeverity(ctx context.Context, policyID string) ([]RiskSeverityRow, error) {
	query := `
		SELECT
			ct.control_id,
			maxIf(r.severity, r.severity != '') AS max_severity,
			count(DISTINCT r.risk_id) AS risk_count
		FROM control_threats ct
		INNER JOIN risk_threats rt
			ON rt.threat_reference_id = ct.threat_reference_id
			AND rt.threat_entry_id = ct.threat_entry_id
		INNER JOIN risks r
			ON r.risk_id = rt.risk_id
			AND r.catalog_id = rt.catalog_id
		WHERE r.policy_id = ? OR ct.catalog_id IN (
			SELECT catalog_id FROM controls WHERE policy_id = ?
		)
		GROUP BY ct.control_id
		HAVING max_severity != ''
		ORDER BY ct.control_id`

	rows, err := s.conn.Query(ctx, query, policyID, policyID)
	if err != nil {
		return nil, fmt.Errorf("risk severity query: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var out []RiskSeverityRow
	for rows.Next() {
		var r RiskSeverityRow
		if err := rows.Scan(&r.ControlID, &r.MaxSeverity, &r.RiskCount); err != nil {
			return nil, fmt.Errorf("scan risk severity: %w", err)
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// EvidenceRecord represents a single evidence row for the REST API.
// Fields align with evidence-semconv-alignment.md; new fields use omitempty
// for backward compatibility with minimal payloads.
type EvidenceRecord struct {
	EvidenceID string `json:"evidence_id"`
	PolicyID   string `json:"policy_id"`
	TargetID   string `json:"target_id"`
	TargetName string `json:"target_name,omitempty"`
	TargetType string `json:"target_type,omitempty"`
	TargetEnv  string `json:"target_env,omitempty"`

	EngineName    string `json:"engine_name,omitempty"`
	EngineVersion string `json:"engine_version,omitempty"`
	RuleID        string `json:"rule_id"`
	RuleName      string `json:"rule_name,omitempty"`
	RuleURI       string `json:"rule_uri,omitempty"`

	EvalResult  string `json:"eval_result"`
	EvalMessage string `json:"eval_message,omitempty"`

	ControlID            string   `json:"control_id"`
	ControlCatalogID     string   `json:"control_catalog_id,omitempty"`
	ControlCategory      string   `json:"control_category,omitempty"`
	ControlApplicability []string `json:"control_applicability,omitempty"`
	RequirementID        string   `json:"requirement_id,omitempty"`
	PlanID               string   `json:"plan_id,omitempty"`
	Confidence           string   `json:"confidence,omitempty"`
	StepsExecuted        int      `json:"steps_executed,omitempty"`
	ComplianceStatus     string   `json:"compliance_status,omitempty"`
	RiskLevel            string   `json:"risk_level,omitempty"`
	Frameworks           []string `json:"frameworks,omitempty"`
	Requirements         []string `json:"requirements,omitempty"`

	RemediationAction string `json:"remediation_action,omitempty"`
	RemediationStatus string `json:"remediation_status,omitempty"`
	RemediationDesc   string `json:"remediation_desc,omitempty"`
	ExceptionID       string `json:"exception_id,omitempty"`
	ExceptionActive   *bool  `json:"exception_active,omitempty"`

	EnrichmentStatus string `json:"enrichment_status,omitempty"`

	AttestationRef string `json:"attestation_ref,omitempty"`
	SourceRegistry string `json:"source_registry,omitempty"`
	BlobRef        string `json:"blob_ref,omitempty"`

	Certified bool `json:"certified"`

	Owner       string    `json:"owner,omitempty"`
	CollectedAt time.Time `json:"collected_at"`
}

// nullStr returns nil for empty strings, pointer otherwise.
func nullStr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// nullUint16 returns nil for zero, pointer otherwise.
func nullUint16(v int) *uint16 {
	if v <= 0 {
		return nil
	}
	u := uint16(v)
	return &u
}

// warnEvalMessageIfLarge logs when eval_message may be raw or embedded data
// rather than a short summary (see consts.EvalMessageWarnBytes).
func warnEvalMessageIfLarge(r EvidenceRecord) {
	if len(r.EvalMessage) <= consts.EvalMessageWarnBytes {
		return
	}
	slog.Warn("evidence eval_message exceeds recommended summary size",
		"bytes", len(r.EvalMessage),
		"warn_threshold_bytes", consts.EvalMessageWarnBytes,
		"policy_id", r.PolicyID,
		"evidence_id", r.EvidenceID,
	)
}

// normalizeEvidence applies defaults to an EvidenceRecord before insert.
func normalizeEvidence(r *EvidenceRecord) {
	if r.EvidenceID == "" {
		r.EvidenceID = uuid.New().String()
	}
	if r.EnrichmentStatus == "" {
		r.EnrichmentStatus = "Success"
	}
	if r.ComplianceStatus == "" {
		r.ComplianceStatus = "Unknown"
	}
	if r.EvalResult == "" {
		r.EvalResult = "Unknown"
	}
}

// InsertEvidence batch-inserts evidence records with full semconv column coverage.
func (s *Store) InsertEvidence(ctx context.Context, records []EvidenceRecord) (int, error) {
	batch, err := s.conn.PrepareBatch(ctx, `INSERT INTO evidence (
		evidence_id, target_id, target_name, target_type, target_env,
		engine_name, engine_version, rule_id, rule_name, rule_uri,
		eval_result, eval_message,
		policy_id, control_id, control_catalog_id, control_category,
		control_applicability, requirement_id, plan_id,
		confidence, steps_executed, compliance_status,
		risk_level, requirements,
		remediation_action, remediation_status, remediation_desc,
		exception_id, exception_active,
		enrichment_status,
		attestation_ref, source_registry, blob_ref,
		collected_at
	)`)
	if err != nil {
		return 0, fmt.Errorf("prepare batch: %w", err)
	}
	count := 0
	for _, r := range records {
		normalizeEvidence(&r)
		warnEvalMessageIfLarge(r)
		if err := batch.Append(
			r.EvidenceID,
			r.TargetID, nullStr(r.TargetName), nullStr(r.TargetType), nullStr(r.TargetEnv),
			nullStr(r.EngineName), nullStr(r.EngineVersion), r.RuleID, nullStr(r.RuleName), nullStr(r.RuleURI),
			r.EvalResult, nullStr(r.EvalMessage),
			r.PolicyID, r.ControlID, nullStr(r.ControlCatalogID), nullStr(r.ControlCategory),
			r.ControlApplicability, r.RequirementID, nullStr(r.PlanID),
			nullStr(r.Confidence), nullUint16(r.StepsExecuted), r.ComplianceStatus,
			nullStr(r.RiskLevel), r.Requirements,
			nullStr(r.RemediationAction), nullStr(r.RemediationStatus), nullStr(r.RemediationDesc),
			nullStr(r.ExceptionID), r.ExceptionActive,
			r.EnrichmentStatus,
			nullStr(r.AttestationRef), nullStr(r.SourceRegistry), nullStr(r.BlobRef),
			r.CollectedAt,
		); err != nil {
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
		coalesce(engine_name, '') AS engine_name,
		coalesce(engine_version, '') AS engine_version,
		rule_id,
		coalesce(rule_name, '') AS rule_name,
		eval_result,
		coalesce(eval_message, '') AS eval_message,
		control_id,
		coalesce(control_catalog_id, '') AS control_catalog_id,
		coalesce(control_category, '') AS control_category,
		requirement_id,
		coalesce(plan_id, '') AS plan_id,
		coalesce(confidence, '') AS confidence,
		compliance_status,
		coalesce(risk_level, '') AS risk_level,
		requirements,
		enrichment_status,
		coalesce(attestation_ref, '') AS attestation_ref,
		coalesce(source_registry, '') AS source_registry,
		coalesce(blob_ref, '') AS blob_ref,
		certified,
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
	if f.EngineVersion != "" {
		query += ` AND engine_version = ?`
		args = append(args, f.EngineVersion)
	}
	if f.Owner != "" {
		query += ` AND target_id IN (SELECT DISTINCT target_id FROM evidence WHERE 1=1)`
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
	defer func() { _ = rows.Close() }()

	var out []EvidenceRecord
	for rows.Next() {
		var r EvidenceRecord
		if err := rows.Scan(
			&r.EvidenceID, &r.PolicyID, &r.TargetID,
			&r.TargetName, &r.TargetType, &r.TargetEnv,
			&r.EngineName, &r.EngineVersion,
			&r.RuleID, &r.RuleName,
			&r.EvalResult, &r.EvalMessage,
			&r.ControlID, &r.ControlCatalogID, &r.ControlCategory,
			&r.RequirementID, &r.PlanID,
			&r.Confidence, &r.ComplianceStatus,
			&r.RiskLevel, &r.Requirements,
			&r.EnrichmentStatus,
			&r.AttestationRef, &r.SourceRegistry, &r.BlobRef,
			&r.Certified,
			&r.CollectedAt,
		); err != nil {
			return nil, fmt.Errorf("scan evidence: %w", err)
		}
		out = append(out, r)
	}
	return out, nil
}

// CertificationRow represents a single certification verdict.
type CertificationRow struct {
	EvidenceID       string    `json:"evidence_id"`
	Certifier        string    `json:"certifier"`
	CertifierVersion string    `json:"certifier_version"`
	Result           string    `json:"result"`
	Reason           string    `json:"reason"`
	CertifiedAt      time.Time `json:"certified_at,omitempty"`
}

// EvidenceRowLite is a lightweight evidence projection for the certifier pipeline.
type EvidenceRowLite struct {
	EvidenceID       string    `json:"evidence_id"`
	TargetID         string    `json:"target_id"`
	RuleID           string    `json:"rule_id"`
	EvalResult       string    `json:"eval_result"`
	ComplianceStatus string    `json:"compliance_status"`
	EngineName       string    `json:"engine_name"`
	SourceRegistry   string    `json:"source_registry"`
	AttestationRef   string    `json:"attestation_ref"`
	EnrichmentStatus string    `json:"enrichment_status"`
	CollectedAt      time.Time `json:"collected_at"`
}

// InsertCertifications batch-inserts certification verdicts.
func (s *Store) InsertCertifications(ctx context.Context, rows []CertificationRow) error {
	if len(rows) == 0 {
		return nil
	}
	batch, err := s.conn.PrepareBatch(ctx,
		`INSERT INTO certifications (evidence_id, certifier, certifier_version, result, reason)`)
	if err != nil {
		return fmt.Errorf("prepare certifications batch: %w", err)
	}
	for _, r := range rows {
		if err := batch.Append(
			r.EvidenceID, r.Certifier, r.CertifierVersion,
			r.Result, r.Reason,
		); err != nil {
			return fmt.Errorf("append certification: %w", err)
		}
	}
	return batch.Send()
}

// UpdateEvidenceCertified sets the denormalized certified flag on an evidence row.
func (s *Store) UpdateEvidenceCertified(
	ctx context.Context, evidenceID string, certified bool,
) error {
	return s.conn.Exec(ctx,
		`ALTER TABLE evidence UPDATE certified = ? WHERE evidence_id = ?`,
		certified, evidenceID)
}

// QueryCertifications returns certification verdicts for a given evidence row.
func (s *Store) QueryCertifications(
	ctx context.Context, evidenceID string,
) ([]CertificationRow, error) {
	rows, err := s.conn.Query(ctx,
		`SELECT evidence_id, certifier, certifier_version, result, reason, certified_at
		 FROM certifications WHERE evidence_id = ? ORDER BY certified_at DESC`, evidenceID)
	if err != nil {
		return nil, fmt.Errorf("query certifications: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var out []CertificationRow
	for rows.Next() {
		var r CertificationRow
		if err := rows.Scan(
			&r.EvidenceID, &r.Certifier, &r.CertifierVersion,
			&r.Result, &r.Reason, &r.CertifiedAt,
		); err != nil {
			return nil, fmt.Errorf("scan certification: %w", err)
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// QueryRecentEvidence returns lightweight evidence rows for a policy ingested after since.
func (s *Store) QueryRecentEvidence(
	ctx context.Context, policyID string, since time.Time,
) ([]EvidenceRowLite, error) {
	rows, err := s.conn.Query(ctx,
		`SELECT evidence_id, target_id, rule_id, eval_result, compliance_status,
			coalesce(engine_name, '') AS engine_name,
			coalesce(source_registry, '') AS source_registry,
			coalesce(attestation_ref, '') AS attestation_ref,
			enrichment_status, collected_at
		 FROM evidence
		 WHERE policy_id = ? AND ingested_at >= ?
		 ORDER BY ingested_at DESC`, policyID, since)
	if err != nil {
		return nil, fmt.Errorf("query recent evidence: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var out []EvidenceRowLite
	for rows.Next() {
		var r EvidenceRowLite
		if err := rows.Scan(
			&r.EvidenceID, &r.TargetID, &r.RuleID, &r.EvalResult,
			&r.ComplianceStatus, &r.EngineName, &r.SourceRegistry,
			&r.AttestationRef, &r.EnrichmentStatus, &r.CollectedAt,
		); err != nil {
			return nil, fmt.Errorf("scan recent evidence: %w", err)
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// AuditLog represents a stored audit log artifact.
type AuditLog struct {
	AuditID       string    `json:"audit_id"`
	PolicyID      string    `json:"policy_id"`
	AuditStart    time.Time `json:"audit_start"`
	AuditEnd      time.Time `json:"audit_end"`
	Framework     string    `json:"framework,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
	CreatedBy     string    `json:"created_by,omitempty"`
	Content       string    `json:"content"`
	Summary       string    `json:"summary"`
	Model         string    `json:"model,omitempty"`
	PromptVersion string    `json:"prompt_version,omitempty"`
}

// InsertAuditLog stores an AuditLog artifact.
func (s *Store) InsertAuditLog(ctx context.Context, a AuditLog) error {
	if a.AuditID == "" {
		a.AuditID = uuid.New().String()
	}
	return s.conn.Exec(ctx,
		`INSERT INTO audit_logs (audit_id, policy_id, audit_start, audit_end, framework, created_by, content, summary, model, prompt_version) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		a.AuditID, a.PolicyID, a.AuditStart, a.AuditEnd, a.Framework, a.CreatedBy, a.Content, a.Summary, a.Model, a.PromptVersion,
	)
}

// ListAuditLogs returns audit logs for a given policy, optionally filtered by time range.
func (s *Store) ListAuditLogs(ctx context.Context, policyID string, start, end time.Time, limit int) ([]AuditLog, error) {
	query := `SELECT audit_id, policy_id, audit_start, audit_end, framework, created_at, created_by, summary, model, prompt_version FROM audit_logs FINAL WHERE policy_id = ?`
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
	limit = consts.ClampLimit(limit)
	query += fmt.Sprintf(` LIMIT %d`, limit)

	rows, err := s.conn.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list audit logs: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var out []AuditLog
	for rows.Next() {
		var a AuditLog
		if err := rows.Scan(&a.AuditID, &a.PolicyID, &a.AuditStart, &a.AuditEnd, &a.Framework, &a.CreatedAt, &a.CreatedBy, &a.Summary, &a.Model, &a.PromptVersion); err != nil {
			return nil, fmt.Errorf("scan audit log: %w", err)
		}
		out = append(out, a)
	}
	return out, nil
}

// GetAuditLog returns a single audit log with full content.
func (s *Store) GetAuditLog(ctx context.Context, auditID string) (*AuditLog, error) {
	row := s.conn.QueryRow(ctx,
		`SELECT audit_id, policy_id, audit_start, audit_end, framework, created_at, created_by, content, summary, model, prompt_version FROM audit_logs FINAL WHERE audit_id = ?`, auditID)
	var a AuditLog
	if err := row.Scan(&a.AuditID, &a.PolicyID, &a.AuditStart, &a.AuditEnd, &a.Framework, &a.CreatedAt, &a.CreatedBy, &a.Content, &a.Summary, &a.Model, &a.PromptVersion); err != nil {
		return nil, fmt.Errorf("get audit log: %w", err)
	}
	return &a, nil
}

// EvidenceAssessment represents an agent-produced classification for an evidence row.
type EvidenceAssessment struct {
	EvidenceID     string    `json:"evidence_id"`
	PolicyID       string    `json:"policy_id"`
	PlanID         string    `json:"plan_id"`
	Classification string    `json:"classification"`
	Reason         string    `json:"reason"`
	AssessedAt     time.Time `json:"assessed_at"`
	AssessedBy     string    `json:"assessed_by"`
}

// ErrRequirementNotFound is returned by ListRequirementEvidence when the
// requirement ID is not known for the policy (no matching catalog row and no
// evidence rows scoped to that ID).
var ErrRequirementNotFound = errors.New("requirement not found")
var ErrDraftAlreadyPromoted = errors.New("draft already promoted")
var ErrDraftNotFound = errors.New("draft not found")

// ValidClassifications enumerates the allowed 7-state classification values.
var ValidClassifications = map[string]bool{
	"Healthy":        true,
	"Failing":        true,
	"Wrong Source":   true,
	"Wrong Method":   true,
	"Unfit Evidence": true,
	"Stale":          true,
	"No Evidence":    true,
}

// InsertEvidenceAssessments batch-inserts agent classifications.
func (s *Store) InsertEvidenceAssessments(ctx context.Context, assessments []EvidenceAssessment) error {
	if len(assessments) == 0 {
		return nil
	}
	batch, err := s.conn.PrepareBatch(ctx,
		`INSERT INTO evidence_assessments (evidence_id, policy_id, plan_id, classification, reason, assessed_at, assessed_by)`)
	if err != nil {
		return fmt.Errorf("prepare evidence assessments batch: %w", err)
	}
	for _, a := range assessments {
		if err := batch.Append(a.EvidenceID, a.PolicyID, a.PlanID, a.Classification, a.Reason, a.AssessedAt, a.AssessedBy); err != nil {
			return fmt.Errorf("append evidence assessment: %w", err)
		}
	}
	return batch.Send()
}

// Catalog represents a stored catalog artifact (ControlCatalog, ThreatCatalog, etc.).
type Catalog struct {
	CatalogID   string    `json:"catalog_id"`
	CatalogType string    `json:"catalog_type"`
	Title       string    `json:"title"`
	Content     string    `json:"content"`
	PolicyID    string    `json:"policy_id,omitempty"`
	ImportedAt  time.Time `json:"imported_at"`
}

// InsertCatalog stores a raw catalog artifact.
func (s *Store) InsertCatalog(ctx context.Context, c Catalog) error {
	return s.conn.Exec(ctx,
		`INSERT INTO catalogs (catalog_id, catalog_type, title, content, policy_id) VALUES (?, ?, ?, ?, ?)`,
		c.CatalogID, c.CatalogType, c.Title, c.Content, c.PolicyID,
	)
}

// ListCatalogs returns all stored catalogs (without content for efficiency).
func (s *Store) ListCatalogs(ctx context.Context) ([]Catalog, error) {
	rows, err := s.conn.Query(ctx,
		`SELECT catalog_id, catalog_type, title, policy_id, imported_at FROM catalogs FINAL ORDER BY imported_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("list catalogs: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var out []Catalog
	for rows.Next() {
		var c Catalog
		if err := rows.Scan(&c.CatalogID, &c.CatalogType, &c.Title, &c.PolicyID, &c.ImportedAt); err != nil {
			return nil, fmt.Errorf("scan catalog: %w", err)
		}
		out = append(out, c)
	}
	return out, nil
}

// GetCatalog returns a single catalog with full content.
func (s *Store) GetCatalog(ctx context.Context, catalogID string) (*Catalog, error) {
	row := s.conn.QueryRow(ctx,
		`SELECT catalog_id, catalog_type, title, content, policy_id, imported_at FROM catalogs FINAL WHERE catalog_id = ?`, catalogID)
	var c Catalog
	if err := row.Scan(&c.CatalogID, &c.CatalogType, &c.Title, &c.Content, &c.PolicyID, &c.ImportedAt); err != nil {
		return nil, fmt.Errorf("get catalog: %w", err)
	}
	return &c, nil
}

// DraftAuditLog represents an agent-produced audit log awaiting human review.
type DraftAuditLog struct {
	DraftID        string     `json:"draft_id"`
	PolicyID       string     `json:"policy_id"`
	AuditStart     time.Time  `json:"audit_start"`
	AuditEnd       time.Time  `json:"audit_end"`
	Framework      string     `json:"framework,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	Status         string     `json:"status"`
	Content        string     `json:"content"`
	Summary        string     `json:"summary"`
	AgentReasoning string     `json:"agent_reasoning,omitempty"`
	Model          string     `json:"model,omitempty"`
	PromptVersion  string     `json:"prompt_version,omitempty"`
	ReviewedBy     string     `json:"reviewed_by,omitempty"`
	PromotedAt     *time.Time `json:"promoted_at,omitempty"`
	ReviewerEdits  string     `json:"reviewer_edits,omitempty"`
}

// InsertDraftAuditLog stores an agent-produced draft.
func (s *Store) InsertDraftAuditLog(ctx context.Context, d DraftAuditLog) error {
	if d.DraftID == "" {
		d.DraftID = uuid.New().String()
	}
	if d.Status == "" {
		d.Status = "pending_review"
	}
	edits := d.ReviewerEdits
	if edits == "" {
		edits = "{}"
	}
	return s.conn.Exec(ctx,
		`INSERT INTO draft_audit_logs (draft_id, policy_id, audit_start, audit_end, framework, status, content, summary, agent_reasoning, model, prompt_version, reviewer_edits) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		d.DraftID, d.PolicyID, d.AuditStart, d.AuditEnd, d.Framework, d.Status, d.Content, d.Summary, d.AgentReasoning, d.Model, d.PromptVersion, edits,
	)
}

// ListDraftAuditLogs returns drafts filtered by status. Empty status returns all.
func (s *Store) ListDraftAuditLogs(ctx context.Context, status string, limit int) ([]DraftAuditLog, error) {
	query := `SELECT draft_id, policy_id, audit_start, audit_end, framework, created_at, status, summary, agent_reasoning, model, prompt_version, reviewed_by, promoted_at, reviewer_edits FROM draft_audit_logs FINAL`
	var args []any
	if status != "" {
		query += ` WHERE status = ?`
		args = append(args, status)
	}
	query += ` ORDER BY created_at DESC`
	limit = consts.ClampLimit(limit)
	query += fmt.Sprintf(` LIMIT %d`, limit)

	rows, err := s.conn.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list draft audit logs: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var out []DraftAuditLog
	for rows.Next() {
		var d DraftAuditLog
		if err := rows.Scan(&d.DraftID, &d.PolicyID, &d.AuditStart, &d.AuditEnd, &d.Framework, &d.CreatedAt, &d.Status, &d.Summary, &d.AgentReasoning, &d.Model, &d.PromptVersion, &d.ReviewedBy, &d.PromotedAt, &d.ReviewerEdits); err != nil {
			return nil, fmt.Errorf("scan draft audit log: %w", err)
		}
		out = append(out, d)
	}
	return out, nil
}

// GetDraftAuditLog returns a single draft with full content.
func (s *Store) GetDraftAuditLog(ctx context.Context, draftID string) (*DraftAuditLog, error) {
	row := s.conn.QueryRow(ctx,
		`SELECT draft_id, policy_id, audit_start, audit_end, framework, created_at, status, content, summary, agent_reasoning, model, prompt_version, reviewed_by, promoted_at, reviewer_edits FROM draft_audit_logs FINAL WHERE draft_id = ?`, draftID)
	var d DraftAuditLog
	if err := row.Scan(&d.DraftID, &d.PolicyID, &d.AuditStart, &d.AuditEnd, &d.Framework, &d.CreatedAt, &d.Status, &d.Content, &d.Summary, &d.AgentReasoning, &d.Model, &d.PromptVersion, &d.ReviewedBy, &d.PromotedAt, &d.ReviewerEdits); err != nil {
		return nil, fmt.Errorf("get draft audit log: %w", err)
	}
	return &d, nil
}

// UpdateDraftEdits persists reviewer edits (type overrides, notes) on a pending draft.
// Uses ClickHouse ReplacingMergeTree — inserts a new row version with updated reviewer_edits.
func (s *Store) UpdateDraftEdits(ctx context.Context, draftID string, reviewerEdits string) error {
	draft, err := s.GetDraftAuditLog(ctx, draftID)
	if err != nil {
		return fmt.Errorf("%w: %s", ErrDraftNotFound, draftID)
	}
	if draft.Status != "pending_review" {
		return ErrDraftAlreadyPromoted
	}
	return s.conn.Exec(ctx,
		`INSERT INTO draft_audit_logs (draft_id, policy_id, audit_start, audit_end, framework, status, content, summary, agent_reasoning, model, prompt_version, reviewer_edits) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		draft.DraftID, draft.PolicyID, draft.AuditStart, draft.AuditEnd, draft.Framework, draft.Status, draft.Content, draft.Summary, draft.AgentReasoning, draft.Model, draft.PromptVersion, reviewerEdits,
	)
}

// PromoteDraftAuditLog copies a draft to the official audit_logs table and marks it promoted.
func (s *Store) PromoteDraftAuditLog(ctx context.Context, draftID string, reviewedBy string) error {
	draft, err := s.GetDraftAuditLog(ctx, draftID)
	if err != nil {
		return fmt.Errorf("%w: %s", ErrDraftNotFound, draftID)
	}
	if draft.Status == "promoted" {
		return ErrDraftAlreadyPromoted
	}

	mergedContent, err := mergeReviewerEdits(draft.Content, draft.ReviewerEdits)
	if err != nil {
		slog.Warn("reviewer edits merge failed, using original content", "draft_id", draftID, "error", err)
		mergedContent = draft.Content
	}

	official := AuditLog{
		AuditID:       uuid.New().String(),
		PolicyID:      draft.PolicyID,
		AuditStart:    draft.AuditStart,
		AuditEnd:      draft.AuditEnd,
		Framework:     draft.Framework,
		CreatedBy:     reviewedBy,
		Content:       mergedContent,
		Summary:       draft.Summary,
		Model:         draft.Model,
		PromptVersion: draft.PromptVersion,
	}
	if err := s.InsertAuditLog(ctx, official); err != nil {
		return fmt.Errorf("insert promoted audit log: %w", err)
	}

	now := time.Now()
	return s.conn.Exec(ctx,
		`INSERT INTO draft_audit_logs (draft_id, policy_id, audit_start, audit_end, framework, status, content, summary, agent_reasoning, model, prompt_version, reviewed_by, promoted_at, reviewer_edits) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		draft.DraftID, draft.PolicyID, draft.AuditStart, draft.AuditEnd, draft.Framework, "promoted", draft.Content, draft.Summary, draft.AgentReasoning, draft.Model, draft.PromptVersion, reviewedBy, &now, draft.ReviewerEdits,
	)
}

// PostureRow is a per-policy compliance posture aggregate with inventory context.
type PostureRow struct {
	PolicyID       string `json:"policy_id"`
	Title          string `json:"title"`
	Version        string `json:"version,omitempty"`
	TotalRows      uint64 `json:"total_rows"`
	PassedRows     uint64 `json:"passed_rows"`
	FailedRows     uint64 `json:"failed_rows"`
	OtherRows      uint64 `json:"other_rows"`
	LatestAt       string `json:"latest_at,omitempty"`
	TargetCount    uint64 `json:"target_count"`
	ControlCount   uint64 `json:"control_count"`
	LatestEvidence string `json:"latest_evidence_at,omitempty"`
	Owner          string `json:"owner"`
}

// ListPosture returns per-policy evidence posture aggregates with inventory context.
// When start or end are non-zero the evidence window is restricted to that range.
func (s *Store) ListPosture(ctx context.Context, start, end time.Time) ([]PostureRow, error) {
	var (
		evidenceFilter string
		args           []any
	)
	if !start.IsZero() || !end.IsZero() {
		evidenceFilter = " WHERE 1=1"
		if !start.IsZero() {
			evidenceFilter += " AND collected_at >= ?"
			args = append(args, start)
		}
		if !end.IsZero() {
			evidenceFilter += " AND collected_at <= ?"
			args = append(args, end)
		}
	}

	query := `
		SELECT
			p.policy_id,
			p.title,
			coalesce(p.version, '') AS policy_version,
			countIf(e.evidence_id != '') AS total_rows,
			countIf(e.eval_result = 'Passed') AS passed_rows,
			countIf(e.eval_result = 'Failed') AS failed_rows,
			countIf(e.eval_result NOT IN ('Passed', 'Failed')) AS other_rows,
			if(count(e.evidence_id) > 0, toString(max(e.collected_at)), '') AS latest_at,
			uniqIf(e.target_id, e.target_id != '') AS target_count,
			uniqIf(e.control_id, e.control_id != '') AS control_count,
			if(count(e.evidence_id) > 0, toString(max(e.collected_at)), '') AS latest_evidence_at
		FROM (SELECT policy_id, title, version FROM policies FINAL) AS p
		LEFT JOIN (SELECT * FROM evidence` + evidenceFilter + `) e ON e.policy_id = p.policy_id
		GROUP BY p.policy_id, p.title, policy_version
		ORDER BY p.title`

	rows, err := s.conn.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list posture: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var out []PostureRow
	for rows.Next() {
		var r PostureRow
		if err := rows.Scan(
			&r.PolicyID, &r.Title, &r.Version,
			&r.TotalRows, &r.PassedRows, &r.FailedRows, &r.OtherRows,
			&r.LatestAt,
			&r.TargetCount, &r.ControlCount, &r.LatestEvidence,
		); err != nil {
			return nil, fmt.Errorf("scan posture row: %w", err)
		}
		out = append(out, r)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	s.enrichPostureOwners(ctx, out)
	return out, nil
}

// enrichPostureOwners extracts the accountable contact from each policy's YAML content.
func (s *Store) enrichPostureOwners(ctx context.Context, posture []PostureRow) {
	for i, r := range posture {
		row := s.conn.QueryRow(ctx,
			`SELECT content FROM policies FINAL WHERE policy_id = ? LIMIT 1`, r.PolicyID)
		var content string
		if err := row.Scan(&content); err != nil {
			continue
		}
		posture[i].Owner = gemara.ExtractAccountableContact(content)
	}
}

// QueryPolicyPosture returns aggregate evidence counts for a single policy.
func (s *Store) QueryPolicyPosture(ctx context.Context, policyID string) (total, passed, failed uint64, err error) {
	row := s.conn.QueryRow(ctx, `
		SELECT
			count() AS total,
			countIf(eval_result = 'Passed') AS passed,
			countIf(eval_result = 'Failed') AS failed
		FROM evidence
		WHERE policy_id = ?`, policyID)
	if err := row.Scan(&total, &passed, &failed); err != nil {
		return 0, 0, 0, fmt.Errorf("query policy posture: %w", err)
	}
	return total, passed, failed, nil
}

// Notification represents an inbox notification stored in ClickHouse.
type Notification struct {
	NotificationID string    `json:"notification_id"`
	Type           string    `json:"type"`
	PolicyID       string    `json:"policy_id"`
	Payload        string    `json:"payload"`
	Read           bool      `json:"read"`
	CreatedAt      time.Time `json:"created_at"`
}

// InsertNotification persists a new notification.
func (s *Store) InsertNotification(ctx context.Context, n Notification) error {
	if n.NotificationID == "" {
		n.NotificationID = uuid.New().String()
	}
	return s.conn.Exec(ctx,
		`INSERT INTO notifications (notification_id, type, policy_id, payload) VALUES (?, ?, ?, ?)`,
		n.NotificationID, n.Type, n.PolicyID, n.Payload)
}

// ListNotifications returns recent notifications ordered newest-first.
func (s *Store) ListNotifications(ctx context.Context, limit int) ([]Notification, error) {
	limit = consts.ClampLimit(limit)
	rows, err := s.conn.Query(ctx,
		fmt.Sprintf(`SELECT notification_id, type, policy_id, payload, read, created_at
			FROM notifications FINAL
			ORDER BY created_at DESC LIMIT %d`, limit))
	if err != nil {
		return nil, fmt.Errorf("list notifications: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var out []Notification
	for rows.Next() {
		var n Notification
		var readFlag uint8
		if err := rows.Scan(&n.NotificationID, &n.Type, &n.PolicyID, &n.Payload, &readFlag, &n.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan notification: %w", err)
		}
		n.Read = readFlag == 1
		out = append(out, n)
	}
	return out, rows.Err()
}

// MarkRead marks a notification as read using ReplacingMergeTree semantics.
func (s *Store) MarkRead(ctx context.Context, notificationID string) error {
	return s.conn.Exec(ctx,
		`INSERT INTO notifications (notification_id, type, policy_id, payload, read, created_at)
		 SELECT notification_id, type, policy_id, payload, 1, now64(3)
		 FROM notifications FINAL
		 WHERE notification_id = ?`, notificationID)
}

// UnreadCount returns the number of unread notifications.
func (s *Store) UnreadCount(ctx context.Context) (int, error) {
	row := s.conn.QueryRow(ctx,
		`SELECT count() FROM notifications FINAL WHERE read = 0`)
	var count uint64
	if err := row.Scan(&count); err != nil {
		return 0, fmt.Errorf("unread count: %w", err)
	}
	return int(count), nil
}

// RequirementFilter holds query parameters for requirement matrix and evidence APIs.
type RequirementFilter struct {
	PolicyID       string
	Start          time.Time
	End            time.Time
	ControlFamily  string
	Classification string
	Limit          int
	Offset         int
}

// RequirementRow is a single row in the requirement matrix.
type RequirementRow struct {
	CatalogID       string `json:"catalog_id"`
	ControlID       string `json:"control_id"`
	ControlTitle    string `json:"control_title"`
	RequirementID   string `json:"requirement_id"`
	RequirementText string `json:"requirement_text"`
	EvidenceCount   uint64 `json:"evidence_count"`
	LatestEvidence  string `json:"latest_evidence,omitempty"`
	Classification  string `json:"classification,omitempty"`
}

// RequirementEvidenceRow is an evidence row returned in requirement drill-down.
type RequirementEvidenceRow struct {
	EvidenceID     string `json:"evidence_id"`
	TargetID       string `json:"target_id"`
	TargetName     string `json:"target_name,omitempty"`
	RuleID         string `json:"rule_id"`
	EvalResult     string `json:"eval_result"`
	Classification string `json:"classification,omitempty"`
	AssessedAt     string `json:"assessed_at,omitempty"`
	CollectedAt    string `json:"collected_at"`
	SourceRegistry string `json:"source_registry,omitempty"`
}

// ListRequirementMatrix returns requirement rows with evidence aggregates.
// Uses evidence as the base table so rows appear even when
// assessment_requirements has not been populated (no catalog import).
// When assessment_requirements IS populated, requirement text and IDs
// are joined in; otherwise those columns are empty and the view is
// control-level.
func (s *Store) ListRequirementMatrix(ctx context.Context, f RequirementFilter) ([]RequirementRow, error) {
	query := `
		SELECT
			coalesce(ar.catalog_id, '') AS catalog_id,
			e.control_id,
			coalesce(c.title, '') AS control_title,
			coalesce(ar.requirement_id, e.control_id) AS requirement_id,
			coalesce(ar.text, '') AS requirement_text,
			count(DISTINCT e.evidence_id) AS evidence_count,
			if(count(e.evidence_id) > 0, toString(max(e.collected_at)), '') AS latest_evidence,
			coalesce(any(ea_latest.classification), 'Unassessed') AS classification
		FROM evidence e
		LEFT JOIN (SELECT catalog_id, control_id, title FROM controls FINAL) AS c
			ON c.control_id = e.control_id
		LEFT JOIN assessment_requirements ar
			ON ar.control_id = e.control_id AND ar.catalog_id = c.catalog_id
		LEFT JOIN (
			SELECT evidence_id, argMax(classification, assessed_at) AS classification
			FROM evidence_assessments
			GROUP BY evidence_id
		) AS ea_latest ON ea_latest.evidence_id = e.evidence_id
		WHERE e.policy_id = ?`

	args := []any{f.PolicyID}

	if !f.Start.IsZero() {
		query += ` AND e.collected_at >= ?`
		args = append(args, f.Start)
	}
	if !f.End.IsZero() {
		query += ` AND e.collected_at <= ?`
		args = append(args, f.End)
	}
	if f.ControlFamily != "" {
		query += ` AND startsWith(e.control_id, ?)`
		args = append(args, f.ControlFamily)
	}

	query += ` GROUP BY catalog_id, e.control_id, control_title, requirement_id, requirement_text, ea_latest.classification
		ORDER BY e.control_id, requirement_id`

	if f.Limit <= 0 {
		f.Limit = 100
	}
	query += fmt.Sprintf(` LIMIT %d`, f.Limit)
	if f.Offset > 0 {
		query += fmt.Sprintf(` OFFSET %d`, f.Offset)
	}

	rows, err := s.conn.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list requirement matrix: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var out []RequirementRow
	for rows.Next() {
		var r RequirementRow
		if err := rows.Scan(
			&r.CatalogID, &r.ControlID, &r.ControlTitle,
			&r.RequirementID, &r.RequirementText,
			&r.EvidenceCount, &r.LatestEvidence,
			&r.Classification,
		); err != nil {
			return nil, fmt.Errorf("scan requirement matrix row: %w", err)
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// requirementKnownForPolicy reports whether requirementID refers to at least
// one assessment requirement for the policy's controls or to evidence rows for
// that policy (control_id or requirement_id match).
func (s *Store) requirementKnownForPolicy(ctx context.Context, policyID, requirementID string) (bool, error) {
	var evCount uint64
	if err := s.conn.QueryRow(ctx,
		`SELECT count() FROM evidence WHERE policy_id = ? AND (control_id = ? OR requirement_id = ?)`,
		policyID, requirementID, requirementID,
	).Scan(&evCount); err != nil {
		return false, fmt.Errorf("count evidence for requirement: %w", err)
	}
	if evCount > 0 {
		return true, nil
	}

	var arCount uint64
	if err := s.conn.QueryRow(ctx,
		`SELECT count() FROM assessment_requirements ar
		 INNER JOIN (SELECT catalog_id, control_id, policy_id FROM controls FINAL) AS c
		   ON c.catalog_id = ar.catalog_id AND c.control_id = ar.control_id
		 WHERE ar.requirement_id = ? AND c.policy_id = ?`,
		requirementID, policyID,
	).Scan(&arCount); err != nil {
		return false, fmt.Errorf("count assessment requirements for requirement: %w", err)
	}
	return arCount > 0, nil
}

// ListRequirementEvidence returns evidence rows for a specific requirement.
func (s *Store) ListRequirementEvidence(ctx context.Context, requirementID string, f RequirementFilter) ([]RequirementEvidenceRow, error) {
	known, err := s.requirementKnownForPolicy(ctx, f.PolicyID, requirementID)
	if err != nil {
		return nil, err
	}
	if !known {
		return nil, ErrRequirementNotFound
	}

	query := `
		SELECT
			e.evidence_id,
			e.target_id,
			coalesce(e.target_name, '') AS target_name,
			e.rule_id,
			e.eval_result,
			coalesce(ea_latest.classification, '') AS classification,
			coalesce(ea_latest.last_assessed, '') AS assessed_at,
			toString(e.collected_at) AS collected_at,
			coalesce(e.source_registry, '') AS source_registry
		FROM evidence e
		LEFT JOIN (
			SELECT evidence_id,
				argMax(classification, assessed_at) AS classification,
				toString(max(assessed_at)) AS last_assessed
			FROM evidence_assessments
			GROUP BY evidence_id
		) AS ea_latest ON ea_latest.evidence_id = e.evidence_id
		WHERE (e.control_id = ? OR e.control_id IN (
			SELECT control_id FROM assessment_requirements
			WHERE requirement_id = ?
		)) AND e.policy_id = ?`

	args := []any{requirementID, requirementID, f.PolicyID}

	if !f.Start.IsZero() {
		query += ` AND e.collected_at >= ?`
		args = append(args, f.Start)
	}
	if !f.End.IsZero() {
		query += ` AND e.collected_at <= ?`
		args = append(args, f.End)
	}

	query += ` ORDER BY e.collected_at DESC`

	if f.Limit <= 0 {
		f.Limit = 100
	}
	query += fmt.Sprintf(` LIMIT %d`, f.Limit)
	if f.Offset > 0 {
		query += fmt.Sprintf(` OFFSET %d`, f.Offset)
	}

	rows, err := s.conn.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list requirement evidence: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var out []RequirementEvidenceRow
	for rows.Next() {
		var r RequirementEvidenceRow
		if err := rows.Scan(
			&r.EvidenceID, &r.TargetID, &r.TargetName,
			&r.RuleID, &r.EvalResult,
			&r.Classification, &r.AssessedAt,
			&r.CollectedAt, &r.SourceRegistry,
		); err != nil {
			return nil, fmt.Errorf("scan requirement evidence row: %w", err)
		}
		out = append(out, r)
	}
	return out, rows.Err()
}
