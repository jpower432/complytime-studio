// SPDX-License-Identifier: Apache-2.0

package store

import (
	"context"
	"testing"

	gemara "github.com/complytime/complytime-studio/internal/gemara"
)

type fakeControlStore struct {
	controls    []gemara.ControlRow
	assessments []gemara.AssessmentRequirementRow
}

func (f *fakeControlStore) InsertControls(_ context.Context, rows []gemara.ControlRow) error {
	f.controls = append(f.controls, rows...)
	return nil
}

func (f *fakeControlStore) InsertAssessmentRequirements(_ context.Context, rows []gemara.AssessmentRequirementRow) error {
	f.assessments = append(f.assessments, rows...)
	return nil
}

func (f *fakeControlStore) InsertControlThreats(_ context.Context, _ []gemara.ControlThreatRow) error {
	return nil
}

func (f *fakeControlStore) CountControls(_ context.Context, _ string) (int, error) {
	return len(f.controls), nil
}

func TestExtractPolicyCriteria_ParsesControlsAndARs(t *testing.T) {
	content := `
criteria:
  - id: AC-1
    title: Access Control Policy
    description: Establish access control policy
    assessment-requirements:
      - id: AC-1.a
        description: Verify policy exists
      - id: AC-1.b
        description: Review annually
  - id: AC-2
    title: Account Management
    description: Manage system accounts
`
	fake := &fakeControlStore{}
	count, err := ExtractPolicyCriteria(context.Background(), "test-policy", content, fake)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 2 {
		t.Errorf("expected 2 controls, got %d", count)
	}
	if len(fake.controls) != 2 {
		t.Fatalf("expected 2 inserted controls, got %d", len(fake.controls))
	}
	if fake.controls[0].ControlID != "AC-1" || fake.controls[0].Title != "Access Control Policy" {
		t.Errorf("unexpected first control: %+v", fake.controls[0])
	}
	if fake.controls[0].PolicyID != "test-policy" {
		t.Errorf("expected policy_id 'test-policy', got %q", fake.controls[0].PolicyID)
	}
	if len(fake.assessments) != 2 {
		t.Errorf("expected 2 assessment requirements, got %d", len(fake.assessments))
	}
	if fake.assessments[0].RequirementID != "AC-1.a" {
		t.Errorf("expected AR id 'AC-1.a', got %q", fake.assessments[0].RequirementID)
	}
}

func TestExtractPolicyCriteria_EmptyCriteria(t *testing.T) {
	content := `criteria: []`
	fake := &fakeControlStore{}
	count, err := ExtractPolicyCriteria(context.Background(), "test-policy", content, fake)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 controls, got %d", count)
	}
}

func TestExtractPolicyCriteria_NoCriteriaKey(t *testing.T) {
	content := `title: A Policy`
	fake := &fakeControlStore{}
	count, err := ExtractPolicyCriteria(context.Background(), "test-policy", content, fake)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 controls for missing criteria, got %d", count)
	}
}

func TestExtractPolicyCriteria_InvalidYAML(t *testing.T) {
	fake := &fakeControlStore{}
	_, err := ExtractPolicyCriteria(context.Background(), "test", "not: [valid", fake)
	if err == nil {
		t.Fatal("expected error for invalid YAML")
	}
}

func TestExtractPolicyCriteria_CatalogIDMatchesPolicyID(t *testing.T) {
	content := `
criteria:
  - id: SC-1
    title: System Communications
    description: Protect communications
`
	fake := &fakeControlStore{}
	_, err := ExtractPolicyCriteria(context.Background(), "my-policy", content, fake)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fake.controls[0].CatalogID != "my-policy" {
		t.Errorf("expected catalog_id to match policy_id 'my-policy', got %q", fake.controls[0].CatalogID)
	}
}
