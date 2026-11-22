package dna_test

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"time"

	dna "github.com/domainreseller/go-dna/v2"
)

func ExampleNewClient() {
	client, err := dna.NewClient(os.Getenv("DNA_RESELLER_ID"), os.Getenv("DNA_API_KEY"),
		dna.WithEnvironment(dna.OTE), // the test environment: its own key, no real charges
		dna.WithUserAgent("my-panel/1.0"),
	)
	if err != nil {
		log.Fatal(err)
	}
	account, err := client.Account.Get(context.Background())
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%s: %.2f USD\n", account.ResellerName, account.USDBalance)
}

func ExampleDomainService_Search() {
	client, _ := dna.NewClientFromEnv()
	a, err := client.Domains.Search(context.Background(), "example.com")
	if err != nil {
		log.Fatal(err)
	}
	if a.Available() {
		fmt.Printf("%s is available for %.2f %s\n", a.DomainName, a.Price, a.Currency)
	} else {
		fmt.Printf("%s is taken: %s\n", a.DomainName, a.Reason)
	}
}

func ExampleDomainService_Register() {
	client, _ := dna.NewClientFromEnv(dna.WithEnvironment(dna.OTE))
	ctx := context.Background()

	handle, err := client.Contacts.Create(ctx, dna.Contact{
		FirstName: "Ada", LastName: "Lovelace", Email: "ada@example.com",
		Address: "1 Example Street", City: "Istanbul", State: "Istanbul", PostalCode: "34000", Country: "TR",
		PhoneCountryCode: "90", Phone: "555 123 45 67",
	})
	if err != nil {
		log.Fatal(err)
	}
	res, err := client.Domains.Register(ctx, dna.DomainRegisterRequest{
		DomainName:  "example.com",
		Years:       1,
		NameServers: []string{"ns1.example.net", "ns2.example.net"},
		Handles:     dna.SameHandle(handle),
	})
	switch {
	case dna.OutcomeUnknown(err):
		log.Fatal("the request may have gone through; check client.Domains.Info before retrying: ", err)
	case err != nil:
		log.Fatal(err)
	}
	fmt.Println(res.Status, res.ExpirationDate)
}

func ExampleDomainService_All() {
	client, _ := dna.NewClientFromEnv()
	for d, err := range client.Domains.All(context.Background(), dna.DomainListRequest{}) {
		if err != nil {
			log.Fatal(err)
		}
		if d.RemainingDays < 30 {
			fmt.Println(d.DomainName, "expires on", d.ExpirationDate.Format(time.DateOnly))
		}
	}
}

func ExampleDNSService_Add() {
	client, _ := dna.NewClientFromEnv()
	err := client.DNS.Add(context.Background(), dna.DNSAddRequest{
		DomainName: "example.com",
		Name:       "www",
		Type:       dna.RecordTypeA,
		TTL:        600,
		Values:     []string{"192.0.2.10"},
	})
	if err != nil {
		log.Fatal(err)
	}
}

func ExampleSSLService_Renew() {
	client, _ := dna.NewClientFromEnv()
	ctx := context.Background()

	// A certificate is addressed by its ID, not by its domain name: a domain
	// can have several. Find the active certificate of the domain.
	var oldID string
	for cert, err := range client.SSL.All(ctx, dna.SSLListRequest{DomainName: "example.com"}) {
		if err != nil {
			log.Fatal(err)
		}
		if cert.Status == "Active" {
			oldID = cert.CertificateID
			break
		}
	}
	if oldID == "" {
		log.Fatal("example.com has no active certificate")
	}

	// Renew charges the balance and answers with a new certificate. The API
	// allows it from 30 days before the subscription ends.
	renewal, err := client.SSL.Renew(ctx, oldID)
	switch {
	case dna.OutcomeUnknown(err):
		log.Fatal("the renewal may have gone through; calling Renew again returns it without a second charge: ", err)
	case err != nil:
		log.Fatal(err)
	}

	// Configure the new certificate within 15 days. A second Initialize would
	// be an unrelated purchase, not a renewal.
	csr, err := client.SSL.GenerateCSR(ctx, dna.SSLGenerateCSRRequest{
		DomainName: "example.com", Country: "TR", State: "Istanbul", Locality: "Istanbul",
		Organization: "Example Ltd", OrganizationUnit: "IT",
	})
	if err != nil {
		log.Fatal(err)
	}
	_, err = client.SSL.Configure(ctx, dna.SSLConfigureRequest{
		CertificateID: renewal.CertificateID, // the new certificate, not oldID
		CSR:           csr.CSR,
		ApproverMail:  "admin@example.com", AuthenticationType: dna.AuthenticationEmail, WebServerType: dna.WebServerApache,
		Contact: dna.SSLContact{
			FirstName: "Ada", LastName: "Lovelace", CompanyName: "Example Ltd", Email: "ada@example.com",
			Address: "1 Example Street", City: "Istanbul", PostalCode: "34000", Country: "TR",
			PhoneCountryCode: "90", Phone: "5551234567",
		},
	})
	if err != nil {
		log.Fatal(err)
	}
	st, err := client.SSL.Status(ctx, renewal.CertificateID)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(st.Status, st.IsIssued)
}

func ExampleKindOf() {
	client, _ := dna.NewClientFromEnv()
	_, err := client.Domains.Info(context.Background(), "example.com")

	var apiErr *dna.APIError
	switch {
	case err == nil:
	case dna.IsNotFound(err):
		fmt.Println("not in this account (or not in this environment)")
	case dna.KindOf(err) == dna.KindInvalidInput:
		fmt.Println("fix the input:", err) // nothing was sent
	case errors.As(err, &apiErr):
		fmt.Println(apiErr.Code, apiErr.Message, apiErr.Hint(), apiErr.CorrelationID)
	default:
		log.Fatal(err)
	}
}

// NameServersRequest is a request type of your own: embed BaseRequest,
// define Endpoint, and override Query, Body or Validate as needed. Use the
// same pattern for an endpoint this package does not wrap yet.
type NameServersRequest struct {
	dna.BaseRequest
	DomainName string
}

func (NameServersRequest) Endpoint() dna.Endpoint {
	return dna.Endpoint{Method: http.MethodGet, Path: "/api/v1/domains/info", Safety: dna.Read}
}

func (r NameServersRequest) Query() url.Values {
	return url.Values{"DomainName": {r.DomainName}}
}

func (r NameServersRequest) Validate() error {
	if r.DomainName == "" {
		return errors.New("DomainName is required")
	}
	return nil
}

func ExampleClient_Do() {
	client, _ := dna.NewClientFromEnv()

	// Decode only the fields you need.
	var out struct {
		NameServers []string `json:"nameservers"`
	}
	resp, err := client.Do(context.Background(), NameServersRequest{DomainName: "example.com"}, &out)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(out.NameServers, resp.CorrelationID, resp.Quota)
}

func ExampleWithReadOnly() {
	client, _ := dna.NewClientFromEnv(dna.WithReadOnly())
	err := client.Domains.Lock(context.Background(), "example.com")
	fmt.Println(errors.Is(err, dna.ErrReadOnly))
}
