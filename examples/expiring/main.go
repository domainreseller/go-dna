// Expiring: list every domain that expires within N days, across all pages.
//
//	go run ./examples/expiring -days 30
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"time"

	dna "github.com/domainreseller/go-dna/v2"
)

func main() {
	days := flag.Int("days", 30, "show domains expiring within this many days")
	flag.Parse()

	// Read-only: this program cannot change anything, even by mistake.
	client, err := dna.NewClientFromEnv(dna.WithReadOnly())
	if err != nil {
		log.Fatal(err)
	}

	n := 0
	for d, err := range client.Domains.All(context.Background(), dna.DomainListRequest{
		Statuses: []dna.DomainStatus{dna.DomainStatusActive},
	}) {
		if err != nil {
			log.Fatal(err)
		}
		if d.RemainingDays <= *days {
			n++
			fmt.Printf("%-40s %s  %3d days  renewal: %s\n", d.DomainName, d.ExpirationDate.Format(time.DateOnly), d.RemainingDays, d.RenewalMode)
		}
	}
	fmt.Printf("%d domain(s) expire within %d days\n", n, *days)
}
