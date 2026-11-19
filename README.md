<div align="center">
  <a href="README-TR.md">TR <img style="padding-top: 8px" src="https://raw.githubusercontent.com/yammadev/flag-icons/master/png/TR.png" alt="TR" height="20" /></a>
  <a href="README.md"> | EN <img style="padding-top: 8px" src="https://raw.githubusercontent.com/yammadev/flag-icons/master/png/US.png" alt="EN" height="20" /></a>
  <a href="README-DE.md"> | DE <img style="padding-top: 8px" src="https://raw.githubusercontent.com/yammadev/flag-icons/master/png/DE.png" alt="DE" height="20" /></a>
  <a href="README-SA.md"> | SA <img style="padding-top: 8px" src="https://raw.githubusercontent.com/yammadev/flag-icons/master/png/SA.png" alt="AR" height="20" /></a>
  <a href="README-NL.md"> | NL <img style="padding-top: 8px" src="https://raw.githubusercontent.com/yammadev/flag-icons/master/png/NL.png" alt="NL" height="20" /></a>
  <a href="README-AZ.md"> | AZ <img style="padding-top: 8px" src="https://raw.githubusercontent.com/yammadev/flag-icons/master/png/AZ.png" alt="AZ" height="20" /></a>
  <a href="README-CN.md"> | CN <img style="padding-top: 8px" src="https://raw.githubusercontent.com/yammadev/flag-icons/master/png/CN.png" alt="CN" height="20" /></a>
  <a href="README-FR.md"> | FR <img style="padding-top: 8px" src="https://raw.githubusercontent.com/yammadev/flag-icons/master/png/FR.png" alt="FR" height="20" /></a>
  <a href="README-IT.md"> | IT <img style="padding-top: 8px" src="https://raw.githubusercontent.com/yammadev/flag-icons/master/png/IT.png" alt="IT" height="20" /></a>
</div>

# go-dna: the Go SDK for the Domain Name API

[![Go Reference](https://pkg.go.dev/badge/github.com/domainreseller/go-dna/v2.svg)](https://pkg.go.dev/github.com/domainreseller/go-dna/v2)
[![CI](https://github.com/domainreseller/go-dna/actions/workflows/ci.yml/badge.svg)](https://github.com/domainreseller/go-dna/actions/workflows/ci.yml)
[![License](https://img.shields.io/badge/license-Apache--2.0-blue.svg)](LICENSE)

Typed Go client for the [Domain Name API](https://www.domainnameapi.com) reseller REST API: domain search,
registration, renewal and transfer, DNS, DNSSEC, contacts, SSL certificates, hosting, orders, the product
catalogue and your balance.

- **All 64 API operations**, grouped the same way as the [`dna` CLI](https://github.com/domainreseller/cli):
  `dna domain info` is `client.Domains.Info`, `dna dns add` is `client.DNS.Add`.
- **Typed results and enums**: `d.Status == dna.DomainStatusPendingDelete`, `d.ExpirationDate.Before(...)`,
  no `map[string]any`.
- **Every operation is a request type** that embeds `dna.BaseRequest` and validates itself before anything is
  sent. Write your own the same way.
- **Safe by default**: billable requests are never retried blindly, `WithReadOnly` blocks changes, the client
  paces itself under the rate limits and is safe for concurrent use.
- **Standard library only.** Go 1.23 or newer.

```go
client, _ := dna.NewClient(resellerID, apiKey, dna.WithEnvironment(dna.OTE))

a, err := client.Domains.Search(ctx, "example.com")
if err == nil && a.Available() {
	fmt.Printf("%s is available for %.2f %s\n", a.DomainName, a.Price, a.Currency)
}
```

## Contents

- [Install](#install)
- [Credentials and environments](#credentials-and-environments)
- [Services](#services)
- [Usage](#usage)
- [Requests and BaseRequest](#requests-and-baserequest)
- [Errors](#errors)
- [Safety and reliability](#safety-and-reliability)
- [Client options](#client-options)
- [Testing your code](#testing-your-code)
- [Migrating from v1](#migrating-from-v1)
- [Development](#development)

## Install

```bash
go get github.com/domainreseller/go-dna/v2
```

```go
import dna "github.com/domainreseller/go-dna/v2"
```

Runnable programs are in [examples](examples): `quickstart`, `register`, `expiring`, `dns`, `ssl` and
`custom-request`.

```bash
go run ./examples/quickstart example.com
```

## Credentials and environments

You need two API credentials from the reseller panel (**API settings**). Panel login details do not work.

| Credential  | Looks like                             |
|-------------|----------------------------------------|
| Reseller ID | `xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx` |
| API key     | a long secret string                   |

There are two environments. **Start with OTE**, the test environment: it has its own data and its own API key,
and nothing you do there costs money or touches live domains.

```go
// Test environment (OTE)
client, err := dna.NewClient(resellerID, oteAPIKey, dna.WithEnvironment(dna.OTE))

// Production (the default): real registrations, real charges
client, err := dna.NewClient(resellerID, apiKey)
```

`NewClientFromEnv` reads the same variables as the `dna` CLI:

| Variable          | Meaning                                                       |
|-------------------|---------------------------------------------------------------|
| `DNA_RESELLER_ID` | reseller ID                                                   |
| `DNA_API_KEY`     | API key of the selected environment                           |
| `DNA_ENV`         | `production` (default) or `ote`                               |
| `DNA_TEST_MODE`   | `true` selects OTE (as in the other DomainNameAPI SDKs)       |
| `DNA_BASE_URL`    | a custom base URL, e.g. a proxy                               |

```go
client, err := dna.NewClientFromEnv(dna.WithUserAgent("my-panel/1.4"))
```

A wrong key or environment fails on the first request with an `*APIError` of kind `KindAuth` and a hint;
`client.Account.Get(ctx)` is the cheapest way to check.

## Services

| Service           | CLI group     | Operations                                                                                                                                                        |
|-------------------|---------------|-------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `client.Account`  | `dna account` | `Get`, `Transactions`, `AllTransactions`                                                                                                                          |
| `client.Contacts` | `dna contact` | `Get`, `Create`, `Update`, `Verification`, `ResendVerification`                                                                                                   |
| `client.Domains`  | `dna domain`  | `Search`, `BulkSearch`, `List`, `All`, `Info`, `Register`, `RegisterWithContacts`, `Renew`, `RenewCheck`, `Restore`, `Transfer`, `TransferCheck`, `TransferStatus`, `Lock`, `Unlock`, `SetPrivacy`, `UpdateAuthCode`, `SetNameServers`, `AddHost`, `UpdateHost`, `DeleteHost`, `UpdateContacts`, `SetContactHandles`, `Documents`, `UploadDocuments`, `Forwarding`, `SetForwarding`, `DeleteForwarding` |
| `client.DNS`      | `dna dns`     | `List`, `Add`, `Update`, `Delete`, `Apply`                                                                                                                        |
| `client.DNSSEC`   | `dna dnssec`  | `Get`, `Add`, `Remove`, `Disable`                                                                                                                                 |
| `client.SSL`      | `dna ssl`     | `GenerateCSR`, `CheckCSR`, `Initialize`, `Configure`, `Status`, `Get`, `Content`, `Reissue`, `Approvers`, `Revoke`                                                |
| `client.Hosting`  | `dna hosting` | `Packages`, `Plans`, `Create`, `Wait`, `Get`, `GetByToken`, `Status`                                                                                              |
| `client.Orders`   | `dna order`   | `Get`                                                                                                                                                             |
| `client.Products` | `dna product` | `List`, `Info`, `TLDs`, `AllTLDs`, `TLD`, `SSLs`                                                                                                                  |

Methods take the required inputs as arguments (`Info(ctx, "example.com")`) or, when an operation has optional
inputs, a request struct (`Register(ctx, dna.DomainRegisterRequest{...})`). The full reference with every field
is on [pkg.go.dev](https://pkg.go.dev/github.com/domainreseller/go-dna/v2).

## Usage

All snippets assume `ctx := context.Background()` and a `client`.

### Search

```go
a, err := client.Domains.Search(ctx, "example.com")
// a.Available(), a.Price, a.Currency, a.IsPremium, a.IsDocumentRequired, a.Reason

results, err := client.Domains.BulkSearch(ctx, "example.com", "example.net", "example.org")
for _, r := range results {
	fmt.Println(r.DomainName, r.Status, r.Price)
}
```

`BulkSearch` splits lists longer than 30 domains into several requests. Prices use the account's currency, or
the one set with `dna.WithCurrency("TRY")`.

### Register

With contact handles (create one with `Contacts.Create`):

```go
handle, err := client.Contacts.Create(ctx, dna.Contact{
	FirstName: "Ada", LastName: "Lovelace", Email: "ada@example.com",
	Address: "1 Example Street", City: "Istanbul", State: "Istanbul", PostalCode: "34000",
	Country: "TR", PhoneCountryCode: "90", Phone: "5551234567",
})

res, err := client.Domains.Register(ctx, dna.DomainRegisterRequest{
	DomainName:  "example.com",
	Years:       1,
	NameServers: []string{"ns1.example.net", "ns2.example.net"},
	Handles:     dna.SameHandle(handle), // or dna.ContactHandles{Registrant: "DNA-1", Admin: "DNA-2", ...}
})
```

With inline contacts:

```go
res, err := client.Domains.RegisterWithContacts(ctx, dna.DomainRegisterWithContactsRequest{
	DomainName:  "example.com",
	NameServers: []string{"ns1.example.net", "ns2.example.net"},
	Contacts:    dna.SameContact(owner), // or dna.DomainContacts{Registrant: &owner, Admin: &admin, ...}
})
```

Contacts are tidied before sending: fields are trimmed, the country is upper-cased and phone fields keep only
their digits (`"+90 (555) 123-45-67"` works). Some TLDs need registry attributes: find them with
`client.Products.TLD(ctx, "com.tr")` and its `RequiredAttributes()`, and pass them as `TLDAttributes`.

If the API answers but refuses (`"success": false`), the error is an `*dna.OperationError` with the API's
message.

### Renew, transfer, restore

```go
check, err := client.Domains.RenewCheck(ctx, "example.com", 1) // check.Renewable, check.PremiumPrice
res, err := client.Domains.Renew(ctx, "example.com", 1)

tc, err := client.Domains.TransferCheck(ctx, "example.com", "auth-code") // tc.Transferable, tc.AuthCodeValid
res, err := client.Domains.Transfer(ctx, dna.DomainTransferRequest{DomainName: "example.com", AuthCode: "auth-code"})
st, err := client.Domains.TransferStatus(ctx, "example.com")

err = client.Domains.Restore(ctx, "example.com") // during the redemption period
```

### Domain details and the domain list

```go
d, err := client.Domains.Info(ctx, "example.com")
fmt.Println(d.Status, d.ExpirationDate, d.RemainingDays, d.Locked, d.NameServers, d.AuthCode)
if owner, ok := d.Contact(dna.ContactTypeRegistrant); ok {
	fmt.Println(owner.Email)
}

// One page
page, err := client.Domains.List(ctx, dna.DomainListRequest{ListOptions: dna.ListOptions{Limit: 50}})

// Every domain; pages are fetched for you
for d, err := range client.Domains.All(ctx, dna.DomainListRequest{
	Statuses: []dna.DomainStatus{dna.DomainStatusActive},
}) {
	if err != nil {
		return err
	}
	fmt.Println(d.DomainName, d.ExpirationDate)
}
```

### Lock, privacy, nameservers, child nameservers

```go
err = client.Domains.Lock(ctx, "example.com")
err = client.Domains.Unlock(ctx, "example.com")
err = client.Domains.SetPrivacy(ctx, "example.com", true)
code, err := client.Domains.UpdateAuthCode(ctx, "example.com")

err = client.Domains.SetNameServers(ctx, dna.DomainNameServersSetRequest{
	DomainName:  "example.com",
	NameServers: []string{"ns1.example.net", "ns2.example.net"},
	// DNSSECMode: dna.DNSSECModeDisable, // only for DNSSEC-signed domains
})

err = client.Domains.AddHost(ctx, "example.com", "ns1.example.com", "192.0.2.10", "2001:db8::10") // IP version detected
err = client.Domains.DeleteHost(ctx, "example.com", "ns1.example.com")
```

### Contacts of a domain

```go
err = client.Domains.SetContactHandles(ctx, "example.com", dna.ContactHandles{Admin: "DNA-2", Tech: "DNA-2"})
err = client.Domains.UpdateContacts(ctx, "example.com", dna.DomainContacts{Billing: &billing})

details, err := client.Contacts.Get(ctx, "DNA-1")           // the contact and its verification state
result, err := client.Contacts.Update(ctx, "DNA-1", contact) // updates every domain using the handle
```

### DNS zone

```go
err = client.DNS.Add(ctx, dna.DNSAddRequest{
	DomainName: "example.com", Name: "www", Type: dna.RecordTypeA, TTL: 600, Values: []string{"192.0.2.10"},
})
err = client.DNS.Update(ctx, dna.DNSUpdateRequest{ // replaces all values of the set
	DomainName: "example.com", RecordName: "www", Type: dna.RecordTypeA, Values: []string{"192.0.2.20"},
})
err = client.DNS.Delete(ctx, dna.DNSDeleteRequest{
	DomainName: "example.com", Name: "www", Type: dna.RecordTypeA, Value: "192.0.2.20",
})

sets, err := client.DNS.List(ctx, "example.com")
for _, s := range sets {
	fmt.Println(s.Name, s.Type, s.TTL, s.Values())
}

res, err := client.DNS.Apply(ctx, "example.com") // switch the domain to the platform nameservers
```

Names can be relative (`www`), the apex (`""` or `"@"`) or fully qualified. Record types are case-insensitive
here (`"txt"` is sent as `TXT`). Validation catches what the API would reject: a CNAME at the apex, a CAA value
without three parts, an MX value without a priority. The test environment answers every zone call with HTTP
500, so zone code can only be tried in production.

### DNSSEC

```go
ds, err := dna.ParseDS("2371 13 2 1F987CC6583E92DF0890718C42...") // as your DNS provider shows it
results, err := client.DNSSEC.Add(ctx, "example.com", ds)
info, err := client.DNSSEC.Get(ctx, "example.com") // info.Status, info.DSRecords
res, err := client.DNSSEC.Disable(ctx, "example.com")
```

### URL forwarding

```go
err = client.Domains.SetForwarding(ctx, "example.com", "https://www.example.org", dna.ForwardTypeStandard)
f, err := client.Domains.Forwarding(ctx, "example.com") // nil when the domain has none
err = client.Domains.DeleteForwarding(ctx, "example.com")
```

### Registry documents (.tr)

```go
doc, err := dna.DocumentFromFile(dna.DocumentTypeIdentityCard, "id.pdf")
res, err := client.Domains.UploadDocuments(ctx, "example.com.tr", doc)
page, err := client.Domains.Documents(ctx, dna.DomainDocumentsListRequest{DomainName: "example.com.tr"})
```

Files are base64-encoded for you; format (PDF, JPG, PNG, DOC, DOCX, ODT) and size (10 MB per file, 50 MB per
request) are checked before upload.

### SSL certificates

```go
csr, err := client.SSL.GenerateCSR(ctx, dna.SSLGenerateCSRRequest{
	DomainName: "example.com", Country: "TR", State: "Istanbul", Locality: "Istanbul",
	Organization: "Example Ltd", OrganizationUnit: "IT",
})
_, err = client.SSL.Initialize(ctx, "example.com", "PositiveSSL") // charges the balance
_, err = client.SSL.Configure(ctx, dna.SSLConfigureRequest{
	DomainName: "example.com", CSR: csr.CSR, ApproverMail: "admin@example.com",
	AuthenticationType: dna.AuthenticationEmail, WebServerType: dna.WebServerApache, Contact: sslContact,
})
st, err := client.SSL.Status(ctx, "example.com") // poll until st.IsIssued
cert, err := client.SSL.Content(ctx, "example.com")
os.WriteFile("example.com.crt", []byte(cert.Chain()), 0o644)
```

### Hosting

```go
packages, err := client.Hosting.Packages(ctx)
created, err := client.Hosting.Create(ctx, "example.com", "Linux-Starter") // charges the balance

ctx, cancel := context.WithTimeout(ctx, 30*time.Minute)
defer cancel()
st, err := client.Hosting.Wait(ctx, created.ServiceID) // polls until active
fmt.Println(st.PanelAddress, st.Username, st.Password)
```

Never call `Create` again for a hosting that is still provisioning: that charges twice.

### Balance, prices and orders

```go
account, err := client.Account.Get(ctx)
usd, _ := account.Balance("USD")

for tx, err := range client.Account.AllTransactions(ctx, dna.AccountTransactionsRequest{From: monthStart}) {
	if err != nil {
		return err
	}
	fmt.Println(tx.CreationTime, tx.Type, tx.Change())
}

if tld, err := client.Products.TLD(ctx, "com.tr"); err == nil && tld != nil {
	price, _ := tld.RegisterPrice(1)
	fmt.Println(tld.Name, price)
}

info, err := client.Products.Info(ctx, "com", dna.OrderRenew, 2) // price of a 2-year .com renewal
order, err := client.Orders.Get(ctx, "ORDER-CODE")
```

## Requests and BaseRequest

Every API operation is a request type: `DomainInfoRequest`, `DNSAddRequest`, `SSLConfigureRequest`, ... They
all implement one interface:

```go
type Request interface {
	Endpoint() Endpoint      // method, path and safety class
	Query() url.Values       // query string, or nil
	Body() any               // JSON body, or nil
	Validate() error         // checked before anything is sent
}
```

Go has no inheritance, so the "base class" is a struct you embed. `dna.BaseRequest` supplies the defaults (no
query, no body, nothing to validate); a request overrides a method by defining it on its own type. Only
`Endpoint` has no default. A request from this package looks like this:

```go
type DomainRenewRequest struct {
	BaseRequest
	DomainName string
	Years      int
}

func (DomainRenewRequest) Endpoint() Endpoint {
	return Endpoint{Method: http.MethodPost, Path: "/api/v1/domains/renew", Safety: Billable}
}

func (r DomainRenewRequest) Body() any { return domainBody(r.DomainName).set("period", r.Years) }

func (r DomainRenewRequest) Validate() error {
	var v validator
	v.required("domainName", r.DomainName)
	v.check(r.Years >= 1 && r.Years <= 10, "years", "must be between 1 and 10")
	return v.err()
}
```

Service methods build these for you. `client.Do` sends any `Request`, including types of your own, for example
to read only the fields you need or to call an endpoint this package does not wrap yet:

```go
type ExpiryRequest struct {
	dna.BaseRequest
	DomainName string
}

func (ExpiryRequest) Endpoint() dna.Endpoint {
	return dna.Endpoint{Method: http.MethodGet, Path: "/api/v1/domains/info", Safety: dna.Read}
}

func (r ExpiryRequest) Query() url.Values { return url.Values{"DomainName": {r.DomainName}} }

var out struct {
	ExpirationDate dna.Time `json:"expirationDate"`
}
resp, err := client.Do(ctx, ExpiryRequest{DomainName: "example.com"}, &out)
```

Your request gets everything the built-in ones get: credentials, retries, pacing, error normalisation,
correlation IDs, tracing and read-only enforcement. `client.NewHTTPRequest(ctx, req)` builds the exact HTTP
request without sending it, to log or review a billable request before it goes out.

## Errors

| Error                  | When                                                       | Useful fields                                                    |
|------------------------|------------------------------------------------------------|------------------------------------------------------------------|
| `*dna.InputError`      | `Validate` rejected the request; **nothing was sent**      | `Request`, `Fields`                                              |
| `*dna.APIError`        | the API answered with a non-2xx status                     | `StatusCode`, `Code`, `Message`, `Details`, `Fields`, `Data`, `CorrelationID`, `Hint()` |
| `*dna.OperationError`  | the API answered 2xx with `"success": false`               | `Code`, `Message`, `CorrelationID`                               |
| `*dna.NetworkError`    | the request did not complete (DNS, TLS, timeout, reset)    | `Err`, `Timeout()`                                               |
| `dna.ErrReadOnly`      | a read-only client blocked a change; nothing was sent      | use `errors.Is`                                                  |

`dna.KindOf(err)` classifies any of them: `KindAuth`, `KindNotFound`, `KindValidation`, `KindRejected`,
`KindConflict`, `KindRateLimited`, `KindServer`, `KindNetwork`, `KindInvalidInput`, `KindReadOnly`. Helpers
answer the common questions:

```go
_, err := client.Domains.Renew(ctx, "example.com", 1)
var apiErr *dna.APIError
switch {
case err == nil:
case dna.OutcomeUnknown(err):
	// The renewal may have gone through. Check the domain before trying again.
case dna.IsInsufficientBalance(err):
case dna.IsNotFound(err):
case dna.HasCode(err, dna.CodePrivacyNotSupported):
case errors.As(err, &apiErr):
	log.Println(apiErr.Code, apiErr.Message, apiErr.Hint(), apiErr.CorrelationID)
}
```

Branch on `Code`, not on `Message`: messages are localised. API codes look like
`Dna.DomainService:Domain:10007`; see the [error reference](https://www.domainnameapi.com/docs/reference/errors.md).
Quote the `CorrelationID` when contacting support.

## Safety and reliability

- **Safety classes.** Every request is `Read`, `Write`, `Destructive` or `Billable` (`req.Endpoint().Safety`).
  Some POST and PUT requests are reads (`DomainSearchRequest`, `SSLStatusRequest`).
- **Read-only mode.** `dna.WithReadOnly()` blocks everything that is not a read, for reporting jobs and for
  exploring production safely.
- **Retries.** 429 responses are always retried, honouring `Retry-After`: the API rejects them before doing
  anything. Network errors and 502/503/504 are retried only for reads and idempotent writes, so a registration
  or renewal is never sent twice. After a failed billable request, check `dna.OutcomeUnknown(err)`.
- **Pacing.** The client spaces requests to stay under the per-key limits (5 requests/second, 2 searches/second,
  one zone call per second). Turn it off with `dna.WithPacing(false)` if you pace yourself.
- **Quota.** `Response.Quota` exposes the soft-quota headers. Over the limit the API slows responses down (up to
  5 s) instead of rejecting them; the default 60 s timeout leaves room for that.
- **Correlation IDs.** Each request carries an `X-Correlation-ID`. Set your own with
  `dna.WithCorrelationID(ctx, id)` to tie API calls to your logs.
- **Observability.** `dna.WithLogger(slog.Default())` logs every exchange; `dna.WithTrace(func(dna.TraceEvent))`
  hands you every request, response and retry, for metrics or error reporting. The API key is never in them.
- **Concurrency.** A `Client` is safe for concurrent use. Create one and share it.
- **Tolerant decoding.** Dates without a timezone, `null`, `[]` instead of `{}`, numeric or named statuses and
  roles in any case (`"registrant"`, `"ADMINISTRATIVE"`) all decode.

## Client options

| Option                          | Effect                                                                                   |
|---------------------------------|------------------------------------------------------------------------------------------|
| `WithEnvironment(dna.OTE)`      | use the test environment (default `dna.Production`)                                      |
| `WithBaseURL(url)`              | send requests to a proxy or a test server                                                |
| `WithHTTPClient(hc)`            | your own `*http.Client` (transport, proxy, TLS); redirects are never followed            |
| `WithTimeout(d)`                | per-attempt timeout (default 60 s)                                                       |
| `WithUserAgent("app/1.0")`      | put your product in front of the User-Agent                                              |
| `WithCurrency("TRY")`           | currency of search and catalogue prices                                                  |
| `WithRetries(n)`                | retry count (default 2)                                                                  |
| `WithPacing(false)`             | turn client-side pacing off                                                              |
| `WithReadOnly()`                | block every non-read request                                                             |
| `WithLogger(l)`                 | log exchanges to a `*slog.Logger`                                                        |
| `WithTrace(fn)`                 | observe every request, response and retry                                                |

## Testing your code

Point the client at an `httptest.Server`; nothing else needs mocking:

```go
srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, `{"success":true,"info":{"domainName":"example.com","status":"AVAILABLE","price":11.99,"currency":"USD"}}`)
}))
defer srv.Close()

client, _ := dna.NewClient("reseller-id", "api-key", dna.WithBaseURL(srv.URL), dna.WithPacing(false))
a, err := client.Domains.Search(ctx, "example.com")
```

## Migrating from v1

v2 is a new design: typed results instead of `map[string]any`, errors instead of `{"result": "ERROR"}`
envelopes, and the full API instead of the php-dna method set. The module path changed to
`github.com/domainreseller/go-dna/v2`, so v1 keeps working until you switch.

| v1                                                  | v2                                                                     |
|-----------------------------------------------------|------------------------------------------------------------------------|
| `dna.New(id, key, true)`                            | `dna.NewClient(id, key, dna.WithEnvironment(dna.OTE))`                 |
| `res["result"] == "OK"`                             | `err == nil`                                                           |
| `GetResellerDetails`, `GetCurrentBalance`           | `Account.Get` (+ `Account.Balance("USD")`)                             |
| `CheckAvailability`                                 | `Domains.Search`, `Domains.BulkSearch`                                 |
| `GetList`                                           | `Domains.List`, `Domains.All`                                          |
| `GetTldList`                                        | `Products.TLDs`, `Products.AllTLDs`, `Products.TLD`                    |
| `GetDetails`, `GetContacts`, `SyncFromRegistry`     | `Domains.Info`                                                         |
| `RegisterWithContactInfo`                           | `Domains.RegisterWithContacts` (or `Contacts.Create` + `Domains.Register`) |
| `Renew`, `Transfer`, `CheckTransfer`                | `Domains.Renew`, `Domains.Transfer`, `Domains.TransferCheck`           |
| `ModifyNameServer`                                  | `Domains.SetNameServers`                                               |
| `EnableTheftProtectionLock` / `Disable...`          | `Domains.Lock` / `Domains.Unlock`                                      |
| `ModifyPrivacyProtectionStatus`                     | `Domains.SetPrivacy`                                                   |
| `SaveContacts`                                      | `Domains.UpdateContacts`, `Domains.SetContactHandles`                  |
| `AddChildNameServer`, `ModifyChildNameServer`, `DeleteChildNameServer` | `Domains.AddHost`, `Domains.UpdateHost`, `Domains.DeleteHost` |
| `GetResourceRecords`                                | `DNS.List`                                                             |
| `AddResourceRecord`, `AddResourceRecordValue`       | `DNS.Add`                                                              |
| `EditResourceRecord`, `SetResourceRecordSet`, `ReplaceResourceRecordValue` | `DNS.Update`                                    |
| `DeleteResourceRecord`, `RemoveResourceRecordValue`, `DeleteResourceRecordSet` | `DNS.Delete` (once per value)               |
| `ApplyResourceRecords`                              | `DNS.Apply`                                                            |
| `GetForwarding`, `SetForwarding`, `DeleteForwarding`| `Domains.Forwarding`, `Domains.SetForwarding`, `Domains.DeleteForwarding` |
| `CancelTransfer`, `ApproveTransfer`, `RejectTransfer` | not in the published API; send a [request of your own](#requests-and-baserequest) if your account has them |
| `IsTrTLD`, `ValidateContact`                        | `strings.HasSuffix(name, ".tr")`; validation runs on every request     |
| `GetRequestData`, `GetResponseData`, ...            | `WithTrace`, `WithLogger`, `Response` from `client.Do`                 |
| telemetry (`SetErrorReportingEnabled`, Sentry)      | removed: nothing leaves your process; forward `WithTrace` events to your own reporting |
| `SetSSLVerification(false)`                         | `WithHTTPClient` with your own TLS settings                            |

## Development

```bash
go test ./...          # unit tests; none touches the network
go vet ./...
gofmt -l .
```

`testdata/openapi.json` is the API's OpenAPI document. The tests check every request type against it (method,
path, query names, body fields, required fields), every response type's JSON names against the response
schemas and every enum value, and fail when the API gains an operation without a request type. To update it:

```bash
curl -fsSL -A "go-dna-spec-update" -o testdata/openapi.json https://api.domainresellerapi.com/swagger/v1/swagger.json
go test ./...
```

See [AGENTS.md](AGENTS.md) for the conventions of this repository.

## Support

- API documentation: <https://www.domainnameapi.com/docs/>
- CLI and MCP server: <https://github.com/domainreseller/cli>
- Website: <https://www.domainnameapi.com>

## License

Apache License 2.0. See [LICENSE](LICENSE).
