package dna

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	mrand "math/rand/v2"
	"net"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Response describes a completed HTTP exchange.
type Response struct {
	// StatusCode is the HTTP status of the final attempt.
	StatusCode int
	// Header holds the response headers.
	Header http.Header
	// Body is the raw response body.
	Body []byte
	// CorrelationID identifies the request; quote it when contacting support.
	CorrelationID string
	// Quota is the soft-quota state after the request, or nil.
	Quota *Quota
	// Duration is the total time, including retries and pacing.
	Duration time.Duration
	// Attempts is the number of HTTP attempts.
	Attempts int
}

// Quota mirrors the API's soft-quota headers. The quota never rejects
// requests: once Used exceeds Limit, the API delays each response (by Delay
// seconds, at most 5) until points decay by 20% per minute. Lookups cost 1
// point, everything else 10.
type Quota struct {
	Scope string  `json:"scope"`        // area: domain, contact, ssl, products, order, deposit, hosting
	Cost  float64 `json:"cost"`         // points this request cost
	Used  float64 `json:"used"`         // points in the bucket after this request
	Limit float64 `json:"limit"`        // where slowing down starts (200)
	Delay float64 `json:"delaySeconds"` // seconds this response was held back
	Reset float64 `json:"resetSeconds"` // seconds until the bucket is back under Limit
}

func parseQuota(h http.Header) *Quota {
	if h.Get("X-Quota-Limit") == "" && h.Get("X-Quota-Used") == "" {
		return nil
	}
	f := func(k string) float64 {
		n, _ := strconv.ParseFloat(strings.TrimSpace(h.Get(k)), 64)
		return n
	}
	return &Quota{
		Scope: h.Get("X-Quota-Scope"),
		Cost:  f("X-Quota-Cost"),
		Used:  f("X-Quota-Used"),
		Limit: f("X-Quota-Limit"),
		Delay: f("X-Quota-Delay"),
		Reset: f("X-Quota-Reset"),
	}
}

// TraceKind is the kind of a TraceEvent.
type TraceKind string

const (
	TraceRequest  TraceKind = "request"  // an attempt is about to be sent
	TraceResponse TraceKind = "response" // an attempt finished (with a status or an error)
	TraceRetry    TraceKind = "retry"    // the client waits and tries again
)

// TraceEvent describes one step of a request, for logging and debugging.
// It never contains the API key.
type TraceEvent struct {
	Kind          TraceKind
	Method        string
	URL           string
	Safety        Safety
	CorrelationID string
	Attempt       int           // 1-based
	StatusCode    int           // 0 before a response or after a transport error
	Duration      time.Duration // time spent on this attempt
	Wait          time.Duration // for TraceRetry: how long the client waits
	// Body is the request or response body, truncated to 16 KiB. It can
	// hold contact details and auth codes: mask it before storing it.
	Body  []byte
	Quota *Quota
	Err   error
}

type correlationKey struct{}

// WithCorrelationID makes requests sent with ctx carry id in the
// X-Correlation-ID header. Without it the client generates one per request.
// The API logs it; quote it when contacting support.
func WithCorrelationID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, correlationKey{}, id)
}

func correlationID(ctx context.Context) string {
	if id, _ := ctx.Value(correlationKey{}).(string); strings.TrimSpace(id) != "" {
		return strings.TrimSpace(id)
	}
	var b [16]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}

// requestName returns the type name of r without package or pointer, e.g.
// "DomainInfoRequest".
func requestName(r Request) string {
	t := reflect.TypeOf(r)
	for t != nil && t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t == nil || t.Name() == "" {
		return "request"
	}
	return t.Name()
}

// prepare validates r and checks the client's policy. It returns the
// endpoint and the encoded body.
func (c *Client) prepare(r Request) (Endpoint, []byte, error) {
	if r == nil || (reflect.ValueOf(r).Kind() == reflect.Pointer && reflect.ValueOf(r).IsNil()) {
		return Endpoint{}, nil, errors.New("dna: nil request")
	}
	ep := r.Endpoint()
	ep.Method = strings.ToUpper(strings.TrimSpace(ep.Method))
	switch ep.Method {
	case "":
		ep.Method = http.MethodGet
	case http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch:
	default:
		return ep, nil, fmt.Errorf("dna: %s: unsupported method %q", requestName(r), ep.Method)
	}
	if !strings.HasPrefix(ep.Path, "/") {
		ep.Path = "/" + ep.Path
	}
	if err := r.Validate(); err != nil {
		var ie *InputError
		if !errors.As(err, &ie) {
			ie = &InputError{Fields: []FieldError{{Message: err.Error()}}}
		}
		if ie.Request == "" {
			ie.Request = requestName(r)
		}
		return ep, nil, ie
	}
	if c.readOnly && ep.Safety != Read {
		return ep, nil, fmt.Errorf("%w: %s (%s %s) is a %s request", ErrReadOnly, requestName(r), ep.Method, ep.Path, ep.Safety)
	}
	var body []byte
	if b := r.Body(); b != nil {
		var err error
		if body, err = json.Marshal(b); err != nil {
			return ep, nil, fmt.Errorf("dna: %s: encode body: %w", requestName(r), err)
		}
	}
	return ep, body, nil
}

func (c *Client) newHTTPRequest(ctx context.Context, ep Endpoint, r Request, body []byte, corrID string) (*http.Request, error) {
	u := strings.TrimRight(c.baseURL.String(), "/") + ep.Path
	if q := r.Query(); len(q) > 0 {
		u += "?" + q.Encode()
	}
	var rd io.Reader
	if body != nil {
		rd = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, ep.Method, u, rd)
	if err != nil {
		return nil, fmt.Errorf("dna: %s: %w", requestName(r), err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("X-API-KEY", c.apiKey)
	req.Header["__reseller"] = []string{c.resellerID} // non-canonical name, sent verbatim
	req.Header.Set("X-Correlation-ID", corrID)
	req.Header.Set("X-App", c.app)
	req.Header.Set("X-Version", c.appVersion)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return req, nil
}

// NewHTTPRequest validates r and builds the *http.Request Do would send,
// without sending it. Use it to log or review a billable request before
// sending it. The request carries your credentials in its headers.
func (c *Client) NewHTTPRequest(ctx context.Context, r Request) (*http.Request, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	ep, body, err := c.prepare(r)
	if err != nil {
		return nil, err
	}
	return c.newHTTPRequest(ctx, ep, r, body, correlationID(ctx))
}

// Do validates r, sends it and decodes a successful JSON response into out
// (when out is non-nil and the response has a body). Pass a
// *json.RawMessage to keep the body as it is. The services call Do; call it
// yourself for your own Request types.
//
// Errors:
//   - *InputError: Validate rejected the request; nothing was sent.
//   - ErrReadOnly (wrapped): the client is read-only; nothing was sent.
//   - *APIError: the API answered with a non-2xx status.
//   - *NetworkError: the request did not complete.
//
// Retries: 429 responses are always retried, honouring Retry-After (the
// API rejects them before doing anything). Network errors and 502/503/504
// are retried only for Read and Idempotent requests.
func (c *Client) Do(ctx context.Context, r Request, out any) (*Response, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	ep, body, err := c.prepare(r)
	if err != nil {
		return nil, err
	}
	corrID := correlationID(ctx)
	idempotent := ep.Safety == Read || ep.Idempotent
	start := time.Now()

	for attempt := 1; ; attempt++ {
		if c.pacer != nil {
			if err := c.pacer.wait(ctx, ep.Path); err != nil {
				return nil, &NetworkError{Method: ep.Method, Path: ep.Path, Err: err, safety: Read}
			}
		}
		req, err := c.newHTTPRequest(ctx, ep, r, body, corrID)
		if err != nil {
			return nil, err
		}
		ev := TraceEvent{Method: ep.Method, URL: req.URL.String(), Safety: ep.Safety, CorrelationID: corrID, Attempt: attempt}
		c.emit(ev, TraceRequest, func(e *TraceEvent) { e.Body = truncate(body) })

		t0 := time.Now()
		hresp, err := c.httpClient.Do(req)
		if err != nil {
			c.emit(ev, TraceResponse, func(e *TraceEvent) { e.Duration, e.Err = time.Since(t0), err })
			nerr := &NetworkError{Method: ep.Method, Path: ep.Path, Err: err, safety: ep.Safety}
			if ctx.Err() != nil {
				nerr.Err = ctx.Err()
				return nil, nerr
			}
			if idempotent && attempt <= c.maxRetries && retryableNetErr(err) {
				wait := backoff(attempt)
				c.emit(ev, TraceRetry, func(e *TraceEvent) { e.Wait, e.Err = wait, err })
				if err := sleep(ctx, wait); err != nil {
					nerr.Err = err
					return nil, nerr
				}
				continue
			}
			return nil, nerr
		}
		respBody, readErr := io.ReadAll(io.LimitReader(hresp.Body, 64<<20))
		hresp.Body.Close()
		resp := &Response{
			StatusCode:    hresp.StatusCode,
			Header:        hresp.Header,
			Body:          respBody,
			CorrelationID: firstNonEmpty(hresp.Header.Get("X-Correlation-Id"), corrID),
			Quota:         parseQuota(hresp.Header),
			Duration:      time.Since(start),
			Attempts:      attempt,
		}
		c.emit(ev, TraceResponse, func(e *TraceEvent) {
			e.StatusCode, e.Duration, e.Body, e.Quota, e.Err = resp.StatusCode, time.Since(t0), truncate(respBody), resp.Quota, readErr
		})
		if readErr != nil {
			return resp, &NetworkError{Method: ep.Method, Path: ep.Path, Err: readErr, safety: ep.Safety}
		}

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			if out != nil && len(bytes.TrimSpace(respBody)) > 0 {
				if raw, ok := out.(*json.RawMessage); ok {
					*raw = append((*raw)[:0], respBody...)
				} else if err := json.Unmarshal(respBody, out); err != nil {
					return resp, fmt.Errorf("dna: %s %s: decode response: %w", ep.Method, ep.Path, err)
				}
			}
			return resp, nil
		}

		retry := resp.StatusCode == http.StatusTooManyRequests ||
			(idempotent && (resp.StatusCode == 502 || resp.StatusCode == 503 || resp.StatusCode == 504))
		if retry && attempt <= c.maxRetries {
			if wait := retryAfter(hresp.Header, attempt); wait <= c.maxWait {
				c.emit(ev, TraceRetry, func(e *TraceEvent) { e.StatusCode, e.Wait = resp.StatusCode, wait })
				if err := sleep(ctx, wait); err != nil {
					return resp, &NetworkError{Method: ep.Method, Path: ep.Path, Err: err, safety: Read}
				}
				continue
			}
		}
		apiErr := newAPIError(ep.Method, ep.Path, resp.StatusCode, hresp.Header, respBody)
		if apiErr.CorrelationID == "" {
			apiErr.CorrelationID = corrID
		}
		return resp, apiErr
	}
}

// call is Do for the services: it returns the decoded value.
func call[T any](ctx context.Context, c *Client, r Request) (*T, error) {
	out, _, err := send[T](ctx, c, r)
	return out, err
}

// send is call that also returns the response, for services that check a
// "success" flag and need the correlation ID for the error.
func send[T any](ctx context.Context, c *Client, r Request) (*T, *Response, error) {
	out := new(T)
	resp, err := c.Do(ctx, r, out)
	if err != nil {
		return nil, resp, err
	}
	return out, resp, nil
}

// operationError builds the error for a 2xx answer that reports failure.
func operationError(r Request, resp *Response, code, message string) *OperationError {
	ep := r.Endpoint()
	e := &OperationError{Method: ep.Method, Path: ep.Path, Code: strings.TrimSpace(code), Message: strings.TrimSpace(message)}
	if resp != nil {
		e.CorrelationID = resp.CorrelationID
	}
	return e
}

// exec is Do for operations without a response body.
func exec(ctx context.Context, c *Client, r Request) error {
	_, err := c.Do(ctx, r, nil)
	return err
}

func (c *Client) emit(base TraceEvent, kind TraceKind, fill func(*TraceEvent)) {
	if len(c.trace) == 0 {
		return
	}
	ev := base
	ev.Kind = kind
	fill(&ev)
	for _, fn := range c.trace {
		fn(ev)
	}
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

func truncate(b []byte) []byte {
	const max = 16 << 10
	if len(b) > max {
		return append(append([]byte{}, b[:max]...), "…(truncated)"...)
	}
	return b
}

func retryableNetErr(err error) bool {
	var ne net.Error
	if errors.As(err, &ne) {
		return true
	}
	return errors.Is(err, io.ErrUnexpectedEOF) || errors.Is(err, io.EOF) ||
		strings.Contains(err.Error(), "connection reset")
}

func backoff(attempt int) time.Duration {
	base := 500 * time.Millisecond << (attempt - 1)
	return base + time.Duration(mrand.Int64N(int64(250*time.Millisecond)))
}

func retryAfter(h http.Header, attempt int) time.Duration {
	if s := strings.TrimSpace(h.Get("Retry-After")); s != "" {
		if n, err := strconv.ParseFloat(s, 64); err == nil && n >= 0 {
			return time.Duration(n*float64(time.Second)) + 50*time.Millisecond
		}
		if t, err := http.ParseTime(s); err == nil {
			if d := time.Until(t); d > 0 {
				return d
			}
		}
	}
	return backoff(attempt)
}

func sleep(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return ctx.Err()
	}
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

// pacer spaces requests per rate-limit class.
type pacer struct {
	mu   sync.Mutex
	last map[string]time.Time
}

func newPacer() *pacer { return &pacer{last: map[string]time.Time{}} }

func paceClass(path string) (string, time.Duration) {
	p := strings.ToLower(path)
	switch {
	case strings.HasPrefix(p, "/api/v1/domains/zones"):
		return "zones", time.Second
	case p == "/api/v1/domains/search" || p == "/api/v1/domains/bulk-search":
		return "search", 500 * time.Millisecond
	case p == "/api/v1/domains/info", strings.HasPrefix(p, "/api/v1/products/tlds"):
		return "", 0 // not rate limited
	}
	return "general", 200 * time.Millisecond
}

func (p *pacer) wait(ctx context.Context, path string) error {
	class, gap := paceClass(path)
	if gap == 0 {
		return nil
	}
	p.mu.Lock()
	now := time.Now()
	next := p.last[class].Add(gap)
	if next.Before(now) {
		next = now
	}
	p.last[class] = next
	if class == "search" { // searches also count against the overall budget
		if g := p.last["general"]; g.Before(next) {
			p.last["general"] = next
		}
	}
	p.mu.Unlock()
	return sleep(ctx, time.Until(next))
}
