// SSL: generate a CSR, buy or renew a certificate, configure it and wait
// until it is issued. Buying and renewing charge the balance, so the program
// stops after listing the packages and your certificates unless you pass
// -yes with -package or -renew.
//
// An existing certificate is addressed by its certificate id, never by its
// domain name: a domain can have several certificates.
//
//	go run ./examples/ssl -domain example.com
//	go run ./examples/ssl -domain example.com -package PositiveSSL -approver admin@example.com -yes
//	go run ./examples/ssl -renew 3fa85f64-5717-4562-b3fc-2c963f66afa6 -approver admin@example.com -yes
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
	pkg := flag.String("package", "", "SSL package code to buy (see the list)")
	renew := flag.String("renew", "", "id of the certificate to renew instead of buying one (see the list)")
	approver := flag.String("approver", "", "approver e-mail address")
	yes := flag.Bool("yes", false, "really buy or renew (charges the balance)")
	flag.Parse()

	client, err := dna.NewClientFromEnv()
	if err != nil {
		log.Fatal(err)
	}
	ctx := context.Background()

	if *renew != "" {
		// The new CSR must carry the domain of the certificate being renewed.
		old, err := client.SSL.Get(ctx, *renew)
		if err != nil {
			log.Fatal(err)
		}
		*domain = old.DomainName
		fmt.Printf("certificate to renew: %s (%s); its subscription ends on %s\n",
			old.DomainName, old.PackageCode, old.Subscription.ExpiresAt.Format(time.DateOnly))
	}
	if *domain == "" {
		log.Fatal("-domain is required")
	}

	if (*pkg == "" && *renew == "") || !*yes {
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

		fmt.Println("\nyour certificates for", *domain+":")
		for cert, err := range client.SSL.All(ctx, dna.SSLListRequest{DomainName: *domain}) {
			if err != nil {
				log.Fatal(err)
			}
			fmt.Printf("%s %-22s %-16s subscription ends %s\n",
				cert.CertificateID, cert.Status, cert.PackageCode, cert.SubscriptionExpiresAt.Format(time.DateOnly))
		}
		fmt.Println("\npass -package CODE -approver MAIL -yes to buy one,")
		fmt.Println("or -renew CERTIFICATE_ID -approver MAIL -yes to renew one (from 30 days before its subscription ends)")
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

	// Buying and renewing both answer with the id of a new certificate that
	// waits for configuration. A renewal is Renew + Configure; a second
	// Initialize would be an unrelated purchase.
	var order *dna.SSLOperationResult
	if *renew != "" {
		order, err = client.SSL.Renew(ctx, *renew)
	} else {
		order, err = client.SSL.Initialize(ctx, *domain, *pkg)
	}
	if err != nil {
		log.Fatal(err)
	}
	id := order.CertificateID
	fmt.Println("certificate id:", id, "(keep it: every later call takes it; configure within 15 days)")

	if _, err := client.SSL.Configure(ctx, dna.SSLConfigureRequest{
		CertificateID: id, CSR: csr.CSR, PrivateKey: csr.PrivateKey,
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
		st, err := client.SSL.Status(ctx, id)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println("status:", st.Status, st.StatusMessage)
		if st.IsIssued {
			break
		}
		time.Sleep(time.Minute)
	}
	cert, err := client.SSL.Content(ctx, id)
	if err != nil {
		log.Fatal(err)
	}
	if err := os.WriteFile(*domain+".crt", []byte(cert.Chain()), 0o644); err != nil {
		log.Fatal(err)
	}
	fmt.Println("wrote", *domain+".crt and", *domain+".key")
}
