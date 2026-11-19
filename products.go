package dna

import (
	"context"
	"iter"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// ProductService is the catalogue: TLDs, SSL certificates and other
// products with your prices. CLI: dna product ...
type ProductService service

// Product is a product of the catalogue.
type Product struct {
	ID              string      `json:"id"`
	ParentProductID string      `json:"parentProductId"`
	Name            string      `json:"productName"`
	ProductCode     string      `json:"productCode"`
	EntityID        string      `json:"entityId"`
	ProductType     ProductType `json:"productType"`
}

// ProductInfo is the price of a product for one order type and period.
type ProductInfo struct {
	ParentProductID string      `json:"parentProductId"`
	Name            string      `json:"productName"`
	ProductCode     string      `json:"productCode"`
	EntityID        string      `json:"entityId"`
	ProductType     ProductType `json:"productType"`
	SellingPrice    float64     `json:"sellingPrice"`
	SellingCurrency string      `json:"sellingCurrency"`
	Subproducts     []struct {
		ParentProductID string      `json:"parentProductId"`
		Name            string      `json:"productName"`
		ProductCode     string      `json:"productCode"`
		ProductType     ProductType `json:"productType"`
		EntityID        string      `json:"entityId"`
		BuyingPrice     float64     `json:"buyingPrice"`
		BuyingCurrency  string      `json:"buyingCurrency"`
		SellingPrice    float64     `json:"sellingPrice"`
		SellingCurrency string      `json:"sellingCurrency"`
	} `json:"subproducts"`
}

// TLD is a top-level domain you can sell: prices, periods, rules and the
// registry attributes it needs.
type TLD struct {
	// Name is the TLD without the leading dot, e.g. "com" or "com.tr".
	Name                  string `json:"name"`
	MinRegistrationPeriod int    `json:"minRegistrationPeriod"`
	MaxRegistrationPeriod int    `json:"maxRegistrationPeriod"`
	RegistrationPeriods   []int  `json:"registrationPeriods"`
	RenewalPeriods        []int  `json:"renewalPeriods"`
	// Grace and lifecycle periods, in days.
	AddGracePeriod           int `json:"addGracePeriod"`
	AutoRenewGracePeriod     int `json:"autoRenewGracePeriod"`
	FinalizationPeriod       int `json:"finalizationPeriod"`
	FailurePeriod            int `json:"failurePeriod"`
	PaymentPeriod            int `json:"paymentPeriod"`
	DeletionRestorablePeriod int `json:"deletionRestorablePeriod"`
	DeletionHoldPeriod       int `json:"deletionHoldPeriod"`
	TransferPeriod           int `json:"transferPeriod"`
	RegisterPeriod           int `json:"registerPeriod"`
	RenewalPeriod            int `json:"renewalPeriod"`
	// Capabilities.
	IsSupportTransfer          bool            `json:"isSupportTransfer"`
	IsSupportIDN               bool            `json:"isSupportIdn"`
	IsRequireDocument          bool            `json:"isRequireDocument"`
	IsSupportTrustee           bool            `json:"isSupportTrustee"`
	IsTrusteeRequired          bool            `json:"isTrusteeRequired"`
	IsSaleAllowed              bool            `json:"isSaleAllowed"`
	IsTransferLockPossible     bool            `json:"isTransferLockPossible"`
	IsHandleUpdateAvailable    bool            `json:"isHandleUpdateAvailable"`
	IsPrivacyAvailable         bool            `json:"isPrivacyAvailable"`
	IsDNSSECAvailable          bool            `json:"isDnssecAvailable"`
	IsIPv6Available            bool            `json:"isIpv6Available"`
	IsExplicitRenewalSupported bool            `json:"isExplicitRenewalSupported"`
	AvailableInTestEnvironment bool            `json:"availableInTestEnvironment"`
	IsRequiredAuthCode         bool            `json:"isRequiredAuthCode"`
	RegistryDNSCheck           bool            `json:"registryDnsCheck"`
	IsTransferCheckSupported   bool            `json:"isTransferCheckSupported"`
	IsGlueRecordAllowed        bool            `json:"isGlueRecordAllowed"`
	IsHostObjectAccepted       bool            `json:"isHostObjectAccepted"`
	Prices                     []TLDPriceGroup `json:"prices"`
	// Attributes are the registry attributes of the TLD; pass the required
	// ones as TLDAttributes when registering.
	Attributes []TLDAttribute `json:"attributes"`
}

// TLDPriceGroup holds the prices of one price group.
type TLDPriceGroup struct {
	PriceGroup string  `json:"priceGroup"`
	Register   []Price `json:"register"`
	Renew      []Price `json:"renew"`
	Transfer   []Price `json:"transfer"`
	Restore    *Price  `json:"restore"`
	Refund     *Price  `json:"refund"`
	Backorder  *Price  `json:"backorder"`
}

// TLDAttribute is a registry attribute of a TLD.
type TLDAttribute struct {
	Key         string `json:"key"`
	Description string `json:"description"`
	Required    bool   `json:"isRequired"`
	Type        string `json:"type"`
	Options     []struct {
		Value string `json:"value"`
	} `json:"options"`
}

// RegisterPrice returns the registration price for a period in years.
func (t *TLD) RegisterPrice(years int) (Price, bool) {
	return t.price(func(g TLDPriceGroup) []Price { return g.Register }, years)
}

// RenewPrice returns the renewal price for a period in years.
func (t *TLD) RenewPrice(years int) (Price, bool) {
	return t.price(func(g TLDPriceGroup) []Price { return g.Renew }, years)
}

// TransferPrice returns the transfer price for a period in years.
func (t *TLD) TransferPrice(years int) (Price, bool) {
	return t.price(func(g TLDPriceGroup) []Price { return g.Transfer }, years)
}

func (t *TLD) price(list func(TLDPriceGroup) []Price, years int) (Price, bool) {
	for _, g := range t.Prices {
		for _, p := range list(g) {
			if p.Period == years {
				return p, true
			}
		}
	}
	return Price{}, false
}

// RequiredAttributes returns the attributes a registration must include.
func (t *TLD) RequiredAttributes() []TLDAttribute {
	var out []TLDAttribute
	for _, a := range t.Attributes {
		if a.Required {
			out = append(out, a)
		}
	}
	return out
}

// SSLProduct is an SSL certificate product.
type SSLProduct struct {
	Name             string  `json:"name"`
	ProductCode      string  `json:"productCode"` // the package code SSLService.Initialize takes
	VendorName       string  `json:"vendorName"`
	ValidationType   string  `json:"validationType"`
	IsWildcard       bool    `json:"isWildcard"`
	IsMultiDomain    bool    `json:"isMultiDomain"`
	DefaultSANCount  int     `json:"defaultSanCount"`
	MaxSANCount      int     `json:"maxSanCount"`
	WarrantyAmount   float64 `json:"warrantyAmount"`
	IssueTimeMinutes int     `json:"issueTimeMinutes"`
	IsReissueFree    bool    `json:"isReissueFree"`
	RequiresCSR      bool    `json:"requiresCsr"`
	Prices           []struct {
		PriceGroup string  `json:"priceGroup"`
		Create     []Price `json:"create"`
	} `json:"prices"`
}

// ProductListRequest lists products. CLI: dna product list
type ProductListRequest struct {
	BaseRequest
	ListOptions
	FilterText  string
	ProductType ProductType // 0 for all
}

func (ProductListRequest) Endpoint() Endpoint {
	return Endpoint{Method: http.MethodGet, Path: "/api/v1/products", Safety: Read}
}

func (r ProductListRequest) Query() url.Values {
	return newParams().str("FilterText", r.FilterText).num("ProductType", int(r.ProductType)).page(r.ListOptions).Values
}

func (r ProductListRequest) Validate() error {
	var v validator
	r.ListOptions.validate(&v)
	return v.err()
}

// ProductInfoRequest reads the price of a product for an order type and
// period. CLI: dna product info
type ProductInfoRequest struct {
	BaseRequest
	// ProductName is a TLD without the dot ("com", "com.tr") or another
	// product name.
	ProductName string
	OrderType   OrderType
	// Years is the period.
	Years int
}

func (ProductInfoRequest) Endpoint() Endpoint {
	return Endpoint{Method: http.MethodGet, Path: "/api/v1/products/info", Safety: Read}
}

func (r ProductInfoRequest) Query() url.Values {
	p := newParams().str("ProductName", strings.TrimPrefix(trim(r.ProductName), "."))
	p.Set("OrderType", strconv.Itoa(int(r.OrderType)))
	p.Set("Period", strconv.Itoa(r.Years))
	return p.Values
}

func (r ProductInfoRequest) Validate() error {
	var v validator
	v.required("productName", r.ProductName)
	_, known := orderTypeNames[r.OrderType]
	v.check(known, "orderType", "is required (e.g. OrderRegister, OrderRenew)")
	v.check(r.Years >= 1 && r.Years <= 10, "years", "must be between 1 and 10")
	return v.err()
}

// TLDListRequest lists TLDs with prices and rules. CLI: dna product tlds
type TLDListRequest struct {
	BaseRequest
	ListOptions
	// FilterText keeps TLDs matching this text, e.g. "com.tr".
	FilterText string
	// Currency of the prices; empty uses the client's (WithCurrency) or the
	// account's.
	Currency string
}

func (TLDListRequest) Endpoint() Endpoint {
	return Endpoint{Method: http.MethodGet, Path: "/api/v1/products/tlds", Safety: Read}
}

func (r TLDListRequest) Query() url.Values {
	return newParams().str("FilterText", r.FilterText).str("Currency", r.Currency).page(r.ListOptions).Values
}

func (r TLDListRequest) Validate() error {
	var v validator
	r.ListOptions.validate(&v)
	return v.err()
}

// SSLProductListRequest lists SSL certificate products.
// CLI: dna product ssls
type SSLProductListRequest struct {
	BaseRequest
	ListOptions
	FilterText string
	VendorName string
	// Currency of the prices; empty uses the client's or the account's.
	Currency string
	// ValidationType filters by the API's numeric validation type; 0 for all.
	ValidationType int
	// IsWildcard and IsMultiDomain filter when set.
	IsWildcard, IsMultiDomain *bool
}

func (SSLProductListRequest) Endpoint() Endpoint {
	return Endpoint{Method: http.MethodGet, Path: "/api/v1/products/ssls", Safety: Read}
}

func (r SSLProductListRequest) Query() url.Values {
	return newParams().str("FilterText", r.FilterText).str("VendorName", r.VendorName).str("Currency", r.Currency).
		num("ValidationType", r.ValidationType).boolean("IsWildcard", r.IsWildcard).boolean("IsMultiDomain", r.IsMultiDomain).
		page(r.ListOptions).Values
}

func (r SSLProductListRequest) Validate() error {
	var v validator
	r.ListOptions.validate(&v)
	return v.err()
}

// List returns one page of products.
func (s *ProductService) List(ctx context.Context, req ProductListRequest) (*Page[Product], error) {
	return call[Page[Product]](ctx, s.client, req)
}

// Info returns the price of a product for an order type and period, e.g.
// the price of a 2-year .com renewal:
//
//	info, err := client.Products.Info(ctx, "com", dna.OrderRenew, 2)
func (s *ProductService) Info(ctx context.Context, productName string, orderType OrderType, years int) (*ProductInfo, error) {
	return call[ProductInfo](ctx, s.client, ProductInfoRequest{ProductName: productName, OrderType: orderType, Years: years})
}

// TLDs returns one page of the TLD catalogue. It is not rate limited; cache
// it rather than asking for every search.
func (s *ProductService) TLDs(ctx context.Context, req TLDListRequest) (*Page[TLD], error) {
	if req.Currency == "" {
		req.Currency = s.client.currency
	}
	return call[Page[TLD]](ctx, s.client, req)
}

// AllTLDs iterates over the whole TLD catalogue, fetching pages as needed.
func (s *ProductService) AllTLDs(ctx context.Context, req TLDListRequest) iter.Seq2[TLD, error] {
	return paginate(req.ListOptions, func(o ListOptions) (*Page[TLD], error) {
		req.ListOptions = o
		return s.TLDs(ctx, req)
	})
}

// TLD returns one TLD by name ("com.tr" or ".com.tr"), or nil (and no
// error) when the catalogue does not have it.
func (s *ProductService) TLD(ctx context.Context, name string) (*TLD, error) {
	name = strings.ToLower(strings.TrimPrefix(strings.TrimSpace(name), "."))
	tlds, err := collect(s.AllTLDs(ctx, TLDListRequest{FilterText: name}))
	if err != nil {
		return nil, err
	}
	for i := range tlds {
		if strings.EqualFold(tlds[i].Name, name) {
			return &tlds[i], nil
		}
	}
	return nil, nil
}

// SSLs returns one page of SSL certificate products.
func (s *ProductService) SSLs(ctx context.Context, req SSLProductListRequest) (*Page[SSLProduct], error) {
	if req.Currency == "" {
		req.Currency = s.client.currency
	}
	return call[Page[SSLProduct]](ctx, s.client, req)
}
