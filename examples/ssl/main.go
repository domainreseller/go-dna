// SSL: generate a CSR, buy a certificate, configure it and wait until it is
// issued. Buying charges the balance, so the program stops after listing
// the packages unless you pass -package and -yes.
//
//	go run ./examples/ssl -domain example.com
//	go run ./examples/ssl -domain example.com -package PositiveSSL -approver admin@example.com -yes
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	dna "github.com/domainreseller/go-dna/v2"
)

func main() {
	domain := flag.String("domain", "", "domain of the certificate")
	pkg := flag.String("package", "", "SSL package code (see the list)")
	approver := flag.String("approver", "", "approver e-mail address")
	yes := flag.Bool("yes", false, "really buy (charges the balance)")
	flag.Parse()
	if *domain == "" {
		log.Fatal("-domain is required")
	}

	client, err := dna.NewClientFromEnv()
	if err != nil {
		log.Fatal(err)
	}
	ctx := context.Background()

	if *pkg == "" || !*yes {
		products, err := client.Products.SSLs(ctx, dna.SSLProductListRequest{})
		if err != nil {
			log.Fatal(err)
		}
		for _, p := range products.Items {
			price := ""
			if len(p.Prices) > 0 && len(p.Prices[0].Create) > 0 {
				price = p.Prices[0].Create[0].String()
			}
			fmt.Printf("%-24s %-12s wildcard=%-5v %s\n", p.ProductCode, p.ValidationType, p.IsWildcard, price)
		}
		fmt.Println("pass -package CODE -approver MAIL -yes to buy one")
		return
	}

	csr, err := client.SSL.GenerateCSR(ctx, dna.SSLGenerateCSRRequest{
		DomainName: *domain, Country: "TR", State: "Istanbul", Locality: "Istanbul",
		Organization: "Example Ltd", OrganizationUnit: "IT",
	})
	if err != nil {
		log.Fatal(err)
	}
	if err := os.WriteFile(*domain+".key", []byte(csr.PrivateKey), 0o600); err != nil {
		log.Fatal(err)
	}

	if _, err := client.SSL.Initialize(ctx, *domain, *pkg); err != nil {
		log.Fatal(err)
	}
	if _, err := client.SSL.Configure(ctx, dna.SSLConfigureRequest{
		DomainName: *domain, CSR: csr.CSR, PrivateKey: csr.PrivateKey,
		ApproverMail: *approver, AuthenticationType: dna.AuthenticationEmail, WebServerType: dna.WebServerApache,
		Contact: dna.SSLContact{
			FirstName: "Ada", LastName: "Lovelace", CompanyName: "Example Ltd", Email: *approver,
			Address: "1 Example Street", City: "Istanbul", PostalCode: "34000", Country: "TR",
			PhoneCountryCode: "90", Phone: "5551234567",
		},
	}); err != nil {
		log.Fatal(err)
	}

	for {
		st, err := client.SSL.Status(ctx, *domain)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println("status:", st.Status, st.StatusMessage)
		if st.IsIssued {
			break
		}
		time.Sleep(time.Minute)
	}
	cert, err := client.SSL.Content(ctx, *domain)
	if err != nil {
		log.Fatal(err)
	}
	if err := os.WriteFile(*domain+".crt", []byte(cert.Chain()), 0o644); err != nil {
		log.Fatal(err)
	}
	fmt.Println("wrote", *domain+".crt and", *domain+".key")
}
