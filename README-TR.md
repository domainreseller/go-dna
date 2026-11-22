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

# go-dna: Domain Name API için Go SDK'sı

[![Go Reference](https://pkg.go.dev/badge/github.com/domainreseller/go-dna/v2.svg)](https://pkg.go.dev/github.com/domainreseller/go-dna/v2)
[![CI](https://github.com/domainreseller/go-dna/actions/workflows/ci.yml/badge.svg)](https://github.com/domainreseller/go-dna/actions/workflows/ci.yml)
[![License](https://img.shields.io/badge/license-Apache--2.0-blue.svg)](LICENSE)

[Domain Name API](https://www.domainnameapi.com) bayi REST API'si için tip güvenli Go istemcisi: alan adı sorgulama,
kayıt, yenileme ve transfer, DNS, DNSSEC, iletişim kayıtları, SSL sertifikaları, hosting, siparişler, ürün
kataloğu ve bakiyeniz.

- **API'nin 67 işleminin tamamı**, [`dna` CLI](https://github.com/domainreseller/cli) ile aynı şekilde gruplanmıştır:
  `dna domain info` komutu `client.Domains.Info`, `dna dns add` komutu `client.DNS.Add` karşılığıdır.
- **Tip güvenli sonuçlar ve enum'lar**: `d.Status == dna.DomainStatusPendingDelete`, `d.ExpirationDate.Before(...)`,
  `map[string]any` yok.
- **Her işlem bir istek tipidir**; `dna.BaseRequest` gömer ve herhangi bir şey gönderilmeden önce kendini
  doğrular. Kendi isteklerinizi de aynı şekilde yazabilirsiniz.
- **Varsayılan olarak güvenli**: ücretlendirilen istekler körlemesine yeniden denenmez, `WithReadOnly`
  değişiklikleri engeller, istemci hız sınırlarının altında kalacak şekilde kendini ayarlar ve eşzamanlı
  kullanıma uygundur.
- **Yalnızca standart kütüphane.** Go 1.23 veya üzeri.

```go
client, _ := dna.NewClient(resellerID, apiKey, dna.WithEnvironment(dna.OTE))

a, err := client.Domains.Search(ctx, "example.com")
if err == nil && a.Available() {
	fmt.Printf("%s is available for %.2f %s\n", a.DomainName, a.Price, a.Currency)
}
```

## İçindekiler

- [Kurulum](#kurulum)
- [Kimlik bilgileri ve ortamlar](#kimlik-bilgileri-ve-ortamlar)
- [Servisler](#servisler)
- [Kullanım](#kullanım)
- [API istekleri ve BaseRequest](#api-istekleri-ve-baserequest)
- [Hatalar](#hatalar)
- [Güvenlik ve güvenilirlik](#güvenlik-ve-güvenilirlik)
- [Client seçenekleri](#client-seçenekleri)
- [Kodunuzu test etme](#kodunuzu-test-etme)
- [v1'den geçiş](#v1den-geçiş)
- [Geliştirme](#geliştirme)

## Kurulum

```bash
go get github.com/domainreseller/go-dna/v2
```

```go
import dna "github.com/domainreseller/go-dna/v2"
```

Çalıştırılabilir programlar [examples](examples) klasöründedir: `quickstart`, `register`, `expiring`, `dns`, `ssl` ve
`custom-request`.

```bash
go run ./examples/quickstart example.com
```

## Kimlik bilgileri ve ortamlar

Bayi panelinden (**API ayarları**) iki API kimlik bilgisi almanız gerekir. Panel giriş bilgileri çalışmaz.

| Kimlik bilgisi | Görünümü                               |
|----------------|----------------------------------------|
| Reseller ID    | `xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx` |
| API key        | uzun bir gizli dize                    |

İki ortam vardır. **Test ortamı olan OTE ile başlayın**: kendi verisi ve kendi API anahtarı vardır; orada
yaptığınız hiçbir şey ücret doğurmaz ve canlı alan adlarına dokunmaz.

```go
// Test ortamı (OTE)
client, err := dna.NewClient(resellerID, oteAPIKey, dna.WithEnvironment(dna.OTE))

// Production (varsayılan): gerçek kayıtlar, gerçek ücretlendirme
client, err := dna.NewClient(resellerID, apiKey)
```

`NewClientFromEnv`, `dna` CLI ile aynı değişkenleri okur:

| Değişken          | Anlamı                                                        |
|-------------------|---------------------------------------------------------------|
| `DNA_RESELLER_ID` | reseller ID                                                   |
| `DNA_API_KEY`     | seçilen ortamın API anahtarı                                  |
| `DNA_ENV`         | `production` (varsayılan) veya `ote`                          |
| `DNA_TEST_MODE`   | `true` OTE'yi seçer (diğer DomainNameAPI SDK'larındaki gibi)  |
| `DNA_BASE_URL`    | özel bir temel URL, ör. bir proxy                             |

```go
client, err := dna.NewClientFromEnv(dna.WithUserAgent("my-panel/1.4"))
```

Yanlış bir anahtar veya ortam, ilk istekte `KindAuth` türünde ve bir ipucu içeren `*APIError` ile başarısız olur;
kontrol etmenin en ucuz yolu `client.Account.Get(ctx)` çağrısıdır.

## Servisler

| Servis            | CLI grubu     | İşlemler                                                                                                                                                          |
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

Metotlar, zorunlu girdileri argüman olarak alır (`Info(ctx, "example.com")`); bir işlemin isteğe bağlı girdileri
varsa bir istek struct'ı alır (`Register(ctx, dna.DomainRegisterRequest{...})`). Her alanı içeren tam başvuru
[pkg.go.dev](https://pkg.go.dev/github.com/domainreseller/go-dna/v2) adresindedir.

## Kullanım

Tüm kod parçaları `ctx := context.Background()` ve bir `client` olduğunu varsayar.

### Arama

```go
a, err := client.Domains.Search(ctx, "example.com")
// a.Available(), a.Price, a.Currency, a.IsPremium, a.IsDocumentRequired, a.Reason

results, err := client.Domains.BulkSearch(ctx, "example.com", "example.net", "example.org")
for _, r := range results {
	fmt.Println(r.DomainName, r.Status, r.Price)
}
```

`BulkSearch`, 30 alan adından uzun listeleri birkaç isteğe böler. Fiyatlar hesabın para birimiyle ya da
`dna.WithCurrency("TRY")` ile ayarlanan para birimiyle gelir.

### Kayıt

İletişim handle'ları ile (bir tane `Contacts.Create` ile oluşturun):

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
	Handles:     dna.SameHandle(handle), // veya dna.ContactHandles{Registrant: "DNA-1", Admin: "DNA-2", ...}
})
```

Satır içi iletişim bilgileri ile:

```go
res, err := client.Domains.RegisterWithContacts(ctx, dna.DomainRegisterWithContactsRequest{
	DomainName:  "example.com",
	NameServers: []string{"ns1.example.net", "ns2.example.net"},
	Contacts:    dna.SameContact(owner), // veya dna.DomainContacts{Registrant: &owner, Admin: &admin, ...}
})
```

İletişim bilgileri gönderilmeden önce düzenlenir: alanların baş ve sonundaki boşluklar silinir, ülke kodu büyük
harfe çevrilir ve telefon alanlarında yalnızca rakamlar tutulur (`"+90 (555) 123-45-67"` çalışır). Bazı TLD'ler
sicil (registry) öznitelikleri ister: bunları `client.Products.TLD(ctx, "com.tr")` ve `RequiredAttributes()`
ile bulun, `TLDAttributes` olarak iletin.

API yanıt verip isteği reddederse (`"success": false`), hata API'nin mesajını taşıyan bir
`*dna.OperationError` olur.

### Yenileme, transfer, geri alma

```go
check, err := client.Domains.RenewCheck(ctx, "example.com", 1) // check.Renewable, check.PremiumPrice
res, err := client.Domains.Renew(ctx, "example.com", 1)

tc, err := client.Domains.TransferCheck(ctx, "example.com", "auth-code") // tc.Transferable, tc.AuthCodeValid
res, err := client.Domains.Transfer(ctx, dna.DomainTransferRequest{DomainName: "example.com", AuthCode: "auth-code"})
st, err := client.Domains.TransferStatus(ctx, "example.com")

err = client.Domains.Restore(ctx, "example.com") // itfa (redemption) döneminde
```

### Alan adı ayrıntıları ve alan adı listesi

```go
d, err := client.Domains.Info(ctx, "example.com")
fmt.Println(d.Status, d.ExpirationDate, d.RemainingDays, d.Locked, d.NameServers, d.AuthCode)
if owner, ok := d.Contact(dna.ContactTypeRegistrant); ok {
	fmt.Println(owner.Email)
}

// Tek sayfa
page, err := client.Domains.List(ctx, dna.DomainListRequest{ListOptions: dna.ListOptions{Limit: 50}})

// Tüm alan adları; sayfalar sizin için getirilir
for d, err := range client.Domains.All(ctx, dna.DomainListRequest{
	Statuses: []dna.DomainStatus{dna.DomainStatusActive},
}) {
	if err != nil {
		return err
	}
	fmt.Println(d.DomainName, d.ExpirationDate)
}
```

### Kilit, gizlilik, name server'lar, child name server'lar

```go
err = client.Domains.Lock(ctx, "example.com")
err = client.Domains.Unlock(ctx, "example.com")
err = client.Domains.SetPrivacy(ctx, "example.com", true)
code, err := client.Domains.UpdateAuthCode(ctx, "example.com")

err = client.Domains.SetNameServers(ctx, dna.DomainNameServersSetRequest{
	DomainName:  "example.com",
	NameServers: []string{"ns1.example.net", "ns2.example.net"},
	// DNSSECMode: dna.DNSSECModeDisable, // yalnızca DNSSEC imzalı alan adları için
})

err = client.Domains.AddHost(ctx, "example.com", "ns1.example.com", "192.0.2.10", "2001:db8::10") // IP sürümü otomatik algılanır
err = client.Domains.DeleteHost(ctx, "example.com", "ns1.example.com")
```

### Bir alan adının iletişim kayıtları

```go
err = client.Domains.SetContactHandles(ctx, "example.com", dna.ContactHandles{Admin: "DNA-2", Tech: "DNA-2"})
err = client.Domains.UpdateContacts(ctx, "example.com", dna.DomainContacts{Billing: &billing})

details, err := client.Contacts.Get(ctx, "DNA-1")           // iletişim kaydı ve doğrulama durumu
result, err := client.Contacts.Update(ctx, "DNA-1", contact) // handle'ı kullanan tüm alan adlarını günceller
```

### DNS bölgesi

```go
err = client.DNS.Add(ctx, dna.DNSAddRequest{
	DomainName: "example.com", Name: "www", Type: dna.RecordTypeA, TTL: 600, Values: []string{"192.0.2.10"},
})
err = client.DNS.Update(ctx, dna.DNSUpdateRequest{ // kümedeki tüm değerleri değiştirir
	DomainName: "example.com", RecordName: "www", Type: dna.RecordTypeA, Values: []string{"192.0.2.20"},
})
err = client.DNS.Delete(ctx, dna.DNSDeleteRequest{
	DomainName: "example.com", Name: "www", Type: dna.RecordTypeA, Value: "192.0.2.20",
})

sets, err := client.DNS.List(ctx, "example.com")
for _, s := range sets {
	fmt.Println(s.Name, s.Type, s.TTL, s.Values())
}

res, err := client.DNS.Apply(ctx, "example.com") // alan adını platformun name server'larına geçirir
```

İsimler göreli (`www`), kök (`""` veya `"@"`) ya da tam nitelikli olabilir. Kayıt türleri burada büyük/küçük
harfe duyarsızdır (`"txt"`, `TXT` olarak gönderilir). Doğrulama, API'nin reddedeceği durumları yakalar: kökte
CNAME, üç parçası olmayan CAA değeri, önceliği olmayan MX değeri. Test ortamı her bölge çağrısına HTTP 500
yanıtı verir; bu yüzden bölge kodu yalnızca production'da denenebilir.

### DNSSEC

```go
ds, err := dna.ParseDS("2371 13 2 1F987CC6583E92DF0890718C42...") // DNS sağlayıcınızın gösterdiği haliyle
results, err := client.DNSSEC.Add(ctx, "example.com", ds)
info, err := client.DNSSEC.Get(ctx, "example.com") // info.Status, info.DSRecords
res, err := client.DNSSEC.Disable(ctx, "example.com")
```

### URL yönlendirme

```go
err = client.Domains.SetForwarding(ctx, "example.com", "https://www.example.org", dna.ForwardTypeStandard)
f, err := client.Domains.Forwarding(ctx, "example.com") // alan adında yönlendirme yoksa nil
err = client.Domains.DeleteForwarding(ctx, "example.com")
```

### Sicil belgeleri (.tr)

```go
doc, err := dna.DocumentFromFile(dna.DocumentTypeIdentityCard, "id.pdf")
res, err := client.Domains.UploadDocuments(ctx, "example.com.tr", doc)
page, err := client.Domains.Documents(ctx, dna.DomainDocumentsListRequest{DomainName: "example.com.tr"})
```

Dosyalar sizin için base64 ile kodlanır; biçim (PDF, JPG, PNG, DOC, DOCX, ODT) ve boyut (dosya başına 10 MB,
istek başına 50 MB) yüklemeden önce kontrol edilir.

### SSL sertifikaları

Bir alan adının birden fazla sertifikası olabilir; bu yüzden mevcut bir sertifika alan adıyla değil,
**sertifika kimliğiyle** (bir GUID) adreslenir. `Initialize` ve `Renew` bu kimliği döndürür (saklayın), `SSL.List`
ise onu alan adına göre bulur. Sertifika kimliği beklenen yere alan adı verilirse istek, hiçbir şey gönderilmeden
önce `*dna.InputError` ile reddedilir.

```go
csr, err := client.SSL.GenerateCSR(ctx, dna.SSLGenerateCSRRequest{
	DomainName: "example.com", Country: "TR", State: "Istanbul", Locality: "Istanbul",
	Organization: "Example Ltd", OrganizationUnit: "IT",
})
order, err := client.SSL.Initialize(ctx, "example.com", "PositiveSSL") // bakiyeden ücret keser
id := order.CertificateID                                              // saklayın: sonraki her çağrı bunu alır

_, err = client.SSL.Configure(ctx, dna.SSLConfigureRequest{ // 15 gün içinde
	CertificateID: id, CSR: csr.CSR, ApproverMail: "admin@example.com",
	AuthenticationType: dna.AuthenticationEmail, WebServerType: dna.WebServerApache, Contact: sslContact,
})
st, err := client.SSL.Status(ctx, id) // st.IsIssued olana kadar sorgulayın
cert, err := client.SSL.Content(ctx, id)
os.WriteFile("example.com.crt", []byte(cert.Chain()), 0o644)
```

`Initialize` veya `Renew` sonrasında 15 gün içinde yapılandırılmayan sertifika iptal edilir.

`SSL.List` sertifikalarınızı en yeniden eskiye doğru döndürür (`SSL.All` hepsinin üzerinde dolaşır).
`SSL.Approvers`, mevcut bir sertifika için veya satın almadan önce, e-posta doğrulamasının kabul ettiği adresleri
listeler:

```go
page, err := client.SSL.List(ctx, dna.SSLListRequest{DomainName: "example.com"})
for _, c := range page.Items {
	fmt.Println(c.CertificateID, c.Status, c.SubscriptionExpiresAt)
}

mails, err := client.SSL.Approvers(ctx, dna.SSLApproversRequest{CertificateID: id})
mails, err = client.SSL.Approvers(ctx, dna.SSLApproversRequest{DomainName: "example.com", PackageCode: "PositiveSSL"})
```

Bir sertifikayı yenilemek için `Renew` işlevini çağırın, ardından döndürdüğü yeni sertifikayı `Configure` ile
yapılandırın. `Renew` bakiyeden ücret keser ve abonelik bitimine en fazla 30 gün kala yapılabilir. İkinci bir
`Initialize` ile yenilemeyin: bu, satın alma fiyatı üzerinden ilgisiz yeni bir satın almadır ve eski sertifika hiçbir zaman
`Renewed` durumuna geçmez.

```go
renewal, err := client.SSL.Renew(ctx, id) // bakiyeden ücret keser
_, err = client.SSL.Configure(ctx, dna.SSLConfigureRequest{
	CertificateID: renewal.CertificateID, // yeni sertifika; eskisi kendi kimliğini korur
	CSR: newCSR.CSR, ApproverMail: "admin@example.com",
	AuthenticationType: dna.AuthenticationEmail, WebServerType: dna.WebServerApache, Contact: sslContact,
})
```

### Hosting

```go
packages, err := client.Hosting.Packages(ctx)
created, err := client.Hosting.Create(ctx, "example.com", "Linux-Starter") // bakiyeden ücret keser

ctx, cancel := context.WithTimeout(ctx, 30*time.Minute)
defer cancel()
st, err := client.Hosting.Wait(ctx, created.ServiceID) // aktif olana kadar sorgular
fmt.Println(st.PanelAddress, st.Username, st.Password)
```

Hâlâ hazırlanmakta olan bir hosting için `Create` işlevini bir daha çağırmayın: iki kez ücret alınır.

### Bakiye, fiyatlar ve siparişler

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

info, err := client.Products.Info(ctx, "com", dna.OrderRenew, 2) // 2 yıllık .com yenilemesinin fiyatı
order, err := client.Orders.Get(ctx, "ORDER-CODE")
```

## API istekleri ve BaseRequest

Her API işlemi bir istek tipidir: `DomainInfoRequest`, `DNSAddRequest`, `SSLConfigureRequest`, ... Hepsi tek bir
arayüzü uygular:

```go
type Request interface {
	Endpoint() Endpoint      // metot, yol ve güvenlik sınıfı
	Query() url.Values       // sorgu dizesi veya nil
	Body() any               // JSON gövdesi veya nil
	Validate() error         // herhangi bir şey gönderilmeden önce kontrol edilir
}
```

Go'da kalıtım yoktur; bu yüzden "temel sınıf", gömdüğünüz bir struct'tır. `dna.BaseRequest` varsayılanları sağlar
(sorgu yok, gövde yok, doğrulanacak bir şey yok); bir istek, bir metodu kendi tipinde tanımlayarak onu geçersiz
kılar. Yalnızca `Endpoint` için varsayılan yoktur. Bu paketteki bir istek şöyle görünür:

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

Servis metotları bunları sizin için oluşturur. `client.Do`, kendi tiplerinizi de içeren herhangi bir `Request`
gönderir; örneğin yalnızca ihtiyacınız olan alanları okumak ya da bu paketin henüz sarmalamadığı bir uç noktayı
çağırmak için:

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

İsteğiniz, yerleşik isteklerin sahip olduğu her şeye sahip olur: kimlik bilgileri, yeniden denemeler, hız
ayarı, hata normalleştirme, korelasyon ID'leri, izleme ve salt okunur kısıtlama. `client.NewHTTPRequest(ctx, req)`,
ücretlendirilen bir isteği gönderilmeden önce loglamak veya gözden geçirmek için tam HTTP isteğini göndermeden
oluşturur.

## Hatalar

| Hata                   | Ne zaman                                                   | Faydalı alanlar                                                  |
|------------------------|------------------------------------------------------------|------------------------------------------------------------------|
| `*dna.InputError`      | `Validate` isteği reddetti; **hiçbir şey gönderilmedi**    | `Request`, `Fields`                                              |
| `*dna.APIError`        | API 2xx olmayan bir durum koduyla yanıt verdi              | `StatusCode`, `Code`, `Message`, `Details`, `Fields`, `Data`, `CorrelationID`, `Hint()` |
| `*dna.OperationError`  | API 2xx ve `"success": false` ile yanıt verdi              | `Code`, `Message`, `CorrelationID`                               |
| `*dna.NetworkError`    | istek tamamlanmadı (DNS, TLS, zaman aşımı, bağlantı kopması) | `Err`, `Timeout()`                                             |
| `dna.ErrReadOnly`      | salt okunur bir istemci bir değişikliği engelledi; hiçbir şey gönderilmedi | `errors.Is` kullanın                         |

`dna.KindOf(err)` bunların hepsini sınıflandırır: `KindAuth`, `KindNotFound`, `KindValidation`, `KindRejected`,
`KindConflict`, `KindRateLimited`, `KindServer`, `KindNetwork`, `KindInvalidInput`, `KindReadOnly`. Yardımcı
fonksiyonlar sık sorulan soruları yanıtlar:

```go
_, err := client.Domains.Renew(ctx, "example.com", 1)
var apiErr *dna.APIError
switch {
case err == nil:
case dna.OutcomeUnknown(err):
	// Yenileme gerçekleşmiş olabilir. Tekrar denemeden önce alan adını kontrol edin.
case dna.IsInsufficientBalance(err):
case dna.IsNotFound(err):
case dna.HasCode(err, dna.CodePrivacyNotSupported):
case errors.As(err, &apiErr):
	log.Println(apiErr.Code, apiErr.Message, apiErr.Hint(), apiErr.CorrelationID)
}
```

`Message` üzerinden değil, `Code` üzerinden dallanın: mesajlar yerelleştirilir. API kodları
`Dna.DomainService:Domain:10007` gibi görünür; [hata başvurusuna](https://www.domainnameapi.com/docs/reference/errors.md) bakın.
Destekle iletişime geçerken `CorrelationID` değerini belirtin.

## Güvenlik ve güvenilirlik

- **Güvenlik sınıfları.** Her istek `Read`, `Write`, `Destructive` veya `Billable` sınıfındandır (`req.Endpoint().Safety`).
  Bazı POST ve PUT istekleri okumadır (`DomainSearchRequest`, `SSLStatusRequest`).
- **Salt okunur mod.** `dna.WithReadOnly()`, okuma olmayan her şeyi engeller; raporlama işleri ve production'ı
  güvenle keşfetmek içindir.
- **Yeniden denemeler.** 429 yanıtları her zaman, `Retry-After` değerine uyularak yeniden denenir: API bunları
  herhangi bir işlem yapmadan reddeder. Ağ hataları ve 502/503/504 yalnızca okumalar ve idempotent yazmalar için
  yeniden denenir; böylece bir kayıt veya yenileme asla iki kez gönderilmez. Ücretlendirilen bir istek
  başarısız olduktan sonra `dna.OutcomeUnknown(err)` değerini kontrol edin.
- **Hız ayarı.** İstemci, anahtar başına sınırların altında kalmak için istekleri aralıklandırır (saniyede 5 istek,
  saniyede 2 arama, saniyede bir bölge çağrısı). Hız ayarını kendiniz yapıyorsanız `dna.WithPacing(false)` ile kapatın.
- **Kota.** `Response.Quota`, yumuşak kota başlıklarını sunar. Sınır aşıldığında API yanıtları reddetmek yerine
  yavaşlatır (en fazla 5 sn); varsayılan 60 sn zaman aşımı buna yer bırakır.
- **Korelasyon ID'leri.** Her istek bir `X-Correlation-ID` taşır. API çağrılarını loglarınızla ilişkilendirmek için
  `dna.WithCorrelationID(ctx, id)` ile kendi değerinizi ayarlayın.
- **Gözlemlenebilirlik.** `dna.WithLogger(slog.Default())` her alışverişi loglar; `dna.WithTrace(func(dna.TraceEvent))`
  size her isteği, yanıtı ve yeniden denemeyi iletir; metrikler veya hata raporlama için kullanılabilir. API
  anahtarı bunların hiçbirinde yer almaz.
- **Eşzamanlılık.** Bir `Client` eşzamanlı kullanıma uygundur. Bir tane oluşturun ve paylaşın.
- **Toleranslı çözümleme.** Saat dilimi olmayan tarihler, `null`, `{}` yerine `[]`, sayısal veya adlandırılmış durumlar
  ve herhangi bir harf biçimindeki roller (`"registrant"`, `"ADMINISTRATIVE"`) çözümlenir.

## Client seçenekleri

| Seçenek                         | Etkisi                                                                                   |
|---------------------------------|------------------------------------------------------------------------------------------|
| `WithEnvironment(dna.OTE)`      | test ortamını kullanır (varsayılan `dna.Production`)                                     |
| `WithBaseURL(url)`              | istekleri bir proxy'ye veya test sunucusuna gönderir                                     |
| `WithHTTPClient(hc)`            | kendi `*http.Client`'ınız (transport, proxy, TLS); yönlendirmeler asla izlenmez          |
| `WithTimeout(d)`                | deneme başına zaman aşımı (varsayılan 60 sn)                                             |
| `WithUserAgent("app/1.0")`      | ürününüzü User-Agent'ın başına ekler                                                     |
| `WithCurrency("TRY")`           | arama ve katalog fiyatlarının para birimi                                                |
| `WithRetries(n)`                | yeniden deneme sayısı (varsayılan 2)                                                     |
| `WithPacing(false)`             | istemci tarafı hız ayarını kapatır                                                       |
| `WithReadOnly()`                | okuma olmayan her isteği engeller                                                        |
| `WithLogger(l)`                 | alışverişleri bir `*slog.Logger`'a loglar                                                |
| `WithTrace(fn)`                 | her isteği, yanıtı ve yeniden denemeyi gözlemler                                         |

## Kodunuzu test etme

İstemciyi bir `httptest.Server`'a yönlendirin; başka hiçbir şeyi mock'lamanız gerekmez:

```go
srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, `{"success":true,"info":{"domainName":"example.com","status":"AVAILABLE","price":11.99,"currency":"USD"}}`)
}))
defer srv.Close()

client, _ := dna.NewClient("reseller-id", "api-key", dna.WithBaseURL(srv.URL), dna.WithPacing(false))
a, err := client.Domains.Search(ctx, "example.com")
```

## v1'den geçiş

v2 yeni bir tasarımdır: `map[string]any` yerine tip güvenli sonuçlar, `{"result": "ERROR"}` zarfları yerine
hatalar ve php-dna metot kümesi yerine API'nin tamamı. Modül yolu `github.com/domainreseller/go-dna/v2` olarak
değişti; bu yüzden siz geçiş yapana kadar v1 çalışmaya devam eder.

| v1                                                  | v2                                                                     |
|-----------------------------------------------------|------------------------------------------------------------------------|
| `dna.New(id, key, true)`                            | `dna.NewClient(id, key, dna.WithEnvironment(dna.OTE))`                 |
| `res["result"] == "OK"`                             | `err == nil`                                                           |
| `GetResellerDetails`, `GetCurrentBalance`           | `Account.Get` (+ `Account.Balance("USD")`)                             |
| `CheckAvailability`                                 | `Domains.Search`, `Domains.BulkSearch`                                 |
| `GetList`                                           | `Domains.List`, `Domains.All`                                          |
| `GetTldList`                                        | `Products.TLDs`, `Products.AllTLDs`, `Products.TLD`                    |
| `GetDetails`, `GetContacts`, `SyncFromRegistry`     | `Domains.Info`                                                         |
| `RegisterWithContactInfo`                           | `Domains.RegisterWithContacts` (veya `Contacts.Create` + `Domains.Register`) |
| `Renew`, `Transfer`, `CheckTransfer`                | `Domains.Renew`, `Domains.Transfer`, `Domains.TransferCheck`           |
| `ModifyNameServer`                                  | `Domains.SetNameServers`                                               |
| `EnableTheftProtectionLock` / `Disable...`          | `Domains.Lock` / `Domains.Unlock`                                      |
| `ModifyPrivacyProtectionStatus`                     | `Domains.SetPrivacy`                                                   |
| `SaveContacts`                                      | `Domains.UpdateContacts`, `Domains.SetContactHandles`                  |
| `AddChildNameServer`, `ModifyChildNameServer`, `DeleteChildNameServer` | `Domains.AddHost`, `Domains.UpdateHost`, `Domains.DeleteHost` |
| `GetResourceRecords`                                | `DNS.List`                                                             |
| `AddResourceRecord`, `AddResourceRecordValue`       | `DNS.Add`                                                              |
| `EditResourceRecord`, `SetResourceRecordSet`, `ReplaceResourceRecordValue` | `DNS.Update`                                    |
| `DeleteResourceRecord`, `RemoveResourceRecordValue`, `DeleteResourceRecordSet` | `DNS.Delete` (her değer için bir kez)       |
| `ApplyResourceRecords`                              | `DNS.Apply`                                                            |
| `GetForwarding`, `SetForwarding`, `DeleteForwarding`| `Domains.Forwarding`, `Domains.SetForwarding`, `Domains.DeleteForwarding` |
| `CancelTransfer`, `ApproveTransfer`, `RejectTransfer` | yayımlanan API'de yok; hesabınızda varsa [kendi isteğinizi](#api-istekleri-ve-baserequest) gönderin |
| `IsTrTLD`, `ValidateContact`                        | `strings.HasSuffix(name, ".tr")`; doğrulama her istekte çalışır        |
| `GetRequestData`, `GetResponseData`, ...            | `WithTrace`, `WithLogger`, `client.Do`'dan dönen `Response`            |
| telemetry (`SetErrorReportingEnabled`, Sentry)      | kaldırıldı: süreciniz dışına hiçbir şey çıkmaz; `WithTrace` olaylarını kendi raporlama sisteminize iletin |
| `SetSSLVerification(false)`                         | kendi TLS ayarlarınızla `WithHTTPClient`                               |

## Geliştirme

```bash
go test ./...          # birim testleri; hiçbiri ağa erişmez
go vet ./...
gofmt -l .
```

`testdata/openapi.json`, API'nin OpenAPI belgesidir. Testler her istek tipini bu belgeye göre (metot, yol, sorgu
adları, gövde alanları, zorunlu alanlar), her yanıt tipinin JSON adlarını yanıt şemalarına göre ve her enum
değerini kontrol eder; API, istek tipi olmayan bir işlem kazandığında başarısız olur. Güncellemek için:

```bash
curl -fsSL -A "go-dna-spec-update" -o testdata/openapi.json https://api.domainresellerapi.com/swagger/v1/swagger.json
go test ./...
```

Bu deponun kuralları için [AGENTS.md](AGENTS.md) dosyasına bakın.

## Destek

- API dokümantasyonu: <https://www.domainnameapi.com/docs/>
- CLI ve MCP sunucusu: <https://github.com/domainreseller/cli>
- Web sitesi: <https://www.domainnameapi.com>

## Lisans

Apache License 2.0. [LICENSE](LICENSE) dosyasına bakın.
