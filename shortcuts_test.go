package dna

import (
	"strings"
	"testing"
)

// TestEveryServiceMethod calls each service method once and checks that it
// sends the request its CLI command sends.
func TestEveryServiceMethod(t *testing.T) {
	contact := sampleContact()
	ds := DS{KeyTag: 1, Algorithm: 13, DigestType: 2, Digest: "AB"}
	for _, tc := range []struct {
		method, path string
		call         func(c *Client) error
	}{
		{"GET", "/api/v1/deposit/accounts/me", func(c *Client) error { _, err := c.Account.Get(ctx); return err }},
		{"GET", "/api/v1/resellers/me", func(c *Client) error { _, err := c.Account.Profile(ctx); return err }},
		{"GET", "/api/v1/deposit/transactions", func(c *Client) error {
			_, err := c.Account.Transactions(ctx, AccountTransactionsRequest{})
			return err
		}},
		{"GET", "/api/v1/contacts/DNA-1", func(c *Client) error { _, err := c.Contacts.Get(ctx, "DNA-1"); return err }},
		{"PUT", "/api/v1/contacts/DNA-1", func(c *Client) error { _, err := c.Contacts.Update(ctx, "DNA-1", contact); return err }},
		{"GET", "/api/v1/contacts/verification/check", func(c *Client) error { _, err := c.Contacts.Verification(ctx, "DNA-1"); return err }},
		{"POST", "/api/v1/contacts/verification/request", func(c *Client) error { return c.Contacts.ResendVerification(ctx, "DNA-1") }},
		{"POST", "/api/v1/contacts", func(c *Client) error { _, err := c.Contacts.Create(ctx, contact); return err }},
		{"POST", "/api/v1/domains/search", func(c *Client) error { _, err := c.Domains.Search(ctx, "a.com"); return err }},
		{"POST", "/api/v1/domains/bulk-search", func(c *Client) error { _, err := c.Domains.BulkSearch(ctx, "a.com"); return err }},
		{"GET", "/api/v1/domains", func(c *Client) error { _, err := c.Domains.List(ctx, DomainListRequest{}); return err }},
		{"GET", "/api/v1/domains/info", func(c *Client) error { _, err := c.Domains.Info(ctx, "a.com"); return err }},
		{"POST", "/api/v1/domains/register", func(c *Client) error {
			_, err := c.Domains.Register(ctx, DomainRegisterRequest{DomainName: "a.com", NameServers: []string{"ns1.x.net"}, Handles: SameHandle("H")})
			return err
		}},
		{"POST", "/api/v1/domains/register-with-contacts", func(c *Client) error {
			_, err := c.Domains.RegisterWithContacts(ctx, DomainRegisterWithContactsRequest{DomainName: "a.com", NameServers: []string{"ns1.x.net"}, Contacts: SameContact(contact)})
			return err
		}},
		{"POST", "/api/v1/domains/renew", func(c *Client) error { _, err := c.Domains.Renew(ctx, "a.com", 1); return err }},
		{"POST", "/api/v1/domains/renew/check", func(c *Client) error { _, err := c.Domains.RenewCheck(ctx, "a.com", 1); return err }},
		{"POST", "/api/v1/domains/restore", func(c *Client) error { return c.Domains.Restore(ctx, "a.com") }},
		{"POST", "/api/v1/domains/transfer", func(c *Client) error {
			_, err := c.Domains.Transfer(ctx, DomainTransferRequest{DomainName: "a.com", AuthCode: "x"})
			return err
		}},
		{"POST", "/api/v1/domains/transfers/check", func(c *Client) error { _, err := c.Domains.TransferCheck(ctx, "a.com", "x"); return err }},
		{"POST", "/api/v1/domains/transfers/query", func(c *Client) error { _, err := c.Domains.TransferStatus(ctx, "a.com"); return err }},
		{"POST", "/api/v1/domains/lock", func(c *Client) error { return c.Domains.Lock(ctx, "a.com") }},
		{"POST", "/api/v1/domains/unlock", func(c *Client) error { return c.Domains.Unlock(ctx, "a.com") }},
		{"POST", "/api/v1/domains/privacy", func(c *Client) error { return c.Domains.SetPrivacy(ctx, "a.com", true) }},
		{"PUT", "/api/v1/domains/auth", func(c *Client) error { _, err := c.Domains.UpdateAuthCode(ctx, "a.com"); return err }},
		{"PUT", "/api/v1/domains/dns/name-server", func(c *Client) error {
			return c.Domains.SetNameServers(ctx, DomainNameServersSetRequest{DomainName: "a.com", NameServers: []string{"ns1.x.net"}})
		}},
		{"POST", "/api/v1/domains/dns/host", func(c *Client) error { return c.Domains.AddHost(ctx, "a.com", "ns1.a.com", "192.0.2.1") }},
		{"PUT", "/api/v1/domains/dns/host", func(c *Client) error {
			return c.Domains.UpdateHost(ctx, DomainHostUpdateRequest{DomainName: "a.com", HostName: "ns1.a.com", IPAddresses: []string{"192.0.2.1"}})
		}},
		{"DELETE", "/api/v1/domains/dns/host", func(c *Client) error { return c.Domains.DeleteHost(ctx, "a.com", "ns1.a.com") }},
		{"PUT", "/api/v1/domains/contacts/update", func(c *Client) error { return c.Domains.UpdateContacts(ctx, "a.com", SameContact(contact)) }},
		{"PUT", "/api/v1/domains/contacts/update-with-handle", func(c *Client) error {
			return c.Domains.SetContactHandles(ctx, "a.com", SameHandle("H"))
		}},
		{"GET", "/api/v1/domains/documents", func(c *Client) error {
			_, err := c.Domains.Documents(ctx, DomainDocumentsListRequest{DomainName: "a.com.tr"})
			return err
		}},
		{"POST", "/api/v1/domains/documents", func(c *Client) error {
			_, err := c.Domains.UploadDocuments(ctx, "a.com.tr", DocumentUpload{Type: DocumentTypePassport, FileName: "p.pdf", Content: []byte("x")})
			return err
		}},
		{"GET", "/api/v1/domains/forwards", func(c *Client) error { _, err := c.Domains.Forwarding(ctx, "a.com"); return err }},
		{"POST", "/api/v1/domains/forwards", func(c *Client) error {
			return c.Domains.SetForwarding(ctx, "a.com", "https://b.com", ForwardTypeStandard)
		}},
		{"DELETE", "/api/v1/domains/forwards", func(c *Client) error { return c.Domains.DeleteForwarding(ctx, "a.com") }},
		{"GET", "/api/v1/domains/zones", func(c *Client) error { _, err := c.DNS.List(ctx, "a.com"); return err }},
		{"POST", "/api/v1/domains/zones", func(c *Client) error {
			return c.DNS.Add(ctx, DNSAddRequest{DomainName: "a.com", Name: "www", Type: RecordTypeA, Values: []string{"192.0.2.1"}})
		}},
		{"PUT", "/api/v1/domains/zones", func(c *Client) error {
			return c.DNS.Update(ctx, DNSUpdateRequest{DomainName: "a.com", RecordName: "www", Type: RecordTypeA, Values: []string{"192.0.2.1"}})
		}},
		{"DELETE", "/api/v1/domains/zones", func(c *Client) error {
			return c.DNS.Delete(ctx, DNSDeleteRequest{DomainName: "a.com", Name: "www", Type: RecordTypeA, Value: "192.0.2.1"})
		}},
		{"POST", "/api/v1/domains/zones/apply", func(c *Client) error { _, err := c.DNS.Apply(ctx, "a.com"); return err }},
		{"GET", "/api/v1/domains/dns/dnssec", func(c *Client) error { _, err := c.DNSSEC.Get(ctx, "a.com"); return err }},
		{"POST", "/api/v1/domains/dns/dnssec", func(c *Client) error { _, err := c.DNSSEC.Add(ctx, "a.com", ds); return err }},
		{"DELETE", "/api/v1/domains/dns/dnssec", func(c *Client) error { _, err := c.DNSSEC.Remove(ctx, "a.com", ds); return err }},
		{"POST", "/api/v1/domains/dns/dnssec/disable", func(c *Client) error { _, err := c.DNSSEC.Disable(ctx, "a.com"); return err }},
		{"POST", "/api/v1/ssls/generate-csr", func(c *Client) error {
			_, err := c.SSL.GenerateCSR(ctx, SSLGenerateCSRRequest{DomainName: "a.com", Country: "TR", State: "x", Locality: "x", Organization: "x", OrganizationUnit: "x"})
			return err
		}},
		{"POST", "/api/v1/ssls/check-csr", func(c *Client) error { _, err := c.SSL.CheckCSR(ctx, "csr"); return err }},
		{"POST", "/api/v1/ssls/initialize", func(c *Client) error { _, err := c.SSL.Initialize(ctx, "a.com", "P"); return err }},
		{"PUT", "/api/v1/ssls/configure", func(c *Client) error {
			_, err := c.SSL.Configure(ctx, SSLConfigureRequest{CertificateID: testCertificateID, CSR: "csr", Contact: sampleSSLContact()})
			return err
		}},
		{"PUT", "/api/v1/ssls/check-status", func(c *Client) error { _, err := c.SSL.Status(ctx, testCertificateID); return err }},
		{"GET", "/api/v1/ssls/certificate", func(c *Client) error { _, err := c.SSL.Get(ctx, testCertificateID); return err }},
		{"GET", "/api/v1/ssls/certificate-content", func(c *Client) error { _, err := c.SSL.Content(ctx, testCertificateID); return err }},
		{"POST", "/api/v1/ssls/reissue", func(c *Client) error {
			_, err := c.SSL.Reissue(ctx, SSLReissueRequest{CertificateID: testCertificateID})
			return err
		}},
		{"GET", "/api/v1/ssls/approvers", func(c *Client) error {
			_, err := c.SSL.Approvers(ctx, SSLApproversRequest{DomainName: "a.com", PackageCode: "P"})
			return err
		}},
		{"DELETE", "/api/v1/ssls/revoke", func(c *Client) error { _, err := c.SSL.Revoke(ctx, testCertificateID); return err }},
		{"POST", "/api/v1/ssls/renew", func(c *Client) error { _, err := c.SSL.Renew(ctx, testCertificateID); return err }},
		{"GET", "/api/v1/ssls", func(c *Client) error { _, err := c.SSL.List(ctx, SSLListRequest{}); return err }},
		{"GET", "/api/v1/hostings/packages", func(c *Client) error { _, err := c.Hosting.Packages(ctx); return err }},
		{"GET", "/api/v1/hostings/plans", func(c *Client) error { _, err := c.Hosting.Plans(ctx, "h"); return err }},
		{"POST", "/api/v1/hostings/create", func(c *Client) error { _, err := c.Hosting.Create(ctx, "a.com", "plan"); return err }},
		{"GET", "/api/v1/hostings", func(c *Client) error {
			_, err := c.Hosting.Get(ctx, HostingGetRequest{ServiceID: "HST-1"})
			return err
		}},
		{"GET", "/api/v1/hostings/by-token", func(c *Client) error { _, err := c.Hosting.GetByToken(ctx, "tok"); return err }},
		{"GET", "/api/v1/hostings/check-status", func(c *Client) error {
			_, err := c.Hosting.Status(ctx, HostingStatusRequest{ServiceID: "HST-1"})
			return err
		}},
		{"GET", "/api/v1/order/O-1", func(c *Client) error { _, err := c.Orders.Get(ctx, "O-1"); return err }},
		{"GET", "/api/v1/products", func(c *Client) error { _, err := c.Products.List(ctx, ProductListRequest{}); return err }},
		{"GET", "/api/v1/products/info", func(c *Client) error { _, err := c.Products.Info(ctx, "com", OrderRegister, 1); return err }},
		{"GET", "/api/v1/products/tlds", func(c *Client) error { _, err := c.Products.TLDs(ctx, TLDListRequest{}); return err }},
		{"GET", "/api/v1/products/ssls", func(c *Client) error { _, err := c.Products.SSLs(ctx, SSLProductListRequest{}); return err }},
	} {
		t.Run(tc.method+" "+tc.path, func(t *testing.T) {
			// A generic successful answer that every service accepts.
			body := `{"success":true,"isSuccess":true,"contactUpdateSuccess":true,"handle":"H","info":{},"infos":[{}]}`
			if (tc.method == "GET" && strings.HasSuffix(tc.path, "/zones")) || strings.HasSuffix(tc.path, "/packages") || strings.HasSuffix(tc.path, "/plans") {
				body = `[]`
			}
			f := newFakeAPI(t, ok(body))
			if err := tc.call(f.client()); err != nil {
				t.Fatal(err)
			}
			if r := f.last(); r.Method != tc.method || r.Path != tc.path {
				t.Errorf("sent %s %s", r.Method, r.Path)
			}
		})
	}
}
