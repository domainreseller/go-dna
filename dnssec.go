package dna

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// DNSSECService publishes and removes DS records at the registry.
// CLI: dna dnssec ...
type DNSSECService service

// DNSSECInfo is the DNSSEC state of a domain.
type DNSSECInfo struct {
	DomainName string `json:"domainName"`
	// Available reports whether the TLD supports DNSSEC.
	Available          bool              `json:"isDnssecAvailable"`
	Status             DNSSECStatus      `json:"status"`
	DelegationSigned   bool              `json:"delegationSigned"`
	MaxSigLife         int               `json:"maxSigLife"`
	LastChangedAt      Time              `json:"lastChangedAt"`
	DSRecords          []DSRecord        `json:"dsRecords"`
	AllowedAlgorithms  []DNSSECAlgorithm `json:"allowedAlgorithms"`
	AllowedDigestTypes []DSDigestType    `json:"allowedDigestTypes"`
	MaxDSRecordCount   int               `json:"maxDsRecordCount"`
}

// DSRecord is a DS record at the registry.
type DSRecord struct {
	KeyTag                    int             `json:"keyTag"`
	Algorithm                 DNSSECAlgorithm `json:"algorithm"`
	DigestType                DSDigestType    `json:"digestType"`
	Digest                    string          `json:"digest"`
	Status                    DSRecordStatus  `json:"status"`
	IsAlgorithmNotRecommended bool            `json:"isAlgorithmNotRecommended"`
	CreatedAt                 Time            `json:"createdAt"`
	ActivatedAt               Time            `json:"activatedAt"`
	RemovedAt                 Time            `json:"removedAt"`
}

// DS is a DS record to publish or remove, as your DNS provider shows it.
type DS struct {
	KeyTag     int             `json:"keyTag"`
	Algorithm  DNSSECAlgorithm `json:"algorithm"`
	DigestType DSDigestType    `json:"digestType"`
	Digest     string          `json:"digest"`
}

// ParseDS parses a DS record in presentation format:
// "2371 13 2 1F987CC6583E92DF0890718C42..." or a whole zone-file line such
// as "example.com. 3600 IN DS 2371 13 2 1F98...". Spaces inside the digest
// are removed.
func ParseDS(s string) (DS, error) {
	f := strings.Fields(s)
	for i, w := range f {
		if strings.EqualFold(w, "DS") {
			f = f[i+1:]
			break
		}
	}
	if len(f) < 4 {
		return DS{}, fmt.Errorf("dna: %q is not a DS record (want KEYTAG ALGORITHM DIGESTTYPE DIGEST)", s)
	}
	keyTag, err1 := strconv.Atoi(f[0])
	alg, err2 := ParseDNSSECAlgorithm(f[1])
	dt, err3 := ParseDSDigestType(f[2])
	if err1 != nil || err2 != nil || err3 != nil {
		return DS{}, fmt.Errorf("dna: %q is not a DS record (want KEYTAG ALGORITHM DIGESTTYPE DIGEST)", s)
	}
	return DS{KeyTag: keyTag, Algorithm: alg, DigestType: dt, Digest: strings.ToUpper(strings.Join(f[3:], ""))}, nil
}

// String formats the record as "2371 13 2 1F98...".
func (d DS) String() string {
	return fmt.Sprintf("%d %d %d %s", d.KeyTag, d.Algorithm, d.DigestType, d.Digest)
}

func (d DS) validate(v *validator, field string) {
	v.check(d.KeyTag >= 0 && d.KeyTag <= 65535, field+".keyTag", "must be between 0 and 65535")
	v.check(d.Algorithm >= 1 && d.Algorithm <= 255, field+".algorithm", "is required (e.g. AlgorithmECDSAP256SHA256)")
	v.check(d.DigestType >= 1 && d.DigestType <= 255, field+".digestType", "is required (e.g. DigestSHA256)")
	v.required(field+".digest", d.Digest)
	v.maxLen(field+".digest", d.Digest, 128)
}

// DNSSECOperationResult reports one DNSSEC operation.
type DNSSECOperationResult struct {
	KeyTag        int                 `json:"keyTag"`
	OperationType DNSSECOperationType `json:"operationType"`
	Success       bool                `json:"isSuccess"`
	// Pending means the registry accepted the change but has not confirmed
	// it yet.
	Pending bool   `json:"isPending"`
	Message string `json:"resultMessage"`
}

// DNSSECGetRequest reads the DNSSEC state of a domain. CLI: dna dnssec get
type DNSSECGetRequest struct {
	BaseRequest
	DomainName string
}

func (DNSSECGetRequest) Endpoint() Endpoint {
	return Endpoint{Method: http.MethodGet, Path: "/api/v1/domains/dns/dnssec", Safety: Read}
}

func (r DNSSECGetRequest) Query() url.Values {
	return newParams().str("DomainName", r.DomainName).Values
}

func (r DNSSECGetRequest) Validate() error { return requireDomain(r.DomainName) }

// DNSSECAddRequest publishes DS records at the registry.
// CLI: dna dnssec add
type DNSSECAddRequest struct {
	BaseRequest
	DomainName string
	Records    []DS
}

func (DNSSECAddRequest) Endpoint() Endpoint {
	return Endpoint{Method: http.MethodPost, Path: "/api/v1/domains/dns/dnssec", Safety: Write}
}

func (r DNSSECAddRequest) Body() any {
	records := make([]DS, len(r.Records))
	for i, d := range r.Records {
		d.Digest = strings.ToUpper(strings.Join(strings.Fields(d.Digest), ""))
		records[i] = d
	}
	return object{"domainName": trim(r.DomainName), "records": records}
}

func (r DNSSECAddRequest) Validate() error {
	var v validator
	v.required("domainName", r.DomainName)
	v.requiredList("records", len(r.Records))
	for i, d := range r.Records {
		d.validate(&v, fmt.Sprintf("records[%d]", i))
	}
	return v.err()
}

// DNSSECRemoveRequest removes one DS record. CLI: dna dnssec remove
type DNSSECRemoveRequest struct {
	BaseRequest
	DomainName string
	Record     DS
}

func (DNSSECRemoveRequest) Endpoint() Endpoint {
	return Endpoint{Method: http.MethodDelete, Path: "/api/v1/domains/dns/dnssec", Safety: Destructive}
}

func (r DNSSECRemoveRequest) Query() url.Values {
	p := newParams().str("DomainName", r.DomainName)
	p.Set("KeyTag", strconv.Itoa(r.Record.KeyTag))
	p.Set("Algorithm", strconv.Itoa(int(r.Record.Algorithm)))
	p.Set("DigestType", strconv.Itoa(int(r.Record.DigestType)))
	p.Set("Digest", strings.ToUpper(strings.Join(strings.Fields(r.Record.Digest), "")))
	return p.Values
}

func (r DNSSECRemoveRequest) Validate() error {
	var v validator
	v.required("domainName", r.DomainName)
	r.Record.validate(&v, "record")
	return v.err()
}

// DNSSECDisableRequest removes every DS record of a domain.
// CLI: dna dnssec disable
type DNSSECDisableRequest struct {
	BaseRequest
	DomainName string
}

func (DNSSECDisableRequest) Endpoint() Endpoint {
	return Endpoint{Method: http.MethodPost, Path: "/api/v1/domains/dns/dnssec/disable", Safety: Destructive}
}

func (r DNSSECDisableRequest) Body() any       { return domainBody(r.DomainName) }
func (r DNSSECDisableRequest) Validate() error { return requireDomain(r.DomainName) }

// Get returns the DNSSEC state of a domain: status, DS records and the
// algorithms the registry accepts.
func (s *DNSSECService) Get(ctx context.Context, domainName string) (*DNSSECInfo, error) {
	return call[DNSSECInfo](ctx, s.client, DNSSECGetRequest{DomainName: domainName})
}

// Add publishes DS records from your DNS provider (see ParseDS). Wrong DS
// records make the domain unresolvable for validating resolvers: copy them
// exactly. Check Success or Pending of each result.
func (s *DNSSECService) Add(ctx context.Context, domainName string, records ...DS) ([]DNSSECOperationResult, error) {
	res, err := call[struct {
		Items []DNSSECOperationResult `json:"items"`
	}](ctx, s.client, DNSSECAddRequest{DomainName: domainName, Records: records})
	if err != nil {
		return nil, err
	}
	return res.Items, nil
}

// Remove removes one DS record.
func (s *DNSSECService) Remove(ctx context.Context, domainName string, record DS) (*DNSSECOperationResult, error) {
	return s.operation(ctx, DNSSECRemoveRequest{DomainName: domainName, Record: record})
}

// Disable removes every DS record of a domain.
func (s *DNSSECService) Disable(ctx context.Context, domainName string) (*DNSSECOperationResult, error) {
	return s.operation(ctx, DNSSECDisableRequest{DomainName: domainName})
}

func (s *DNSSECService) operation(ctx context.Context, req Request) (*DNSSECOperationResult, error) {
	res, resp, err := send[DNSSECOperationResult](ctx, s.client, req)
	if err != nil {
		return nil, err
	}
	if !res.Success && !res.Pending {
		return nil, operationError(req, resp, "", res.Message)
	}
	return res, nil
}
