package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestHealthHandlerReturnsOKWhenDependenciesAreHealthy(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.GET("/health", newHealthHandler(
		func(ctx context.Context) error { return nil },
		func(ctx context.Context) error { return nil },
	))

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", w.Code, http.StatusOK, w.Body.String())
	}

	var response healthResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if response.Status != "ok" {
		t.Fatalf("status = %q, want ok", response.Status)
	}
	if response.Checks["mysql"] != "ok" {
		t.Fatalf("mysql check = %q, want ok", response.Checks["mysql"])
	}
	if response.Checks["redis"] != "ok" {
		t.Fatalf("redis check = %q, want ok", response.Checks["redis"])
	}
	if len(response.Errors) != 0 {
		t.Fatalf("errors = %#v, want empty", response.Errors)
	}
}

func TestHealthHandlerReturnsServiceUnavailableWhenMySQLFails(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.GET("/health", newHealthHandler(
		func(ctx context.Context) error { return errors.New("mysql unavailable") },
		func(ctx context.Context) error { return nil },
	))

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d; body=%s", w.Code, http.StatusServiceUnavailable, w.Body.String())
	}

	var response healthResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if response.Status != "degraded" {
		t.Fatalf("status = %q, want degraded", response.Status)
	}
	if response.Checks["mysql"] != "error" {
		t.Fatalf("mysql check = %q, want error", response.Checks["mysql"])
	}
	if response.Checks["redis"] != "ok" {
		t.Fatalf("redis check = %q, want ok", response.Checks["redis"])
	}
	if response.Errors["mysql"] != "mysql unavailable" {
		t.Fatalf("mysql error = %q, want mysql unavailable", response.Errors["mysql"])
	}
}

func TestHealthHandlerReturnsServiceUnavailableWhenRedisFails(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.GET("/health", newHealthHandler(
		func(ctx context.Context) error { return nil },
		func(ctx context.Context) error { return errors.New("redis unavailable") },
	))

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d; body=%s", w.Code, http.StatusServiceUnavailable, w.Body.String())
	}

	var response healthResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if response.Status != "degraded" {
		t.Fatalf("status = %q, want degraded", response.Status)
	}
	if response.Checks["mysql"] != "ok" {
		t.Fatalf("mysql check = %q, want ok", response.Checks["mysql"])
	}
	if response.Checks["redis"] != "error" {
		t.Fatalf("redis check = %q, want error", response.Checks["redis"])
	}
	if response.Errors["redis"] != "redis unavailable" {
		t.Fatalf("redis error = %q, want redis unavailable", response.Errors["redis"])
	}
}
