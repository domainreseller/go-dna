package dna

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestNewClient(t *testing.T) {
	c, err := NewClient("  "+testResellerID+" ", " "+testAPIKey+"\n")
	if err != nil {
		t.Fatal(err)
	}
	if c.BaseURL() != ProductionURL || c.Environment() != Production {
		t.Errorf("default environment: got %s %s", c.BaseURL(), c.Environment())
	}
	if c.resellerID != testResellerID || c.apiKey != testAPIKey {
		t.Errorf("credentials are not trimmed: %q %q", c.resellerID, c.apiKey)
	}
	if c.Domains == nil || c.DNS == nil || c.Products == nil {
		t.Fatal("services are not set")
	}

	c, err = NewClient(testResellerID, testAPIKey, WithEnvironment(OTE))
	if err != nil || c.BaseURL() != OTEURL || c.Environment() != OTE {
		t.Errorf("WithEnvironment(OTE): %v %v", c, err)
	}

	for _, tc := range []struct{ id, key string }{{"", "k"}, {"r", ""}, {"  ", "  "}} {
		if _, err := NewClient(tc.id, tc.key); !errors.Is(err, ErrNoCredentials) || !IsAuth(err) {
			t.Errorf("NewClient(%q, %q): got %v, want ErrNoCredentials", tc.id, tc.key, err)
		}
	}
	for _, raw := range []string{"", "ftp://x", "not a url", "https://"} {
		if _, err := NewClient("r", "k", WithBaseURL(raw)); err == nil {
			t.Errorf("WithBaseURL(%q) accepted", raw)
		}
	}
	if _, err := NewClient("r", "k", WithHTTPClient(nil)); err == nil {
		t.Error("WithHTTPClient(nil) accepted")
	}
	if _, err := NewClient("r", "k", WithEnvironment("staging")); err == nil {
		t.Error("unknown environment accepted")
	}
}

func TestParseEnvironment(t *testing.T) {
	for in, want := range map[string]Environment{
		"": Production, "production": Production, "PROD": Production, "live": Production, "false": Production, "0": Production,
		"ote": OTE, "Test": OTE, "sandbox": OTE, "true": OTE, "1": OTE, " yes ": OTE, "on": OTE,
	} {
		got, err := ParseEnvironment(in)
		if err != nil || got != want {
			t.Errorf("ParseEnvironment(%q) = %v, %v; want %v", in, got, err, want)
		}
	}
	if _, err := ParseEnvironment("staging"); err == nil {
		t.Error("ParseEnvironment(staging) accepted")
	}
}

func TestNewClientFromEnv(t *testing.T) {
	t.Setenv(EnvResellerID, testResellerID)
	t.Setenv(EnvAPIKey, testAPIKey)
	t.Setenv(EnvEnv, "")
	t.Setenv(EnvTestMode, "true")
	t.Setenv(EnvBaseURL, "")
	c, err := NewClientFromEnv()
	if err != nil || c.Environment() != OTE {
		t.Fatalf("DNA_TEST_MODE=true: %v %v", c, err)
	}

	t.Setenv(EnvEnv, "production")
	if c, _ := NewClientFromEnv(); c.Environment() != Production {
		t.Error("DNA_ENV should win over DNA_TEST_MODE")
	}

	t.Setenv(EnvBaseURL, "http://127.0.0.1:9")
	if c, _ := NewClientFromEnv(); c.BaseURL() != "http://127.0.0.1:9" {
		t.Errorf("DNA_BASE_URL: got %s", c.BaseURL())
	}
	if c, _ := NewClientFromEnv(WithBaseURL("http://127.0.0.1:10")); c.BaseURL() != "http://127.0.0.1:10" {
		t.Error("options should win over the environment")
	}

	t.Setenv(EnvAPIKey, "")
	if _, err := NewClientFromEnv(); !errors.Is(err, ErrNoCredentials) || !strings.Contains(err.Error(), EnvAPIKey) {
		t.Errorf("missing key: %v", err)
	}
}

func TestRequestHeaders(t *testing.T) {
	f := newFakeAPI(t, ok(`{"resellerId":"r"}`), ok(`{}`))
	c := f.client(WithUserAgent("my-panel/1.4"))

	if _, err := c.Account.Get(context.Background()); err != nil {
		t.Fatal(err)
	}
	r := f.last()
	if r.Method != http.MethodGet || r.Path != "/api/v1/deposit/accounts/me" {
		t.Errorf("got %s %s", r.Method, r.Path)
	}
	if got := r.Header.Get("X-Api-Key"); got != testAPIKey {
		t.Errorf("X-API-KEY = %q", got)
	}
	if got := r.Header.Get("__reseller"); got != testResellerID {
		t.Errorf("__reseller = %q", got)
	}
	if got := r.Header.Get("User-Agent"); got != "my-panel/1.4 "+UserAgent {
		t.Errorf("User-Agent = %q", got)
	}
	if got := r.Header.Get("Accept"); got != "application/json" {
		t.Errorf("Accept = %q", got)
	}
	if got := r.Header.Get("Content-Type"); got != "" {
		t.Errorf("a GET has Content-Type %q", got)
	}
	if !regexp.MustCompile(`^[0-9a-f]{32}$`).MatchString(r.Header.Get("X-Correlation-Id")) {
		t.Errorf("X-Correlation-ID = %q", r.Header.Get("X-Correlation-Id"))
	}

	if err := c.Domains.Lock(context.Background(), "example.com"); err != nil {
		t.Fatal(err)
	}
	r = f.last()
	if got := r.Header.Get("Content-Type"); got != "application/json" {
		t.Errorf("a POST has Content-Type %q", got)
	}
	if string(r.Body) != `{"domainName":"example.com"}` {
		t.Errorf("body = %s", r.Body)
	}
}

func TestValidationStopsTheRequest(t *testing.T) {
	f := newFakeAPI(t, ok(`{}`))
	c := f.client()
	_, err := c.Domains.Renew(context.Background(), " ", 0)
	var ie *InputError
	if !errors.As(err, &ie) {
		t.Fatalf("got %v, want *InputError", err)
	}
	if ie.Request != "DomainRenewRequest" || len(ie.Fields) != 2 {
		t.Errorf("InputError = %+v", ie)
	}
	if KindOf(err) != KindInvalidInput {
		t.Errorf("KindOf = %s", KindOf(err))
	}
	if !strings.Contains(err.Error(), "domainName: is required") || !strings.Contains(err.Error(), "years: must be between 1 and 10") {
		t.Errorf("message: %v", err)
	}
	if n := len(f.requests()); n != 0 {
		t.Errorf("%d requests were sent", n)
	}
}

func TestReadOnly(t *testing.T) {
	f := newFakeAPI(t, ok(`{"success":true,"info":{"status":"AVAILABLE"}}`))
	c := f.client(WithReadOnly())
	if !c.ReadOnly() {
		t.Fatal("ReadOnly() = false")
	}
	_, err := c.Domains.Renew(context.Background(), "example.com", 1)
	if !errors.Is(err, ErrReadOnly) || KindOf(err) != KindReadOnly {
		t.Fatalf("billable request: got %v", err)
	}
	if !strings.Contains(err.Error(), "DomainRenewRequest") || !strings.Contains(err.Error(), "billable") {
		t.Errorf("message: %v", err)
	}
	if err := c.Domains.Lock(context.Background(), "example.com"); !errors.Is(err, ErrReadOnly) {
		t.Errorf("write request: got %v", err)
	}
	if len(f.requests()) != 0 {
		t.Fatal("a blocked request was sent")
	}
	// A POST that only reads is allowed.
	if _, err := c.Domains.Search(context.Background(), "example.com"); err != nil {
		t.Errorf("search: %v", err)
	}
}

func TestRetriesRateLimitEvenForBillable(t *testing.T) {
	f := newFakeAPI(t,
		reply{Status: 429, Header: map[string]string{"Retry-After": "0"}},
		ok(`{"success":true,"domainId":"d1","status":"Active"}`),
	)
	res, err := f.client().Domains.Renew(context.Background(), "example.com", 1)
	if err != nil {
		t.Fatal(err)
	}
	if res.DomainID != "d1" || len(f.requests()) != 2 {
		t.Errorf("result %+v after %d requests", res, len(f.requests()))
	}
}

func TestRateLimitAfterRetries(t *testing.T) {
	f := newFakeAPI(t, reply{Status: 429, Header: map[string]string{"Retry-After": "0"}})
	_, err := f.client(WithRetries(1)).Account.Get(context.Background())
	if !IsRateLimited(err) || len(f.requests()) != 2 {
		t.Fatalf("got %v after %d requests", err, len(f.requests()))
	}
	var ae *APIError
	errors.As(err, &ae)
	if ae.Code != CodeRateLimited || !ae.Retryable() {
		t.Errorf("APIError = %+v", ae)
	}
}

func TestServerErrorsRetriedOnlyWhenSafe(t *testing.T) {
	gateway := reply{Status: 503, Header: map[string]string{"Retry-After": "0"}}

	f := newFakeAPI(t, gateway, ok(`{"resellerId":"r"}`))
	if _, err := f.client().Account.Get(context.Background()); err != nil || len(f.requests()) != 2 {
		t.Errorf("read: %v after %d requests", err, len(f.requests()))
	}

	f = newFakeAPI(t, gateway, ok(`{}`))
	if err := f.client().Domains.SetNameServers(context.Background(), DomainNameServersSetRequest{DomainName: "example.com", NameServers: []string{"ns1.example.net"}}); err != nil || len(f.requests()) != 2 {
		t.Errorf("idempotent write: %v after %d requests", err, len(f.requests()))
	}

	f = newFakeAPI(t, gateway, ok(`{"success":true}`))
	_, err := f.client().Domains.Renew(context.Background(), "example.com", 1)
	if KindOf(err) != KindServer || len(f.requests()) != 1 {
		t.Errorf("billable: %v after %d requests; want one attempt", err, len(f.requests()))
	}

	f = newFakeAPI(t, reply{Status: 500, Body: `{"error":{"code":"Dna.DomainService:General:10001","message":"An internal error has occurred."}}`})
	_, err = f.client().Account.Get(context.Background())
	if KindOf(err) != KindServer || len(f.requests()) != 1 {
		t.Errorf("500: %v after %d requests; 500 is not retried", err, len(f.requests()))
	}
}

// brokenServer accepts connections and closes them without answering.
func brokenServer(t *testing.T) (string, *atomic.Int32) {
	t.Helper()
	var n atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n.Add(1)
		conn, _, err := w.(http.Hijacker).Hijack()
		if err == nil {
			conn.Close()
		}
	}))
	t.Cleanup(srv.Close)
	return srv.URL, &n
}

func TestNetworkErrors(t *testing.T) {
	u, n := brokenServer(t)
	c, _ := NewClient("r", "k", WithBaseURL(u), WithPacing(false), WithRetries(1))

	_, err := c.Account.Get(context.Background())
	var ne *NetworkError
	if !errors.As(err, &ne) || KindOf(err) != KindNetwork {
		t.Fatalf("read: got %v", err)
	}
	if n.Load() != 2 {
		t.Errorf("read: %d attempts, want 2 (retried)", n.Load())
	}
	if OutcomeUnknown(err) {
		t.Error("a failed read has a known outcome")
	}

	n.Store(0)
	_, err = c.Domains.Register(context.Background(), DomainRegisterRequest{DomainName: "example.com", NameServers: []string{"ns1.example.net"}, Handles: SameHandle("DNA-1")})
	if !errors.As(err, &ne) || n.Load() != 1 {
		t.Fatalf("billable: %v after %d attempts; want one", err, n.Load())
	}
	if !OutcomeUnknown(err) {
		t.Error("a failed registration must report OutcomeUnknown")
	}
}

func TestRedirectIsAuthError(t *testing.T) {
	f := newFakeAPI(t, reply{Status: http.StatusFound, Header: map[string]string{"Location": "/Error/Suspend"}})
	_, err := f.client().Account.Get(context.Background())
	if !IsAuth(err) || !HasCode(err, CodeInvalidReseller) {
		t.Fatalf("got %v", err)
	}
	if len(f.requests()) != 1 {
		t.Errorf("the redirect was followed: %d requests", len(f.requests()))
	}
}

func TestContextCancel(t *testing.T) {
	f := newFakeAPI(t, ok(`{}`))
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := f.client().Account.Get(ctx)
	if !errors.Is(err, context.Canceled) || KindOf(err) != KindNetwork {
		t.Fatalf("got %v", err)
	}
}

func TestCorrelationID(t *testing.T) {
	f := newFakeAPI(t,
		ok(`{}`),
		reply{Status: 404, Body: `{"error":{"code":"Dna.DomainService:Domain:10007","message":"not found"}}`, Header: map[string]string{"X-Correlation-Id": "from-server"}},
		reply{Status: 404, Body: `{"error":{"code":"x","message":"y"}}`},
	)
	c := f.client()
	ctx := WithCorrelationID(context.Background(), "my-trace-1")
	if _, err := c.Account.Get(ctx); err != nil {
		t.Fatal(err)
	}
	if got := f.last().Header.Get("X-Correlation-Id"); got != "my-trace-1" {
		t.Errorf("header = %q", got)
	}
	_, err := c.Domains.Info(ctx, "example.com")
	var ae *APIError
	if !errors.As(err, &ae) || ae.CorrelationID != "from-server" {
		t.Errorf("server correlation id: %+v", err)
	}
	_, err = c.Domains.Info(ctx, "example.com")
	if !errors.As(err, &ae) || ae.CorrelationID != "my-trace-1" {
		t.Errorf("fallback correlation id: %+v", err)
	}
}

func TestTraceAndLogger(t *testing.T) {
	f := newFakeAPI(t, ok(`{"resellerId":"r"}`))
	var events []TraceEvent
	var logs bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logs, &slog.HandlerOptions{Level: slog.LevelDebug}))
	c := f.client(WithTrace(func(ev TraceEvent) { events = append(events, ev) }), WithLogger(logger))
	if _, err := c.Account.Get(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(events) != 2 || events[0].Kind != TraceRequest || events[1].Kind != TraceResponse || events[1].StatusCode != 200 {
		t.Fatalf("events = %+v", events)
	}
	if events[0].Safety != Read || events[0].CorrelationID == "" {
		t.Errorf("event = %+v", events[0])
	}
	for _, ev := range events {
		if strings.Contains(ev.URL+string(ev.Body), testAPIKey) {
			t.Error("a trace event contains the API key")
		}
	}
	if !strings.Contains(logs.String(), "dna: request") || !strings.Contains(logs.String(), "dna: response") {
		t.Errorf("logs: %s", logs.String())
	}
	if strings.Contains(logs.String(), testAPIKey) {
		t.Error("the logs contain the API key")
	}
}

// pingRequest is a request type defined outside the package's own set, the
// way a user would write one.
type pingRequest struct {
	BaseRequest
	Name string
}

func (pingRequest) Endpoint() Endpoint {
	return Endpoint{Method: http.MethodPost, Path: "api/v1/ping", Safety: Read}
}
func (r pingRequest) Query() url.Values { return url.Values{"verbose": {"1"}} }
func (r pingRequest) Body() any         { return map[string]string{"name": r.Name} }

func TestDoWithOwnRequest(t *testing.T) {
	f := newFakeAPI(t, ok(`{"pong":true}`))
	c := f.client()
	var raw json.RawMessage
	resp, err := c.Do(context.Background(), pingRequest{Name: "x"}, &raw)
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != `{"pong":true}` || resp.StatusCode != 200 || resp.Attempts != 1 {
		t.Errorf("raw %s, resp %+v", raw, resp)
	}
	r := f.last()
	if r.Path != "/api/v1/ping" || r.Query.Get("verbose") != "1" || string(r.Body) != `{"name":"x"}` {
		t.Errorf("request %+v", r)
	}

	var typed struct {
		Pong bool `json:"pong"`
	}
	if _, err := c.Do(context.Background(), pingRequest{}, &typed); err != nil || !typed.Pong {
		t.Errorf("typed decode: %v %+v", err, typed)
	}
	if _, err := c.Do(context.Background(), nil, nil); err == nil {
		t.Error("nil request accepted")
	}
}

func TestNewHTTPRequestDoesNotSend(t *testing.T) {
	f := newFakeAPI(t, ok(`{}`))
	c := f.client()
	req, err := c.NewHTTPRequest(context.Background(), DomainRenewRequest{DomainName: "example.com", Years: 2})
	if err != nil {
		t.Fatal(err)
	}
	body, _ := req.GetBody()
	b, _ := io.ReadAll(body)
	if req.Method != http.MethodPost || req.URL.Path != "/api/v1/domains/renew" || string(b) != `{"domainName":"example.com","period":2}` {
		t.Errorf("%s %s %s", req.Method, req.URL, b)
	}
	if len(f.requests()) != 0 {
		t.Error("NewHTTPRequest sent the request")
	}
	if _, err := c.NewHTTPRequest(context.Background(), DomainRenewRequest{}); err == nil {
		t.Error("NewHTTPRequest skipped validation")
	}
}

func TestQuotaHeaders(t *testing.T) {
	f := newFakeAPI(t, reply{Status: 200, Body: `{}`, Header: map[string]string{
		"X-Quota-Scope": "domain", "X-Quota-Cost": "10", "X-Quota-Used": "120.5", "X-Quota-Limit": "200", "X-Quota-Delay": "0", "X-Quota-Reset": "3",
	}})
	resp, err := f.client().Do(context.Background(), AccountGetRequest{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	want := Quota{Scope: "domain", Cost: 10, Used: 120.5, Limit: 200, Delay: 0, Reset: 3}
	if resp.Quota == nil || *resp.Quota != want {
		t.Errorf("quota = %+v", resp.Quota)
	}
	if parseQuota(http.Header{}) != nil {
		t.Error("quota without headers")
	}
}

func TestDecodeError(t *testing.T) {
	f := newFakeAPI(t, ok(`{"resellerId": 12`))
	if _, err := f.client().Account.Get(context.Background()); err == nil || !strings.Contains(err.Error(), "decode response") {
		t.Errorf("got %v", err)
	}
}

func TestPacing(t *testing.T) {
	for path, want := range map[string]string{
		"/api/v1/domains/zones":       "zones",
		"/api/v1/domains/zones/apply": "zones",
		"/api/v1/domains/search":      "search",
		"/api/v1/domains/bulk-search": "search",
		"/api/v1/domains/info":        "",
		"/api/v1/products/tlds":       "",
		"/api/v1/domains/renew":       "general",
	} {
		if got, _ := paceClass(path); got != want {
			t.Errorf("paceClass(%s) = %q, want %q", path, got, want)
		}
	}
	p := newPacer()
	start := time.Now()
	for i := 0; i < 3; i++ {
		if err := p.wait(context.Background(), "/api/v1/domains/renew"); err != nil {
			t.Fatal(err)
		}
	}
	if d := time.Since(start); d < 390*time.Millisecond {
		t.Errorf("three general requests took %s, want at least 400ms", d)
	}
}

func TestRetryAfter(t *testing.T) {
	if d := retryAfter(http.Header{"Retry-After": {"2"}}, 1); d < 2*time.Second || d > 3*time.Second {
		t.Errorf("Retry-After 2: %s", d)
	}
	if d := retryAfter(http.Header{}, 1); d < 500*time.Millisecond || d > time.Second {
		t.Errorf("no header: %s", d)
	}
	if !retryableNetErr(&net.OpError{Op: "dial", Err: errors.New("refused")}) || !retryableNetErr(io.ErrUnexpectedEOF) {
		t.Error("transport errors should be retryable")
	}
	if retryableNetErr(errors.New("x")) {
		t.Error("an arbitrary error is not a transport error")
	}
}
