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

# go-dna: l'SDK Go per la Domain Name API

[![Go Reference](https://pkg.go.dev/badge/github.com/domainreseller/go-dna/v2.svg)](https://pkg.go.dev/github.com/domainreseller/go-dna/v2)
[![CI](https://github.com/domainreseller/go-dna/actions/workflows/ci.yml/badge.svg)](https://github.com/domainreseller/go-dna/actions/workflows/ci.yml)
[![License](https://img.shields.io/badge/license-Apache--2.0-blue.svg)](LICENSE)

Client Go tipizzato per l'API REST per rivenditori di [Domain Name API](https://www.domainnameapi.com): ricerca
di domini, registrazione, rinnovo e trasferimento, DNS, DNSSEC, contatti, certificati SSL, hosting, ordini,
catalogo prodotti e saldo.

- **Tutte le 67 operazioni dell'API**, raggruppate come nella [CLI `dna`](https://github.com/domainreseller/cli):
  `dna domain info` corrisponde a `client.Domains.Info`, `dna dns add` a `client.DNS.Add`.
- **Risultati ed enumerazioni tipizzati**: `d.Status == dna.DomainStatusPendingDelete`, `d.ExpirationDate.Before(...)`,
  senza `map[string]any`.
- **Ogni operazione è un tipo di richiesta** che incorpora `dna.BaseRequest` e si valida da sola prima
  dell'invio. Scrivi le tue allo stesso modo.
- **Sicuro per impostazione predefinita**: le richieste a pagamento non vengono mai ritentate alla cieca,
  `WithReadOnly` blocca le modifiche, il client regola il proprio ritmo entro i limiti di frequenza ed è sicuro
  per l'uso concorrente.
- **Solo libreria standard.** Go 1.23 o successivo.

```go
client, _ := dna.NewClient(resellerID, apiKey, dna.WithEnvironment(dna.OTE))

a, err := client.Domains.Search(ctx, "example.com")
if err == nil && a.Available() {
	fmt.Printf("%s is available for %.2f %s\n", a.DomainName, a.Price, a.Currency)
}
```

## Indice

- [Installazione](#installazione)
- [Credenziali e ambienti](#credenziali-e-ambienti)
- [Servizi](#servizi)
- [Utilizzo](#utilizzo)
- [Richieste e BaseRequest](#richieste-e-baserequest)
- [Errori](#errori)
- [Sicurezza e affidabilità](#sicurezza-e-affidabilità)
- [Opzioni del client](#opzioni-del-client)
- [Testare il proprio codice](#testare-il-proprio-codice)
- [Migrare dalla v1](#migrare-dalla-v1)
- [Sviluppo](#sviluppo)

## Installazione

```bash
go get github.com/domainreseller/go-dna/v2
```

```go
import dna "github.com/domainreseller/go-dna/v2"
```

I programmi eseguibili si trovano in [examples](examples): `quickstart`, `register`, `expiring`, `dns`, `ssl` e
`custom-request`.

```bash
go run ./examples/quickstart example.com
```

## Credenziali e ambienti

Servono due credenziali API, disponibili nel pannello rivenditore (**API settings**). Le credenziali di accesso
al pannello non funzionano.

| Credenziale | Aspetto                                |
|-------------|----------------------------------------|
| Reseller ID | `xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx` |
| API key     | una lunga stringa segreta              |

Esistono due ambienti. **Inizia con OTE**, l'ambiente di test: ha dati propri e una propria chiave API, e nulla
di ciò che fai lì costa denaro o tocca domini reali.

```go
// Ambiente di test (OTE)
client, err := dna.NewClient(resellerID, oteAPIKey, dna.WithEnvironment(dna.OTE))

// Produzione (predefinito): registrazioni reali, addebiti reali
client, err := dna.NewClient(resellerID, apiKey)
```

`NewClientFromEnv` legge le stesse variabili della CLI `dna`:

| Variabile         | Significato                                                          |
|-------------------|----------------------------------------------------------------------|
| `DNA_RESELLER_ID` | ID rivenditore                                                       |
| `DNA_API_KEY`     | chiave API dell'ambiente selezionato                                 |
| `DNA_ENV`         | `production` (predefinito) oppure `ote`                              |
| `DNA_TEST_MODE`   | `true` seleziona OTE (come negli altri SDK DomainNameAPI)            |
| `DNA_BASE_URL`    | un URL di base personalizzato, ad es. un proxy                       |
| `DNA_APP`         | l'applicazione che usa l'SDK, inviata come `X-App`                   |
| `DNA_APP_VERSION` | la versione di quell'applicazione, inviata come `X-Version`          |

```go
client, err := dna.NewClientFromEnv(dna.WithUserAgent("my-panel/1.4"))
```

Una chiave o un ambiente errati falliscono alla prima richiesta con un `*APIError` di tipo `KindAuth` e un
suggerimento; `client.Account.Get(ctx)` è il modo meno costoso per verificare.

## Servizi

| Servizio          | Gruppo CLI    | Operazioni                                                                                                                                                        |
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

I metodi accettano come argomenti gli input obbligatori (`Info(ctx, "example.com")`) oppure, quando
un'operazione ha input facoltativi, una struct di richiesta (`Register(ctx, dna.DomainRegisterRequest{...})`). Il
riferimento completo con tutti i campi è su [pkg.go.dev](https://pkg.go.dev/github.com/domainreseller/go-dna/v2).

## Utilizzo

Tutti gli snippet presuppongono `ctx := context.Background()` e un `client`.

### Ricerca

```go
a, err := client.Domains.Search(ctx, "example.com")
// a.Available(), a.Price, a.Currency, a.IsPremium, a.IsDocumentRequired, a.Reason

results, err := client.Domains.BulkSearch(ctx, "example.com", "example.net", "example.org")
for _, r := range results {
	fmt.Println(r.DomainName, r.Status, r.Price)
}
```

`BulkSearch` divide le liste di più di 30 domini in più richieste. I prezzi usano la valuta dell'account, oppure
quella impostata con `dna.WithCurrency("TRY")`.

### Registrazione

Con gli handle dei contatti (creane uno con `Contacts.Create`):

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
	Handles:     dna.SameHandle(handle), // oppure dna.ContactHandles{Registrant: "DNA-1", Admin: "DNA-2", ...}
})
```

Con contatti inline:

```go
res, err := client.Domains.RegisterWithContacts(ctx, dna.DomainRegisterWithContactsRequest{
	DomainName:  "example.com",
	NameServers: []string{"ns1.example.net", "ns2.example.net"},
	Contacts:    dna.SameContact(owner), // oppure dna.DomainContacts{Registrant: &owner, Admin: &admin, ...}
})
```

I contatti vengono normalizzati prima dell'invio: gli spazi nei campi vengono rimossi, il paese viene portato in
maiuscolo e i campi telefonici conservano solo le cifre (`"+90 (555) 123-45-67"` funziona). Alcuni TLD richiedono
attributi di registro: trovali con `client.Products.TLD(ctx, "com.tr")` e il suo `RequiredAttributes()`, e
passali come `TLDAttributes`.

Se l'API risponde ma rifiuta l'operazione (`"success": false`), l'errore è un `*dna.OperationError` con il
messaggio dell'API.

### Rinnovo, trasferimento, ripristino

```go
check, err := client.Domains.RenewCheck(ctx, "example.com", 1) // check.Renewable, check.PremiumPrice
res, err := client.Domains.Renew(ctx, "example.com", 1)

tc, err := client.Domains.TransferCheck(ctx, "example.com", "auth-code") // tc.Transferable, tc.AuthCodeValid
res, err := client.Domains.Transfer(ctx, dna.DomainTransferRequest{DomainName: "example.com", AuthCode: "auth-code"})
st, err := client.Domains.TransferStatus(ctx, "example.com")

err = client.Domains.Restore(ctx, "example.com") // durante il periodo di redemption
```

### Dettagli del dominio ed elenco dei domini

```go
d, err := client.Domains.Info(ctx, "example.com")
fmt.Println(d.Status, d.ExpirationDate, d.RemainingDays, d.Locked, d.NameServers, d.AuthCode)
if owner, ok := d.Contact(dna.ContactTypeRegistrant); ok {
	fmt.Println(owner.Email)
}

// Una pagina
page, err := client.Domains.List(ctx, dna.DomainListRequest{ListOptions: dna.ListOptions{Limit: 50}})

// Tutti i domini; le pagine vengono recuperate per te
for d, err := range client.Domains.All(ctx, dna.DomainListRequest{
	Statuses: []dna.DomainStatus{dna.DomainStatusActive},
}) {
	if err != nil {
		return err
	}
	fmt.Println(d.DomainName, d.ExpirationDate)
}
```

### Blocco, privacy, nameserver, nameserver figli

```go
err = client.Domains.Lock(ctx, "example.com")
err = client.Domains.Unlock(ctx, "example.com")
err = client.Domains.SetPrivacy(ctx, "example.com", true)
code, err := client.Domains.UpdateAuthCode(ctx, "example.com")

err = client.Domains.SetNameServers(ctx, dna.DomainNameServersSetRequest{
	DomainName:  "example.com",
	NameServers: []string{"ns1.example.net", "ns2.example.net"},
	// DNSSECMode: dna.DNSSECModeDisable, // solo per i domini firmati con DNSSEC
})

err = client.Domains.AddHost(ctx, "example.com", "ns1.example.com", "192.0.2.10", "2001:db8::10") // versione IP rilevata
err = client.Domains.DeleteHost(ctx, "example.com", "ns1.example.com")
```

### Contatti di un dominio

```go
err = client.Domains.SetContactHandles(ctx, "example.com", dna.ContactHandles{Admin: "DNA-2", Tech: "DNA-2"})
err = client.Domains.UpdateContacts(ctx, "example.com", dna.DomainContacts{Billing: &billing})

details, err := client.Contacts.Get(ctx, "DNA-1")           // il contatto e il suo stato di verifica
result, err := client.Contacts.Update(ctx, "DNA-1", contact) // aggiorna tutti i domini che usano l'handle
```

### Zona DNS

```go
err = client.DNS.Add(ctx, dna.DNSAddRequest{
	DomainName: "example.com", Name: "www", Type: dna.RecordTypeA, TTL: 600, Values: []string{"192.0.2.10"},
})
err = client.DNS.Update(ctx, dna.DNSUpdateRequest{ // sostituisce tutti i valori del set
	DomainName: "example.com", RecordName: "www", Type: dna.RecordTypeA, Values: []string{"192.0.2.20"},
})
err = client.DNS.Delete(ctx, dna.DNSDeleteRequest{
	DomainName: "example.com", Name: "www", Type: dna.RecordTypeA, Value: "192.0.2.20",
})

sets, err := client.DNS.List(ctx, "example.com")
for _, s := range sets {
	fmt.Println(s.Name, s.Type, s.TTL, s.Values())
}

res, err := client.DNS.Apply(ctx, "example.com") // passa il dominio ai nameserver della piattaforma
```

I nomi possono essere relativi (`www`), l'apex (`""` o `"@"`) o completi (FQDN). Qui i tipi di record non
distinguono tra maiuscole e minuscole (`"txt"` viene inviato come `TXT`). La validazione intercetta ciò che l'API
rifiuterebbe: un CNAME all'apex, un valore CAA senza tre parti, un valore MX senza priorità. L'ambiente di test
risponde a ogni chiamata di zona con HTTP 500, quindi il codice per le zone può essere provato solo in
produzione.

### DNSSEC

```go
ds, err := dna.ParseDS("2371 13 2 1F987CC6583E92DF0890718C42...") // come lo mostra il tuo provider DNS
results, err := client.DNSSEC.Add(ctx, "example.com", ds)
info, err := client.DNSSEC.Get(ctx, "example.com") // info.Status, info.DSRecords
res, err := client.DNSSEC.Disable(ctx, "example.com")
```

### Inoltro URL

```go
err = client.Domains.SetForwarding(ctx, "example.com", "https://www.example.org", dna.ForwardTypeStandard)
f, err := client.Domains.Forwarding(ctx, "example.com") // nil quando il dominio non ne ha
err = client.Domains.DeleteForwarding(ctx, "example.com")
```

### Documenti del registro (.tr)

```go
doc, err := dna.DocumentFromFile(dna.DocumentTypeIdentityCard, "id.pdf")
res, err := client.Domains.UploadDocuments(ctx, "example.com.tr", doc)
page, err := client.Domains.Documents(ctx, dna.DomainDocumentsListRequest{DomainName: "example.com.tr"})
```

I file vengono codificati in base64 per te; il formato (PDF, JPG, PNG, DOC, DOCX, ODT) e la dimensione (10 MB
per file, 50 MB per richiesta) vengono verificati prima del caricamento.

### Certificati SSL

Un dominio può avere più certificati, quindi un certificato esistente si indica con il suo **ID del
certificato** (un GUID), mai con il nome di dominio. `Initialize` e `Renew` restituiscono l'ID (conservalo) e
`SSL.List` lo trova a partire dal nome di dominio. Un nome di dominio passato al posto di un ID del certificato
viene rifiutato con un `*dna.InputError` prima che venga inviato qualcosa.

```go
csr, err := client.SSL.GenerateCSR(ctx, dna.SSLGenerateCSRRequest{
	DomainName: "example.com", Country: "TR", State: "Istanbul", Locality: "Istanbul",
	Organization: "Example Ltd", OrganizationUnit: "IT",
})
order, err := client.SSL.Initialize(ctx, "example.com", "PositiveSSL") // addebita il saldo
id := order.CertificateID                                              // conservalo: ogni chiamata successiva lo richiede

_, err = client.SSL.Configure(ctx, dna.SSLConfigureRequest{ // entro 15 giorni
	CertificateID: id, CSR: csr.CSR, ApproverMail: "admin@example.com",
	AuthenticationType: dna.AuthenticationEmail, WebServerType: dna.WebServerApache, Contact: sslContact,
})
st, err := client.SSL.Status(ctx, id) // interroga finché st.IsIssued
cert, err := client.SSL.Content(ctx, id)
os.WriteFile("example.com.crt", []byte(cert.Chain()), 0o644)
```

Un certificato non configurato entro 15 giorni da `Initialize` o `Renew` viene annullato.

`SSL.List` restituisce i tuoi certificati, dal più recente (`SSL.All` li scorre tutti). `SSL.Approvers` elenca
gli indirizzi accettati dalla validazione via e-mail, per un certificato esistente o prima di un acquisto:

```go
page, err := client.SSL.List(ctx, dna.SSLListRequest{DomainName: "example.com"})
for _, c := range page.Items {
	fmt.Println(c.CertificateID, c.Status, c.SubscriptionExpiresAt)
}

mails, err := client.SSL.Approvers(ctx, dna.SSLApproversRequest{CertificateID: id})
mails, err = client.SSL.Approvers(ctx, dna.SSLApproversRequest{DomainName: "example.com", PackageCode: "PositiveSSL"})
```

Per rinnovare un certificato chiama `Renew` e poi configura con `Configure` il nuovo certificato restituito.
`Renew` addebita il saldo ed è possibile a partire da 30 giorni prima della scadenza dell'abbonamento. Non
rinnovare con un secondo `Initialize`: è un acquisto separato, al prezzo di acquisto, e il vecchio certificato non
passa mai allo stato `Renewed`.

```go
renewal, err := client.SSL.Renew(ctx, id) // addebita il saldo
_, err = client.SSL.Configure(ctx, dna.SSLConfigureRequest{
	CertificateID: renewal.CertificateID, // il nuovo certificato; il vecchio mantiene il proprio ID
	CSR: newCSR.CSR, ApproverMail: "admin@example.com",
	AuthenticationType: dna.AuthenticationEmail, WebServerType: dna.WebServerApache, Contact: sslContact,
})
```

### Hosting

```go
packages, err := client.Hosting.Packages(ctx)
created, err := client.Hosting.Create(ctx, "example.com", "Linux-Starter") // addebita il saldo

ctx, cancel := context.WithTimeout(ctx, 30*time.Minute)
defer cancel()
st, err := client.Hosting.Wait(ctx, created.ServiceID) // interroga finché non è attivo
fmt.Println(st.PanelAddress, st.Username, st.Password)
```

Non chiamare mai `Create` di nuovo per un hosting ancora in fase di provisioning: l'addebito avverrebbe due
volte.

### Saldo, prezzi e ordini

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

info, err := client.Products.Info(ctx, "com", dna.OrderRenew, 2) // prezzo di un rinnovo biennale di un .com
order, err := client.Orders.Get(ctx, "ORDER-CODE")
```

## Richieste e BaseRequest

Ogni operazione dell'API è un tipo di richiesta: `DomainInfoRequest`, `DNSAddRequest`,
`SSLConfigureRequest`, ... Tutte implementano un'unica interfaccia:

```go
type Request interface {
	Endpoint() Endpoint      // metodo, percorso e classe di sicurezza
	Query() url.Values       // query string, oppure nil
	Body() any               // corpo JSON, oppure nil
	Validate() error         // verificato prima dell'invio
}
```

Go non ha l'ereditarietà, quindi la «classe base» è una struct da incorporare. `dna.BaseRequest` fornisce i
valori predefiniti (nessuna query, nessun corpo, nulla da validare); una richiesta sovrascrive un metodo
definendolo sul proprio tipo. Solo `Endpoint` non ha un valore predefinito. Una richiesta di questo pacchetto si
presenta così:

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

I metodi dei servizi le costruiscono per te. `client.Do` invia qualsiasi `Request`, compresi i tuoi tipi, ad
esempio per leggere solo i campi che ti servono o per chiamare un endpoint che questo pacchetto non incapsula
ancora:

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

La tua richiesta ottiene tutto ciò che ottengono quelle integrate: credenziali, tentativi ripetuti, controllo
del ritmo, normalizzazione degli errori, ID di correlazione, tracciamento e applicazione della modalità di sola
lettura. `client.NewHTTPRequest(ctx, req)` costruisce l'esatta richiesta HTTP senza inviarla, per registrare o
rivedere una richiesta a pagamento prima che parta.

## Errori

| Errore                 | Quando                                                     | Campi utili                                                      |
|------------------------|------------------------------------------------------------|------------------------------------------------------------------|
| `*dna.InputError`      | `Validate` ha rifiutato la richiesta; **non è stato inviato nulla** | `Request`, `Fields`                                     |
| `*dna.APIError`        | l'API ha risposto con uno stato non 2xx                    | `StatusCode`, `Code`, `Message`, `Details`, `Fields`, `Data`, `CorrelationID`, `Hint()` |
| `*dna.OperationError`  | l'API ha risposto 2xx con `"success": false`               | `Code`, `Message`, `CorrelationID`                               |
| `*dna.NetworkError`    | la richiesta non è stata completata (DNS, TLS, timeout, reset) | `Err`, `Timeout()`                                           |
| `dna.ErrReadOnly`      | un client di sola lettura ha bloccato una modifica; non è stato inviato nulla | usa `errors.Is`                              |

`dna.KindOf(err)` classifica ciascuno di essi: `KindAuth`, `KindNotFound`, `KindValidation`, `KindRejected`,
`KindConflict`, `KindRateLimited`, `KindServer`, `KindNetwork`, `KindInvalidInput`, `KindReadOnly`. Alcune
funzioni di supporto rispondono alle domande più comuni:

```go
_, err := client.Domains.Renew(ctx, "example.com", 1)
var apiErr *dna.APIError
switch {
case err == nil:
case dna.OutcomeUnknown(err):
	// Il rinnovo potrebbe essere andato a buon fine. Controlla il dominio prima di riprovare.
case dna.IsInsufficientBalance(err):
case dna.IsNotFound(err):
case dna.HasCode(err, dna.CodePrivacyNotSupported):
case errors.As(err, &apiErr):
	log.Println(apiErr.Code, apiErr.Message, apiErr.Hint(), apiErr.CorrelationID)
}
```

Distingui i casi in base a `Code`, non a `Message`: i messaggi sono localizzati. I codici dell'API hanno questo
aspetto: `Dna.DomainService:Domain:10007`; vedi il [riferimento degli errori](https://www.domainnameapi.com/docs/reference/errors.md).
Indica il `CorrelationID` quando contatti il supporto.

## Sicurezza e affidabilità

- **Classi di sicurezza.** Ogni richiesta è `Read`, `Write`, `Destructive` o `Billable`
  (`req.Endpoint().Safety`). Alcune richieste POST e PUT sono letture (`DomainSearchRequest`,
  `SSLStatusRequest`).
- **Modalità di sola lettura.** `dna.WithReadOnly()` blocca tutto ciò che non è una lettura, per i job di
  reportistica e per esplorare la produzione in sicurezza.
- **Tentativi ripetuti.** Le risposte 429 vengono sempre ritentate, rispettando `Retry-After`: l'API le rifiuta
  prima di fare qualsiasi cosa. Gli errori di rete e i 502/503/504 vengono ritentati solo per le letture e per le
  scritture idempotenti, così una registrazione o un rinnovo non viene mai inviato due volte. Dopo il fallimento
  di una richiesta a pagamento, controlla `dna.OutcomeUnknown(err)`.
- **Controllo del ritmo.** Il client distanzia le richieste per restare entro i limiti per chiave (5
  richieste/secondo, 2 ricerche/secondo, una chiamata di zona al secondo). Disattivalo con
  `dna.WithPacing(false)` se regoli tu il ritmo.
- **Quota.** `Response.Quota` espone gli header della quota soft. Oltre il limite l'API rallenta le risposte
  (fino a 5 s) invece di rifiutarle; il timeout predefinito di 60 s lascia margine per questo.
- **ID di correlazione.** Ogni richiesta porta un `X-Correlation-ID`. Imposta il tuo con
  `dna.WithCorrelationID(ctx, id)` per collegare le chiamate API ai tuoi log.
- **Osservabilità.** `dna.WithLogger(slog.Default())` registra ogni scambio; `dna.WithTrace(func(dna.TraceEvent))`
  ti consegna ogni richiesta, risposta e nuovo tentativo, per metriche o segnalazione degli errori. La chiave API
  non vi compare mai.
- **Concorrenza.** Un `Client` è sicuro per l'uso concorrente. Creane uno e condividilo.
- **Decodifica tollerante.** Vengono decodificati le date senza fuso orario, `null`, `[]` al posto di `{}`, gli
  stati numerici o con nome e i ruoli in qualsiasi maiuscolo/minuscolo (`"registrant"`, `"ADMINISTRATIVE"`).

## Opzioni del client

| Opzione                         | Effetto                                                                                  |
|---------------------------------|------------------------------------------------------------------------------------------|
| `WithEnvironment(dna.OTE)`      | usa l'ambiente di test (predefinito `dna.Production`)                                    |
| `WithBaseURL(url)`              | invia le richieste a un proxy o a un server di test                                      |
| `WithHTTPClient(hc)`            | il tuo `*http.Client` (transport, proxy, TLS); i redirect non vengono mai seguiti        |
| `WithTimeout(d)`                | timeout per tentativo (predefinito 60 s)                                                 |
| `WithUserAgent("app/1.0")`      | antepone il tuo prodotto allo User-Agent                                                 |
| `WithApp("wisecp", "2.0.20")`   | indica la tua applicazione e la sua versione (`X-App`, `X-Version`)                      |
| `WithCurrency("TRY")`           | valuta dei prezzi di ricerca e del catalogo                                              |
| `WithRetries(n)`                | numero di tentativi (predefinito 2)                                                      |
| `WithPacing(false)`             | disattiva il controllo del ritmo lato client                                             |
| `WithReadOnly()`                | blocca ogni richiesta che non sia di lettura                                             |
| `WithLogger(l)`                 | registra gli scambi su un `*slog.Logger`                                                 |
| `WithTrace(fn)`                 | osserva ogni richiesta, risposta e nuovo tentativo                                       |

### Identificare la propria applicazione

Ogni richiesta porta gli header `X-App` e `X-Version`, che dicono a DomainNameAPI quale integrazione l'ha inviata.
Per impostazione predefinita valgono `go` e la versione dell'SDK. Indica la tua applicazione con `WithApp` (oppure con
`DNA_APP` e `DNA_APP_VERSION` se usi `NewClientFromEnv`):

```go
client, err := dna.NewClient(resellerID, apiKey, dna.WithApp("WiseCP", "V2.0.20-beta"))
// X-App: wisecp, X-Version: 2.0.20
```

Il nome viene convertito in minuscolo e mantiene solo `a-z`, `0-9`, `.`, `_` e `-`; la versione mantiene solo cifre e
punti. Un nome vuoto o non valido torna a `go`, una versione vuota alla versione dell'SDK. Lo User-Agent non cambia.

## Testare il proprio codice

Punta il client a un `httptest.Server`; non serve simulare nient'altro:

```go
srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, `{"success":true,"info":{"domainName":"example.com","status":"AVAILABLE","price":11.99,"currency":"USD"}}`)
}))
defer srv.Close()

client, _ := dna.NewClient("reseller-id", "api-key", dna.WithBaseURL(srv.URL), dna.WithPacing(false))
a, err := client.Domains.Search(ctx, "example.com")
```

## Migrare dalla v1

La v2 è un nuovo progetto: risultati tipizzati al posto di `map[string]any`, errori al posto delle buste
`{"result": "ERROR"}` e l'API completa al posto dell'insieme di metodi di php-dna. Il percorso del modulo è
cambiato in `github.com/domainreseller/go-dna/v2`, quindi la v1 continua a funzionare finché non passi alla v2.

| v1                                                  | v2                                                                     |
|-----------------------------------------------------|------------------------------------------------------------------------|
| `dna.New(id, key, true)`                            | `dna.NewClient(id, key, dna.WithEnvironment(dna.OTE))`                 |
| `res["result"] == "OK"`                             | `err == nil`                                                           |
| `GetResellerDetails`, `GetCurrentBalance`           | `Account.Get` (+ `Account.Balance("USD")`)                             |
| `CheckAvailability`                                 | `Domains.Search`, `Domains.BulkSearch`                                 |
| `GetList`                                           | `Domains.List`, `Domains.All`                                          |
| `GetTldList`                                        | `Products.TLDs`, `Products.AllTLDs`, `Products.TLD`                    |
| `GetDetails`, `GetContacts`, `SyncFromRegistry`     | `Domains.Info`                                                         |
| `RegisterWithContactInfo`                           | `Domains.RegisterWithContacts` (oppure `Contacts.Create` + `Domains.Register`) |
| `Renew`, `Transfer`, `CheckTransfer`                | `Domains.Renew`, `Domains.Transfer`, `Domains.TransferCheck`           |
| `ModifyNameServer`                                  | `Domains.SetNameServers`                                               |
| `EnableTheftProtectionLock` / `Disable...`          | `Domains.Lock` / `Domains.Unlock`                                      |
| `ModifyPrivacyProtectionStatus`                     | `Domains.SetPrivacy`                                                   |
| `SaveContacts`                                      | `Domains.UpdateContacts`, `Domains.SetContactHandles`                  |
| `AddChildNameServer`, `ModifyChildNameServer`, `DeleteChildNameServer` | `Domains.AddHost`, `Domains.UpdateHost`, `Domains.DeleteHost` |
| `GetResourceRecords`                                | `DNS.List`                                                             |
| `AddResourceRecord`, `AddResourceRecordValue`       | `DNS.Add`                                                              |
| `EditResourceRecord`, `SetResourceRecordSet`, `ReplaceResourceRecordValue` | `DNS.Update`                                    |
| `DeleteResourceRecord`, `RemoveResourceRecordValue`, `DeleteResourceRecordSet` | `DNS.Delete` (una volta per valore)         |
| `ApplyResourceRecords`                              | `DNS.Apply`                                                            |
| `GetForwarding`, `SetForwarding`, `DeleteForwarding`| `Domains.Forwarding`, `Domains.SetForwarding`, `Domains.DeleteForwarding` |
| `CancelTransfer`, `ApproveTransfer`, `RejectTransfer` | non presenti nell'API pubblicata; invia una [richiesta personalizzata](#richieste-e-baserequest) se il tuo account le prevede |
| `IsTrTLD`, `ValidateContact`                        | `strings.HasSuffix(name, ".tr")`; la validazione viene eseguita a ogni richiesta |
| `GetRequestData`, `GetResponseData`, ...            | `WithTrace`, `WithLogger`, `Response` da `client.Do`                   |
| telemetria (`SetErrorReportingEnabled`, Sentry)     | rimossa: nulla lascia il tuo processo; inoltra gli eventi di `WithTrace` al tuo sistema di segnalazione |
| `SetSSLVerification(false)`                         | `WithHTTPClient` con le tue impostazioni TLS                           |

## Sviluppo

```bash
go test ./...          # test unitari; nessuno accede alla rete
go vet ./...
gofmt -l .
```

`testdata/openapi.json` è il documento OpenAPI dell'API. I test verificano ogni tipo di richiesta rispetto ad
esso (metodo, percorso, nomi delle query, campi del corpo, campi obbligatori), i nomi JSON di ogni tipo di
risposta rispetto agli schemi di risposta e ogni valore di enumerazione, e falliscono quando l'API acquisisce
un'operazione senza un tipo di richiesta. Per aggiornarlo:

```bash
curl -fsSL -A "go-dna-spec-update" -o testdata/openapi.json https://api.domainresellerapi.com/swagger/v1/swagger.json
go test ./...
```

Vedi [AGENTS.md](AGENTS.md) per le convenzioni di questo repository.

## Supporto

- Documentazione dell'API: <https://www.domainnameapi.com/docs/>
- CLI e server MCP: <https://github.com/domainreseller/cli>
- Sito web: <https://www.domainnameapi.com>

## Licenza

Apache License 2.0. Vedi [LICENSE](LICENSE).
