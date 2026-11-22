package dna

import (
	"context"
	"iter"
	"net/http"
	"net/url"
	"strings"
)

// SSLService buys, configures, issues, reissues, renews and revokes SSL
// certificates. CLI: dna ssl ...
//
// The flow: find a package with Products.SSLs, GenerateCSR (or bring your
// own), Initialize (charges the balance; keep the CertificateID it returns),
// Configure within 15 days, then poll Status until the certificate is issued
// and download it with Content.
//
// A domain can have several certificates, so every operation on an existing
// certificate takes its certificate id (a GUID), never the domain name.
// Initialize and Renew return the id; List finds it by domain name.
//
// To renew a certificate, call Renew (charges the balance) and Configure the
// new certificate it returns. Do not call Initialize again for that: it is an
// unrelated purchase at the purchase price, and the old certificate would
// never become "Renewed".
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
	Success bool   `json:"success"`
	Message string `json:"message"`
	// CertificateID identifies the certificate. Store it: every later call
	// for the certificate takes it. After Renew it is the id of the new
	// (renewal) certificate, not of the one that was renewed.
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
	CertificateID string `json:"certificateId"`
	DomainName    string `json:"domainName"`
	Status        string `json:"status"`
	StatusMessage string `json:"statusMessage"`
	IsConfigured  bool   `json:"isConfigured"`
	IsIssued      bool   `json:"isIssued"`
	ExpiresAt     Time   `json:"certificateExpiresAt"`
}

// Certificate is the full record of a certificate.
type Certificate struct {
	CertificateID string `json:"certificateId"`
	DomainName    string `json:"domainName"`
	Status        string `json:"status"`
	IsConfigured  bool   `json:"isConfigured"`
	IsIssued      bool   `json:"isIssued"`
	PackageName   string `json:"packageName"`
	PackageCode   string `json:"packageCode"`
	// Subscription is the paid term. Renew is allowed from 30 days before
	// Subscription.ExpiresAt.
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

// CertificateListItem is a certificate in the certificate list. SSL.Get
// returns more.
type CertificateListItem struct {
	// CertificateID is what the other SSL operations take.
	CertificateID string `json:"certificateId"`
	DomainName    string `json:"domainName"`
	PackageCode   string `json:"packageCode"`
	PackageName   string `json:"packageName"`
	// Status is the state of the certificate: "Active", "WaitingForInstall"
	// (bought, not configured yet), "WaitingForValidation", "Cancelled",
	// "Renewed" (superseded by its renewal), ...
	Status string `json:"status"`
	// StatusReason explains the status, mainly for "Cancelled": "None",
	// "ConfigurationWindowExpired", "ResellerRevoked", ...
	StatusReason string `json:"statusReason"`
	IsConfigured bool   `json:"isConfigured"`
	IsIssued     bool   `json:"isIssued"`
	// PurchasedAt and SubscriptionExpiresAt are the paid term.
	PurchasedAt           Time `json:"purchasedAt"`
	SubscriptionExpiresAt Time `json:"subscriptionExpiresAt"`
	// CertificateExpiresAt is the end of the technical validity; zero until
	// the certificate is issued.
	CertificateExpiresAt Time `json:"certificateExpiresAt"`
	// RenewedFromCertificateID is set on a renewal: the id of the
	// certificate it renews.
	RenewedFromCertificateID string `json:"renewedFromCertificateId"`
}

// CertificateContent is an issued certificate and its chain, PEM-encoded.
type CertificateContent struct {
	CertificateID string `json:"certificateId"`
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
// your balance. The result carries the CertificateID of the new certificate,
// which must be configured within 15 days. CLI: dna ssl initialize
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
	// CertificateID is the certificate to configure: the CertificateID
	// returned by Initialize or Renew.
	CertificateID string
	// CSR is PEM-encoded. Its common name must match the certificate's
	// domain.
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
	return object{"certificateId": trim(r.CertificateID), "csr": trim(r.CSR), "contact": r.Contact.normalized()}.
		set("privateKey", r.PrivateKey).
		set("approverMail", r.ApproverMail).
		set("authenticationType", string(r.AuthenticationType)).
		set("webServerType", string(r.WebServerType)).
		set("sanList", strings.Join(trimAll(r.SANList), ",")).
		set("wildcardList", strings.Join(trimAll(r.WildcardList), ","))
}

func (r SSLConfigureRequest) Validate() error {
	var v validator
	v.certificateID("certificateId", r.CertificateID)
	v.required("csr", r.CSR)
	v.check(r.ApproverMail == "" || strings.Contains(r.ApproverMail, "@"), "approverMail", "is not an e-mail address")
	r.Contact.validate(&v)
	return v.err()
}

// SSLStatusRequest refreshes and reads the state of a certificate.
// CLI: dna ssl status
type SSLStatusRequest struct {
	BaseRequest
	CertificateID string
}

func (SSLStatusRequest) Endpoint() Endpoint {
	return Endpoint{Method: http.MethodPut, Path: "/api/v1/ssls/check-status", Safety: Read}
}

func (r SSLStatusRequest) Body() any       { return certificateBody(r.CertificateID) }
func (r SSLStatusRequest) Validate() error { return requireCertificateID(r.CertificateID) }

// SSLGetRequest reads the full record of a certificate. CLI: dna ssl get
type SSLGetRequest struct {
	BaseRequest
	CertificateID string
}

func (SSLGetRequest) Endpoint() Endpoint {
	return Endpoint{Method: http.MethodGet, Path: "/api/v1/ssls/certificate", Safety: Read}
}

func (r SSLGetRequest) Query() url.Values {
	return newParams().str("CertificateId", r.CertificateID).Values
}

func (r SSLGetRequest) Validate() error { return requireCertificateID(r.CertificateID) }

// SSLContentRequest downloads an issued certificate. CLI: dna ssl content
type SSLContentRequest struct {
	BaseRequest
	CertificateID string
}

func (SSLContentRequest) Endpoint() Endpoint {
	return Endpoint{Method: http.MethodGet, Path: "/api/v1/ssls/certificate-content", Safety: Read}
}

func (r SSLContentRequest) Query() url.Values {
	return newParams().str("CertificateId", r.CertificateID).Values
}

func (r SSLContentRequest) Validate() error { return requireCertificateID(r.CertificateID) }

// SSLReissueRequest reissues a certificate with a new CSR, key or approver.
// CLI: dna ssl reissue
type SSLReissueRequest struct {
	BaseRequest
	CertificateID string
	ApproverMail  string
	CSR           string
	PrivateKey    string
}

func (SSLReissueRequest) Endpoint() Endpoint {
	return Endpoint{Method: http.MethodPost, Path: "/api/v1/ssls/reissue", Safety: Write}
}

func (r SSLReissueRequest) Body() any {
	return certificateBody(r.CertificateID).set("approverMail", r.ApproverMail).set("csr", r.CSR).set("privateKey", r.PrivateKey)
}

func (r SSLReissueRequest) Validate() error { return requireCertificateID(r.CertificateID) }

// SSLApproversRequest lists the e-mail addresses allowed for e-mail
// validation. Set CertificateID for an existing certificate, or DomainName
// and PackageCode before a purchase. CLI: dna ssl approvers
type SSLApproversRequest struct {
	BaseRequest
	// CertificateID is an existing certificate. The API takes the domain and
	// the package from it and ignores DomainName and PackageCode.
	CertificateID string
	// DomainName is the domain you are about to buy a certificate for. It
	// needs PackageCode.
	DomainName string
	// PackageCode is the package you are about to buy: an
	// SSLProduct.ProductCode from Products.SSLs.
	PackageCode string
}

func (SSLApproversRequest) Endpoint() Endpoint {
	return Endpoint{Method: http.MethodGet, Path: "/api/v1/ssls/approvers", Safety: Read}
}

func (r SSLApproversRequest) Query() url.Values {
	return newParams().str("CertificateId", r.CertificateID).str("DomainName", r.DomainName).str("PackageCode", r.PackageCode).Values
}

func (r SSLApproversRequest) Validate() error {
	var v validator
	switch {
	case trim(r.CertificateID) != "":
		v.certificateID("certificateId", r.CertificateID)
	case trim(r.DomainName) == "":
		v.add("certificateId", "or domainName with packageCode is required: pass CertificateID for an existing certificate, or DomainName and PackageCode before a purchase")
	case trim(r.PackageCode) == "":
		v.add("packageCode", "is required with domainName (the package you are about to buy, see Products.SSLs); for an existing certificate pass CertificateID instead")
	}
	return v.err()
}

// SSLRevokeRequest revokes a certificate. This cannot be undone.
// CLI: dna ssl revoke
type SSLRevokeRequest struct {
	BaseRequest
	CertificateID string
}

func (SSLRevokeRequest) Endpoint() Endpoint {
	return Endpoint{Method: http.MethodDelete, Path: "/api/v1/ssls/revoke", Safety: Destructive}
}

func (r SSLRevokeRequest) Query() url.Values {
	return newParams().str("CertificateId", r.CertificateID).Values
}

func (r SSLRevokeRequest) Validate() error { return requireCertificateID(r.CertificateID) }

// SSLRenewRequest renews a certificate: it opens a new one-year certificate
// linked to it. It charges your balance (the renewal price).
// CLI: dna ssl renew
type SSLRenewRequest struct {
	BaseRequest
	// CertificateID is the certificate to renew.
	CertificateID string
}

func (SSLRenewRequest) Endpoint() Endpoint {
	return Endpoint{Method: http.MethodPost, Path: "/api/v1/ssls/renew", Safety: Billable}
}

func (r SSLRenewRequest) Body() any       { return certificateBody(r.CertificateID) }
func (r SSLRenewRequest) Validate() error { return requireCertificateID(r.CertificateID) }

// SSLListRequest lists your certificates, newest first. CLI: dna ssl list
type SSLListRequest struct {
	BaseRequest
	// ListOptions pages the list. The API's default page size is 10 and the
	// largest is 1000. Sort is not supported: the order is always newest
	// first.
	ListOptions
	// DomainName keeps only the certificates of this domain. For wildcard
	// certificates both "*.example.com" and "example.com" match.
	DomainName string
}

func (SSLListRequest) Endpoint() Endpoint {
	return Endpoint{Method: http.MethodGet, Path: "/api/v1/ssls", Safety: Read}
}

func (r SSLListRequest) Query() url.Values {
	return newParams().str("DomainName", r.DomainName).num("SkipCount", r.Skip).num("MaxResultCount", r.Limit).Values
}

func (r SSLListRequest) Validate() error {
	var v validator
	r.ListOptions.validate(&v)
	v.check(r.Limit <= 1000, "limit", "must be at most 1000")
	v.check(trim(r.Sort) == "", "sort", "is not supported: certificates are always listed newest first")
	return v.err()
}

// GenerateCSR returns a new CSR and its private key.
func (s *SSLService) GenerateCSR(ctx context.Context, req SSLGenerateCSRRequest) (*CSR, error) {
	return call[CSR](ctx, s.client, req)
}

// CheckCSR decodes a PEM-encoded CSR.
func (s *SSLService) CheckCSR(ctx context.Context, csr string) (*CSRInfo, error) {
	return call[CSRInfo](ctx, s.client, SSLCheckCSRRequest{CSR: csr})
}

// Initialize buys a certificate package for a domain and charges your
// balance. Store the CertificateID of the result: Configure and every later
// call take it. Configure the certificate within 15 days; after that it is
// cancelled (refundable packages are refunded automatically).
//
// Do not use Initialize to renew a certificate: see Renew.
func (s *SSLService) Initialize(ctx context.Context, domainName, packageCode string) (*SSLOperationResult, error) {
	return s.operation(ctx, SSLInitializeRequest{DomainName: domainName, PackageCode: packageCode})
}

// Configure submits the CSR, validation method and contact of a certificate
// bought with Initialize or Renew. Then poll Status until IsIssued.
func (s *SSLService) Configure(ctx context.Context, req SSLConfigureRequest) (*SSLOperationResult, error) {
	return s.operation(ctx, req)
}

// Status refreshes and returns the state of a certificate. certificateID is
// the CertificateID returned by Initialize or Renew (List finds it by domain
// name).
func (s *SSLService) Status(ctx context.Context, certificateID string) (*CertificateStatus, error) {
	return call[CertificateStatus](ctx, s.client, SSLStatusRequest{CertificateID: certificateID})
}

// Get returns the full record of a certificate: subscription,
// configuration and contacts.
func (s *SSLService) Get(ctx context.Context, certificateID string) (*Certificate, error) {
	return call[Certificate](ctx, s.client, SSLGetRequest{CertificateID: certificateID})
}

// Content returns the issued certificate and its chain.
func (s *SSLService) Content(ctx context.Context, certificateID string) (*CertificateContent, error) {
	return call[CertificateContent](ctx, s.client, SSLContentRequest{CertificateID: certificateID})
}

// Reissue reissues a certificate with a new CSR, key or approver.
func (s *SSLService) Reissue(ctx context.Context, req SSLReissueRequest) (*SSLOperationResult, error) {
	return s.operation(ctx, req)
}

// Approvers lists the e-mail addresses allowed for e-mail validation: of an
// existing certificate, or of a domain and package before a purchase.
//
//	mails, err := client.SSL.Approvers(ctx, dna.SSLApproversRequest{CertificateID: id})
//	mails, err := client.SSL.Approvers(ctx, dna.SSLApproversRequest{DomainName: "example.com", PackageCode: "PositiveSSL"})
func (s *SSLService) Approvers(ctx context.Context, req SSLApproversRequest) ([]string, error) {
	res, err := call[struct {
		Mails []string `json:"mails"`
	}](ctx, s.client, req)
	if err != nil {
		return nil, err
	}
	return res.Mails, nil
}

// Revoke revokes a certificate. This cannot be undone.
func (s *SSLService) Revoke(ctx context.Context, certificateID string) (*SSLOperationResult, error) {
	return s.operation(ctx, SSLRevokeRequest{CertificateID: certificateID})
}

// Renew renews a certificate and charges your balance (the renewal price).
// It opens a new one-year certificate linked to the old one; the package and
// the domain are taken from the old certificate.
//
// The result's CertificateID is the id of the new certificate, which waits
// for configuration: pass it to Configure within 15 days, then poll Status.
// The old certificate keeps its own id, stays valid until it expires and
// becomes "Renewed" once the new one is active.
//
// The API allows a renewal from 30 days before the subscription ends
// (Certificate.Subscription.ExpiresAt) and only once per certificate; it
// refuses earlier, repeated or otherwise impossible renewals with an
// *APIError. Calling Renew again while the renewal still waits for
// configuration returns the same renewal and does not charge again, so that
// is the way to recover when OutcomeUnknown(err) is true.
func (s *SSLService) Renew(ctx context.Context, certificateID string) (*SSLOperationResult, error) {
	return s.operation(ctx, SSLRenewRequest{CertificateID: certificateID})
}

// List returns one page of your certificates, newest first. Use it to find
// the CertificateID of a certificate by its domain name.
func (s *SSLService) List(ctx context.Context, req SSLListRequest) (*Page[CertificateListItem], error) {
	return call[Page[CertificateListItem]](ctx, s.client, req)
}

// All iterates over every matching certificate, newest first, fetching pages
// as needed:
//
//	for cert, err := range client.SSL.All(ctx, dna.SSLListRequest{DomainName: "example.com"}) {
//		if err != nil {
//			return err
//		}
//		fmt.Println(cert.CertificateID, cert.Status, cert.SubscriptionExpiresAt)
//	}
func (s *SSLService) All(ctx context.Context, req SSLListRequest) iter.Seq2[CertificateListItem, error] {
	return paginate(req.ListOptions, func(o ListOptions) (*Page[CertificateListItem], error) {
		req.ListOptions = o
		return s.List(ctx, req)
	})
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
