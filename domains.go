package dna

import (
	"context"
	"iter"
	"net/http"
	"net/url"
	"strings"
)

// DomainService searches, registers, renews, transfers and manages domains.
// CLI: dna domain ...
type DomainService service

// Availability is the result of a domain search.
type Availability struct {
	DomainName string `json:"domainName"`
	TLD        string `json:"tld"`
	// Status is the registry answer: "AVAILABLE", "NOTAVAILABLE", ...
	// Use Available to test it.
	Status string `json:"status"`
	// Reason explains a status other than AVAILABLE, e.g. "Domain exists".
	Reason             string  `json:"reason"`
	Price              float64 `json:"price"`
	Currency           string  `json:"currency"`
	Period             int     `json:"period"`
	IsPremium          bool    `json:"isPremium"`
	IsDocumentRequired bool    `json:"isDocumentRequired"` // the registry asks for documents (Domains.UploadDocuments)
}

// Available reports whether the domain can be registered.
func (a Availability) Available() bool { return strings.EqualFold(a.Status, "AVAILABLE") }

// DomainSearchResult is the API's answer to DomainSearchRequest.
type DomainSearchResult struct {
	Success bool         `json:"success"`
	Code    string       `json:"code"`
	Message string       `json:"operationMessage"`
	Reason  string       `json:"reason"`
	Info    Availability `json:"info"`
}

// DomainBulkSearchResult is the API's answer to DomainBulkSearchRequest.
type DomainBulkSearchResult struct {
	Success bool           `json:"success"`
	Code    string         `json:"code"`
	Message string         `json:"operationMessage"`
	Reason  string         `json:"reason"`
	Infos   []Availability `json:"infos"`
}

// NameServer is a nameserver of a domain in the domain list.
type NameServer struct {
	Name              string `json:"name"`
	IsChildNameServer bool   `json:"isChildnameServer"`
	IsGlueRecord      bool   `json:"isGlueRecord"`
	IsLinked          bool   `json:"isLinked"`
}

// Domain is a domain in the domain list. Domains.Info returns more.
type Domain struct {
	DomainName    string       `json:"domainName"`
	DomainNameIDN string       `json:"domainNameIdn"`
	IsIDN         bool         `json:"isIdnName"`
	Status        DomainStatus `json:"status"`
	// StatusCode is the registry (EPP) status, e.g. "clientTransferProhibited".
	StatusCode        string `json:"statusCode"`
	StartDate         Time   `json:"startDate"`
	ExpirationDate    Time   `json:"expirationDate"`
	TransferDate      Time   `json:"transferDate"`
	UpdatedDate       Time   `json:"updatedDate"`
	RGPExpirationDate Time   `json:"rgpExpirationDate"` // end of the redemption grace period
	AGPExpirationDate Time   `json:"agpExpirationDate"` // end of the add grace period
	FailureDate       Time   `json:"failureDate"`
	FinalizationDate  Time   `json:"finalizationDate"`
	RemainingDays     int    `json:"remainingDay"`
	Locked            bool   `json:"lockStatus"`
	PrivacyProtection bool   `json:"privacyProtectionStatus"`
	RenewalMode       string `json:"renewalMode"`
	ObjectClass       string `json:"objectClass"`
	// Contacts maps the contact roles to their handles.
	Contacts    Attributes   `json:"contacts"`
	NameServers []NameServer `json:"nameservers"`
}

// Host is a child nameserver (glue record) such as ns1.example.com.
type Host struct {
	Name        string      `json:"name"`
	IPAddresses []IPAddress `json:"ipAddresses"`
}

// IPAddress is an address of a child nameserver.
type IPAddress struct {
	Address string    `json:"ipAddress"`
	Version IPVersion `json:"ipVersion,omitempty"`
}

// DomainInfo is the full record of a domain.
type DomainInfo struct {
	DomainName    string       `json:"domainName"`
	DomainNameIDN string       `json:"domainNameIdn"`
	IsIDN         bool         `json:"isIdnName"`
	Status        DomainStatus `json:"status"`
	// StatusCode is the registry (EPP) status, e.g. "clientTransferProhibited".
	StatusCode string `json:"statusCode"`
	// StatusReason explains the status, e.g. "RddsVerificationPending".
	StatusReason string `json:"statusReason"`
	// RGPStatus is the registry's redemption grace period status.
	RGPStatus      string `json:"rgpStatues"`
	StartDate      Time   `json:"startDate"`
	ExpirationDate Time   `json:"expirationDate"`
	TransferDate   Time   `json:"transferDate"`
	UpdatedDate    Time   `json:"updatedDate"`
	RemainingDays  int    `json:"remainingDay"`
	ResellerID     string `json:"resellerId"`
	RenewalMode    string `json:"renewalMode"`
	// AuthCode is the transfer authorization (EPP) code.
	AuthCode          string   `json:"authCode"`
	Locked            bool     `json:"lockStatus"`
	PrivacyProtection bool     `json:"privacyProtectionStatus"`
	ObjectID          string   `json:"objectId"`
	IsForwarded       bool     `json:"isForwarded"`
	UseTrusteeContact bool     `json:"useTrusteeContact"`
	NameServers       []string `json:"nameservers"`
	// Hosts are the child nameservers registered under the domain.
	Hosts                []Host     `json:"hosts"`
	Contacts             []Contact  `json:"contacts"`
	AdditionalAttributes Attributes `json:"additionalAttributes"`
}

// Contact returns the contact in a role.
func (d *DomainInfo) Contact(role ContactType) (Contact, bool) {
	for _, c := range d.Contacts {
		if c.Type == role {
			return c, true
		}
	}
	return Contact{}, false
}

// DomainOperationResult is the API's answer to a registration, renewal or
// transfer.
type DomainOperationResult struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	// DomainID identifies the domain on the platform.
	DomainID string `json:"domainId"`
	// Status is the domain's status after the operation, e.g. "Active" or a
	// waiting status.
	Status         string `json:"status"`
	ExpirationDate Time   `json:"expirationDate"`
}

// RenewCheck says whether and at what price a domain can be renewed.
type RenewCheck struct {
	Renewable       bool    `json:"renewabilityStatus"`
	RenewalLock     bool    `json:"renewalLock"`
	ErrorMessageKey string  `json:"errorMessageKey"`
	IsPremium       bool    `json:"isPremium"`
	PremiumPrice    float64 `json:"premiumPrice"`
	PremiumCurrency string  `json:"premiumCurrency"`
}

// ContactHandles names the contact handle of each role.
type ContactHandles struct {
	Registrant, Admin, Tech, Billing string
}

// SameHandle uses one handle for all four roles.
func SameHandle(handle string) ContactHandles {
	return ContactHandles{Registrant: handle, Admin: handle, Tech: handle, Billing: handle}
}

type contactHandle struct {
	Type   ContactType `json:"contactType"`
	Handle string      `json:"handle"`
}

// items returns the roles that are set, in the order Registrant, Admin,
// Tech, Billing.
func (h ContactHandles) items() []contactHandle {
	var out []contactHandle
	for _, it := range []contactHandle{
		{ContactTypeRegistrant, h.Registrant}, {ContactTypeAdmin, h.Admin},
		{ContactTypeTech, h.Tech}, {ContactTypeBilling, h.Billing},
	} {
		if it.Handle = trim(it.Handle); it.Handle != "" {
			out = append(out, it)
		}
	}
	return out
}

func (h ContactHandles) validate(v *validator, field string, all bool) {
	n := len(h.items())
	switch {
	case all && n < 4:
		v.add(field, "needs a handle for every role (Registrant, Admin, Tech, Billing); SameHandle sets all four")
	case n == 0:
		v.add(field, "needs at least one handle")
	}
}

// DomainContacts holds the contact of each role.
type DomainContacts struct {
	Registrant, Admin, Tech, Billing *Contact
}

// SameContact uses one contact for all four roles.
func SameContact(c Contact) DomainContacts {
	return DomainContacts{Registrant: &c, Admin: &c, Tech: &c, Billing: &c}
}

// list returns the contacts that are set, each with its role.
func (d DomainContacts) list() []Contact {
	var out []Contact
	for _, it := range []struct {
		role ContactType
		c    *Contact
	}{{ContactTypeRegistrant, d.Registrant}, {ContactTypeAdmin, d.Admin}, {ContactTypeTech, d.Tech}, {ContactTypeBilling, d.Billing}} {
		if it.c != nil {
			c := it.c.info()
			c.Type = it.role
			out = append(out, c)
		}
	}
	return out
}

func (d DomainContacts) validate(v *validator, field string, all bool) {
	for _, it := range []struct {
		role ContactType
		c    *Contact
	}{{ContactTypeRegistrant, d.Registrant}, {ContactTypeAdmin, d.Admin}, {ContactTypeTech, d.Tech}, {ContactTypeBilling, d.Billing}} {
		switch {
		case it.c != nil:
			it.c.validate(v, field+"."+string(it.role)+".")
		case all:
			v.add(field+"."+string(it.role), "is required (SameContact sets all four roles)")
		}
	}
	if !all && len(d.list()) == 0 {
		v.add(field, "needs at least one contact")
	}
}

// ---------------------------------------------------------------------------
// Requests

// DomainSearchRequest checks the availability and price of a domain.
// CLI: dna domain search
type DomainSearchRequest struct {
	BaseRequest
	DomainName string
	// Currency of the price, e.g. "USD" or "TRY". Empty uses the account's.
	Currency string
}

func (DomainSearchRequest) Endpoint() Endpoint {
	return Endpoint{Method: http.MethodPost, Path: "/api/v1/domains/search", Safety: Read}
}

func (r DomainSearchRequest) Body() any {
	return domainBody(r.DomainName).set("currency", r.Currency)
}

func (r DomainSearchRequest) Validate() error {
	var v validator
	v.required("domainName", r.DomainName)
	return v.err()
}

// MaxBulkSearch is the number of domains one DomainBulkSearchRequest may
// check. Domains.BulkSearch splits longer lists for you.
const MaxBulkSearch = 30

// DomainBulkSearchRequest checks several domains at once (at most
// MaxBulkSearch). CLI: dna domain bulk-search
type DomainBulkSearchRequest struct {
	BaseRequest
	DomainNames []string
	// Currency of the prices, e.g. "USD" or "TRY". Empty uses the account's.
	Currency string
}

func (DomainBulkSearchRequest) Endpoint() Endpoint {
	return Endpoint{Method: http.MethodPost, Path: "/api/v1/domains/bulk-search", Safety: Read}
}

func (r DomainBulkSearchRequest) Body() any {
	names := trimAll(r.DomainNames)
	body := make([]object, len(names))
	for i, name := range names {
		body[i] = domainBody(name).set("currency", r.Currency)
	}
	return body
}

func (r DomainBulkSearchRequest) Validate() error {
	var v validator
	n := len(trimAll(r.DomainNames))
	v.requiredList("domainNames", n)
	v.check(n <= MaxBulkSearch, "domainNames", "may hold at most 30 domains; Domains.BulkSearch splits longer lists")
	return v.err()
}

// DomainListRequest lists your domains. CLI: dna domain list
type DomainListRequest struct {
	BaseRequest
	ListOptions
	// DomainName keeps domains whose name contains this text.
	DomainName string
	// Statuses keeps only domains in these states.
	Statuses []DomainStatus
}

func (DomainListRequest) Endpoint() Endpoint {
	return Endpoint{Method: http.MethodGet, Path: "/api/v1/domains", Safety: Read}
}

func (r DomainListRequest) Query() url.Values {
	return newParams().str("DomainName", r.DomainName).joined("StatusIds", intStrings(r.Statuses)).page(r.ListOptions).Values
}

func (r DomainListRequest) Validate() error {
	var v validator
	r.ListOptions.validate(&v)
	return v.err()
}

// DomainInfoRequest reads the full record of a domain. CLI: dna domain info
type DomainInfoRequest struct {
	BaseRequest
	DomainName string
}

func (DomainInfoRequest) Endpoint() Endpoint {
	return Endpoint{Method: http.MethodGet, Path: "/api/v1/domains/info", Safety: Read}
}

func (r DomainInfoRequest) Query() url.Values {
	return newParams().str("DomainName", r.DomainName).Values
}

func (r DomainInfoRequest) Validate() error { return requireDomain(r.DomainName) }

// DomainRegisterRequest registers a domain with existing contact handles.
// It charges your balance. CLI: dna domain register
type DomainRegisterRequest struct {
	BaseRequest
	DomainName string
	// Years is the registration period, 1 to 10 (0 means 1).
	Years int
	// NameServers are the nameserver host names; most registries want two
	// or more.
	NameServers []string
	// Handles are the contact handles of the four roles (see SameHandle and
	// Contacts.Create).
	Handles ContactHandles
	// TLDAttributes are the registry attributes some TLDs require; see
	// TLD.Attributes in Products.TLDs.
	TLDAttributes Attributes
}

func (DomainRegisterRequest) Endpoint() Endpoint {
	return Endpoint{Method: http.MethodPost, Path: "/api/v1/domains/register", Safety: Billable}
}

func (r DomainRegisterRequest) Body() any {
	return domainBody(r.DomainName).
		set("period", r.Years).
		set("nameServers", trimAll(r.NameServers)).
		set("handles", r.Handles.items()).
		set("tldAttributes", r.TLDAttributes)
}

func (r DomainRegisterRequest) Validate() error {
	var v validator
	v.required("domainName", r.DomainName)
	v.between("years", r.Years, 1, 10)
	v.requiredList("nameServers", len(trimAll(r.NameServers)))
	r.Handles.validate(&v, "handles", true)
	return v.err()
}

// DomainRegisterWithContactsRequest registers a domain with the contact
// details given inline. It charges your balance.
// CLI: dna domain register-with-contacts
type DomainRegisterWithContactsRequest struct {
	BaseRequest
	DomainName string
	// Years is the registration period, 1 to 10 (0 means 1).
	Years       int
	NameServers []string
	// Contacts are required for all four roles unless UseTrusteeContact is
	// set (see SameContact).
	Contacts      DomainContacts
	TLDAttributes Attributes
	// UseTrusteeContact uses the registrar's trustee contact, for TLDs with
	// TLD.IsSupportTrustee.
	UseTrusteeContact bool
}

func (DomainRegisterWithContactsRequest) Endpoint() Endpoint {
	return Endpoint{Method: http.MethodPost, Path: "/api/v1/domains/register-with-contacts", Safety: Billable}
}

func (r DomainRegisterWithContactsRequest) Body() any {
	b := domainBody(r.DomainName).
		set("period", r.Years).
		set("nameServers", trimAll(r.NameServers)).
		set("tldAttributes", r.TLDAttributes)
	if contacts := r.Contacts.list(); len(contacts) > 0 {
		b["contacts"] = contacts
	}
	if r.UseTrusteeContact {
		b["useTrusteeContact"] = true
	}
	return b
}

func (r DomainRegisterWithContactsRequest) Validate() error {
	var v validator
	v.required("domainName", r.DomainName)
	v.between("years", r.Years, 1, 10)
	v.requiredList("nameServers", len(trimAll(r.NameServers)))
	if !r.UseTrusteeContact || len(r.Contacts.list()) > 0 {
		r.Contacts.validate(&v, "contacts", !r.UseTrusteeContact)
	}
	return v.err()
}

// DomainRenewRequest renews a domain. It charges your balance.
// CLI: dna domain renew
type DomainRenewRequest struct {
	BaseRequest
	DomainName string
	// Years to add, 1 to 10.
	Years int
}

func (DomainRenewRequest) Endpoint() Endpoint {
	return Endpoint{Method: http.MethodPost, Path: "/api/v1/domains/renew", Safety: Billable}
}

func (r DomainRenewRequest) Body() any { return domainBody(r.DomainName).set("period", r.Years) }

func (r DomainRenewRequest) Validate() error {
	var v validator
	v.required("domainName", r.DomainName)
	v.check(r.Years >= 1 && r.Years <= 10, "years", "must be between 1 and 10")
	return v.err()
}

// DomainRenewCheckRequest checks whether a domain can be renewed.
// CLI: dna domain renew-check
type DomainRenewCheckRequest struct {
	BaseRequest
	DomainName string
	// Years to add, 1 to 10 (0 lets the API choose).
	Years int
}

func (DomainRenewCheckRequest) Endpoint() Endpoint {
	return Endpoint{Method: http.MethodPost, Path: "/api/v1/domains/renew/check", Safety: Read}
}

func (r DomainRenewCheckRequest) Body() any { return domainBody(r.DomainName).set("period", r.Years) }

func (r DomainRenewCheckRequest) Validate() error {
	var v validator
	v.required("domainName", r.DomainName)
	v.between("years", r.Years, 1, 10)
	return v.err()
}

// DomainRestoreRequest restores a deleted domain during the registry's
// redemption period (DomainStatusPendingDelete). It charges your balance;
// restore fees are usually high. CLI: dna domain restore
type DomainRestoreRequest struct {
	BaseRequest
	DomainName string
}

func (DomainRestoreRequest) Endpoint() Endpoint {
	return Endpoint{Method: http.MethodPost, Path: "/api/v1/domains/restore", Safety: Billable}
}

func (r DomainRestoreRequest) Body() any       { return domainBody(r.DomainName) }
func (r DomainRestoreRequest) Validate() error { return requireDomain(r.DomainName) }

func requireDomain(name string) error {
	var v validator
	v.required("domainName", name)
	return v.err()
}

// ---------------------------------------------------------------------------
// Service methods

// Search checks whether a domain is available and what it costs. Prices use
// the client's currency (WithCurrency) or the account's. It allows 2
// searches per second; the client paces them.
//
//	a, err := client.Domains.Search(ctx, "example.com")
//	if err == nil && a.Available() {
//		fmt.Println(a.Price, a.Currency)
//	}
func (s *DomainService) Search(ctx context.Context, domainName string) (*Availability, error) {
	req := DomainSearchRequest{DomainName: domainName, Currency: s.client.currency}
	res, resp, err := send[DomainSearchResult](ctx, s.client, req)
	if err != nil {
		return nil, err
	}
	if !res.Success {
		return nil, operationError(req, resp, res.Code, firstNonEmpty(res.Reason, res.Message))
	}
	return &res.Info, nil
}

// BulkSearch checks several domains. Lists longer than MaxBulkSearch are
// sent in several requests. Each result has its own Status. When a request
// fails, the results of the earlier ones are returned with the error.
func (s *DomainService) BulkSearch(ctx context.Context, domainNames ...string) ([]Availability, error) {
	names := trimAll(domainNames)
	if len(names) == 0 {
		return nil, &InputError{Request: "DomainBulkSearchRequest", Fields: []FieldError{{Field: "domainNames", Message: "needs at least one value"}}}
	}
	var out []Availability
	for start := 0; start < len(names); start += MaxBulkSearch {
		end := min(start+MaxBulkSearch, len(names))
		req := DomainBulkSearchRequest{DomainNames: names[start:end], Currency: s.client.currency}
		res, resp, err := send[DomainBulkSearchResult](ctx, s.client, req)
		if err != nil {
			return out, err
		}
		if !res.Success && len(res.Infos) == 0 {
			return out, operationError(req, resp, res.Code, firstNonEmpty(res.Reason, res.Message))
		}
		out = append(out, res.Infos...)
	}
	return out, nil
}

// List returns one page of your domains.
func (s *DomainService) List(ctx context.Context, req DomainListRequest) (*Page[Domain], error) {
	return call[Page[Domain]](ctx, s.client, req)
}

// All iterates over every matching domain, fetching pages as needed:
//
//	for d, err := range client.Domains.All(ctx, dna.DomainListRequest{}) {
//		if err != nil {
//			return err
//		}
//		if d.RemainingDays < 30 {
//			fmt.Println(d.DomainName, "expires", d.ExpirationDate)
//		}
//	}
func (s *DomainService) All(ctx context.Context, req DomainListRequest) iter.Seq2[Domain, error] {
	return paginate(req.ListOptions, func(o ListOptions) (*Page[Domain], error) {
		req.ListOptions = o
		return s.List(ctx, req)
	})
}

// Info returns the full record of a domain: dates, status, lock and
// privacy, nameservers, child hosts, contacts and the auth code. It is not
// rate limited.
func (s *DomainService) Info(ctx context.Context, domainName string) (*DomainInfo, error) {
	return call[DomainInfo](ctx, s.client, DomainInfoRequest{DomainName: domainName})
}

// Register registers a domain with existing contact handles and charges
// your balance. Check availability and price with Search first.
//
// If the API answers but refuses (the domain is taken, a rule failed), the
// error is an *OperationError. After a network error, check
// OutcomeUnknown(err) and the domain's state before trying again.
func (s *DomainService) Register(ctx context.Context, req DomainRegisterRequest) (*DomainOperationResult, error) {
	return s.operation(ctx, req)
}

// RegisterWithContacts registers a domain with inline contact details and
// charges your balance. See Register.
func (s *DomainService) RegisterWithContacts(ctx context.Context, req DomainRegisterWithContactsRequest) (*DomainOperationResult, error) {
	return s.operation(ctx, req)
}

// Renew adds years (1 to 10) to a domain and charges your balance. Use
// RenewCheck first to see whether it is renewable and at what price.
func (s *DomainService) Renew(ctx context.Context, domainName string, years int) (*DomainOperationResult, error) {
	return s.operation(ctx, DomainRenewRequest{DomainName: domainName, Years: years})
}

// RenewCheck reports whether a domain can be renewed for years (0 lets the
// API choose) and the premium price, if any.
func (s *DomainService) RenewCheck(ctx context.Context, domainName string, years int) (*RenewCheck, error) {
	return call[RenewCheck](ctx, s.client, DomainRenewCheckRequest{DomainName: domainName, Years: years})
}

// Restore brings back a deleted domain during the redemption period and
// charges your balance.
func (s *DomainService) Restore(ctx context.Context, domainName string) error {
	return exec(ctx, s.client, DomainRestoreRequest{DomainName: domainName})
}

// operation sends a request answered with a DomainOperationResult and turns
// "success": false into an *OperationError.
func (s *DomainService) operation(ctx context.Context, req Request) (*DomainOperationResult, error) {
	res, resp, err := send[DomainOperationResult](ctx, s.client, req)
	if err != nil {
		return nil, err
	}
	if !res.Success {
		return nil, operationError(req, resp, "", res.Message)
	}
	return res, nil
}
