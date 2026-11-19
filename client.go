package dna

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

// Base URLs of the two API environments.
const (
	// ProductionURL is the live environment: real domains, real charges.
	ProductionURL = "https://api.domainresellerapi.com"
	// OTEURL is the test environment (Operational Test & Evaluation). It has
	// its own data and its own API key; nothing you do there touches live
	// domains or your live balance.
	OTEURL = "https://ote.domainresellerapi.com"
)

// Environment selects the API deployment a client talks to.
type Environment string

const (
	// Production is the live environment (the default).
	Production Environment = "production"
	// OTE is the test environment. Use your OTE API key with it.
	OTE Environment = "ote"
)

// BaseURL returns the environment's base URL.
func (e Environment) BaseURL() string {
	if e == OTE {
		return OTEURL
	}
	return ProductionURL
}

// ParseEnvironment accepts "production", "prod", "live" and "ote", "test",
// "sandbox". It also accepts test-mode flags as stored by panels and the
// other DomainNameAPI SDKs: "true", "1", "on", "yes" mean OTE and "false",
// "0", "off", "no" mean production.
func ParseEnvironment(s string) (Environment, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "", "production", "prod", "live", "false", "0", "off", "no":
		return Production, nil
	case "ote", "test", "testing", "sandbox", "true", "1", "on", "yes":
		return OTE, nil
	}
	return "", fmt.Errorf("dna: unknown environment %q (use \"production\" or \"ote\")", s)
}

// Environment variables read by NewClientFromEnv. They are the same ones the
// dna CLI uses.
const (
	EnvResellerID = "DNA_RESELLER_ID"
	EnvAPIKey     = "DNA_API_KEY"
	EnvEnv        = "DNA_ENV"       // "production" or "ote"
	EnvTestMode   = "DNA_TEST_MODE" // "true" selects OTE
	EnvBaseURL    = "DNA_BASE_URL"  // overrides the environment's URL
)

// Client talks to the DomainNameAPI REST API. The services (Domains, DNS,
// SSL, ...) group the operations the same way the dna CLI groups its
// commands.
//
// A Client is safe for concurrent use. Create one and reuse it: it paces
// requests to stay under the API's rate limits.
type Client struct {
	// Account: balance and balance transactions.
	Account *AccountService
	// Contacts: reusable contact handles.
	Contacts *ContactService
	// Domains: search, register, renew, transfer and manage domains.
	Domains *DomainService
	// DNS: zone records of domains on the platform nameservers.
	DNS *DNSService
	// DNSSEC: DS records of domains.
	DNSSEC *DNSSECService
	// SSL: certificates.
	SSL *SSLService
	// Hosting: web hosting packages and services.
	Hosting *HostingService
	// Orders: orders and their lines.
	Orders *OrderService
	// Products: the catalogue and prices of TLDs, SSL and other products.
	Products *ProductService

	baseURL    *url.URL
	resellerID string
	apiKey     string
	httpClient *http.Client
	userAgent  string
	currency   string
	maxRetries int
	maxWait    time.Duration
	readOnly   bool
	pacer      *pacer
	trace      []func(TraceEvent)
}

type service struct{ client *Client }

// Option configures a Client.
type Option func(*Client) error

// WithEnvironment selects Production (the default) or OTE.
func WithEnvironment(env Environment) Option {
	return func(c *Client) error {
		if env != Production && env != OTE {
			return fmt.Errorf("dna: unknown environment %q", env)
		}
		return c.setBaseURL(env.BaseURL())
	}
}

// WithBaseURL sends requests to a custom base URL, for a proxy or a test
// server.
func WithBaseURL(raw string) Option {
	return func(c *Client) error { return c.setBaseURL(raw) }
}

// WithHTTPClient uses hc for every request. The client copies it and turns
// off redirect following: the API answers 302 for an unknown or suspended
// reseller ID, which becomes an authentication error.
func WithHTTPClient(hc *http.Client) Option {
	return func(c *Client) error {
		if hc == nil {
			return errors.New("dna: WithHTTPClient: nil *http.Client")
		}
		cp := *hc
		cp.CheckRedirect = noRedirect
		c.httpClient = &cp
		return nil
	}
}

// WithTimeout sets the timeout of each HTTP attempt (default 60s). Keep it
// well above 5s: when the soft quota is exhausted the API delays responses
// by up to 5s instead of rejecting them.
func WithTimeout(d time.Duration) Option {
	return func(c *Client) error {
		if d < 0 {
			return errors.New("dna: WithTimeout: negative timeout")
		}
		c.httpClient.Timeout = d
		return nil
	}
}

// WithUserAgent puts your product in front of the User-Agent header, e.g.
// "my-panel/1.4". It helps DomainNameAPI support find your requests.
func WithUserAgent(product string) Option {
	return func(c *Client) error {
		if product = strings.TrimSpace(product); product != "" {
			c.userAgent = product + " " + UserAgent
		}
		return nil
	}
}

// WithCurrency sets the currency ("USD", "TRY", ...) of the prices returned
// by Domains.Search, Domains.BulkSearch, Products.TLDs and Products.SSLs
// when a request does not name one. Without it the API uses the account's
// currency.
func WithCurrency(code string) Option {
	return func(c *Client) error {
		c.currency = strings.ToUpper(strings.TrimSpace(code))
		return nil
	}
}

// WithRetries sets how often a failed request is retried (default 2).
// Rate-limited requests (429) are always retried: the API rejects them
// before doing anything. Network errors and 502/503/504 are retried only
// for requests that are safe to repeat, so a registration is never sent
// twice.
func WithRetries(n int) Option {
	return func(c *Client) error {
		if n < 0 {
			n = 0
		}
		c.maxRetries = n
		return nil
	}
}

// WithPacing turns client-side request pacing on (the default) or off.
// Pacing spaces requests to stay under the per-key rate limits: 5 requests
// per second overall, 2 per second for domain search, and one DNS zone call
// per second.
func WithPacing(enabled bool) Option {
	return func(c *Client) error {
		c.pacer = nil
		if enabled {
			c.pacer = newPacer()
		}
		return nil
	}
}

// WithReadOnly blocks every request that would change something. Blocked
// requests return an error that wraps ErrReadOnly; nothing is sent. Use it
// while exploring, in reporting jobs and in tests against production.
func WithReadOnly() Option {
	return func(c *Client) error { c.readOnly = true; return nil }
}

// WithTrace calls fn for every request, response and retry. Use it for
// logging, metrics or error reporting. Events never contain the API key.
// fn runs on the goroutine that sent the request and must be fast.
func WithTrace(fn func(TraceEvent)) Option {
	return func(c *Client) error {
		if fn != nil {
			c.trace = append(c.trace, fn)
		}
		return nil
	}
}

// WithLogger logs every exchange to l: requests and responses at debug
// level, retries at warn level and failures at error level.
func WithLogger(l *slog.Logger) Option {
	return WithTrace(func(ev TraceEvent) {
		if l == nil {
			return
		}
		attrs := []any{
			slog.String("method", ev.Method), slog.String("url", ev.URL),
			slog.Int("attempt", ev.Attempt), slog.String("correlation_id", ev.CorrelationID),
		}
		switch ev.Kind {
		case TraceRequest:
			l.Debug("dna: request", attrs...)
		case TraceRetry:
			l.Warn("dna: retrying", append(attrs, slog.Int("status", ev.StatusCode), slog.Duration("wait", ev.Wait), slog.Any("error", ev.Err))...)
		case TraceResponse:
			attrs = append(attrs, slog.Int("status", ev.StatusCode), slog.Duration("duration", ev.Duration))
			switch {
			case ev.Err != nil:
				l.Error("dna: request failed", append(attrs, slog.Any("error", ev.Err))...)
			case ev.StatusCode >= 400:
				l.Error("dna: error response", attrs...)
			default:
				l.Debug("dna: response", attrs...)
			}
		}
	})
}

// NewClient creates a client for a reseller ID and API key.
//
// Find both in the DomainNameAPI reseller panel under API settings: the
// reseller ID is a UUID, and production and OTE have different API keys.
// Panel login details do not work. Surrounding whitespace is trimmed.
//
//	client, err := dna.NewClient(resellerID, apiKey, dna.WithEnvironment(dna.OTE))
func NewClient(resellerID, apiKey string, opts ...Option) (*Client, error) {
	c := &Client{
		resellerID: strings.TrimSpace(resellerID),
		apiKey:     strings.TrimSpace(apiKey),
		httpClient: &http.Client{Timeout: 60 * time.Second, CheckRedirect: noRedirect},
		userAgent:  UserAgent,
		maxRetries: 2,
		maxWait:    30 * time.Second,
		pacer:      newPacer(),
	}
	if c.resellerID == "" || c.apiKey == "" {
		return nil, ErrNoCredentials
	}
	if err := c.setBaseURL(ProductionURL); err != nil {
		return nil, err
	}
	for _, opt := range opts {
		if opt == nil {
			continue
		}
		if err := opt(c); err != nil {
			return nil, err
		}
	}
	c.Account = (*AccountService)(&service{c})
	c.Contacts = (*ContactService)(&service{c})
	c.Domains = (*DomainService)(&service{c})
	c.DNS = (*DNSService)(&service{c})
	c.DNSSEC = (*DNSSECService)(&service{c})
	c.SSL = (*SSLService)(&service{c})
	c.Hosting = (*HostingService)(&service{c})
	c.Orders = (*OrderService)(&service{c})
	c.Products = (*ProductService)(&service{c})
	return c, nil
}

// NewClientFromEnv creates a client from the environment variables the dna
// CLI uses: DNA_RESELLER_ID, DNA_API_KEY, DNA_ENV ("production" or "ote"; or
// DNA_TEST_MODE=true) and DNA_BASE_URL. Options passed here win over the
// environment.
func NewClientFromEnv(opts ...Option) (*Client, error) {
	var envOpts []Option
	mode := os.Getenv(EnvEnv)
	if mode == "" {
		mode = os.Getenv(EnvTestMode)
	}
	env, err := ParseEnvironment(mode)
	if err != nil {
		return nil, err
	}
	envOpts = append(envOpts, WithEnvironment(env))
	if u := strings.TrimSpace(os.Getenv(EnvBaseURL)); u != "" {
		envOpts = append(envOpts, WithBaseURL(u))
	}
	resellerID, apiKey := os.Getenv(EnvResellerID), os.Getenv(EnvAPIKey)
	if strings.TrimSpace(resellerID) == "" || strings.TrimSpace(apiKey) == "" {
		return nil, fmt.Errorf("%w: %s and %s must both be set", ErrNoCredentials, EnvResellerID, EnvAPIKey)
	}
	return NewClient(resellerID, apiKey, append(envOpts, opts...)...)
}

// BaseURL returns the base URL requests go to.
func (c *Client) BaseURL() string { return c.baseURL.String() }

// Environment returns OTE when the client talks to the test environment and
// Production otherwise (also for custom base URLs).
func (c *Client) Environment() Environment {
	if c.baseURL.String() == OTEURL {
		return OTE
	}
	return Production
}

// ReadOnly reports whether the client was created WithReadOnly.
func (c *Client) ReadOnly() bool { return c.readOnly }

func (c *Client) setBaseURL(raw string) error {
	u, err := url.Parse(strings.TrimRight(strings.TrimSpace(raw), "/"))
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return fmt.Errorf("dna: invalid base URL %q (want e.g. %s)", raw, ProductionURL)
	}
	c.baseURL = u
	return nil
}

func noRedirect(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
