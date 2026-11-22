// Package dna is the Go client for the DomainNameAPI reseller REST API:
// domains, DNS, DNSSEC, contacts, SSL certificates, hosting, orders, the
// product catalogue and your balance.
//
// # Getting started
//
// Create a client with your reseller ID and API key (reseller panel → API
// settings). Start in the test environment (OTE), which has its own key and
// data:
//
//	client, err := dna.NewClient(resellerID, apiKey, dna.WithEnvironment(dna.OTE))
//	if err != nil {
//		log.Fatal(err)
//	}
//	a, err := client.Domains.Search(ctx, "example.com")
//	if err != nil {
//		log.Fatal(err)
//	}
//	fmt.Println(a.Available(), a.Price, a.Currency)
//
// NewClientFromEnv reads the same environment variables as the dna CLI
// (DNA_RESELLER_ID, DNA_API_KEY, DNA_ENV).
//
// # Services
//
// The client's services mirror the command groups of the dna CLI:
// client.Domains.Info is `dna domain info`, client.DNS.Add is `dna dns
// add`, and so on. Methods return typed results; enums such as DomainStatus
// have names (DomainStatusPendingDelete) and String methods; list
// endpoints have iterators (Domains.All, Account.AllTransactions,
// Products.AllTLDs, SSL.All) that fetch pages for you.
//
// # Requests
//
// Every operation is a request type (DomainInfoRequest, DNSAddRequest, ...)
// that implements Request. Each one embeds BaseRequest and defines its
// Endpoint, plus the Query, Body and Validate methods it needs. Service
// methods build these for you; Client.Do sends any Request, including your
// own types for endpoints this package does not wrap yet. Validate runs
// before anything is sent, so a missing field never costs a round trip.
//
// # Errors
//
// Errors are typed: *APIError (non-2xx answers, with the API's Code),
// *OperationError (2xx with "success": false), *InputError (rejected by
// Validate) and *NetworkError. KindOf classifies any of them; IsNotFound,
// IsAuth, IsRateLimited and IsInsufficientBalance answer the common
// questions. After a failed billable request, check OutcomeUnknown before
// retrying: it may have gone through.
//
// # Safety
//
// Every request has a Safety class (Read, Write, Destructive, Billable).
// WithReadOnly blocks everything but reads. The client retries rate-limited
// requests, retries network errors only for requests that are safe to
// repeat (never a registration or renewal), paces requests to stay under
// the API's rate limits, and is safe for concurrent use.
package dna
