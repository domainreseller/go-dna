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

# go-dna: حزمة SDK للغة Go لواجهة Domain Name API

[![Go Reference](https://pkg.go.dev/badge/github.com/domainreseller/go-dna/v2.svg)](https://pkg.go.dev/github.com/domainreseller/go-dna/v2)
[![CI](https://github.com/domainreseller/go-dna/actions/workflows/ci.yml/badge.svg)](https://github.com/domainreseller/go-dna/actions/workflows/ci.yml)
[![License](https://img.shields.io/badge/license-Apache--2.0-blue.svg)](LICENSE)

عميل Go مُنمَّط لواجهة REST API الخاصة بالموزّعين في [Domain Name API](https://www.domainnameapi.com): البحث عن
النطاقات وتسجيلها وتجديدها ونقلها، وDNS، وDNSSEC، وجهات الاتصال، وشهادات SSL، والاستضافة، والطلبات، وكتالوج
المنتجات، ورصيدك.

- **جميع عمليات API السبع والستين**، مجمّعة بالطريقة نفسها المتّبعة في [`dna` CLI](https://github.com/domainreseller/cli):
  الأمر `dna domain info` يقابله `client.Domains.Info`، والأمر `dna dns add` يقابله `client.DNS.Add`.
- **نتائج وقيم تعداد مُنمَّطة**: `d.Status == dna.DomainStatusPendingDelete` و `d.ExpirationDate.Before(...)`،
  بلا `map[string]any`.
- **كل عملية هي نوع طلب** يضمّن `dna.BaseRequest` ويتحقق من صحة نفسه قبل إرسال أي شيء. يمكنك كتابة طلباتك
  الخاصة بالطريقة نفسها.
- **آمنة افتراضيًا**: لا تُعاد محاولة الطلبات المُحاسَب عليها بشكل أعمى، و`WithReadOnly` يمنع التغييرات، ويضبط
  العميل وتيرة طلباته بنفسه تحت حدود المعدّل، ويصلح للاستخدام المتزامن بأمان.
- **المكتبة القياسية فقط.** يتطلب Go 1.23 أو أحدث.

```go
client, _ := dna.NewClient(resellerID, apiKey, dna.WithEnvironment(dna.OTE))

a, err := client.Domains.Search(ctx, "example.com")
if err == nil && a.Available() {
	fmt.Printf("%s is available for %.2f %s\n", a.DomainName, a.Price, a.Currency)
}
```

## المحتويات

- [التثبيت](#التثبيت)
- [بيانات الاعتماد والبيئات](#بيانات-الاعتماد-والبيئات)
- [الخدمات](#الخدمات)
- [الاستخدام](#الاستخدام)
- [الطلبات و BaseRequest](#الطلبات-و-baserequest)
- [الأخطاء](#الأخطاء)
- [الأمان والموثوقية](#الأمان-والموثوقية)
- [خيارات العميل](#خيارات-العميل)
- [اختبار شيفرتك](#اختبار-شيفرتك)
- [الانتقال من v1](#الانتقال-من-v1)
- [التطوير](#التطوير)

## التثبيت

```bash
go get github.com/domainreseller/go-dna/v2
```

```go
import dna "github.com/domainreseller/go-dna/v2"
```

تجد برامج قابلة للتشغيل في [examples](examples): `quickstart` و `register` و `expiring` و `dns` و `ssl` و
`custom-request`.

```bash
go run ./examples/quickstart example.com
```

## بيانات الاعتماد والبيئات

تحتاج إلى بيانَي اعتماد لـ API من لوحة الموزّع (**API settings**). لا تعمل بيانات الدخول إلى اللوحة.

| بيان الاعتماد | شكله                                   |
|---------------|----------------------------------------|
| Reseller ID   | `xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx` |
| API key       | سلسلة سرية طويلة                       |

هناك بيئتان. **ابدأ بـ OTE**، وهي بيئة الاختبار: لها بياناتها الخاصة ومفتاح API خاص بها،
ولا يكلّف أي شيء تفعله فيها مالًا ولا يمسّ النطاقات الحية.

```go
// بيئة الاختبار (OTE)
client, err := dna.NewClient(resellerID, oteAPIKey, dna.WithEnvironment(dna.OTE))

// الإنتاج (الافتراضي): تسجيلات حقيقية ورسوم حقيقية
client, err := dna.NewClient(resellerID, apiKey)
```

تقرأ `NewClientFromEnv` المتغيرات نفسها التي يقرأها `dna` CLI:

| المتغير           | المعنى                                                        |
|-------------------|---------------------------------------------------------------|
| `DNA_RESELLER_ID` | معرّف الموزّع                                                 |
| `DNA_API_KEY`     | مفتاح API للبيئة المحددة                                      |
| `DNA_ENV`         | `production` (الافتراضي) أو `ote`                             |
| `DNA_TEST_MODE`   | القيمة `true` تختار OTE (كما في حزم DomainNameAPI SDK الأخرى) |
| `DNA_BASE_URL`    | عنوان URL أساسي مخصص، مثل وسيط (proxy)                        |

```go
client, err := dna.NewClientFromEnv(dna.WithUserAgent("my-panel/1.4"))
```

يفشل المفتاح الخاطئ أو البيئة الخاطئة عند أول طلب بخطأ `*APIError` من النوع `KindAuth` مع تلميح؛
و`client.Account.Get(ctx)` هي أرخص طريقة للتحقق.

## الخدمات

| الخدمة            | مجموعة CLI    | العمليات                                                                                                                                                          |
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

تستقبل الدوال المدخلات المطلوبة كوسائط (`Info(ctx, "example.com")`) أو، حين تكون للعملية مدخلات اختيارية،
بنية طلب (`Register(ctx, dna.DomainRegisterRequest{...})`). المرجع الكامل مع كل الحقول
على [pkg.go.dev](https://pkg.go.dev/github.com/domainreseller/go-dna/v2).

## الاستخدام

تفترض جميع المقتطفات وجود `ctx := context.Background()` و`client`.

### البحث

```go
a, err := client.Domains.Search(ctx, "example.com")
// a.Available(), a.Price, a.Currency, a.IsPremium, a.IsDocumentRequired, a.Reason

results, err := client.Domains.BulkSearch(ctx, "example.com", "example.net", "example.org")
for _, r := range results {
	fmt.Println(r.DomainName, r.Status, r.Price)
}
```

تقسّم `BulkSearch` القوائم التي تزيد على 30 نطاقًا إلى عدة طلبات. تُعرض الأسعار بعملة الحساب، أو
بالعملة التي تُحدَّد بـ `dna.WithCurrency("TRY")`.

### التسجيل

باستخدام معرّفات جهات الاتصال (أنشئ واحدًا بـ `Contacts.Create`):

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
	Handles:     dna.SameHandle(handle), // أو dna.ContactHandles{Registrant: "DNA-1", Admin: "DNA-2", ...}
})
```

باستخدام جهات اتصال مضمَّنة:

```go
res, err := client.Domains.RegisterWithContacts(ctx, dna.DomainRegisterWithContactsRequest{
	DomainName:  "example.com",
	NameServers: []string{"ns1.example.net", "ns2.example.net"},
	Contacts:    dna.SameContact(owner), // أو dna.DomainContacts{Registrant: &owner, Admin: &admin, ...}
})
```

تُنظَّف جهات الاتصال قبل الإرسال: تُقصّ المسافات الزائدة من الحقول، ويُحوَّل رمز الدولة إلى أحرف كبيرة، وتُبقي
حقول الهاتف أرقامها فقط (يصلح `"+90 (555) 123-45-67"`). تحتاج بعض النطاقات العليا (TLD) إلى سمات خاصة بالسجل:
اعثر عليها عبر `client.Products.TLD(ctx, "com.tr")` ودالتها `RequiredAttributes()`، ومرّرها في `TLDAttributes`.

إذا ردّت API لكنها رفضت الطلب (`"success": false`)، فإن الخطأ هو `*dna.OperationError` ومعه
رسالة API.

### التجديد والنقل والاستعادة

```go
check, err := client.Domains.RenewCheck(ctx, "example.com", 1) // check.Renewable, check.PremiumPrice
res, err := client.Domains.Renew(ctx, "example.com", 1)

tc, err := client.Domains.TransferCheck(ctx, "example.com", "auth-code") // tc.Transferable, tc.AuthCodeValid
res, err := client.Domains.Transfer(ctx, dna.DomainTransferRequest{DomainName: "example.com", AuthCode: "auth-code"})
st, err := client.Domains.TransferStatus(ctx, "example.com")

err = client.Domains.Restore(ctx, "example.com") // خلال فترة الاسترداد (redemption)
```

### تفاصيل النطاق وقائمة النطاقات

```go
d, err := client.Domains.Info(ctx, "example.com")
fmt.Println(d.Status, d.ExpirationDate, d.RemainingDays, d.Locked, d.NameServers, d.AuthCode)
if owner, ok := d.Contact(dna.ContactTypeRegistrant); ok {
	fmt.Println(owner.Email)
}

// صفحة واحدة
page, err := client.Domains.List(ctx, dna.DomainListRequest{ListOptions: dna.ListOptions{Limit: 50}})

// كل النطاقات؛ تُجلب الصفحات نيابةً عنك
for d, err := range client.Domains.All(ctx, dna.DomainListRequest{
	Statuses: []dna.DomainStatus{dna.DomainStatusActive},
}) {
	if err != nil {
		return err
	}
	fmt.Println(d.DomainName, d.ExpirationDate)
}
```

### القفل والخصوصية وخوادم الأسماء وخوادم الأسماء الفرعية

```go
err = client.Domains.Lock(ctx, "example.com")
err = client.Domains.Unlock(ctx, "example.com")
err = client.Domains.SetPrivacy(ctx, "example.com", true)
code, err := client.Domains.UpdateAuthCode(ctx, "example.com")

err = client.Domains.SetNameServers(ctx, dna.DomainNameServersSetRequest{
	DomainName:  "example.com",
	NameServers: []string{"ns1.example.net", "ns2.example.net"},
	// DNSSECMode: dna.DNSSECModeDisable, // فقط للنطاقات الموقّعة بـ DNSSEC
})

err = client.Domains.AddHost(ctx, "example.com", "ns1.example.com", "192.0.2.10", "2001:db8::10") // يُكتشف إصدار IP تلقائيًا
err = client.Domains.DeleteHost(ctx, "example.com", "ns1.example.com")
```

### جهات اتصال النطاق

```go
err = client.Domains.SetContactHandles(ctx, "example.com", dna.ContactHandles{Admin: "DNA-2", Tech: "DNA-2"})
err = client.Domains.UpdateContacts(ctx, "example.com", dna.DomainContacts{Billing: &billing})

details, err := client.Contacts.Get(ctx, "DNA-1")           // جهة الاتصال وحالة التحقق منها
result, err := client.Contacts.Update(ctx, "DNA-1", contact) // يحدّث كل نطاق يستخدم المعرّف
```

### منطقة DNS

```go
err = client.DNS.Add(ctx, dna.DNSAddRequest{
	DomainName: "example.com", Name: "www", Type: dna.RecordTypeA, TTL: 600, Values: []string{"192.0.2.10"},
})
err = client.DNS.Update(ctx, dna.DNSUpdateRequest{ // يستبدل كل قيم المجموعة
	DomainName: "example.com", RecordName: "www", Type: dna.RecordTypeA, Values: []string{"192.0.2.20"},
})
err = client.DNS.Delete(ctx, dna.DNSDeleteRequest{
	DomainName: "example.com", Name: "www", Type: dna.RecordTypeA, Value: "192.0.2.20",
})

sets, err := client.DNS.List(ctx, "example.com")
for _, s := range sets {
	fmt.Println(s.Name, s.Type, s.TTL, s.Values())
}

res, err := client.DNS.Apply(ctx, "example.com") // تحويل النطاق إلى خوادم أسماء المنصة
```

يمكن أن تكون الأسماء نسبية (`www`) أو الجذر (`""` أو `"@"`) أو مؤهَّلة بالكامل. أنواع السجلات هنا غير حساسة
لحالة الأحرف (يُرسَل `"txt"` على أنه `TXT`). يلتقط التحقق ما سترفضه API: سجل CNAME عند الجذر، وقيمة CAA
من دون ثلاثة أجزاء، وقيمة MX من دون أولوية. تردّ بيئة الاختبار على كل استدعاء للمنطقة بحالة HTTP
500، لذا لا يمكن تجربة شيفرة المناطق إلا في الإنتاج.

### DNSSEC

```go
ds, err := dna.ParseDS("2371 13 2 1F987CC6583E92DF0890718C42...") // كما يعرضه مزوّد DNS لديك
results, err := client.DNSSEC.Add(ctx, "example.com", ds)
info, err := client.DNSSEC.Get(ctx, "example.com") // info.Status, info.DSRecords
res, err := client.DNSSEC.Disable(ctx, "example.com")
```

### إعادة توجيه URL

```go
err = client.Domains.SetForwarding(ctx, "example.com", "https://www.example.org", dna.ForwardTypeStandard)
f, err := client.Domains.Forwarding(ctx, "example.com") // nil عندما لا يكون للنطاق إعادة توجيه
err = client.Domains.DeleteForwarding(ctx, "example.com")
```

### مستندات السجل (.tr)

```go
doc, err := dna.DocumentFromFile(dna.DocumentTypeIdentityCard, "id.pdf")
res, err := client.Domains.UploadDocuments(ctx, "example.com.tr", doc)
page, err := client.Domains.Documents(ctx, dna.DomainDocumentsListRequest{DomainName: "example.com.tr"})
```

تُرمَّز الملفات بـ base64 نيابةً عنك؛ ويُتحقق من الصيغة (PDF وJPG وPNG وDOC وDOCX وODT) والحجم (10 MB لكل ملف، و50 MB
لكل طلب) قبل الرفع.

### شهادات SSL

قد يكون للنطاق الواحد أكثر من شهادة، ولذلك يُشار إلى الشهادة القائمة بـ**معرّف الشهادة** (GUID) لا باسم النطاق.
تُعيد `Initialize` و`Renew` هذا المعرّف (احتفظ به)، وتعثر عليه `SSL.List` انطلاقًا من اسم النطاق. وإذا مُرِّر اسم نطاق
في موضع معرّف الشهادة رُفض الطلب بخطأ `*dna.InputError` قبل إرسال أي شيء.

```go
csr, err := client.SSL.GenerateCSR(ctx, dna.SSLGenerateCSRRequest{
	DomainName: "example.com", Country: "TR", State: "Istanbul", Locality: "Istanbul",
	Organization: "Example Ltd", OrganizationUnit: "IT",
})
order, err := client.SSL.Initialize(ctx, "example.com", "PositiveSSL") // يخصم من الرصيد
id := order.CertificateID                                              // احتفظ به: كل استدعاء لاحق يتطلبه

_, err = client.SSL.Configure(ctx, dna.SSLConfigureRequest{ // خلال 15 يومًا
	CertificateID: id, CSR: csr.CSR, ApproverMail: "admin@example.com",
	AuthenticationType: dna.AuthenticationEmail, WebServerType: dna.WebServerApache, Contact: sslContact,
})
st, err := client.SSL.Status(ctx, id) // استعلم دوريًا حتى تصبح st.IsIssued صحيحة
cert, err := client.SSL.Content(ctx, id)
os.WriteFile("example.com.crt", []byte(cert.Chain()), 0o644)
```

تُلغى الشهادة التي لا تُهيَّأ خلال 15 يومًا من `Initialize` أو `Renew`.

تُعيد `SSL.List` شهاداتك من الأحدث إلى الأقدم (وتمرّ `SSL.All` عليها جميعًا). وتسرد `SSL.Approvers` العناوين التي
يقبلها التحقق عبر البريد الإلكتروني، لشهادة قائمة أو قبل الشراء:

```go
page, err := client.SSL.List(ctx, dna.SSLListRequest{DomainName: "example.com"})
for _, c := range page.Items {
	fmt.Println(c.CertificateID, c.Status, c.SubscriptionExpiresAt)
}

mails, err := client.SSL.Approvers(ctx, dna.SSLApproversRequest{CertificateID: id})
mails, err = client.SSL.Approvers(ctx, dna.SSLApproversRequest{DomainName: "example.com", PackageCode: "PositiveSSL"})
```

لتجديد شهادة، استدعِ `Renew` ثم هيّئ بـ`Configure` الشهادة الجديدة التي تُعيدها. تخصم `Renew` من الرصيد، وهي متاحة
ابتداءً من 30 يومًا قبل انتهاء الاشتراك. لا تجدّد باستدعاء `Initialize` مرة ثانية: فذلك شراء مستقل بسعر الشراء، ولن
تنتقل الشهادة القديمة أبدًا إلى الحالة `Renewed`.

```go
renewal, err := client.SSL.Renew(ctx, id) // يخصم من الرصيد
_, err = client.SSL.Configure(ctx, dna.SSLConfigureRequest{
	CertificateID: renewal.CertificateID, // الشهادة الجديدة؛ القديمة تحتفظ بمعرّفها
	CSR: newCSR.CSR, ApproverMail: "admin@example.com",
	AuthenticationType: dna.AuthenticationEmail, WebServerType: dna.WebServerApache, Contact: sslContact,
})
```

### الاستضافة

```go
packages, err := client.Hosting.Packages(ctx)
created, err := client.Hosting.Create(ctx, "example.com", "Linux-Starter") // يخصم من الرصيد

ctx, cancel := context.WithTimeout(ctx, 30*time.Minute)
defer cancel()
st, err := client.Hosting.Wait(ctx, created.ServiceID) // يستعلم دوريًا حتى يصبح نشطًا
fmt.Println(st.PanelAddress, st.Username, st.Password)
```

لا تستدعِ `Create` مرة أخرى لاستضافة ما زالت قيد التجهيز: فهذا يخصم الرسوم مرتين.

### الرصيد والأسعار والطلبات

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

info, err := client.Products.Info(ctx, "com", dna.OrderRenew, 2) // سعر تجديد .com لمدة سنتين
order, err := client.Orders.Get(ctx, "ORDER-CODE")
```

## الطلبات و BaseRequest

كل عملية في API هي نوع طلب: `DomainInfoRequest` و `DNSAddRequest` و `SSLConfigureRequest` وغيرها. وجميعها
تنفّذ واجهة واحدة:

```go
type Request interface {
	Endpoint() Endpoint      // الطريقة والمسار وفئة الأمان
	Query() url.Values       // سلسلة الاستعلام، أو nil
	Body() any               // جسم JSON، أو nil
	Validate() error         // يُفحص قبل إرسال أي شيء
}
```

لا توجد وراثة في Go، لذا فإن «الصنف الأساسي» هو بنية تضمّنها في بنيتك. توفّر `dna.BaseRequest` القيم
الافتراضية (لا استعلام، ولا جسم، ولا شيء للتحقق منه)؛ ويتجاوز الطلب دالةً ما بتعريفها على نوعه الخاص. وحدها
`Endpoint` ليس لها تنفيذ افتراضي. يبدو طلب من هذه الحزمة هكذا:

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

تبني دوال الخدمات هذه الطلبات نيابةً عنك. ترسل `client.Do` أي `Request`، بما في ذلك أنواعك الخاصة، مثلًا
لقراءة الحقول التي تحتاجها فقط أو لاستدعاء نقطة نهاية لم تغلّفها هذه الحزمة بعد:

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

يحصل طلبك على كل ما تحصل عليه الطلبات المضمَّنة: بيانات الاعتماد، وإعادة المحاولة، وضبط الوتيرة، وتوحيد
الأخطاء، ومعرّفات الارتباط، والتتبع، وفرض وضع القراءة فقط. تبني `client.NewHTTPRequest(ctx, req)` طلب HTTP
الفعلي من دون إرساله، لتسجّل طلبًا مُحاسَبًا عليه أو تراجعه قبل خروجه.

## الأخطاء

| الخطأ                  | متى                                                        | حقول مفيدة                                                       |
|------------------------|------------------------------------------------------------|------------------------------------------------------------------|
| `*dna.InputError`      | رفضت `Validate` الطلب؛ **لم يُرسَل شيء**                   | `Request`, `Fields`                                              |
| `*dna.APIError`        | ردّت API بحالة غير 2xx                                     | `StatusCode`, `Code`, `Message`, `Details`, `Fields`, `Data`, `CorrelationID`, `Hint()` |
| `*dna.OperationError`  | ردّت API بحالة 2xx مع `"success": false`                   | `Code`, `Message`, `CorrelationID`                               |
| `*dna.NetworkError`    | لم يكتمل الطلب (DNS أو TLS أو انتهاء المهلة أو إعادة ضبط الاتصال) | `Err`, `Timeout()`                                        |
| `dna.ErrReadOnly`      | منع عميل للقراءة فقط تغييرًا؛ لم يُرسَل شيء                | استخدم `errors.Is`                                               |

تصنّف `dna.KindOf(err)` أيًّا منها: `KindAuth` و `KindNotFound` و `KindValidation` و `KindRejected` و
`KindConflict` و `KindRateLimited` و `KindServer` و `KindNetwork` و `KindInvalidInput` و `KindReadOnly`. وتجيب
الدوال المساعدة عن الأسئلة الشائعة:

```go
_, err := client.Domains.Renew(ctx, "example.com", 1)
var apiErr *dna.APIError
switch {
case err == nil:
case dna.OutcomeUnknown(err):
	// ربما تمّ التجديد. تحقّق من النطاق قبل المحاولة مرة أخرى.
case dna.IsInsufficientBalance(err):
case dna.IsNotFound(err):
case dna.HasCode(err, dna.CodePrivacyNotSupported):
case errors.As(err, &apiErr):
	log.Println(apiErr.Code, apiErr.Message, apiErr.Hint(), apiErr.CorrelationID)
}
```

فرّع على `Code` لا على `Message`: فالرسائل مترجَمة محليًا. تبدو رموز API هكذا
`Dna.DomainService:Domain:10007`؛ راجع [مرجع الأخطاء](https://www.domainnameapi.com/docs/reference/errors.md).
اذكر `CorrelationID` عند التواصل مع الدعم.

## الأمان والموثوقية

- **فئات الأمان.** كل طلب هو `Read` أو `Write` أو `Destructive` أو `Billable` (`req.Endpoint().Safety`).
  بعض طلبات POST و PUT هي عمليات قراءة (`DomainSearchRequest` و `SSLStatusRequest`).
- **وضع القراءة فقط.** يمنع `dna.WithReadOnly()` كل ما ليس قراءة، لمهام إعداد التقارير ولاستكشاف
  بيئة الإنتاج بأمان.
- **إعادة المحاولة.** تُعاد محاولة استجابات 429 دائمًا مع احترام `Retry-After`: إذ ترفضها API قبل أن تنفّذ
  أي شيء. أما أخطاء الشبكة والاستجابات 502/503/504 فلا تُعاد محاولتها إلا للقراءات والكتابات المتماثلة القوى (idempotent)، فلا
  يُرسَل تسجيل أو تجديد مرتين أبدًا. بعد فشل طلب مُحاسَب عليه، افحص `dna.OutcomeUnknown(err)`.
- **ضبط الوتيرة.** يباعد العميل بين الطلبات ليبقى تحت الحدود المفروضة لكل مفتاح (5 طلبات/ثانية، وبحثان/ثانية،
  واستدعاء واحد للمنطقة كل ثانية). عطّله بـ `dna.WithPacing(false)` إذا كنت تضبط الوتيرة بنفسك.
- **الحصة.** يكشف `Response.Quota` ترويسات الحصة المرنة. عند تجاوز الحد تبطئ API الاستجابات (حتى
  5 ثوانٍ) بدلًا من رفضها؛ وتترك المهلة الافتراضية البالغة 60 ثانية متسعًا لذلك.
- **معرّفات الارتباط.** يحمل كل طلب ترويسة `X-Correlation-ID`. عيّن معرّفك بنفسك عبر
  `dna.WithCorrelationID(ctx, id)` لربط استدعاءات API بسجلاتك.
- **قابلية المراقبة.** يسجّل `dna.WithLogger(slog.Default())` كل تبادل؛ ويسلّمك `dna.WithTrace(func(dna.TraceEvent))`
  كل طلب واستجابة ومحاولة إعادة، لأغراض المقاييس أو الإبلاغ عن الأخطاء. ولا يظهر مفتاح API فيها أبدًا.
- **التزامن.** الكائن `Client` آمن للاستخدام المتزامن. أنشئ واحدًا وشاركه.
- **فكّ ترميز متسامح.** تُفكّ بنجاح التواريخ بلا منطقة زمنية، و`null`، و`[]` بدل `{}`، والحالات الرقمية أو المسمّاة،
  والأدوار بأي حالة أحرف (`"registrant"` و `"ADMINISTRATIVE"`).

## خيارات العميل

| الخيار                          | التأثير                                                                                  |
|---------------------------------|------------------------------------------------------------------------------------------|
| `WithEnvironment(dna.OTE)`      | استخدام بيئة الاختبار (الافتراضي `dna.Production`)                                       |
| `WithBaseURL(url)`              | إرسال الطلبات إلى وسيط (proxy) أو خادم اختبار                                            |
| `WithHTTPClient(hc)`            | `*http.Client` الخاص بك (النقل، الوسيط، TLS)؛ ولا تُتبَّع إعادة التوجيه أبدًا            |
| `WithTimeout(d)`                | مهلة كل محاولة (الافتراضي 60 s)                                                          |
| `WithUserAgent("app/1.0")`      | وضع اسم منتجك قبل User-Agent                                                             |
| `WithCurrency("TRY")`           | عملة أسعار البحث والكتالوج                                                               |
| `WithRetries(n)`                | عدد إعادات المحاولة (الافتراضي 2)                                                        |
| `WithPacing(false)`             | تعطيل ضبط الوتيرة من جهة العميل                                                          |
| `WithReadOnly()`                | منع كل طلب ليس قراءة                                                                     |
| `WithLogger(l)`                 | تسجيل التبادلات في `*slog.Logger`                                                        |
| `WithTrace(fn)`                 | مراقبة كل طلب واستجابة ومحاولة إعادة                                                     |

## اختبار شيفرتك

وجّه العميل إلى `httptest.Server`؛ ولا حاجة إلى محاكاة أي شيء آخر:

```go
srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, `{"success":true,"info":{"domainName":"example.com","status":"AVAILABLE","price":11.99,"currency":"USD"}}`)
}))
defer srv.Close()

client, _ := dna.NewClient("reseller-id", "api-key", dna.WithBaseURL(srv.URL), dna.WithPacing(false))
a, err := client.Domains.Search(ctx, "example.com")
```

## الانتقال من v1

الإصدار v2 تصميم جديد: نتائج مُنمَّطة بدل `map[string]any`، وأخطاء (errors) بدل أغلفة `{"result": "ERROR"}`،
وAPI كاملة بدل مجموعة دوال php-dna. تغيّر مسار الوحدة إلى
`github.com/domainreseller/go-dna/v2`، لذا يستمر v1 في العمل إلى أن تنتقل.

| v1                                                  | v2                                                                     |
|-----------------------------------------------------|------------------------------------------------------------------------|
| `dna.New(id, key, true)`                            | `dna.NewClient(id, key, dna.WithEnvironment(dna.OTE))`                 |
| `res["result"] == "OK"`                             | `err == nil`                                                           |
| `GetResellerDetails`, `GetCurrentBalance`           | `Account.Get` (+ `Account.Balance("USD")`)                             |
| `CheckAvailability`                                 | `Domains.Search`, `Domains.BulkSearch`                                 |
| `GetList`                                           | `Domains.List`, `Domains.All`                                          |
| `GetTldList`                                        | `Products.TLDs`, `Products.AllTLDs`, `Products.TLD`                    |
| `GetDetails`, `GetContacts`, `SyncFromRegistry`     | `Domains.Info`                                                         |
| `RegisterWithContactInfo`                           | `Domains.RegisterWithContacts` (أو `Contacts.Create` + `Domains.Register`) |
| `Renew`, `Transfer`, `CheckTransfer`                | `Domains.Renew`, `Domains.Transfer`, `Domains.TransferCheck`           |
| `ModifyNameServer`                                  | `Domains.SetNameServers`                                               |
| `EnableTheftProtectionLock` / `Disable...`          | `Domains.Lock` / `Domains.Unlock`                                      |
| `ModifyPrivacyProtectionStatus`                     | `Domains.SetPrivacy`                                                   |
| `SaveContacts`                                      | `Domains.UpdateContacts`, `Domains.SetContactHandles`                  |
| `AddChildNameServer`, `ModifyChildNameServer`, `DeleteChildNameServer` | `Domains.AddHost`, `Domains.UpdateHost`, `Domains.DeleteHost` |
| `GetResourceRecords`                                | `DNS.List`                                                             |
| `AddResourceRecord`, `AddResourceRecordValue`       | `DNS.Add`                                                              |
| `EditResourceRecord`, `SetResourceRecordSet`, `ReplaceResourceRecordValue` | `DNS.Update`                                    |
| `DeleteResourceRecord`, `RemoveResourceRecordValue`, `DeleteResourceRecordSet` | `DNS.Delete` (مرة لكل قيمة)                 |
| `ApplyResourceRecords`                              | `DNS.Apply`                                                            |
| `GetForwarding`, `SetForwarding`, `DeleteForwarding`| `Domains.Forwarding`, `Domains.SetForwarding`, `Domains.DeleteForwarding` |
| `CancelTransfer`, `ApproveTransfer`, `RejectTransfer` | غير موجودة في API المنشورة؛ أرسل [طلبًا من تصميمك](#الطلبات-و-baserequest) إن كان حسابك يتيحها |
| `IsTrTLD`, `ValidateContact`                        | `strings.HasSuffix(name, ".tr")`؛ ويجري التحقق مع كل طلب               |
| `GetRequestData`, `GetResponseData`, ...            | `WithTrace`, `WithLogger`, و`Response` الناتج عن `client.Do`           |
| القياس عن بُعد (`SetErrorReportingEnabled`, Sentry) | أُزيل: لا شيء يغادر عمليتك؛ مرّر أحداث `WithTrace` إلى نظام الإبلاغ الخاص بك |
| `SetSSLVerification(false)`                         | `WithHTTPClient` مع إعدادات TLS الخاصة بك                              |

## التطوير

```bash
go test ./...          # اختبارات الوحدة؛ لا يلمس أيٌّ منها الشبكة
go vet ./...
gofmt -l .
```

الملف `testdata/openapi.json` هو مستند OpenAPI الخاص بـ API. تفحص الاختبارات كل نوع طلب مقابله (الطريقة،
والمسار، وأسماء الاستعلام، وحقول الجسم، والحقول المطلوبة)، وأسماء JSON لكل نوع استجابة مقابل مخططات
الاستجابة، وكل قيمة تعداد، وتفشل عندما تكتسب API عملية جديدة بلا نوع طلب لها. لتحديثه:

```bash
curl -fsSL -A "go-dna-spec-update" -o testdata/openapi.json https://api.domainresellerapi.com/swagger/v1/swagger.json
go test ./...
```

راجع [AGENTS.md](AGENTS.md) لمعرفة أعراف هذا المستودع.

## الدعم

- توثيق API: <https://www.domainnameapi.com/docs/>
- CLI وخادم MCP: <https://github.com/domainreseller/cli>
- الموقع: <https://www.domainnameapi.com>

## الترخيص

Apache License 2.0. راجع [LICENSE](LICENSE).
