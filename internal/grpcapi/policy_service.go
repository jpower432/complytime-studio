// SPDX-License-Identifier: Apache-2.0

package grpcapi

import (
	"context"
	"log/slog"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	pb "github.com/complytime-labs/complytime-core/gen/complytime/v1"
	"github.com/complytime-labs/complytime-core/internal/store"
)

// PolicyServer implements the PolicyService gRPC interface using the
// same store layer as the Echo REST handlers.
type PolicyServer struct {
	pb.UnimplementedPolicyServiceServer
	policies store.PolicyStore
	mappings store.MappingStore
}

func NewPolicyServer(ps store.PolicyStore, ms store.MappingStore) *PolicyServer {
	return &PolicyServer{policies: ps, mappings: ms}
}

func (s *PolicyServer) ListPolicies(ctx context.Context, _ *pb.ListPoliciesRequest) (*pb.ListPoliciesResponse, error) {
	rows, err := s.policies.ListPolicies(ctx)
	if err != nil {
		slog.Error("grpc ListPolicies failed", "error", err)
		return nil, status.Error(codes.Internal, "internal error")
	}
	out := make([]*pb.Policy, len(rows))
	for i, p := range rows {
		out[i] = storeToProto(&p)
	}
	return &pb.ListPoliciesResponse{Policies: out}, nil
}

func (s *PolicyServer) GetPolicy(ctx context.Context, req *pb.GetPolicyRequest) (*pb.GetPolicyResponse, error) {
	if req.GetPolicyId() == "" {
		return nil, status.Error(codes.InvalidArgument, "policy_id is required")
	}
	p, err := s.policies.GetPolicy(ctx, req.GetPolicyId())
	if err != nil {
		slog.Error("grpc GetPolicy failed", "error", err, "id", req.GetPolicyId())
		return nil, status.Error(codes.NotFound, "not found")
	}
	return &pb.GetPolicyResponse{Policy: storeToProto(p)}, nil
}

func storeToProto(p *store.Policy) *pb.Policy {
	return &pb.Policy{
		Id:         p.PolicyID,
		Title:      p.Title,
		Version:    p.Version,
		ImportedBy: p.ImportedBy,
		ImportedAt: timestamppb.New(p.ImportedAt),
		Content:    []byte(p.Content),
	}
}
