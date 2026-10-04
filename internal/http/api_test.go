package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"streamingplatform/internal/domain"
)

type profileApplicationStub struct {
	create func(context.Context, string, string, int, string) (*domain.Profile, error)
	get    func(context.Context, string) (*domain.Profile, error)
}

func (s profileApplicationStub) Create(ctx context.Context, username, email string, age int, country string) (*domain.Profile, error) {
	return s.create(ctx, username, email, age, country)
}

func (s profileApplicationStub) Get(ctx context.Context, id string) (*domain.Profile, error) {
	return s.get(ctx, id)
}

func testAPI(profiles ProfileApplication) http.Handler {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	return NewAPI(Dependencies{Profiles: profiles}, logger).Handler()
}

func TestCreateUserSuccess(t *testing.T) {
	handler := testAPI(profileApplicationStub{
		create: func(_ context.Context, username, email string, age int, country string) (*domain.Profile, error) {
			return &domain.Profile{ID: "user-1", Username: username, Email: email, Age: age, Country: country}, nil
		},
		get: func(context.Context, string) (*domain.Profile, error) { return nil, domain.ErrUserNotFound },
	})

	request := httptest.NewRequest(http.MethodPost, "/users", strings.NewReader(`{"username":"ada","email":"ada@example.com","age":30,"country":"UK"}`))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusCreated {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	var profile domain.Profile
	if err := json.NewDecoder(response.Body).Decode(&profile); err != nil || profile.ID != "user-1" {
		t.Fatalf("decode response: %v profile=%+v", err, profile)
	}
}

func TestCreateUserValidationError(t *testing.T) {
	handler := testAPI(profileApplicationStub{
		create: func(context.Context, string, string, int, string) (*domain.Profile, error) {
			return nil, &domain.ValidationError{Field: "age", Value: 5, Message: "must be at least 13"}
		},
		get: func(context.Context, string) (*domain.Profile, error) { return nil, domain.ErrUserNotFound },
	})
	request := httptest.NewRequest(http.MethodPost, "/users", bytes.NewBufferString(`{"username":"kid","email":"kid@example.com","age":5,"country":"US"}`))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
}

func TestGetUserNotFound(t *testing.T) {
	handler := testAPI(profileApplicationStub{
		create: func(context.Context, string, string, int, string) (*domain.Profile, error) { return nil, nil },
		get:    func(context.Context, string) (*domain.Profile, error) { return nil, domain.ErrUserNotFound },
	})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/users/missing", nil))
	if response.Code != http.StatusNotFound {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
}

func TestRecoveryMiddlewareReturnsInternalServerError(t *testing.T) {
	handler := testAPI(profileApplicationStub{
		create: func(context.Context, string, string, int, string) (*domain.Profile, error) { return nil, nil },
		get: func(context.Context, string) (*domain.Profile, error) {
			panic("database driver invariant")
		},
	})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/users/user-1", nil))
	if response.Code != http.StatusInternalServerError {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
}

func TestHealth(t *testing.T) {
	handler := testAPI(nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/health", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("status=%d", response.Code)
	}
}
