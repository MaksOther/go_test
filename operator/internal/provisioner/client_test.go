package provisioner

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCreateDatabaseErrors(t *testing.T) {
	tests := []struct {
		name            string
		statusCode      int
		wantUnavailable bool
		wantBadRequest  bool
	}{
		{name: "503 is safe to retry", statusCode: http.StatusServiceUnavailable, wantUnavailable: true},
		{name: "400 is rejected", statusCode: http.StatusBadRequest, wantBadRequest: true},
		{name: "500 may have created the database", statusCode: http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
			}))
			defer server.Close()

			_, err := NewClient(server.URL).CreateDatabase(context.Background(), CreateRequest{Name: "orders", Engine: "postgres", SizeGB: 20})
			if err == nil {
				t.Fatal("expected an error")
			}
			if got := errors.Is(err, ErrUnavailable); got != tt.wantUnavailable {
				t.Errorf("errors.Is(err, ErrUnavailable) = %v, want %v", got, tt.wantUnavailable)
			}
			if got := errors.Is(err, ErrBadRequest); got != tt.wantBadRequest {
				t.Errorf("errors.Is(err, ErrBadRequest) = %v, want %v", got, tt.wantBadRequest)
			}
		})
	}
}

func TestCreateDatabaseWhenServerIsDown(t *testing.T) {
	server := httptest.NewServer(http.NotFoundHandler())
	serverURL := server.URL
	server.Close()

	_, err := NewClient(serverURL).CreateDatabase(context.Background(), CreateRequest{Name: "orders", Engine: "postgres", SizeGB: 20})
	if !errors.Is(err, ErrUnavailable) {
		t.Errorf("expected ErrUnavailable, got %v", err)
	}
}
