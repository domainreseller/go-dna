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

# go-dna: das Go-SDK für die Domain Name API

[![Go Reference](https://pkg.go.dev/badge/github.com/domainreseller/go-dna/v2.svg)](https://pkg.go.dev/github.com/domainreseller/go-dna/v2)
[![CI](https://github.com/domainreseller/go-dna/actions/workflows/ci.yml/badge.svg)](https://github.com/domainreseller/go-dna/actions/workflows/ci.yml)
[![License](https://img.shields.io/badge/license-Apache--2.0-blue.svg)](LICENSE)

Typisierter Go-Client für die Reseller-REST-API der [Domain Name API](https://www.domainnameapi.com): Domainsuche,
Registrierung, Verlängerung und Transfer, DNS, DNSSEC, Kontakte, SSL-Zertifikate, Hosting, Bestellungen, der
Produktkatalog und Ihr Guthaben.

- **Alle 67 API-Operationen**, genauso gruppiert wie in der [`dna`-CLI](https://github.com/domainreseller/cli):
  `dna domain info` entspricht `client.Domains.Info`, `dna dns add` entspricht `client.DNS.Add`.
- **Typisierte Ergebnisse und Enums**: `d.Status == dna.DomainStatusPendingDelete`, `d.ExpirationDate.Before(...)`,
  kein `map[string]any`.
- **Jede Operation ist ein Request-Typ**, der `dna.BaseRequest` einbettet und sich selbst validiert, bevor
  etwas gesendet wird. Eigene Requests schreiben Sie auf dieselbe Weise.
- **Sicher von Haus aus**: kostenpflichtige Requests werden nie blind wiederholt, `WithReadOnly` blockiert
  Änderungen, der Client drosselt sich selbst unter den Rate Limits und ist nebenläufig sicher nutzbar.
- **Nur die Standardbibliothek.** Go 1.23 oder neuer.

```go
client, _ := dna.NewClient(resellerID, apiKey, dna.WithEnvironment(dna.OTE))

a, err := client.Domains.Search(ctx, "example.com")
if err == nil && a.Available() {
	fmt.Printf("%s is available for %.2f %s\n", a.DomainName, a.Price, a.Currency)
}
```

## Inhalt

- [Installation](#installation)
- [Zugangsdaten und Umgebungen](#zugangsdaten-und-umgebungen)
- [Services](#services)
- [Verwendung](#verwendung)
- [Requests und BaseRequest](#requests-und-baserequest)
- [Fehler](#fehler)
- [Sicherheit und Zuverlässigkeit](#sicherheit-und-zuverlässigkeit)
- [Client-Optionen](#client-optionen)
- [Den eigenen Code testen](#den-eigenen-code-testen)
- [Migration von v1](#migration-von-v1)
- [Entwicklung](#entwicklung)

## Installation

```bash
go get github.com/domainreseller/go-dna/v2
```

```go
import dna "github.com/domainreseller/go-dna/v2"
```

Lauffähige Programme finden Sie in [examples](examples): `quickstart`, `register`, `expiring`, `dns`, `ssl` und
`custom-request`.

```bash
go run ./examples/quickstart example.com
```

## Zugangsdaten und Umgebungen

Sie benötigen zwei API-Zugangsdaten aus dem Reseller-Panel (**API settings**). Die Anmeldedaten des Panels funktionieren nicht.

| Zugangsdaten | Sieht so aus                           |
|--------------|----------------------------------------|
| Reseller ID  | `xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx` |
| API key      | eine lange geheime Zeichenfolge        |

Es gibt zwei Umgebungen. **Beginnen Sie mit OTE**, der Testumgebung: Sie hat eigene Daten und einen eigenen API-Schlüssel,
und nichts, was Sie dort tun, kostet Geld oder berührt Live-Domains.

```go
// Testumgebung (OTE)
client, err := dna.NewClient(resellerID, oteAPIKey, dna.WithEnvironment(dna.OTE))

// Produktion (Standard): echte Registrierungen, echte Abbuchungen
client, err := dna.NewClient(resellerID, apiKey)
```

`NewClientFromEnv` liest dieselben Variablen wie die `dna`-CLI:

| Variable          | Bedeutung                                                         |
|-------------------|-------------------------------------------------------------------|
| `DNA_RESELLER_ID` | Reseller ID                                                       |
| `DNA_API_KEY`     | API-Schlüssel der gewählten Umgebung                              |
| `DNA_ENV`         | `production` (Standard) oder `ote`                                |
| `DNA_TEST_MODE`   | `true` wählt OTE (wie in den anderen DomainNameAPI-SDKs)          |
| `DNA_BASE_URL`    | eine eigene Basis-URL, z. B. ein Proxy                            |
| `DNA_APP`         | die Anwendung, die das SDK nutzt, gesendet als `X-App`            |
| `DNA_APP_VERSION` | die Version dieser Anwendung, gesendet als `X-Version`            |

```go
client, err := dna.NewClientFromEnv(dna.WithUserAgent("my-panel/1.4"))
```

Ein falscher Schlüssel oder eine falsche Umgebung schlägt beim ersten Request mit einem `*APIError` der Art `KindAuth` und einem Hinweis fehl;
`client.Account.Get(ctx)` ist die günstigste Möglichkeit zur Prüfung.

## Services

| Service           | CLI-Gruppe    | Operationen                                                                                                                                                       |
|-------------------|---------------|-------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `client.Account`  | `dna account` | `Get`, `Profile`, `Transactions`, `AllTransactions`                                                                                                               |
| `client.Contacts` | `dna contact` | `Get`, `Create`, `Update`, `Verification`, `ResendVerification`                                                                                                   |
| `client.Domains`  | `dna domain`  | `Search`, `BulkSearch`, `List`, `All`, `Info`, `Register`, `RegisterWithContacts`, `Renew`, `RenewCheck`, `Restore`, `Transfer`, `TransferCheck`, `TransferStatus`, `Lock`, `Unlock`, `SetPrivacy`, `UpdateAuthCode`, `SetNameServers`, `AddHost`, `UpdateHost`, `DeleteHost`, `UpdateContacts`, `SetContactHandles`, `Documents`, `UploadDocuments`, `Forwarding`, `SetForwarding`, `DeleteForwarding` |
| `client.DNS`      | `dna dns`     | `List`, `Add`, `Update`, `Delete`, `Apply`                                                                                                                        |
| `client.DNSSEC`   | `dna dnssec`  | `Get`, `Add`, `Remove`, `Disable`                                                                                                                                 |
| `client.SSL`      | `dna ssl`     | `GenerateCSR`, `CheckCSR`, `Initialize`, `Configure`, `Status`, `Get`, `Content`, `Reissue`, `Renew`, `Approvers`, `Revoke`, `List`, `All`                        |
| `client.Hosting`  | `dna hosting` | `Packages`, `Plans`, `Create`, `Wait`, `Get`, `GetByToken`, `Status`                                                                                              |
| `client.Orders`   | `dna order`   | `Get`                                                                                                                                                             |
| `client.Products` | `dna product` | `List`, `Info`, `TLDs`, `AllTLDs`, `TLD`, `SSLs`                                                                                                                  |

Methoden nehmen die erforderlichen Eingaben als Argumente (`Info(ctx, "example.com")`) oder, wenn eine Operation optionale
Eingaben hat, eine Request-Struktur (`Register(ctx, dna.DomainRegisterRequest{...})`). Die vollständige Referenz mit allen Feldern
finden Sie auf [pkg.go.dev](https://pkg.go.dev/github.com/domainreseller/go-dna/v2).

## Verwendung

Alle Snippets setzen `ctx := context.Background()` und einen `client` voraus.

### Suche

```go
a, err := client.Domains.Search(ctx, "example.com")
// a.Available(), a.Price, a.Currency, a.IsPremium, a.IsDocumentRequired, a.Reason

results, err := client.Domains.BulkSearch(ctx, "example.com", "example.net", "example.org")
for _, r := range results {
	fmt.Println(r.DomainName, r.Status, r.Price)
}
```

`BulkSearch` teilt Listen mit mehr als 30 Domains auf mehrere Requests auf. Preise verwenden die Währung des Kontos oder
die mit `dna.WithCurrency("TRY")` gesetzte.

### Registrieren

Mit Kontakt-Handles (eines erstellen Sie mit `Contacts.Create`):

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
	Handles:     dna.SameHandle(handle), // oder dna.ContactHandles{Registrant: "DNA-1", Admin: "DNA-2", ...}
})
```

Mit Inline-Kontakten:

```go
res, err := client.Domains.RegisterWithContacts(ctx, dna.DomainRegisterWithContactsRequest{
	DomainName:  "example.com",
	NameServers: []string{"ns1.example.net", "ns2.example.net"},
	Contacts:    dna.SameContact(owner), // oder dna.DomainContacts{Registrant: &owner, Admin: &admin, ...}
})
```

Kontakte werden vor dem Senden bereinigt: Felder werden getrimmt, das Land wird in Großbuchstaben umgewandelt und Telefonfelder behalten nur
ihre Ziffern (`"+90 (555) 123-45-67"` funktioniert). Einige TLDs benötigen Registry-Attribute: Sie finden sie mit
`client.Products.TLD(ctx, "com.tr")` und dessen `RequiredAttributes()` und übergeben sie als `TLDAttributes`.

Antwortet die API, lehnt die Operation aber ab (`"success": false`), ist der Fehler ein `*dna.OperationError` mit der Meldung
der API.

### Verlängern, transferieren, wiederherstellen

```go
check, err := client.Domains.RenewCheck(ctx, "example.com", 1) // check.Renewable, check.PremiumPrice
res, err := client.Domains.Renew(ctx, "example.com", 1)

tc, err := client.Domains.TransferCheck(ctx, "example.com", "auth-code") // tc.Transferable, tc.AuthCodeValid
res, err := client.Domains.Transfer(ctx, dna.DomainTransferRequest{DomainName: "example.com", AuthCode: "auth-code"})
st, err := client.Domains.TransferStatus(ctx, "example.com")

err = client.Domains.Restore(ctx, "example.com") // während der Redemption-Phase
```

### Domain-Details und Domainliste

```go
d, err := client.Domains.Info(ctx, "example.com")
fmt.Println(d.Status, d.ExpirationDate, d.RemainingDays, d.Locked, d.NameServers, d.AuthCode)
if owner, ok := d.Contact(dna.ContactTypeRegistrant); ok {
	fmt.Println(owner.Email)
}

// Eine Seite
page, err := client.Domains.List(ctx, dna.DomainListRequest{ListOptions: dna.ListOptions{Limit: 50}})

// Alle Domains; die Seiten werden für Sie abgerufen
for d, err := range client.Domains.All(ctx, dna.DomainListRequest{
	Statuses: []dna.DomainStatus{dna.DomainStatusActive},
}) {
	if err != nil {
		return err
	}
	fmt.Println(d.DomainName, d.ExpirationDate)
}
```

### Sperre, Datenschutz, Nameserver, Child-Nameserver

```go
err = client.Domains.Lock(ctx, "example.com")
err = client.Domains.Unlock(ctx, "example.com")
err = client.Domains.SetPrivacy(ctx, "example.com", true)
code, err := client.Domains.UpdateAuthCode(ctx, "example.com")

err = client.Domains.SetNameServers(ctx, dna.DomainNameServersSetRequest{
	DomainName:  "example.com",
	NameServers: []string{"ns1.example.net", "ns2.example.net"},
	// DNSSECMode: dna.DNSSECModeDisable, // nur für DNSSEC-signierte Domains
})

err = client.Domains.AddHost(ctx, "example.com", "ns1.example.com", "192.0.2.10", "2001:db8::10") // IP-Version wird erkannt
err = client.Domains.DeleteHost(ctx, "example.com", "ns1.example.com")
```

### Kontakte einer Domain

```go
err = client.Domains.SetContactHandles(ctx, "example.com", dna.ContactHandles{Admin: "DNA-2", Tech: "DNA-2"})
err = client.Domains.UpdateContacts(ctx, "example.com", dna.DomainContacts{Billing: &billing})

details, err := client.Contacts.Get(ctx, "DNA-1")           // der Kontakt und sein Verifizierungsstatus
result, err := client.Contacts.Update(ctx, "DNA-1", contact) // aktualisiert jede Domain, die das Handle verwendet
```

### DNS-Zone

```go
err = client.DNS.Add(ctx, dna.DNSAddRequest{
	DomainName: "example.com", Name: "www", Type: dna.RecordTypeA, TTL: 600, Values: []string{"192.0.2.10"},
})
err = client.DNS.Update(ctx, dna.DNSUpdateRequest{ // ersetzt alle Werte des Sets
	DomainName: "example.com", RecordName: "www", Type: dna.RecordTypeA, Values: []string{"192.0.2.20"},
})
err = client.DNS.Delete(ctx, dna.DNSDeleteRequest{
	DomainName: "example.com", Name: "www", Type: dna.RecordTypeA, Value: "192.0.2.20",
})

sets, err := client.DNS.List(ctx, "example.com")
for _, s := range sets {
	fmt.Println(s.Name, s.Type, s.TTL, s.Values())
}

res, err := client.DNS.Apply(ctx, "example.com") // stellt die Domain auf die Nameserver der Plattform um
```

Namen können relativ (`www`), der Apex (`""` oder `"@"`) oder vollqualifiziert sein. Record-Typen sind hier nicht
von der Groß-/Kleinschreibung abhängig (`"txt"` wird als `TXT` gesendet). Die Validierung fängt ab, was die API ablehnen würde: einen CNAME am Apex, einen CAA-Wert
ohne drei Teile, einen MX-Wert ohne Priorität. Die Testumgebung beantwortet jeden Zonenaufruf mit HTTP
500, Zonen-Code lässt sich daher nur in der Produktion ausprobieren.

### DNSSEC

```go
ds, err := dna.ParseDS("2371 13 2 1F987CC6583E92DF0890718C42...") // so, wie Ihr DNS-Anbieter ihn anzeigt
results, err := client.DNSSEC.Add(ctx, "example.com", ds)
info, err := client.DNSSEC.Get(ctx, "example.com") // info.Status, info.DSRecords
res, err := client.DNSSEC.Disable(ctx, "example.com")
```

### URL-Weiterleitung

```go
err = client.Domains.SetForwarding(ctx, "example.com", "https://www.example.org", dna.ForwardTypeStandard)
f, err := client.Domains.Forwarding(ctx, "example.com") // nil, wenn die Domain keine hat
err = client.Domains.DeleteForwarding(ctx, "example.com")
```

### Registry-Dokumente (.tr)

```go
doc, err := dna.DocumentFromFile(dna.DocumentTypeIdentityCard, "id.pdf")
res, err := client.Domains.UploadDocuments(ctx, "example.com.tr", doc)
page, err := client.Domains.Documents(ctx, dna.DomainDocumentsListRequest{DomainName: "example.com.tr"})
```

Dateien werden für Sie Base64-kodiert; Format (PDF, JPG, PNG, DOC, DOCX, ODT) und Größe (10 MB pro Datei, 50 MB pro
Request) werden vor dem Upload geprüft.

### SSL-Zertifikate

Eine Domain kann mehrere Zertifikate haben. Ein bestehendes Zertifikat wird deshalb über seine
**Zertifikats-ID** (eine GUID) angesprochen, nie über den Domainnamen. `Initialize` und `Renew` geben die ID zurück
(speichern Sie sie), und `SSL.List` findet sie anhand des Domainnamens. Ein Domainname an der Stelle einer
Zertifikats-ID wird mit einem `*dna.InputError` abgelehnt, bevor etwas gesendet wird.

```go
csr, err := client.SSL.GenerateCSR(ctx, dna.SSLGenerateCSRRequest{
	DomainName: "example.com", Country: "TR", State: "Istanbul", Locality: "Istanbul",
	Organization: "Example Ltd", OrganizationUnit: "IT",
})
order, err := client.SSL.Initialize(ctx, "example.com", "PositiveSSL") // belastet das Guthaben
id := order.CertificateID                                              // speichern: jeder weitere Aufruf erwartet sie

_, err = client.SSL.Configure(ctx, dna.SSLConfigureRequest{ // innerhalb von 15 Tagen
	CertificateID: id, CSR: csr.CSR, ApproverMail: "admin@example.com",
	AuthenticationType: dna.AuthenticationEmail, WebServerType: dna.WebServerApache, Contact: sslContact,
})
st, err := client.SSL.Status(ctx, id) // abfragen, bis st.IsIssued gilt
cert, err := client.SSL.Content(ctx, id)
os.WriteFile("example.com.crt", []byte(cert.Chain()), 0o644)
```

Ein Zertifikat, das nicht innerhalb von 15 Tagen nach `Initialize` oder `Renew` konfiguriert wird, wird storniert.

`SSL.List` liefert Ihre Zertifikate, die neuesten zuerst (`SSL.All` iteriert über alle). `SSL.Approvers` listet
die Adressen auf, die die E-Mail-Validierung akzeptiert, für ein bestehendes Zertifikat oder vor einem Kauf:

```go
page, err := client.SSL.List(ctx, dna.SSLListRequest{DomainName: "example.com"})
for _, c := range page.Items {
	fmt.Println(c.CertificateID, c.Status, c.SubscriptionExpiresAt)
}

mails, err := client.SSL.Approvers(ctx, dna.SSLApproversRequest{CertificateID: id})
mails, err = client.SSL.Approvers(ctx, dna.SSLApproversRequest{DomainName: "example.com", PackageCode: "PositiveSSL"})
```

Um ein Zertifikat zu verlängern, rufen Sie `Renew` auf und konfigurieren anschließend mit `Configure` das neue
Zertifikat, das zurückgegeben wird. `Renew` belastet das Guthaben und ist ab 30 Tagen vor Ablauf des Abonnements
möglich. Verlängern Sie nicht mit einem zweiten `Initialize`: Das ist ein unabhängiger Kauf zum Kaufpreis, und das
alte Zertifikat erhält nie den Status `Renewed`.

```go
renewal, err := client.SSL.Renew(ctx, id) // belastet das Guthaben
_, err = client.SSL.Configure(ctx, dna.SSLConfigureRequest{
	CertificateID: renewal.CertificateID, // das neue Zertifikat; das alte behält seine eigene ID
	CSR: newCSR.CSR, ApproverMail: "admin@example.com",
	AuthenticationType: dna.AuthenticationEmail, WebServerType: dna.WebServerApache, Contact: sslContact,
})
```

### Hosting

```go
packages, err := client.Hosting.Packages(ctx)
created, err := client.Hosting.Create(ctx, "example.com", "Linux-Starter") // belastet das Guthaben

ctx, cancel := context.WithTimeout(ctx, 30*time.Minute)
defer cancel()
st, err := client.Hosting.Wait(ctx, created.ServiceID) // fragt ab, bis das Hosting aktiv ist
fmt.Println(st.PanelAddress, st.Username, st.Password)
```

Rufen Sie `Create` nie erneut für ein Hosting auf, das noch bereitgestellt wird: Das belastet Ihr Guthaben doppelt.

### Guthaben, Preise und Bestellungen

```go
account, err := client.Account.Get(ctx)
usd, _ := account.Balance("USD")
profile, err := client.Account.Profile(ctx) // profile.Name, profile.AccountCurrency, profile.Status

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

info, err := client.Products.Info(ctx, "com", dna.OrderRenew, 2) // Preis einer 2-jährigen .com-Verlängerung
order, err := client.Orders.Get(ctx, "ORDER-CODE")
```

## Requests und BaseRequest

Jede API-Operation ist ein Request-Typ: `DomainInfoRequest`, `DNSAddRequest`, `SSLConfigureRequest`, ... Sie
implementieren alle ein Interface:

```go
type Request interface {
	Endpoint() Endpoint      // Methode, Pfad und Sicherheitsklasse
	Query() url.Values       // Query-String oder nil
	Body() any               // JSON-Body oder nil
	Validate() error         // wird geprüft, bevor etwas gesendet wird
}
```

Go kennt keine Vererbung, die „Basisklasse“ ist daher eine Struktur, die Sie einbetten. `dna.BaseRequest` liefert die Standardwerte (keine
Query, kein Body, nichts zu validieren); ein Request überschreibt eine Methode, indem er sie auf seinem eigenen Typ definiert. Nur
`Endpoint` hat keinen Standardwert. Ein Request aus diesem Paket sieht so aus:

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

Die Service-Methoden erstellen diese Requests für Sie. `client.Do` sendet jeden `Request`, auch eigene Typen, zum Beispiel,
um nur die benötigten Felder zu lesen oder einen Endpoint aufzurufen, den dieses Paket noch nicht abdeckt:

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

Ihr Request erhält alles, was die eingebauten erhalten: Zugangsdaten, Retries, Pacing, Fehlernormalisierung,
Correlation-IDs, Tracing und Read-only-Durchsetzung. `client.NewHTTPRequest(ctx, req)` baut den exakten HTTP-Request,
ohne ihn zu senden, um einen kostenpflichtigen Request vor dem Absenden zu protokollieren oder zu prüfen.

## Fehler

| Fehler                 | Wann                                                       | Nützliche Felder                                                 |
|------------------------|------------------------------------------------------------|------------------------------------------------------------------|
| `*dna.InputError`      | `Validate` hat den Request abgelehnt; **es wurde nichts gesendet** | `Request`, `Fields`                                      |
| `*dna.APIError`        | die API hat mit einem Nicht-2xx-Status geantwortet         | `StatusCode`, `Code`, `Message`, `Details`, `Fields`, `Data`, `CorrelationID`, `Hint()` |
| `*dna.OperationError`  | die API hat mit 2xx und `"success": false` geantwortet     | `Code`, `Message`, `CorrelationID`                               |
| `*dna.NetworkError`    | der Request wurde nicht abgeschlossen (DNS, TLS, Timeout, Reset) | `Err`, `Timeout()`                                         |
| `dna.ErrReadOnly`      | ein Read-only-Client hat eine Änderung blockiert; es wurde nichts gesendet | `errors.Is` verwenden                    |

`dna.KindOf(err)` klassifiziert jeden davon: `KindAuth`, `KindNotFound`, `KindValidation`, `KindRejected`,
`KindConflict`, `KindRateLimited`, `KindServer`, `KindNetwork`, `KindInvalidInput`, `KindReadOnly`. Helfer
beantworten die häufigen Fragen:

```go
_, err := client.Domains.Renew(ctx, "example.com", 1)
var apiErr *dna.APIError
switch {
case err == nil:
case dna.OutcomeUnknown(err):
	// Die Verlängerung ist möglicherweise durchgegangen. Prüfen Sie die Domain, bevor Sie es erneut versuchen.
case dna.IsInsufficientBalance(err):
case dna.IsNotFound(err):
case dna.HasCode(err, dna.CodePrivacyNotSupported):
case errors.As(err, &apiErr):
	log.Println(apiErr.Code, apiErr.Message, apiErr.Hint(), apiErr.CorrelationID)
}
```

Verzweigen Sie nach `Code`, nicht nach `Message`: Meldungen sind lokalisiert. API-Codes sehen so aus:
`Dna.DomainService:Domain:10007`; siehe die [Fehlerreferenz](https://www.domainnameapi.com/docs/reference/errors.md).
Nennen Sie die `CorrelationID`, wenn Sie den Support kontaktieren.

## Sicherheit und Zuverlässigkeit

- **Sicherheitsklassen.** Jeder Request ist `Read`, `Write`, `Destructive` oder `Billable` (`req.Endpoint().Safety`).
  Einige POST- und PUT-Requests sind Lesezugriffe (`DomainSearchRequest`, `SSLStatusRequest`).
- **Read-only-Modus.** `dna.WithReadOnly()` blockiert alles, was kein Lesezugriff ist, für Reporting-Jobs und um
  die Produktion gefahrlos zu erkunden.
- **Retries.** 429-Antworten werden immer wiederholt und berücksichtigen `Retry-After`: Die API weist sie ab, bevor sie
  etwas tut. Netzwerkfehler und 502/503/504 werden nur für Lesezugriffe und idempotente Schreibzugriffe wiederholt, sodass eine Registrierung
  oder Verlängerung nie zweimal gesendet wird. Prüfen Sie nach einem fehlgeschlagenen kostenpflichtigen Request `dna.OutcomeUnknown(err)`.
- **Pacing.** Der Client verteilt Requests zeitlich, um unter den Limits pro Schlüssel zu bleiben (5 Requests/Sekunde, 2 Suchen/Sekunde,
  ein Zonenaufruf pro Sekunde). Schalten Sie es mit `dna.WithPacing(false)` ab, wenn Sie selbst drosseln.
- **Quota.** `Response.Quota` stellt die Soft-Quota-Header bereit. Bei Überschreitung des Limits verlangsamt die API die Antworten (bis zu
  5 s), statt sie abzuweisen; das Standard-Timeout von 60 s lässt dafür Spielraum.
- **Correlation-IDs.** Jeder Request trägt einen `X-Correlation-ID`. Setzen Sie eine eigene mit
  `dna.WithCorrelationID(ctx, id)`, um API-Aufrufe mit Ihren Logs zu verknüpfen.
- **Observability.** `dna.WithLogger(slog.Default())` protokolliert jeden Austausch; `dna.WithTrace(func(dna.TraceEvent))`
  liefert Ihnen jeden Request, jede Response und jeden Retry, für Metriken oder Fehlerberichte. Der API-Schlüssel ist nie darin enthalten.
- **Nebenläufigkeit.** Ein `Client` ist nebenläufig sicher nutzbar. Erstellen Sie einen und teilen Sie ihn.
- **Tolerantes Dekodieren.** Datumsangaben ohne Zeitzone, `null`, `[]` statt `{}`, numerische oder benannte Status und
  Rollen in beliebiger Schreibweise (`"registrant"`, `"ADMINISTRATIVE"`) werden alle dekodiert.

## Client-Optionen

| Option                          | Wirkung                                                                                  |
|---------------------------------|------------------------------------------------------------------------------------------|
| `WithEnvironment(dna.OTE)`      | die Testumgebung verwenden (Standard `dna.Production`)                                   |
| `WithBaseURL(url)`              | Requests an einen Proxy oder einen Testserver senden                                     |
| `WithHTTPClient(hc)`            | Ihr eigener `*http.Client` (Transport, Proxy, TLS); Redirects werden nie verfolgt        |
| `WithTimeout(d)`                | Timeout pro Versuch (Standard 60 s)                                                      |
| `WithUserAgent("app/1.0")`      | Ihr Produkt dem User-Agent voranstellen                                                  |
| `WithApp("wisecp", "2.0.20")`   | Ihre Anwendung und deren Version nennen (`X-App`, `X-Version`)                           |
| `WithCurrency("TRY")`           | Währung der Such- und Katalogpreise                                                      |
| `WithRetries(n)`                | Anzahl der Retries (Standard 2)                                                          |
| `WithPacing(false)`             | clientseitiges Pacing abschalten                                                         |
| `WithReadOnly()`                | jeden Request blockieren, der kein Lesezugriff ist                                       |
| `WithLogger(l)`                 | Austausch in einen `*slog.Logger` protokollieren                                         |
| `WithTrace(fn)`                 | jeden Request, jede Response und jeden Retry beobachten                                  |

### Die eigene Anwendung angeben

Jeder Request trägt die Header `X-App` und `X-Version`, an denen DomainNameAPI erkennt, welche Integration ihn
gesendet hat. Standardmäßig sind das `go` und die SDK-Version. Nennen Sie Ihre eigene Anwendung mit `WithApp` (oder mit
`DNA_APP` und `DNA_APP_VERSION`, wenn Sie `NewClientFromEnv` verwenden):

```go
client, err := dna.NewClient(resellerID, apiKey, dna.WithApp("WiseCP", "V2.0.20-beta"))
// X-App: wisecp, X-Version: 2.0.20
```

Der Name wird kleingeschrieben und behält nur `a-z`, `0-9`, `.`, `_` und `-`; die Version behält nur Ziffern und
Punkte. Ein leerer oder ungültiger Name fällt auf `go` zurück, eine leere Version auf die SDK-Version. Der User-Agent
bleibt unverändert.

## Den eigenen Code testen

Richten Sie den Client auf einen `httptest.Server`; sonst muss nichts gemockt werden:

```go
srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, `{"success":true,"info":{"domainName":"example.com","status":"AVAILABLE","price":11.99,"currency":"USD"}}`)
}))
defer srv.Close()

client, _ := dna.NewClient("reseller-id", "api-key", dna.WithBaseURL(srv.URL), dna.WithPacing(false))
a, err := client.Domains.Search(ctx, "example.com")
```

## Migration von v1

v2 ist ein neues Design: typisierte Ergebnisse statt `map[string]any`, Fehler statt `{"result": "ERROR"}`-
Umschläge und die vollständige API statt des Methodensatzes von php-dna. Der Modulpfad hat sich zu
`github.com/domainreseller/go-dna/v2` geändert, v1 funktioniert also weiter, bis Sie umsteigen.

| v1                                                  | v2                                                                     |
|-----------------------------------------------------|------------------------------------------------------------------------|
| `dna.New(id, key, true)`                            | `dna.NewClient(id, key, dna.WithEnvironment(dna.OTE))`                 |
| `res["result"] == "OK"`                             | `err == nil`                                                           |
| `GetResellerDetails`, `GetCurrentBalance`           | `Account.Get` (+ `Account.Balance("USD")`)                             |
| `CheckAvailability`                                 | `Domains.Search`, `Domains.BulkSearch`                                 |
| `GetList`                                           | `Domains.List`, `Domains.All`                                          |
| `GetTldList`                                        | `Products.TLDs`, `Products.AllTLDs`, `Products.TLD`                    |
| `GetDetails`, `GetContacts`, `SyncFromRegistry`     | `Domains.Info`                                                         |
| `RegisterWithContactInfo`                           | `Domains.RegisterWithContacts` (oder `Contacts.Create` + `Domains.Register`) |
| `Renew`, `Transfer`, `CheckTransfer`                | `Domains.Renew`, `Domains.Transfer`, `Domains.TransferCheck`           |
| `ModifyNameServer`                                  | `Domains.SetNameServers`                                               |
| `EnableTheftProtectionLock` / `Disable...`          | `Domains.Lock` / `Domains.Unlock`                                      |
| `ModifyPrivacyProtectionStatus`                     | `Domains.SetPrivacy`                                                   |
| `SaveContacts`                                      | `Domains.UpdateContacts`, `Domains.SetContactHandles`                  |
| `AddChildNameServer`, `ModifyChildNameServer`, `DeleteChildNameServer` | `Domains.AddHost`, `Domains.UpdateHost`, `Domains.DeleteHost` |
| `GetResourceRecords`                                | `DNS.List`                                                             |
| `AddResourceRecord`, `AddResourceRecordValue`       | `DNS.Add`                                                              |
| `EditResourceRecord`, `SetResourceRecordSet`, `ReplaceResourceRecordValue` | `DNS.Update`                                    |
| `DeleteResourceRecord`, `RemoveResourceRecordValue`, `DeleteResourceRecordSet` | `DNS.Delete` (einmal pro Wert)              |
| `ApplyResourceRecords`                              | `DNS.Apply`                                                            |
| `GetForwarding`, `SetForwarding`, `DeleteForwarding`| `Domains.Forwarding`, `Domains.SetForwarding`, `Domains.DeleteForwarding` |
| `CancelTransfer`, `ApproveTransfer`, `RejectTransfer` | nicht in der veröffentlichten API; senden Sie einen [eigenen Request](#requests-und-baserequest), falls Ihr Konto sie hat |
| `IsTrTLD`, `ValidateContact`                        | `strings.HasSuffix(name, ".tr")`; die Validierung läuft bei jedem Request |
| `GetRequestData`, `GetResponseData`, ...            | `WithTrace`, `WithLogger`, `Response` von `client.Do`                  |
| telemetry (`SetErrorReportingEnabled`, Sentry)      | entfernt: nichts verlässt Ihren Prozess; leiten Sie `WithTrace`-Events an Ihr eigenes Reporting weiter |
| `SetSSLVerification(false)`                         | `WithHTTPClient` mit Ihren eigenen TLS-Einstellungen                   |

## Entwicklung

```bash
go test ./...          # Unit-Tests; keiner greift auf das Netzwerk zu
go vet ./...
gofmt -l .
```

`testdata/openapi.json` ist das OpenAPI-Dokument der API. Die Tests prüfen jeden Request-Typ dagegen (Methode,
Pfad, Query-Namen, Body-Felder, Pflichtfelder), die JSON-Namen jedes Response-Typs gegen die Response-
Schemas sowie jeden Enum-Wert und schlagen fehl, wenn die API eine Operation ohne Request-Typ erhält. So aktualisieren Sie es:

```bash
curl -fsSL -A "go-dna-spec-update" -o testdata/openapi.json https://api.domainresellerapi.com/swagger/v1/swagger.json
go test ./...
```

Die Konventionen dieses Repositorys finden Sie in [AGENTS.md](AGENTS.md).

## Support

- API-Dokumentation: <https://www.domainnameapi.com/docs/>
- CLI und MCP-Server: <https://github.com/domainreseller/cli>
- Website: <https://www.domainnameapi.com>

## Lizenz

Apache License 2.0. Siehe [LICENSE](LICENSE).
