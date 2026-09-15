package provisioner

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"
)

const (
	StateProvisioning = "PROVISIONING"
	StateReady        = "READY"
	StateFailed       = "FAILED"
)

var (
	ErrNotFound    = errors.New("database not found")
	ErrUnavailable = errors.New("provisioning API is unavailable")
	ErrBadRequest  = errors.New("provisioning API rejected the request")
)

type Database struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Engine   string `json:"engine"`
	SizeGB   int    `json:"sizeGB"`
	State    string `json:"state"`
	Endpoint string `json:"endpoint,omitempty"`
}

type CreateRequest struct {
	Name   string `json:"name"`
	Engine string `json:"engine"`
	SizeGB int    `json:"sizeGB"`
}

type Client struct {
	baseURL    string
	httpClient *http.Client
}

func NewClient(baseURL string) *Client {
	return &Client{
		baseURL:    strings.TrimSuffix(baseURL, "/"),
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

func (c *Client) CreateDatabase(ctx context.Context, request CreateRequest) (*Database, error) {
	body, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}

	var database Database
	if err := c.do(ctx, http.MethodPost, "/databases", body, http.StatusCreated, &database); err != nil {
		return nil, err
	}
	return &database, nil
}

func (c *Client) GetDatabase(ctx context.Context, id string) (*Database, error) {
	var database Database
	if err := c.do(ctx, http.MethodGet, "/databases/"+id, nil, http.StatusOK, &database); err != nil {
		return nil, err
	}
	return &database, nil
}

func (c *Client) DeleteDatabase(ctx context.Context, id string) error {
	return c.do(ctx, http.MethodDelete, "/databases/"+id, nil, http.StatusNoContent, nil)
}

func (c *Client) do(ctx context.Context, method, path string, body []byte, expectedStatus int, result any) error {
	var bodyReader io.Reader
	if body != nil {
		bodyReader = bytes.NewReader(body)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, bodyReader)
	if err != nil {
		return err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		var opErr *net.OpError
		if errors.As(err, &opErr) && opErr.Op == "dial" {
			return fmt.Errorf("%w: %v", ErrUnavailable, err)
		}
		return fmt.Errorf("%s %s: %w", method, path, err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case expectedStatus:
	case http.StatusNotFound:
		return ErrNotFound
	case http.StatusServiceUnavailable:
		return ErrUnavailable
	case http.StatusBadRequest:
		return fmt.Errorf("%w: %s %s", ErrBadRequest, method, path)
	default:
		return fmt.Errorf("%s %s: unexpected status %d", method, path, resp.StatusCode)
	}

	if result == nil {
		return nil
	}
	if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
		return fmt.Errorf("%s %s: failed to read response: %w", method, path, err)
	}
	return nil
}
