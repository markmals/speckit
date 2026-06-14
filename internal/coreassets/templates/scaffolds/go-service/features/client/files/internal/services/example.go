package services

import (
	"context"
	"net/http"
	"net/url"
)

// Client is an example upstream JSON-API client. Construct it with New, then call
// its methods from your handlers. It owns an http.Client bounded by
// DefaultTimeout and sends a bearer token. Model your real clients on this:
// project only the fields you consume, and route every call through GetJSON.
type Client struct {
	HTTPClient *http.Client
	BaseURL    string
	APIKey     string
}

// New returns a Client for an upstream at baseURL authenticated with apiKey.
func New(baseURL, apiKey string) *Client {
	return &Client{
		HTTPClient: &http.Client{Timeout: DefaultTimeout},
		BaseURL:    baseURL,
		APIKey:     apiKey,
	}
}

// Thing projects the subset of an upstream resource this service consumes — keep
// these structs to the fields you actually use, not the upstream's full surface.
type Thing struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// GetThing fetches GET {BaseURL}/things/{id} from the upstream API.
func (c *Client) GetThing(ctx context.Context, id string) (*Thing, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL+"/things/"+url.PathEscape(id), nil)
	if err != nil {
		return nil, err
	}
	if c.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.APIKey)
	}
	var t Thing
	if err := GetJSON(c.HTTPClient, req, &t); err != nil {
		return nil, err
	}
	return &t, nil
}
