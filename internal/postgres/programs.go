// SPDX-License-Identifier: Apache-2.0

package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrProgramNotFound        = errors.New("program not found")
	ErrProgramVersionConflict = errors.New("program version conflict")
	ErrJobNotFound            = errors.New("job not found")
)

type Program struct {
	ID                string          `json:"id"`
	Name              string          `json:"name"`
	GuidanceCatalogID *string         `json:"guidance_catalog_id"`
	Framework         string          `json:"framework"`
	Applicability     []string        `json:"applicability"`
	Status            string          `json:"status"`
	Health            *string         `json:"health"`
	Owner             *string         `json:"owner"`
	Description       *string         `json:"description"`
	Metadata          json.RawMessage `json:"metadata"`
	PolicyIDs         []string        `json:"policy_ids"`
	Environments      []string        `json:"environments"`
	Version           int             `json:"version"`
	GreenPct          int             `json:"green_pct"`
	RedPct            int             `json:"red_pct"`
	ScorePct          int             `json:"score_pct"`
	CreatedAt         time.Time       `json:"created_at"`
	UpdatedAt         time.Time       `json:"updated_at"`
}

type Job struct {
	ID        string    `json:"id"`
	ProgramID string    `json:"program_id"`
	Agent     string    `json:"agent"`
	UserID    string    `json:"user_id"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type ProgramPG struct {
	pool *pgxpool.Pool
}

func NewProgramPG(pool *pgxpool.Pool) *ProgramPG {
	return &ProgramPG{pool: pool}
}

func (s *ProgramPG) ListPrograms(ctx context.Context) ([]Program, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, name, guidance_catalog_id, framework, applicability, status,
			health, owner, description, metadata, policy_ids, environments,
			version, green_pct, red_pct, score_pct, created_at, updated_at
		FROM programs
		WHERE deleted_at IS NULL
		ORDER BY created_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("list programs: %w", err)
	}
	defer rows.Close()

	var out []Program
	for rows.Next() {
		var p Program
		if err := rows.Scan(
			&p.ID, &p.Name, &p.GuidanceCatalogID, &p.Framework, &p.Applicability, &p.Status,
			&p.Health, &p.Owner, &p.Description, &p.Metadata, &p.PolicyIDs, &p.Environments,
			&p.Version, &p.GreenPct, &p.RedPct, &p.ScorePct, &p.CreatedAt, &p.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("list programs scan: %w", err)
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (s *ProgramPG) GetProgram(ctx context.Context, id string) (*Program, error) {
	row := s.pool.QueryRow(ctx, `
		SELECT id, name, guidance_catalog_id, framework, applicability, status,
			health, owner, description, metadata, policy_ids, environments,
			version, green_pct, red_pct, score_pct, created_at, updated_at
		FROM programs
		WHERE id = $1 AND deleted_at IS NULL`, id)
	var p Program
	if err := row.Scan(
		&p.ID, &p.Name, &p.GuidanceCatalogID, &p.Framework, &p.Applicability, &p.Status,
		&p.Health, &p.Owner, &p.Description, &p.Metadata, &p.PolicyIDs, &p.Environments,
		&p.Version, &p.GreenPct, &p.RedPct, &p.ScorePct, &p.CreatedAt, &p.UpdatedAt,
	); err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("get program %s: %w", id, ErrProgramNotFound)
		}
		return nil, fmt.Errorf("get program %s: %w", id, err)
	}
	return &p, nil
}

func (s *ProgramPG) CreateProgram(ctx context.Context, p Program) (*Program, error) {
	md := p.Metadata
	if len(md) == 0 {
		md = json.RawMessage([]byte("{}"))
	}
	app := p.Applicability
	if app == nil {
		app = []string{}
	}
	pol := p.PolicyIDs
	if pol == nil {
		pol = []string{}
	}
	env := p.Environments
	if env == nil {
		env = []string{}
	}
	status := p.Status
	if status == "" {
		status = "intake"
	}
	ver := p.Version
	if ver == 0 {
		ver = 1
	}
	if (p.GuidanceCatalogID == nil || *p.GuidanceCatalogID == "") && p.Framework != "" {
		var resolved string
		_ = s.pool.QueryRow(ctx, `
			SELECT catalog_id FROM catalogs
			WHERE title ILIKE '%' || $1 || '%' OR catalog_id ILIKE '%' || $1 || '%'
			LIMIT 1`, p.Framework).Scan(&resolved)
		if resolved != "" {
			p.GuidanceCatalogID = &resolved
		}
	}

	greenPct := p.GreenPct
	if greenPct == 0 {
		greenPct = 90
	}
	redPct := p.RedPct
	if redPct == 0 {
		redPct = 50
	}

	row := s.pool.QueryRow(ctx, `
		INSERT INTO programs (
			name, guidance_catalog_id, framework, applicability, status,
			health, owner, description, metadata, policy_ids, environments,
			version, green_pct, red_pct
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14
		)
		RETURNING id, name, guidance_catalog_id, framework, applicability, status,
			health, owner, description, metadata, policy_ids, environments,
			version, green_pct, red_pct, score_pct, created_at, updated_at`,
		p.Name, p.GuidanceCatalogID, p.Framework, app, status,
		p.Health, p.Owner, p.Description, md, pol, env,
		ver, greenPct, redPct,
	)

	var out Program
	if err := row.Scan(
		&out.ID, &out.Name, &out.GuidanceCatalogID, &out.Framework, &out.Applicability, &out.Status,
		&out.Health, &out.Owner, &out.Description, &out.Metadata, &out.PolicyIDs, &out.Environments,
		&out.Version, &out.GreenPct, &out.RedPct, &out.ScorePct, &out.CreatedAt, &out.UpdatedAt,
	); err != nil {
		return nil, fmt.Errorf("create program: %w", err)
	}
	return &out, nil
}

func (s *ProgramPG) UpdateProgram(ctx context.Context, p Program) error {
	md := p.Metadata
	if len(md) == 0 {
		md = json.RawMessage([]byte("{}"))
	}
	app := p.Applicability
	if app == nil {
		app = []string{}
	}
	pol := p.PolicyIDs
	if pol == nil {
		pol = []string{}
	}
	env := p.Environments
	if env == nil {
		env = []string{}
	}

	tag, err := s.pool.Exec(ctx, `
		UPDATE programs SET
			name = $1,
			guidance_catalog_id = $2,
			framework = $3,
			applicability = $4,
			status = $5,
			health = $6,
			owner = $7,
			description = $8,
			metadata = $9,
			policy_ids = $10,
			environments = $11,
			green_pct = $12,
			red_pct = $13,
			version = version + 1,
			updated_at = now()
		WHERE id = $14 AND version = $15 AND deleted_at IS NULL`,
		p.Name, p.GuidanceCatalogID, p.Framework, app, p.Status,
		p.Health, p.Owner, p.Description, md, pol, env,
		p.GreenPct, p.RedPct,
		p.ID, p.Version,
	)
	if err != nil {
		return fmt.Errorf("update program %s: %w", p.ID, err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("update program %s: %w", p.ID, ErrProgramVersionConflict)
	}
	return nil
}

func (s *ProgramPG) DeleteProgram(ctx context.Context, id string) error {
	tag, err := s.pool.Exec(ctx, `
		UPDATE programs SET deleted_at = now(), updated_at = now()
		WHERE id = $1 AND deleted_at IS NULL`, id)
	if err != nil {
		return fmt.Errorf("delete program %s: %w", id, err)
	}
	if tag.RowsAffected() == 0 {
		slog.Warn("delete program: no row updated", "id", id)
		return fmt.Errorf("delete program %s: %w", id, ErrProgramNotFound)
	}
	return nil
}

func (s *ProgramPG) ListJobs(ctx context.Context, programID string) ([]Job, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, program_id, agent, user_id, status, created_at, updated_at
		FROM jobs
		WHERE program_id = $1
		ORDER BY created_at DESC`, programID)
	if err != nil {
		return nil, fmt.Errorf("list jobs: %w", err)
	}
	defer rows.Close()

	var out []Job
	for rows.Next() {
		var j Job
		if err := rows.Scan(&j.ID, &j.ProgramID, &j.Agent, &j.UserID, &j.Status, &j.CreatedAt, &j.UpdatedAt); err != nil {
			return nil, fmt.Errorf("list jobs scan: %w", err)
		}
		out = append(out, j)
	}
	return out, rows.Err()
}

func (s *ProgramPG) CreateJob(ctx context.Context, j Job) (*Job, error) {
	st := j.Status
	if st == "" {
		st = "pending"
	}
	row := s.pool.QueryRow(ctx, `
		INSERT INTO jobs (program_id, agent, user_id, status)
		VALUES ($1, $2, $3, $4)
		RETURNING id, program_id, agent, user_id, status, created_at, updated_at`,
		j.ProgramID, j.Agent, j.UserID, st,
	)
	var out Job
	if err := row.Scan(&out.ID, &out.ProgramID, &out.Agent, &out.UserID, &out.Status, &out.CreatedAt, &out.UpdatedAt); err != nil {
		return nil, fmt.Errorf("create job: %w", err)
	}
	return &out, nil
}

func (s *ProgramPG) UpdateJobStatus(ctx context.Context, id, status string) error {
	tag, err := s.pool.Exec(ctx, `
		UPDATE jobs SET status = $1, updated_at = now()
		WHERE id = $2`, status, id)
	if err != nil {
		return fmt.Errorf("update job status %s: %w", id, err)
	}
	if tag.RowsAffected() == 0 {
		slog.Warn("update job status: no row updated", "id", id)
		return fmt.Errorf("update job status %s: %w", id, ErrJobNotFound)
	}
	return nil
}
