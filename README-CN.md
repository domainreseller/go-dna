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

# go-dna：Domain Name API 的 Go SDK

[![Go Reference](https://pkg.go.dev/badge/github.com/domainreseller/go-dna/v2.svg)](https://pkg.go.dev/github.com/domainreseller/go-dna/v2)
[![CI](https://github.com/domainreseller/go-dna/actions/workflows/ci.yml/badge.svg)](https://github.com/domainreseller/go-dna/actions/workflows/ci.yml)
[![License](https://img.shields.io/badge/license-Apache--2.0-blue.svg)](LICENSE)

[Domain Name API](https://www.domainnameapi.com) 经销商 REST API 的强类型 Go 客户端：域名搜索、
注册、续费与转入、DNS、DNSSEC、联系人、SSL 证书、主机托管、订单、产品
目录以及账户余额。

- **全部 67 个 API 操作**，分组方式与 [`dna` CLI](https://github.com/domainreseller/cli) 一致：
  `dna domain info` 对应 `client.Domains.Info`，`dna dns add` 对应 `client.DNS.Add`。
- **类型化的结果与枚举**：`d.Status == dna.DomainStatusPendingDelete`、`d.ExpirationDate.Before(...)`，
  不再使用 `map[string]any`。
- **每个操作都是一个请求类型**，内嵌 `dna.BaseRequest`，并在发送任何内容之前先对自身进行校验。
  你也可以用同样的方式编写自己的请求。
- **默认安全**：计费请求绝不会被盲目重试，`WithReadOnly` 会阻止任何变更，客户端会自行
  控制请求节奏以遵守速率限制，并且可并发安全使用。
- **仅依赖标准库。** 需要 Go 1.23 或更高版本。

```go
client, _ := dna.NewClient(resellerID, apiKey, dna.WithEnvironment(dna.OTE))

a, err := client.Domains.Search(ctx, "example.com")
if err == nil && a.Available() {
	fmt.Printf("%s is available for %.2f %s\n", a.DomainName, a.Price, a.Currency)
}
```

## 目录

- [安装](#安装)
- [凭据与环境](#凭据与环境)
- [服务](#服务)
- [用法](#用法)
- [请求与 BaseRequest](#请求与-baserequest)
- [错误](#错误)
- [安全与可靠性](#安全与可靠性)
- [客户端选项](#客户端选项)
- [测试你的代码](#测试你的代码)
- [从 v1 迁移](#从-v1-迁移)
- [开发](#开发)

## 安装

```bash
go get github.com/domainreseller/go-dna/v2
```

```go
import dna "github.com/domainreseller/go-dna/v2"
```

可运行的程序位于 [examples](examples)：`quickstart`、`register`、`expiring`、`dns`、`ssl` 和
`custom-request`。

```bash
go run ./examples/quickstart example.com
```

## 凭据与环境

你需要从经销商面板（**API settings**）获取两个 API 凭据。面板的登录信息无法使用。

| 凭据        | 形式                                   |
|-------------|----------------------------------------|
| Reseller ID | `xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx` |
| API key     | 一长串密钥字符串                       |

共有两个环境。**请先从 OTE 开始**，即测试环境：它有独立的数据和独立的 API key，
在其中的任何操作都不会产生费用，也不会影响线上域名。

```go
// 测试环境 (OTE)
client, err := dna.NewClient(resellerID, oteAPIKey, dna.WithEnvironment(dna.OTE))

// 生产环境（默认）：真实注册，真实扣费
client, err := dna.NewClient(resellerID, apiKey)
```

`NewClientFromEnv` 读取与 `dna` CLI 相同的变量：

| 变量              | 含义                                                          |
|-------------------|---------------------------------------------------------------|
| `DNA_RESELLER_ID` | 经销商 ID                                                     |
| `DNA_API_KEY`     | 所选环境的 API key                                            |
| `DNA_ENV`         | `production`（默认）或 `ote`                                  |
| `DNA_TEST_MODE`   | `true` 选择 OTE（与其他 DomainNameAPI SDK 一致）              |
| `DNA_BASE_URL`    | 自定义基础 URL，例如代理                                      |
| `DNA_APP`         | 使用 SDK 的应用，作为 `X-App` 发送                            |
| `DNA_APP_VERSION` | 该应用的版本，作为 `X-Version` 发送                            |

```go
client, err := dna.NewClientFromEnv(dna.WithUserAgent("my-panel/1.4"))
```

密钥或环境错误时，第一次请求就会失败，返回类型为 `KindAuth` 的 `*APIError` 并附带提示；
`client.Account.Get(ctx)` 是最省事的检查方式。

## 服务

| 服务              | CLI 分组      | 操作                                                                                                                                                              |
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

方法将必需的输入作为参数传入（`Info(ctx, "example.com")`）；当某个操作带有可选输入时，
则传入一个请求结构体（`Register(ctx, dna.DomainRegisterRequest{...})`）。包含所有字段的完整参考
见 [pkg.go.dev](https://pkg.go.dev/github.com/domainreseller/go-dna/v2)。

## 用法

以下所有代码片段均假定已有 `ctx := context.Background()` 和一个 `client`。

### 搜索

```go
a, err := client.Domains.Search(ctx, "example.com")
// a.Available(), a.Price, a.Currency, a.IsPremium, a.IsDocumentRequired, a.Reason

results, err := client.Domains.BulkSearch(ctx, "example.com", "example.net", "example.org")
for _, r := range results {
	fmt.Println(r.DomainName, r.Status, r.Price)
}
```

`BulkSearch` 会把超过 30 个域名的列表拆分为多个请求。价格使用账户的币种，或使用
`dna.WithCurrency("TRY")` 设置的币种。

### 注册

使用联系人句柄（用 `Contacts.Create` 创建）：

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
	Handles:     dna.SameHandle(handle), // 或 dna.ContactHandles{Registrant: "DNA-1", Admin: "DNA-2", ...}
})
```

使用内联联系人：

```go
res, err := client.Domains.RegisterWithContacts(ctx, dna.DomainRegisterWithContactsRequest{
	DomainName:  "example.com",
	NameServers: []string{"ns1.example.net", "ns2.example.net"},
	Contacts:    dna.SameContact(owner), // 或 dna.DomainContacts{Registrant: &owner, Admin: &admin, ...}
})
```

联系人信息在发送前会先整理：字段会去除首尾空白，国家代码转为大写，电话字段只保留
数字（`"+90 (555) 123-45-67"` 可以直接使用）。部分 TLD 需要注册局属性：用
`client.Products.TLD(ctx, "com.tr")` 及其 `RequiredAttributes()` 查找，并通过 `TLDAttributes` 传入。

如果 API 有响应但拒绝了请求（`"success": false`），返回的错误是 `*dna.OperationError`，
其中带有 API 的消息。

### 续费、转入、恢复

```go
check, err := client.Domains.RenewCheck(ctx, "example.com", 1) // check.Renewable, check.PremiumPrice
res, err := client.Domains.Renew(ctx, "example.com", 1)

tc, err := client.Domains.TransferCheck(ctx, "example.com", "auth-code") // tc.Transferable, tc.AuthCodeValid
res, err := client.Domains.Transfer(ctx, dna.DomainTransferRequest{DomainName: "example.com", AuthCode: "auth-code"})
st, err := client.Domains.TransferStatus(ctx, "example.com")

err = client.Domains.Restore(ctx, "example.com") // 在赎回期内
```

### 域名详情与域名列表

```go
d, err := client.Domains.Info(ctx, "example.com")
fmt.Println(d.Status, d.ExpirationDate, d.RemainingDays, d.Locked, d.NameServers, d.AuthCode)
if owner, ok := d.Contact(dna.ContactTypeRegistrant); ok {
	fmt.Println(owner.Email)
}

// 单页
page, err := client.Domains.List(ctx, dna.DomainListRequest{ListOptions: dna.ListOptions{Limit: 50}})

// 所有域名；分页由 SDK 自动获取
for d, err := range client.Domains.All(ctx, dna.DomainListRequest{
	Statuses: []dna.DomainStatus{dna.DomainStatusActive},
}) {
	if err != nil {
		return err
	}
	fmt.Println(d.DomainName, d.ExpirationDate)
}
```

### 锁定、隐私保护、域名服务器、子域名服务器

```go
err = client.Domains.Lock(ctx, "example.com")
err = client.Domains.Unlock(ctx, "example.com")
err = client.Domains.SetPrivacy(ctx, "example.com", true)
code, err := client.Domains.UpdateAuthCode(ctx, "example.com")

err = client.Domains.SetNameServers(ctx, dna.DomainNameServersSetRequest{
	DomainName:  "example.com",
	NameServers: []string{"ns1.example.net", "ns2.example.net"},
	// DNSSECMode: dna.DNSSECModeDisable, // 仅适用于已启用 DNSSEC 签名的域名
})

err = client.Domains.AddHost(ctx, "example.com", "ns1.example.com", "192.0.2.10", "2001:db8::10") // 自动识别 IP 版本
err = client.Domains.DeleteHost(ctx, "example.com", "ns1.example.com")
```

### 域名的联系人

```go
err = client.Domains.SetContactHandles(ctx, "example.com", dna.ContactHandles{Admin: "DNA-2", Tech: "DNA-2"})
err = client.Domains.UpdateContacts(ctx, "example.com", dna.DomainContacts{Billing: &billing})

details, err := client.Contacts.Get(ctx, "DNA-1")           // 联系人及其验证状态
result, err := client.Contacts.Update(ctx, "DNA-1", contact) // 更新所有使用该句柄的域名
```

### DNS 区域

```go
err = client.DNS.Add(ctx, dna.DNSAddRequest{
	DomainName: "example.com", Name: "www", Type: dna.RecordTypeA, TTL: 600, Values: []string{"192.0.2.10"},
})
err = client.DNS.Update(ctx, dna.DNSUpdateRequest{ // 替换该记录集的所有值
	DomainName: "example.com", RecordName: "www", Type: dna.RecordTypeA, Values: []string{"192.0.2.20"},
})
err = client.DNS.Delete(ctx, dna.DNSDeleteRequest{
	DomainName: "example.com", Name: "www", Type: dna.RecordTypeA, Value: "192.0.2.20",
})

sets, err := client.DNS.List(ctx, "example.com")
for _, s := range sets {
	fmt.Println(s.Name, s.Type, s.TTL, s.Values())
}

res, err := client.DNS.Apply(ctx, "example.com") // 将域名切换到平台的域名服务器
```

名称可以是相对名称（`www`）、根域（`""` 或 `"@"`）或完全限定名称。此处记录类型不区分
大小写（`"txt"` 会作为 `TXT` 发送）。校验会拦截 API 会拒绝的内容：根域上的 CNAME、
不足三个部分的 CAA 值、缺少优先级的 MX 值。测试环境对每个区域调用都返回 HTTP
500，因此区域相关代码只能在生产环境中试用。

### DNSSEC

```go
ds, err := dna.ParseDS("2371 13 2 1F987CC6583E92DF0890718C42...") // 与你的 DNS 提供商显示的格式一致
results, err := client.DNSSEC.Add(ctx, "example.com", ds)
info, err := client.DNSSEC.Get(ctx, "example.com") // info.Status, info.DSRecords
res, err := client.DNSSEC.Disable(ctx, "example.com")
```

### URL 转发

```go
err = client.Domains.SetForwarding(ctx, "example.com", "https://www.example.org", dna.ForwardTypeStandard)
f, err := client.Domains.Forwarding(ctx, "example.com") // 域名没有转发时为 nil
err = client.Domains.DeleteForwarding(ctx, "example.com")
```

### 注册局文档 (.tr)

```go
doc, err := dna.DocumentFromFile(dna.DocumentTypeIdentityCard, "id.pdf")
res, err := client.Domains.UploadDocuments(ctx, "example.com.tr", doc)
page, err := client.Domains.Documents(ctx, dna.DomainDocumentsListRequest{DomainName: "example.com.tr"})
```

文件会自动进行 base64 编码；上传前会检查格式（PDF、JPG、PNG、DOC、DOCX、ODT）和大小（每个文件 10 MB，每次
请求 50 MB）。

### SSL 证书

一个域名可以有多张证书，因此已有的证书通过**证书 ID**（一个 GUID）来指定，而不是域名。`Initialize` 和 `Renew`
会返回该 ID（请保存），`SSL.List` 可按域名查到它。在需要证书 ID 的位置传入域名时，请求会在发送之前被拒绝，并返回
`*dna.InputError`。

```go
csr, err := client.SSL.GenerateCSR(ctx, dna.SSLGenerateCSRRequest{
	DomainName: "example.com", Country: "TR", State: "Istanbul", Locality: "Istanbul",
	Organization: "Example Ltd", OrganizationUnit: "IT",
})
order, err := client.SSL.Initialize(ctx, "example.com", "PositiveSSL") // 从余额中扣费
id := order.CertificateID                                              // 请保存：之后的每次调用都需要它

_, err = client.SSL.Configure(ctx, dna.SSLConfigureRequest{ // 15 天内完成
	CertificateID: id, CSR: csr.CSR, ApproverMail: "admin@example.com",
	AuthenticationType: dna.AuthenticationEmail, WebServerType: dna.WebServerApache, Contact: sslContact,
})
st, err := client.SSL.Status(ctx, id) // 轮询直到 st.IsIssued
cert, err := client.SSL.Content(ctx, id)
os.WriteFile("example.com.crt", []byte(cert.Chain()), 0o644)
```

在 `Initialize` 或 `Renew` 之后 15 天内未配置的证书会被取消。

`SSL.List` 按从新到旧的顺序返回你的证书（`SSL.All` 会遍历全部证书）。`SSL.Approvers` 列出邮件验证可接受的地址，
既可用于已有证书，也可用于购买之前：

```go
page, err := client.SSL.List(ctx, dna.SSLListRequest{DomainName: "example.com"})
for _, c := range page.Items {
	fmt.Println(c.CertificateID, c.Status, c.SubscriptionExpiresAt)
}

mails, err := client.SSL.Approvers(ctx, dna.SSLApproversRequest{CertificateID: id})
mails, err = client.SSL.Approvers(ctx, dna.SSLApproversRequest{DomainName: "example.com", PackageCode: "PositiveSSL"})
```

续费证书时，先调用 `Renew`，再用 `Configure` 配置它返回的新证书。`Renew` 会从余额中扣费，并且从订阅到期前 30 天
起才可使用。不要通过再次调用 `Initialize` 来续费：那是一笔按购买价格计费的独立新购，旧证书也永远不会变为 `Renewed`
状态。

```go
renewal, err := client.SSL.Renew(ctx, id) // 从余额中扣费
_, err = client.SSL.Configure(ctx, dna.SSLConfigureRequest{
	CertificateID: renewal.CertificateID, // 新证书；旧证书保留自己的 ID
	CSR: newCSR.CSR, ApproverMail: "admin@example.com",
	AuthenticationType: dna.AuthenticationEmail, WebServerType: dna.WebServerApache, Contact: sslContact,
})
```

### 主机托管

```go
packages, err := client.Hosting.Packages(ctx)
created, err := client.Hosting.Create(ctx, "example.com", "Linux-Starter") // 从余额中扣费

ctx, cancel := context.WithTimeout(ctx, 30*time.Minute)
defer cancel()
st, err := client.Hosting.Wait(ctx, created.ServiceID) // 轮询直到激活
fmt.Println(st.PanelAddress, st.Username, st.Password)
```

对于仍在开通中的主机托管，切勿再次调用 `Create`：这会重复扣费。

### 余额、价格与订单

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

info, err := client.Products.Info(ctx, "com", dna.OrderRenew, 2) // .com 续费 2 年的价格
order, err := client.Orders.Get(ctx, "ORDER-CODE")
```

## 请求与 BaseRequest

每个 API 操作都是一个请求类型：`DomainInfoRequest`、`DNSAddRequest`、`SSLConfigureRequest` 等。它们
都实现同一个接口：

```go
type Request interface {
	Endpoint() Endpoint      // 方法、路径和安全级别
	Query() url.Values       // 查询字符串，或 nil
	Body() any               // JSON 请求体，或 nil
	Validate() error         // 在发送任何内容之前检查
}
```

Go 没有继承，所以“基类”是一个被内嵌的结构体。`dna.BaseRequest` 提供默认实现（无
查询、无请求体、无需校验）；请求通过在自己的类型上定义同名方法来覆盖默认实现。只有
`Endpoint` 没有默认实现。本包中的一个请求如下所示：

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

服务方法会为你构建这些请求。`client.Do` 可以发送任何 `Request`，包括你自己的类型，例如
只读取所需的字段，或调用本包尚未封装的端点：

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

你的请求享有内置请求的全部能力：凭据、重试、节奏控制、错误规范化、
关联 ID、追踪以及只读强制。`client.NewHTTPRequest(ctx, req)` 会构建出确切的 HTTP
请求但不发送，便于在计费请求发出之前记录或审查。

## 错误

| 错误                   | 触发时机                                                   | 常用字段                                                         |
|------------------------|------------------------------------------------------------|------------------------------------------------------------------|
| `*dna.InputError`      | `Validate` 拒绝了请求；**没有发送任何内容**                | `Request`, `Fields`                                              |
| `*dna.APIError`        | API 返回了非 2xx 状态码                                    | `StatusCode`, `Code`, `Message`, `Details`, `Fields`, `Data`, `CorrelationID`, `Hint()` |
| `*dna.OperationError`  | API 返回 2xx，但带有 `"success": false`                    | `Code`, `Message`, `CorrelationID`                               |
| `*dna.NetworkError`    | 请求未完成（DNS、TLS、超时、连接重置）                     | `Err`, `Timeout()`                                               |
| `dna.ErrReadOnly`      | 只读客户端阻止了变更；没有发送任何内容                     | 使用 `errors.Is`                                                 |

`dna.KindOf(err)` 可对其中任何一种进行分类：`KindAuth`、`KindNotFound`、`KindValidation`、`KindRejected`、
`KindConflict`、`KindRateLimited`、`KindServer`、`KindNetwork`、`KindInvalidInput`、`KindReadOnly`。辅助函数
可回答常见问题：

```go
_, err := client.Domains.Renew(ctx, "example.com", 1)
var apiErr *dna.APIError
switch {
case err == nil:
case dna.OutcomeUnknown(err):
	// 续费可能已经成功。再次尝试之前先检查该域名。
case dna.IsInsufficientBalance(err):
case dna.IsNotFound(err):
case dna.HasCode(err, dna.CodePrivacyNotSupported):
case errors.As(err, &apiErr):
	log.Println(apiErr.Code, apiErr.Message, apiErr.Hint(), apiErr.CorrelationID)
}
```

请根据 `Code` 而不是 `Message` 来分支处理：消息是本地化的。API 错误码形如
`Dna.DomainService:Domain:10007`；参见[错误参考](https://www.domainnameapi.com/docs/reference/errors.md)。
联系支持团队时请提供 `CorrelationID`。

## 安全与可靠性

- **安全级别。** 每个请求都属于 `Read`、`Write`、`Destructive` 或 `Billable`（`req.Endpoint().Safety`）。
  部分 POST 和 PUT 请求实际上是读取操作（`DomainSearchRequest`、`SSLStatusRequest`）。
- **只读模式。** `dna.WithReadOnly()` 会阻止所有非读取操作，适用于报表任务，也适合
  安全地探索生产环境。
- **重试。** 429 响应总是会被重试，并遵循 `Retry-After`：API 在处理任何内容之前就拒绝了这类请求。
  网络错误和 502/503/504 仅对读取操作和幂等的写入操作重试，因此注册
  或续费绝不会被发送两次。计费请求失败后，请检查 `dna.OutcomeUnknown(err)`。
- **节奏控制。** 客户端会控制请求间隔，使其低于每个密钥的限制（每秒 5 个请求、每秒 2 次搜索、
  每秒一次区域调用）。如果你自行控制节奏，可用 `dna.WithPacing(false)` 将其关闭。
- **配额。** `Response.Quota` 暴露软配额响应头。超出限制时，API 会放慢响应（最多
  5 秒）而不是拒绝请求；默认的 60 秒超时为此留有余地。
- **关联 ID。** 每个请求都带有 `X-Correlation-ID`。可用
  `dna.WithCorrelationID(ctx, id)` 设置你自己的值，以便将 API 调用与你的日志关联起来。
- **可观测性。** `dna.WithLogger(slog.Default())` 会记录每次交互；`dna.WithTrace(func(dna.TraceEvent))`
  会把每个请求、响应和重试交给你，用于指标或错误上报。其中绝不包含 API key。
- **并发。** `Client` 可并发安全使用。创建一个并共享即可。
- **宽容的解码。** 不带时区的日期、`null`、用 `[]` 代替 `{}`、数字或名称形式的状态，以及
  任意大小写的角色（`"registrant"`、`"ADMINISTRATIVE"`）都可以解码。

## 客户端选项

| 选项                            | 作用                                                                                     |
|---------------------------------|------------------------------------------------------------------------------------------|
| `WithEnvironment(dna.OTE)`      | 使用测试环境（默认为 `dna.Production`）                                                  |
| `WithBaseURL(url)`              | 将请求发送到代理或测试服务器                                                             |
| `WithHTTPClient(hc)`            | 使用你自己的 `*http.Client`（传输层、代理、TLS）；绝不会跟随重定向                       |
| `WithTimeout(d)`                | 每次尝试的超时时间（默认 60 s）                                                          |
| `WithUserAgent("app/1.0")`      | 将你的产品名放在 User-Agent 前面                                                         |
| `WithApp("wisecp", "2.0.20")`   | 声明你的应用及其版本（`X-App`、`X-Version`）                                                |
| `WithCurrency("TRY")`           | 搜索价格与目录价格的币种                                                                 |
| `WithRetries(n)`                | 重试次数（默认 2）                                                                       |
| `WithPacing(false)`             | 关闭客户端节奏控制                                                                       |
| `WithReadOnly()`                | 阻止所有非读取请求                                                                       |
| `WithLogger(l)`                 | 将交互记录到 `*slog.Logger`                                                              |
| `WithTrace(fn)`                 | 观察每个请求、响应和重试                                                                 |

### 标识你的应用

每个请求都带有 `X-App` 和 `X-Version` 请求头，告诉 DomainNameAPI 是哪个集成发送的请求。默认值为 `go` 和 SDK
版本。用 `WithApp` 声明你自己的应用（使用 `NewClientFromEnv` 时也可以用 `DNA_APP` 和 `DNA_APP_VERSION`）：

```go
client, err := dna.NewClient(resellerID, apiKey, dna.WithApp("WiseCP", "V2.0.20-beta"))
// X-App: wisecp, X-Version: 2.0.20
```

名称会转为小写，只保留 `a-z`、`0-9`、`.`、`_` 和 `-`；版本只保留数字和点。名称为空或无效时回退为 `go`，版本为空时
回退为 SDK 版本。User-Agent 不受影响。

## 测试你的代码

将客户端指向一个 `httptest.Server`；其他部分都不需要 mock：

```go
srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, `{"success":true,"info":{"domainName":"example.com","status":"AVAILABLE","price":11.99,"currency":"USD"}}`)
}))
defer srv.Close()

client, _ := dna.NewClient("reseller-id", "api-key", dna.WithBaseURL(srv.URL), dna.WithPacing(false))
a, err := client.Domains.Search(ctx, "example.com")
```

## 从 v1 迁移

v2 是全新的设计：用类型化的结果取代 `map[string]any`，用 error 取代 `{"result": "ERROR"}`
信封，并提供完整的 API，而不再只是 php-dna 的方法集合。模块路径已更改为
`github.com/domainreseller/go-dna/v2`，因此在你切换之前，v1 仍可继续使用。

| v1                                                  | v2                                                                     |
|-----------------------------------------------------|------------------------------------------------------------------------|
| `dna.New(id, key, true)`                            | `dna.NewClient(id, key, dna.WithEnvironment(dna.OTE))`                 |
| `res["result"] == "OK"`                             | `err == nil`                                                           |
| `GetResellerDetails`, `GetCurrentBalance`           | `Account.Get` (+ `Account.Balance("USD")`)                             |
| `CheckAvailability`                                 | `Domains.Search`, `Domains.BulkSearch`                                 |
| `GetList`                                           | `Domains.List`, `Domains.All`                                          |
| `GetTldList`                                        | `Products.TLDs`, `Products.AllTLDs`, `Products.TLD`                    |
| `GetDetails`, `GetContacts`, `SyncFromRegistry`     | `Domains.Info`                                                         |
| `RegisterWithContactInfo`                           | `Domains.RegisterWithContacts`（或 `Contacts.Create` + `Domains.Register`） |
| `Renew`, `Transfer`, `CheckTransfer`                | `Domains.Renew`, `Domains.Transfer`, `Domains.TransferCheck`           |
| `ModifyNameServer`                                  | `Domains.SetNameServers`                                               |
| `EnableTheftProtectionLock` / `Disable...`          | `Domains.Lock` / `Domains.Unlock`                                      |
| `ModifyPrivacyProtectionStatus`                     | `Domains.SetPrivacy`                                                   |
| `SaveContacts`                                      | `Domains.UpdateContacts`, `Domains.SetContactHandles`                  |
| `AddChildNameServer`, `ModifyChildNameServer`, `DeleteChildNameServer` | `Domains.AddHost`, `Domains.UpdateHost`, `Domains.DeleteHost` |
| `GetResourceRecords`                                | `DNS.List`                                                             |
| `AddResourceRecord`, `AddResourceRecordValue`       | `DNS.Add`                                                              |
| `EditResourceRecord`, `SetResourceRecordSet`, `ReplaceResourceRecordValue` | `DNS.Update`                                    |
| `DeleteResourceRecord`, `RemoveResourceRecordValue`, `DeleteResourceRecordSet` | `DNS.Delete`（每个值调用一次）              |
| `ApplyResourceRecords`                              | `DNS.Apply`                                                            |
| `GetForwarding`, `SetForwarding`, `DeleteForwarding`| `Domains.Forwarding`, `Domains.SetForwarding`, `Domains.DeleteForwarding` |
| `CancelTransfer`, `ApproveTransfer`, `RejectTransfer` | 不在已发布的 API 中；如果你的账户支持这些操作，请发送[你自己的请求](#请求与-baserequest) |
| `IsTrTLD`, `ValidateContact`                        | `strings.HasSuffix(name, ".tr")`；每个请求都会执行校验                 |
| `GetRequestData`, `GetResponseData`, ...            | `WithTrace`, `WithLogger`, 来自 `client.Do` 的 `Response`               |
| 遥测 (`SetErrorReportingEnabled`, Sentry)      | 已移除：不会有任何数据离开你的进程；请将 `WithTrace` 事件转发到你自己的上报系统 |
| `SetSSLVerification(false)`                         | `WithHTTPClient` 搭配你自己的 TLS 设置                                 |

## 开发

```bash
go test ./...          # 单元测试；均不访问网络
go vet ./...
gofmt -l .
```

`testdata/openapi.json` 是 API 的 OpenAPI 文档。测试会对照它检查每个请求类型（方法、
路径、查询参数名、请求体字段、必填字段）、每个响应类型的 JSON 字段名是否符合响应
schema，以及每个枚举值；当 API 新增了没有对应请求类型的操作时，测试会失败。更新方法：

```bash
curl -fsSL -A "go-dna-spec-update" -o testdata/openapi.json https://api.domainresellerapi.com/swagger/v1/swagger.json
go test ./...
```

本仓库的约定请参见 [AGENTS.md](AGENTS.md)。

## 支持

- API 文档：<https://www.domainnameapi.com/docs/>
- CLI 与 MCP 服务器：<https://github.com/domainreseller/cli>
- 网站：<https://www.domainnameapi.com>

## 许可证

Apache License 2.0。参见 [LICENSE](LICENSE)。
