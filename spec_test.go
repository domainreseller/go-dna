package dna

// These tests check every request type against the API's OpenAPI document
// in testdata/openapi.json: the method and path exist, query parameters and
// body fields have the exact names the API uses, required fields are sent,
// and the JSON names of the response types exist in the response schemas.
// TestSpecCoverage fails when the API gains an operation without a request
// type here.
//
// To update the document, download it from
// https://api.domainresellerapi.com/swagger/v1/swagger.json.

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"
)

type specCase struct {
	req  Request
	resp any // the Go type the response decodes into; nil for no body
}

func specCases() []specCase {
	c := sampleContact()
	from := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	sub := ChargeSubtract
	return []specCase{
		// account
		{AccountGetRequest{}, Account{}},
		{AccountTransactionsRequest{
			ListOptions: ListOptions{Skip: 10, Limit: 20, Sort: "creationTime desc"}, TransactionTypes: []TransactionType{TransactionRegister, TransactionRenew},
			ProductTypes: []ProductType{ProductDomain}, ChargeType: &sub, Currencies: []string{"USD"}, From: from, To: from.AddDate(0, 1, 0),
			TraceID: "abc", Description: "renew", FilterText: "example", Amount: 9.99,
		}, Page[Transaction]{}},

		// contact
		{ContactGetRequest{Handle: "DNA-1"}, ContactDetails{}},
		{ContactCreateRequest{Contact: c}, ContactCreateResult{}},
		{ContactUpdateRequest{Handle: "DNA-1", Contact: c}, ContactUpdateResult{}},
		{ContactVerificationRequest{Handle: "DNA-1"}, struct {
			Results []ContactVerification `json:"results"`
		}{}},
		{ContactResendVerificationRequest{Handle: "DNA-1"}, nil},

		// domain
		{DomainSearchRequest{DomainName: "example.com", Currency: "USD"}, DomainSearchResult{}},
		{DomainBulkSearchRequest{DomainNames: []string{"example.com", "example.net"}, Currency: "USD"}, DomainBulkSearchResult{}},
		{DomainListRequest{ListOptions: ListOptions{Skip: 5, Limit: 50, Sort: "expirationDate"}, DomainName: "example", Statuses: []DomainStatus{DomainStatusActive, DomainStatusPendingDelete}}, Page[Domain]{}},
		{DomainInfoRequest{DomainName: "example.com"}, DomainInfo{}},
		{DomainRegisterRequest{DomainName: "example.com", Years: 2, NameServers: []string{"ns1.example.net", "ns2.example.net"}, Handles: SameHandle("DNA-1"), TLDAttributes: Attributes{"x": "y"}}, DomainOperationResult{}},
		{DomainRegisterWithContactsRequest{DomainName: "example.com", Years: 1, NameServers: []string{"ns1.example.net"}, Contacts: SameContact(c), TLDAttributes: Attributes{"x": "y"}, UseTrusteeContact: true}, DomainOperationResult{}},
		{DomainRenewRequest{DomainName: "example.com", Years: 1}, DomainOperationResult{}},
		{DomainRenewCheckRequest{DomainName: "example.com", Years: 1}, RenewCheck{}},
		{DomainRestoreRequest{DomainName: "example.com"}, nil},
		{DomainTransferRequest{DomainName: "example.com", AuthCode: "Xy7!kP2q", Years: 1}, DomainOperationResult{}},
		{DomainTransferCheckRequest{DomainName: "example.com", AuthCode: "Xy7!kP2q"}, TransferCheck{}},
		{DomainTransferStatusRequest{DomainName: "example.com"}, TransferStatus{}},
		{DomainLockRequest{DomainName: "example.com"}, nil},
		{DomainUnlockRequest{DomainName: "example.com"}, nil},
		{DomainPrivacyRequest{DomainName: "example.com", Enabled: true}, nil},
		{DomainUpdateAuthCodeRequest{DomainName: "example.com"}, AuthCode{}},
		{DomainNameServersSetRequest{DomainName: "example.com", NameServers: []string{"ns1.example.net"}, DNSSECMode: DNSSECModeKeep}, nil},
		{DomainHostAddRequest{DomainName: "example.com", HostName: "ns1.example.com", IPAddresses: []string{"192.0.2.1", "2001:db8::1"}}, nil},
		{DomainHostUpdateRequest{DomainName: "example.com", HostName: "ns1.example.com", NewHostName: "ns3.example.com", IPAddresses: []string{"192.0.2.2"}}, nil},
		{DomainHostDeleteRequest{DomainName: "example.com", HostName: "ns1.example.com"}, nil},
		{DomainContactsUpdateRequest{DomainName: "example.com", Contacts: SameContact(c)}, nil},
		{DomainContactsSetHandlesRequest{DomainName: "example.com", Handles: SameHandle("DNA-1")}, nil},
		{DomainDocumentsListRequest{ListOptions: ListOptions{Limit: 10}, DomainName: "example.com.tr", Types: []DocumentType{DocumentTypeIdentityCard}}, Page[Document]{}},
		{DomainDocumentsUploadRequest{DomainName: "example.com.tr", Documents: []DocumentUpload{{Type: DocumentTypeIdentityCard, FileName: "id.pdf", Content: []byte("%PDF-1.4")}}}, DocumentUploadResult{}},
		{DomainForwardGetRequest{DomainName: "example.com"}, Forwarding{}},
		{DomainForwardSetRequest{DomainName: "example.com", URL: "https://www.example.org", Type: ForwardTypeFrame}, nil},
		{DomainForwardDeleteRequest{DomainName: "example.com"}, nil},

		// dns
		{DNSListRequest{DomainName: "example.com"}, []RecordSet{}},
		{DNSAddRequest{DomainName: "example.com", Name: "www", Type: "a", TTL: 600, Values: []string{"192.0.2.1"}}, nil},
		{DNSUpdateRequest{DomainName: "example.com", RecordName: "www", Name: "web", Type: RecordTypeA, TTL: 600, Values: []string{"192.0.2.2"}}, nil},
		{DNSDeleteRequest{DomainName: "example.com", Name: "www", Type: RecordTypeA, Value: "192.0.2.1"}, nil},
		{DNSApplyRequest{DomainName: "example.com"}, ZoneApplyResult{}},

		// dnssec
		{DNSSECGetRequest{DomainName: "example.com"}, DNSSECInfo{}},
		{DNSSECAddRequest{DomainName: "example.com", Records: []DS{{KeyTag: 2371, Algorithm: AlgorithmECDSAP256SHA256, DigestType: DigestSHA256, Digest: "1F98"}}}, struct {
			Items []DNSSECOperationResult `json:"items"`
		}{}},
		{DNSSECRemoveRequest{DomainName: "example.com", Record: DS{KeyTag: 2371, Algorithm: 13, DigestType: 2, Digest: "1F98"}}, DNSSECOperationResult{}},
		{DNSSECDisableRequest{DomainName: "example.com"}, DNSSECOperationResult{}},

		// ssl
		{SSLGenerateCSRRequest{DomainName: "example.com", Country: "TR", State: "Istanbul", Locality: "Istanbul", Organization: "Example Ltd", OrganizationUnit: "IT", AlternativeNames: []string{"www.example.com"}}, CSR{}},
		{SSLCheckCSRRequest{CSR: "-----BEGIN CERTIFICATE REQUEST-----"}, CSRInfo{}},
		{SSLInitializeRequest{DomainName: "example.com", PackageCode: "PositiveSSL"}, SSLOperationResult{}},
		{SSLConfigureRequest{DomainName: "example.com", CSR: "csr", PrivateKey: "key", ApproverMail: "admin@example.com", AuthenticationType: AuthenticationEmail, WebServerType: WebServerApache, SANList: []string{"a.example.com"}, WildcardList: []string{"*.example.com"}, Contact: sampleSSLContact()}, SSLOperationResult{}},
		{SSLStatusRequest{DomainName: "example.com"}, CertificateStatus{}},
		{SSLGetRequest{DomainName: "example.com"}, Certificate{}},
		{SSLContentRequest{DomainName: "example.com"}, CertificateContent{}},
		{SSLReissueRequest{DomainName: "example.com", ApproverMail: "admin@example.com", CSR: "csr", PrivateKey: "key"}, SSLOperationResult{}},
		{SSLApproversRequest{DomainName: "example.com", PackageCode: "PositiveSSL"}, struct {
			Mails []string `json:"mails"`
		}{}},
		{SSLRevokeRequest{DomainName: "example.com"}, SSLOperationResult{}},

		// hosting
		{HostingPackagesRequest{}, []HostingPackage{}},
		{HostingPlansRequest{HostingID: "6f1c0000-0000-4000-8000-000000000001"}, []ServicePlan{}},
		{HostingCreateRequest{DomainName: "example.com", ServicePlan: "Linux-Starter"}, HostingCreateResult{}},
		{HostingGetRequest{ServiceID: "HST-1"}, Hosting{}},
		{HostingGetByTokenRequest{Token: "tok"}, Hosting{}},
		{HostingStatusRequest{DomainName: "example.com"}, HostingStatus{}},

		// order
		{OrderGetRequest{OrderCode: "ORD-1"}, Order{}},

		// product
		{ProductListRequest{ListOptions: ListOptions{Limit: 10}, FilterText: "com", ProductType: ProductDomain}, Page[Product]{}},
		{ProductInfoRequest{ProductName: "com", OrderType: OrderRenew, Years: 2}, ProductInfo{}},
		{TLDListRequest{ListOptions: ListOptions{Limit: 10}, FilterText: "com.tr", Currency: "USD"}, Page[TLD]{}},
		{SSLProductListRequest{ListOptions: ListOptions{Limit: 10}, FilterText: "ssl", VendorName: "Sectigo", Currency: "USD", ValidationType: 1, IsWildcard: Bool(true), IsMultiDomain: Bool(false)}, Page[SSLProduct]{}},
	}
}

// responseFieldsNotInSpec lists Go fields that are deliberately not in a
// response schema (the type is shared with a request).
var responseFieldsNotInSpec = map[string]bool{
	"Certificate.contacts[].title": true, // SSLContact is also the configure input, which has a title
}

// ---------------------------------------------------------------------------
// A small reader for the OpenAPI document.

type openAPI struct {
	Paths      map[string]map[string]specOp `json:"paths"`
	Components struct {
		Schemas map[string]map[string]any `json:"schemas"`
	} `json:"components"`
}

type specOp struct {
	OperationID string `json:"operationId"`
	Parameters  []struct {
		Name     string `json:"name"`
		In       string `json:"in"`
		Required bool   `json:"required"`
	} `json:"parameters"`
	RequestBody *struct {
		Content map[string]struct {
			Schema map[string]any `json:"schema"`
		} `json:"content"`
	} `json:"requestBody"`
	Responses map[string]struct {
		Content map[string]struct {
			Schema map[string]any `json:"schema"`
		} `json:"content"`
	} `json:"responses"`
}

var (
	specOnce sync.Once
	spec     openAPI
	specErr  error
)

func loadSpec(t *testing.T) *openAPI {
	t.Helper()
	specOnce.Do(func() {
		b, err := os.ReadFile("testdata/openapi.json")
		if err != nil {
			specErr = err
			return
		}
		specErr = json.Unmarshal(b, &spec)
	})
	if specErr != nil {
		t.Fatal(specErr)
	}
	return &spec
}

func (d *openAPI) resolve(s map[string]any) map[string]any {
	for s != nil {
		if ref, ok := s["$ref"].(string); ok {
			s = d.Components.Schemas[strings.TrimPrefix(ref, "#/components/schemas/")]
			continue
		}
		if all, ok := s["allOf"].([]any); ok && len(all) == 1 {
			s, _ = all[0].(map[string]any)
			continue
		}
		return s
	}
	return nil
}

// find returns the spec path template and operation matching method and a
// concrete path.
func (d *openAPI) find(method, path string) (string, *specOp) {
	for tmpl, ops := range d.Paths {
		re := "^" + regexp.MustCompile(`\\\{[^}]+\\\}`).ReplaceAllString(regexp.QuoteMeta(tmpl), `[^/]+`) + "$"
		if !regexp.MustCompile(re).MatchString(path) {
			continue
		}
		if op, ok := ops[strings.ToLower(method)]; ok {
			return tmpl, &op
		}
	}
	return "", nil
}

func firstSchema[T any](content map[string]T, get func(T) map[string]any) map[string]any {
	for _, ct := range []string{"application/json", "text/json", "text/plain"} {
		if c, ok := content[ct]; ok {
			return get(c)
		}
	}
	for _, c := range content {
		return get(c)
	}
	return nil
}

// ---------------------------------------------------------------------------

func TestSpecCoverage(t *testing.T) {
	d := loadSpec(t)
	covered := map[string]bool{}
	for _, c := range specCases() {
		ep := c.req.Endpoint()
		tmpl, op := d.find(ep.Method, ep.Path)
		if op == nil {
			t.Errorf("%s: %s %s is not in the OpenAPI document", requestName(c.req), ep.Method, ep.Path)
			continue
		}
		key := ep.Method + " " + tmpl
		if covered[key] {
			t.Errorf("%s: %s is covered twice", requestName(c.req), key)
		}
		covered[key] = true
	}
	var missing []string
	for tmpl, ops := range d.Paths {
		for method := range ops {
			if key := strings.ToUpper(method) + " " + tmpl; !covered[key] {
				missing = append(missing, key)
			}
		}
	}
	sort.Strings(missing)
	for _, m := range missing {
		t.Errorf("the API has %s but this package has no request type for it", m)
	}
}

func TestSpecRequests(t *testing.T) {
	d := loadSpec(t)
	c := &Client{baseURL: mustURL("https://api.example.test"), userAgent: UserAgent, apiKey: "k", resellerID: "r"}
	for _, sc := range specCases() {
		name := requestName(sc.req)
		t.Run(name, func(t *testing.T) {
			ep := sc.req.Endpoint()
			_, op := d.find(ep.Method, ep.Path)
			if op == nil {
				t.Skip("not in the spec (reported by TestSpecCoverage)")
			}
			if err := sc.req.Validate(); err != nil {
				t.Fatalf("Validate rejects the complete sample: %v", err)
			}
			if ep.Method == http.MethodGet && ep.Safety != Read {
				t.Errorf("GET request has safety %s", ep.Safety)
			}
			hreq, err := c.NewHTTPRequest(context.Background(), sc.req)
			if err != nil {
				t.Fatal(err)
			}

			// Query parameters: names must match the spec exactly.
			params := map[string]bool{}
			for _, p := range op.Parameters {
				if p.In != "query" {
					continue
				}
				params[p.Name] = true
				if p.Required && !hreq.URL.Query().Has(p.Name) {
					t.Errorf("required query parameter %q is not sent", p.Name)
				}
			}
			for k := range hreq.URL.Query() {
				if !params[k] {
					t.Errorf("query parameter %q is not in the spec (want one of %v)", k, sortedKeys(params))
				}
			}

			// Body.
			body := sc.req.Body()
			switch {
			case op.RequestBody == nil && body != nil:
				t.Errorf("sends a body, but the operation takes none")
			case op.RequestBody != nil && body == nil:
				t.Errorf("sends no body, but the operation takes one")
			case body != nil:
				schema := firstSchema(op.RequestBody.Content, func(c struct {
					Schema map[string]any `json:"schema"`
				}) map[string]any {
					return c.Schema
				})
				b, _ := json.Marshal(body)
				var v any
				if err := json.Unmarshal(b, &v); err != nil {
					t.Fatal(err)
				}
				for _, e := range checkJSON(d, v, schema, "body") {
					t.Error(e)
				}
			}

			// Response type.
			if sc.resp == nil {
				return
			}
			var schema map[string]any
			for _, code := range []string{"200", "201", "202"} {
				if r, ok := op.Responses[code]; ok {
					schema = firstSchema(r.Content, func(c struct {
						Schema map[string]any `json:"schema"`
					}) map[string]any {
						return c.Schema
					})
					break
				}
			}
			if schema == nil {
				t.Fatalf("the spec has no response body, but the request decodes into %T", sc.resp)
			}
			root := reflect.TypeOf(sc.resp).Name()
			if root == "" {
				root = "response"
			}
			for _, e := range checkGoType(d, reflect.TypeOf(sc.resp), schema, root) {
				t.Error(e)
			}
		})
	}
}

// checkJSON checks that every key of v exists in schema and every required
// key is present.
func checkJSON(d *openAPI, v any, schema map[string]any, path string) []string {
	schema = d.resolve(schema)
	if schema == nil {
		return nil
	}
	var errs []string
	switch x := v.(type) {
	case map[string]any:
		props, _ := schema["properties"].(map[string]any)
		if props == nil {
			return nil // free-form map such as tldAttributes
		}
		for k, sub := range x {
			ps, ok := props[k].(map[string]any)
			if !ok {
				errs = append(errs, fmt.Sprintf("%s.%s is not in the spec (want one of %v)", path, k, sortedKeys(props)))
				continue
			}
			errs = append(errs, checkJSON(d, sub, ps, path+"."+k)...)
		}
		req, _ := schema["required"].([]any)
		for _, r := range req {
			if _, ok := x[r.(string)]; !ok {
				errs = append(errs, fmt.Sprintf("%s.%s is required by the spec but not sent", path, r))
			}
		}
	case []any:
		items, _ := schema["items"].(map[string]any)
		for i, it := range x {
			errs = append(errs, checkJSON(d, it, items, fmt.Sprintf("%s[%d]", path, i))...)
		}
	}
	return errs
}

var timeType = reflect.TypeOf(Time{})

// checkGoType checks that the JSON names of a response type exist in its
// schema.
func checkGoType(d *openAPI, t reflect.Type, schema map[string]any, path string) []string {
	schema = d.resolve(schema)
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if schema == nil || t == timeType {
		return nil
	}
	var errs []string
	switch t.Kind() {
	case reflect.Slice:
		items, _ := schema["items"].(map[string]any)
		if items == nil {
			return []string{path + " is a slice, but the spec has no array"}
		}
		return checkGoType(d, t.Elem(), items, path+"[]")
	case reflect.Struct:
		props, _ := schema["properties"].(map[string]any)
		for i := 0; i < t.NumField(); i++ {
			f := t.Field(i)
			name := strings.Split(f.Tag.Get("json"), ",")[0]
			if !f.IsExported() || name == "-" || name == "" {
				continue
			}
			fp := path + "." + name
			ps, ok := props[name].(map[string]any)
			if !ok {
				if !responseFieldsNotInSpec[fp] {
					errs = append(errs, fmt.Sprintf("%s (%s) is not in the response schema (want one of %v)", fp, f.Name, sortedKeys(props)))
				}
				continue
			}
			errs = append(errs, checkGoType(d, f.Type, ps, fp)...)
		}
	}
	return errs
}

func sortedKeys[V any](m map[string]V) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// TestRequestsEmbedBaseRequest keeps the request types uniform.
func TestRequestsEmbedBaseRequest(t *testing.T) {
	base := reflect.TypeOf(BaseRequest{})
	for _, sc := range specCases() {
		rt := reflect.TypeOf(sc.req)
		f, ok := rt.FieldByName("BaseRequest")
		if !ok || !f.Anonymous || f.Type != base {
			t.Errorf("%s does not embed BaseRequest", rt.Name())
		}
		if !strings.HasSuffix(rt.Name(), "Request") {
			t.Errorf("%s: request type names end in Request", rt.Name())
		}
	}
}
