package dna

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestAPIErrorNormalisation(t *testing.T) {
	for _, tc := range []struct {
		name    string
		status  int
		header  http.Header
		body    string
		code    string
		message string
		kind    ErrorKind
	}{
		{
			name: "abp envelope", status: 404,
			body: `{"error":{"code":"Dna.DomainService:Domain:10007","message":"The domain name you requested could not be found.","details":"The domain could not be found.","data":[],"validationErrors":null}}`,
			code: CodeDomainNotFound, message: "The domain name you requested could not be found.", kind: KindNotFound,
		},
		{
			name: "flat object", status: 409, body: `{"code":"X:1","message":"busy"}`,
			code: "X:1", message: "busy", kind: KindConflict,
		},
		{
			name: "validation without code", status: 400,
			body: `{"error":{"code":null,"message":"Validation failed.","details":"...","validationErrors":[{"message":"The DomainName field is required.","members":["$.domainName"]}]}}`,
			code: "HTTP_400", message: "Validation failed.", kind: KindValidation,
		},
		{name: "plain-text missing key", status: 401, body: "Unauthorized! API Key is missing.", code: CodeMissingAPIKey, message: "Unauthorized! API Key is missing.", kind: KindAuth},
		{name: "plain-text missing reseller", status: 401, body: "Unauthorized! ResellerId is missing.", code: CodeMissingResellerID, kind: KindAuth},
		{name: "plain-text invalid key", status: 401, body: "Unauthorized! Invalid API Key.", code: CodeInvalidAPIKey, kind: KindAuth},
		{name: "sub-reseller", status: 401, body: "Unauthorized! You do not have authorisation for this operation.", code: CodeSubresellerNotAllowed, kind: KindAuth},
		{name: "empty 401", status: 401, code: CodeUnauthorized, message: "unauthorized", kind: KindAuth},
		{name: "redirect", status: 302, code: CodeInvalidReseller, kind: KindAuth},
		{name: "rate limited", status: 429, header: http.Header{"Retry-After": {"2"}}, code: CodeRateLimited, message: "rate limit exceeded (retry after 2s)", kind: KindRateLimited},
		{name: "empty 500", status: 500, code: "HTTP_500", message: "internal server error", kind: KindServer},
		{name: "html 502", status: 502, body: "<html><body>Bad gateway</body></html>", code: "HTTP_502", message: "bad gateway", kind: KindServer},
		{
			name: "boilerplate replaced by details", status: 400,
			body: `{"error":{"code":"Dna.DomainService:Dns:10024","message":"Exception of type 'Volo.Abp.BusinessException' was thrown.","details":"Dna Record already exists."}}`,
			code: "Dna.DomainService:Dns:10024", message: "Record already exists.", kind: KindValidation,
		},
		{
			name: "code echo without details", status: 400,
			body: `{"error":{"code":"Dna.DomainService:Dns:10030","message":"Dna.DomainService:Dns:10030","details":null}}`,
			code: "Dna.DomainService:Dns:10030", message: "the operation could not be completed", kind: KindValidation,
		},
		{name: "business rule", status: 403, body: `{"error":{"code":"","message":"Hosting already exists"}}`, code: "HTTP_403", message: "Hosting already exists", kind: KindRejected},
		{name: "payment required", status: 402, body: `{"error":{"code":"Dna.DomainService:Deposit:10001","message":"Insufficient balance","data":{"usdBalance":1.5,"requiredAmount":11.99}}}`, code: "Dna.DomainService:Deposit:10001", kind: KindRejected},
		{name: "teapot", status: 418, code: "HTTP_418", message: "i'm a teapot", kind: KindUnknown},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h := tc.header
			if h == nil {
				h = http.Header{}
			}
			e := newAPIError("GET", "/api/v1/x", tc.status, h, []byte(tc.body))
			if e.Code != tc.code {
				t.Errorf("Code = %q, want %q", e.Code, tc.code)
			}
			if tc.message != "" && e.Message != tc.message {
				t.Errorf("Message = %q, want %q", e.Message, tc.message)
			}
			if e.Message == "" {
				t.Error("Message is empty")
			}
			if e.Kind() != tc.kind || KindOf(e) != tc.kind {
				t.Errorf("Kind = %s, want %s", e.Kind(), tc.kind)
			}
		})
	}
}

func TestAPIErrorDetails(t *testing.T) {
	e := newAPIError("POST", "/api/v1/domains/search", 400, http.Header{"X-Correlation-Id": {"c-1"}},
		[]byte(`{"error":{"message":"Validation failed.","validationErrors":[{"message":"The DomainName field is required.","members":["$.domainName"]},{"message":"Bad.","members":["Registrant.Country","Admin.Country"]}]}}`))
	if len(e.Fields) != 2 || e.Fields[0].Field != "domainName" || e.Fields[1].Field != "Registrant.Country,Admin.Country" {
		t.Errorf("Fields = %+v", e.Fields)
	}
	if e.CorrelationID != "c-1" {
		t.Errorf("CorrelationID = %q", e.CorrelationID)
	}
	want := "dna: POST /api/v1/domains/search: 400 HTTP_400: Validation failed.; domainName: The DomainName field is required.; Registrant.Country,Admin.Country: Bad."
	if e.Error() != want {
		t.Errorf("Error() =\n%s\nwant\n%s", e.Error(), want)
	}

	e = newAPIError("GET", "/x", 429, http.Header{"Retry-After": {"1.5"}}, nil)
	if e.RetryAfter != 1500*time.Millisecond || !e.Retryable() || e.Hint() == "" {
		t.Errorf("429: %+v hint %q", e, e.Hint())
	}

	e = newAPIError("POST", "/x", 402, http.Header{}, []byte(`{"error":{"code":"Dna.DomainService:Deposit:10001","message":"x","data":{"usdBalance":1.5}}}`))
	if e.Data["usdBalance"] != 1.5 || !IsInsufficientBalance(e) {
		t.Errorf("402: %+v", e)
	}
	e = newAPIError("POST", "/x", 400, http.Header{}, []byte(`{"error":{"code":"Dna.DepositService:Deposit:10002","message":"Insufficient balance."}}`))
	if !IsInsufficientBalance(fmt.Errorf("wrapped: %w", e)) {
		t.Error("IsInsufficientBalance does not see the deposit code")
	}
}

func TestOutcomeUnknown(t *testing.T) {
	for _, code := range []string{CodeRegisteredNotRecorded, CodeRenewedNotRecorded, CodeTransferNotRecorded, CodeSSLSentToCA, CodeSSLOutcomeUnverified} {
		e := &APIError{StatusCode: 500, Code: code, Message: "x"}
		if !e.OutcomeUnknown() || !OutcomeUnknown(e) || e.Retryable() {
			t.Errorf("%s: OutcomeUnknown %v Retryable %v", code, e.OutcomeUnknown(), e.Retryable())
		}
		if !strings.Contains(e.Hint(), "already") {
			t.Errorf("%s: hint %q", code, e.Hint())
		}
	}
	if OutcomeUnknown(&APIError{StatusCode: 404, Code: CodeDomainNotFound}) {
		t.Error("not found has a known outcome")
	}
	if !OutcomeUnknown(&NetworkError{Err: context.DeadlineExceeded, safety: Billable}) {
		t.Error("a billable request that timed out has an unknown outcome")
	}
	if OutcomeUnknown(&NetworkError{Err: context.DeadlineExceeded, safety: Read}) || OutcomeUnknown(nil) {
		t.Error("a read has a known outcome")
	}
}

func TestKindOfAndHelpers(t *testing.T) {
	op := &OperationError{Method: "POST", Path: "/api/v1/domains/register", Code: "2302", Message: "Domain is not available"}
	for err, want := range map[error]ErrorKind{
		nil:                                 "",
		errors.New("other"):                 KindUnknown,
		&InputError{Request: "X"}:           KindInvalidInput,
		op:                                  KindRejected,
		&NetworkError{Err: errors.New("x")}: KindNetwork,
		fmt.Errorf("x: %w", ErrReadOnly):    KindReadOnly,
		ErrNoCredentials:                    KindAuth,
		fmt.Errorf("wrapped: %w", &APIError{StatusCode: 404}): KindNotFound,
	} {
		if got := KindOf(err); got != want {
			t.Errorf("KindOf(%v) = %q, want %q", err, got, want)
		}
	}
	if !HasCode(op, "2302") || HasCode(op, "x") || HasCode(errors.New("2302"), "2302") {
		t.Error("HasCode")
	}
	if op.Error() != "dna: POST /api/v1/domains/register: 2302: Domain is not available" {
		t.Errorf("OperationError.Error() = %q", op.Error())
	}
	if !IsNotFound(&APIError{StatusCode: 404}) || IsNotFound(&APIError{StatusCode: 400}) {
		t.Error("IsNotFound")
	}
	ne := &NetworkError{Method: "GET", Path: "/x", Err: context.DeadlineExceeded}
	if !ne.Timeout() || !errors.Is(ne, context.DeadlineExceeded) || ne.Error() != "dna: GET /x: context deadline exceeded" {
		t.Errorf("NetworkError: %v timeout=%v", ne, ne.Timeout())
	}
	ie := &InputError{Request: "DNSAddRequest", Fields: []FieldError{{Field: "type", Message: "is required"}, {Message: "general"}}}
	if ie.Error() != "dna: invalid DNSAddRequest: type: is required; general" {
		t.Errorf("InputError.Error() = %q", ie.Error())
	}
}
