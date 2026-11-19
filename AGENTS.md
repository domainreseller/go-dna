# AGENTS.md

Guidance for AI coding agents working on this repository.

## What this is

`go-dna` is the Go SDK for the Domain Name API reseller REST API (module `github.com/domainreseller/go-dna/v2`,
package `dna`). It mirrors the [`dna` CLI](https://github.com/domainreseller/cli): the same operations, the same
groups, the same environment variables and the same safety classes. It has nothing to do with php-dna or the
SOAP API.

## Commands

```bash
go test ./...      # all tests; none touches the network
go vet ./...
gofmt -l .         # must print nothing
```

## Layout

| file | content |
|---|---|
| `client.go` | `Client`, `NewClient`, `NewClientFromEnv`, options, environments |
| `transport.go` | `Do`, retries, pacing, quota, tracing, correlation IDs |
| `request.go` | the `Request` interface, `BaseRequest`, `Endpoint`, `Safety`, `ListOptions`, query/body/validation helpers |
| `errors.go` | error types, kinds, codes and the normalisation of API error responses |
| `types.go`, `enums.go` | `Time`, `Attributes`, `Page`, pagination, enums |
| `account.go`, `contacts.go`, `domains.go`, `domain_*.go`, `dns.go`, `dnssec.go`, `ssl.go`, `hosting.go`, `orders.go`, `products.go` | one file per CLI group: models, request types, service methods |
| `testdata/openapi.json` | the API's OpenAPI document, the source of truth |

## Adding or changing an operation

1. Update `testdata/openapi.json` (see README, "Development") and run `go test ./...`. `TestSpecCoverage` fails
   for any operation without a request type.
2. Add a request type named after the CLI command (`dna domain renew-check` → `DomainRenewCheckRequest`). It
   embeds `BaseRequest`, defines `Endpoint` (method, path, safety) and overrides `Query`, `Body` and `Validate`
   as needed. Request fields are Go-friendly; the wire format lives in `Query` and `Body`.
3. Add the service method. Required inputs are arguments; operations with optional inputs take the request
   struct. Turn `"success": false` answers into an `*OperationError`.
4. Add the case to `specCases` in `spec_test.go` (every field set) and to `TestEveryServiceMethod`, plus a
   behaviour test in `services_test.go`.
5. New integer enums go in `enums.go` and in `TestSpecEnums`.

## Rules

- **Safety class** is a product decision and matches the CLI: `Billable` charges the balance, `Destructive`
  deletes or is hard to undo, `Write` is free and reversible, `Read` has no side effects even when the method is
  POST or PUT. Set `Idempotent` only when sending the request twice is harmless.
- **Never retry a billable or non-idempotent request after a network error.** Do not weaken the retry rules in
  `transport.go`.
- **No telemetry.** Nothing may leave the user's process except the API requests they make.
- **Standard library only.**
- Tests never call the real API; use the `fakeAPI` helper in `helpers_test.go`.
- English for code, comments and docs. Error messages say what happened and what to do.
- `README.md` is the source; keep the eight `README-XX.md` translations in sync when it changes.
- Commits: imperative subject line. Do not add `Co-Authored-By` trailers for AI tools.
