// Register: create a contact handle and register a domain with it.
// Registration charges the balance, so the program only previews the
// request unless you pass -yes. Use the test environment (DNA_ENV=ote).
//
//	go run ./examples/register -domain my-test-domain-123.com
//	go run ./examples/register -domain my-test-domain-123.com -yes
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"

	dna "github.com/domainreseller/go-dna/v2"
)

func main() {
	domain := flag.String("domain", "", "domain to register")
	years := flag.Int("years", 1, "registration period in years")
	yes := flag.Bool("yes", false, "really register (charges the balance)")
	flag.Parse()
	if *domain == "" {
		log.Fatal("-domain is required")
	}

	client, err := dna.NewClientFromEnv()
	if err != nil {
		log.Fatal(err)
	}
	ctx := context.Background()

	a, err := client.Domains.Search(ctx, *domain)
	if err != nil {
		log.Fatal(err)
	}
	if !a.Available() {
		log.Fatalf("%s is not available: %s", *domain, a.Reason)
	}
	fmt.Printf("%s costs %.2f %s\n", *domain, a.Price, a.Currency)

	owner := dna.Contact{
		FirstName: "Ada", LastName: "Lovelace", CompanyName: "Example Ltd", Email: "ada@example.com",
		Address: "1 Example Street", City: "Istanbul", State: "Istanbul", PostalCode: "34000", Country: "TR",
		PhoneCountryCode: "+90", Phone: "555 123 45 67", // tidied to digits for you
	}
	req := dna.DomainRegisterRequest{
		DomainName:  *domain,
		Years:       *years,
		NameServers: []string{"ns1.domainnameapi.com", "ns2.domainnameapi.com"},
		Handles:     dna.SameHandle("DNA-HANDLE"), // replaced below
	}

	if !*yes {
		// Preview the exact request without sending it.
		preview, err := client.NewHTTPRequest(ctx, req)
		if err != nil {
			log.Fatal(err)
		}
		body, _ := preview.GetBody()
		b, _ := io.ReadAll(body)
		fmt.Printf("would send %s %s\n%s\nrun again with -yes to register\n", preview.Method, preview.URL, b)
		return
	}

	handle, err := client.Contacts.Create(ctx, owner)
	if err != nil {
		log.Fatal(err)
	}
	req.Handles = dna.SameHandle(handle)

	res, err := client.Domains.Register(ctx, req)
	var refused *dna.OperationError
	switch {
	case dna.OutcomeUnknown(err):
		log.Fatalf("the registration may have gone through; check `client.Domains.Info` before retrying: %v", err)
	case dna.IsInsufficientBalance(err):
		log.Fatal("the balance is too low")
	case errors.As(err, &refused):
		log.Fatalf("the registry refused: %s", refused.Message)
	case err != nil:
		log.Fatal(err)
	}
	fmt.Printf("registered %s: status %s, expires %s\n", *domain, res.Status, res.ExpirationDate)
}
