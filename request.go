package dna

import (
	"net/url"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Safety says what a request does to your account. The client uses it to
// decide what may be retried and what WithReadOnly blocks.
type Safety int

const (
	// Read requests have no side effects, even when they use POST or PUT
	// (DomainSearchRequest is a POST, SSLStatusRequest a PUT).
	Read Safety = iota
	// Write requests change something, free of charge and reversibly.
	Write
	// Destructive requests delete something or are hard to undo.
	Destructive
	// Billable requests charge the reseller balance.
	Billable
)

func (s Safety) String() string {
	switch s {
	case Read:
		return "read"
	case Write:
		return "write"
	case Destructive:
		return "destructive"
	case Billable:
		return "billable"
	}
	return "Safety(" + strconv.Itoa(int(s)) + ")"
}

// Endpoint is where a request goes and what kind of request it is.
type Endpoint struct {
	// Method is the HTTP method: GET, POST, PUT or DELETE.
	Method string
	// Path is the path below the base URL, already escaped,
	// e.g. "/api/v1/domains/info".
	Path string
	// Safety classifies the request.
	Safety Safety
	// Idempotent marks a non-read request that is safe to send again after
	// a network error or a 5xx response (the same nameservers set twice
	// give the same result). Read requests are always idempotent.
	Idempotent bool
}

// Request is one call to the DomainNameAPI REST API.
//
// Every XxxRequest type in this package implements it. They all embed
// BaseRequest, which supplies Query, Body and Validate, and define their own
// Endpoint plus whichever of the other three they need. Your own types can do
// the same to call an endpoint this package does not wrap yet:
//
//	type PingRequest struct {
//		dna.BaseRequest
//	}
//
//	func (PingRequest) Endpoint() dna.Endpoint {
//		return dna.Endpoint{Method: http.MethodGet, Path: "/api/v1/deposit/accounts/me"}
//	}
//
//	var out map[string]any
//	_, err := client.Do(ctx, PingRequest{}, &out)
type Request interface {
	// Endpoint returns the method, path and safety class.
	Endpoint() Endpoint
	// Query returns the query string parameters, or nil.
	Query() url.Values
	// Body returns the value to send as the JSON body, or nil for none.
	Body() any
	// Validate checks the request before it is sent. A non-nil error stops
	// the request; nothing reaches the API.
	Validate() error
}

// BaseRequest is the base every request type embeds. It provides the
// default behaviour (no query string, no body, nothing to validate); a
// request overrides a method by defining it on its own type. Only Endpoint
// has no default: each request must say where it goes.
type BaseRequest struct{}

// Query returns nil: no query string.
func (BaseRequest) Query() url.Values { return nil }

// Body returns nil: no request body.
func (BaseRequest) Body() any { return nil }

// Validate returns nil: nothing to check.
func (BaseRequest) Validate() error { return nil }

// ListOptions pages through list endpoints. The zero value asks for the
// first page with the API's default page size.
type ListOptions struct {
	// Skip is the number of items to skip (SkipCount).
	Skip int
	// Limit is the page size (MaxResultCount); 0 uses the API default.
	Limit int
	// Sort is a sort expression such as "expirationDate desc" (Sorting).
	Sort string
}

func (o ListOptions) validate(v *validator) {
	v.check(o.Skip >= 0, "skip", "must not be negative")
	v.check(o.Limit >= 0, "limit", "must not be negative")
}

// object is a JSON request body. Keys given in the literal are always sent;
// set adds a key only when the value is not empty.
type object map[string]any

func (o object) set(key string, v any) object {
	switch x := v.(type) {
	case nil:
		return o
	case string:
		if strings.TrimSpace(x) == "" {
			return o
		}
		v = strings.TrimSpace(x)
	case int:
		if x == 0 {
			return o
		}
	}
	switch rv := reflect.ValueOf(v); rv.Kind() {
	case reflect.Slice, reflect.Map:
		if rv.Len() == 0 {
			return o
		}
	case reflect.Pointer:
		if rv.IsNil() {
			return o
		}
	}
	o[key] = v
	return o
}

// domainBody is the body of the many requests that only name a domain.
func domainBody(name string) object { return object{"domainName": trim(name)} }

// certificateBody is the body of the requests that name an SSL certificate.
func certificateBody(id string) object { return object{"certificateId": trim(id)} }

// params builds a query string, skipping empty values so the API sees only
// what the caller set.
type params struct{ url.Values }

func newParams() params { return params{url.Values{}} }

func (p params) str(key, v string) params {
	if v = strings.TrimSpace(v); v != "" {
		p.Set(key, v)
	}
	return p
}

func (p params) num(key string, v int) params {
	if v != 0 {
		p.Set(key, strconv.Itoa(v))
	}
	return p
}

func (p params) float(key string, v float64) params {
	if v != 0 {
		p.Set(key, strconv.FormatFloat(v, 'f', -1, 64))
	}
	return p
}

func (p params) boolean(key string, v *bool) params {
	if v != nil {
		p.Set(key, strconv.FormatBool(*v))
	}
	return p
}

func (p params) date(key string, t time.Time) params {
	if !t.IsZero() {
		p.Set(key, t.UTC().Format(time.RFC3339))
	}
	return p
}

// each adds the key once per value: TransactionTypes=1&TransactionTypes=2.
func (p params) each(key string, values []string) params {
	for _, v := range values {
		if v = strings.TrimSpace(v); v != "" {
			p.Add(key, v)
		}
	}
	return p
}

// joined adds the values as one comma-separated value: StatusIds=0,7.
func (p params) joined(key string, values []string) params {
	var parts []string
	for _, v := range values {
		if v = strings.TrimSpace(v); v != "" {
			parts = append(parts, v)
		}
	}
	if len(parts) > 0 {
		p.Set(key, strings.Join(parts, ","))
	}
	return p
}

func (p params) page(o ListOptions) params {
	return p.num("SkipCount", o.Skip).num("MaxResultCount", o.Limit).str("Sorting", o.Sort)
}

// intStrings formats integer enum values for the query string.
func intStrings[E ~int](values []E) []string {
	out := make([]string, len(values))
	for i, v := range values {
		out[i] = strconv.Itoa(int(v))
	}
	return out
}

// validator collects field errors for a request's Validate method.
type validator struct{ fields []FieldError }

func (v *validator) add(field, msg string) {
	v.fields = append(v.fields, FieldError{Field: field, Message: msg})
}

func (v *validator) check(ok bool, field, msg string) {
	if !ok {
		v.add(field, msg)
	}
}

func (v *validator) required(field, value string) {
	v.check(strings.TrimSpace(value) != "", field, "is required")
}

func (v *validator) requiredList(field string, n int) {
	v.check(n > 0, field, "needs at least one value")
}

// between checks an optional number: 0 means "not set" and passes.
func (v *validator) between(field string, n, min, max int) {
	if n != 0 && (n < min || n > max) {
		v.add(field, "must be between "+strconv.Itoa(min)+" and "+strconv.Itoa(max))
	}
}

func (v *validator) maxLen(field, value string, max int) {
	if len([]rune(value)) > max {
		v.add(field, "must be at most "+strconv.Itoa(max)+" characters")
	}
}

// certificateIDPattern is a GUID: 8-4-4-4-12 hexadecimal digits.
var certificateIDPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}(-[0-9a-fA-F]{4}){3}-[0-9a-fA-F]{12}$`)

// certificateID checks the id of an SSL certificate. The API used to address
// certificates by domain name, and both are strings, so a domain name passed
// by an old caller is caught here instead of reaching the API.
func (v *validator) certificateID(field, value string) {
	switch value = strings.TrimSpace(value); {
	case value == "":
		v.add(field, "is required: pass the CertificateID returned by SSL.Initialize or SSL.Renew, or look it up with SSL.List")
	case !certificateIDPattern.MatchString(value):
		v.add(field, "must be a certificate id (GUID). Certificates are no longer addressed by domain name: "+
			"use the CertificateID returned by SSL.Initialize or SSL.Renew, or look it up with SSL.List(ctx, SSLListRequest{DomainName: domainName})")
	}
}

// requireCertificateID validates a request whose only input is a certificate id.
func requireCertificateID(id string) error {
	var v validator
	v.certificateID("certificateId", id)
	return v.err()
}

func (v *validator) err() error {
	if len(v.fields) == 0 {
		return nil
	}
	return &InputError{Fields: v.fields}
}

func trim(s string) string { return strings.TrimSpace(s) }

func trimAll(list []string) []string {
	out := make([]string, 0, len(list))
	for _, s := range list {
		if s = strings.TrimSpace(s); s != "" {
			out = append(out, s)
		}
	}
	return out
}
