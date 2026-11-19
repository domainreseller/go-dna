package dna

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// ErrorKind is a coarse, stable classification of failures. Branch on it
// (or on the helpers IsNotFound, IsAuth, ...) instead of parsing messages.
type ErrorKind string

const (
	// KindAuth: wrong or missing credentials, the key of the other
	// environment, or an unknown or suspended reseller (HTTP 401 or a redirect).
	KindAuth ErrorKind = "auth"
	// KindNotFound: the object does not exist or is not yours (HTTP 404).
	KindNotFound ErrorKind = "not_found"
	// KindValidation: the API rejected the input (HTTP 400, 422).
	KindValidation ErrorKind = "validation"
	// KindRejected: a business rule refused the operation (HTTP 402, 403:
	// insufficient balance, transfer lock, TLD rules, ...), or the API answered
	// 2xx with success: false (OperationError).
	KindRejected ErrorKind = "rejected"
	// KindConflict: the object is in a conflicting state (HTTP 409).
	KindConflict ErrorKind = "conflict"
	// KindRateLimited: still rate limited after the automatic retries (HTTP 429).
	KindRateLimited ErrorKind = "rate_limited"
	// KindServer: the API failed (HTTP 5xx); often transient.
	KindServer ErrorKind = "server"
	// KindNetwork: the request did not complete: DNS, TLS, timeout, reset.
	KindNetwork ErrorKind = "network"
	// KindInvalidInput: Validate rejected the request; nothing was sent.
	KindInvalidInput ErrorKind = "invalid_input"
	// KindReadOnly: the client was created WithReadOnly and the request
	// would change something; nothing was sent.
	KindReadOnly ErrorKind = "read_only"
	// KindUnknown: anything else.
	KindUnknown ErrorKind = "unknown"
)

// Codes this package sets in APIError.Code when the API sends none.
const (
	CodeInvalidReseller       = "INVALID_RESELLER"        // the reseller ID is unknown, inactive or suspended in this environment
	CodeInvalidAPIKey         = "INVALID_API_KEY"         // the key does not match (production and OTE keys differ)
	CodeMissingAPIKey         = "MISSING_API_KEY"         // the X-API-KEY header was empty
	CodeMissingResellerID     = "MISSING_RESELLER_ID"     // the __reseller header was empty
	CodeSubresellerNotAllowed = "SUBRESELLER_NOT_ALLOWED" // sub-reseller accounts cannot use the API
	CodeUnauthorized          = "UNAUTHORIZED"            // any other 401
	CodeRateLimited           = "RATE_LIMITED"            // 429
	// Otherwise the code is "HTTP_<status>", e.g. HTTP_502.
)

// Error codes the API sends. The full catalogue is at
// https://www.domainnameapi.com/docs/reference/errors.md.
const (
	CodeDomainNotFound          = "Dna.DomainService:Domain:10007" // not found, or not yours
	CodePrivacyNotSupported     = "Dna.DomainService:Domain:10040" // the TLD has no WHOIS privacy
	CodeForwardingNotFound      = "Dna.DomainService:Domain:10061" // the domain has no URL forwarding
	CodeRegisteredNotRecorded   = "Dna.DomainService:Domain:10058" // registered at the registry, not recorded yet
	CodeRenewedNotRecorded      = "Dna.DomainService:Domain:10059" // renewed at the registry, not recorded yet
	CodeTransferNotRecorded     = "Dna.DomainService:Domain:10060" // transfer started at the registry, not recorded yet
	CodeSSLSentToCA             = "Dna.SslService:Ssl:10053"       // sent to the certificate authority
	CodeSSLOutcomeUnverified    = "Dna.SslService:Ssl:10054"       // the certificate authority's answer is unknown
	CodeVerificationNotPending  = "Dna.DomainService:Contact:Verification:10005"
	CodeVerificationAlreadyDone = "Dna.DomainService:Contact:Verification:10003"
)

// outcomeUnknownCodes mean "the registry or CA may already have done it":
// never repeat the request before checking the object's state.
var outcomeUnknownCodes = map[string]bool{
	CodeRegisteredNotRecorded: true,
	CodeRenewedNotRecorded:    true,
	CodeTransferNotRecorded:   true,
	CodeSSLSentToCA:           true,
	CodeSSLOutcomeUnverified:  true,
}

var (
	// ErrNoCredentials is returned by NewClient when the reseller ID or the
	// API key is empty.
	ErrNoCredentials = errors.New("dna: reseller ID and API key are required (find both in the reseller panel under API settings, or set DNA_RESELLER_ID and DNA_API_KEY)")
	// ErrReadOnly is wrapped by the error a read-only client returns for a
	// request that would change something. Test with errors.Is.
	ErrReadOnly = errors.New("dna: blocked by read-only mode")
)

// FieldError is one problem with one input field.
type FieldError struct {
	// Field is the input field, e.g. "domainName" or "Registrant.Country".
	Field string `json:"field,omitempty"`
	// Message says what is wrong.
	Message string `json:"message"`
}

func (f FieldError) String() string {
	if f.Field == "" {
		return f.Message
	}
	return f.Field + ": " + f.Message
}

// APIError is returned for every response with a non-2xx status.
type APIError struct {
	// StatusCode is the HTTP status.
	StatusCode int
	// Code is the API's error code, e.g. "Dna.DomainService:Domain:10007",
	// or one of the Code* constants of this package when the API sends none.
	// Branch on Code, not on Message: messages are localised.
	Code string
	// Message is the API's message.
	Message string
	// Details is the API's longer explanation, when it sends one.
	Details string
	// Fields lists per-field problems of a validation error.
	Fields []FieldError
	// Data holds extra facts some errors carry, e.g. the balances on an
	// insufficient-balance error.
	Data map[string]any
	// CorrelationID identifies the request; quote it when contacting support.
	CorrelationID string
	// RetryAfter is the wait the API asked for on a 429.
	RetryAfter time.Duration
	// Method and Path identify the request.
	Method, Path string
	// Body is the raw response body.
	Body []byte
}

func (e *APIError) Error() string {
	var sb strings.Builder
	sb.WriteString("dna: ")
	if e.Method != "" {
		sb.WriteString(e.Method + " " + e.Path + ": ")
	}
	fmt.Fprintf(&sb, "%d %s: %s", e.StatusCode, e.Code, e.Message)
	for _, f := range e.Fields {
		sb.WriteString("; " + f.String())
	}
	return sb.String()
}

// Kind classifies the error by its HTTP status.
func (e *APIError) Kind() ErrorKind {
	switch s := e.StatusCode; {
	case s == http.StatusUnauthorized, s == http.StatusFound, s == http.StatusMovedPermanently:
		return KindAuth
	case s == http.StatusNotFound:
		return KindNotFound
	case s == http.StatusBadRequest, s == http.StatusUnprocessableEntity:
		return KindValidation
	case s == http.StatusPaymentRequired, s == http.StatusForbidden:
		return KindRejected
	case s == http.StatusConflict:
		return KindConflict
	case s == http.StatusTooManyRequests:
		return KindRateLimited
	case s >= 500:
		return KindServer
	}
	return KindUnknown
}

// Retryable reports whether sending the same request again later may
// succeed: rate limits and server errors, except when the API says the
// operation may already have happened (see OutcomeUnknown).
func (e *APIError) Retryable() bool {
	if e.OutcomeUnknown() {
		return false
	}
	k := e.Kind()
	return k == KindRateLimited || (k == KindServer && e.StatusCode != http.StatusNotImplemented)
}

// OutcomeUnknown reports whether the API said the registry or certificate
// authority may already have carried out the operation. Check the object's
// state (Domains.Info, SSL.Status, Account.Transactions) before repeating it.
func (e *APIError) OutcomeUnknown() bool { return outcomeUnknownCodes[e.Code] }

// Hint returns a short suggestion for fixing the error, or "".
func (e *APIError) Hint() string {
	switch e.Code {
	case CodeInvalidReseller:
		return "The reseller ID is unknown, inactive or suspended in this environment. Check the reseller ID (a UUID from the panel's API settings) and the environment (WithEnvironment)."
	case CodeInvalidAPIKey:
		return "Production and OTE use different API keys. Make sure the key belongs to the environment the client uses."
	case CodeMissingAPIKey, CodeMissingResellerID:
		return "Pass both the reseller ID and the API key to NewClient."
	case CodeSubresellerNotAllowed:
		return "Sub-reseller accounts cannot use the API. Use the main reseller's credentials."
	case CodeRateLimited:
		return "The API allows 5 requests/second per key (2/s for domain search). Slow down, or keep pacing on (the default)."
	}
	if e.OutcomeUnknown() {
		return "The operation may already have been carried out. Check the object's state before sending the request again."
	}
	switch e.Kind() {
	case KindNotFound:
		return "Check the spelling and the environment: production and OTE hold different domains. Domains.List shows what this account holds."
	case KindServer:
		if e.CorrelationID != "" {
			return "Server-side failure. If it persists, contact support and quote correlation ID " + e.CorrelationID + "."
		}
	}
	return ""
}

// NetworkError is returned when a request did not complete: DNS, TLS,
// proxy, timeout, connection reset or a cancelled context.
type NetworkError struct {
	// Method and Path identify the request.
	Method, Path string
	// Err is the underlying error.
	Err error

	safety Safety
}

func (e *NetworkError) Error() string {
	return fmt.Sprintf("dna: %s %s: %v", e.Method, e.Path, e.Err)
}

func (e *NetworkError) Unwrap() error { return e.Err }

// Timeout reports whether the request timed out.
func (e *NetworkError) Timeout() bool {
	var ne net.Error
	return errors.As(e.Err, &ne) && ne.Timeout()
}

// OutcomeUnknown reports whether the request may have reached the API
// before it failed. Reads are harmless to repeat; check the state before
// repeating anything else.
func (e *NetworkError) OutcomeUnknown() bool { return e.safety != Read }

// InputError is returned when a request's Validate method rejects it.
// Nothing was sent.
type InputError struct {
	// Request is the request type, e.g. "DomainRenewRequest".
	Request string
	// Fields lists the problems.
	Fields []FieldError
}

func (e *InputError) Error() string {
	parts := make([]string, len(e.Fields))
	for i, f := range e.Fields {
		parts[i] = f.String()
	}
	name := e.Request
	if name == "" {
		name = "request"
	}
	return "dna: invalid " + name + ": " + strings.Join(parts, "; ")
}

// OperationError is returned when the API answered 2xx but reported that
// the operation did not happen ("success": false), for example a
// registration of a domain that is not available.
type OperationError struct {
	// Method and Path identify the request.
	Method, Path string
	// Code is the API's result code, when it sends one.
	Code string
	// Message is the API's explanation.
	Message string
	// CorrelationID identifies the request; quote it when contacting support.
	CorrelationID string
}

func (e *OperationError) Error() string {
	msg := e.Message
	if msg == "" {
		msg = "the API reported that the operation failed"
	}
	if e.Code != "" {
		msg = e.Code + ": " + msg
	}
	return fmt.Sprintf("dna: %s %s: %s", e.Method, e.Path, msg)
}

// KindOf classifies any error returned by this package. It returns "" for
// nil and KindUnknown for errors of other packages.
func KindOf(err error) ErrorKind {
	if err == nil {
		return ""
	}
	var (
		ae *APIError
		ne *NetworkError
		ie *InputError
		oe *OperationError
	)
	switch {
	case errors.As(err, &ae):
		return ae.Kind()
	case errors.As(err, &ne):
		return KindNetwork
	case errors.As(err, &ie):
		return KindInvalidInput
	case errors.As(err, &oe):
		return KindRejected
	case errors.Is(err, ErrReadOnly):
		return KindReadOnly
	case errors.Is(err, ErrNoCredentials):
		return KindAuth
	}
	return KindUnknown
}

// IsNotFound reports whether err means the object does not exist (or is not
// yours).
func IsNotFound(err error) bool { return KindOf(err) == KindNotFound }

// IsAuth reports whether err is an authentication failure.
func IsAuth(err error) bool { return KindOf(err) == KindAuth }

// IsRateLimited reports whether err is a rate limit the retries could not
// absorb.
func IsRateLimited(err error) bool { return KindOf(err) == KindRateLimited }

// IsInsufficientBalance reports whether err says the balance is too low.
func IsInsufficientBalance(err error) bool {
	var ae *APIError
	if !errors.As(err, &ae) {
		return false
	}
	return ae.StatusCode == http.StatusPaymentRequired ||
		ae.Code == "Dna.DomainService:Deposit:10001" || ae.Code == "Dna.DepositService:Deposit:10002"
}

// HasCode reports whether err is an *APIError or *OperationError with the
// given code.
func HasCode(err error, code string) bool {
	var ae *APIError
	if errors.As(err, &ae) {
		return ae.Code == code
	}
	var oe *OperationError
	return errors.As(err, &oe) && oe.Code == code
}

// OutcomeUnknown reports whether a failed request may still have been
// carried out: a write that failed in transport, or an API answer that says
// so. Check the state before sending such a request again; repeating a
// registration or renewal blindly can charge twice.
func OutcomeUnknown(err error) bool {
	var ae *APIError
	if errors.As(err, &ae) {
		return ae.OutcomeUnknown()
	}
	var ne *NetworkError
	return errors.As(err, &ne) && ne.OutcomeUnknown()
}

// ---------------------------------------------------------------------------
// Response normalisation

// boilerplate matches the generic .NET exception text the API sometimes
// sends instead of a useful message.
var boilerplate = regexp.MustCompile(`(?i)exception of type '[^']*' was thrown`)

type errorInfo struct {
	Code             *string         `json:"code"`
	Message          string          `json:"message"`
	Details          string          `json:"details"`
	Data             json.RawMessage `json:"data"` // an object, or [] when empty
	ValidationErrors []struct {
		Message string   `json:"message"`
		Members []string `json:"members"`
	} `json:"validationErrors"`
}

type errorEnvelope struct {
	Error *errorInfo `json:"error"`
	// Some endpoints answer with a flat object.
	errorInfo
}

func newAPIError(method, path string, status int, header http.Header, body []byte) *APIError {
	e := &APIError{
		StatusCode:    status,
		Method:        method,
		Path:          path,
		Body:          body,
		CorrelationID: header.Get("X-Correlation-Id"),
	}
	text := strings.TrimSpace(string(body))

	switch status {
	case http.StatusFound, http.StatusMovedPermanently:
		// The gateway redirects to /Error/Suspend when the __reseller value
		// is unknown, inactive or suspended.
		e.Code = CodeInvalidReseller
		e.Message = "authentication failed: the reseller ID is unknown, inactive or suspended"
		return e
	case http.StatusTooManyRequests:
		e.Code = CodeRateLimited
		e.Message = "rate limit exceeded"
		if s := header.Get("Retry-After"); s != "" {
			if n, err := strconv.ParseFloat(s, 64); err == nil {
				e.RetryAfter = time.Duration(n * float64(time.Second))
				e.Message += " (retry after " + s + "s)"
			}
		}
		return e
	}

	var env errorEnvelope
	if strings.HasPrefix(text, "{") && json.Unmarshal(body, &env) == nil {
		info := &env.errorInfo
		if env.Error != nil {
			info = env.Error
		}
		if info.Code != nil {
			e.Code = *info.Code
		}
		e.Message, e.Details = info.Message, info.Details
		if d := bytes.TrimSpace(info.Data); len(d) > 0 && d[0] == '{' {
			_ = json.Unmarshal(d, &e.Data)
		}
		for _, v := range info.ValidationErrors {
			members := make([]string, len(v.Members))
			for i, m := range v.Members {
				members[i] = strings.TrimPrefix(m, "$.")
			}
			e.Fields = append(e.Fields, FieldError{Field: strings.Join(members, ","), Message: v.Message})
		}
	} else if text != "" && !strings.HasPrefix(text, "<") {
		// 401 bodies are plain text; HTML error pages are dropped.
		e.Message = text
	}

	if status == http.StatusUnauthorized && e.Code == "" {
		lower := strings.ToLower(e.Message)
		switch {
		case strings.Contains(lower, "api key is missing"):
			e.Code = CodeMissingAPIKey
		case strings.Contains(lower, "resellerid is missing"):
			e.Code = CodeMissingResellerID
		case strings.Contains(lower, "invalid api key"):
			e.Code = CodeInvalidAPIKey
		case strings.Contains(lower, "authorisation"), strings.Contains(lower, "authorization"):
			e.Code = CodeSubresellerNotAllowed
		default:
			e.Code = CodeUnauthorized
		}
	}

	// Replace boilerplate and code echoes with the details text.
	if boilerplate.MatchString(e.Message) || (e.Message != "" && e.Message == e.Code) {
		d := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(e.Details), "Dna "))
		if d != "" && !boilerplate.MatchString(d) && d != e.Code {
			e.Message = d
		} else {
			e.Message = "the operation could not be completed"
		}
	}
	if e.Code == "" {
		e.Code = "HTTP_" + strconv.Itoa(status)
	}
	if strings.TrimSpace(e.Message) == "" {
		e.Message = strings.ToLower(http.StatusText(status))
		if e.Message == "" {
			e.Message = "unexpected status"
		}
	}
	return e
}
