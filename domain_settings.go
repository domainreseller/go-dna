package dna

import (
	"context"
	"net"
	"net/http"
	"net/url"
	"strings"
)

// DomainLockRequest turns the transfer lock on. CLI: dna domain lock
type DomainLockRequest struct {
	BaseRequest
	DomainName string
}

func (DomainLockRequest) Endpoint() Endpoint {
	return Endpoint{Method: http.MethodPost, Path: "/api/v1/domains/lock", Safety: Write, Idempotent: true}
}

func (r DomainLockRequest) Body() any       { return domainBody(r.DomainName) }
func (r DomainLockRequest) Validate() error { return requireDomain(r.DomainName) }

// DomainUnlockRequest turns the transfer lock off, which a transfer to
// another registrar needs. CLI: dna domain unlock
type DomainUnlockRequest struct {
	BaseRequest
	DomainName string
}

func (DomainUnlockRequest) Endpoint() Endpoint {
	return Endpoint{Method: http.MethodPost, Path: "/api/v1/domains/unlock", Safety: Write, Idempotent: true}
}

func (r DomainUnlockRequest) Body() any       { return domainBody(r.DomainName) }
func (r DomainUnlockRequest) Validate() error { return requireDomain(r.DomainName) }

// DomainPrivacyRequest turns WHOIS privacy protection on or off.
// CLI: dna domain privacy
type DomainPrivacyRequest struct {
	BaseRequest
	DomainName string
	Enabled    bool
}

func (DomainPrivacyRequest) Endpoint() Endpoint {
	return Endpoint{Method: http.MethodPost, Path: "/api/v1/domains/privacy", Safety: Write, Idempotent: true}
}

func (r DomainPrivacyRequest) Body() any {
	return object{"domainName": trim(r.DomainName), "privacyStatus": r.Enabled}
}

func (r DomainPrivacyRequest) Validate() error { return requireDomain(r.DomainName) }

// AuthCode is a domain's transfer authorization (EPP) code.
type AuthCode struct {
	DomainName string `json:"domainName"`
	AuthCode   string `json:"authCode"`
}

// DomainUpdateAuthCodeRequest asks the registry for a new transfer auth
// code. CLI: dna domain update-auth-code
type DomainUpdateAuthCodeRequest struct {
	BaseRequest
	DomainName string
}

func (DomainUpdateAuthCodeRequest) Endpoint() Endpoint {
	return Endpoint{Method: http.MethodPut, Path: "/api/v1/domains/auth", Safety: Write}
}

func (r DomainUpdateAuthCodeRequest) Body() any       { return domainBody(r.DomainName) }
func (r DomainUpdateAuthCodeRequest) Validate() error { return requireDomain(r.DomainName) }

// DomainNameServersSetRequest replaces the nameservers of a domain.
// CLI: dna domain nameservers set
type DomainNameServersSetRequest struct {
	BaseRequest
	DomainName string
	// NameServers is the complete new list of nameserver host names.
	NameServers []string
	// DNSSECMode matters only for a DNSSEC-signed domain. The zero value,
	// DNSSECModeBlock, refuses the change while DS records exist.
	DNSSECMode DNSSECMode
}

func (DomainNameServersSetRequest) Endpoint() Endpoint {
	return Endpoint{Method: http.MethodPut, Path: "/api/v1/domains/dns/name-server", Safety: Write, Idempotent: true}
}

func (r DomainNameServersSetRequest) Body() any {
	return object{"domainName": trim(r.DomainName), "nameServers": lowerAll(r.NameServers), "dnssecMode": r.DNSSECMode}
}

func (r DomainNameServersSetRequest) Validate() error {
	var v validator
	v.required("domainName", r.DomainName)
	v.requiredList("nameServers", len(trimAll(r.NameServers)))
	_, known := dnssecModeNames[r.DNSSECMode]
	v.check(known, "dnssecMode", "must be DNSSECModeBlock, DNSSECModeKeep or DNSSECModeDisable")
	return v.err()
}

// DomainHostAddRequest creates a child nameserver (glue record) such as
// ns1.example.com. CLI: dna domain host add
type DomainHostAddRequest struct {
	BaseRequest
	DomainName string
	// HostName is the full host name, e.g. "ns1.example.com".
	HostName string
	// IPAddresses are IPv4 or IPv6 addresses; the version is detected.
	IPAddresses []string
}

func (DomainHostAddRequest) Endpoint() Endpoint {
	return Endpoint{Method: http.MethodPost, Path: "/api/v1/domains/dns/host", Safety: Write}
}

func (r DomainHostAddRequest) Body() any {
	return object{"domainName": trim(r.DomainName), "hostName": trim(r.HostName), "ipAddresses": ipAddresses(r.IPAddresses)}
}

func (r DomainHostAddRequest) Validate() error {
	var v validator
	v.required("domainName", r.DomainName)
	v.required("hostName", r.HostName)
	validateIPs(&v, r.IPAddresses)
	return v.err()
}

// DomainHostUpdateRequest changes a child nameserver's IP addresses, and
// optionally its name. CLI: dna domain host update
type DomainHostUpdateRequest struct {
	BaseRequest
	DomainName string
	HostName   string
	// NewHostName renames the host; leave it empty to keep the name.
	NewHostName string
	// IPAddresses is the complete new list of addresses.
	IPAddresses []string
}

func (DomainHostUpdateRequest) Endpoint() Endpoint {
	return Endpoint{Method: http.MethodPut, Path: "/api/v1/domains/dns/host", Safety: Write, Idempotent: true}
}

func (r DomainHostUpdateRequest) Body() any {
	return object{"domainName": trim(r.DomainName), "hostName": trim(r.HostName), "ipAddresses": ipAddresses(r.IPAddresses)}.
		set("newHostName", r.NewHostName)
}

func (r DomainHostUpdateRequest) Validate() error {
	var v validator
	v.required("domainName", r.DomainName)
	v.required("hostName", r.HostName)
	validateIPs(&v, r.IPAddresses)
	return v.err()
}

// DomainHostDeleteRequest deletes a child nameserver.
// CLI: dna domain host delete
type DomainHostDeleteRequest struct {
	BaseRequest
	DomainName string
	HostName   string
}

func (DomainHostDeleteRequest) Endpoint() Endpoint {
	return Endpoint{Method: http.MethodDelete, Path: "/api/v1/domains/dns/host", Safety: Destructive}
}

func (r DomainHostDeleteRequest) Query() url.Values {
	return newParams().str("DomainName", r.DomainName).str("HostName", r.HostName).Values
}

func (r DomainHostDeleteRequest) Validate() error {
	var v validator
	v.required("domainName", r.DomainName)
	v.required("hostName", r.HostName)
	return v.err()
}

func ipAddresses(list []string) []IPAddress {
	var out []IPAddress
	for _, s := range trimAll(list) {
		ver := IPVersion6
		if ip := net.ParseIP(s); ip != nil && ip.To4() != nil {
			ver = IPVersion4
		}
		out = append(out, IPAddress{Address: s, Version: ver})
	}
	return out
}

func validateIPs(v *validator, list []string) {
	ips := trimAll(list)
	v.requiredList("ipAddresses", len(ips))
	for _, s := range ips {
		v.check(net.ParseIP(s) != nil, "ipAddresses", s+" is not an IP address")
	}
}

func lowerAll(list []string) []string {
	out := trimAll(list)
	for i := range out {
		out[i] = strings.ToLower(out[i])
	}
	return out
}

// Lock turns the transfer lock on.
func (s *DomainService) Lock(ctx context.Context, domainName string) error {
	return exec(ctx, s.client, DomainLockRequest{DomainName: domainName})
}

// Unlock turns the transfer lock off, which a transfer to another registrar
// needs.
func (s *DomainService) Unlock(ctx context.Context, domainName string) error {
	return exec(ctx, s.client, DomainUnlockRequest{DomainName: domainName})
}

// SetPrivacy turns WHOIS privacy protection on or off. TLDs without privacy
// answer with CodePrivacyNotSupported.
func (s *DomainService) SetPrivacy(ctx context.Context, domainName string, enabled bool) error {
	return exec(ctx, s.client, DomainPrivacyRequest{DomainName: domainName, Enabled: enabled})
}

// UpdateAuthCode asks the registry for a new transfer auth code and returns
// it. To read the current code, use Info(...).AuthCode.
func (s *DomainService) UpdateAuthCode(ctx context.Context, domainName string) (string, error) {
	res, err := call[AuthCode](ctx, s.client, DomainUpdateAuthCodeRequest{DomainName: domainName})
	if err != nil {
		return "", err
	}
	return res.AuthCode, nil
}

// SetNameServers replaces the nameservers of a domain.
//
//	err := client.Domains.SetNameServers(ctx, dna.DomainNameServersSetRequest{
//		DomainName:  "example.com",
//		NameServers: []string{"ns1.example.net", "ns2.example.net"},
//	})
func (s *DomainService) SetNameServers(ctx context.Context, req DomainNameServersSetRequest) error {
	return exec(ctx, s.client, req)
}

// AddHost creates a child nameserver such as ns1.example.com with one or
// more IPv4/IPv6 addresses.
func (s *DomainService) AddHost(ctx context.Context, domainName, hostName string, ipAddresses ...string) error {
	return exec(ctx, s.client, DomainHostAddRequest{DomainName: domainName, HostName: hostName, IPAddresses: ipAddresses})
}

// UpdateHost changes a child nameserver's addresses or name.
func (s *DomainService) UpdateHost(ctx context.Context, req DomainHostUpdateRequest) error {
	return exec(ctx, s.client, req)
}

// DeleteHost deletes a child nameserver.
func (s *DomainService) DeleteHost(ctx context.Context, domainName, hostName string) error {
	return exec(ctx, s.client, DomainHostDeleteRequest{DomainName: domainName, HostName: hostName})
}
