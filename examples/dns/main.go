// DNS: list, add, replace and delete zone records.
//
// The test environment answers every zone call with HTTP 500, so run this
// in production against a domain you do not mind changing.
//
//	go run ./examples/dns -domain example.com
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"strings"

	dna "github.com/domainreseller/go-dna/v2"
)

func main() {
	domain := flag.String("domain", "", "domain whose zone to change")
	flag.Parse()
	if *domain == "" {
		log.Fatal("-domain is required")
	}

	client, err := dna.NewClientFromEnv()
	if err != nil {
		log.Fatal(err)
	}
	ctx := context.Background()

	// Add a value to the "test" A record set (the set is created if needed).
	err = client.DNS.Add(ctx, dna.DNSAddRequest{
		DomainName: *domain, Name: "test", Type: dna.RecordTypeA, TTL: 600, Values: []string{"192.0.2.10"},
	})
	if err != nil {
		log.Fatal(err)
	}

	// Replace every value of the set.
	err = client.DNS.Update(ctx, dna.DNSUpdateRequest{
		DomainName: *domain, RecordName: "test", Type: dna.RecordTypeA, TTL: 300, Values: []string{"192.0.2.20", "192.0.2.21"},
	})
	if err != nil {
		log.Fatal(err)
	}

	sets, err := client.DNS.List(ctx, *domain)
	if err != nil {
		log.Fatal(err)
	}
	for _, s := range sets {
		fmt.Printf("%-30s %-6s %6d  %s\n", s.Name, s.Type, s.TTL, strings.Join(s.Values(), ", "))
	}

	// Remove the values again, one by one.
	for _, v := range []string{"192.0.2.20", "192.0.2.21"} {
		if err := client.DNS.Delete(ctx, dna.DNSDeleteRequest{DomainName: *domain, Name: "test", Type: dna.RecordTypeA, Value: v}); err != nil {
			log.Fatal(err)
		}
	}

	// Records resolve publicly only on the platform nameservers. Apply
	// switches the domain to them (and away from any other DNS provider):
	//
	//	res, err := client.DNS.Apply(ctx, *domain)
}
