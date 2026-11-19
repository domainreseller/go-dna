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

# go-dna: Domain Name API üçün Go SDK-sı

[![Go Reference](https://pkg.go.dev/badge/github.com/domainreseller/go-dna/v2.svg)](https://pkg.go.dev/github.com/domainreseller/go-dna/v2)
[![CI](https://github.com/domainreseller/go-dna/actions/workflows/ci.yml/badge.svg)](https://github.com/domainreseller/go-dna/actions/workflows/ci.yml)
[![License](https://img.shields.io/badge/license-Apache--2.0-blue.svg)](LICENSE)

[Domain Name API](https://www.domainnameapi.com) reseller REST API-si üçün tipli Go klienti: domen axtarışı,
qeydiyyat, yenilənmə və transfer, DNS, DNSSEC, kontaktlar, SSL sertifikatları, hosting, sifarişlər, məhsul
kataloqu və balansınız.

- **API-nin bütün 64 əməliyyatı**, [`dna` CLI](https://github.com/domainreseller/cli) ilə eyni şəkildə qruplaşdırılıb:
  `dna domain info` əmri `client.Domains.Info`, `dna dns add` əmri `client.DNS.Add` qarşılığıdır.
- **Tipli nəticələr və enum-lar**: `d.Status == dna.DomainStatusPendingDelete`, `d.ExpirationDate.Before(...)`,
  `map[string]any` yoxdur.
- **Hər əməliyyat bir sorğu tipidir**; `dna.BaseRequest`-i embed edir və hər hansı bir şey göndərilməzdən əvvəl
  özünü yoxlayır. Öz sorğularınızı da eyni qaydada yaza bilərsiniz.
- **Susmaya görə təhlükəsiz**: ödənişli sorğular kor-koranə təkrarlanmır, `WithReadOnly` dəyişiklikləri
  bloklayır, klient özünü sürət limitlərinin altında saxlayır və eyni vaxtda istifadə üçün təhlükəsizdir.
- **Yalnız standart kitabxana.** Go 1.23 və ya daha yeni.

```go
client, _ := dna.NewClient(resellerID, apiKey, dna.WithEnvironment(dna.OTE))

a, err := client.Domains.Search(ctx, "example.com")
if err == nil && a.Available() {
	fmt.Printf("%s is available for %.2f %s\n", a.DomainName, a.Price, a.Currency)
}
```

## Mündəricat

- [Quraşdırma](#quraşdırma)
- [Giriş məlumatları və mühitlər](#giriş-məlumatları-və-mühitlər)
- [Servislər](#servislər)
- [Nümunələr](#nümunələr)
- [Sorğular və BaseRequest](#sorğular-və-baserequest)
- [Xətalar](#xətalar)
- [Təhlükəsizlik və etibarlılıq](#təhlükəsizlik-və-etibarlılıq)
- [Client seçimləri](#client-seçimləri)
- [Kodunuzun test edilməsi](#kodunuzun-test-edilməsi)
- [v1-dən keçid](#v1-dən-keçid)
- [Layihənin hazırlanması](#layihənin-hazırlanması)

## Quraşdırma

```bash
go get github.com/domainreseller/go-dna/v2
```

```go
import dna "github.com/domainreseller/go-dna/v2"
```

İşlək proqramlar [examples](examples) qovluğundadır: `quickstart`, `register`, `expiring`, `dns`, `ssl` və
`custom-request`.

```bash
go run ./examples/quickstart example.com
```

## Giriş məlumatları və mühitlər

Reseller panelindən (**API settings**) iki API giriş məlumatı götürməlisiniz. Panelə giriş məlumatları işləmir.

| Giriş məlumatı | Necə görünür                           |
|----------------|----------------------------------------|
| Reseller ID    | `xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx` |
| API key        | uzun gizli sətir                       |

İki mühit var. **Test mühiti olan OTE ilə başlayın**: onun öz məlumatları və öz API açarı var; orada
etdiyiniz heç bir şey pul tutmur və canlı domenlərə toxunmur.

```go
// Test mühiti (OTE)
client, err := dna.NewClient(resellerID, oteAPIKey, dna.WithEnvironment(dna.OTE))

// Production (susmaya görə): real qeydiyyatlar, real ödənişlər
client, err := dna.NewClient(resellerID, apiKey)
```

`NewClientFromEnv`, `dna` CLI ilə eyni dəyişənləri oxuyur:

| Dəyişən           | Mənası                                                        |
|-------------------|---------------------------------------------------------------|
| `DNA_RESELLER_ID` | reseller ID                                                   |
| `DNA_API_KEY`     | seçilmiş mühitin API açarı                                    |
| `DNA_ENV`         | `production` (susmaya görə) və ya `ote`                       |
| `DNA_TEST_MODE`   | `true` OTE-ni seçir (digər DomainNameAPI SDK-larındakı kimi)  |
| `DNA_BASE_URL`    | xüsusi baza URL, məsələn proksi                               |

```go
client, err := dna.NewClientFromEnv(dna.WithUserAgent("my-panel/1.4"))
```

Yanlış açar və ya mühit ilk sorğuda `KindAuth` növündə və bir işarə (hint) ilə `*APIError` xətası verir;
yoxlamağın ən ucuz yolu `client.Account.Get(ctx)` çağırışıdır.

## Servislər

| Servis            | CLI qrupu     | Əməliyyatlar                                                                                                                                                      |
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

Metodlar məcburi girişləri arqument kimi qəbul edir (`Info(ctx, "example.com")`); əməliyyatın isteğe bağlı
girişləri varsa, sorğu struct-ı qəbul edir (`Register(ctx, dna.DomainRegisterRequest{...})`). Hər sahəni əhatə
edən tam sorğu-məlumat [pkg.go.dev](https://pkg.go.dev/github.com/domainreseller/go-dna/v2) ünvanındadır.

## Nümunələr

Bütün kod parçaları `ctx := context.Background()` və bir `client` olduğunu fərz edir.

### Axtarış

```go
a, err := client.Domains.Search(ctx, "example.com")
// a.Available(), a.Price, a.Currency, a.IsPremium, a.IsDocumentRequired, a.Reason

results, err := client.Domains.BulkSearch(ctx, "example.com", "example.net", "example.org")
for _, r := range results {
	fmt.Println(r.DomainName, r.Status, r.Price)
}
```

`BulkSearch` 30 domendən uzun siyahıları bir neçə sorğuya bölür. Qiymətlər hesabın valyutası ilə və ya
`dna.WithCurrency("TRY")` ilə təyin edilmiş valyuta ilə gəlir.

### Qeydiyyat

Kontakt handle-ları ilə (birini `Contacts.Create` ilə yaradın):

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
	Handles:     dna.SameHandle(handle), // və ya dna.ContactHandles{Registrant: "DNA-1", Admin: "DNA-2", ...}
})
```

Sətir daxili kontaktlarla:

```go
res, err := client.Domains.RegisterWithContacts(ctx, dna.DomainRegisterWithContactsRequest{
	DomainName:  "example.com",
	NameServers: []string{"ns1.example.net", "ns2.example.net"},
	Contacts:    dna.SameContact(owner), // və ya dna.DomainContacts{Registrant: &owner, Admin: &admin, ...}
})
```

Kontaktlar göndərilməzdən əvvəl səliqəyə salınır: sahələrin əvvəlindəki və sonundakı boşluqlar silinir, ölkə
kodu böyük hərflərə çevrilir, telefon sahələrində isə yalnız rəqəmlər saxlanılır (`"+90 (555) 123-45-67"`
işləyir). Bəzi TLD-lər reyestr (registry) atributları tələb edir: onları `client.Products.TLD(ctx, "com.tr")`
və `RequiredAttributes()` ilə tapın, `TLDAttributes` kimi ötürün.

API cavab verib sorğunu rədd edərsə (`"success": false`), xəta API-nin mesajını daşıyan
`*dna.OperationError` olur.

### Yenilənmə, transfer, bərpa

```go
check, err := client.Domains.RenewCheck(ctx, "example.com", 1) // check.Renewable, check.PremiumPrice
res, err := client.Domains.Renew(ctx, "example.com", 1)

tc, err := client.Domains.TransferCheck(ctx, "example.com", "auth-code") // tc.Transferable, tc.AuthCodeValid
res, err := client.Domains.Transfer(ctx, dna.DomainTransferRequest{DomainName: "example.com", AuthCode: "auth-code"})
st, err := client.Domains.TransferStatus(ctx, "example.com")

err = client.Domains.Restore(ctx, "example.com") // bərpa (redemption) dövründə
```

### Domen təfərrüatları və domen siyahısı

```go
d, err := client.Domains.Info(ctx, "example.com")
fmt.Println(d.Status, d.ExpirationDate, d.RemainingDays, d.Locked, d.NameServers, d.AuthCode)
if owner, ok := d.Contact(dna.ContactTypeRegistrant); ok {
	fmt.Println(owner.Email)
}

// Bir səhifə
page, err := client.Domains.List(ctx, dna.DomainListRequest{ListOptions: dna.ListOptions{Limit: 50}})

// Bütün domenlər; səhifələr sizin üçün gətirilir
for d, err := range client.Domains.All(ctx, dna.DomainListRequest{
	Statuses: []dna.DomainStatus{dna.DomainStatusActive},
}) {
	if err != nil {
		return err
	}
	fmt.Println(d.DomainName, d.ExpirationDate)
}
```

### Kilid, məxfilik, name server-lər, child name server-lər

```go
err = client.Domains.Lock(ctx, "example.com")
err = client.Domains.Unlock(ctx, "example.com")
err = client.Domains.SetPrivacy(ctx, "example.com", true)
code, err := client.Domains.UpdateAuthCode(ctx, "example.com")

err = client.Domains.SetNameServers(ctx, dna.DomainNameServersSetRequest{
	DomainName:  "example.com",
	NameServers: []string{"ns1.example.net", "ns2.example.net"},
	// DNSSECMode: dna.DNSSECModeDisable, // yalnız DNSSEC ilə imzalanmış domenlər üçün
})

err = client.Domains.AddHost(ctx, "example.com", "ns1.example.com", "192.0.2.10", "2001:db8::10") // IP versiyası avtomatik aşkarlanır
err = client.Domains.DeleteHost(ctx, "example.com", "ns1.example.com")
```

### Domenin kontaktları

```go
err = client.Domains.SetContactHandles(ctx, "example.com", dna.ContactHandles{Admin: "DNA-2", Tech: "DNA-2"})
err = client.Domains.UpdateContacts(ctx, "example.com", dna.DomainContacts{Billing: &billing})

details, err := client.Contacts.Get(ctx, "DNA-1")           // kontakt və onun doğrulama vəziyyəti
result, err := client.Contacts.Update(ctx, "DNA-1", contact) // handle-dan istifadə edən bütün domenləri yeniləyir
```

### DNS zonası

```go
err = client.DNS.Add(ctx, dna.DNSAddRequest{
	DomainName: "example.com", Name: "www", Type: dna.RecordTypeA, TTL: 600, Values: []string{"192.0.2.10"},
})
err = client.DNS.Update(ctx, dna.DNSUpdateRequest{ // çoxluqdakı bütün dəyərləri əvəz edir
	DomainName: "example.com", RecordName: "www", Type: dna.RecordTypeA, Values: []string{"192.0.2.20"},
})
err = client.DNS.Delete(ctx, dna.DNSDeleteRequest{
	DomainName: "example.com", Name: "www", Type: dna.RecordTypeA, Value: "192.0.2.20",
})

sets, err := client.DNS.List(ctx, "example.com")
for _, s := range sets {
	fmt.Println(s.Name, s.Type, s.TTL, s.Values())
}

res, err := client.DNS.Apply(ctx, "example.com") // domeni platformanın name server-lərinə keçirir
```

Adlar nisbi (`www`), kök (`""` və ya `"@"`) ya da tam ixtisaslaşmış ola bilər. Qeyd növləri burada böyük/kiçik
hərfə həssas deyil (`"txt"`, `TXT` kimi göndərilir). Doğrulama API-nin rədd edəcəyi halları tutur: kökdə
CNAME, üç hissəsi olmayan CAA dəyəri, prioriteti olmayan MX dəyəri. Test mühiti hər zona çağırışına HTTP 500
ilə cavab verir; buna görə zona kodunu yalnız production-da sınamaq mümkündür.

### DNSSEC

```go
ds, err := dna.ParseDS("2371 13 2 1F987CC6583E92DF0890718C42...") // DNS provayderinizin göstərdiyi şəkildə
results, err := client.DNSSEC.Add(ctx, "example.com", ds)
info, err := client.DNSSEC.Get(ctx, "example.com") // info.Status, info.DSRecords
res, err := client.DNSSEC.Disable(ctx, "example.com")
```

### URL yönləndirmə

```go
err = client.Domains.SetForwarding(ctx, "example.com", "https://www.example.org", dna.ForwardTypeStandard)
f, err := client.Domains.Forwarding(ctx, "example.com") // domendə yönləndirmə yoxdursa nil
err = client.Domains.DeleteForwarding(ctx, "example.com")
```

### Reyestr sənədləri (.tr)

```go
doc, err := dna.DocumentFromFile(dna.DocumentTypeIdentityCard, "id.pdf")
res, err := client.Domains.UploadDocuments(ctx, "example.com.tr", doc)
page, err := client.Domains.Documents(ctx, dna.DomainDocumentsListRequest{DomainName: "example.com.tr"})
```

Fayllar sizin üçün base64 ilə kodlaşdırılır; format (PDF, JPG, PNG, DOC, DOCX, ODT) və ölçü (hər fayl üçün 10 MB,
hər sorğu üçün 50 MB) yükləmədən əvvəl yoxlanılır.

### SSL sertifikatları

```go
csr, err := client.SSL.GenerateCSR(ctx, dna.SSLGenerateCSRRequest{
	DomainName: "example.com", Country: "TR", State: "Istanbul", Locality: "Istanbul",
	Organization: "Example Ltd", OrganizationUnit: "IT",
})
_, err = client.SSL.Initialize(ctx, "example.com", "PositiveSSL") // balansdan ödəniş çıxır
_, err = client.SSL.Configure(ctx, dna.SSLConfigureRequest{
	DomainName: "example.com", CSR: csr.CSR, ApproverMail: "admin@example.com",
	AuthenticationType: dna.AuthenticationEmail, WebServerType: dna.WebServerApache, Contact: sslContact,
})
st, err := client.SSL.Status(ctx, "example.com") // st.IsIssued olana qədər sorğulayın
cert, err := client.SSL.Content(ctx, "example.com")
os.WriteFile("example.com.crt", []byte(cert.Chain()), 0o644)
```

### Hosting

```go
packages, err := client.Hosting.Packages(ctx)
created, err := client.Hosting.Create(ctx, "example.com", "Linux-Starter") // balansdan ödəniş çıxır

ctx, cancel := context.WithTimeout(ctx, 30*time.Minute)
defer cancel()
st, err := client.Hosting.Wait(ctx, created.ServiceID) // aktiv olana qədər sorğulayır
fmt.Println(st.PanelAddress, st.Username, st.Password)
```

Hələ hazırlanmaqda olan hosting üçün `Create`-i bir daha çağırmayın: iki dəfə ödəniş çıxılır.

### Balans, qiymətlər və sifarişlər

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

info, err := client.Products.Info(ctx, "com", dna.OrderRenew, 2) // 2 illik .com yenilənməsinin qiyməti
order, err := client.Orders.Get(ctx, "ORDER-CODE")
```

## Sorğular və BaseRequest

Hər API əməliyyatı bir sorğu tipidir: `DomainInfoRequest`, `DNSAddRequest`, `SSLConfigureRequest`, ... Hamısı bir
interfeysi həyata keçirir:

```go
type Request interface {
	Endpoint() Endpoint      // metod, yol və təhlükəsizlik sinfi
	Query() url.Values       // query sətri və ya nil
	Body() any               // JSON gövdəsi və ya nil
	Validate() error         // hər hansı bir şey göndərilməzdən əvvəl yoxlanılır
}
```

Go-da varislik yoxdur; buna görə "baza sinif" embed etdiyiniz bir struct-dır. `dna.BaseRequest` susmaya görə
davranışları təmin edir (query yoxdur, gövdə yoxdur, doğrulanacaq bir şey yoxdur); sorğu metodu öz tipində
təyin etməklə onu əvəz edir. Yalnız `Endpoint` üçün susmaya görə variant yoxdur. Bu paketdəki sorğu belə görünür:

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

Servis metodları bunları sizin üçün yaradır. `client.Do`, öz tipləriniz də daxil olmaqla, istənilən `Request`-i
göndərir; məsələn, yalnız lazım olan sahələri oxumaq və ya bu paketin hələ əhatə etmədiyi endpoint-i
çağırmaq üçün:

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

Sorğunuz daxili sorğuların aldığı hər şeyi alır: giriş məlumatları, təkrar cəhdlər, sürət tənzimi, xətaların
normallaşdırılması, korrelyasiya ID-ləri, izləmə və yalnız oxumaq məhdudiyyəti. `client.NewHTTPRequest(ctx, req)`
ödənişli sorğunu göndərilməzdən əvvəl loglamaq və ya nəzərdən keçirmək üçün dəqiq HTTP sorğusunu göndərmədən
qurur.

## Xətalar

| Xəta                   | Nə vaxt                                                    | Faydalı sahələr                                                  |
|------------------------|------------------------------------------------------------|------------------------------------------------------------------|
| `*dna.InputError`      | `Validate` sorğunu rədd etdi; **heç nə göndərilmədi**      | `Request`, `Fields`                                              |
| `*dna.APIError`        | API 2xx olmayan status ilə cavab verdi                     | `StatusCode`, `Code`, `Message`, `Details`, `Fields`, `Data`, `CorrelationID`, `Hint()` |
| `*dna.OperationError`  | API 2xx və `"success": false` ilə cavab verdi              | `Code`, `Message`, `CorrelationID`                               |
| `*dna.NetworkError`    | sorğu tamamlanmadı (DNS, TLS, vaxt aşımı, bağlantının kəsilməsi) | `Err`, `Timeout()`                                         |
| `dna.ErrReadOnly`      | yalnız oxumaq rejimindəki klient dəyişikliyi blokladı; heç nə göndərilmədi | `errors.Is` istifadə edin                    |

`dna.KindOf(err)` bunların hamısını təsnif edir: `KindAuth`, `KindNotFound`, `KindValidation`, `KindRejected`,
`KindConflict`, `KindRateLimited`, `KindServer`, `KindNetwork`, `KindInvalidInput`, `KindReadOnly`. Köməkçi
funksiyalar tez-tez verilən suallara cavab verir:

```go
_, err := client.Domains.Renew(ctx, "example.com", 1)
var apiErr *dna.APIError
switch {
case err == nil:
case dna.OutcomeUnknown(err):
	// Yenilənmə həyata keçmiş ola bilər. Yenidən cəhd etməzdən əvvəl domeni yoxlayın.
case dna.IsInsufficientBalance(err):
case dna.IsNotFound(err):
case dna.HasCode(err, dna.CodePrivacyNotSupported):
case errors.As(err, &apiErr):
	log.Println(apiErr.Code, apiErr.Message, apiErr.Hint(), apiErr.CorrelationID)
}
```

`Message` üzrə yox, `Code` üzrə budaqlandırın: mesajlar lokallaşdırılır. API kodları
`Dna.DomainService:Domain:10007` kimi görünür; [xəta sorğu-məlumatına](https://www.domainnameapi.com/docs/reference/errors.md) baxın.
Dəstək xidmətinə müraciət edərkən `CorrelationID` dəyərini qeyd edin.

## Təhlükəsizlik və etibarlılıq

- **Təhlükəsizlik sinifləri.** Hər sorğu `Read`, `Write`, `Destructive` və ya `Billable` sinfindədir (`req.Endpoint().Safety`).
  Bəzi POST və PUT sorğuları oxumadır (`DomainSearchRequest`, `SSLStatusRequest`).
- **Yalnız oxumaq rejimi.** `dna.WithReadOnly()` oxuma olmayan hər şeyi bloklayır; hesabat işləri və production-u
  təhlükəsiz araşdırmaq üçündür.
- **Təkrar cəhdlər.** 429 cavabları həmişə, `Retry-After` nəzərə alınmaqla təkrarlanır: API onları heç bir
  əməliyyat etmədən rədd edir. Şəbəkə xətaları və 502/503/504 yalnız oxumalar və idempotent yazmalar üçün
  təkrarlanır; beləliklə qeydiyyat və ya yenilənmə heç vaxt iki dəfə göndərilmir. Ödənişli sorğu uğursuz
  olduqdan sonra `dna.OutcomeUnknown(err)` dəyərini yoxlayın.
- **Sürət tənzimi.** Klient açar başına limitlərin altında qalmaq üçün sorğular arasında fasilə qoyur (saniyədə 5 sorğu,
  saniyədə 2 axtarış, saniyədə bir zona çağırışı). Sürəti özünüz tənzimləyirsinizsə, `dna.WithPacing(false)` ilə söndürün.
- **Kvota.** `Response.Quota` yumşaq kvota başlıqlarını təqdim edir. Limit aşıldıqda API cavabları rədd etmək əvəzinə
  yavaşladır (ən çoxu 5 san); susmaya görə 60 san vaxt aşımı buna yer buraxır.
- **Korrelyasiya ID-ləri.** Hər sorğu `X-Correlation-ID` daşıyır. API çağırışlarını loglarınızla əlaqələndirmək üçün
  `dna.WithCorrelationID(ctx, id)` ilə öz dəyərinizi təyin edin.
- **Müşahidə oluna bilmə.** `dna.WithLogger(slog.Default())` hər mübadiləni loglayır; `dna.WithTrace(func(dna.TraceEvent))`
  sizə hər sorğunu, cavabı və təkrar cəhdi ötürür; metrikalar və ya xəta hesabatı üçün istifadə edilə bilər. API
  açarı onların heç birində yer almır.
- **Paralellik.** `Client` eyni vaxtda istifadə üçün təhlükəsizdir. Bir dənə yaradın və paylaşın.
- **Tolerant dekodlama.** Saat qurşağı olmayan tarixlər, `null`, `{}` əvəzinə `[]`, rəqəmli və ya adlı statuslar
  və istənilən hərf registrində rollar (`"registrant"`, `"ADMINISTRATIVE"`) dekodlanır.

## Client seçimləri

| Seçim                           | Təsiri                                                                                   |
|---------------------------------|------------------------------------------------------------------------------------------|
| `WithEnvironment(dna.OTE)`      | test mühitindən istifadə edir (susmaya görə `dna.Production`)                            |
| `WithBaseURL(url)`              | sorğuları proksiyə və ya test serverinə göndərir                                         |
| `WithHTTPClient(hc)`            | öz `*http.Client`-iniz (transport, proksi, TLS); yönləndirmələr heç vaxt izlənmir        |
| `WithTimeout(d)`                | cəhd başına vaxt aşımı (susmaya görə 60 san)                                             |
| `WithUserAgent("app/1.0")`      | məhsulunuzu User-Agent-in əvvəlinə əlavə edir                                            |
| `WithCurrency("TRY")`           | axtarış və kataloq qiymətlərinin valyutası                                               |
| `WithRetries(n)`                | təkrar cəhd sayı (susmaya görə 2)                                                        |
| `WithPacing(false)`             | klient tərəfi sürət tənzimini söndürür                                                   |
| `WithReadOnly()`                | oxuma olmayan hər sorğunu bloklayır                                                      |
| `WithLogger(l)`                 | mübadilələri `*slog.Logger`-ə loglayır                                                   |
| `WithTrace(fn)`                 | hər sorğunu, cavabı və təkrar cəhdi müşahidə edir                                        |

## Kodunuzun test edilməsi

Klienti `httptest.Server`-ə yönəldin; başqa heç nəyi mock etmək lazım deyil:

```go
srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, `{"success":true,"info":{"domainName":"example.com","status":"AVAILABLE","price":11.99,"currency":"USD"}}`)
}))
defer srv.Close()

client, _ := dna.NewClient("reseller-id", "api-key", dna.WithBaseURL(srv.URL), dna.WithPacing(false))
a, err := client.Domains.Search(ctx, "example.com")
```

## v1-dən keçid

v2 yeni dizayndır: `map[string]any` əvəzinə tipli nəticələr, `{"result": "ERROR"}` zərfləri əvəzinə
xətalar və php-dna metod dəsti əvəzinə bütün API. Modul yolu `github.com/domainreseller/go-dna/v2` olaraq
dəyişdi; buna görə siz keçid edənə qədər v1 işləməyə davam edir.

| v1                                                  | v2                                                                     |
|-----------------------------------------------------|------------------------------------------------------------------------|
| `dna.New(id, key, true)`                            | `dna.NewClient(id, key, dna.WithEnvironment(dna.OTE))`                 |
| `res["result"] == "OK"`                             | `err == nil`                                                           |
| `GetResellerDetails`, `GetCurrentBalance`           | `Account.Get` (+ `Account.Balance("USD")`)                             |
| `CheckAvailability`                                 | `Domains.Search`, `Domains.BulkSearch`                                 |
| `GetList`                                           | `Domains.List`, `Domains.All`                                          |
| `GetTldList`                                        | `Products.TLDs`, `Products.AllTLDs`, `Products.TLD`                    |
| `GetDetails`, `GetContacts`, `SyncFromRegistry`     | `Domains.Info`                                                         |
| `RegisterWithContactInfo`                           | `Domains.RegisterWithContacts` (və ya `Contacts.Create` + `Domains.Register`) |
| `Renew`, `Transfer`, `CheckTransfer`                | `Domains.Renew`, `Domains.Transfer`, `Domains.TransferCheck`           |
| `ModifyNameServer`                                  | `Domains.SetNameServers`                                               |
| `EnableTheftProtectionLock` / `Disable...`          | `Domains.Lock` / `Domains.Unlock`                                      |
| `ModifyPrivacyProtectionStatus`                     | `Domains.SetPrivacy`                                                   |
| `SaveContacts`                                      | `Domains.UpdateContacts`, `Domains.SetContactHandles`                  |
| `AddChildNameServer`, `ModifyChildNameServer`, `DeleteChildNameServer` | `Domains.AddHost`, `Domains.UpdateHost`, `Domains.DeleteHost` |
| `GetResourceRecords`                                | `DNS.List`                                                             |
| `AddResourceRecord`, `AddResourceRecordValue`       | `DNS.Add`                                                              |
| `EditResourceRecord`, `SetResourceRecordSet`, `ReplaceResourceRecordValue` | `DNS.Update`                                    |
| `DeleteResourceRecord`, `RemoveResourceRecordValue`, `DeleteResourceRecordSet` | `DNS.Delete` (hər dəyər üçün bir dəfə)      |
| `ApplyResourceRecords`                              | `DNS.Apply`                                                            |
| `GetForwarding`, `SetForwarding`, `DeleteForwarding`| `Domains.Forwarding`, `Domains.SetForwarding`, `Domains.DeleteForwarding` |
| `CancelTransfer`, `ApproveTransfer`, `RejectTransfer` | dərc edilmiş API-də yoxdur; hesabınızda varsa [öz sorğunuzu](#sorğular-və-baserequest) göndərin |
| `IsTrTLD`, `ValidateContact`                        | `strings.HasSuffix(name, ".tr")`; doğrulama hər sorğuda işləyir        |
| `GetRequestData`, `GetResponseData`, ...            | `WithTrace`, `WithLogger`, `client.Do`-dan qayıdan `Response`          |
| telemetry (`SetErrorReportingEnabled`, Sentry)      | silindi: prosesinizdən kənara heç nə çıxmır; `WithTrace` hadisələrini öz hesabat sisteminizə yönləndirin |
| `SetSSLVerification(false)`                         | öz TLS tənzimləmələrinizlə `WithHTTPClient`                            |

## Layihənin hazırlanması

```bash
go test ./...          # vahid testlər; heç biri şəbəkəyə çıxmır
go vet ./...
gofmt -l .
```

`testdata/openapi.json` API-nin OpenAPI sənədidir. Testlər hər sorğu tipini bu sənədə qarşı (metod, yol, query
adları, gövdə sahələri, məcburi sahələr), hər cavab tipinin JSON adlarını cavab sxemlərinə qarşı və hər enum
dəyərini yoxlayır; API-yə sorğu tipi olmayan yeni əməliyyat əlavə olunduqda uğursuz olur. Yeniləmək üçün:

```bash
curl -fsSL -A "go-dna-spec-update" -o testdata/openapi.json https://api.domainresellerapi.com/swagger/v1/swagger.json
go test ./...
```

Bu depozitarinin qaydaları üçün [AGENTS.md](AGENTS.md) faylına baxın.

## Dəstək

- API sənədləşməsi: <https://www.domainnameapi.com/docs/>
- CLI və MCP serveri: <https://github.com/domainreseller/cli>
- Veb sayt: <https://www.domainnameapi.com>

## Lisenziya

Apache License 2.0. [LICENSE](LICENSE) faylına baxın.
