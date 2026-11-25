// Package apiclient is a typed Go client for the shop REST API
// (docs/API_CONTRACT.md). It moves only Base64 ciphertexts and public
// artifacts; encryption/decryption is done by the caller (pkg/ckks).
// It is used by the end-to-end tests and the benchmark.
package apiclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client talks to the server.
type Client struct {
	BaseURL string
	HTTP    *http.Client
	// OnExchange, when set, receives every raw request/response pair
	// (used to save sample JSON for the report).
	OnExchange func(method, path string, reqBody []byte, status int, respBody []byte)
}

// New returns a client for baseURL.
func New(baseURL string) *Client {
	return &Client{BaseURL: baseURL, HTTP: &http.Client{Timeout: 120 * time.Second}}
}

// Error is a non-2xx response.
type Error struct {
	Status int
	Code   string `json:"error"`
	Detail string `json:"detail"`
}

func (e *Error) Error() string { return fmt.Sprintf("HTTP %d %s: %s", e.Status, e.Code, e.Detail) }

// Do sends a raw request body (may be nil) and decodes a 2xx JSON response
// into out. It returns *Error for non-2xx responses.
func (c *Client) Do(ctx context.Context, method, path string, body []byte, contentType string, out any) (int, error) {
	var rd io.Reader
	if body != nil {
		rd = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.BaseURL+path, rd)
	if err != nil {
		return 0, err
	}
	if body != nil {
		if contentType == "" {
			contentType = "application/json"
		}
		req.Header.Set("Content-Type", contentType)
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return 0, err
	}
	defer func() { _ = resp.Body.Close() }()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp.StatusCode, err
	}
	if c.OnExchange != nil {
		c.OnExchange(method, path, body, resp.StatusCode, data)
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		e := &Error{Status: resp.StatusCode}
		_ = json.Unmarshal(data, e)
		return resp.StatusCode, e
	}
	if out != nil {
		if err := json.Unmarshal(data, out); err != nil {
			return resp.StatusCode, fmt.Errorf("decode response: %w", err)
		}
	}
	return resp.StatusCode, nil
}

func (c *Client) doJSON(ctx context.Context, method, path string, in, out any) (int, error) {
	var body []byte
	if in != nil {
		var err error
		if body, err = json.Marshal(in); err != nil {
			return 0, err
		}
	}
	return c.Do(ctx, method, path, body, "", out)
}

// ActivateRequest is the body of POST /accounts.
type ActivateRequest struct {
	UserID     string `json:"user_id"`
	Params     string `json:"params"`
	PublicKey  string `json:"public_key"`
	RelinKey   string `json:"relin_key"`
	EncBalance string `json:"enc_balance"`
}

// ActivateResponse is the 201 body of POST /accounts.
type ActivateResponse struct {
	Status     string `json:"status"`
	UserID     string `json:"user_id"`
	EncBalance string `json:"enc_balance"`
}

// BalanceResponse is returned by CheckBalance and Topup.
type BalanceResponse struct {
	UserID     string `json:"user_id"`
	EncBalance string `json:"enc_balance"`
}

// TransactRequest is the body of POST /accounts/{id}/transactions.
type TransactRequest struct {
	ProductID    string  `json:"product_id"`
	DiscountCode *string `json:"discount_code,omitempty"`
	EncQuantity  string  `json:"enc_quantity"`
	EncMetadata  string  `json:"enc_metadata"`
}

// TransactResponse is the 201 body of a transaction.
type TransactResponse struct {
	TransactionID string `json:"transaction_id"`
	EncTotal      string `json:"enc_total"`
	EncBalance    string `json:"enc_balance"`
}

// HistoryItem is one GetTransact entry.
type HistoryItem struct {
	TransactionID string `json:"transaction_id"`
	EncMetadata   string `json:"enc_metadata"`
	EncTotal      string `json:"enc_total"`
	CreatedAt     string `json:"created_at"`
}

// HistoryResponse is the body of GET /accounts/{id}/transactions.
type HistoryResponse struct {
	UserID       string        `json:"user_id"`
	Transactions []HistoryItem `json:"transactions"`
}

// Product and Discount mirror data/catalog.json.
type Product struct {
	ProductID string  `json:"product_id"`
	NumericID int     `json:"numeric_id"`
	Name      string  `json:"name"`
	BasePrice float64 `json:"base_price"`
}

// Discount is a discount rule.
type Discount struct {
	Code      string  `json:"code"`
	NumericID int     `json:"numeric_id"`
	Percent   float64 `json:"percent"`
	Fixed     float64 `json:"fixed"`
}

// Catalog is the body of GET /catalog.
type Catalog struct {
	Products  []Product  `json:"products"`
	Discounts []Discount `json:"discounts"`
}

// Health calls GET /healthz.
func (c *Client) Health(ctx context.Context) error {
	_, err := c.doJSON(ctx, http.MethodGet, "/healthz", nil, nil)
	return err
}

// Catalog calls GET /catalog.
func (c *Client) Catalog(ctx context.Context) (Catalog, error) {
	var out Catalog
	_, err := c.doJSON(ctx, http.MethodGet, "/catalog", nil, &out)
	return out, err
}

// Activate calls POST /accounts.
func (c *Client) Activate(ctx context.Context, req ActivateRequest) (ActivateResponse, error) {
	var out ActivateResponse
	_, err := c.doJSON(ctx, http.MethodPost, "/accounts", req, &out)
	return out, err
}

// Balance calls GET /accounts/{id}/balance.
func (c *Client) Balance(ctx context.Context, userID string) (BalanceResponse, error) {
	var out BalanceResponse
	_, err := c.doJSON(ctx, http.MethodGet, "/accounts/"+userID+"/balance", nil, &out)
	return out, err
}

// Topup calls POST /accounts/{id}/topup.
func (c *Client) Topup(ctx context.Context, userID, encAmount string) (BalanceResponse, error) {
	var out BalanceResponse
	_, err := c.doJSON(ctx, http.MethodPost, "/accounts/"+userID+"/topup", map[string]string{"enc_amount": encAmount}, &out)
	return out, err
}

// Transact calls POST /accounts/{id}/transactions.
func (c *Client) Transact(ctx context.Context, userID string, req TransactRequest) (TransactResponse, error) {
	var out TransactResponse
	_, err := c.doJSON(ctx, http.MethodPost, "/accounts/"+userID+"/transactions", req, &out)
	return out, err
}

// History calls GET /accounts/{id}/transactions.
func (c *Client) History(ctx context.Context, userID string) (HistoryResponse, error) {
	var out HistoryResponse
	_, err := c.doJSON(ctx, http.MethodGet, "/accounts/"+userID+"/transactions", nil, &out)
	return out, err
}
