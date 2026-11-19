package dna

import (
	"context"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// HostingService orders and inspects web hosting. CLI: dna hosting ...
//
// The flow: pick a plan from Packages, Create (charges the balance; the API
// answers 202 while provisioning runs), then Wait (or poll Status) until the
// hosting is active. Never create again for a hosting stuck in
// provisioning: that charges twice.
type HostingService service

// Poll intervals of HostingService.Wait (variables so tests can shorten them).
var (
	hostingPollFast = 5 * time.Second
	hostingPollSlow = 30 * time.Second
)

// Hosting provisioning states (HostingStatus.Status, Hosting.Status).
const (
	HostingProvisioning = "Provisioning"
	HostingActive       = "Active"
	HostingFailed       = "Failed" // the balance is refunded automatically
	HostingSuspended    = "Suspended"
	HostingDeleted      = "Deleted"
)

// HostingPackage is a hosting package and the plans it offers.
type HostingPackage struct {
	Name                string `json:"name"`
	RequiredMinLeasing  int    `json:"requiredMinLeasing"`
	PermittedMaxLeasing int    `json:"permittedMaxLeasing"`
	ServicePlans        []struct {
		// ServicePlan is the value HostingService.Create takes.
		ServicePlan     string `json:"servicePlan"`
		OperatingSystem string `json:"operatingSystem"` // part of the plan; cannot be changed later
		PanelType       string `json:"panelType"`
	} `json:"servicePlans"`
}

// ServicePlan is a hosting plan and its limits. A nil limit means
// unlimited, not zero.
type ServicePlan struct {
	Name             string `json:"name"`
	ExternalID       string `json:"externalId"`
	DiskQuotaMB      *int64 `json:"diskQuotaMb"`
	BandwidthLimitMB *int64 `json:"bandwidthLimitMb"`
	MaxDomains       *int   `json:"maxDomains"`
	MaxSubDomains    *int   `json:"maxSubDomains"`
	MaxDatabases     *int   `json:"maxDatabases"`
	MaxFTPAccounts   *int   `json:"maxFtpAccounts"`
	MaxEmailAccounts *int   `json:"maxEmailAccounts"`
	HasDedicatedIP   bool   `json:"hasDedicatedIp"`
	HasShellAccess   bool   `json:"hasShellAccess"`
}

// HostingCreateResult is the API's answer to HostingCreateRequest.
type HostingCreateResult struct {
	// ServiceID ("HST-...") identifies the hosting; store it.
	ServiceID string `json:"serviceId"`
	Status    string `json:"status"`
	// The panel access fields are empty until the hosting is active.
	PanelAddress string `json:"panelAddress"`
	Username     string `json:"username"`
	Password     string `json:"password"`
	Token        string `json:"token"`
}

// Hosting is a hosting service.
type Hosting struct {
	// HostingID is the UUID HostingService.Plans takes.
	HostingID          string `json:"hostingId"`
	ServiceID          string `json:"serviceId"`
	DomainName         string `json:"domainName"`
	PackageName        string `json:"packageName"`
	LeaseOn            Time   `json:"leaseOn"`
	ExpiresOn          Time   `json:"expiredOn"`
	Status             string `json:"status"`
	ExternalCustomerID string `json:"externalCustomerId"`
}

// HostingStatus is the provisioning state of a hosting. The panel access
// fields are set only once Status is HostingActive.
type HostingStatus struct {
	ServiceID     string `json:"serviceId"`
	DomainName    string `json:"domainName"`
	Status        string `json:"status"`
	StatusMessage string `json:"statusMessage"`
	PanelAddress  string `json:"panelAddress"`
	Username      string `json:"username"`
	Password      string `json:"password"`
	Token         string `json:"token"`
	FailureReason string `json:"failureReason"`
}

// Active reports whether the hosting is ready to use.
func (s HostingStatus) Active() bool { return strings.EqualFold(s.Status, HostingActive) }

// Failed reports whether provisioning failed.
func (s HostingStatus) Failed() bool { return strings.EqualFold(s.Status, HostingFailed) }

// HostingPackagesRequest lists hosting packages. CLI: dna hosting packages
type HostingPackagesRequest struct{ BaseRequest }

func (HostingPackagesRequest) Endpoint() Endpoint {
	return Endpoint{Method: http.MethodGet, Path: "/api/v1/hostings/packages", Safety: Read}
}

// HostingPlansRequest lists service plans and their limits.
// CLI: dna hosting plans
type HostingPlansRequest struct {
	BaseRequest
	// HostingID limits the list to the plans available to one hosting
	// (Hosting.HostingID); optional.
	HostingID string
}

func (HostingPlansRequest) Endpoint() Endpoint {
	return Endpoint{Method: http.MethodGet, Path: "/api/v1/hostings/plans", Safety: Read}
}

func (r HostingPlansRequest) Query() url.Values {
	return newParams().str("hostingId", r.HostingID).Values
}

// HostingCreateRequest orders a hosting for a domain. It charges your
// balance. CLI: dna hosting create
type HostingCreateRequest struct {
	BaseRequest
	DomainName string
	// ServicePlan comes from HostingService.Packages.
	ServicePlan string
}

func (HostingCreateRequest) Endpoint() Endpoint {
	return Endpoint{Method: http.MethodPost, Path: "/api/v1/hostings/create", Safety: Billable}
}

func (r HostingCreateRequest) Body() any {
	return object{"domainName": trim(r.DomainName), "servicePlan": trim(r.ServicePlan)}
}

func (r HostingCreateRequest) Validate() error {
	var v validator
	v.required("domainName", r.DomainName)
	v.required("servicePlan", r.ServicePlan)
	return v.err()
}

// HostingGetRequest reads a hosting by service ID or domain name (set
// exactly one). CLI: dna hosting get
type HostingGetRequest struct {
	BaseRequest
	ServiceID  string
	DomainName string
}

func (HostingGetRequest) Endpoint() Endpoint {
	return Endpoint{Method: http.MethodGet, Path: "/api/v1/hostings", Safety: Read}
}

func (r HostingGetRequest) Query() url.Values {
	return newParams().str("serviceId", r.ServiceID).str("domainName", r.DomainName).Values
}

func (r HostingGetRequest) Validate() error { return exactlyOne(r.ServiceID, r.DomainName) }

// HostingGetByTokenRequest reads a hosting by the token Create returned.
// CLI: dna hosting get-by-token
type HostingGetByTokenRequest struct {
	BaseRequest
	Token string
}

func (HostingGetByTokenRequest) Endpoint() Endpoint {
	return Endpoint{Method: http.MethodGet, Path: "/api/v1/hostings/by-token", Safety: Read}
}

func (r HostingGetByTokenRequest) Query() url.Values {
	return newParams().str("token", r.Token).Values
}

func (r HostingGetByTokenRequest) Validate() error {
	var v validator
	v.required("token", r.Token)
	return v.err()
}

// HostingStatusRequest reads the provisioning state of a hosting by service
// ID (preferred) or domain name (set exactly one). CLI: dna hosting status
type HostingStatusRequest struct {
	BaseRequest
	ServiceID  string
	DomainName string
}

func (HostingStatusRequest) Endpoint() Endpoint {
	return Endpoint{Method: http.MethodGet, Path: "/api/v1/hostings/check-status", Safety: Read}
}

func (r HostingStatusRequest) Query() url.Values {
	return newParams().str("ServiceId", r.ServiceID).str("DomainName", r.DomainName).Values
}

func (r HostingStatusRequest) Validate() error { return exactlyOne(r.ServiceID, r.DomainName) }

func exactlyOne(serviceID, domainName string) error {
	var v validator
	v.check((trim(serviceID) == "") != (trim(domainName) == ""), "serviceId", "set either the service ID or the domain name")
	return v.err()
}

// Packages lists the hosting packages and their service plans.
func (s *HostingService) Packages(ctx context.Context) ([]HostingPackage, error) {
	res, err := call[[]HostingPackage](ctx, s.client, HostingPackagesRequest{})
	if err != nil {
		return nil, err
	}
	return *res, nil
}

// Plans lists service plans and their limits; hostingID ("" for all)
// limits them to one hosting.
func (s *HostingService) Plans(ctx context.Context, hostingID string) ([]ServicePlan, error) {
	res, err := call[[]ServicePlan](ctx, s.client, HostingPlansRequest{HostingID: hostingID})
	if err != nil {
		return nil, err
	}
	return *res, nil
}

// Create orders a hosting and charges your balance. Provisioning continues
// after the call returns: pass the result's ServiceID to Wait.
func (s *HostingService) Create(ctx context.Context, domainName, servicePlan string) (*HostingCreateResult, error) {
	return call[HostingCreateResult](ctx, s.client, HostingCreateRequest{DomainName: domainName, ServicePlan: servicePlan})
}

// Get returns a hosting by service ID or domain name.
func (s *HostingService) Get(ctx context.Context, req HostingGetRequest) (*Hosting, error) {
	return call[Hosting](ctx, s.client, req)
}

// GetByToken returns a hosting by the token Create returned.
func (s *HostingService) GetByToken(ctx context.Context, token string) (*Hosting, error) {
	return call[Hosting](ctx, s.client, HostingGetByTokenRequest{Token: token})
}

// Status returns the provisioning state of a hosting.
func (s *HostingService) Status(ctx context.Context, req HostingStatusRequest) (*HostingStatus, error) {
	return call[HostingStatus](ctx, s.client, req)
}

// Wait polls Status until the hosting is active and returns its panel
// access. It polls every 5 seconds for 2 minutes, then every 30 seconds,
// until ctx ends: give ctx a deadline (provisioning rarely takes more than
// 30 minutes). A failed provisioning returns an *OperationError with the
// failure reason.
//
//	ctx, cancel := context.WithTimeout(ctx, 30*time.Minute)
//	defer cancel()
//	st, err := client.Hosting.Wait(ctx, created.ServiceID)
func (s *HostingService) Wait(ctx context.Context, serviceID string) (*HostingStatus, error) {
	req := HostingStatusRequest{ServiceID: serviceID}
	start := time.Now()
	for {
		st, err := s.Status(ctx, req)
		if err != nil {
			return nil, err
		}
		switch {
		case st.Active():
			return st, nil
		case st.Failed():
			return nil, operationError(req, nil, HostingFailed, firstNonEmpty(st.FailureReason, st.StatusMessage, "hosting provisioning failed"))
		}
		interval := hostingPollFast
		if time.Since(start) > 2*time.Minute {
			interval = hostingPollSlow
		}
		if err := sleep(ctx, interval); err != nil {
			return st, &NetworkError{Method: http.MethodGet, Path: req.Endpoint().Path, Err: err, safety: Read}
		}
	}
}
