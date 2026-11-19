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

# go-dna : le SDK Go pour la Domain Name API

[![Go Reference](https://pkg.go.dev/badge/github.com/domainreseller/go-dna/v2.svg)](https://pkg.go.dev/github.com/domainreseller/go-dna/v2)
[![CI](https://github.com/domainreseller/go-dna/actions/workflows/ci.yml/badge.svg)](https://github.com/domainreseller/go-dna/actions/workflows/ci.yml)
[![License](https://img.shields.io/badge/license-Apache--2.0-blue.svg)](LICENSE)

Client Go typé pour l'API REST revendeur de [Domain Name API](https://www.domainnameapi.com) : recherche de
domaines, enregistrement, renouvellement et transfert, DNS, DNSSEC, contacts, certificats SSL, hébergement,
commandes, catalogue de produits et solde.

- **Les 64 opérations de l'API**, regroupées comme dans la [CLI `dna`](https://github.com/domainreseller/cli) :
  `dna domain info` correspond à `client.Domains.Info`, `dna dns add` à `client.DNS.Add`.
- **Résultats et énumérations typés** : `d.Status == dna.DomainStatusPendingDelete`, `d.ExpirationDate.Before(...)`,
  sans `map[string]any`.
- **Chaque opération est un type de requête** qui embarque `dna.BaseRequest` et se valide elle-même avant tout
  envoi. Écrivez les vôtres de la même façon.
- **Sûr par défaut** : les requêtes facturables ne sont jamais rejouées aveuglément, `WithReadOnly` bloque les
  modifications, le client se cadence sous les limites de débit et peut être utilisé de façon concurrente.
- **Bibliothèque standard uniquement.** Go 1.23 ou plus récent.

```go
client, _ := dna.NewClient(resellerID, apiKey, dna.WithEnvironment(dna.OTE))

a, err := client.Domains.Search(ctx, "example.com")
if err == nil && a.Available() {
	fmt.Printf("%s is available for %.2f %s\n", a.DomainName, a.Price, a.Currency)
}
```

## Sommaire

- [Installation](#installation)
- [Identifiants et environnements](#identifiants-et-environnements)
- [Services](#services)
- [Utilisation](#utilisation)
- [Requêtes et BaseRequest](#requêtes-et-baserequest)
- [Erreurs](#erreurs)
- [Sécurité et fiabilité](#sécurité-et-fiabilité)
- [Options du client](#options-du-client)
- [Tester votre code](#tester-votre-code)
- [Migrer depuis la v1](#migrer-depuis-la-v1)
- [Développement](#développement)

## Installation

```bash
go get github.com/domainreseller/go-dna/v2
```

```go
import dna "github.com/domainreseller/go-dna/v2"
```

Des programmes exécutables se trouvent dans [examples](examples) : `quickstart`, `register`, `expiring`, `dns`,
`ssl` et `custom-request`.

```bash
go run ./examples/quickstart example.com
```

## Identifiants et environnements

Vous avez besoin de deux identifiants d'API, disponibles dans le panneau revendeur (**API settings**). Les
identifiants de connexion au panneau ne fonctionnent pas.

| Identifiant | Aspect                                 |
|-------------|----------------------------------------|
| Reseller ID | `xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx` |
| API key     | une longue chaîne secrète              |

Il existe deux environnements. **Commencez par OTE**, l'environnement de test : il possède ses propres données et
sa propre clé d'API, et rien de ce que vous y faites ne coûte d'argent ni n'affecte de domaines en production.

```go
// Environnement de test (OTE)
client, err := dna.NewClient(resellerID, oteAPIKey, dna.WithEnvironment(dna.OTE))

// Production (par défaut) : vrais enregistrements, vraie facturation
client, err := dna.NewClient(resellerID, apiKey)
```

`NewClientFromEnv` lit les mêmes variables que la CLI `dna` :

| Variable          | Signification                                                        |
|-------------------|----------------------------------------------------------------------|
| `DNA_RESELLER_ID` | ID revendeur                                                         |
| `DNA_API_KEY`     | clé d'API de l'environnement sélectionné                             |
| `DNA_ENV`         | `production` (par défaut) ou `ote`                                   |
| `DNA_TEST_MODE`   | `true` sélectionne OTE (comme dans les autres SDK DomainNameAPI)     |
| `DNA_BASE_URL`    | une URL de base personnalisée, par ex. un proxy                      |

```go
client, err := dna.NewClientFromEnv(dna.WithUserAgent("my-panel/1.4"))
```

Une clé ou un environnement incorrect échoue dès la première requête avec une `*APIError` de type `KindAuth` et
une indication ; `client.Account.Get(ctx)` est le moyen le moins coûteux de vérifier.

## Services

| Service           | Groupe CLI    | Opérations                                                                                                                                                        |
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

Les méthodes prennent les entrées obligatoires en arguments (`Info(ctx, "example.com")`) ou, lorsqu'une opération
a des entrées facultatives, une structure de requête (`Register(ctx, dna.DomainRegisterRequest{...})`). La
référence complète avec tous les champs se trouve sur [pkg.go.dev](https://pkg.go.dev/github.com/domainreseller/go-dna/v2).

## Utilisation

Tous les extraits supposent `ctx := context.Background()` et un `client`.

### Recherche

```go
a, err := client.Domains.Search(ctx, "example.com")
// a.Available(), a.Price, a.Currency, a.IsPremium, a.IsDocumentRequired, a.Reason

results, err := client.Domains.BulkSearch(ctx, "example.com", "example.net", "example.org")
for _, r := range results {
	fmt.Println(r.DomainName, r.Status, r.Price)
}
```

`BulkSearch` découpe les listes de plus de 30 domaines en plusieurs requêtes. Les prix sont exprimés dans la
devise du compte, ou dans celle définie avec `dna.WithCurrency("TRY")`.

### Enregistrement

Avec des identifiants de contact (créez-en un avec `Contacts.Create`) :

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
	Handles:     dna.SameHandle(handle), // ou dna.ContactHandles{Registrant: "DNA-1", Admin: "DNA-2", ...}
})
```

Avec des contacts en ligne :

```go
res, err := client.Domains.RegisterWithContacts(ctx, dna.DomainRegisterWithContactsRequest{
	DomainName:  "example.com",
	NameServers: []string{"ns1.example.net", "ns2.example.net"},
	Contacts:    dna.SameContact(owner), // ou dna.DomainContacts{Registrant: &owner, Admin: &admin, ...}
})
```

Les contacts sont normalisés avant l'envoi : les champs sont débarrassés des espaces superflus, le pays est mis
en majuscules et les champs de téléphone ne conservent que leurs chiffres (`"+90 (555) 123-45-67"` fonctionne).
Certains TLD exigent des attributs de registre : trouvez-les avec `client.Products.TLD(ctx, "com.tr")` et son
`RequiredAttributes()`, puis transmettez-les sous forme de `TLDAttributes`.

Si l'API répond mais refuse l'opération (`"success": false`), l'erreur est une `*dna.OperationError` portant le
message de l'API.

### Renouvellement, transfert, restauration

```go
check, err := client.Domains.RenewCheck(ctx, "example.com", 1) // check.Renewable, check.PremiumPrice
res, err := client.Domains.Renew(ctx, "example.com", 1)

tc, err := client.Domains.TransferCheck(ctx, "example.com", "auth-code") // tc.Transferable, tc.AuthCodeValid
res, err := client.Domains.Transfer(ctx, dna.DomainTransferRequest{DomainName: "example.com", AuthCode: "auth-code"})
st, err := client.Domains.TransferStatus(ctx, "example.com")

err = client.Domains.Restore(ctx, "example.com") // pendant la période de rédemption
```

### Détails d'un domaine et liste des domaines

```go
d, err := client.Domains.Info(ctx, "example.com")
fmt.Println(d.Status, d.ExpirationDate, d.RemainingDays, d.Locked, d.NameServers, d.AuthCode)
if owner, ok := d.Contact(dna.ContactTypeRegistrant); ok {
	fmt.Println(owner.Email)
}

// Une page
page, err := client.Domains.List(ctx, dna.DomainListRequest{ListOptions: dna.ListOptions{Limit: 50}})

// Tous les domaines ; les pages sont récupérées pour vous
for d, err := range client.Domains.All(ctx, dna.DomainListRequest{
	Statuses: []dna.DomainStatus{dna.DomainStatusActive},
}) {
	if err != nil {
		return err
	}
	fmt.Println(d.DomainName, d.ExpirationDate)
}
```

### Verrouillage, confidentialité, serveurs de noms, serveurs de noms enfants

```go
err = client.Domains.Lock(ctx, "example.com")
err = client.Domains.Unlock(ctx, "example.com")
err = client.Domains.SetPrivacy(ctx, "example.com", true)
code, err := client.Domains.UpdateAuthCode(ctx, "example.com")

err = client.Domains.SetNameServers(ctx, dna.DomainNameServersSetRequest{
	DomainName:  "example.com",
	NameServers: []string{"ns1.example.net", "ns2.example.net"},
	// DNSSECMode: dna.DNSSECModeDisable, // uniquement pour les domaines signés avec DNSSEC
})

err = client.Domains.AddHost(ctx, "example.com", "ns1.example.com", "192.0.2.10", "2001:db8::10") // version IP détectée
err = client.Domains.DeleteHost(ctx, "example.com", "ns1.example.com")
```

### Contacts d'un domaine

```go
err = client.Domains.SetContactHandles(ctx, "example.com", dna.ContactHandles{Admin: "DNA-2", Tech: "DNA-2"})
err = client.Domains.UpdateContacts(ctx, "example.com", dna.DomainContacts{Billing: &billing})

details, err := client.Contacts.Get(ctx, "DNA-1")           // le contact et son état de vérification
result, err := client.Contacts.Update(ctx, "DNA-1", contact) // met à jour tous les domaines utilisant l'identifiant
```

### Zone DNS

```go
err = client.DNS.Add(ctx, dna.DNSAddRequest{
	DomainName: "example.com", Name: "www", Type: dna.RecordTypeA, TTL: 600, Values: []string{"192.0.2.10"},
})
err = client.DNS.Update(ctx, dna.DNSUpdateRequest{ // remplace toutes les valeurs du jeu
	DomainName: "example.com", RecordName: "www", Type: dna.RecordTypeA, Values: []string{"192.0.2.20"},
})
err = client.DNS.Delete(ctx, dna.DNSDeleteRequest{
	DomainName: "example.com", Name: "www", Type: dna.RecordTypeA, Value: "192.0.2.20",
})

sets, err := client.DNS.List(ctx, "example.com")
for _, s := range sets {
	fmt.Println(s.Name, s.Type, s.TTL, s.Values())
}

res, err := client.DNS.Apply(ctx, "example.com") // bascule le domaine sur les serveurs de noms de la plateforme
```

Les noms peuvent être relatifs (`www`), l'apex (`""` ou `"@"`) ou complets (FQDN). Les types d'enregistrement ne
sont pas sensibles à la casse ici (`"txt"` est envoyé comme `TXT`). La validation détecte ce que l'API
rejetterait : un CNAME à l'apex, une valeur CAA sans trois parties, une valeur MX sans priorité.
L'environnement de test répond à chaque appel de zone par un HTTP 500 ; le code de zone ne peut donc être
essayé qu'en production.

### DNSSEC

```go
ds, err := dna.ParseDS("2371 13 2 1F987CC6583E92DF0890718C42...") // tel que l'affiche votre fournisseur DNS
results, err := client.DNSSEC.Add(ctx, "example.com", ds)
info, err := client.DNSSEC.Get(ctx, "example.com") // info.Status, info.DSRecords
res, err := client.DNSSEC.Disable(ctx, "example.com")
```

### Redirection d'URL

```go
err = client.Domains.SetForwarding(ctx, "example.com", "https://www.example.org", dna.ForwardTypeStandard)
f, err := client.Domains.Forwarding(ctx, "example.com") // nil lorsque le domaine n'en a pas
err = client.Domains.DeleteForwarding(ctx, "example.com")
```

### Documents du registre (.tr)

```go
doc, err := dna.DocumentFromFile(dna.DocumentTypeIdentityCard, "id.pdf")
res, err := client.Domains.UploadDocuments(ctx, "example.com.tr", doc)
page, err := client.Domains.Documents(ctx, dna.DomainDocumentsListRequest{DomainName: "example.com.tr"})
```

Les fichiers sont encodés en base64 pour vous ; le format (PDF, JPG, PNG, DOC, DOCX, ODT) et la taille (10 Mo
par fichier, 50 Mo par requête) sont vérifiés avant l'envoi.

### Certificats SSL

```go
csr, err := client.SSL.GenerateCSR(ctx, dna.SSLGenerateCSRRequest{
	DomainName: "example.com", Country: "TR", State: "Istanbul", Locality: "Istanbul",
	Organization: "Example Ltd", OrganizationUnit: "IT",
})
_, err = client.SSL.Initialize(ctx, "example.com", "PositiveSSL") // débite le solde
_, err = client.SSL.Configure(ctx, dna.SSLConfigureRequest{
	DomainName: "example.com", CSR: csr.CSR, ApproverMail: "admin@example.com",
	AuthenticationType: dna.AuthenticationEmail, WebServerType: dna.WebServerApache, Contact: sslContact,
})
st, err := client.SSL.Status(ctx, "example.com") // interrogez jusqu'à st.IsIssued
cert, err := client.SSL.Content(ctx, "example.com")
os.WriteFile("example.com.crt", []byte(cert.Chain()), 0o644)
```

### Hébergement

```go
packages, err := client.Hosting.Packages(ctx)
created, err := client.Hosting.Create(ctx, "example.com", "Linux-Starter") // débite le solde

ctx, cancel := context.WithTimeout(ctx, 30*time.Minute)
defer cancel()
st, err := client.Hosting.Wait(ctx, created.ServiceID) // interroge jusqu'à l'activation
fmt.Println(st.PanelAddress, st.Username, st.Password)
```

N'appelez jamais `Create` une seconde fois pour un hébergement encore en cours de provisionnement : cela
facture deux fois.

### Solde, prix et commandes

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

info, err := client.Products.Info(ctx, "com", dna.OrderRenew, 2) // prix d'un renouvellement de .com sur 2 ans
order, err := client.Orders.Get(ctx, "ORDER-CODE")
```

## Requêtes et BaseRequest

Chaque opération de l'API est un type de requête : `DomainInfoRequest`, `DNSAddRequest`,
`SSLConfigureRequest`, ... Toutes implémentent une même interface :

```go
type Request interface {
	Endpoint() Endpoint      // méthode, chemin et classe de sécurité
	Query() url.Values       // chaîne de requête, ou nil
	Body() any               // corps JSON, ou nil
	Validate() error         // vérifié avant tout envoi
}
```

Go n'a pas d'héritage ; la « classe de base » est donc une structure que l'on embarque. `dna.BaseRequest` fournit
les valeurs par défaut (pas de query, pas de corps, rien à valider) ; une requête redéfinit une méthode en la
déclarant sur son propre type. Seule `Endpoint` n'a pas de valeur par défaut. Une requête de ce paquet ressemble
à ceci :

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

Les méthodes des services les construisent pour vous. `client.Do` envoie n'importe quelle `Request`, y compris
vos propres types, par exemple pour ne lire que les champs dont vous avez besoin ou pour appeler un endpoint que
ce paquet n'encapsule pas encore :

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

Votre requête bénéficie de tout ce dont bénéficient les requêtes intégrées : identifiants, nouvelles tentatives,
cadencement, normalisation des erreurs, identifiants de corrélation, traçage et application du mode lecture
seule. `client.NewHTTPRequest(ctx, req)` construit la requête HTTP exacte sans l'envoyer, pour journaliser ou
relire une requête facturable avant son départ.

## Erreurs

| Erreur                 | Quand                                                      | Champs utiles                                                    |
|------------------------|------------------------------------------------------------|------------------------------------------------------------------|
| `*dna.InputError`      | `Validate` a rejeté la requête ; **rien n'a été envoyé**   | `Request`, `Fields`                                              |
| `*dna.APIError`        | l'API a répondu avec un statut non 2xx                     | `StatusCode`, `Code`, `Message`, `Details`, `Fields`, `Data`, `CorrelationID`, `Hint()` |
| `*dna.OperationError`  | l'API a répondu 2xx avec `"success": false`                | `Code`, `Message`, `CorrelationID`                               |
| `*dna.NetworkError`    | la requête n'a pas abouti (DNS, TLS, délai dépassé, réinitialisation) | `Err`, `Timeout()`                                    |
| `dna.ErrReadOnly`      | un client en lecture seule a bloqué une modification ; rien n'a été envoyé | utilisez `errors.Is`                          |

`dna.KindOf(err)` classe chacune d'elles : `KindAuth`, `KindNotFound`, `KindValidation`, `KindRejected`,
`KindConflict`, `KindRateLimited`, `KindServer`, `KindNetwork`, `KindInvalidInput`, `KindReadOnly`. Des
fonctions utilitaires répondent aux questions courantes :

```go
_, err := client.Domains.Renew(ctx, "example.com", 1)
var apiErr *dna.APIError
switch {
case err == nil:
case dna.OutcomeUnknown(err):
	// Le renouvellement a peut-être abouti. Vérifiez le domaine avant de réessayer.
case dna.IsInsufficientBalance(err):
case dna.IsNotFound(err):
case dna.HasCode(err, dna.CodePrivacyNotSupported):
case errors.As(err, &apiErr):
	log.Println(apiErr.Code, apiErr.Message, apiErr.Hint(), apiErr.CorrelationID)
}
```

Branchez-vous sur `Code`, pas sur `Message` : les messages sont localisés. Les codes de l'API ressemblent à
`Dna.DomainService:Domain:10007` ; voir la [référence des erreurs](https://www.domainnameapi.com/docs/reference/errors.md).
Indiquez le `CorrelationID` lorsque vous contactez le support.

## Sécurité et fiabilité

- **Classes de sécurité.** Chaque requête est `Read`, `Write`, `Destructive` ou `Billable`
  (`req.Endpoint().Safety`). Certaines requêtes POST et PUT sont des lectures (`DomainSearchRequest`,
  `SSLStatusRequest`).
- **Mode lecture seule.** `dna.WithReadOnly()` bloque tout ce qui n'est pas une lecture, pour les tâches de
  reporting et pour explorer la production en toute sécurité.
- **Nouvelles tentatives.** Les réponses 429 sont toujours rejouées, en respectant `Retry-After` : l'API les
  rejette avant de rien faire. Les erreurs réseau et les 502/503/504 ne sont rejouées que pour les lectures et
  les écritures idempotentes, de sorte qu'un enregistrement ou un renouvellement n'est jamais envoyé deux fois.
  Après l'échec d'une requête facturable, vérifiez `dna.OutcomeUnknown(err)`.
- **Cadencement.** Le client espace les requêtes pour rester sous les limites par clé (5 requêtes/seconde,
  2 recherches/seconde, un appel de zone par seconde). Désactivez-le avec `dna.WithPacing(false)` si vous
  cadencez vous-même.
- **Quota.** `Response.Quota` expose les en-têtes de quota souple. Au-delà de la limite, l'API ralentit les
  réponses (jusqu'à 5 s) au lieu de les rejeter ; le délai d'expiration par défaut de 60 s laisse de la marge
  pour cela.
- **Identifiants de corrélation.** Chaque requête porte un `X-Correlation-ID`. Définissez le vôtre avec
  `dna.WithCorrelationID(ctx, id)` pour relier les appels d'API à vos journaux.
- **Observabilité.** `dna.WithLogger(slog.Default())` journalise chaque échange ; `dna.WithTrace(func(dna.TraceEvent))`
  vous transmet chaque requête, réponse et nouvelle tentative, pour les métriques ou le suivi des erreurs. La
  clé d'API n'y figure jamais.
- **Concurrence.** Un `Client` peut être utilisé de façon concurrente. Créez-en un et partagez-le.
- **Décodage tolérant.** Les dates sans fuseau horaire, `null`, `[]` au lieu de `{}`, les statuts numériques ou
  nommés et les rôles quelle que soit la casse (`"registrant"`, `"ADMINISTRATIVE"`) sont tous décodés.

## Options du client

| Option                          | Effet                                                                                    |
|---------------------------------|------------------------------------------------------------------------------------------|
| `WithEnvironment(dna.OTE)`      | utilise l'environnement de test (par défaut `dna.Production`)                            |
| `WithBaseURL(url)`              | envoie les requêtes à un proxy ou à un serveur de test                                   |
| `WithHTTPClient(hc)`            | votre propre `*http.Client` (transport, proxy, TLS) ; les redirections ne sont jamais suivies |
| `WithTimeout(d)`                | délai d'expiration par tentative (60 s par défaut)                                       |
| `WithUserAgent("app/1.0")`      | place votre produit devant le User-Agent                                                 |
| `WithCurrency("TRY")`           | devise des prix de recherche et du catalogue                                             |
| `WithRetries(n)`                | nombre de nouvelles tentatives (2 par défaut)                                            |
| `WithPacing(false)`             | désactive le cadencement côté client                                                     |
| `WithReadOnly()`                | bloque toute requête qui n'est pas une lecture                                           |
| `WithLogger(l)`                 | journalise les échanges dans un `*slog.Logger`                                           |
| `WithTrace(fn)`                 | observe chaque requête, réponse et nouvelle tentative                                    |

## Tester votre code

Pointez le client vers un `httptest.Server` ; rien d'autre n'a besoin d'être simulé :

```go
srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, `{"success":true,"info":{"domainName":"example.com","status":"AVAILABLE","price":11.99,"currency":"USD"}}`)
}))
defer srv.Close()

client, _ := dna.NewClient("reseller-id", "api-key", dna.WithBaseURL(srv.URL), dna.WithPacing(false))
a, err := client.Domains.Search(ctx, "example.com")
```

## Migrer depuis la v1

La v2 est une nouvelle conception : des résultats typés au lieu de `map[string]any`, des erreurs au lieu
d'enveloppes `{"result": "ERROR"}`, et l'API complète au lieu de l'ensemble de méthodes de php-dna. Le chemin du
module a changé en `github.com/domainreseller/go-dna/v2` ; la v1 continue donc de fonctionner jusqu'à ce que vous
changiez.

| v1                                                  | v2                                                                     |
|-----------------------------------------------------|------------------------------------------------------------------------|
| `dna.New(id, key, true)`                            | `dna.NewClient(id, key, dna.WithEnvironment(dna.OTE))`                 |
| `res["result"] == "OK"`                             | `err == nil`                                                           |
| `GetResellerDetails`, `GetCurrentBalance`           | `Account.Get` (+ `Account.Balance("USD")`)                             |
| `CheckAvailability`                                 | `Domains.Search`, `Domains.BulkSearch`                                 |
| `GetList`                                           | `Domains.List`, `Domains.All`                                          |
| `GetTldList`                                        | `Products.TLDs`, `Products.AllTLDs`, `Products.TLD`                    |
| `GetDetails`, `GetContacts`, `SyncFromRegistry`     | `Domains.Info`                                                         |
| `RegisterWithContactInfo`                           | `Domains.RegisterWithContacts` (ou `Contacts.Create` + `Domains.Register`) |
| `Renew`, `Transfer`, `CheckTransfer`                | `Domains.Renew`, `Domains.Transfer`, `Domains.TransferCheck`           |
| `ModifyNameServer`                                  | `Domains.SetNameServers`                                               |
| `EnableTheftProtectionLock` / `Disable...`          | `Domains.Lock` / `Domains.Unlock`                                      |
| `ModifyPrivacyProtectionStatus`                     | `Domains.SetPrivacy`                                                   |
| `SaveContacts`                                      | `Domains.UpdateContacts`, `Domains.SetContactHandles`                  |
| `AddChildNameServer`, `ModifyChildNameServer`, `DeleteChildNameServer` | `Domains.AddHost`, `Domains.UpdateHost`, `Domains.DeleteHost` |
| `GetResourceRecords`                                | `DNS.List`                                                             |
| `AddResourceRecord`, `AddResourceRecordValue`       | `DNS.Add`                                                              |
| `EditResourceRecord`, `SetResourceRecordSet`, `ReplaceResourceRecordValue` | `DNS.Update`                                    |
| `DeleteResourceRecord`, `RemoveResourceRecordValue`, `DeleteResourceRecordSet` | `DNS.Delete` (une fois par valeur)          |
| `ApplyResourceRecords`                              | `DNS.Apply`                                                            |
| `GetForwarding`, `SetForwarding`, `DeleteForwarding`| `Domains.Forwarding`, `Domains.SetForwarding`, `Domains.DeleteForwarding` |
| `CancelTransfer`, `ApproveTransfer`, `RejectTransfer` | absent de l'API publiée ; envoyez une [requête personnalisée](#requêtes-et-baserequest) si votre compte les propose |
| `IsTrTLD`, `ValidateContact`                        | `strings.HasSuffix(name, ".tr")` ; la validation s'exécute sur chaque requête |
| `GetRequestData`, `GetResponseData`, ...            | `WithTrace`, `WithLogger`, `Response` de `client.Do`                   |
| télémétrie (`SetErrorReportingEnabled`, Sentry)     | supprimée : rien ne quitte votre processus ; transmettez les événements `WithTrace` à votre propre outil de suivi |
| `SetSSLVerification(false)`                         | `WithHTTPClient` avec vos propres paramètres TLS                       |

## Développement

```bash
go test ./...          # tests unitaires ; aucun n'accède au réseau
go vet ./...
gofmt -l .
```

`testdata/openapi.json` est le document OpenAPI de l'API. Les tests vérifient chaque type de requête par rapport
à lui (méthode, chemin, noms de query, champs du corps, champs obligatoires), les noms JSON de chaque type de
réponse par rapport aux schémas de réponse ainsi que chaque valeur d'énumération, et échouent lorsque l'API
gagne une opération sans type de requête. Pour le mettre à jour :

```bash
curl -fsSL -A "go-dna-spec-update" -o testdata/openapi.json https://api.domainresellerapi.com/swagger/v1/swagger.json
go test ./...
```

Voir [AGENTS.md](AGENTS.md) pour les conventions de ce dépôt.

## Support

- Documentation de l'API : <https://www.domainnameapi.com/docs/>
- CLI et serveur MCP : <https://github.com/domainreseller/cli>
- Site web : <https://www.domainnameapi.com>

## Licence

Apache License 2.0. Voir [LICENSE](LICENSE).
