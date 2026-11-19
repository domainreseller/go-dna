// Custom request: send a request type of your own through the client.
//
// Every operation of the package is a request type that embeds
// dna.BaseRequest. Yours can do the same: define Endpoint, override Query,
// Body or Validate when needed, and send it with client.Do. You get the
// client's credentials, retries, pacing, error handling and tracing.
//
//	go run ./examples/custom-request example.com
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"

	dna "github.com/domainreseller/go-dna/v2"
)

// ExpiryRequest reads only the expiry fields of a domain.
type ExpiryRequest struct {
	dna.BaseRequest
	DomainName string
}

func (ExpiryRequest) Endpoint() dna.Endpoint {
	return dna.Endpoint{Method: http.MethodGet, Path: "/api/v1/domains/info", Safety: dna.Read}
}

func (r ExpiryRequest) Query() url.Values {
	return url.Values{"DomainName": {r.DomainName}}
}

func (r ExpiryRequest) Validate() error {
	if r.DomainName == "" {
		return errors.New("DomainName is required")
	}
	return nil
}

// Expiry is the part of the response this program needs.
type Expiry struct {
	DomainName     string   `json:"domainName"`
	ExpirationDate dna.Time `json:"expirationDate"`
	RemainingDays  int      `json:"remainingDay"`
}

func main() {
	domain := "example.com"
	if len(os.Args) > 1 {
		domain = os.Args[1]
	}
	client, err := dna.NewClientFromEnv(dna.WithTrace(func(ev dna.TraceEvent) {
		if ev.Kind == dna.TraceResponse {
			fmt.Fprintf(os.Stderr, "%s %s -> %d in %s\n", ev.Method, ev.URL, ev.StatusCode, ev.Duration)
		}
	}))
	if err != nil {
		log.Fatal(err)
	}

	var out Expiry
	resp, err := client.Do(context.Background(), ExpiryRequest{DomainName: domain}, &out)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%s expires %s (%d days); correlation ID %s\n", out.DomainName, out.ExpirationDate, out.RemainingDays, resp.CorrelationID)
}
