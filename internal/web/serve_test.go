// SPDX-License-Identifier: Apache-2.0

package web

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"

	"github.com/labstack/echo/v4"
)

func testAssets() fstest.MapFS {
	return fstest.MapFS{
		"dist/index.html":     {Data: []byte("<html>SPA</html>")},
		"dist/assets/main.js": {Data: []byte("console.log('ok')")},
	}
}

func TestRegisterSPA_ServesIndexAtRoot(t *testing.T) {
	e := echo.New()
	RegisterSPA(e, testAssets())

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if body := rec.Body.String(); body != "<html>SPA</html>" {
		t.Fatalf("expected SPA index, got %q", body)
	}
}

func TestRegisterSPA_ServesStaticAsset(t *testing.T) {
	e := echo.New()
	RegisterSPA(e, testAssets())

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/assets/main.js", nil)
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if body := rec.Body.String(); body != "console.log('ok')" {
		t.Fatalf("expected JS content, got %q", body)
	}
}

func TestRegisterSPA_FallsBackToIndexForUnknownPath(t *testing.T) {
	e := echo.New()
	RegisterSPA(e, testAssets())

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/dashboard/overview", nil)
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if body := rec.Body.String(); body != "<html>SPA</html>" {
		t.Fatalf("expected SPA index for history-mode route, got %q", body)
	}
}

func TestRegisterSPA_API404NotCaptured(t *testing.T) {
	e := echo.New()
	RegisterSPA(e, testAssets())

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/nonexistent", nil)
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for unmatched /api/ route, got %d", rec.Code)
	}
}
