package dna

import (
	"context"
	"net/http"
	"net/url"
	"strings"
)

// SSLService buys, configures, issues, reissues and revokes SSL
// certificates. CLI: dna ssl ...
//
// The flow: find a package with Products.SSLs, GenerateCSR (or bring your
// own), Initialize (charges the balance), Configure, then poll Status until
// the certificate is issued and download it with Content.
type SSLService service

// SSLContact is the contact of a certificate order.
type SSLContact struct {
	Title            string `json:"title,omitempty"`
	FirstName        string `json:"firstName"`
	LastName         string `json:"lastName"`
	CompanyName      string `json:"companyName"`
	Email            string `json:"eMail"`
	Address          string `json:"address"`
	City             string `json:"city"`
	State            string `json:"state,omitempty"`
	PostalCode       string `json:"postalCode"`
	Country          string `json:"country"`
	PhoneCountryCode string `json:"phoneCountryCode"`
	Phone            string `json:"phone"`
	FaxCountryCode   string `json:"faxCountryCode,omitempty"`
	Fax              string `json:"fax,omitempty"`
	// Type is the contact's role, set by the API.
	Type string `json:"type,omitempty"`
}

func (c SSLContact) normalized() SSLContact {
	t := strings.TrimSpace
	c.Title, c.FirstName, c.LastName, c.CompanyName = t(c.Title), t(c.FirstName), t(c.LastName), t(c.CompanyName)
	c.Email, c.Address, c.City, c.State, c.PostalCode = t(c.Email), t(c.Address), t(c.City), t(c.State), t(c.PostalCode)
	c.Country = strings.ToUpper(t(c.Country))
	c.PhoneCountryCode, c.Phone = digits(c.PhoneCountryCode), digits(c.Phone)
	c.FaxCountryCode, c.Fax = digits(c.FaxCountryCode), digits(c.Fax)
	c.Type = ""
	return c
}

func (c SSLContact) validate(v *validator) {
	c = c.normalized()
	for _, f := range []struct{ name, value string }{
		{"firstName", c.FirstName}, {"lastName", c.LastName}, {"companyName", c.CompanyName}, {"email", c.Email},
		{"address", c.Address}, {"city", c.City}, {"postalCode", c.PostalCode}, {"country", c.Country},
		{"phoneCountryCode", c.PhoneCountryCode}, {"phone", c.Phone},
	} {
		v.required("contact."+f.name, f.value)
	}
	v.maxLen("contact.firstName", c.FirstName, 70)
	v.maxLen("contact.lastName", c.LastName, 70)
	v.maxLen("contact.companyName", c.CompanyName, 150)
	v.maxLen("contact.address", c.Address, 200)
	v.maxLen("contact.city", c.City, 20)
	v.maxLen("contact.state", c.State, 20)
	v.maxLen("contact.postalCode", c.PostalCode, 10)
	v.maxLen("contact.phone", c.Phone, 15)
}

// CSR is a certificate signing request and its private key.
type CSR struct {
	CSR string `json:"csr"`
	// PrivateKey is needed to install the certificate. Store it safely.
	PrivateKey string `json:"privateKey"`
}

// CSRInfo is the decoded subject of a CSR.
type CSRInfo struct {
	DomainName       string `json:"domainName"`
	Country          string `json:"country"`
	State            string `json:"state"`
	Locality         string `json:"locality"`
	Organization     string `json:"organization"`
	OrganizationUnit string `json:"organizationUnit"`
	AlternativeNames string `json:"alternativeNames"`
	IsWildcard       bool   `json:"isWildcard"`
	IsSAN            bool   `json:"isSan"`
}

// SSLOperationResult is the API's answer to a certificate operation.
type SSLOperationResult struct {
	Success       bool   `json:"success"`
	Message       string `json:"message"`
	CertificateID string `json:"certificateId"`
	Status        string `json:"status"`
	// DCVDetails tells how to complete domain control validation (e.g. the
	// CNAME record or file to create).
	DCVDetails string `json:"dcvDetails"`
	CSR        string `json:"csr"`
	PrivateKey string `json:"privateKey"`
}

// CertificateStatus is the current state of a certificate.
type CertificateStatus struct {
	DomainName    string `json:"domainName"`
	Status        string `json:"status"`
	StatusMessage string `json:"statusMessage"`
	IsConfigured  bool   `json:"isConfigured"`
	IsIssued      bool   `json:"isIssued"`
	ExpiresAt     Time   `json:"certificateExpiresAt"`
}

// Certificate is the full record of a certificate.
type Certificate struct {
	DomainName   string `json:"domainName"`
	Status       string `json:"status"`
	IsConfigured bool   `json:"isConfigured"`
	IsIssued     bool   `json:"isIssued"`
	PackageName  string `json:"packageName"`
	PackageCode  string `json:"packageCode"`
	Subscription struct {
		PurchasedAt Time `json:"purchasedAt"`
		ExpiresAt   Time `json:"expiresAt"`
	} `json:"subscription"`
	Configuration struct {
		ConfiguredAt       Time   `json:"configuredAt"`
		ConfigurationLink  string `json:"configurationLink"`
		Pin                string `json:"pin"`
		WebServer          string `json:"webServer"`
		AuthenticationType string `json:"authenticationType"`
		SANList            string `json:"sanList"`
		WildcardList       string `json:"wildcardList"`
		ApproverMail       string `json:"approverMail"`
		CSR                string `json:"csr"`
		DCVDetails         string `json:"dcvDetails"`
	} `json:"configuration"`
	Issued struct {
		IssuedAt   Time `json:"issuedAt"`
		ValidUntil Time `json:"validUntil"`
	} `json:"certificate"`
	Contacts []SSLContact `json:"contacts"`
}

// CertificateContent is an issued certificate and its chain, PEM-encoded.
type CertificateContent struct {
	Certificate   string `json:"certificate"`
	Intermediate1 string `json:"intermediateCertificate1"`
	Intermediate2 string `json:"intermediateCertificate2"`
	Root          string `json:"rootCertificate"`
}

// Chain returns the certificate followed by its intermediates, the bundle
// most web servers want.
func (c CertificateContent) Chain() string {
	var parts []string
	for _, p := range []string{c.Certificate, c.Intermediate1, c.Intermediate2} {
		if p = strings.TrimSpace(p); p != "" {
			parts = append(parts, p)
		}
	}
	return strings.Join(parts, "\n") + "\n"
}

// SSLGenerateCSRRequest generates a CSR and private key.
// CLI: dna ssl generate-csr
type SSLGenerateCSRRequest struct {
	BaseRequest
	DomainName string
	// Country is the 2-letter ISO code.
	Country, State, Locality       string
	Organization, OrganizationUnit string
	// AlternativeNames are extra names (SAN).
	AlternativeNames []string
}

func (SSLGenerateCSRRequest) Endpoint() Endpoint {
	return Endpoint{Method: http.MethodPost, Path: "/api/v1/ssls/generate-csr", Safety: Read}
}

func (r SSLGenerateCSRRequest) Body() any {
	return object{
		"domainName": trim(r.DomainName), "country": strings.ToUpper(trim(r.Country)), "state": trim(r.State),
		"locality": trim(r.Locality), "organization": trim(r.Organization), "organizationUnit": trim(r.OrganizationUnit),
	}.set("alternativeNames", trimAll(r.AlternativeNames))
}

func (r SSLGenerateCSRRequest) Validate() error {
	var v validator
	v.required("domainName", r.DomainName)
	v.maxLen("domainName", trim(r.DomainName), 72)
	v.required("country", r.Country)
	v.check(r.Country == "" || len(trim(r.Country)) == 2, "country", "must be a 2-letter ISO code")
	v.required("state", r.State)
	v.required("locality", r.Locality)
	v.required("organization", r.Organization)
	v.required("organizationUnit", r.OrganizationUnit)
	return v.err()
}

// SSLCheckCSRRequest decodes a CSR. CLI: dna ssl check-csr
type SSLCheckCSRRequest struct {
	BaseRequest
	// CSR is PEM-encoded.
	CSR string
}

func (SSLCheckCSRRequest) Endpoint() Endpoint {
	return Endpoint{Method: http.MethodPost, Path: "/api/v1/ssls/check-csr", Safety: Read}
}

func (r SSLCheckCSRRequest) Body() any { return object{"csr": trim(r.CSR)} }

func (r SSLCheckCSRRequest) Validate() error {
	var v validator
	v.required("csr", r.CSR)
	return v.err()
}

// SSLInitializeRequest buys a certificate package for a domain. It charges
// your balance. CLI: dna ssl initialize
type SSLInitializeRequest struct {
	BaseRequest
	DomainName string
	// PackageCode is an SSLProduct.ProductCode from Products.SSLs.
	PackageCode string
}

func (SSLInitializeRequest) Endpoint() Endpoint {
	return Endpoint{Method: http.MethodPost, Path: "/api/v1/ssls/initialize", Safety: Billable}
}

func (r SSLInitializeRequest) Body() any {
	return object{"domainName": trim(r.DomainName), "packageCode": trim(r.PackageCode)}
}

func (r SSLInitializeRequest) Validate() error {
	var v validator
	v.required("domainName", r.DomainName)
	v.required("packageCode", r.PackageCode)
	return v.err()
}

// SSLConfigureRequest submits the CSR, validation method and contact of a
// certificate for issuance. CLI: dna ssl configure
type SSLConfigureRequest struct {
	BaseRequest
	DomainName string
	// CSR is PEM-encoded.
	CSR string
	// PrivateKey is PEM-encoded; optional when the CSR was made elsewhere.
	PrivateKey string
	// ApproverMail receives the approval e-mail (AuthenticationEmail). List
	// the allowed addresses with Approvers.
	ApproverMail       string
	AuthenticationType AuthenticationType
	WebServerType      WebServerType
	// SANList and WildcardList are additional names.
	SANList, WildcardList []string
	Contact               SSLContact
}

func (SSLConfigureRequest) Endpoint() Endpoint {
	return Endpoint{Method: http.MethodPut, Path: "/api/v1/ssls/configure", Safety: Write}
}

func (r SSLConfigureRequest) Body() any {
	return object{"domainName": trim(r.DomainName), "csr": trim(r.CSR), "contact": r.Contact.normalized()}.
		set("privateKey", r.PrivateKey).
		set("approverMail", r.ApproverMail).
		set("authenticationType", string(r.AuthenticationType)).
		set("webServerType", string(r.WebServerType)).
		set("sanList", strings.Join(trimAll(r.SANList), ",")).
		set("wildcardList", strings.Join(trimAll(r.WildcardList), ","))
}

func (r SSLConfigureRequest) Validate() error {
	var v validator
	v.required("domainName", r.DomainName)
	v.required("csr", r.CSR)
	v.check(r.ApproverMail == "" || strings.Contains(r.ApproverMail, "@"), "approverMail", "is not an e-mail address")
	r.Contact.validate(&v)
	return v.err()
}

// SSLStatusRequest refreshes and reads the state of a certificate.
// CLI: dna ssl status
type SSLStatusRequest struct {
	BaseRequest
	DomainName string
}

func (SSLStatusRequest) Endpoint() Endpoint {
	return Endpoint{Method: http.MethodPut, Path: "/api/v1/ssls/check-status", Safety: Read}
}

func (r SSLStatusRequest) Body() any       { return domainBody(r.DomainName) }
func (r SSLStatusRequest) Validate() error { return requireDomain(r.DomainName) }

// SSLGetRequest reads the full record of a certificate. CLI: dna ssl get
type SSLGetRequest struct {
	BaseRequest
	DomainName string
}

func (SSLGetRequest) Endpoint() Endpoint {
	return Endpoint{Method: http.MethodGet, Path: "/api/v1/ssls/certificate", Safety: Read}
}

func (r SSLGetRequest) Query() url.Values {
	return newParams().str("DomainName", r.DomainName).Values
}

func (r SSLGetRequest) Validate() error { return requireDomain(r.DomainName) }

// SSLContentRequest downloads an issued certificate. CLI: dna ssl content
type SSLContentRequest struct {
	BaseRequest
	DomainName string
}

func (SSLContentRequest) Endpoint() Endpoint {
	return Endpoint{Method: http.MethodGet, Path: "/api/v1/ssls/certificate-content", Safety: Read}
}

func (r SSLContentRequest) Query() url.Values {
	return newParams().str("DomainName", r.DomainName).Values
}

func (r SSLContentRequest) Validate() error { return requireDomain(r.DomainName) }

// SSLReissueRequest reissues a certificate with a new CSR, key or approver.
// CLI: dna ssl reissue
type SSLReissueRequest struct {
	BaseRequest
	DomainName   string
	ApproverMail string
	CSR          string
	PrivateKey   string
}

func (SSLReissueRequest) Endpoint() Endpoint {
	return Endpoint{Method: http.MethodPost, Path: "/api/v1/ssls/reissue", Safety: Write}
}

func (r SSLReissueRequest) Body() any {
	return domainBody(r.DomainName).set("approverMail", r.ApproverMail).set("csr", r.CSR).set("privateKey", r.PrivateKey)
}

func (r SSLReissueRequest) Validate() error { return requireDomain(r.DomainName) }

// SSLApproversRequest lists the e-mail addresses allowed for e-mail
// validation. CLI: dna ssl approvers
type SSLApproversRequest struct {
	BaseRequest
	DomainName string
	// PackageCode narrows the list to a package; optional.
	PackageCode string
}

func (SSLApproversRequest) Endpoint() Endpoint {
	return Endpoint{Method: http.MethodGet, Path: "/api/v1/ssls/approvers", Safety: Read}
}

func (r SSLApproversRequest) Query() url.Values {
	return newParams().str("DomainName", r.DomainName).str("PackageCode", r.PackageCode).Values
}

func (r SSLApproversRequest) Validate() error { return requireDomain(r.DomainName) }

// SSLRevokeRequest revokes a certificate. This cannot be undone.
// CLI: dna ssl revoke
type SSLRevokeRequest struct {
	BaseRequest
	DomainName string
}

func (SSLRevokeRequest) Endpoint() Endpoint {
	return Endpoint{Method: http.MethodDelete, Path: "/api/v1/ssls/revoke", Safety: Destructive}
}

func (r SSLRevokeRequest) Query() url.Values {
	return newParams().str("DomainName", r.DomainName).Values
}

func (r SSLRevokeRequest) Validate() error { return requireDomain(r.DomainName) }

// GenerateCSR returns a new CSR and its private key.
func (s *SSLService) GenerateCSR(ctx context.Context, req SSLGenerateCSRRequest) (*CSR, error) {
	return call[CSR](ctx, s.client, req)
}

// CheckCSR decodes a PEM-encoded CSR.
func (s *SSLService) CheckCSR(ctx context.Context, csr string) (*CSRInfo, error) {
	return call[CSRInfo](ctx, s.client, SSLCheckCSRRequest{CSR: csr})
}

// Initialize buys a certificate package for a domain and charges your
// balance. Then call Configure.
func (s *SSLService) Initialize(ctx context.Context, domainName, packageCode string) (*SSLOperationResult, error) {
	return s.operation(ctx, SSLInitializeRequest{DomainName: domainName, PackageCode: packageCode})
}

// Configure submits the CSR, validation method and contact. Then poll
// Status until IsIssued.
func (s *SSLService) Configure(ctx context.Context, req SSLConfigureRequest) (*SSLOperationResult, error) {
	return s.operation(ctx, req)
}

// Status refreshes and returns the state of a certificate.
func (s *SSLService) Status(ctx context.Context, domainName string) (*CertificateStatus, error) {
	return call[CertificateStatus](ctx, s.client, SSLStatusRequest{DomainName: domainName})
}

// Get returns the full record of a certificate: subscription,
// configuration and contacts.
func (s *SSLService) Get(ctx context.Context, domainName string) (*Certificate, error) {
	return call[Certificate](ctx, s.client, SSLGetRequest{DomainName: domainName})
}

// Content returns the issued certificate and its chain.
func (s *SSLService) Content(ctx context.Context, domainName string) (*CertificateContent, error) {
	return call[CertificateContent](ctx, s.client, SSLContentRequest{DomainName: domainName})
}

// Reissue reissues a certificate with a new CSR, key or approver.
func (s *SSLService) Reissue(ctx context.Context, req SSLReissueRequest) (*SSLOperationResult, error) {
	return s.operation(ctx, req)
}

// Approvers lists the e-mail addresses allowed for e-mail validation.
// packageCode is optional.
func (s *SSLService) Approvers(ctx context.Context, domainName, packageCode string) ([]string, error) {
	res, err := call[struct {
		Mails []string `json:"mails"`
	}](ctx, s.client, SSLApproversRequest{DomainName: domainName, PackageCode: packageCode})
	if err != nil {
		return nil, err
	}
	return res.Mails, nil
}

// Revoke revokes a certificate. This cannot be undone.
func (s *SSLService) Revoke(ctx context.Context, domainName string) (*SSLOperationResult, error) {
	return s.operation(ctx, SSLRevokeRequest{DomainName: domainName})
}

func (s *SSLService) operation(ctx context.Context, req Request) (*SSLOperationResult, error) {
	res, resp, err := send[SSLOperationResult](ctx, s.client, req)
	if err != nil {
		return nil, err
	}
	if !res.Success {
		return nil, operationError(req, resp, "", res.Message)
	}
	return res, nil
}
