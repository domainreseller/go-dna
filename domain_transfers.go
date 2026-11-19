package dna

import (
	"context"
	"net/http"
)

// TransferCheck says whether a domain can be transferred in.
type TransferCheck struct {
	// Transferable is the overall answer.
	Transferable         bool       `json:"transferAvailabilityStatus"`
	MessageKey           string     `json:"messageKey"`
	Message              string     `json:"message"`
	AuthCodeRequired     bool       `json:"authCodeIsRequired"`
	AuthCodeValid        bool       `json:"authCodeIsValid"`
	UserTransferRequired bool       `json:"userTransferRequired"`
	TransferLock         bool       `json:"transferLock"` // the losing registrar still locks the domain
	DomainStatuses       []string   `json:"domainStatuses"`
	AdditionalAttributes Attributes `json:"additionalAttributes"`
	IsPremium            bool       `json:"isPremium"`
	PremiumPrice         float64    `json:"premiumPrice"`
	PremiumCurrency      string     `json:"premiumCurrency"`
}

// TransferStatus is the state of a pending incoming transfer.
type TransferStatus struct {
	DomainName        string `json:"domainName"`
	Status            string `json:"status"`
	RequestedDate     string `json:"requestedDate"`
	FinalDate         string `json:"finalDate"`
	RequestedClientID string `json:"requestedClientId"`
	ActionClientID    string `json:"actionClientId"`
}

// DomainTransferRequest starts an incoming transfer. It charges your
// balance. CLI: dna domain transfer
type DomainTransferRequest struct {
	BaseRequest
	DomainName string
	// AuthCode is the transfer authorization (EPP) code from the current
	// registrar.
	AuthCode string
	// Years added on transfer, 1 to 10 (0 lets the registry decide; usually 1).
	Years int
}

func (DomainTransferRequest) Endpoint() Endpoint {
	return Endpoint{Method: http.MethodPost, Path: "/api/v1/domains/transfer", Safety: Billable}
}

func (r DomainTransferRequest) Body() any {
	// The auth code is sent as given: surrounding spaces can be part of it.
	b := domainBody(r.DomainName).set("period", r.Years)
	if r.AuthCode != "" {
		b["authCode"] = r.AuthCode
	}
	return b
}

func (r DomainTransferRequest) Validate() error {
	var v validator
	v.required("domainName", r.DomainName)
	v.between("years", r.Years, 1, 10)
	return v.err()
}

// DomainTransferCheckRequest checks whether a domain can be transferred in
// with an auth code. CLI: dna domain transfer-check
type DomainTransferCheckRequest struct {
	BaseRequest
	DomainName string
	AuthCode   string
}

func (DomainTransferCheckRequest) Endpoint() Endpoint {
	return Endpoint{Method: http.MethodPost, Path: "/api/v1/domains/transfers/check", Safety: Read}
}

func (r DomainTransferCheckRequest) Body() any {
	return object{"domainName": trim(r.DomainName), "authCode": r.AuthCode}
}

func (r DomainTransferCheckRequest) Validate() error {
	var v validator
	v.required("domainName", r.DomainName)
	v.required("authCode", r.AuthCode)
	return v.err()
}

// DomainTransferStatusRequest reads the state of a pending transfer.
// CLI: dna domain transfer-status
type DomainTransferStatusRequest struct {
	BaseRequest
	DomainName string
}

func (DomainTransferStatusRequest) Endpoint() Endpoint {
	return Endpoint{Method: http.MethodPost, Path: "/api/v1/domains/transfers/query", Safety: Read}
}

func (r DomainTransferStatusRequest) Body() any       { return domainBody(r.DomainName) }
func (r DomainTransferStatusRequest) Validate() error { return requireDomain(r.DomainName) }

// Transfer starts an incoming transfer and charges your balance. Check it
// first with TransferCheck; follow it with TransferStatus.
func (s *DomainService) Transfer(ctx context.Context, req DomainTransferRequest) (*DomainOperationResult, error) {
	return s.operation(ctx, req)
}

// TransferCheck reports whether a domain can be transferred in with
// authCode: lock state, auth code validity, registry statuses and premium
// price.
func (s *DomainService) TransferCheck(ctx context.Context, domainName, authCode string) (*TransferCheck, error) {
	return call[TransferCheck](ctx, s.client, DomainTransferCheckRequest{DomainName: domainName, AuthCode: authCode})
}

// TransferStatus returns the state of a pending incoming transfer.
func (s *DomainService) TransferStatus(ctx context.Context, domainName string) (*TransferStatus, error) {
	return call[TransferStatus](ctx, s.client, DomainTransferStatusRequest{DomainName: domainName})
}
