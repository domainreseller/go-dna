package dna

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

type namedEnum interface {
	~int
	String() string
}

// checkEnum checks that every value of an enum has a name, parses back and
// decodes from its name.
func checkEnum[E namedEnum](t *testing.T, names map[E]string, parse func(string) (E, error)) {
	t.Helper()
	for v, name := range names {
		if v.String() != name {
			t.Errorf("%d.String() = %q, want %q", v, v.String(), name)
		}
		if parse != nil {
			if got, err := parse(strings.ToLower(name)); err != nil || got != v {
				t.Errorf("parse(%q) = %v, %v", name, got, err)
			}
		}
		var got E
		if err := json.Unmarshal([]byte(`"`+name+`"`), &got); err != nil || got != v {
			t.Errorf("unmarshal %q = %v, %v", name, got, err)
		}
	}
}

func TestEveryEnum(t *testing.T) {
	checkEnum(t, domainStatusNames, ParseDomainStatus)
	checkEnum(t, dnssecModeNames, ParseDNSSECMode)
	checkEnum(t, documentTypeNames, ParseDocumentType)
	checkEnum(t, documentStatusNames, nil)
	checkEnum(t, documentOperationNames, nil)
	checkEnum(t, dnssecStatusNames, nil)
	checkEnum(t, dnssecOperationNames, nil)
	checkEnum(t, dsRecordStatusNames, nil)
	checkEnum(t, dnssecAlgorithmNames, ParseDNSSECAlgorithm)
	checkEnum(t, dsDigestTypeNames, ParseDSDigestType)
	checkEnum(t, transactionTypeNames, ParseTransactionType)
	checkEnum(t, transactionStatusNames, nil)
	checkEnum(t, chargeTypeNames, nil)
	checkEnum(t, productTypeNames, ParseProductType)
	checkEnum(t, orderTypeNames, ParseOrderType)
	checkEnum(t, orderStatusNames, nil)
	checkEnum(t, orderProductStatusNames, nil)
	for _, ct := range ContactTypes {
		if got, err := ParseContactType(string(ct)); err != nil || got != ct {
			t.Errorf("ParseContactType(%q) = %q, %v", ct, got, err)
		}
	}
}

// TestSpecEnums checks every integer enum of the spec: each property with
// an enum must be listed here, and each of its values must have a name.
func TestSpecEnums(t *testing.T) {
	const (
		domains = "Dna.DomainService.Domains."
		dnssec  = "Dna.DomainService.Dnssec."
		deposit = "Dna.DepositService.Accounts."
		orders  = "Dna.OrderService.Orders."
		product = "Dna.ProductService.Products."
	)
	known := map[string]map[int]string{
		domains + "DomainDto.status":                              toInts(domainStatusNames),
		domains + "DomainDocumentInfoDto.status":                  toInts(documentStatusNames),
		domains + "DomainDocumentInfoDto.type":                    toInts(documentTypeNames),
		domains + "DomainDocumentInfoDto.operationType":           toInts(documentOperationNames),
		domains + "DomainDocumentUploadItemInput.documentType":    toInts(documentTypeNames),
		domains + "DomainDocumentUploadedItemDto.type":            toInts(documentTypeNames),
		dnssec + "DnssecInfoBaseDto.status":                       toInts(dnssecStatusNames),
		dnssec + "DsRecordBaseDto.status":                         toInts(dsRecordStatusNames),
		dnssec + "DnssecOperationResultBaseDto.operationType":     toInts(dnssecOperationNames),
		"Dna.DomainService.Dns.DnsEditNameServerInput.dnssecMode": toInts(dnssecModeNames),
		deposit + "AccountTransactionDto.type":                    toInts(transactionTypeNames),
		deposit + "AccountTransactionDto.status":                  toInts(transactionStatusNames),
		deposit + "AccountTransactionDto.chargeType":              toInts(chargeTypeNames),
		deposit + "AccountTransactionDto.productType":             toInts(productTypeNames),
		orders + "OrderDto.status":                                toInts(orderStatusNames),
		orders + "OrderProductDto.status":                         toInts(orderProductStatusNames),
		orders + "OrderProductDto.orderType":                      toInts(orderTypeNames),
		orders + "OrderProductDto.productType":                    toInts(productTypeNames),
		product + "ProductDto.productType":                        toInts(productTypeNames),
		product + "ProductInfoDto.productType":                    toInts(productTypeNames),
		product + "SubproductDto.productType":                     toInts(productTypeNames),
	}
	d := loadSpec(t)
	seen := map[string]bool{}
	for schema, s := range d.Components.Schemas {
		props, _ := s["properties"].(map[string]any)
		for prop, p := range props {
			values, _ := p.(map[string]any)["enum"].([]any)
			if len(values) == 0 {
				continue
			}
			key := schema + "." + prop
			names, ok := known[key]
			if !ok {
				t.Errorf("%s is an enum without a Go type here", key)
				continue
			}
			seen[key] = true
			for _, v := range values {
				if n := int(v.(float64)); names[n] == "" {
					t.Errorf("%s: value %d has no name", key, n)
				}
			}
		}
	}
	for key := range known {
		if !seen[key] {
			t.Errorf("%s is not an enum in the spec any more", key)
		}
	}
}

func toInts[E ~int](m map[E]string) map[int]string {
	out := make(map[int]string, len(m))
	for k, v := range m {
		out[int(k)] = v
	}
	return out
}

func TestDocumentFromFile(t *testing.T) {
	path := t.TempDir() + "/id.pdf"
	if err := os.WriteFile(path, []byte("%PDF-1.4"), 0o600); err != nil {
		t.Fatal(err)
	}
	d, err := DocumentFromFile(DocumentTypeIdentityCard, path)
	if err != nil || d.FileName != "id.pdf" || string(d.Content) != "%PDF-1.4" || d.Type != DocumentTypeIdentityCard {
		t.Errorf("%+v %v", d, err)
	}
	if _, err := DocumentFromFile(DocumentTypeIdentityCard, path+".missing"); err == nil {
		t.Error("missing file accepted")
	}
}
