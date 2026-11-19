package dna

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"
)

var ctx = context.Background()

// bodyOf decodes a recorded JSON body.
func bodyOf(t *testing.T, r recorded) any {
	t.Helper()
	var v any
	if err := json.Unmarshal(r.Body, &v); err != nil {
		t.Fatalf("body %s: %v", r.Body, err)
	}
	return v
}

func jsonEq(t *testing.T, got []byte, want string) {
	t.Helper()
	var g, w any
	if err := json.Unmarshal(got, &g); err != nil {
		t.Fatalf("got %s: %v", got, err)
	}
	if err := json.Unmarshal([]byte(want), &w); err != nil {
		t.Fatalf("want %s: %v", want, err)
	}
	if !reflect.DeepEqual(g, w) {
		t.Errorf("body\n got %s\nwant %s", got, want)
	}
}

// ---------------------------------------------------------------------------
// account

func TestAccount(t *testing.T) {
	f := newFakeAPI(t, ok(`{"resellerId":"r1","resellerName":"example","resellerGroupName":"Reseller","tryBalance":1250.5,"usdBalance":601.64}`))
	a, err := f.client().Account.Get(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if a.ResellerID != "r1" || a.USDBalance != 601.64 || a.TRYBalance != 1250.5 {
		t.Errorf("%+v", a)
	}
}

func TestAccountTransactions(t *testing.T) {
	page := func(skip int) string {
		return fmt.Sprintf(`{"totalCount":3,"items":[{"traceId":"t%d","type":2,"status":1,"chargeType":1,"balance":88.01,"oldBalance":100,"creationTime":"2026-11-01T10:00:00"}]}`, skip)
	}
	f := newFakeAPI(t, ok(page(0)), ok(page(1)), ok(page(2)))
	sub := ChargeSubtract
	var got []Transaction
	for tx, err := range f.client().Account.AllTransactions(ctx, AccountTransactionsRequest{
		ListOptions: ListOptions{Limit: 1}, TransactionTypes: []TransactionType{TransactionRegister, TransactionRenew}, ChargeType: &sub,
		From: time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC),
	}) {
		if err != nil {
			t.Fatal(err)
		}
		got = append(got, tx)
	}
	if len(got) != 3 || got[0].Type != TransactionRenew || got[0].Change() != -11.99 {
		t.Fatalf("got %+v", got)
	}
	reqs := f.requests()
	q := reqs[0].Query
	if !reflect.DeepEqual(q["TransactionTypes"], []string{"1", "2"}) || q.Get("ChargeType") != "1" || q.Get("StartDate") != "2026-11-01T00:00:00Z" || q.Get("MaxResultCount") != "1" {
		t.Errorf("query %v", q)
	}
	if reqs[1].Query.Get("SkipCount") != "1" || reqs[2].Query.Get("SkipCount") != "2" {
		t.Errorf("paging: %v %v", reqs[1].Query, reqs[2].Query)
	}
}

// ---------------------------------------------------------------------------
// contacts

func TestContactsCreate(t *testing.T) {
	f := newFakeAPI(t, ok(`{"handle":"DNA-9","operationMessage":"ok","isSuccess":true}`), ok(`{"handle":null,"operationMessage":"Invalid phone","isSuccess":false}`))
	c := f.client()
	contact := sampleContact()
	contact.Phone = "(555) 123 45 67"
	contact.Type, contact.Handle = ContactTypeAdmin, "ignored"
	handle, err := c.Contacts.Create(ctx, contact)
	if err != nil || handle != "DNA-9" {
		t.Fatalf("%q %v", handle, err)
	}
	body := bodyOf(t, f.last()).(map[string]any)["contact"].(map[string]any)
	if body["phone"] != "5551234567" || body["eMail"] != "ada@example.com" || body["contactType"] != nil || body["handle"] != nil {
		t.Errorf("contact body %v", body)
	}
	_, err = c.Contacts.Create(ctx, sampleContact())
	var oe *OperationError
	if !errors.As(err, &oe) || oe.Message != "Invalid phone" || oe.CorrelationID == "" {
		t.Errorf("isSuccess false: %v", err)
	}
}

func TestContactsOther(t *testing.T) {
	f := newFakeAPI(t,
		ok(`{"contact":{"firstName":"Ada","contactType":"Registrant","handle":"DNA-1"},"verificationObjects":[{"fieldValue":"ada@example.com","isUserVerified":true}]}`),
		ok(`{"results":[{"fieldType":"Email","isUserVerified":false,"tokenExpireDate":"2026-11-20T10:00:00"}]}`),
		apiErr(400, CodeVerificationAlreadyDone, "already verified"),
		ok(`{"handle":"DNA-1","contactUpdateSuccess":true,"stats":{"totalDomainCount":2,"successCount":1,"splitCount":1},"affectedDomains":[{"domainName":"example.com.tr","newHandle":"DNA-2"}]}`),
	)
	c := f.client()
	d, err := c.Contacts.Get(ctx, "DNA 1")
	if err != nil || d.Contact.Handle != "DNA-1" || !d.Verifications[0].Verified {
		t.Fatalf("%+v %v", d, err)
	}
	if f.last().Path != "/api/v1/contacts/DNA 1" {
		t.Errorf("path %s", f.last().Path)
	}
	v, err := c.Contacts.Verification(ctx, "DNA-1")
	if err != nil || len(v) != 1 || v[0].Verified || v[0].TokenExpiresAt.IsZero() || f.last().Query.Get("HandleCode") != "DNA-1" {
		t.Fatalf("%+v %v", v, err)
	}
	if err := c.Contacts.ResendVerification(ctx, "DNA-1"); err != nil {
		t.Errorf("already verified should be nil: %v", err)
	}
	jsonEq(t, f.last().Body, `{"handleCode":"DNA-1"}`)
	res, err := c.Contacts.Update(ctx, "DNA-1", sampleContact())
	if err != nil || res.Stats.SplitCount != 1 || res.AffectedDomains[0].NewHandle != "DNA-2" || f.last().Method != http.MethodPut {
		t.Errorf("%+v %v", res, err)
	}
}

// ---------------------------------------------------------------------------
// domains

func TestDomainSearch(t *testing.T) {
	f := newFakeAPI(t,
		ok(`{"operationMessage":"Command completed successfully","code":"1000","success":true,"info":{"domainName":"example.com","tld":"COM","isPremium":null,"currency":"TRY","period":1,"price":399.9,"status":"AVAILABLE"}}`),
		ok(`{"operationMessage":"Unsupported TLD","code":"2306","reason":"TLD not supported","success":false,"info":null}`),
	)
	c := f.client(WithCurrency("try"))
	a, err := c.Domains.Search(ctx, " example.com ")
	if err != nil || !a.Available() || a.Price != 399.9 || a.IsPremium {
		t.Fatalf("%+v %v", a, err)
	}
	jsonEq(t, f.last().Body, `{"domainName":"example.com","currency":"TRY"}`)
	_, err = c.Domains.Search(ctx, "example.invalid")
	if !HasCode(err, "2306") || KindOf(err) != KindRejected || !strings.Contains(err.Error(), "TLD not supported") {
		t.Errorf("success false: %v", err)
	}
}

func TestDomainBulkSearchSplitsLongLists(t *testing.T) {
	f := newFakeAPI(t, ok(`{"success":true,"infos":[{"domainName":"a","status":"AVAILABLE"}]}`))
	names := make([]string, 65)
	for i := range names {
		names[i] = fmt.Sprintf("name%d.com", i)
	}
	res, err := f.client().Domains.BulkSearch(ctx, names...)
	if err != nil || len(res) != 3 {
		t.Fatalf("%d results, %v", len(res), err)
	}
	reqs := f.requests()
	var sizes []int
	for _, r := range reqs {
		sizes = append(sizes, len(bodyOf(t, r).([]any)))
	}
	if !reflect.DeepEqual(sizes, []int{30, 30, 5}) {
		t.Errorf("chunk sizes %v", sizes)
	}
	jsonEq(t, []byte(mustJSON(bodyOf(t, reqs[2]).([]any)[0])), `{"domainName":"name60.com"}`)
	if _, err := f.client().Domains.BulkSearch(ctx, " "); KindOf(err) != KindInvalidInput {
		t.Errorf("empty list: %v", err)
	}
}

func mustJSON(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}

func TestDomainListAndAll(t *testing.T) {
	f := newFakeAPI(t,
		ok(`{"totalCount":0,"items":[{"domainName":"a.com","status":0,"expirationDate":"2027-01-01T00:00:00","lockStatus":true,"contacts":{"Registrant":"DNA-1"},"nameservers":[{"name":"ns1.x.net","isChildnameServer":false}]},{"domainName":"b.com","status":7}]}`),
		ok(`{"totalCount":0,"items":[{"domainName":"c.com","status":12}]}`),
	)
	var got []string
	for d, err := range f.client().Domains.All(ctx, DomainListRequest{ListOptions: ListOptions{Limit: 2}, Statuses: []DomainStatus{DomainStatusActive, DomainStatusPendingDelete}}) {
		if err != nil {
			t.Fatal(err)
		}
		got = append(got, d.DomainName+"="+d.Status.String())
	}
	if !reflect.DeepEqual(got, []string{"a.com=Active", "b.com=PendingDelete", "c.com=PendingHold"}) {
		t.Errorf("%v", got)
	}
	reqs := f.requests()
	if len(reqs) != 2 || reqs[0].Query.Get("StatusIds") != "0,7" || reqs[1].Query.Get("SkipCount") != "2" {
		t.Errorf("requests %+v", reqs)
	}
}

func TestDomainAllStopsOnError(t *testing.T) {
	f := newFakeAPI(t, apiErr(500, "x", "boom"))
	n := 0
	for _, err := range f.client(WithRetries(0)).Domains.All(ctx, DomainListRequest{}) {
		n++
		if KindOf(err) != KindServer {
			t.Errorf("got %v", err)
		}
	}
	if n != 1 {
		t.Errorf("yielded %d times", n)
	}
}

func TestDomainRegister(t *testing.T) {
	f := newFakeAPI(t,
		ok(`{"success":true,"message":"ok","domainId":"6f1c","status":"Active","expirationDate":"2027-11-19T00:00:00"}`),
		ok(`{"success":false,"message":"Domain is not available for registration","domainId":null,"status":null,"expirationDate":null}`),
	)
	c := f.client()
	req := DomainRegisterRequest{
		DomainName: "example.com", Years: 2, NameServers: []string{"NS1.example.net", " ns2.example.net "},
		Handles: ContactHandles{Registrant: "DNA-1", Admin: "DNA-2", Tech: "DNA-2", Billing: "DNA-3"}, TLDAttributes: Attributes{"lang": "en"},
	}
	res, err := c.Domains.Register(ctx, req)
	if err != nil || res.DomainID != "6f1c" || res.ExpirationDate.Year() != 2027 {
		t.Fatalf("%+v %v", res, err)
	}
	jsonEq(t, f.last().Body, `{"domainName":"example.com","period":2,"nameServers":["NS1.example.net","ns2.example.net"],
		"handles":[{"contactType":"Registrant","handle":"DNA-1"},{"contactType":"Admin","handle":"DNA-2"},{"contactType":"Tech","handle":"DNA-2"},{"contactType":"Billing","handle":"DNA-3"}],
		"tldAttributes":{"lang":"en"}}`)

	_, err = c.Domains.Register(ctx, req)
	var oe *OperationError
	if !errors.As(err, &oe) || oe.Message != "Domain is not available for registration" || oe.Path != "/api/v1/domains/register" {
		t.Errorf("success false: %v", err)
	}

	_, err = c.Domains.Register(ctx, DomainRegisterRequest{DomainName: "example.com", NameServers: []string{"ns1.x.net"}, Handles: ContactHandles{Registrant: "DNA-1"}})
	if KindOf(err) != KindInvalidInput || !strings.Contains(err.Error(), "every role") {
		t.Errorf("missing roles: %v", err)
	}
	if len(f.requests()) != 2 {
		t.Error("an invalid request was sent")
	}
}

func TestDomainRegisterWithContacts(t *testing.T) {
	f := newFakeAPI(t, ok(`{"success":true}`))
	c := f.client()
	contact := sampleContact()
	if _, err := c.Domains.RegisterWithContacts(ctx, DomainRegisterWithContactsRequest{DomainName: "example.com", NameServers: []string{"ns1.x.net"}, Contacts: SameContact(contact)}); err != nil {
		t.Fatal(err)
	}
	contacts := bodyOf(t, f.last()).(map[string]any)["contacts"].([]any)
	var roles []string
	for _, c := range contacts {
		m := c.(map[string]any)
		roles = append(roles, m["contactType"].(string))
		if _, has := m["handle"]; has {
			t.Error("a contact sends a handle")
		}
	}
	if !reflect.DeepEqual(roles, []string{"Registrant", "Admin", "Tech", "Billing"}) {
		t.Errorf("roles %v", roles)
	}

	_, err := c.Domains.RegisterWithContacts(ctx, DomainRegisterWithContactsRequest{DomainName: "example.com", NameServers: []string{"ns1.x.net"}, Contacts: DomainContacts{Registrant: &contact}})
	if err == nil || !strings.Contains(err.Error(), "contacts.Admin: is required") {
		t.Errorf("missing roles: %v", err)
	}
	// The trustee contact replaces them.
	if _, err := c.Domains.RegisterWithContacts(ctx, DomainRegisterWithContactsRequest{DomainName: "example.de", NameServers: []string{"ns1.x.net"}, UseTrusteeContact: true}); err != nil {
		t.Errorf("trustee: %v", err)
	}
	jsonEq(t, f.last().Body, `{"domainName":"example.de","nameServers":["ns1.x.net"],"useTrusteeContact":true}`)
}

func TestDomainRenewAndBalance(t *testing.T) {
	f := newFakeAPI(t,
		apiErr(400, "Dna.DepositService:Deposit:10002", "Insufficient balance."),
		ok(`{"renewabilityStatus":true,"renewalLock":false,"isPremium":true,"premiumPrice":120,"premiumCurrency":"USD"}`),
	)
	c := f.client()
	_, err := c.Domains.Renew(ctx, "example.com", 1)
	if !IsInsufficientBalance(err) {
		t.Fatalf("got %v", err)
	}
	rc, err := c.Domains.RenewCheck(ctx, "example.com", 0)
	if err != nil || !rc.Renewable || rc.PremiumPrice != 120 {
		t.Fatalf("%+v %v", rc, err)
	}
	jsonEq(t, f.last().Body, `{"domainName":"example.com"}`)
}

func TestDomainTransfer(t *testing.T) {
	f := newFakeAPI(t,
		ok(`{"messageKey":"TransferAvailable","authCodeIsRequired":true,"authCodeIsValid":true,"transferLock":false,"transferAvailabilityStatus":true,"domainStatuses":["ok"],"additionalAttributes":[],"premiumPrice":0}`),
		ok(`{"success":true,"status":"WaitingForIncomingTransfer"}`),
		ok(`{"domainName":"example.com","status":"pending","requestedDate":"2026-11-19"}`),
	)
	c := f.client()
	tc, err := c.Domains.TransferCheck(ctx, "example.com", " Xy7!kP2q")
	if err != nil || !tc.Transferable || !tc.AuthCodeValid {
		t.Fatalf("%+v %v", tc, err)
	}
	jsonEq(t, f.last().Body, `{"domainName":"example.com","authCode":" Xy7!kP2q"}`)
	if _, err := c.Domains.Transfer(ctx, DomainTransferRequest{DomainName: "example.com", AuthCode: "Xy7!kP2q", Years: 1}); err != nil {
		t.Fatal(err)
	}
	jsonEq(t, f.last().Body, `{"domainName":"example.com","authCode":"Xy7!kP2q","period":1}`)
	st, err := c.Domains.TransferStatus(ctx, "example.com")
	if err != nil || st.Status != "pending" {
		t.Errorf("%+v %v", st, err)
	}
}

func TestDomainSettings(t *testing.T) {
	f := newFakeAPI(t, ok(``), ok(``), ok(`{"domainName":"example.com","authCode":"New!Code1"}`), ok(``), ok(``), ok(``), ok(``))
	c := f.client()

	if err := c.Domains.SetPrivacy(ctx, "example.com", false); err != nil {
		t.Fatal(err)
	}
	jsonEq(t, f.last().Body, `{"domainName":"example.com","privacyStatus":false}`)

	if err := c.Domains.Unlock(ctx, "example.com"); err != nil || f.last().Path != "/api/v1/domains/unlock" {
		t.Fatal(err)
	}

	code, err := c.Domains.UpdateAuthCode(ctx, "example.com")
	if err != nil || code != "New!Code1" || f.last().Method != http.MethodPut {
		t.Fatalf("%q %v", code, err)
	}

	if err := c.Domains.SetNameServers(ctx, DomainNameServersSetRequest{DomainName: "example.com", NameServers: []string{"NS1.Example.net", "ns2.example.net"}, DNSSECMode: DNSSECModeDisable}); err != nil {
		t.Fatal(err)
	}
	jsonEq(t, f.last().Body, `{"domainName":"example.com","nameServers":["ns1.example.net","ns2.example.net"],"dnssecMode":2}`)

	if err := c.Domains.AddHost(ctx, "example.com", "ns1.example.com", "192.0.2.10", "2001:db8::10"); err != nil {
		t.Fatal(err)
	}
	jsonEq(t, f.last().Body, `{"domainName":"example.com","hostName":"ns1.example.com","ipAddresses":[{"ipAddress":"192.0.2.10","ipVersion":"v4"},{"ipAddress":"2001:db8::10","ipVersion":"v6"}]}`)

	if err := c.Domains.UpdateHost(ctx, DomainHostUpdateRequest{DomainName: "example.com", HostName: "ns1.example.com", IPAddresses: []string{"192.0.2.20"}}); err != nil {
		t.Fatal(err)
	}
	jsonEq(t, f.last().Body, `{"domainName":"example.com","hostName":"ns1.example.com","ipAddresses":[{"ipAddress":"192.0.2.20","ipVersion":"v4"}]}`)

	if err := c.Domains.DeleteHost(ctx, "example.com", "ns1.example.com"); err != nil {
		t.Fatal(err)
	}
	if r := f.last(); r.Method != http.MethodDelete || r.Query.Get("HostName") != "ns1.example.com" || len(r.Body) != 0 {
		t.Errorf("delete host %+v", r)
	}

	if err := c.Domains.AddHost(ctx, "example.com", "ns1.example.com", "999.1.1.1"); err == nil || !strings.Contains(err.Error(), "not an IP address") {
		t.Errorf("bad IP: %v", err)
	}
	if err := c.Domains.SetNameServers(ctx, DomainNameServersSetRequest{DomainName: "example.com", NameServers: []string{"ns1.x.net"}, DNSSECMode: 7}); err == nil {
		t.Error("unknown DNSSEC mode accepted")
	}
}

func TestDomainContacts(t *testing.T) {
	f := newFakeAPI(t, ok(``))
	c := f.client()
	if err := c.Domains.SetContactHandles(ctx, "example.com", ContactHandles{Admin: "DNA-2", Tech: " DNA-2 "}); err != nil {
		t.Fatal(err)
	}
	jsonEq(t, f.last().Body, `{"domainName":"example.com","handleItems":[{"contactType":"Admin","handle":"DNA-2"},{"contactType":"Tech","handle":"DNA-2"}]}`)
	admin := sampleContact()
	if err := c.Domains.UpdateContacts(ctx, "example.com", DomainContacts{Admin: &admin}); err != nil {
		t.Fatal(err)
	}
	contacts := bodyOf(t, f.last()).(map[string]any)["contacts"].([]any)
	if len(contacts) != 1 || contacts[0].(map[string]any)["contactType"] != "Admin" {
		t.Errorf("contacts %v", contacts)
	}
	if err := c.Domains.SetContactHandles(ctx, "example.com", ContactHandles{}); KindOf(err) != KindInvalidInput {
		t.Errorf("no handles: %v", err)
	}
}

func TestDomainDocuments(t *testing.T) {
	f := newFakeAPI(t,
		ok(`{"totalCount":1,"items":[{"name":"id.pdf","type":1,"status":2,"operationType":1,"uploadedDate":"2026-11-01T10:00:00"}]}`),
		ok(`{"uploaded":[{"name":"id.pdf","size":8,"type":1}],"duplicates":[],"failed":[],"totalDocumentCount":1,"totalDocumentSize":8}`),
	)
	c := f.client()
	page, err := c.Domains.Documents(ctx, DomainDocumentsListRequest{DomainName: "example.com.tr", Types: []DocumentType{DocumentTypeIdentityCard, DocumentTypeTaxCertificate}})
	if err != nil || page.Items[0].Status != DocumentStatusPending || page.Items[0].Type.String() != "IdentityCard" {
		t.Fatalf("%+v %v", page, err)
	}
	if f.last().Query.Get("DocumentTypes") != "1,3" {
		t.Errorf("query %v", f.last().Query)
	}
	doc := DocumentUpload{Type: DocumentTypeIdentityCard, FileName: "id.pdf", Content: []byte("%PDF-1.4")}
	res, err := c.Domains.UploadDocuments(ctx, "example.com.tr", doc)
	if err != nil || res.TotalDocumentCount != 1 {
		t.Fatalf("%+v %v", res, err)
	}
	jsonEq(t, f.last().Body, `{"domainName":"example.com.tr","documents":[{"documentType":1,"fileName":"id.pdf","fileContent":"`+base64.StdEncoding.EncodeToString(doc.Content)+`"}]}`)

	for _, bad := range []DocumentUpload{
		{Type: DocumentTypeIdentityCard, FileName: "id.exe", Content: []byte("x")},
		{Type: 77, FileName: "id.pdf", Content: []byte("x")},
		{Type: DocumentTypePassport, FileName: "p.png"},
		{Type: DocumentTypePassport, FileName: "p.png", Content: make([]byte, MaxDocumentSize+1)},
	} {
		if _, err := c.Domains.UploadDocuments(ctx, "example.com.tr", bad); KindOf(err) != KindInvalidInput {
			t.Errorf("%s/%d: %v", bad.FileName, bad.Type, err)
		}
	}
}

func TestDomainForwarding(t *testing.T) {
	notSet := apiErr(404, CodeForwardingNotFound, "Redirect information not found")
	f := newFakeAPI(t,
		ok(`{"redirectAlias":"https://www.example.org/landing","redirectType":"Frame"}`),
		notSet,
		apiErr(404, CodeDomainNotFound, "not found"),
		ok(``),
		notSet,
	)
	c := f.client()
	fw, err := c.Domains.Forwarding(ctx, "example.com")
	if err != nil || fw.URL != "https://www.example.org/landing" || fw.Type != ForwardTypeFrame {
		t.Fatalf("%+v %v", fw, err)
	}
	if f.last().Query.Get("domainName") != "example.com" {
		t.Errorf("query %v", f.last().Query)
	}
	if fw, err := c.Domains.Forwarding(ctx, "example.com"); fw != nil || err != nil {
		t.Errorf("not set: %+v %v", fw, err)
	}
	if _, err := c.Domains.Forwarding(ctx, "nope.com"); !IsNotFound(err) {
		t.Errorf("unknown domain: %v", err)
	}
	if err := c.Domains.SetForwarding(ctx, "example.com", "https://www.example.org", ""); err != nil {
		t.Fatal(err)
	}
	jsonEq(t, f.last().Body, `{"domainName":"example.com","redirectAddress":"https://www.example.org","forwardType":"Standard"}`)
	if err := c.Domains.DeleteForwarding(ctx, "example.com"); err != nil {
		t.Errorf("delete when not set: %v", err)
	}
	if r := f.last(); r.Method != http.MethodDelete || r.Query.Get("DomainName") != "example.com" {
		t.Errorf("%+v", r)
	}
	for _, u := range []string{"www.example.org", "ftp://x", "https://"} {
		if err := c.Domains.SetForwarding(ctx, "example.com", u, ForwardTypeStandard); KindOf(err) != KindInvalidInput {
			t.Errorf("%q: %v", u, err)
		}
	}
}

// ---------------------------------------------------------------------------
// dns and dnssec

func TestDNS(t *testing.T) {
	f := newFakeAPI(t,
		ok(`[{"name":"example.com.","ttl":3600,"type":"A","changeType":null,"records":[{"content":"192.0.2.1","disabled":false}]},
			{"name":"www.example.com.","ttl":300,"type":"CNAME","records":[{"content":"example.com.","disabled":false}]},
			{"name":"example.com.","ttl":3600,"type":"TXT","records":[{"content":"\"v=spf1 ~all\""}]}]`),
		ok(``), ok(``), ok(``),
		ok(`{"nameServersChanged":true,"previousNameServers":["ns1.other.net"],"currentNameServers":["ns1.platform.net"],"systemNameServers":["ns1.platform.net"]}`),
	)
	c := f.client()
	sets, err := c.DNS.List(ctx, "example.com")
	if err != nil || len(sets) != 3 || !sets[1].Matches("example.com", "www", RecordTypeCNAME) || sets[0].Values()[0] != "192.0.2.1" {
		t.Fatalf("%+v %v", sets, err)
	}

	if err := c.DNS.Add(ctx, DNSAddRequest{DomainName: "example.com", Name: "@", Type: "txt", Values: []string{"v=spf1 include:_spf.example.net ~all"}}); err != nil {
		t.Fatal(err)
	}
	r := f.last()
	if r.Query.Get("domainName") != "example.com" {
		t.Errorf("query %v", r.Query)
	}
	jsonEq(t, r.Body, `{"zoneStruct":{"name":"","type":"TXT","ttl":3600,"contents":["v=spf1 include:_spf.example.net ~all"]}}`)

	if err := c.DNS.Update(ctx, DNSUpdateRequest{DomainName: "example.com", RecordName: "www", Type: RecordTypeA, TTL: 600, Values: []string{"192.0.2.20"}}); err != nil {
		t.Fatal(err)
	}
	r = f.last()
	if r.Method != http.MethodPut || r.Query.Get("recordName") != "www" {
		t.Errorf("%+v", r)
	}
	jsonEq(t, r.Body, `{"zoneStruct":{"name":"www","type":"A","ttl":600,"contents":["192.0.2.20"]}}`)

	if err := c.DNS.Delete(ctx, DNSDeleteRequest{DomainName: "example.com", Name: "www", Type: "a", Value: "192.0.2.20"}); err != nil {
		t.Fatal(err)
	}
	q := f.last().Query
	if q.Get("Name") != "www" || q.Get("RecordType") != "A" || q.Get("Record") != "192.0.2.20" {
		t.Errorf("delete query %v", q)
	}

	res, err := c.DNS.Apply(ctx, "example.com")
	if err != nil || !res.NameServersChanged || res.CurrentNameServers[0] != "ns1.platform.net" {
		t.Fatalf("%+v %v", res, err)
	}
	if len(f.requests()) != 5 {
		t.Errorf("%d requests", len(f.requests()))
	}
}

func TestDNSValidation(t *testing.T) {
	c := newFakeAPI(t).client()
	for name, req := range map[string]DNSAddRequest{
		"no type":       {DomainName: "example.com", Name: "www", Values: []string{"192.0.2.1"}},
		"unknown type":  {DomainName: "example.com", Name: "www", Type: "SPF", Values: []string{"x"}},
		"no values":     {DomainName: "example.com", Name: "www", Type: RecordTypeA},
		"cname at apex": {DomainName: "example.com", Name: "@", Type: RecordTypeCNAME, Values: []string{"other.example.net"}},
		"cname to ip":   {DomainName: "example.com", Name: "www", Type: RecordTypeCNAME, Values: []string{"192.0.2.1"}},
		"caa parts":     {DomainName: "example.com", Type: RecordTypeCAA, Values: []string{"issue letsencrypt.org"}},
		"mx format":     {DomainName: "example.com", Type: RecordTypeMX, Values: []string{"mail.example.com"}},
		"a not ipv4":    {DomainName: "example.com", Type: RecordTypeA, Values: []string{"2001:db8::1"}},
		"aaaa not ipv6": {DomainName: "example.com", Type: RecordTypeAAAA, Values: []string{"192.0.2.1"}},
		"ttl too high":  {DomainName: "example.com", Type: RecordTypeA, TTL: 90000, Values: []string{"192.0.2.1"}},
		"name with @":   {DomainName: "example.com", Name: "@www", Type: RecordTypeA, Values: []string{"192.0.2.1"}},
		"no domain":     {Type: RecordTypeA, Values: []string{"192.0.2.1"}},
	} {
		if err := c.DNS.Add(ctx, req); KindOf(err) != KindInvalidInput {
			t.Errorf("%s: %v", name, err)
		}
	}
	for name, req := range map[string]DNSAddRequest{
		"caa":   {DomainName: "example.com", Type: "caa", Values: []string{`0 issue "letsencrypt.org"`}},
		"mx":    {DomainName: "example.com", Type: RecordTypeMX, Values: []string{"10 mail.example.com"}},
		"cname": {DomainName: "example.com", Name: "www", Type: RecordTypeCNAME, Values: []string{"example.com."}},
		"alias": {DomainName: "example.com", Name: "@", Type: RecordTypeALIAS, Values: []string{"lb.example.net"}},
	} {
		if err := req.Validate(); err != nil {
			t.Errorf("%s rejected: %v", name, err)
		}
	}
}

func TestDNSSEC(t *testing.T) {
	f := newFakeAPI(t,
		ok(`{"domainName":"example.com","isDnssecAvailable":true,"status":2,"dsRecords":[{"keyTag":2371,"algorithm":13,"digestType":2,"digest":"1F98","status":1}],"allowedAlgorithms":[8,13],"allowedDigestTypes":[2],"maxSigLife":null}`),
		ok(`{"items":[{"keyTag":2371,"operationType":0,"isSuccess":false,"isPending":true}]}`),
		ok(`{"keyTag":2371,"operationType":1,"isSuccess":true}`),
		ok(`{"operationType":2,"isSuccess":false,"isPending":false,"resultMessage":"Registry refused"}`),
	)
	c := f.client()
	info, err := c.DNSSEC.Get(ctx, "example.com")
	if err != nil || info.Status != DNSSECStatusSigned || info.DSRecords[0].Algorithm != AlgorithmECDSAP256SHA256 || info.AllowedAlgorithms[0] != AlgorithmRSASHA256 {
		t.Fatalf("%+v %v", info, err)
	}
	ds, _ := ParseDS("2371 13 2 1f98 7cc6")
	res, err := c.DNSSEC.Add(ctx, "example.com", ds)
	if err != nil || len(res) != 1 || !res[0].Pending {
		t.Fatalf("%+v %v", res, err)
	}
	jsonEq(t, f.last().Body, `{"domainName":"example.com","records":[{"keyTag":2371,"algorithm":13,"digestType":2,"digest":"1F987CC6"}]}`)
	if _, err := c.DNSSEC.Remove(ctx, "example.com", ds); err != nil {
		t.Fatal(err)
	}
	q := f.last().Query
	if q.Get("KeyTag") != "2371" || q.Get("Algorithm") != "13" || q.Get("DigestType") != "2" || q.Get("Digest") != "1F987CC6" {
		t.Errorf("remove query %v", q)
	}
	_, err = c.DNSSEC.Disable(ctx, "example.com")
	var oe *OperationError
	if !errors.As(err, &oe) || oe.Message != "Registry refused" {
		t.Errorf("disable: %v", err)
	}
	if _, err := c.DNSSEC.Add(ctx, "example.com", DS{KeyTag: 1}); KindOf(err) != KindInvalidInput {
		t.Errorf("incomplete DS: %v", err)
	}
}

// ---------------------------------------------------------------------------
// ssl, hosting, orders, products

func TestSSL(t *testing.T) {
	f := newFakeAPI(t,
		ok(`{"csr":"-----BEGIN CERTIFICATE REQUEST-----","privateKey":"-----BEGIN PRIVATE KEY-----"}`),
		ok(`{"success":true,"certificateId":"c1","status":"Initialized"}`),
		ok(`{"success":true,"status":"Configured","dcvDetails":"CNAME _abc.example.com"}`),
		ok(`{"domainName":"example.com","status":"Issued","isIssued":true,"certificateExpiresAt":"2027-11-19T00:00:00"}`),
		ok(`{"certificate":"CERT","intermediateCertificate1":"INT1","intermediateCertificate2":"","rootCertificate":"ROOT"}`),
		ok(`{"mails":["admin@example.com","webmaster@example.com"]}`),
		ok(`{"success":false,"message":"Certificate already revoked"}`),
	)
	c := f.client()
	csr, err := c.SSL.GenerateCSR(ctx, SSLGenerateCSRRequest{DomainName: "example.com", Country: "tr", State: "Istanbul", Locality: "Istanbul", Organization: "Example", OrganizationUnit: "IT"})
	if err != nil || csr.PrivateKey == "" {
		t.Fatalf("%+v %v", csr, err)
	}
	if bodyOf(t, f.last()).(map[string]any)["country"] != "TR" {
		t.Errorf("country not upper-cased: %s", f.last().Body)
	}
	if res, err := c.SSL.Initialize(ctx, "example.com", "PositiveSSL"); err != nil || res.CertificateID != "c1" {
		t.Fatalf("%+v %v", res, err)
	}
	res, err := c.SSL.Configure(ctx, SSLConfigureRequest{
		DomainName: "example.com", CSR: csr.CSR, ApproverMail: "admin@example.com", AuthenticationType: AuthenticationCNAME,
		WebServerType: WebServerApache, SANList: []string{"a.example.com", "b.example.com"}, Contact: sampleSSLContact(),
	})
	if err != nil || res.DCVDetails == "" {
		t.Fatalf("%+v %v", res, err)
	}
	body := bodyOf(t, f.last()).(map[string]any)
	if body["sanList"] != "a.example.com,b.example.com" || body["authenticationType"] != "Cname" || body["contact"].(map[string]any)["eMail"] != "ada@example.com" {
		t.Errorf("configure body %v", body)
	}
	st, err := c.SSL.Status(ctx, "example.com")
	if err != nil || !st.IsIssued || st.ExpiresAt.Year() != 2027 || f.last().Method != http.MethodPut {
		t.Fatalf("%+v %v", st, err)
	}
	cc, err := c.SSL.Content(ctx, "example.com")
	if err != nil || cc.Chain() != "CERT\nINT1\n" {
		t.Fatalf("%+v %v", cc, err)
	}
	mails, err := c.SSL.Approvers(ctx, "example.com", "")
	if err != nil || len(mails) != 2 || f.last().Query.Has("PackageCode") {
		t.Fatalf("%v %v %v", mails, err, f.last().Query)
	}
	if _, err := c.SSL.Revoke(ctx, "example.com"); KindOf(err) != KindRejected {
		t.Errorf("revoke: %v", err)
	}
	if _, err := c.SSL.Configure(ctx, SSLConfigureRequest{DomainName: "example.com", CSR: "x"}); err == nil || !strings.Contains(err.Error(), "contact.companyName") {
		t.Errorf("missing contact: %v", err)
	}
}

func TestHostingWait(t *testing.T) {
	hostingPollFast, hostingPollSlow = time.Millisecond, time.Millisecond
	t.Cleanup(func() { hostingPollFast, hostingPollSlow = 5*time.Second, 30*time.Second })

	f := newFakeAPI(t,
		reply{Status: 202, Body: `{"serviceId":"HST-1","status":"Provisioning","username":"","password":""}`},
		ok(`{"serviceId":"HST-1","status":"Provisioning"}`),
		ok(`{"serviceId":"HST-1","status":"Active","panelAddress":"https://panel.example.net","username":"u","password":"p"}`),
	)
	c := f.client()
	created, err := c.Hosting.Create(ctx, "example.com", "Linux-Starter")
	if err != nil || created.ServiceID != "HST-1" {
		t.Fatalf("%+v %v", created, err)
	}
	st, err := c.Hosting.Wait(ctx, created.ServiceID)
	if err != nil || !st.Active() || st.Password != "p" {
		t.Fatalf("%+v %v", st, err)
	}
	reqs := f.requests()
	if len(reqs) != 3 || reqs[2].Query.Get("ServiceId") != "HST-1" {
		t.Errorf("requests %+v", reqs)
	}

	f = newFakeAPI(t, ok(`{"serviceId":"HST-2","status":"Failed","failureReason":"Out of capacity"}`))
	_, err = f.client().Hosting.Wait(ctx, "HST-2")
	var oe *OperationError
	if !errors.As(err, &oe) || oe.Message != "Out of capacity" {
		t.Errorf("failed: %v", err)
	}

	f = newFakeAPI(t, ok(`{"status":"Provisioning"}`))
	short, cancel := context.WithTimeout(ctx, 30*time.Millisecond)
	defer cancel()
	if _, err := f.client().Hosting.Wait(short, "HST-3"); !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("deadline: %v", err)
	}
}

func TestHostingLookups(t *testing.T) {
	f := newFakeAPI(t,
		ok(`[{"name":"Linux","requiredMinLeasing":1,"permittedMaxLeasing":36,"servicePlans":[{"servicePlan":"Linux-Starter","operatingSystem":"Linux","panelType":"Plesk"}]}]`),
		ok(`[{"name":"Starter","diskQuotaMb":10240,"maxDomains":null}]`),
		ok(`{"hostingId":"h1","serviceId":"HST-1","domainName":"example.com","expiredOn":"2027-01-01T00:00:00"}`),
	)
	c := f.client()
	pk, err := c.Hosting.Packages(ctx)
	if err != nil || pk[0].ServicePlans[0].ServicePlan != "Linux-Starter" {
		t.Fatalf("%+v %v", pk, err)
	}
	plans, err := c.Hosting.Plans(ctx, "")
	if err != nil || *plans[0].DiskQuotaMB != 10240 || plans[0].MaxDomains != nil {
		t.Fatalf("%+v %v", plans, err)
	}
	h, err := c.Hosting.Get(ctx, HostingGetRequest{DomainName: "example.com"})
	if err != nil || h.ServiceID != "HST-1" || h.ExpiresOn.Year() != 2027 || f.last().Query.Get("domainName") != "example.com" {
		t.Fatalf("%+v %v", h, err)
	}
	if _, err := c.Hosting.Get(ctx, HostingGetRequest{DomainName: "example.com", ServiceID: "HST-1"}); KindOf(err) != KindInvalidInput {
		t.Errorf("both set: %v", err)
	}
	if _, err := c.Hosting.Status(ctx, HostingStatusRequest{}); KindOf(err) != KindInvalidInput {
		t.Errorf("none set: %v", err)
	}
}

func TestOrdersAndProducts(t *testing.T) {
	f := newFakeAPI(t,
		ok(`{"orderCode":"ORD/1","status":1,"amount":11.99,"products":[{"productType":1,"orderType":1,"status":1,"period":1}]}`),
		ok(`{"productName":"com","productType":1,"sellingPrice":13.99,"sellingCurrency":"USD"}`),
		ok(`{"totalCount":2,"items":[{"name":"com.tr","prices":[{"priceGroup":"Reseller","register":[{"period":1,"price":9.99,"currency":"USD"}]}],"attributes":[{"key":"trIdentityNumber","isRequired":true}]},{"name":"tr"}]}`),
		ok(`{"totalCount":1,"items":[{"name":"PositiveSSL","productCode":"PSSL","prices":[{"priceGroup":"Reseller","create":[{"period":1,"price":5,"currency":"USD"}]}]}]}`),
	)
	c := f.client(WithCurrency("USD"))
	o, err := c.Orders.Get(ctx, "ORD/1")
	if err != nil || o.Status != OrderStatusActive || o.Products[0].OrderType != OrderRegister {
		t.Fatalf("%+v %v", o, err)
	}
	if f.last().Path != "/api/v1/order/ORD/1" {
		t.Errorf("path %q", f.last().Path)
	}
	info, err := c.Products.Info(ctx, ".com", OrderRenew, 2)
	if err != nil || info.SellingPrice != 13.99 {
		t.Fatalf("%+v %v", info, err)
	}
	if q := f.last().Query; q.Get("ProductName") != "com" || q.Get("OrderType") != "2" || q.Get("Period") != "2" {
		t.Errorf("info query %v", q)
	}
	tld, err := c.Products.TLD(ctx, ".COM.TR")
	if err != nil || tld == nil || tld.Name != "com.tr" || len(tld.RequiredAttributes()) != 1 {
		t.Fatalf("%+v %v", tld, err)
	}
	if p, ok := tld.RegisterPrice(1); !ok || p.Price != 9.99 {
		t.Errorf("price %v", p)
	}
	if q := f.last().Query; q.Get("FilterText") != "com.tr" || q.Get("Currency") != "USD" {
		t.Errorf("tld query %v", q)
	}
	ssl, err := c.Products.SSLs(ctx, SSLProductListRequest{IsWildcard: Bool(false)})
	if err != nil || ssl.Items[0].ProductCode != "PSSL" || f.last().Query.Get("IsWildcard") != "false" {
		t.Fatalf("%+v %v %v", ssl, err, f.last().Query)
	}
	if _, err := c.Products.Info(ctx, "com", 0, 1); KindOf(err) != KindInvalidInput {
		t.Errorf("missing order type: %v", err)
	}
}
