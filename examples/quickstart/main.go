// Quickstart: check the credentials, show the balance and search a domain.
//
//	export DNA_RESELLER_ID=xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
//	export DNA_API_KEY=your-ote-api-key
//	export DNA_ENV=ote
//	go run ./examples/quickstart example.com
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	dna "github.com/domainreseller/go-dna/v2"
)

func main() {
	domain := "example.com"
	if len(os.Args) > 1 {
		domain = os.Args[1]
	}

	client, err := dna.NewClientFromEnv()
	if err != nil {
		log.Fatal(err)
	}
	ctx := context.Background()

	account, err := client.Account.Get(ctx)
	if err != nil {
		log.Fatal(err) // a wrong key or environment shows up here
	}
	fmt.Printf("%s (%s): %.2f USD, %.2f TRY\n", account.ResellerName, client.Environment(), account.USDBalance, account.TRYBalance)

	a, err := client.Domains.Search(ctx, domain)
	if err != nil {
		log.Fatal(err)
	}
	if a.Available() {
		fmt.Printf("%s is available: %.2f %s for %d year(s)\n", a.DomainName, a.Price, a.Currency, a.Period)
	} else {
		fmt.Printf("%s is not available (%s)\n", a.DomainName, a.Reason)
	}
}
