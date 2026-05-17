// SPDX-License-Identifier: Apache-2.0

package grpcapi

import (
	"context"
	"testing"
	"time"

	pb "github.com/complytime-labs/complytime-core/gen/complytime/v1"
	"github.com/complytime-labs/complytime-core/internal/gemara"
	"github.com/complytime-labs/complytime-core/internal/store"
)

type mockPolicyStore struct {
	policies []store.Policy
}

func (m *mockPolicyStore) InsertPolicy(_ context.Context, _ store.Policy) error { return nil }
func (m *mockPolicyStore) ListPolicies(_ context.Context) ([]store.Policy, error) {
	return m.policies, nil
}
func (m *mockPolicyStore) GetPolicy(_ context.Context, id string) (*store.Policy, error) {
	for _, p := range m.policies {
		if p.PolicyID == id {
			return &p, nil
		}
	}
	return nil, context.DeadlineExceeded
}

type mockMappingStore struct{}

func (m *mockMappingStore) InsertMapping(_ context.Context, _ store.MappingDocument) error {
	return nil
}
func (m *mockMappingStore) ListMappings(_ context.Context, _ string) ([]store.MappingDocument, error) {
	return nil, nil
}
func (m *mockMappingStore) ListAllMappings(_ context.Context) ([]store.MappingDocument, error) {
	return nil, nil
}
func (m *mockMappingStore) QueryMappings(_ context.Context, _, _ string, _ int) ([]gemara.MappingEntry, error) {
	return nil, nil
}
func (m *mockMappingStore) InsertMappingEntries(_ context.Context, _ []gemara.MappingEntry) error {
	return nil
}
func (m *mockMappingStore) DeleteMappingEntries(_ context.Context, _, _ string) error {
	return nil
}
func (m *mockMappingStore) CountMappingEntries(_ context.Context, _ string) (int, error) {
	return 0, nil
}

func TestListPolicies(t *testing.T) {
	now := time.Now()
	ps := &mockPolicyStore{
		policies: []store.Policy{
			{PolicyID: "pol-1", Title: "Test Policy", Version: "1.0", ImportedAt: now},
		},
	}
	srv := NewPolicyServer(ps, &mockMappingStore{})

	resp, err := srv.ListPolicies(context.Background(), &pb.ListPoliciesRequest{})
	if err != nil {
		t.Fatalf("ListPolicies returned error: %v", err)
	}
	if len(resp.GetPolicies()) != 1 {
		t.Fatalf("expected 1 policy, got %d", len(resp.GetPolicies()))
	}
	if resp.GetPolicies()[0].GetId() != "pol-1" {
		t.Errorf("expected policy id pol-1, got %s", resp.GetPolicies()[0].GetId())
	}
}

func TestGetPolicy(t *testing.T) {
	ps := &mockPolicyStore{
		policies: []store.Policy{
			{PolicyID: "pol-1", Title: "Test Policy"},
		},
	}
	srv := NewPolicyServer(ps, &mockMappingStore{})

	resp, err := srv.GetPolicy(context.Background(), &pb.GetPolicyRequest{PolicyId: "pol-1"})
	if err != nil {
		t.Fatalf("GetPolicy returned error: %v", err)
	}
	if resp.GetPolicy().GetTitle() != "Test Policy" {
		t.Errorf("expected title 'Test Policy', got %s", resp.GetPolicy().GetTitle())
	}
}

func TestGetPolicyMissingID(t *testing.T) {
	srv := NewPolicyServer(&mockPolicyStore{}, &mockMappingStore{})
	_, err := srv.GetPolicy(context.Background(), &pb.GetPolicyRequest{})
	if err == nil {
		t.Fatal("expected error for missing policy_id")
	}
}

func TestGetPolicyNotFound(t *testing.T) {
	srv := NewPolicyServer(&mockPolicyStore{}, &mockMappingStore{})
	_, err := srv.GetPolicy(context.Background(), &pb.GetPolicyRequest{PolicyId: "nonexistent"})
	if err == nil {
		t.Fatal("expected error for nonexistent policy")
	}
}
