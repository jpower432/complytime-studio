// SPDX-License-Identifier: Apache-2.0

package service

import (
	"context"
	"fmt"
	"time"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"

	studiov1 "github.com/complytime/complytime-studio/gen/studio/v1"
	"github.com/complytime/complytime-studio/internal/store"
)

// StudioService implements the ConnectRPC StudioServiceHandler interface,
// delegating to the same store interfaces used by the public REST handlers.
type StudioService struct {
	policies  store.PolicyStore
	mappings  store.MappingStore
	evidence  store.EvidenceStore
	posture   store.PostureStore
	catalogs  store.CatalogStore
	auditLogs store.AuditLogStore
	drafts    store.DraftAuditLogStore
	threats   store.ThreatStore
	risks     store.RiskStore
	pub       store.EventPublisher
}

// New creates a StudioService from the gateway's shared store interfaces.
func New(s store.Stores) *StudioService {
	return &StudioService{
		policies:  s.Policies,
		mappings:  s.Mappings,
		evidence:  s.Evidence,
		posture:   s.Posture,
		catalogs:  s.Catalogs,
		auditLogs: s.AuditLogs,
		drafts:    s.DraftAuditLogs,
		threats:   s.Threats,
		risks:     s.Risks,
		pub:       s.EventPublisher,
	}
}

// ---------------------------------------------------------------------------
// Policies
// ---------------------------------------------------------------------------

func (s *StudioService) ListPolicies(
	ctx context.Context,
	_ *connect.Request[studiov1.ListPoliciesRequest],
) (*connect.Response[studiov1.ListPoliciesResponse], error) {
	rows, err := s.policies.ListPolicies(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	out := make([]*studiov1.Policy, len(rows))
	for i, p := range rows {
		out[i] = policyToProto(p)
	}
	return connect.NewResponse(&studiov1.ListPoliciesResponse{Policies: out}), nil
}

func (s *StudioService) GetPolicy(
	ctx context.Context,
	req *connect.Request[studiov1.GetPolicyRequest],
) (*connect.Response[studiov1.GetPolicyResponse], error) {
	p, err := s.policies.GetPolicy(ctx, req.Msg.PolicyId)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("policy %q: %w", req.Msg.PolicyId, err))
	}
	var mappings []*studiov1.MappingDocument
	if s.mappings != nil {
		docs, err := s.mappings.ListMappings(ctx, req.Msg.PolicyId)
		if err == nil {
			mappings = make([]*studiov1.MappingDocument, len(docs))
			for i, m := range docs {
				mappings[i] = mappingToProto(m)
			}
		}
	}
	return connect.NewResponse(&studiov1.GetPolicyResponse{
		Policy:   policyToProto(*p),
		Mappings: mappings,
	}), nil
}

// ---------------------------------------------------------------------------
// Evidence
// ---------------------------------------------------------------------------

func (s *StudioService) QueryEvidence(
	ctx context.Context,
	req *connect.Request[studiov1.QueryEvidenceRequest],
) (*connect.Response[studiov1.QueryEvidenceResponse], error) {
	f := store.EvidenceFilter{
		PolicyIDs:     req.Msg.PolicyIds,
		ControlID:     req.Msg.ControlId,
		TargetName:    req.Msg.TargetName,
		TargetType:    req.Msg.TargetType,
		TargetEnv:     req.Msg.TargetEnv,
		EngineVersion: req.Msg.EngineVersion,
		Owner:         req.Msg.Owner,
		Start:         tsToTime(req.Msg.Start),
		End:           tsToTime(req.Msg.End),
		Limit:         int(req.Msg.Limit),
		Offset:        int(req.Msg.Offset),
	}
	rows, err := s.evidence.QueryEvidence(ctx, f)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	out := make([]*studiov1.EvidenceRecord, len(rows))
	for i, r := range rows {
		out[i] = evidenceToProto(r)
	}
	return connect.NewResponse(&studiov1.QueryEvidenceResponse{Records: out}), nil
}

func (s *StudioService) IngestEvidence(
	ctx context.Context,
	req *connect.Request[studiov1.IngestEvidenceRequest],
) (*connect.Response[studiov1.IngestEvidenceResponse], error) {
	records, policyID, err := store.ParseAndFlattenEvidence(ctx, []byte(req.Msg.YamlContent))
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	count, err := s.evidence.InsertEvidence(ctx, records)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if s.pub != nil && count > 0 && policyID != "" {
		s.pub.PublishEvidence(policyID, count)
	}
	return connect.NewResponse(&studiov1.IngestEvidenceResponse{
		Inserted: int32(count),
		PolicyId: policyID,
	}), nil
}

// ---------------------------------------------------------------------------
// Posture
// ---------------------------------------------------------------------------

func (s *StudioService) ListPosture(
	ctx context.Context,
	req *connect.Request[studiov1.ListPostureRequest],
) (*connect.Response[studiov1.ListPostureResponse], error) {
	if s.posture == nil {
		return connect.NewResponse(&studiov1.ListPostureResponse{}), nil
	}
	rows, err := s.posture.ListPosture(ctx, tsToTime(req.Msg.Start), tsToTime(req.Msg.End))
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	out := make([]*studiov1.PostureRow, len(rows))
	for i, r := range rows {
		out[i] = &studiov1.PostureRow{
			PolicyId:       r.PolicyID,
			Title:          r.Title,
			Version:        r.Version,
			TotalRows:      r.TotalRows,
			PassedRows:     r.PassedRows,
			FailedRows:     r.FailedRows,
			OtherRows:      r.OtherRows,
			LatestAt:       r.LatestAt,
			TargetCount:    r.TargetCount,
			ControlCount:   r.ControlCount,
			LatestEvidence: r.LatestEvidence,
			Owner:          r.Owner,
		}
	}
	return connect.NewResponse(&studiov1.ListPostureResponse{Rows: out}), nil
}

// ---------------------------------------------------------------------------
// Catalogs
// ---------------------------------------------------------------------------

func (s *StudioService) ListCatalogs(
	ctx context.Context,
	_ *connect.Request[studiov1.ListCatalogsRequest],
) (*connect.Response[studiov1.ListCatalogsResponse], error) {
	if s.catalogs == nil {
		return connect.NewResponse(&studiov1.ListCatalogsResponse{}), nil
	}
	rows, err := s.catalogs.ListCatalogs(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	out := make([]*studiov1.Catalog, len(rows))
	for i, c := range rows {
		out[i] = &studiov1.Catalog{
			CatalogId:   c.CatalogID,
			CatalogType: c.CatalogType,
			Title:       c.Title,
			PolicyId:    c.PolicyID,
			ImportedAt:  timestamppb.New(c.ImportedAt),
		}
	}
	return connect.NewResponse(&studiov1.ListCatalogsResponse{Catalogs: out}), nil
}

// ---------------------------------------------------------------------------
// Mappings
// ---------------------------------------------------------------------------

func (s *StudioService) ListMappings(
	ctx context.Context,
	req *connect.Request[studiov1.ListMappingsRequest],
) (*connect.Response[studiov1.ListMappingsResponse], error) {
	if s.mappings == nil {
		return connect.NewResponse(&studiov1.ListMappingsResponse{}), nil
	}
	rows, err := s.mappings.ListMappings(ctx, req.Msg.SourceCatalogId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	out := make([]*studiov1.MappingDocument, len(rows))
	for i, m := range rows {
		out[i] = mappingToProto(m)
	}
	return connect.NewResponse(&studiov1.ListMappingsResponse{Mappings: out}), nil
}

// ---------------------------------------------------------------------------
// Audit Logs
// ---------------------------------------------------------------------------

func (s *StudioService) ListAuditLogs(
	ctx context.Context,
	req *connect.Request[studiov1.ListAuditLogsRequest],
) (*connect.Response[studiov1.ListAuditLogsResponse], error) {
	rows, err := s.auditLogs.ListAuditLogs(
		ctx,
		req.Msg.PolicyId,
		tsToTime(req.Msg.Start),
		tsToTime(req.Msg.End),
		int(req.Msg.Limit),
	)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	out := make([]*studiov1.AuditLog, len(rows))
	for i, a := range rows {
		out[i] = &studiov1.AuditLog{
			AuditId:       a.AuditID,
			PolicyId:      a.PolicyID,
			AuditStart:    timestamppb.New(a.AuditStart),
			AuditEnd:      timestamppb.New(a.AuditEnd),
			Framework:     a.Framework,
			CreatedAt:     timestamppb.New(a.CreatedAt),
			CreatedBy:     a.CreatedBy,
			Content:       a.Content,
			Summary:       a.Summary,
			Model:         a.Model,
			PromptVersion: a.PromptVersion,
		}
	}
	return connect.NewResponse(&studiov1.ListAuditLogsResponse{AuditLogs: out}), nil
}

// ---------------------------------------------------------------------------
// Draft Audit Logs
// ---------------------------------------------------------------------------

func (s *StudioService) CreateDraftAuditLog(
	ctx context.Context,
	req *connect.Request[studiov1.CreateDraftAuditLogRequest],
) (*connect.Response[studiov1.CreateDraftAuditLogResponse], error) {
	if s.drafts == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, fmt.Errorf("draft audit logs not available"))
	}
	d := store.DraftAuditLog{
		PolicyID:       req.Msg.PolicyId,
		Content:        req.Msg.Content,
		Summary:        req.Msg.Summary,
		AgentReasoning: req.Msg.AgentReasoning,
		Model:          req.Msg.Model,
		PromptVersion:  req.Msg.PromptVersion,
	}
	if err := s.drafts.InsertDraftAuditLog(ctx, d); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if s.pub != nil && d.DraftID != "" {
		s.pub.PublishDraftAuditLog(d.DraftID, d.PolicyID, d.Summary)
	}
	return connect.NewResponse(&studiov1.CreateDraftAuditLogResponse{
		DraftId: d.DraftID,
	}), nil
}

// ---------------------------------------------------------------------------
// Threats
// ---------------------------------------------------------------------------

func (s *StudioService) ListThreats(
	ctx context.Context,
	req *connect.Request[studiov1.ListThreatsRequest],
) (*connect.Response[studiov1.ListThreatsResponse], error) {
	if s.threats == nil {
		return connect.NewResponse(&studiov1.ListThreatsResponse{}), nil
	}
	rows, err := s.threats.QueryThreats(ctx, req.Msg.CatalogId, req.Msg.PolicyId, int(req.Msg.Limit))
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	out := make([]*studiov1.ThreatRow, len(rows))
	for i, t := range rows {
		out[i] = &studiov1.ThreatRow{
			CatalogId:   t.CatalogID,
			ThreatId:    t.ThreatID,
			Title:       t.Title,
			Description: t.Description,
			GroupId:     t.GroupID,
			PolicyId:    t.PolicyID,
		}
	}
	return connect.NewResponse(&studiov1.ListThreatsResponse{Threats: out}), nil
}

// ---------------------------------------------------------------------------
// Risks
// ---------------------------------------------------------------------------

func (s *StudioService) ListRisks(
	ctx context.Context,
	req *connect.Request[studiov1.ListRisksRequest],
) (*connect.Response[studiov1.ListRisksResponse], error) {
	if s.risks == nil {
		return connect.NewResponse(&studiov1.ListRisksResponse{}), nil
	}
	rows, err := s.risks.QueryRisks(ctx, req.Msg.CatalogId, req.Msg.PolicyId, int(req.Msg.Limit))
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	out := make([]*studiov1.RiskRow, len(rows))
	for i, r := range rows {
		out[i] = &studiov1.RiskRow{
			CatalogId:   r.CatalogID,
			RiskId:      r.RiskID,
			Title:       r.Title,
			Description: r.Description,
			Severity:    r.Severity,
			GroupId:     r.GroupID,
			Impact:      r.Impact,
			PolicyId:    r.PolicyID,
		}
	}
	return connect.NewResponse(&studiov1.ListRisksResponse{Risks: out}), nil
}

// ---------------------------------------------------------------------------
// Conversion helpers
// ---------------------------------------------------------------------------

func policyToProto(p store.Policy) *studiov1.Policy {
	return &studiov1.Policy{
		PolicyId:     p.PolicyID,
		Title:        p.Title,
		Version:      p.Version,
		OciReference: p.OCIReference,
		Content:      p.Content,
		ImportedAt:   timestamppb.New(p.ImportedAt),
		ImportedBy:   p.ImportedBy,
	}
}

func mappingToProto(m store.MappingDocument) *studiov1.MappingDocument {
	return &studiov1.MappingDocument{
		MappingId:       m.MappingID,
		SourceCatalogId: m.SourceCatalogID,
		TargetCatalogId: m.TargetCatalogID,
		Framework:       m.Framework,
		Content:         m.Content,
		ImportedAt:      timestamppb.New(m.ImportedAt),
	}
}

func evidenceToProto(r store.EvidenceRecord) *studiov1.EvidenceRecord {
	return &studiov1.EvidenceRecord{
		EvidenceId:           r.EvidenceID,
		PolicyId:             r.PolicyID,
		TargetId:             r.TargetID,
		TargetName:           r.TargetName,
		TargetType:           r.TargetType,
		TargetEnv:            r.TargetEnv,
		EngineName:           r.EngineName,
		EngineVersion:        r.EngineVersion,
		RuleId:               r.RuleID,
		RuleName:             r.RuleName,
		RuleUri:              r.RuleURI,
		EvalResult:           r.EvalResult,
		EvalMessage:          r.EvalMessage,
		ControlId:            r.ControlID,
		ControlCatalogId:     r.ControlCatalogID,
		ControlCategory:      r.ControlCategory,
		ControlApplicability: r.ControlApplicability,
		RequirementId:        r.RequirementID,
		PlanId:               r.PlanID,
		Confidence:           r.Confidence,
		StepsExecuted:        int32(r.StepsExecuted),
		ComplianceStatus:     r.ComplianceStatus,
		RiskLevel:            r.RiskLevel,
		Frameworks:           r.Frameworks,
		Requirements:         r.Requirements,
		RemediationAction:    r.RemediationAction,
		RemediationStatus:    r.RemediationStatus,
		RemediationDesc:      r.RemediationDesc,
		ExceptionId:          r.ExceptionID,
		ExceptionActive:      r.ExceptionActive,
		EnrichmentStatus:     r.EnrichmentStatus,
		AttestationRef:       r.AttestationRef,
		SourceRegistry:       r.SourceRegistry,
		BlobRef:              r.BlobRef,
		Certified:            r.Certified,
		Owner:                r.Owner,
		CollectedAt:          timestamppb.New(r.CollectedAt),
		Classification:       r.Classification,
	}
}

func tsToTime(ts *timestamppb.Timestamp) time.Time {
	if ts == nil {
		return time.Time{}
	}
	return ts.AsTime()
}

