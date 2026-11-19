package dna

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// The API sends many fields as bare integers. The types below give them
// names. Each has a String method (DomainStatusPendingDelete.String() is
// "PendingDelete") and a Parse function that accepts the name in any case
// or the number. When decoding, the numeric enums accept a number or a
// name; a name this package does not know decodes to -1.

type intEnum interface{ ~int }

func enumString[E intEnum](v E, names map[E]string, typ string) string {
	if n, ok := names[v]; ok {
		return n
	}
	return typ + "(" + strconv.Itoa(int(v)) + ")"
}

func enumNames[E intEnum](names map[E]string) string {
	keys := make([]int, 0, len(names))
	for k := range names {
		keys = append(keys, int(k))
	}
	sort.Ints(keys)
	parts := make([]string, len(keys))
	for i, k := range keys {
		parts[i] = names[E(k)]
	}
	return strings.Join(parts, ", ")
}

func parseEnum[E intEnum](s string, names map[E]string, typ string) (E, error) {
	s = strings.TrimSpace(s)
	for v, n := range names {
		if strings.EqualFold(n, s) {
			return v, nil
		}
	}
	if n, err := strconv.Atoi(s); err == nil {
		if _, ok := names[E(n)]; ok {
			return E(n), nil
		}
	}
	return -1, fmt.Errorf("dna: %q is not a valid %s (allowed: %s)", s, typ, enumNames(names))
}

func unmarshalEnum[E intEnum](b []byte, dst *E, names map[E]string) error {
	s := strings.TrimSpace(string(b))
	switch {
	case s == "null" || s == "":
		return nil
	case s[0] == '"':
		var str string
		if err := json.Unmarshal(b, &str); err != nil {
			return err
		}
		if v, err := parseEnum(str, names, ""); err == nil {
			*dst = v
		} else if n, err := strconv.Atoi(strings.TrimSpace(str)); err == nil {
			*dst = E(n)
		} else {
			*dst = -1
		}
		return nil
	}
	n, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return fmt.Errorf("dna: cannot decode %s as an enum", s)
	}
	*dst = E(int(n))
	return nil
}

// ---------------------------------------------------------------------------
// Domains

// DomainStatus is the lifecycle state of a domain.
type DomainStatus int

const (
	DomainStatusActive                     DomainStatus = 0
	DomainStatusWaitingForRegistration     DomainStatus = 1
	DomainStatusWaitingForDocument         DomainStatus = 2  // registry documents must be uploaded (Domains.UploadDocuments)
	DomainStatusWaitingForIncomingTransfer DomainStatus = 3  // internal transfer from another reseller
	DomainStatusTransferredOut             DomainStatus = 4  // moved to another registrar
	DomainStatusTransferRejected           DomainStatus = 5  // an incoming transfer was rejected
	DomainStatusPendingDelete              DomainStatus = 7  // in the redemption period; Domains.Restore may bring it back
	DomainStatusDeleted                    DomainStatus = 8  // gone
	DomainStatusConfirmationEmailSend      DomainStatus = 9  // transfer confirmation e-mail sent
	DomainStatusWaitingForOutgoingTransfer DomainStatus = 11 // transfer to another registrar in progress
	DomainStatusPendingHold                DomainStatus = 12 // suspended, e.g. the registrant e-mail was not verified in 15 days
	DomainStatusModificationPending        DomainStatus = 15
	DomainStatusInCase                     DomainStatus = 18 // subject to a legal process
	DomainStatusPendingRestore             DomainStatus = 19
)

var domainStatusNames = map[DomainStatus]string{
	0: "Active", 1: "WaitingForRegistration", 2: "WaitingForDocument", 3: "WaitingForIncomingTransfer",
	4: "TransferredOut", 5: "TransferRejected", 7: "PendingDelete", 8: "Deleted", 9: "ConfirmationEmailSend",
	11: "WaitingForOutgoingTransfer", 12: "PendingHold", 15: "ModificationPending", 18: "InCase", 19: "PendingRestore",
}

func (s DomainStatus) String() string { return enumString(s, domainStatusNames, "DomainStatus") }

// UnmarshalJSON accepts the number or the name.
func (s *DomainStatus) UnmarshalJSON(b []byte) error { return unmarshalEnum(b, s, domainStatusNames) }

// ParseDomainStatus parses a name ("pendingdelete") or number ("7").
func ParseDomainStatus(s string) (DomainStatus, error) {
	return parseEnum(s, domainStatusNames, "DomainStatus")
}

// ContactType is a contact role. A registration needs all four.
type ContactType string

const (
	ContactTypeRegistrant ContactType = "Registrant" // the domain owner
	ContactTypeAdmin      ContactType = "Admin"
	ContactTypeTech       ContactType = "Tech"
	ContactTypeBilling    ContactType = "Billing"
)

// ContactTypes are the four roles, in the order registries list them.
var ContactTypes = []ContactType{ContactTypeRegistrant, ContactTypeAdmin, ContactTypeTech, ContactTypeBilling}

// ParseContactType accepts the role in any case, plus the long forms
// "Administrative" and "Technical".
func ParseContactType(s string) (ContactType, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "registrant", "owner":
		return ContactTypeRegistrant, nil
	case "admin", "administrative":
		return ContactTypeAdmin, nil
	case "tech", "technical":
		return ContactTypeTech, nil
	case "billing":
		return ContactTypeBilling, nil
	}
	return "", fmt.Errorf("dna: %q is not a valid ContactType (allowed: Registrant, Admin, Tech, Billing)", s)
}

// UnmarshalJSON normalises the spellings the API returns ("registrant",
// "ADMINISTRATIVE", ...) to the constants. Unknown values are kept as sent.
func (t *ContactType) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return nil
	}
	if v, err := ParseContactType(s); err == nil {
		*t = v
	} else {
		*t = ContactType(s)
	}
	return nil
}

// ForwardType is a URL forwarding mode.
type ForwardType string

const (
	ForwardTypeStandard ForwardType = "Standard" // HTTP 301 redirect
	ForwardTypeFrame    ForwardType = "Frame"    // shows the target in a frame (masked URL)
)

// DNSSECMode says what to do with DNSSEC when the nameservers of a signed
// domain change.
type DNSSECMode int

const (
	// DNSSECModeBlock rejects the change while DS records exist (the default).
	DNSSECModeBlock DNSSECMode = 0
	// DNSSECModeKeep keeps the DS records: you confirm the new DNS provider
	// serves a matching DNSKEY.
	DNSSECModeKeep DNSSECMode = 1
	// DNSSECModeDisable removes all DS records first, then changes the
	// nameservers.
	DNSSECModeDisable DNSSECMode = 2
)

var dnssecModeNames = map[DNSSECMode]string{0: "Block", 1: "KeepDnssec", 2: "DisableDnssec"}

func (m DNSSECMode) String() string { return enumString(m, dnssecModeNames, "DNSSECMode") }

// UnmarshalJSON accepts the number or the name.
func (m *DNSSECMode) UnmarshalJSON(b []byte) error { return unmarshalEnum(b, m, dnssecModeNames) }

// ParseDNSSECMode parses a name ("KeepDnssec") or number.
func ParseDNSSECMode(s string) (DNSSECMode, error) {
	return parseEnum(s, dnssecModeNames, "DNSSECMode")
}

// IPVersion is the IP version of a child nameserver address.
type IPVersion string

const (
	IPVersion4 IPVersion = "v4"
	IPVersion6 IPVersion = "v6" // rejected when the TLD has no IPv6 glue
)

// ---------------------------------------------------------------------------
// Documents

// DocumentType is the kind of a registry document (mostly for .tr domains).
type DocumentType int

const (
	DocumentTypeIdentityCard                 DocumentType = 1
	DocumentTypePassport                     DocumentType = 2
	DocumentTypeTaxCertificate               DocumentType = 3
	DocumentTypeActivityCertificate          DocumentType = 4
	DocumentTypeSignatureCircular            DocumentType = 5
	DocumentTypeAuthorizationLetter          DocumentType = 6
	DocumentTypePetition                     DocumentType = 7
	DocumentTypeTradeRegistryGazette         DocumentType = 8
	DocumentTypeTrademarkCertificate         DocumentType = 9
	DocumentTypeEducationCertificate         DocumentType = 10 // .edu.tr
	DocumentTypeOrganizationStatute          DocumentType = 11 // .org.tr
	DocumentTypeGovernmentApproval           DocumentType = 12 // .gov.tr, .bank, .mil.tr
	DocumentTypeDomainOwnershipProof         DocumentType = 13
	DocumentTypePowerOfAttorney              DocumentType = 14
	DocumentTypeBarAssociationCertificate    DocumentType = 15 // .av.tr
	DocumentTypeMedicalDiplomaOrLicense      DocumentType = 16 // .dr.tr
	DocumentTypeMunicipalOfficialLetter      DocumentType = 17 // .bel.tr
	DocumentTypeEducationInstitutionLicense  DocumentType = 18 // .k12.tr, .edu.tr
	DocumentTypeChamberOfCommerceCertificate DocumentType = 19
	DocumentTypeTrademarkTransferAgreement   DocumentType = 20
	DocumentTypeKepAuthorization             DocumentType = 21 // .kep.tr
	DocumentTypeOther                        DocumentType = 99
)

var documentTypeNames = map[DocumentType]string{
	1: "IdentityCard", 2: "Passport", 3: "TaxCertificate", 4: "ActivityCertificate", 5: "SignatureCircular",
	6: "AuthorizationLetter", 7: "Petition", 8: "TradeRegistryGazette", 9: "TrademarkCertificate",
	10: "EducationCertificate", 11: "OrganizationStatute", 12: "GovernmentApproval", 13: "DomainOwnershipProof",
	14: "PowerOfAttorney", 15: "BarAssociationCertificate", 16: "MedicalDiplomaOrLicense",
	17: "MunicipalOfficialLetter", 18: "EducationInstitutionLicense", 19: "ChamberOfCommerceCertificate",
	20: "TrademarkTransferAgreement", 21: "KepAuthorization", 99: "Other",
}

func (t DocumentType) String() string { return enumString(t, documentTypeNames, "DocumentType") }

// UnmarshalJSON accepts the number or the name.
func (t *DocumentType) UnmarshalJSON(b []byte) error { return unmarshalEnum(b, t, documentTypeNames) }

// ParseDocumentType parses a name ("IdentityCard") or number.
func ParseDocumentType(s string) (DocumentType, error) {
	return parseEnum(s, documentTypeNames, "DocumentType")
}

// DocumentStatus is the review state of a registry document.
type DocumentStatus int

const (
	DocumentStatusWaiting   DocumentStatus = 1 // expected, not uploaded
	DocumentStatusPending   DocumentStatus = 2 // uploaded, not reviewed yet
	DocumentStatusApproved  DocumentStatus = 3
	DocumentStatusRejected  DocumentStatus = 4
	DocumentStatusExpired   DocumentStatus = 5
	DocumentStatusCancelled DocumentStatus = 6 // application or owner change cancelled
)

var documentStatusNames = map[DocumentStatus]string{1: "Waiting", 2: "Pending", 3: "Approved", 4: "Rejected", 5: "Expired", 6: "Cancelled"}

func (s DocumentStatus) String() string { return enumString(s, documentStatusNames, "DocumentStatus") }

// UnmarshalJSON accepts the number or the name.
func (s *DocumentStatus) UnmarshalJSON(b []byte) error {
	return unmarshalEnum(b, s, documentStatusNames)
}

// DocumentOperationType is the operation a document was requested for.
type DocumentOperationType int

const (
	DocumentOperationCreate      DocumentOperationType = 1 // new registration
	DocumentOperationOwnerChange DocumentOperationType = 2 // trade / registrant change
	DocumentOperationDelete      DocumentOperationType = 3
)

var documentOperationNames = map[DocumentOperationType]string{1: "Create", 2: "OwnerChange", 3: "Delete"}

func (t DocumentOperationType) String() string {
	return enumString(t, documentOperationNames, "DocumentOperationType")
}

// UnmarshalJSON accepts the number or the name.
func (t *DocumentOperationType) UnmarshalJSON(b []byte) error {
	return unmarshalEnum(b, t, documentOperationNames)
}

// ---------------------------------------------------------------------------
// DNS and DNSSEC

// RecordType is a DNS record type. The API is case-sensitive about it; the
// requests of this package fix the case for you ("txt" becomes "TXT",
// "dnskey" becomes "DNSKey").
type RecordType string

const (
	RecordTypeA     RecordType = "A"
	RecordTypeAAAA  RecordType = "AAAA"
	RecordTypeALIAS RecordType = "ALIAS" // a CNAME-like record allowed at the apex
	RecordTypeCAA   RecordType = "CAA"   // content has exactly 3 parts: 0 issue "letsencrypt.org"
	RecordTypeCNAME RecordType = "CNAME" // needs a name: not allowed at the apex
	RecordTypeDS    RecordType = "DS"
	RecordTypeMX    RecordType = "MX" // content is "PRIORITY HOST", e.g. "10 mail.example.com"
	RecordTypeNAPTR RecordType = "NAPTR"
	RecordTypeNS    RecordType = "NS"
	RecordTypePTR   RecordType = "PTR"
	RecordTypeSOA   RecordType = "SOA"
	RecordTypeSRV   RecordType = "SRV"
	RecordTypeTLSA  RecordType = "TLSA"
	RecordTypeTXT   RecordType = "TXT" // the API adds the surrounding quotes
)

// recordTypes lists every type the zone API knows, with its exact case.
var recordTypes = []RecordType{
	"A", "AAAA", "ALIAS", "NS", "CNAME", "MX", "SOA", "TXT", "PTR", "SRV", "DNSKey", "DS", "NSEC", "NSEC3",
	"NSEC3Param", "RRSIG", "AFSDB", "ATMA", "CAA", "CERT", "DHCID", "DName", "HInfo", "ISDN", "LOC", "MB", "MG",
	"MInfo", "MR", "NAPTR", "NSAP", "RP", "RT", "TLSA", "X25",
}

// ParseRecordType returns the record type with the case the API expects.
func ParseRecordType(s string) (RecordType, error) {
	s = strings.TrimSpace(s)
	for _, t := range recordTypes {
		if strings.EqualFold(string(t), s) {
			return t, nil
		}
	}
	return "", fmt.Errorf("dna: %q is not a DNS record type the zone API supports", s)
}

// normalize returns t with the API's case, or t unchanged when unknown.
func (t RecordType) normalize() RecordType {
	if v, err := ParseRecordType(string(t)); err == nil {
		return v
	}
	return RecordType(strings.TrimSpace(string(t)))
}

// DNSSECStatus is the DNSSEC state of a domain.
type DNSSECStatus int

const (
	DNSSECStatusUnsigned       DNSSECStatus = 0 // no DS record at the registry
	DNSSECStatusPendingEnable  DNSSECStatus = 1 // DS add sent, awaiting confirmation
	DNSSECStatusSigned         DNSSECStatus = 2 // at least one active DS record
	DNSSECStatusPendingDisable DNSSECStatus = 3 // DS removal sent, awaiting confirmation
	DNSSECStatusError          DNSSECStatus = 4 // the last DNSSEC operation failed
)

var dnssecStatusNames = map[DNSSECStatus]string{0: "Unsigned", 1: "PendingEnable", 2: "Signed", 3: "PendingDisable", 4: "Error"}

func (s DNSSECStatus) String() string { return enumString(s, dnssecStatusNames, "DNSSECStatus") }

// UnmarshalJSON accepts the number or the name.
func (s *DNSSECStatus) UnmarshalJSON(b []byte) error { return unmarshalEnum(b, s, dnssecStatusNames) }

// DNSSECOperationType is the kind of a DNSSEC operation.
type DNSSECOperationType int

const (
	DNSSECOperationAdd       DNSSECOperationType = 0
	DNSSECOperationRemove    DNSSECOperationType = 1
	DNSSECOperationRemoveAll DNSSECOperationType = 2
	DNSSECOperationSync      DNSSECOperationType = 3
)

var dnssecOperationNames = map[DNSSECOperationType]string{0: "Add", 1: "Remove", 2: "RemoveAll", 3: "Sync"}

func (t DNSSECOperationType) String() string {
	return enumString(t, dnssecOperationNames, "DNSSECOperationType")
}

// UnmarshalJSON accepts the number or the name.
func (t *DNSSECOperationType) UnmarshalJSON(b []byte) error {
	return unmarshalEnum(b, t, dnssecOperationNames)
}

// DSRecordStatus is the state of a DS record. Several can be active during
// a key rollover.
type DSRecordStatus int

const (
	DSRecordPending  DSRecordStatus = 0
	DSRecordActive   DSRecordStatus = 1
	DSRecordRemoving DSRecordStatus = 2
	DSRecordRemoved  DSRecordStatus = 3
	DSRecordFailed   DSRecordStatus = 4
)

var dsRecordStatusNames = map[DSRecordStatus]string{0: "Pending", 1: "Active", 2: "Removing", 3: "Removed", 4: "Failed"}

func (s DSRecordStatus) String() string { return enumString(s, dsRecordStatusNames, "DSRecordStatus") }

// UnmarshalJSON accepts the number or the name.
func (s *DSRecordStatus) UnmarshalJSON(b []byte) error {
	return unmarshalEnum(b, s, dsRecordStatusNames)
}

// DNSSECAlgorithm is an IANA DNSSEC algorithm number.
type DNSSECAlgorithm int

const (
	AlgorithmRSAMD5           DNSSECAlgorithm = 1
	AlgorithmDSA              DNSSECAlgorithm = 3
	AlgorithmRSASHA1          DNSSECAlgorithm = 5
	AlgorithmDSANSEC3SHA1     DNSSECAlgorithm = 6
	AlgorithmRSASHA1NSEC3SHA1 DNSSECAlgorithm = 7
	AlgorithmRSASHA256        DNSSECAlgorithm = 8
	AlgorithmRSASHA512        DNSSECAlgorithm = 10
	AlgorithmECCGOST          DNSSECAlgorithm = 12
	AlgorithmECDSAP256SHA256  DNSSECAlgorithm = 13
	AlgorithmECDSAP384SHA384  DNSSECAlgorithm = 14
	AlgorithmED25519          DNSSECAlgorithm = 15
	AlgorithmED448            DNSSECAlgorithm = 16
)

var dnssecAlgorithmNames = map[DNSSECAlgorithm]string{
	1: "RsaMd5", 3: "Dsa", 5: "RsaSha1", 6: "DsaNsec3Sha1", 7: "RsaSha1Nsec3Sha1", 8: "RsaSha256",
	10: "RsaSha512", 12: "EccGost", 13: "EcdsaP256Sha256", 14: "EcdsaP384Sha384", 15: "Ed25519", 16: "Ed448",
}

func (a DNSSECAlgorithm) String() string {
	return enumString(a, dnssecAlgorithmNames, "DNSSECAlgorithm")
}

// UnmarshalJSON accepts the number or the name.
func (a *DNSSECAlgorithm) UnmarshalJSON(b []byte) error {
	return unmarshalEnum(b, a, dnssecAlgorithmNames)
}

// ParseDNSSECAlgorithm parses a name ("EcdsaP256Sha256") or number ("13").
func ParseDNSSECAlgorithm(s string) (DNSSECAlgorithm, error) {
	return parseEnum(s, dnssecAlgorithmNames, "DNSSECAlgorithm")
}

// DSDigestType is an IANA DS digest type.
type DSDigestType int

const (
	DigestSHA1   DSDigestType = 1
	DigestSHA256 DSDigestType = 2
	DigestGOST   DSDigestType = 3
	DigestSHA384 DSDigestType = 4
)

var dsDigestTypeNames = map[DSDigestType]string{1: "Sha1", 2: "Sha256", 3: "Gost", 4: "Sha384"}

func (d DSDigestType) String() string { return enumString(d, dsDigestTypeNames, "DSDigestType") }

// UnmarshalJSON accepts the number or the name.
func (d *DSDigestType) UnmarshalJSON(b []byte) error { return unmarshalEnum(b, d, dsDigestTypeNames) }

// ParseDSDigestType parses a name ("Sha256") or number ("2").
func ParseDSDigestType(s string) (DSDigestType, error) {
	return parseEnum(s, dsDigestTypeNames, "DSDigestType")
}

// ---------------------------------------------------------------------------
// SSL

// AuthenticationType is the domain control validation method of a
// certificate.
type AuthenticationType string

const (
	AuthenticationEmail AuthenticationType = "Email" // approval e-mail to ApproverMail
	AuthenticationCNAME AuthenticationType = "Cname" // DNS CNAME record
	AuthenticationHTTP  AuthenticationType = "Http"  // HTTP file on the site
	AuthenticationFile  AuthenticationType = "File"  // file-based validation
)

// WebServerType is the web server software a certificate is issued for.
type WebServerType string

const (
	WebServerApache WebServerType = "Apache"
	WebServerIIS    WebServerType = "IIS"
	WebServerAOL    WebServerType = "AOL"
	WebServerC2Net  WebServerType = "C2Net"
)

// ---------------------------------------------------------------------------
// Account, orders and products

// TransactionType is the kind of a balance transaction.
type TransactionType int

const (
	TransactionRegister         TransactionType = 1
	TransactionRenew            TransactionType = 2
	TransactionTransfer         TransactionType = 3
	TransactionRestore          TransactionType = 4
	TransactionDeposit          TransactionType = 5
	TransactionBackorder        TransactionType = 6
	TransactionRefund           TransactionType = 7
	TransactionDeduction        TransactionType = 8
	TransactionPreRegister      TransactionType = 9
	TransactionShopping         TransactionType = 10
	TransactionResellerPayment  TransactionType = 11
	TransactionInternalTransfer TransactionType = 12
)

var transactionTypeNames = map[TransactionType]string{
	1: "Register", 2: "Renew", 3: "Transfer", 4: "Restore", 5: "Deposit", 6: "Backorder", 7: "Refund",
	8: "Deduction", 9: "PreRegister", 10: "Shopping", 11: "ResellerPayment", 12: "InternalTransfer",
}

func (t TransactionType) String() string {
	return enumString(t, transactionTypeNames, "TransactionType")
}

// UnmarshalJSON accepts the number or the name.
func (t *TransactionType) UnmarshalJSON(b []byte) error {
	return unmarshalEnum(b, t, transactionTypeNames)
}

// ParseTransactionType parses a name ("Register") or number.
func ParseTransactionType(s string) (TransactionType, error) {
	return parseEnum(s, transactionTypeNames, "TransactionType")
}

// TransactionStatus is the outcome of a balance transaction.
type TransactionStatus int

const (
	TransactionFailed     TransactionStatus = 0
	TransactionSuccess    TransactionStatus = 1
	TransactionCanceled   TransactionStatus = 2
	TransactionProcessing TransactionStatus = 3
	TransactionRefunded   TransactionStatus = 4
)

var transactionStatusNames = map[TransactionStatus]string{0: "Failed", 1: "Success", 2: "Canceled", 3: "Processing", 4: "Refunded"}

func (s TransactionStatus) String() string {
	return enumString(s, transactionStatusNames, "TransactionStatus")
}

// UnmarshalJSON accepts the number or the name.
func (s *TransactionStatus) UnmarshalJSON(b []byte) error {
	return unmarshalEnum(b, s, transactionStatusNames)
}

// ChargeType is the direction of a balance transaction.
type ChargeType int

const (
	ChargeAdd      ChargeType = 0 // credit
	ChargeSubtract ChargeType = 1 // debit
)

var chargeTypeNames = map[ChargeType]string{0: "Add", 1: "Subtract"}

func (t ChargeType) String() string { return enumString(t, chargeTypeNames, "ChargeType") }

// UnmarshalJSON accepts the number or the name.
func (t *ChargeType) UnmarshalJSON(b []byte) error { return unmarshalEnum(b, t, chargeTypeNames) }

// ProductType is the kind of a product.
type ProductType int

const (
	ProductDomain        ProductType = 1
	ProductSSL           ProductType = 2
	ProductDeposit       ProductType = 3
	ProductDomainTrustee ProductType = 4
	ProductHosting       ProductType = 5
	ProductOwnerChange   ProductType = 6
)

var productTypeNames = map[ProductType]string{1: "Domain", 2: "SSL", 3: "Deposit", 4: "DomainTrustee", 5: "Hosting", 6: "OwnerChange"}

func (t ProductType) String() string { return enumString(t, productTypeNames, "ProductType") }

// UnmarshalJSON accepts the number or the name.
func (t *ProductType) UnmarshalJSON(b []byte) error { return unmarshalEnum(b, t, productTypeNames) }

// ParseProductType parses a name ("SSL") or number.
func ParseProductType(s string) (ProductType, error) {
	return parseEnum(s, productTypeNames, "ProductType")
}

// OrderType is the kind of an order.
type OrderType int

const (
	OrderRegister            OrderType = 1
	OrderRenew               OrderType = 2
	OrderTransfer            OrderType = 3
	OrderRestore             OrderType = 4
	OrderSubresellerTransfer OrderType = 5
	OrderBackorder           OrderType = 6
	OrderInternalTransfer    OrderType = 7
	OrderCharge              OrderType = 8
	OrderTrustee             OrderType = 9
	OrderLeasing             OrderType = 10
	OrderCreate              OrderType = 11
	OrderOwnerChange         OrderType = 12
)

var orderTypeNames = map[OrderType]string{
	1: "Register", 2: "Renew", 3: "Transfer", 4: "Restore", 5: "SubresellerTransfer", 6: "Backorder",
	7: "InternalTransfer", 8: "Charge", 9: "Trustee", 10: "Leasing", 11: "Create", 12: "OwnerChange",
}

func (t OrderType) String() string { return enumString(t, orderTypeNames, "OrderType") }

// UnmarshalJSON accepts the number or the name.
func (t *OrderType) UnmarshalJSON(b []byte) error { return unmarshalEnum(b, t, orderTypeNames) }

// ParseOrderType parses a name ("Renew") or number.
func ParseOrderType(s string) (OrderType, error) { return parseEnum(s, orderTypeNames, "OrderType") }

// OrderStatus is the state of an order.
type OrderStatus int

const (
	OrderStatusActive     OrderStatus = 1
	OrderStatusPending    OrderStatus = 2
	OrderStatusCanceled   OrderStatus = 3
	OrderStatusInProgress OrderStatus = 4
)

var orderStatusNames = map[OrderStatus]string{1: "Active", 2: "Pending", 3: "Canceled", 4: "InProgress"}

func (s OrderStatus) String() string { return enumString(s, orderStatusNames, "OrderStatus") }

// UnmarshalJSON accepts the number or the name.
func (s *OrderStatus) UnmarshalJSON(b []byte) error { return unmarshalEnum(b, s, orderStatusNames) }

// OrderProductStatus is the state of an order line.
type OrderProductStatus int

const (
	OrderProductOrdered    OrderProductStatus = 1
	OrderProductCanceled   OrderProductStatus = 2
	OrderProductInProgress OrderProductStatus = 3
)

var orderProductStatusNames = map[OrderProductStatus]string{1: "Ordered", 2: "Canceled", 3: "InProgress"}

func (s OrderProductStatus) String() string {
	return enumString(s, orderProductStatusNames, "OrderProductStatus")
}

// UnmarshalJSON accepts the number or the name.
func (s *OrderProductStatus) UnmarshalJSON(b []byte) error {
	return unmarshalEnum(b, s, orderProductStatusNames)
}
