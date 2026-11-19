package dna

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestTime(t *testing.T) {
	for in, want := range map[string]time.Time{
		`"2024-03-15T10:20:30"`:         time.Date(2024, 3, 15, 10, 20, 30, 0, time.UTC),
		`"2024-03-15T10:20:30.1234567"`: time.Date(2024, 3, 15, 10, 20, 30, 123456700, time.UTC),
		`"2024-03-15T13:20:30+03:00"`:   time.Date(2024, 3, 15, 10, 20, 30, 0, time.UTC),
		`"2024-03-15T10:20:30Z"`:        time.Date(2024, 3, 15, 10, 20, 30, 0, time.UTC),
		`"2024-03-15 10:20:30"`:         time.Date(2024, 3, 15, 10, 20, 30, 0, time.UTC),
		`"2024-03-15"`:                  time.Date(2024, 3, 15, 0, 0, 0, 0, time.UTC),
		`"0001-01-01T00:00:00"`:         {},
		`"not-a-date"`:                  {},
		`""`:                            {},
		`null`:                          {},
		`12`:                            {},
	} {
		var got Time
		if err := json.Unmarshal([]byte(in), &got); err != nil {
			t.Errorf("%s: %v", in, err)
			continue
		}
		if !got.Equal(want) {
			t.Errorf("%s: got %v, want %v", in, got.Time, want)
		}
	}

	b, _ := json.Marshal(struct {
		A Time `json:"a"`
		B Time `json:"b"`
	}{A: Time{time.Date(2024, 3, 15, 10, 20, 30, 0, time.UTC)}})
	if string(b) != `{"a":"2024-03-15T10:20:30Z","b":null}` {
		t.Errorf("marshal: %s", b)
	}
	if (Time{}).String() != "" || (Time{time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC)}).String() != "2024-01-02T03:04:05Z" {
		t.Error("String")
	}
}

func TestAttributes(t *testing.T) {
	for in, want := range map[string]Attributes{
		`{"a":"x","b":1,"c":true,"d":null}`: {"a": "x", "b": "1", "c": "true", "d": ""},
		`{}`:                                {},
		`[]`:                                nil,
		`null`:                              nil,
	} {
		var got Attributes
		if err := json.Unmarshal([]byte(in), &got); err != nil {
			t.Errorf("%s: %v", in, err)
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("%s: got %#v, want %#v", in, got, want)
		}
	}
}

func TestIntEnums(t *testing.T) {
	var s DomainStatus
	for in, want := range map[string]DomainStatus{
		`7`: DomainStatusPendingDelete, `"PendingDelete"`: DomainStatusPendingDelete, `"pendingdelete"`: DomainStatusPendingDelete,
		`"3"`: DomainStatusWaitingForIncomingTransfer, `"clientHold"`: -1, `0`: DomainStatusActive, `42`: 42,
	} {
		s = 99
		if err := json.Unmarshal([]byte(in), &s); err != nil || s != want {
			t.Errorf("%s: got %d (%v), want %d", in, s, err, want)
		}
	}
	if DomainStatusPendingDelete.String() != "PendingDelete" || DomainStatus(42).String() != "DomainStatus(42)" {
		t.Error("String")
	}
	if v, err := ParseDomainStatus(" waitingfordocument "); err != nil || v != DomainStatusWaitingForDocument {
		t.Errorf("ParseDomainStatus: %v %v", v, err)
	}
	if _, err := ParseDomainStatus("nope"); err == nil || !strings.Contains(err.Error(), "Active, WaitingForRegistration") {
		t.Errorf("ParseDomainStatus(nope): %v", err)
	}
	if v, _ := ParseDNSSECAlgorithm("13"); v != AlgorithmECDSAP256SHA256 {
		t.Error("ParseDNSSECAlgorithm by number")
	}
	if v, _ := ParseOrderType("renew"); v != OrderRenew {
		t.Error("ParseOrderType")
	}
	// Enums are sent as numbers.
	b, _ := json.Marshal(map[string]any{"s": DomainStatusDeleted, "m": DNSSECModeDisable})
	if string(b) != `{"m":2,"s":8}` {
		t.Errorf("marshal: %s", b)
	}
}

func TestContactType(t *testing.T) {
	for in, want := range map[string]ContactType{
		`"registrant"`: ContactTypeRegistrant, `"ADMINISTRATIVE"`: ContactTypeAdmin, `"Admin"`: ContactTypeAdmin,
		`"technical"`: ContactTypeTech, `"billing"`: ContactTypeBilling, `"Reseller"`: "Reseller", `null`: "",
	} {
		var got ContactType
		if err := json.Unmarshal([]byte(in), &got); err != nil || got != want {
			t.Errorf("%s: got %q (%v), want %q", in, got, err, want)
		}
	}
	if _, err := ParseContactType("owner2"); err == nil {
		t.Error("ParseContactType accepted a bad role")
	}
}

func TestRecordType(t *testing.T) {
	for in, want := range map[string]RecordType{"txt": "TXT", " cname ": "CNAME", "dnskey": "DNSKey", "Nsec3param": "NSEC3Param", "AAAA": "AAAA"} {
		if got, err := ParseRecordType(in); err != nil || got != want {
			t.Errorf("ParseRecordType(%q) = %q, %v", in, got, err)
		}
	}
	if _, err := ParseRecordType("SPF"); err == nil {
		t.Error("SPF is not a zone record type")
	}
}

func TestParseDS(t *testing.T) {
	for in, want := range map[string]DS{
		"2371 13 2 1f987cc6583e92df":                        {2371, AlgorithmECDSAP256SHA256, DigestSHA256, "1F987CC6583E92DF"},
		"example.com. 3600 IN DS 2371 13 2 1F987CC6 583E92": {2371, 13, 2, "1F987CC6583E92"},
		"60485 RsaSha256 Sha1 2BB183AF5F22588179A53B0A98":   {60485, AlgorithmRSASHA256, DigestSHA1, "2BB183AF5F22588179A53B0A98"},
	} {
		got, err := ParseDS(in)
		if err != nil || got != want {
			t.Errorf("ParseDS(%q) = %+v, %v; want %+v", in, got, err, want)
		}
	}
	for _, in := range []string{"", "2371 13 2", "x 13 2 ABC", "1 99 2 ABC"} {
		if _, err := ParseDS(in); err == nil {
			t.Errorf("ParseDS(%q) accepted", in)
		}
	}
	if (DS{2371, 13, 2, "AB"}).String() != "2371 13 2 AB" {
		t.Error("DS.String")
	}
}

func TestFQDNAndMatches(t *testing.T) {
	for _, tc := range []struct{ domain, name, want string }{
		{"example.com", "", "example.com."},
		{"example.com", "@", "example.com."},
		{"Example.com.", "example.com", "example.com."},
		{"example.com", "www", "www.example.com."},
		{"example.com", "WWW.example.com.", "www.example.com."},
		{"example.com", "a.b", "a.b.example.com."},
	} {
		if got := FQDN(tc.domain, tc.name); got != tc.want {
			t.Errorf("FQDN(%q, %q) = %q, want %q", tc.domain, tc.name, got, tc.want)
		}
	}
	set := RecordSet{Name: "www.example.com.", Type: "CNAME"}
	if !set.Matches("example.com", "www", "cname") || set.Matches("example.com", "www", "A") || set.Matches("example.com", "@", "CNAME") {
		t.Error("RecordSet.Matches")
	}
}

func TestContactNormalisation(t *testing.T) {
	c := Contact{
		FirstName: " Ada ", LastName: "Lovelace", Email: " ada@example.com", Address: "x", City: "Istanbul", State: "Istanbul",
		PostalCode: "34000", Country: "tr", PhoneCountryCode: "+90", Phone: "(555) 123-45 67", Fax: "212 123 45 67",
		Type: ContactTypeAdmin, Handle: "DNA-1",
	}
	got := c.info()
	if got.FirstName != "Ada" || got.Email != "ada@example.com" || got.Country != "TR" || got.PhoneCountryCode != "90" ||
		got.Phone != "5551234567" || got.Fax != "2121234567" || got.Type != "" || got.Handle != "" {
		t.Errorf("info() = %+v", got)
	}
	var v validator
	c.validate(&v, "")
	if v.err() != nil {
		t.Errorf("valid contact rejected: %v", v.err())
	}
	v = validator{}
	Contact{Country: "TUR", Email: "nope"}.validate(&v, "c.")
	msg := v.err().Error()
	for _, want := range []string{"c.firstName: is required", "c.email: is not an e-mail address", "c.country: must be a 2-letter ISO code", "c.phone: is required"} {
		if !strings.Contains(msg, want) {
			t.Errorf("missing %q in %s", want, msg)
		}
	}
}

func TestModelHelpers(t *testing.T) {
	if !(Availability{Status: "available"}).Available() || (Availability{Status: "NOTAVAILABLE"}).Available() {
		t.Error("Availability.Available")
	}
	a := &Account{TRYBalance: 10, USDBalance: 2.5}
	if v, ok := a.Balance("usd"); !ok || v != 2.5 {
		t.Error("Balance(usd)")
	}
	if v, ok := a.Balance("TRY"); !ok || v != 10 {
		t.Error("Balance(TRY)")
	}
	if _, ok := a.Balance("EUR"); ok {
		t.Error("Balance(EUR) should not exist")
	}
	if got := (Transaction{Balance: 88.01, OldBalance: 100}).Change(); got != -11.99 {
		t.Errorf("Change = %v", got)
	}
	tld := &TLD{
		Prices:     []TLDPriceGroup{{Register: []Price{{1, 11.99, "USD"}, {2, 23.98, "USD"}}, Renew: []Price{{1, 13.99, "USD"}}}},
		Attributes: []TLDAttribute{{Key: "a", Required: true}, {Key: "b"}},
	}
	if p, ok := tld.RegisterPrice(2); !ok || p.Price != 23.98 || p.String() != "23.98 USD" {
		t.Errorf("RegisterPrice(2) = %v %v", p, ok)
	}
	if _, ok := tld.TransferPrice(1); ok {
		t.Error("TransferPrice should be missing")
	}
	if r := tld.RequiredAttributes(); len(r) != 1 || r[0].Key != "a" {
		t.Errorf("RequiredAttributes = %+v", r)
	}
	cc := CertificateContent{Certificate: "CERT\n", Intermediate1: "INT1", Root: "ROOT"}
	if cc.Chain() != "CERT\nINT1\n" {
		t.Errorf("Chain = %q", cc.Chain())
	}
	if !(HostingStatus{Status: "active"}).Active() || !(HostingStatus{Status: "Failed"}).Failed() {
		t.Error("HostingStatus")
	}
	if Read.String() != "read" || Billable.String() != "billable" || Safety(9).String() != "Safety(9)" {
		t.Error("Safety.String")
	}
}

// TestDecodeDomainInfo decodes a real response, including the API's
// quirks: dates without offset, null fields and mixed-case contact roles.
func TestDecodeDomainInfo(t *testing.T) {
	const body = `{"domainName":"example.com","domainNameIdn":"example.com","isIdnName":false,"startDate":"2024-03-15T10:20:30",
	"expirationDate":"2027-03-15T10:20:30","transferDate":null,"updatedDate":"2026-10-01T08:00:00","resellerId":"r1","renewalMode":"Manual",
	"authCode":"Ab1!xyZ9","lockStatus":true,"privacyProtectionStatus":false,"remainingDay":126,"statusCode":"clientTransferProhibited",
	"objectId":"123456_DOMAIN_COM-VRSN","rgpStatues":null,"status":"Active","statusReason":null,"isForwarded":false,"useTrusteeContact":false,
	"id":4321,"nameservers":["ns1.example.net","ns2.example.net"],
	"hosts":[{"name":"ns1.example.com","ipAddresses":[{"ipAddress":"192.0.2.10"},{"ipAddress":"192.0.2.11"}]},{"name":"ns2.example.com","ipAddresses":[]}],
	"contacts":[{"firstName":"Ada","contactType":"registrant","handle":"DNA-R-1","discloseFlag":false},{"firstName":"Grace","contactType":"ADMINISTRATIVE","handle":"DNA-A-2"}],
	"additionalAttributes":[]}`
	var d DomainInfo
	if err := json.Unmarshal([]byte(body), &d); err != nil {
		t.Fatal(err)
	}
	if d.Status != DomainStatusActive || !d.Locked || d.RemainingDays != 126 || d.AuthCode != "Ab1!xyZ9" {
		t.Errorf("fields: %+v", d)
	}
	if !d.ExpirationDate.Equal(time.Date(2027, 3, 15, 10, 20, 30, 0, time.UTC)) || !d.TransferDate.IsZero() {
		t.Errorf("dates: %v %v", d.ExpirationDate, d.TransferDate)
	}
	if len(d.Hosts) != 2 || len(d.Hosts[0].IPAddresses) != 2 || d.Hosts[0].IPAddresses[1].Address != "192.0.2.11" {
		t.Errorf("hosts: %+v", d.Hosts)
	}
	if c, ok := d.Contact(ContactTypeAdmin); !ok || c.FirstName != "Grace" || c.Handle != "DNA-A-2" {
		t.Errorf("admin contact: %+v %v", c, ok)
	}
	if _, ok := d.Contact(ContactTypeBilling); ok {
		t.Error("billing contact should be missing")
	}
	if d.AdditionalAttributes != nil {
		t.Errorf("attributes: %#v", d.AdditionalAttributes)
	}

	// A numeric status and bad dates still decode.
	if err := json.Unmarshal([]byte(`{"status":4,"expirationDate":"not-a-date","startDate":"0001-01-01T00:00:00","remainingDay":null,"additionalAttributes":null}`), &d); err != nil {
		t.Fatal(err)
	}
	if d.Status != DomainStatusTransferredOut || !d.ExpirationDate.IsZero() || !d.StartDate.IsZero() {
		t.Errorf("quirks: %+v", d)
	}
}

func TestPaginate(t *testing.T) {
	items := []int{1, 2, 3, 4, 5}
	var skips []int
	fetch := func(total int64) func(ListOptions) (*Page[int], error) {
		return func(o ListOptions) (*Page[int], error) {
			skips = append(skips, o.Skip)
			end := min(o.Skip+o.Limit, len(items))
			return &Page[int]{Items: items[o.Skip:end], TotalCount: total}, nil
		}
	}

	got, err := collect(paginate(ListOptions{Limit: 2}, fetch(5)))
	if err != nil || !reflect.DeepEqual(got, items) || !reflect.DeepEqual(skips, []int{0, 2, 4}) {
		t.Errorf("with total: %v %v skips %v", got, err, skips)
	}

	skips = nil
	got, _ = collect(paginate(ListOptions{Limit: 2}, fetch(0)))
	if !reflect.DeepEqual(got, items) || !reflect.DeepEqual(skips, []int{0, 2, 4}) {
		t.Errorf("without total: %v skips %v", got, skips)
	}

	// Stopping early fetches no further page.
	skips = nil
	for v := range paginate(ListOptions{Limit: 2}, fetch(5)) {
		if v == 2 {
			break
		}
	}
	if len(skips) != 1 {
		t.Errorf("break: fetched %v", skips)
	}

	// The API caps the page size below Limit: TotalCount keeps it going.
	skips = nil
	capped := func(o ListOptions) (*Page[int], error) {
		skips = append(skips, o.Skip)
		end := min(o.Skip+2, len(items))
		return &Page[int]{Items: items[o.Skip:end], TotalCount: 5}, nil
	}
	if got, _ := collect(paginate(ListOptions{}, capped)); !reflect.DeepEqual(got, items) {
		t.Errorf("capped page size: %v skips %v", got, skips)
	}
}
