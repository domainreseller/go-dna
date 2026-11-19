package dna

import (
	"context"
	"net/http"
)

// DomainContactsUpdateRequest replaces contacts of a domain with inline
// contact details. Roles left nil stay as they are.
// CLI: dna domain contacts update
type DomainContactsUpdateRequest struct {
	BaseRequest
	DomainName string
	// Contacts are the new contacts. Changing the registrant may start an
	// owner-change process, with documents on some TLDs.
	Contacts DomainContacts
}

func (DomainContactsUpdateRequest) Endpoint() Endpoint {
	return Endpoint{Method: http.MethodPut, Path: "/api/v1/domains/contacts/update", Safety: Write}
}

func (r DomainContactsUpdateRequest) Body() any {
	return object{"domainName": trim(r.DomainName), "contacts": r.Contacts.list()}
}

func (r DomainContactsUpdateRequest) Validate() error {
	var v validator
	v.required("domainName", r.DomainName)
	r.Contacts.validate(&v, "contacts", false)
	return v.err()
}

// DomainContactsSetHandlesRequest points contact roles of a domain at
// existing contact handles. Roles left empty stay as they are.
// CLI: dna domain contacts set-handles
type DomainContactsSetHandlesRequest struct {
	BaseRequest
	DomainName string
	Handles    ContactHandles
}

func (DomainContactsSetHandlesRequest) Endpoint() Endpoint {
	return Endpoint{Method: http.MethodPut, Path: "/api/v1/domains/contacts/update-with-handle", Safety: Write}
}

func (r DomainContactsSetHandlesRequest) Body() any {
	return object{"domainName": trim(r.DomainName), "handleItems": r.Handles.items()}
}

func (r DomainContactsSetHandlesRequest) Validate() error {
	var v validator
	v.required("domainName", r.DomainName)
	r.Handles.validate(&v, "handles", false)
	return v.err()
}

// UpdateContacts replaces contacts of a domain with inline details; roles
// left nil stay as they are.
func (s *DomainService) UpdateContacts(ctx context.Context, domainName string, contacts DomainContacts) error {
	return exec(ctx, s.client, DomainContactsUpdateRequest{DomainName: domainName, Contacts: contacts})
}

// SetContactHandles points contact roles of a domain at existing handles;
// roles left empty stay as they are.
//
//	err := client.Domains.SetContactHandles(ctx, "example.com", dna.ContactHandles{Admin: "DNA-2", Tech: "DNA-2"})
func (s *DomainService) SetContactHandles(ctx context.Context, domainName string, handles ContactHandles) error {
	return exec(ctx, s.client, DomainContactsSetHandlesRequest{DomainName: domainName, Handles: handles})
}
