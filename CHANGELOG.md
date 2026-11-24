# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- Every request carries the headers `X-App` and `X-Version`, which tell DomainNameAPI which integration sent it.
  They default to `go` and the SDK version. `WithApp(name, version)` names your application instead, and
  `NewClientFromEnv` reads `DNA_APP` and `DNA_APP_VERSION`. The name is lower-cased and keeps only `a-z`, `0-9`,
  `.`, `_` and `-` (an empty or invalid name falls back to `go`); the version keeps only digits and dots, so
  `V2.0.20-beta` is sent as `2.0.20` (an empty version falls back to the SDK version). The User-Agent is unchanged.

## [2.1.0] - 2026-11-22

**Breaking, and forced by the API.** On 2026-11-20 the API changed its SSL endpoints: a domain can now have
several certificates, so an existing certificate is addressed by its `certificateId` (a GUID) and no longer by
its domain name. The API rejects the old requests, so the SSL calls of 2.0.0 that name a domain no longer work
in production or OTE. What you must change is listed under "Changed".

### Changed

- **Breaking:** `SSL.Status`, `SSL.Get`, `SSL.Content` and `SSL.Revoke` take the certificate ID instead of the
  domain name. Keep the `CertificateID` that `SSL.Initialize` returns and pass it to the later calls. For
  certificates you already have, look the ID up with
  `SSL.List(ctx, dna.SSLListRequest{DomainName: "example.com"})`.
- **Breaking:** `SSLConfigureRequest`, `SSLReissueRequest`, `SSLStatusRequest`, `SSLGetRequest`,
  `SSLContentRequest` and `SSLRevokeRequest` have a `CertificateID` field in place of `DomainName`. The CSR
  given to `SSL.Configure` must carry the domain of the certificate.
- **Breaking:** `SSL.Approvers(ctx, domainName, packageCode)` is now
  `SSL.Approvers(ctx, dna.SSLApproversRequest{...})`. Set `CertificateID` for an existing certificate, or
  `DomainName` and `PackageCode` before a purchase; with a domain name the package code is no longer optional.
- A domain name passed where a certificate ID belongs fails with an `*InputError` before anything is sent. Both
  are strings, so the compiler cannot catch `SSL.Status(ctx, "example.com")`; the error says to use the ID from
  `SSL.Initialize` or `SSL.Renew`, or to look it up with `SSL.List`.
- A certificate must now be configured within 15 days of `SSL.Initialize` or `SSL.Renew` (the API's window was
  7 days), and a renewal is `SSL.Renew` followed by `SSL.Configure`. Do not renew with a second
  `SSL.Initialize`: that is an unrelated purchase at the purchase price, and the old certificate never becomes
  `Renewed`.
- `APIError.Hint` for a certificate that is not found points to `SSL.List` instead of `Domains.List`.
- `testdata/openapi.json` is the API's current OpenAPI document (67 operations).

### Added

- `SSL.Renew(ctx, certificateID)` and `SSLRenewRequest`: renews a certificate. It charges the balance, so it is
  a `Billable` request: never retried after a network error and blocked by `WithReadOnly`. The result carries
  the ID of the new certificate, which you then configure.
- `SSL.List`, the iterator `SSL.All`, `SSLListRequest` and `CertificateListItem`: your certificates, newest
  first, optionally of one domain.
- `CertificateID` on `Certificate`, `CertificateStatus` and `CertificateContent`.
- `Account.Profile`, `AccountProfileRequest` and `AccountProfile`: the reseller's profile, with the new enums
  `AccountCurrency` and `ResellerStatus`.

## [2.0.0] - 2026-11-19

A new SDK, designed for Go and for the REST API instead of ported from php-dna. The module path is now
`github.com/domainreseller/go-dna/v2`; v1 keeps working under the old path. See "Migrating from v1" in the
README.

### Added

- All 64 operations of the REST API, grouped like the `dna` CLI: `Account`, `Contacts`, `Domains`, `DNS`,
  `DNSSEC`, `SSL`, `Hosting`, `Orders` and `Products`. New compared to v1: contact handles and verification,
  registration with handles, renew check, restore, auth code update, registry documents, DNSSEC, SSL
  certificates, hosting, orders, balance transactions and product prices.
- Typed results, typed enums with names (`DomainStatusPendingDelete`) and tolerant decoding of the API's
  quirks: dates without a timezone, `[]` instead of `{}`, numeric or named statuses, roles in any case.
- A request type per operation (`DomainRegisterRequest`, `DNSAddRequest`, ...) implementing `Request`. Every
  request embeds `BaseRequest`, validates itself before sending and has a safety class (`Read`, `Write`,
  `Destructive`, `Billable`). `Client.Do` sends your own request types too; `Client.NewHTTPRequest` previews a
  request without sending it.
- Typed errors (`APIError`, `OperationError` for `"success": false`, `InputError`, `NetworkError`) with
  `KindOf`, `IsNotFound`, `IsAuth`, `IsRateLimited`, `IsInsufficientBalance`, `HasCode`, `OutcomeUnknown` and
  `APIError.Hint`.
- Iterators over paged lists: `Domains.All`, `Account.AllTransactions`, `Products.AllTLDs`.
- `NewClientFromEnv` with the CLI's environment variables, `WithReadOnly`, `WithCurrency`, `WithLogger`,
  `WithTrace`, `WithCorrelationID`, quota headers in `Response.Quota`, and `Hosting.Wait`.
- Retries on 429 for every request and on network errors and 502/503/504 only for requests that are safe to
  repeat; client-side pacing under the rate limits; a client that is safe for concurrent use.
- Tests that check every request type, response type and enum against the API's OpenAPI document.

### Removed

- The php-dna compatible API: `dna.New`, the `map[string]any` results, the `{"result": "ERROR"}` envelopes and
  the diagnostics getters (`GetRequestData`, ...). Use `WithTrace` or `WithLogger` instead.
- Telemetry: the SDK no longer sends anything to Sentry. Forward `WithTrace` events to your own reporting.
- `CancelTransfer`, `ApproveTransfer` and `RejectTransfer`, which called endpoints that are not part of the
  published API.

## [1.0.1] - 2026-11-14

### Changed

- Error reporting now has its own Sentry project: `DefaultSentryDSN` points to DomainNameAPI's
  go-dna project, so telemetry is active by default as documented. Turn it off with
  `SetErrorReportingEnabled(false)`; `WithSentryDSN` still overrides the target.

## [1.0.0] - 2026-11-09

### Added

- First release: a faithful Go port of php-dna (4.0.4 plus the fixes of php-dna commit f946d69), the
  DomainNameAPI REST client.
- `dna.New(resellerID, apiKey, testMode, ...Option)` with trimmed credentials and a test-mode flag that
  also accepts the strings `"true"`, `"1"`, `"on"` and `"yes"`.
- All php-dna methods with the same requests on the wire and the same `map[string]any` / `[]any`
  return values: account (`GetResellerDetails`, `GetCurrentBalance`), search and lists
  (`CheckAvailability`, `GetList`, `GetTldList`), domains (`GetDetails`, `SyncFromRegistry`,
  `GetContacts`, `SaveContacts`, `ModifyNameServer`, theft-protection lock, privacy, `Renew`,
  `RegisterWithContactInfo`), child nameservers, transfers (`Transfer`, `CancelTransfer`,
  `ApproveTransfer`, `RejectTransfer`, `CheckTransfer`; transfers never send contacts), DNS zone management (record and record-set
  methods with zone-call pacing), URL forwarding, utilities (`IsTrTLD`, `ValidateContact`),
  diagnostics and settings.
- php-dna's error mapping: `API_<status>` codes with the gateway's own code as `GatewayCode`,
  synthesised messages for empty error bodies (never the raw body), flattened validation errors, ABP
  boilerplate replacement, 301/302 as credential errors and transport failures (timeout, DNS, TLS,
  reset) as errors with `CURL_<errno>` codes.
- Refusals and failed steps are reported instead of false successes: `success:false` from register,
  transfer and renew (`REGISTER_DOMAIN`, `TRANSFER_DOMAIN`, `RENEW_DOMAIN`), a failed automatic zone
  apply, currencies the account does not report (`BALANCE`), and forwarding lookups on an unknown
  domain. `DeleteForwarding` on an unset forwarding succeeds with `Removed: false`.
- `.tr` TRABIS attribute normalization and contact completion/normalization (names, phone numbers,
  country names).
- Sentry telemetry (errors and sampled performance records) with credential masking, a minimal mode
  and an off switch; the DSN is empty until the DomainNameAPI-Go Sentry project exists.
- Ordered `dna.Contacts` input with `AllRoles` and `ContactsFromMap` helpers, and `RegisterOptions`
  for `RegisterWithContactInfo`'s optional arguments.
- Injectable HTTP transport (`WithHTTPClient`), `User-Agent: DomainNameAPI-Go/1.0.0`, 30 s / 10 s
  timeouts and optional TLS verification opt-out.
- Conformance test running all 134 php-dna fixtures, unit tests, runnable examples for every method,
  README in nine languages and GitHub Actions CI.

[1.0.0]: https://github.com/domainreseller/go-dna/releases/tag/v1.0.0
