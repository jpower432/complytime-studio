// SPDX-License-Identifier: Apache-2.0

package proxy

import (
	"errors"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestBuildValidateResponse_JSON(t *testing.T) {
	res := &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: `{"valid":true,"errors":[]}`},
		},
	}
	out, err := buildValidateResponse(res)
	if err != nil {
		t.Fatal(err)
	}
	if !out.Valid || len(out.Errors) != 0 {
		t.Fatalf("got %+v", out)
	}
}

func TestBuildValidateResponse_InfersValid(t *testing.T) {
	res := &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: `{"errors":[]}`},
		},
	}
	out, err := buildValidateResponse(res)
	if err != nil {
		t.Fatal(err)
	}
	if !out.Valid {
		t.Fatalf("expected valid true, got %+v", out)
	}
}

func TestBuildValidateResponse_IsError(t *testing.T) {
	res := &mcp.CallToolResult{
		IsError: true,
		Content: []mcp.Content{
			&mcp.TextContent{Text: `not-json`},
		},
	}
	out, err := buildValidateResponse(res)
	if err != nil {
		t.Fatal(err)
	}
	if out.Valid || len(out.Errors) != 1 {
		t.Fatalf("got %+v", out)
	}
}

func TestExtractMigratedYAML(t *testing.T) {
	res := &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: `{"yaml":"hello: world\n"}`},
		},
	}
	got, err := extractMigratedYAML(res)
	if err != nil {
		t.Fatal(err)
	}
	if got != "hello: world\n" {
		t.Fatalf("got %q", got)
	}
}

func TestIsMCPUnavailable(t *testing.T) {
	if !isMCPUnavailable(mcp.ErrConnectionClosed) {
		t.Fatal("expected connection closed as unavailable")
	}
	if !isMCPUnavailable(errors.New("broken pipe on write")) {
		t.Fatal("expected broken pipe as unavailable")
	}
	if isMCPUnavailable(errors.New("invalid tool arguments")) {
		t.Fatal("did not expect generic error as unavailable")
	}
}
