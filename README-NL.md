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

# go-dna: de Go SDK voor de Domain Name API

[![Go Reference](https://pkg.go.dev/badge/github.com/domainreseller/go-dna/v2.svg)](https://pkg.go.dev/github.com/domainreseller/go-dna/v2)
[![CI](https://github.com/domainreseller/go-dna/actions/workflows/ci.yml/badge.svg)](https://github.com/domainreseller/go-dna/actions/workflows/ci.yml)
[![License](https://img.shields.io/badge/license-Apache--2.0-blue.svg)](LICENSE)

Getypeerde Go-client voor de reseller-REST-API van de [Domain Name API](https://www.domainnameapi.com): domeinen
zoeken, registreren, verlengen en verhuizen, DNS, DNSSEC, contacten, SSL-certificaten, hosting, bestellingen, de
productcatalogus en je saldo.

- **Alle 67 API-operaties**, op dezelfde manier gegroepeerd als in de [`dna` CLI](https://github.com/domainreseller/cli):
  `dna domain info` is `client.Domains.Info`, `dna dns add` is `client.DNS.Add`.
- **Getypeerde resultaten en enums**: `d.Status == dna.DomainStatusPendingDelete`, `d.ExpirationDate.Before(...)`,
  geen `map[string]any`.
- **Elke operatie is een request-type** dat `dna.BaseRequest` embedt en zichzelf valideert voordat er iets
  wordt verstuurd. Eigen requests schrijf je op dezelfde manier.
- **Standaard veilig**: kostenplichtige requests worden nooit blind herhaald, `WithReadOnly` blokkeert
  wijzigingen, de client remt zichzelf af onder de rate limits en is veilig voor gelijktijdig gebruik.
- **Alleen de standaardbibliotheek.** Go 1.23 of nieuwer.

```go
client, _ := dna.NewClient(resellerID, apiKey, dna.WithEnvironment(dna.OTE))

a, err := client.Domains.Search(ctx, "example.com")
if err == nil && a.Available() {
	fmt.Printf("%s is available for %.2f %s\n", a.DomainName, a.Price, a.Currency)
}
```

## Inhoud

- [Installatie](#installatie)
- [Inloggegevens en omgevingen](#inloggegevens-en-omgevingen)
- [Services](#services)
- [Gebruik](#gebruik)
- [Requests en BaseRequest](#requests-en-baserequest)
- [Fouten](#fouten)
- [Veiligheid en betrouwbaarheid](#veiligheid-en-betrouwbaarheid)
- [Client-opties](#client-opties)
- [Je code testen](#je-code-testen)
- [Migreren van v1](#migreren-van-v1)
- [Ontwikkeling](#ontwikkeling)

## Installatie

```bash
go get github.com/domainreseller/go-dna/v2
```

```go
import dna "github.com/domainreseller/go-dna/v2"
```

Uitvoerbare programma's staan in [examples](examples): `quickstart`, `register`, `expiring`, `dns`, `ssl` en
`custom-request`.

```bash
go run ./examples/quickstart example.com
```

## Inloggegevens en omgevingen

Je hebt twee API-inloggegevens nodig uit het resellerpaneel (**API settings**). De inloggegevens van het paneel werken niet.

| Inloggegevens | Ziet eruit als                         |
|---------------|----------------------------------------|
| Reseller ID   | `xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx` |
| API key       | een lange geheime tekenreeks           |

Er zijn twee omgevingen. **Begin met OTE**, de testomgeving: die heeft eigen gegevens en een eigen API-sleutel,
en niets wat je daar doet kost geld of raakt live domeinen.

```go
// Testomgeving (OTE)
client, err := dna.NewClient(resellerID, oteAPIKey, dna.WithEnvironment(dna.OTE))

// Productie (de standaard): echte registraties, echte afschrijvingen
client, err := dna.NewClient(resellerID, apiKey)
```

`NewClientFromEnv` leest dezelfde variabelen als de `dna` CLI:

| Variabele         | Betekenis                                                     |
|-------------------|---------------------------------------------------------------|
| `DNA_RESELLER_ID` | reseller ID                                                   |
| `DNA_API_KEY`     | API-sleutel van de gekozen omgeving                           |
| `DNA_ENV`         | `production` (standaard) of `ote`                             |
| `DNA_TEST_MODE`   | `true` kiest OTE (zoals in de andere DomainNameAPI-SDK's)     |
| `DNA_BASE_URL`    | een eigen basis-URL, bijv. een proxy                          |
| `DNA_APP`         | de applicatie die de SDK gebruikt, verstuurd als `X-App`      |
| `DNA_APP_VERSION` | de versie van die applicatie, verstuurd als `X-Version`       |

```go
client, err := dna.NewClientFromEnv(dna.WithUserAgent("my-panel/1.4"))
```

Een verkeerde sleutel of omgeving mislukt bij de eerste request met een `*APIError` van het soort `KindAuth` en een hint;
`client.Account.Get(ctx)` is de goedkoopste manier om dit te controleren.

## Services

| Service           | CLI-groep     | Operaties                                                                                                                                                         |
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

Methoden nemen de vereiste invoer als argumenten (`Info(ctx, "example.com")`) of, wanneer een operatie optionele
invoer heeft, een request-struct (`Register(ctx, dna.DomainRegisterRequest{...})`). De volledige referentie met alle velden
staat op [pkg.go.dev](https://pkg.go.dev/github.com/domainreseller/go-dna/v2).

## Gebruik

Alle snippets gaan uit van `ctx := context.Background()` en een `client`.

### Zoeken

```go
a, err := client.Domains.Search(ctx, "example.com")
// a.Available(), a.Price, a.Currency, a.IsPremium, a.IsDocumentRequired, a.Reason

results, err := client.Domains.BulkSearch(ctx, "example.com", "example.net", "example.org")
for _, r := range results {
	fmt.Println(r.DomainName, r.Status, r.Price)
}
```

`BulkSearch` splitst lijsten met meer dan 30 domeinen op in meerdere requests. Prijzen gebruiken de valuta van het account of
de valuta die is ingesteld met `dna.WithCurrency("TRY")`.

### Registreren

Met contact-handles (maak er een aan met `Contacts.Create`):

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
	Handles:     dna.SameHandle(handle), // of dna.ContactHandles{Registrant: "DNA-1", Admin: "DNA-2", ...}
})
```

Met inline contacten:

```go
res, err := client.Domains.RegisterWithContacts(ctx, dna.DomainRegisterWithContactsRequest{
	DomainName:  "example.com",
	NameServers: []string{"ns1.example.net", "ns2.example.net"},
	Contacts:    dna.SameContact(owner), // of dna.DomainContacts{Registrant: &owner, Admin: &admin, ...}
})
```

Contacten worden vóór het versturen opgeschoond: velden worden getrimd, het land wordt omgezet naar hoofdletters en telefoonvelden behouden alleen
hun cijfers (`"+90 (555) 123-45-67"` werkt). Sommige TLD's vereisen registry-attributen: je vindt ze met
`client.Products.TLD(ctx, "com.tr")` en de `RequiredAttributes()` ervan, en geeft ze mee als `TLDAttributes`.

Als de API antwoordt maar weigert (`"success": false`), is de fout een `*dna.OperationError` met de melding
van de API.

### Verlengen, verhuizen, herstellen

```go
check, err := client.Domains.RenewCheck(ctx, "example.com", 1) // check.Renewable, check.PremiumPrice
res, err := client.Domains.Renew(ctx, "example.com", 1)

tc, err := client.Domains.TransferCheck(ctx, "example.com", "auth-code") // tc.Transferable, tc.AuthCodeValid
res, err := client.Domains.Transfer(ctx, dna.DomainTransferRequest{DomainName: "example.com", AuthCode: "auth-code"})
st, err := client.Domains.TransferStatus(ctx, "example.com")

err = client.Domains.Restore(ctx, "example.com") // tijdens de redemption-periode
```

### Domeindetails en de domeinlijst

```go
d, err := client.Domains.Info(ctx, "example.com")
fmt.Println(d.Status, d.ExpirationDate, d.RemainingDays, d.Locked, d.NameServers, d.AuthCode)
if owner, ok := d.Contact(dna.ContactTypeRegistrant); ok {
	fmt.Println(owner.Email)
}

// Eén pagina
page, err := client.Domains.List(ctx, dna.DomainListRequest{ListOptions: dna.ListOptions{Limit: 50}})

// Elk domein; de pagina's worden voor je opgehaald
for d, err := range client.Domains.All(ctx, dna.DomainListRequest{
	Statuses: []dna.DomainStatus{dna.DomainStatusActive},
}) {
	if err != nil {
		return err
	}
	fmt.Println(d.DomainName, d.ExpirationDate)
}
```

### Vergrendeling, privacy, nameservers, child-nameservers

```go
err = client.Domains.Lock(ctx, "example.com")
err = client.Domains.Unlock(ctx, "example.com")
err = client.Domains.SetPrivacy(ctx, "example.com", true)
code, err := client.Domains.UpdateAuthCode(ctx, "example.com")

err = client.Domains.SetNameServers(ctx, dna.DomainNameServersSetRequest{
	DomainName:  "example.com",
	NameServers: []string{"ns1.example.net", "ns2.example.net"},
	// DNSSECMode: dna.DNSSECModeDisable, // alleen voor DNSSEC-getekende domeinen
})

err = client.Domains.AddHost(ctx, "example.com", "ns1.example.com", "192.0.2.10", "2001:db8::10") // IP-versie wordt herkend
err = client.Domains.DeleteHost(ctx, "example.com", "ns1.example.com")
```

### Contacten van een domein

```go
err = client.Domains.SetContactHandles(ctx, "example.com", dna.ContactHandles{Admin: "DNA-2", Tech: "DNA-2"})
err = client.Domains.UpdateContacts(ctx, "example.com", dna.DomainContacts{Billing: &billing})

details, err := client.Contacts.Get(ctx, "DNA-1")           // het contact en de verificatiestatus ervan
result, err := client.Contacts.Update(ctx, "DNA-1", contact) // werkt elk domein bij dat de handle gebruikt
```

### DNS-zone

```go
err = client.DNS.Add(ctx, dna.DNSAddRequest{
	DomainName: "example.com", Name: "www", Type: dna.RecordTypeA, TTL: 600, Values: []string{"192.0.2.10"},
})
err = client.DNS.Update(ctx, dna.DNSUpdateRequest{ // vervangt alle waarden van de set
	DomainName: "example.com", RecordName: "www", Type: dna.RecordTypeA, Values: []string{"192.0.2.20"},
})
err = client.DNS.Delete(ctx, dna.DNSDeleteRequest{
	DomainName: "example.com", Name: "www", Type: dna.RecordTypeA, Value: "192.0.2.20",
})

sets, err := client.DNS.List(ctx, "example.com")
for _, s := range sets {
	fmt.Println(s.Name, s.Type, s.TTL, s.Values())
}

res, err := client.DNS.Apply(ctx, "example.com") // zet het domein om naar de nameservers van het platform
```

Namen kunnen relatief (`www`), de apex (`""` of `"@"`) of volledig gekwalificeerd zijn. Recordtypen zijn hier
niet hoofdlettergevoelig (`"txt"` wordt verstuurd als `TXT`). De validatie vangt af wat de API zou weigeren: een CNAME op de apex, een CAA-waarde
zonder drie delen, een MX-waarde zonder prioriteit. De testomgeving beantwoordt elke zone-aanroep met HTTP
500, dus zonecode kan alleen in productie worden uitgeprobeerd.

### DNSSEC

```go
ds, err := dna.ParseDS("2371 13 2 1F987CC6583E92DF0890718C42...") // zoals je DNS-provider het toont
results, err := client.DNSSEC.Add(ctx, "example.com", ds)
info, err := client.DNSSEC.Get(ctx, "example.com") // info.Status, info.DSRecords
res, err := client.DNSSEC.Disable(ctx, "example.com")
```

### URL-doorstuurfunctie

```go
err = client.Domains.SetForwarding(ctx, "example.com", "https://www.example.org", dna.ForwardTypeStandard)
f, err := client.Domains.Forwarding(ctx, "example.com") // nil als het domein er geen heeft
err = client.Domains.DeleteForwarding(ctx, "example.com")
```

### Registrydocumenten (.tr)

```go
doc, err := dna.DocumentFromFile(dna.DocumentTypeIdentityCard, "id.pdf")
res, err := client.Domains.UploadDocuments(ctx, "example.com.tr", doc)
page, err := client.Domains.Documents(ctx, dna.DomainDocumentsListRequest{DomainName: "example.com.tr"})
```

Bestanden worden voor je base64-gecodeerd; formaat (PDF, JPG, PNG, DOC, DOCX, ODT) en grootte (10 MB per bestand, 50 MB per
request) worden vóór het uploaden gecontroleerd.

### SSL-certificaten

Een domein kan meerdere certificaten hebben. Een bestaand certificaat wordt daarom aangeduid met zijn
**certificaat-ID** (een GUID), nooit met de domeinnaam. `Initialize` en `Renew` geven het ID terug (bewaar het) en
`SSL.List` zoekt het op aan de hand van de domeinnaam. Een domeinnaam op de plaats van een certificaat-ID wordt
geweigerd met een `*dna.InputError` voordat er iets wordt verzonden.

```go
csr, err := client.SSL.GenerateCSR(ctx, dna.SSLGenerateCSRRequest{
	DomainName: "example.com", Country: "TR", State: "Istanbul", Locality: "Istanbul",
	Organization: "Example Ltd", OrganizationUnit: "IT",
})
order, err := client.SSL.Initialize(ctx, "example.com", "PositiveSSL") // schrijft af van het saldo
id := order.CertificateID                                              // bewaar het: elke volgende aanroep heeft het nodig

_, err = client.SSL.Configure(ctx, dna.SSLConfigureRequest{ // binnen 15 dagen
	CertificateID: id, CSR: csr.CSR, ApproverMail: "admin@example.com",
	AuthenticationType: dna.AuthenticationEmail, WebServerType: dna.WebServerApache, Contact: sslContact,
})
st, err := client.SSL.Status(ctx, id) // blijf pollen tot st.IsIssued
cert, err := client.SSL.Content(ctx, id)
os.WriteFile("example.com.crt", []byte(cert.Chain()), 0o644)
```

Een certificaat dat niet binnen 15 dagen na `Initialize` of `Renew` is geconfigureerd, wordt geannuleerd.

`SSL.List` geeft je certificaten terug, de nieuwste eerst (`SSL.All` loopt ze allemaal af). `SSL.Approvers`
geeft de adressen die e-mailvalidatie accepteert, voor een bestaand certificaat of vóór een aankoop:

```go
page, err := client.SSL.List(ctx, dna.SSLListRequest{DomainName: "example.com"})
for _, c := range page.Items {
	fmt.Println(c.CertificateID, c.Status, c.SubscriptionExpiresAt)
}

mails, err := client.SSL.Approvers(ctx, dna.SSLApproversRequest{CertificateID: id})
mails, err = client.SSL.Approvers(ctx, dna.SSLApproversRequest{DomainName: "example.com", PackageCode: "PositiveSSL"})
```

Om een certificaat te verlengen roep je `Renew` aan en configureer je daarna met `Configure` het nieuwe
certificaat dat je terugkrijgt. `Renew` schrijft af van het saldo en kan vanaf 30 dagen voor het einde van het
abonnement. Verleng niet met een tweede `Initialize`: dat is een losstaande aankoop tegen de aankoopprijs, en het
oude certificaat krijgt nooit de status `Renewed`.

```go
renewal, err := client.SSL.Renew(ctx, id) // schrijft af van het saldo
_, err = client.SSL.Configure(ctx, dna.SSLConfigureRequest{
	CertificateID: renewal.CertificateID, // het nieuwe certificaat; het oude behoudt zijn eigen ID
	CSR: newCSR.CSR, ApproverMail: "admin@example.com",
	AuthenticationType: dna.AuthenticationEmail, WebServerType: dna.WebServerApache, Contact: sslContact,
})
```

### Hosting

```go
packages, err := client.Hosting.Packages(ctx)
created, err := client.Hosting.Create(ctx, "example.com", "Linux-Starter") // schrijft af van het saldo

ctx, cancel := context.WithTimeout(ctx, 30*time.Minute)
defer cancel()
st, err := client.Hosting.Wait(ctx, created.ServiceID) // pollt tot de hosting actief is
fmt.Println(st.PanelAddress, st.Username, st.Password)
```

Roep `Create` nooit opnieuw aan voor een hosting die nog wordt ingericht: dat schrijft twee keer af.

### Saldo, prijzen en bestellingen

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

info, err := client.Products.Info(ctx, "com", dna.OrderRenew, 2) // prijs van een .com-verlenging van 2 jaar
order, err := client.Orders.Get(ctx, "ORDER-CODE")
```

## Requests en BaseRequest

Elke API-operatie is een request-type: `DomainInfoRequest`, `DNSAddRequest`, `SSLConfigureRequest`, ... Ze
implementeren allemaal dezelfde interface:

```go
type Request interface {
	Endpoint() Endpoint      // methode, pad en veiligheidsklasse
	Query() url.Values       // query string, of nil
	Body() any               // JSON-body, of nil
	Validate() error         // gecontroleerd voordat er iets wordt verstuurd
}
```

Go kent geen overerving, dus de "basisklasse" is een struct die je embedt. `dna.BaseRequest` levert de standaardwaarden (geen
query, geen body, niets te valideren); een request overschrijft een methode door die op zijn eigen type te definiëren. Alleen
`Endpoint` heeft geen standaardwaarde. Een request uit dit pakket ziet er zo uit:

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

De servicemethoden bouwen deze requests voor je. `client.Do` verstuurt elke `Request`, ook eigen types, bijvoorbeeld
om alleen de velden te lezen die je nodig hebt of om een endpoint aan te roepen dat dit pakket nog niet omvat:

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

Je request krijgt alles wat de ingebouwde requests krijgen: inloggegevens, retries, pacing, foutnormalisatie,
correlation-ID's, tracing en read-only-afdwinging. `client.NewHTTPRequest(ctx, req)` bouwt de exacte HTTP-request
zonder die te versturen, om een kostenplichtige request te loggen of te controleren voordat hij de deur uit gaat.

## Fouten

| Fout                   | Wanneer                                                    | Nuttige velden                                                   |
|------------------------|------------------------------------------------------------|------------------------------------------------------------------|
| `*dna.InputError`      | `Validate` heeft de request afgewezen; **er is niets verstuurd** | `Request`, `Fields`                                        |
| `*dna.APIError`        | de API antwoordde met een niet-2xx-status                  | `StatusCode`, `Code`, `Message`, `Details`, `Fields`, `Data`, `CorrelationID`, `Hint()` |
| `*dna.OperationError`  | de API antwoordde 2xx met `"success": false`               | `Code`, `Message`, `CorrelationID`                               |
| `*dna.NetworkError`    | de request is niet voltooid (DNS, TLS, timeout, reset)     | `Err`, `Timeout()`                                               |
| `dna.ErrReadOnly`      | een read-only-client blokkeerde een wijziging; er is niets verstuurd | gebruik `errors.Is`                                    |

`dna.KindOf(err)` classificeert elk daarvan: `KindAuth`, `KindNotFound`, `KindValidation`, `KindRejected`,
`KindConflict`, `KindRateLimited`, `KindServer`, `KindNetwork`, `KindInvalidInput`, `KindReadOnly`. Hulpfuncties
beantwoorden de gebruikelijke vragen:

```go
_, err := client.Domains.Renew(ctx, "example.com", 1)
var apiErr *dna.APIError
switch {
case err == nil:
case dna.OutcomeUnknown(err):
	// De verlenging is mogelijk doorgegaan. Controleer het domein voordat je het opnieuw probeert.
case dna.IsInsufficientBalance(err):
case dna.IsNotFound(err):
case dna.HasCode(err, dna.CodePrivacyNotSupported):
case errors.As(err, &apiErr):
	log.Println(apiErr.Code, apiErr.Message, apiErr.Hint(), apiErr.CorrelationID)
}
```

Vertak op `Code`, niet op `Message`: meldingen zijn gelokaliseerd. API-codes zien eruit als
`Dna.DomainService:Domain:10007`; zie de [foutenreferentie](https://www.domainnameapi.com/docs/reference/errors.md).
Vermeld de `CorrelationID` wanneer je contact opneemt met de support.

## Veiligheid en betrouwbaarheid

- **Veiligheidsklassen.** Elke request is `Read`, `Write`, `Destructive` of `Billable` (`req.Endpoint().Safety`).
  Sommige POST- en PUT-requests zijn leesacties (`DomainSearchRequest`, `SSLStatusRequest`).
- **Read-only-modus.** `dna.WithReadOnly()` blokkeert alles wat geen leesactie is, voor rapportagejobs en om
  productie veilig te verkennen.
- **Retries.** 429-responses worden altijd herhaald, met inachtneming van `Retry-After`: de API weigert ze voordat hij
  iets doet. Netwerkfouten en 502/503/504 worden alleen herhaald voor leesacties en idempotente schrijfacties, zodat een registratie
  of verlenging nooit twee keer wordt verstuurd. Controleer na een mislukte kostenplichtige request `dna.OutcomeUnknown(err)`.
- **Pacing.** De client spreidt requests om onder de limieten per sleutel te blijven (5 requests/seconde, 2 zoekopdrachten/seconde,
  één zone-aanroep per seconde). Zet het uit met `dna.WithPacing(false)` als je zelf afremt.
- **Quota.** `Response.Quota` geeft de soft-quota-headers vrij. Boven de limiet vertraagt de API responses (tot
  5 s) in plaats van ze te weigeren; de standaard-timeout van 60 s laat daar ruimte voor.
- **Correlation-ID's.** Elke request draagt een `X-Correlation-ID`. Stel er zelf een in met
  `dna.WithCorrelationID(ctx, id)` om API-aanroepen aan je logs te koppelen.
- **Observability.** `dna.WithLogger(slog.Default())` logt elke uitwisseling; `dna.WithTrace(func(dna.TraceEvent))`
  geeft je elke request, response en retry, voor metrics of foutrapportage. De API-sleutel zit er nooit in.
- **Gelijktijdigheid.** Een `Client` is veilig voor gelijktijdig gebruik. Maak er één aan en deel hem.
- **Tolerant decoderen.** Datums zonder tijdzone, `null`, `[]` in plaats van `{}`, numerieke of benoemde statussen en
  rollen in elke schrijfwijze (`"registrant"`, `"ADMINISTRATIVE"`) worden allemaal gedecodeerd.

## Client-opties

| Optie                           | Effect                                                                                   |
|---------------------------------|------------------------------------------------------------------------------------------|
| `WithEnvironment(dna.OTE)`      | gebruik de testomgeving (standaard `dna.Production`)                                     |
| `WithBaseURL(url)`              | verstuur requests naar een proxy of een testserver                                       |
| `WithHTTPClient(hc)`            | je eigen `*http.Client` (transport, proxy, TLS); redirects worden nooit gevolgd          |
| `WithTimeout(d)`                | timeout per poging (standaard 60 s)                                                      |
| `WithUserAgent("app/1.0")`      | zet je product vóór de User-Agent                                                        |
| `WithApp("wisecp", "2.0.20")`   | noem je applicatie en haar versie (`X-App`, `X-Version`)                                 |
| `WithCurrency("TRY")`           | valuta van zoek- en catalogusprijzen                                                     |
| `WithRetries(n)`                | aantal retries (standaard 2)                                                             |
| `WithPacing(false)`             | zet pacing aan de clientzijde uit                                                        |
| `WithReadOnly()`                | blokkeer elke request die geen leesactie is                                              |
| `WithLogger(l)`                 | log uitwisselingen naar een `*slog.Logger`                                               |
| `WithTrace(fn)`                 | observeer elke request, response en retry                                                |

### Je applicatie identificeren

Elke request bevat de headers `X-App` en `X-Version`, waaraan DomainNameAPI ziet welke integratie hem verstuurde.
Standaard zijn dat `go` en de SDK-versie. Noem je eigen applicatie met `WithApp` (of met `DNA_APP` en
`DNA_APP_VERSION` als je `NewClientFromEnv` gebruikt):

```go
client, err := dna.NewClient(resellerID, apiKey, dna.WithApp("WiseCP", "V2.0.20-beta"))
// X-App: wisecp, X-Version: 2.0.20
```

De naam wordt in kleine letters gezet en houdt alleen `a-z`, `0-9`, `.`, `_` en `-` over; de versie houdt alleen
cijfers en punten over. Een lege of ongeldige naam valt terug op `go`, een lege versie op de SDK-versie. De User-Agent
verandert niet.

## Je code testen

Richt de client op een `httptest.Server`; verder hoeft er niets te worden gemockt:

```go
srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, `{"success":true,"info":{"domainName":"example.com","status":"AVAILABLE","price":11.99,"currency":"USD"}}`)
}))
defer srv.Close()

client, _ := dna.NewClient("reseller-id", "api-key", dna.WithBaseURL(srv.URL), dna.WithPacing(false))
a, err := client.Domains.Search(ctx, "example.com")
```

## Migreren van v1

v2 is een nieuw ontwerp: getypeerde resultaten in plaats van `map[string]any`, fouten in plaats van
`{"result": "ERROR"}`-enveloppen en de volledige API in plaats van de methodenset van php-dna. Het modulepad is gewijzigd in
`github.com/domainreseller/go-dna/v2`, dus v1 blijft werken tot je overstapt.

| v1                                                  | v2                                                                     |
|-----------------------------------------------------|------------------------------------------------------------------------|
| `dna.New(id, key, true)`                            | `dna.NewClient(id, key, dna.WithEnvironment(dna.OTE))`                 |
| `res["result"] == "OK"`                             | `err == nil`                                                           |
| `GetResellerDetails`, `GetCurrentBalance`           | `Account.Get` (+ `Account.Balance("USD")`)                             |
| `CheckAvailability`                                 | `Domains.Search`, `Domains.BulkSearch`                                 |
| `GetList`                                           | `Domains.List`, `Domains.All`                                          |
| `GetTldList`                                        | `Products.TLDs`, `Products.AllTLDs`, `Products.TLD`                    |
| `GetDetails`, `GetContacts`, `SyncFromRegistry`     | `Domains.Info`                                                         |
| `RegisterWithContactInfo`                           | `Domains.RegisterWithContacts` (of `Contacts.Create` + `Domains.Register`) |
| `Renew`, `Transfer`, `CheckTransfer`                | `Domains.Renew`, `Domains.Transfer`, `Domains.TransferCheck`           |
| `ModifyNameServer`                                  | `Domains.SetNameServers`                                               |
| `EnableTheftProtectionLock` / `Disable...`          | `Domains.Lock` / `Domains.Unlock`                                      |
| `ModifyPrivacyProtectionStatus`                     | `Domains.SetPrivacy`                                                   |
| `SaveContacts`                                      | `Domains.UpdateContacts`, `Domains.SetContactHandles`                  |
| `AddChildNameServer`, `ModifyChildNameServer`, `DeleteChildNameServer` | `Domains.AddHost`, `Domains.UpdateHost`, `Domains.DeleteHost` |
| `GetResourceRecords`                                | `DNS.List`                                                             |
| `AddResourceRecord`, `AddResourceRecordValue`       | `DNS.Add`                                                              |
| `EditResourceRecord`, `SetResourceRecordSet`, `ReplaceResourceRecordValue` | `DNS.Update`                                    |
| `DeleteResourceRecord`, `RemoveResourceRecordValue`, `DeleteResourceRecordSet` | `DNS.Delete` (eenmaal per waarde)           |
| `ApplyResourceRecords`                              | `DNS.Apply`                                                            |
| `GetForwarding`, `SetForwarding`, `DeleteForwarding`| `Domains.Forwarding`, `Domains.SetForwarding`, `Domains.DeleteForwarding` |
| `CancelTransfer`, `ApproveTransfer`, `RejectTransfer` | niet in de gepubliceerde API; verstuur een [eigen request](#requests-en-baserequest) als je account ze heeft |
| `IsTrTLD`, `ValidateContact`                        | `strings.HasSuffix(name, ".tr")`; validatie draait bij elke request    |
| `GetRequestData`, `GetResponseData`, ...            | `WithTrace`, `WithLogger`, `Response` van `client.Do`                  |
| telemetry (`SetErrorReportingEnabled`, Sentry)      | verwijderd: niets verlaat je proces; stuur `WithTrace`-events door naar je eigen rapportage |
| `SetSSLVerification(false)`                         | `WithHTTPClient` met je eigen TLS-instellingen                         |

## Ontwikkeling

```bash
go test ./...          # unittests; geen enkele raakt het netwerk
go vet ./...
gofmt -l .
```

`testdata/openapi.json` is het OpenAPI-document van de API. De tests controleren elk request-type ertegen (methode,
pad, querynamen, body-velden, verplichte velden), de JSON-namen van elk response-type tegen de response-
schema's en elke enum-waarde, en falen als de API een operatie krijgt zonder request-type. Zo werk je het bij:

```bash
curl -fsSL -A "go-dna-spec-update" -o testdata/openapi.json https://api.domainresellerapi.com/swagger/v1/swagger.json
go test ./...
```

Zie [AGENTS.md](AGENTS.md) voor de conventies van deze repository.

## Support

- API-documentatie: <https://www.domainnameapi.com/docs/>
- CLI en MCP-server: <https://github.com/domainreseller/cli>
- Website: <https://www.domainnameapi.com>

## Licentie

Apache License 2.0. Zie [LICENSE](LICENSE).
