package dna

import (
	"context"
	"net/http"
	"net/url"
	"strings"
	"unicode"
)

// ContactService manages reusable contact handles: create a contact once,
// then register domains with its handle. CLI: dna contact ...
type ContactService service

// Contact is a person or organisation in a contact role.
//
// The requests of this package tidy a contact before sending it: they trim
// every field, upper-case the country and remove "+", spaces and dashes
// from the phone and fax fields.
type Contact struct {
	FirstName   string `json:"firstName"`
	LastName    string `json:"lastName"`
	CompanyName string `json:"companyName,omitempty"` // optional
	Email       string `json:"eMail"`
	Address     string `json:"address"`
	City        string `json:"city"`
	// State is required; repeat the city when there is none.
	State      string `json:"state"`
	PostalCode string `json:"postalCode"`
	// Country is the ISO 3166-1 alpha-2 code, e.g. "TR" ("GB", not "UK").
	Country string `json:"country"`
	// PhoneCountryCode is the calling code without "+", e.g. "90".
	PhoneCountryCode string `json:"phoneCountryCode"`
	// Phone is the number without the calling code, e.g. "5551234567".
	Phone          string `json:"phone"`
	FaxCountryCode string `json:"faxCountryCode,omitempty"`
	Fax            string `json:"fax,omitempty"`
	// DiscloseFlag publishes the contact in WHOIS. Leave it nil to keep the
	// registry default (recommended).
	DiscloseFlag *bool `json:"discloseFlag,omitempty"`
	// Type is the role. The API sets it on the contacts of a domain
	// (DomainInfo.Contacts); the requests that need it set it themselves.
	Type ContactType `json:"contactType,omitempty"`
	// Handle is the contact's handle, e.g. "DNA-12345". Set by the API.
	Handle string `json:"handle,omitempty"`
}

// normalized returns a tidied copy of c, as sent to the API.
func (c Contact) normalized() Contact {
	t := strings.TrimSpace
	c.FirstName, c.LastName, c.CompanyName = t(c.FirstName), t(c.LastName), t(c.CompanyName)
	c.Email, c.Address, c.City, c.State = t(c.Email), t(c.Address), t(c.City), t(c.State)
	c.PostalCode, c.Country = t(c.PostalCode), strings.ToUpper(t(c.Country))
	c.PhoneCountryCode, c.Phone = digits(c.PhoneCountryCode), digits(c.Phone)
	c.FaxCountryCode, c.Fax = digits(c.FaxCountryCode), digits(c.Fax)
	c.Handle = t(c.Handle)
	return c
}

// digits keeps only the digits of s: "+90 (555) 123-45-67" becomes
// "905551234567".
func digits(s string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsDigit(r) {
			return r
		}
		return -1
	}, s)
}

// validate checks the fields the API requires of a contact.
func (c Contact) validate(v *validator, prefix string) {
	c = c.normalized()
	f := func(name string) string { return prefix + name }
	v.required(f("firstName"), c.FirstName)
	v.required(f("lastName"), c.LastName)
	v.required(f("email"), c.Email)
	v.check(c.Email == "" || strings.Contains(c.Email, "@"), f("email"), "is not an e-mail address")
	v.required(f("address"), c.Address)
	v.required(f("city"), c.City)
	v.required(f("state"), c.State)
	v.required(f("postalCode"), c.PostalCode)
	v.required(f("country"), c.Country)
	v.check(c.Country == "" || len(c.Country) == 2, f("country"), "must be a 2-letter ISO code such as TR or US")
	v.required(f("phoneCountryCode"), c.PhoneCountryCode)
	v.check(len(c.PhoneCountryCode) <= 3, f("phoneCountryCode"), "must be at most 3 digits, e.g. 90")
	v.required(f("phone"), c.Phone)
	v.check(len(c.Phone) <= 16, f("phone"), "must be at most 16 digits")
	v.maxLen(f("firstName"), c.FirstName, 80)
	v.maxLen(f("lastName"), c.LastName, 80)
	v.maxLen(f("companyName"), c.CompanyName, 256)
	v.maxLen(f("address"), c.Address, 256)
	v.maxLen(f("city"), c.City, 80)
	v.maxLen(f("state"), c.State, 80)
	v.maxLen(f("postalCode"), c.PostalCode, 15)
}

// info is the contact without role and handle (ContactInformationDto).
func (c Contact) info() Contact {
	c = c.normalized()
	c.Type, c.Handle = "", ""
	return c
}

// ContactVerification is the e-mail or phone verification state of a
// contact. A domain stays usable after registration, but when the
// registrant e-mail is not verified within 15 days the domain is put on
// hold.
type ContactVerification struct {
	OwnerName        string `json:"ownerName"`
	FieldValue       string `json:"fieldValue"` // the e-mail address or phone number
	FieldType        string `json:"fieldType"`
	OperationStatus  string `json:"operationStatus"`
	Verified         bool   `json:"isUserVerified"`
	VerifiedAt       Time   `json:"userVerificationDate"`
	CreatedAt        Time   `json:"createdDate"`
	TokenExpiresAt   Time   `json:"tokenExpireDate"`
	ResentAt         Time   `json:"resendDate"`
	ResendRetryCount int    `json:"resendRetryCount"`
	LastAttemptAt    Time   `json:"lastAttemptDate"`
	AttemptNumber    int    `json:"attemptNumber"`
}

// ContactDetails is a contact handle with its verification state.
type ContactDetails struct {
	Contact       Contact               `json:"contact"`
	Verifications []ContactVerification `json:"verificationObjects"`
}

// ContactCreateResult is the API's answer to ContactCreateRequest.
type ContactCreateResult struct {
	Handle  string `json:"handle"`
	Message string `json:"operationMessage"`
	Success bool   `json:"isSuccess"`
}

// ContactUpdateResult is the API's answer to ContactUpdateRequest: the
// contact itself and every domain that uses it.
type ContactUpdateResult struct {
	Handle  string `json:"handle"`
	Success bool   `json:"contactUpdateSuccess"`
	Stats   struct {
		TotalDomainCount     int `json:"totalDomainCount"`
		SuccessCount         int `json:"successCount"`
		PendingDocumentCount int `json:"pendingDocumentCount"`
		FailedCount          int `json:"failedCount"`
		SplitCount           int `json:"splitCount"`
	} `json:"stats"`
	// AffectedDomains reports each domain. Some registries cannot share a
	// handle and get a new one: store NewHandle when it is set.
	AffectedDomains []struct {
		DomainName string `json:"domainName"`
		Status     string `json:"status"`
		NewHandle  string `json:"newHandle"`
		Reason     string `json:"reason"`
		Message    string `json:"message"`
	} `json:"affectedDomains"`
}

// ContactGetRequest reads a contact handle. CLI: dna contact get
type ContactGetRequest struct {
	BaseRequest
	Handle string
}

func (r ContactGetRequest) Endpoint() Endpoint {
	return Endpoint{Method: http.MethodGet, Path: "/api/v1/contacts/" + url.PathEscape(trim(r.Handle)), Safety: Read}
}

func (r ContactGetRequest) Validate() error {
	var v validator
	v.required("handle", r.Handle)
	return v.err()
}

// ContactCreateRequest creates a contact handle. CLI: dna contact create
type ContactCreateRequest struct {
	BaseRequest
	Contact Contact
}

func (ContactCreateRequest) Endpoint() Endpoint {
	return Endpoint{Method: http.MethodPost, Path: "/api/v1/contacts", Safety: Write}
}

func (r ContactCreateRequest) Body() any { return object{"contact": r.Contact.info()} }

func (r ContactCreateRequest) Validate() error {
	var v validator
	r.Contact.validate(&v, "contact.")
	return v.err()
}

// ContactUpdateRequest replaces a contact handle's details, and those of
// every domain using it. CLI: dna contact update
type ContactUpdateRequest struct {
	BaseRequest
	Handle string
	// Contact is the complete new contact: the update is a full
	// replacement, not a patch.
	Contact Contact
}

func (r ContactUpdateRequest) Endpoint() Endpoint {
	return Endpoint{Method: http.MethodPut, Path: "/api/v1/contacts/" + url.PathEscape(trim(r.Handle)), Safety: Write}
}

func (r ContactUpdateRequest) Body() any { return object{"contact": r.Contact.info()} }

func (r ContactUpdateRequest) Validate() error {
	var v validator
	v.required("handle", r.Handle)
	r.Contact.validate(&v, "contact.")
	return v.err()
}

// ContactVerificationRequest reads the verification state of a contact.
// CLI: dna contact verification
type ContactVerificationRequest struct {
	BaseRequest
	Handle string
}

func (ContactVerificationRequest) Endpoint() Endpoint {
	return Endpoint{Method: http.MethodGet, Path: "/api/v1/contacts/verification/check", Safety: Read}
}

func (r ContactVerificationRequest) Query() url.Values {
	return newParams().str("HandleCode", r.Handle).Values
}

func (r ContactVerificationRequest) Validate() error {
	var v validator
	v.required("handle", r.Handle)
	return v.err()
}

// ContactResendVerificationRequest sends the verification notice again.
// CLI: dna contact resend-verification
type ContactResendVerificationRequest struct {
	BaseRequest
	Handle string
}

func (ContactResendVerificationRequest) Endpoint() Endpoint {
	return Endpoint{Method: http.MethodPost, Path: "/api/v1/contacts/verification/request", Safety: Write}
}

func (r ContactResendVerificationRequest) Body() any { return object{"handleCode": trim(r.Handle)} }

func (r ContactResendVerificationRequest) Validate() error {
	var v validator
	v.required("handle", r.Handle)
	return v.err()
}

// Get returns a contact handle and its verification state.
func (s *ContactService) Get(ctx context.Context, handle string) (*ContactDetails, error) {
	return call[ContactDetails](ctx, s.client, ContactGetRequest{Handle: handle})
}

// Create creates a contact and returns its handle, which
// Domains.Register accepts (see SameHandle).
func (s *ContactService) Create(ctx context.Context, contact Contact) (handle string, err error) {
	req := ContactCreateRequest{Contact: contact}
	res, resp, err := send[ContactCreateResult](ctx, s.client, req)
	if err != nil {
		return "", err
	}
	if !res.Success || res.Handle == "" {
		return "", operationError(req, resp, "", res.Message)
	}
	return res.Handle, nil
}

// Update replaces the details of a contact handle. Every domain using the
// handle is updated too; the result reports each of them.
func (s *ContactService) Update(ctx context.Context, handle string, contact Contact) (*ContactUpdateResult, error) {
	req := ContactUpdateRequest{Handle: handle, Contact: contact}
	res, resp, err := send[ContactUpdateResult](ctx, s.client, req)
	if err != nil {
		return nil, err
	}
	if !res.Success {
		return nil, operationError(req, resp, "", "the contact could not be updated")
	}
	return res, nil
}

// Verification returns the e-mail and phone verification state of a contact.
func (s *ContactService) Verification(ctx context.Context, handle string) ([]ContactVerification, error) {
	res, err := call[struct {
		Results []ContactVerification `json:"results"`
	}](ctx, s.client, ContactVerificationRequest{Handle: handle})
	if err != nil {
		return nil, err
	}
	return res.Results, nil
}

// ResendVerification sends the verification notice of a contact again. It
// also returns nil when there is nothing to verify (already verified, or
// nothing pending).
func (s *ContactService) ResendVerification(ctx context.Context, handle string) error {
	err := exec(ctx, s.client, ContactResendVerificationRequest{Handle: handle})
	if HasCode(err, CodeVerificationAlreadyDone) || HasCode(err, CodeVerificationNotPending) {
		return nil
	}
	return err
}
