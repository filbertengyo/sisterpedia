// Package handlers exposes the shop service as the REST API described in
// docs/API_CONTRACT.md. Request and response bodies are never logged.
package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"mime"
	"net/http"
	"strings"
	"time"

	"github.com/filbertengyo/sisterpedia/internal/server/service"
	"github.com/go-chi/chi/v5"
)

// Options configures the HTTP layer.
type Options struct {
	AllowedOrigins []string
	MaxBodyBytes   int64
	Logger         *log.Logger // nil = log.Default()
}

type api struct {
	svc  *service.Service
	opts Options
	log  *log.Logger
}

// New returns the HTTP handler of the server.
func New(svc *service.Service, opts Options) http.Handler {
	a := &api{svc: svc, opts: opts, log: opts.Logger}
	if a.log == nil {
		a.log = log.Default()
	}
	r := chi.NewRouter()
	r.Use(a.accessLog, a.recoverer, a.cors, securityHeaders)

	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "service": "server"})
	})
	r.Get("/catalog", a.catalog)
	r.Post("/accounts", a.activate)
	r.Get("/accounts/{user_id}/balance", a.balance)
	r.Post("/accounts/{user_id}/topup", a.topup)
	r.Post("/accounts/{user_id}/transactions", a.transact)
	r.Get("/accounts/{user_id}/transactions", a.history)

	r.NotFound(func(w http.ResponseWriter, _ *http.Request) {
		writeError(w, http.StatusNotFound, "not_found", "unknown endpoint")
	})
	r.MethodNotAllowed(func(w http.ResponseWriter, _ *http.Request) {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
	})
	return r
}

// ---- responses -------------------------------------------------------------

type errorBody struct {
	Error  string `json:"error"`
	Detail string `json:"detail"`
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, code, detail string) {
	writeJSON(w, status, errorBody{Error: code, Detail: detail})
}

var statusOf = map[string]int{
	service.CodeInvalidJSON:        http.StatusBadRequest,
	service.CodeInvalidPayload:     http.StatusBadRequest,
	service.CodeContextMismatch:    http.StatusBadRequest,
	service.CodePrivateKeyRejected: http.StatusBadRequest,
	service.CodeEvaluationFailed:   http.StatusBadRequest,
	service.CodeAccountNotFound:    http.StatusNotFound,
	service.CodeProductNotFound:    http.StatusNotFound,
	service.CodeDiscountNotFound:   http.StatusNotFound,
	service.CodeAccountExists:      http.StatusConflict,
}

func (a *api) fail(w http.ResponseWriter, err error) {
	var se *service.Error
	if errors.As(err, &se) {
		status, ok := statusOf[se.Code]
		if !ok {
			a.log.Printf("internal error: %s", se.Detail)
			writeError(w, http.StatusInternalServerError, service.CodeInternal, "internal server error")
			return
		}
		writeError(w, status, se.Code, se.Detail)
		return
	}
	a.log.Printf("internal error: %v", err)
	writeError(w, http.StatusInternalServerError, service.CodeInternal, "internal server error")
}

// ---- request decoding ------------------------------------------------------

type badRequest struct {
	status int
	code   string
	detail string
}

func (b *badRequest) Error() string { return b.detail }

// decode strictly decodes a single JSON object: correct Content-Type, body
// size limit, no unknown fields, no trailing data.
func (a *api) decode(w http.ResponseWriter, r *http.Request, dst any) error {
	if ct := r.Header.Get("Content-Type"); ct != "" {
		mt, _, err := mime.ParseMediaType(ct)
		if err != nil || mt != "application/json" {
			return &badRequest{http.StatusBadRequest, service.CodeInvalidJSON, "Content-Type must be application/json"}
		}
	} else {
		return &badRequest{http.StatusBadRequest, service.CodeInvalidJSON, "Content-Type must be application/json"}
	}
	r.Body = http.MaxBytesReader(w, r.Body, a.opts.MaxBodyBytes)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		var mbe *http.MaxBytesError
		if errors.As(err, &mbe) {
			return &badRequest{http.StatusRequestEntityTooLarge, "payload_too_large", fmt.Sprintf("body exceeds %d bytes", mbe.Limit)}
		}
		return &badRequest{http.StatusBadRequest, service.CodeInvalidJSON, jsonDetail(err)}
	}
	if err := dec.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return &badRequest{http.StatusBadRequest, service.CodeInvalidJSON, "body must contain a single JSON object"}
	}
	return nil
}

func jsonDetail(err error) string {
	var te *json.UnmarshalTypeError
	var se *json.SyntaxError
	switch {
	case errors.As(err, &te):
		return fmt.Sprintf("field %q must be of type %s", te.Field, te.Type)
	case errors.As(err, &se):
		return fmt.Sprintf("malformed JSON at offset %d", se.Offset)
	case errors.Is(err, io.EOF):
		return "empty body"
	case strings.HasPrefix(err.Error(), "json: unknown field"):
		return strings.TrimPrefix(err.Error(), "json: ")
	default:
		return "malformed JSON"
	}
}

func (a *api) badRequest(w http.ResponseWriter, err error) {
	var br *badRequest
	if errors.As(err, &br) {
		writeError(w, br.status, br.code, br.detail)
		return
	}
	writeError(w, http.StatusBadRequest, service.CodeInvalidJSON, "malformed request")
}

// required returns an invalid_json error naming the first missing field.
func required(fields map[string]*string, order ...string) error {
	for _, name := range order {
		if fields[name] == nil {
			return &badRequest{http.StatusBadRequest, service.CodeInvalidJSON, fmt.Sprintf("missing field %q", name)}
		}
	}
	return nil
}

func userIDParam(w http.ResponseWriter, r *http.Request) (string, bool) {
	id := chi.URLParam(r, "user_id")
	if !service.ValidUserID(id) {
		writeError(w, http.StatusBadRequest, service.CodeInvalidJSON, "user_id must match ^[a-z0-9_-]{3,32}$")
		return "", false
	}
	return id, true
}

// ---- endpoints -------------------------------------------------------------

func (a *api) catalog(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, a.svc.Catalog())
}

type activateReq struct {
	UserID     *string `json:"user_id"`
	Params     *string `json:"params"`
	PublicKey  *string `json:"public_key"`
	RelinKey   *string `json:"relin_key"`
	EncBalance *string `json:"enc_balance"`
}

func (a *api) activate(w http.ResponseWriter, r *http.Request) {
	var req activateReq
	if err := a.decode(w, r, &req); err != nil {
		a.badRequest(w, err)
		return
	}
	if err := required(map[string]*string{
		"user_id": req.UserID, "params": req.Params, "public_key": req.PublicKey,
		"relin_key": req.RelinKey, "enc_balance": req.EncBalance,
	}, "user_id", "params", "public_key", "relin_key", "enc_balance"); err != nil {
		a.badRequest(w, err)
		return
	}
	if !service.ValidUserID(*req.UserID) {
		writeError(w, http.StatusBadRequest, service.CodeInvalidJSON, "user_id must match ^[a-z0-9_-]{3,32}$")
		return
	}
	setUser(r, *req.UserID)
	bal, err := a.svc.Activate(r.Context(), service.ActivateInput{
		UserID: *req.UserID, Params: *req.Params, PublicKey: *req.PublicKey, RelinKey: *req.RelinKey, EncBalance: *req.EncBalance,
	})
	if err != nil {
		a.fail(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"status": "activated", "user_id": *req.UserID, "enc_balance": bal})
}

func (a *api) balance(w http.ResponseWriter, r *http.Request) {
	id, ok := userIDParam(w, r)
	if !ok {
		return
	}
	bal, err := a.svc.Balance(r.Context(), id)
	if err != nil {
		a.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"user_id": id, "enc_balance": bal})
}

type topupReq struct {
	EncAmount *string `json:"enc_amount"`
}

func (a *api) topup(w http.ResponseWriter, r *http.Request) {
	id, ok := userIDParam(w, r)
	if !ok {
		return
	}
	var req topupReq
	if err := a.decode(w, r, &req); err != nil {
		a.badRequest(w, err)
		return
	}
	if err := required(map[string]*string{"enc_amount": req.EncAmount}, "enc_amount"); err != nil {
		a.badRequest(w, err)
		return
	}
	bal, tr, err := a.svc.Topup(r.Context(), id, *req.EncAmount)
	if err != nil {
		a.fail(w, err)
		return
	}
	w.Header().Set("Server-Timing", fmt.Sprintf("eval;dur=%.3f", float64(tr.Total())/float64(time.Millisecond)))
	writeJSON(w, http.StatusOK, map[string]string{"user_id": id, "enc_balance": bal})
}

type transactReq struct {
	ProductID    *string `json:"product_id"`
	DiscountCode *string `json:"discount_code"`
	EncQuantity  *string `json:"enc_quantity"`
	EncMetadata  *string `json:"enc_metadata"`
}

func (a *api) transact(w http.ResponseWriter, r *http.Request) {
	id, ok := userIDParam(w, r)
	if !ok {
		return
	}
	var req transactReq
	if err := a.decode(w, r, &req); err != nil {
		a.badRequest(w, err)
		return
	}
	if err := required(map[string]*string{
		"product_id": req.ProductID, "enc_quantity": req.EncQuantity, "enc_metadata": req.EncMetadata,
	}, "product_id", "enc_quantity", "enc_metadata"); err != nil {
		a.badRequest(w, err)
		return
	}
	discount := ""
	if req.DiscountCode != nil {
		discount = *req.DiscountCode
	}
	out, err := a.svc.Transact(r.Context(), service.TransactInput{
		UserID: id, ProductID: *req.ProductID, DiscountCode: discount,
		EncQuantity: *req.EncQuantity, EncMetadata: *req.EncMetadata,
	})
	if err != nil {
		a.fail(w, err)
		return
	}
	w.Header().Set("Server-Timing", fmt.Sprintf("eval;dur=%.3f", float64(out.Trace.Total())/float64(time.Millisecond)))
	writeJSON(w, http.StatusCreated, map[string]string{
		"transaction_id": out.TransactionID, "enc_total": out.EncTotal, "enc_balance": out.EncBalance,
	})
}

type historyItem struct {
	TransactionID string `json:"transaction_id"`
	EncMetadata   string `json:"enc_metadata"`
	EncTotal      string `json:"enc_total"`
	CreatedAt     string `json:"created_at"`
}

func (a *api) history(w http.ResponseWriter, r *http.Request) {
	id, ok := userIDParam(w, r)
	if !ok {
		return
	}
	items, err := a.svc.History(r.Context(), id)
	if err != nil {
		a.fail(w, err)
		return
	}
	out := make([]historyItem, 0, len(items))
	for _, it := range items {
		out = append(out, historyItem{
			TransactionID: it.TransactionID, EncMetadata: it.EncMetadata, EncTotal: it.EncTotal,
			CreatedAt: it.CreatedAt.UTC().Format(time.RFC3339),
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"user_id": id, "transactions": out})
}
