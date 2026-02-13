package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	app "gophkeeper/internal/app/client"

	"github.com/google/uuid"
)

// APIClient talks to the GophKeeper server.
type APIClient struct {
	BaseURL string
	Client  *http.Client
}

// NewAPIClient creates an API client.
func NewAPIClient(baseURL string) *APIClient {
	return &APIClient{
		BaseURL: baseURL,
		Client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// Register registers a new user.
func (c *APIClient) Register(ctx context.Context, username, password string) (string, error) {
	return c.postAuth(ctx, "/register", username, password)
}

// Login logs in an existing user.
func (c *APIClient) Login(ctx context.Context, username, password string) (string, error) {
	return c.postAuth(ctx, "/login", username, password)
}

// UpsertItem uploads an encrypted item.
func (c *APIClient) UpsertItem(ctx context.Context, token string, req app.ItemUpsertRequest) (app.ItemResponse, error) {
	var resp app.ItemResponse
	if err := c.doJSON(ctx, "POST", "/items", token, req, &resp); err != nil {
		return app.ItemResponse{}, err
	}
	return resp, nil
}

// GetItem fetches an encrypted item.
func (c *APIClient) GetItem(ctx context.Context, token string, id uuid.UUID) (app.ItemResponse, error) {
	var resp app.ItemResponse
	if err := c.doJSON(ctx, "GET", fmt.Sprintf("/items/%s", id.String()), token, nil, &resp); err != nil {
		return app.ItemResponse{}, err
	}
	return resp, nil
}

// SyncItems pulls items updated since version.
func (c *APIClient) SyncItems(ctx context.Context, token string, since int64) ([]app.ItemResponse, error) {
	var resp []app.ItemResponse
	path := fmt.Sprintf("/items?since=%d", since)
	if err := c.doJSON(ctx, "GET", path, token, nil, &resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// DeleteItem deletes an encrypted item.
func (c *APIClient) DeleteItem(ctx context.Context, token string, id uuid.UUID, version int64) (app.ItemResponse, error) {
	var resp app.ItemResponse
	path := fmt.Sprintf("/items/%s?version=%d", id.String(), version)
	if err := c.doJSON(ctx, "DELETE", path, token, nil, &resp); err != nil {
		return app.ItemResponse{}, err
	}
	return resp, nil
}

func (c *APIClient) postAuth(ctx context.Context, path, username, password string) (string, error) {
	var resp struct {
		Token string `json:"token"`
	}
	if err := c.doJSON(ctx, "POST", path, "", map[string]string{"username": username, "password": password}, &resp); err != nil {
		return "", err
	}
	return resp.Token, nil
}

func (c *APIClient) doJSON(ctx context.Context, method, path, token string, body, out any) error {
	var buf io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return err
		}
		buf = bytes.NewBuffer(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.BaseURL+path, buf)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	res, err := c.Client.Do(req)
	if err != nil {
		return err
	}
	defer func() {
		_ = res.Body.Close()
	}()
	if res.StatusCode >= 400 {
		var msg struct {
			Error string `json:"error"`
		}
		_ = json.NewDecoder(res.Body).Decode(&msg)
		if msg.Error == "" {
			msg.Error = res.Status
		}
		return fmt.Errorf("server error: %s", msg.Error)
	}
	if out == nil {
		return nil
	}
	return json.NewDecoder(res.Body).Decode(out)
}
