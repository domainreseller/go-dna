package dna

import (
	"context"
	"net/http"
	"net/url"
	"strings"
)

// Forwarding is the URL forwarding of a domain.
type Forwarding struct {
	// URL is the target address.
	URL  string      `json:"redirectAlias"`
	Type ForwardType `json:"redirectType"`
}

// DomainForwardGetRequest reads the URL forwarding of a domain.
// CLI: dna domain forward get
type DomainForwardGetRequest struct {
	BaseRequest
	DomainName string
}

func (DomainForwardGetRequest) Endpoint() Endpoint {
	return Endpoint{Method: http.MethodGet, Path: "/api/v1/domains/forwards", Safety: Read}
}

func (r DomainForwardGetRequest) Query() url.Values {
	return newParams().str("domainName", r.DomainName).Values
}

func (r DomainForwardGetRequest) Validate() error { return requireDomain(r.DomainName) }

// DomainForwardSetRequest forwards a domain to a URL.
// CLI: dna domain forward set
type DomainForwardSetRequest struct {
	BaseRequest
	DomainName string
	// URL is the target, e.g. "https://www.example.org".
	URL string
	// Type is ForwardTypeStandard (a 301 redirect, the default) or
	// ForwardTypeFrame.
	Type ForwardType
}

func (DomainForwardSetRequest) Endpoint() Endpoint {
	return Endpoint{Method: http.MethodPost, Path: "/api/v1/domains/forwards", Safety: Write, Idempotent: true}
}

func (r DomainForwardSetRequest) Body() any {
	t := r.Type
	if t == "" {
		t = ForwardTypeStandard
	}
	return object{"domainName": trim(r.DomainName), "redirectAddress": trim(r.URL), "forwardType": t}
}

func (r DomainForwardSetRequest) Validate() error {
	var v validator
	v.required("domainName", r.DomainName)
	v.required("url", r.URL)
	if u, err := url.Parse(trim(r.URL)); r.URL != "" && (err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "") {
		v.add("url", "must be an http:// or https:// URL")
	}
	v.check(r.Type == "" || r.Type == ForwardTypeStandard || r.Type == ForwardTypeFrame, "type", "must be ForwardTypeStandard or ForwardTypeFrame")
	return v.err()
}

// DomainForwardDeleteRequest removes the URL forwarding of a domain.
// CLI: dna domain forward delete
type DomainForwardDeleteRequest struct {
	BaseRequest
	DomainName string
}

func (DomainForwardDeleteRequest) Endpoint() Endpoint {
	return Endpoint{Method: http.MethodDelete, Path: "/api/v1/domains/forwards", Safety: Destructive}
}

func (r DomainForwardDeleteRequest) Query() url.Values {
	return newParams().str("DomainName", r.DomainName).Values
}

func (r DomainForwardDeleteRequest) Validate() error { return requireDomain(r.DomainName) }

// Forwarding returns the URL forwarding of a domain, or nil (and no error)
// when the domain has none.
func (s *DomainService) Forwarding(ctx context.Context, domainName string) (*Forwarding, error) {
	f, err := call[Forwarding](ctx, s.client, DomainForwardGetRequest{DomainName: domainName})
	if HasCode(err, CodeForwardingNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(f.URL) == "" {
		return nil, nil
	}
	return f, nil
}

// SetForwarding forwards a domain to targetURL with a redirect
// (ForwardTypeStandard) or a frame (ForwardTypeFrame).
func (s *DomainService) SetForwarding(ctx context.Context, domainName, targetURL string, forwardType ForwardType) error {
	return exec(ctx, s.client, DomainForwardSetRequest{DomainName: domainName, URL: targetURL, Type: forwardType})
}

// DeleteForwarding removes the URL forwarding of a domain. It also returns
// nil when the domain had none.
func (s *DomainService) DeleteForwarding(ctx context.Context, domainName string) error {
	err := exec(ctx, s.client, DomainForwardDeleteRequest{DomainName: domainName})
	if HasCode(err, CodeForwardingNotFound) {
		return nil
	}
	return err
}
