package dna

import (
	"context"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// DNSService manages the DNS zone of domains on the platform nameservers.
// CLI: dna dns ...
//
// A record set is all records with the same name and type; it has one TTL,
// and a record is identified by its value. Changes are live at once in the
// platform's DNS, but resolve publicly only when the domain uses the
// platform nameservers (see Apply).
//
// The test environment (OTE) answers every zone call with HTTP 500, so
// zone code can only be tried in production. Use a domain you do not
// mind changing.
type DNSService service

// Record is one value of a record set.
type Record struct {
	Content  string `json:"content"`
	Disabled bool   `json:"disabled"`
}

// RecordSet is the records with one name and type.
type RecordSet struct {
	// Name is fully qualified with a trailing dot, e.g. "www.example.com.".
	Name       string     `json:"name"`
	Type       RecordType `json:"type"`
	TTL        int        `json:"ttl"`
	ChangeType string     `json:"changeType"`
	Records    []Record   `json:"records"`
}

// Values returns the contents of the set's records.
func (s RecordSet) Values() []string {
	out := make([]string, len(s.Records))
	for i, r := range s.Records {
		out[i] = r.Content
	}
	return out
}

// Matches reports whether the set has the given type and name. name may be
// relative ("www"), the apex ("", "@") or fully qualified.
func (s RecordSet) Matches(domainName, name string, t RecordType) bool {
	return strings.EqualFold(string(s.Type), string(t.normalize())) &&
		strings.EqualFold(strings.TrimSuffix(s.Name, "."), strings.TrimSuffix(FQDN(domainName, name), "."))
}

// FQDN returns the fully qualified form of a record name, the form the zone
// listing uses: FQDN("example.com", "www") is "www.example.com.", and ""
// and "@" give "example.com.".
func FQDN(domainName, name string) string {
	domain := strings.ToLower(strings.TrimSuffix(strings.TrimSpace(domainName), "."))
	name = strings.TrimSuffix(strings.TrimSpace(name), ".")
	switch lower := strings.ToLower(name); {
	case name == "" || name == "@" || lower == domain:
		return domain + "."
	case strings.HasSuffix(lower, "."+domain):
		return lower + "."
	}
	return strings.ToLower(name) + "." + domain + "."
}

// ZoneApplyResult reports DNSService.Apply.
type ZoneApplyResult struct {
	// NameServersChanged is false when the domain already used the
	// platform nameservers.
	NameServersChanged  bool     `json:"nameServersChanged"`
	PreviousNameServers []string `json:"previousNameServers"`
	CurrentNameServers  []string `json:"currentNameServers"`
	SystemNameServers   []string `json:"systemNameServers"`
}

// DefaultTTL is the TTL used when a request leaves it 0.
const DefaultTTL = 3600

// DNSListRequest lists the record sets of a domain. CLI: dna dns list
type DNSListRequest struct {
	BaseRequest
	DomainName string
}

func (DNSListRequest) Endpoint() Endpoint {
	return Endpoint{Method: http.MethodGet, Path: "/api/v1/domains/zones", Safety: Read}
}

func (r DNSListRequest) Query() url.Values {
	return newParams().str("domainName", r.DomainName).Values
}

func (r DNSListRequest) Validate() error { return requireDomain(r.DomainName) }

// DNSAddRequest adds values to a record set, creating the set when needed.
// Values already in the set stay. CLI: dna dns add
type DNSAddRequest struct {
	BaseRequest
	DomainName string
	// Name is relative ("www"), the apex ("" or "@") or fully qualified.
	Name string
	Type RecordType
	// TTL in seconds, 1 to 86400 (0 means DefaultTTL).
	TTL int
	// Values are the record contents, e.g. "192.0.2.10", "10 mail.example.com"
	// (MX) or `0 issue "letsencrypt.org"` (CAA). TXT values are quoted by
	// the API.
	Values []string
}

func (DNSAddRequest) Endpoint() Endpoint {
	return Endpoint{Method: http.MethodPost, Path: "/api/v1/domains/zones", Safety: Write}
}

func (r DNSAddRequest) Query() url.Values {
	return newParams().str("domainName", r.DomainName).Values
}

func (r DNSAddRequest) Body() any {
	return object{"zoneStruct": zoneStruct(r.Name, r.Type, r.TTL, r.Values)}
}

func (r DNSAddRequest) Validate() error {
	var v validator
	validateRecordSet(&v, r.DomainName, r.Name, r.Type, r.TTL, r.Values)
	return v.err()
}

// DNSUpdateRequest replaces every value of a record set (and its TTL); it
// can also rename the set. CLI: dna dns update
type DNSUpdateRequest struct {
	BaseRequest
	DomainName string
	// RecordName is the name of the existing set.
	RecordName string
	// Name is the new name; empty keeps RecordName.
	Name string
	Type RecordType
	// TTL in seconds, 1 to 86400 (0 means DefaultTTL).
	TTL int
	// Values replace all values of the set.
	Values []string
}

func (DNSUpdateRequest) Endpoint() Endpoint {
	return Endpoint{Method: http.MethodPut, Path: "/api/v1/domains/zones", Safety: Write, Idempotent: true}
}

func (r DNSUpdateRequest) Query() url.Values {
	return newParams().str("domainName", r.DomainName).str("recordName", r.RecordName).Values
}

func (r DNSUpdateRequest) newName() string {
	if trim(r.Name) == "" {
		return r.RecordName
	}
	return r.Name
}

func (r DNSUpdateRequest) Body() any {
	return object{"zoneStruct": zoneStruct(r.newName(), r.Type, r.TTL, r.Values)}
}

func (r DNSUpdateRequest) Validate() error {
	var v validator
	v.required("recordName", r.RecordName)
	validateRecordSet(&v, r.DomainName, r.newName(), r.Type, r.TTL, r.Values)
	return v.err()
}

// DNSDeleteRequest removes one value from a record set; the set's other
// values stay. CLI: dna dns delete
type DNSDeleteRequest struct {
	BaseRequest
	DomainName string
	Name       string
	Type       RecordType
	// Value is the exact record content to remove.
	Value string
}

func (DNSDeleteRequest) Endpoint() Endpoint {
	return Endpoint{Method: http.MethodDelete, Path: "/api/v1/domains/zones", Safety: Destructive}
}

func (r DNSDeleteRequest) Query() url.Values {
	p := newParams().str("domainName", r.DomainName).str("RecordType", string(r.Type.normalize())).str("Record", r.Value)
	p.Set("Name", recordName(r.Name)) // "" is the apex, and is sent
	return p.Values
}

func (r DNSDeleteRequest) Validate() error {
	var v validator
	v.required("domainName", r.DomainName)
	validateRecordType(&v, r.Type)
	v.required("value", r.Value)
	return v.err()
}

// DNSApplyRequest switches a domain to the platform nameservers so its
// zone is served. CLI: dna dns apply
type DNSApplyRequest struct {
	BaseRequest
	DomainName string
}

func (DNSApplyRequest) Endpoint() Endpoint {
	return Endpoint{Method: http.MethodPost, Path: "/api/v1/domains/zones/apply", Safety: Write, Idempotent: true}
}

func (r DNSApplyRequest) Query() url.Values {
	return newParams().str("domainName", r.DomainName).Values
}

func (r DNSApplyRequest) Validate() error { return requireDomain(r.DomainName) }

func recordName(name string) string {
	if name = trim(name); name == "@" {
		return ""
	}
	return name
}

func zoneStruct(name string, t RecordType, ttl int, values []string) object {
	if ttl == 0 {
		ttl = DefaultTTL
	}
	return object{"name": recordName(name), "type": t.normalize(), "ttl": ttl, "contents": trimAll(values)}
}

func validateRecordType(v *validator, t RecordType) {
	if trim(string(t)) == "" {
		v.add("type", "is required")
	} else if _, err := ParseRecordType(string(t)); err != nil {
		v.add("type", err.Error())
	}
}

func validateRecordSet(v *validator, domainName, name string, t RecordType, ttl int, values []string) {
	v.required("domainName", domainName)
	validateRecordType(v, t)
	v.between("ttl", ttl, 1, 86400)
	v.check(ttl >= 0, "ttl", "must not be negative")
	n := trim(name)
	v.check(n == "@" || !strings.HasPrefix(n, "@"), "name", "must not start with @ (use \"@\" alone for the apex)")
	apex := n == "" || n == "@" || strings.EqualFold(strings.TrimSuffix(n, "."), strings.TrimSuffix(trim(domainName), "."))
	contents := trimAll(values)
	v.requiredList("values", len(contents))
	switch t.normalize() {
	case RecordTypeCNAME:
		v.check(!apex, "name", "a CNAME cannot be at the apex; use an ALIAS record")
		for _, c := range contents {
			v.check(net.ParseIP(c) == nil, "values", "a CNAME points to a host name, not an IP address ("+c+")")
		}
	case RecordTypeMX:
		for _, c := range contents {
			f := strings.Fields(c)
			_, err := strconv.Atoi(firstField(f))
			v.check(len(f) == 2 && err == nil && net.ParseIP(f[1]) == nil, "values", "an MX value is \"PRIORITY HOST\", e.g. \"10 mail.example.com\" ("+c+")")
		}
	case RecordTypeCAA:
		for _, c := range contents {
			v.check(len(strings.Fields(c)) == 3, "values", "a CAA value has 3 parts, e.g. `0 issue \"letsencrypt.org\"` ("+c+")")
		}
	case RecordTypeA:
		for _, c := range contents {
			ip := net.ParseIP(c)
			v.check(ip != nil && ip.To4() != nil, "values", c+" is not an IPv4 address")
		}
	case RecordTypeAAAA:
		for _, c := range contents {
			ip := net.ParseIP(c)
			v.check(ip != nil && ip.To4() == nil, "values", c+" is not an IPv6 address")
		}
	}
}

func firstField(f []string) string {
	if len(f) == 0 {
		return ""
	}
	return f[0]
}

// List returns the record sets of a domain. A domain without a zone has
// none.
func (s *DNSService) List(ctx context.Context, domainName string) ([]RecordSet, error) {
	res, err := call[[]RecordSet](ctx, s.client, DNSListRequest{DomainName: domainName})
	if err != nil {
		return nil, err
	}
	return *res, nil
}

// Add adds values to a record set, creating it (and the zone) when needed.
//
//	err := client.DNS.Add(ctx, dna.DNSAddRequest{
//		DomainName: "example.com", Name: "www", Type: dna.RecordTypeA, Values: []string{"192.0.2.10"},
//	})
func (s *DNSService) Add(ctx context.Context, req DNSAddRequest) error {
	return exec(ctx, s.client, req)
}

// Update replaces all values (and the TTL) of a record set.
func (s *DNSService) Update(ctx context.Context, req DNSUpdateRequest) error {
	return exec(ctx, s.client, req)
}

// Delete removes one value from a record set.
func (s *DNSService) Delete(ctx context.Context, req DNSDeleteRequest) error {
	return exec(ctx, s.client, req)
}

// Apply switches the domain to the platform nameservers, so the records
// managed here become live. This replaces any other DNS provider of the
// domain. It does nothing when the domain already uses them.
func (s *DNSService) Apply(ctx context.Context, domainName string) (*ZoneApplyResult, error) {
	return call[ZoneApplyResult](ctx, s.client, DNSApplyRequest{DomainName: domainName})
}
