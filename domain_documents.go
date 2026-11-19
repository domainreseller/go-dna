package dna

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

// Limits of document uploads.
const (
	MaxDocumentSize       = 10 << 20 // bytes per file
	MaxDocumentUploadSize = 50 << 20 // bytes per request
)

// documentExtensions are the accepted file formats. The extension must
// match the file's real format.
var documentExtensions = map[string]bool{".pdf": true, ".jpg": true, ".jpeg": true, ".png": true, ".doc": true, ".docx": true, ".odt": true}

// Document is a registry document of a domain.
type Document struct {
	Name          string                `json:"name"`
	Type          DocumentType          `json:"type"`
	Status        DocumentStatus        `json:"status"`
	OperationType DocumentOperationType `json:"operationType"`
	UploadedDate  Time                  `json:"uploadedDate"`
}

// DocumentUpload is a file to upload as a registry document.
type DocumentUpload struct {
	Type DocumentType
	// FileName must end in .pdf, .jpg, .jpeg, .png, .doc, .docx or .odt.
	FileName string
	// Content is the file content (at most MaxDocumentSize bytes); it is
	// base64-encoded for you.
	Content []byte
}

// DocumentFromFile reads a file into a DocumentUpload.
func DocumentFromFile(docType DocumentType, path string) (DocumentUpload, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return DocumentUpload{}, fmt.Errorf("dna: read document: %w", err)
	}
	return DocumentUpload{Type: docType, FileName: filepath.Base(path), Content: b}, nil
}

// DocumentUploadResult reports an upload.
type DocumentUploadResult struct {
	Uploaded []struct {
		Name         string       `json:"name"`
		Size         int64        `json:"size"`
		MimeType     string       `json:"mimeType"`
		Type         DocumentType `json:"type"`
		UploadedDate Time         `json:"uploadedDate"`
	} `json:"uploaded"`
	// Duplicates and Failed list file names that were not stored.
	Duplicates         []string `json:"duplicates"`
	Failed             []string `json:"failed"`
	TotalDocumentCount int      `json:"totalDocumentCount"`
	TotalDocumentSize  int64    `json:"totalDocumentSize"`
}

// DomainDocumentsListRequest lists the registry documents of a domain.
// CLI: dna domain documents list
type DomainDocumentsListRequest struct {
	BaseRequest
	ListOptions
	DomainName string
	// Types keeps only these document types.
	Types []DocumentType
}

func (DomainDocumentsListRequest) Endpoint() Endpoint {
	return Endpoint{Method: http.MethodGet, Path: "/api/v1/domains/documents", Safety: Read}
}

func (r DomainDocumentsListRequest) Query() url.Values {
	return newParams().str("DomainName", r.DomainName).joined("DocumentTypes", intStrings(r.Types)).page(r.ListOptions).Values
}

func (r DomainDocumentsListRequest) Validate() error {
	var v validator
	v.required("domainName", r.DomainName)
	r.ListOptions.validate(&v)
	return v.err()
}

// DomainDocumentsUploadRequest uploads the documents a registry asked for
// (domain status DomainStatusWaitingForDocument).
// CLI: dna domain documents upload
type DomainDocumentsUploadRequest struct {
	BaseRequest
	DomainName string
	Documents  []DocumentUpload
}

func (DomainDocumentsUploadRequest) Endpoint() Endpoint {
	return Endpoint{Method: http.MethodPost, Path: "/api/v1/domains/documents", Safety: Write}
}

func (r DomainDocumentsUploadRequest) Body() any {
	docs := make([]object, len(r.Documents))
	for i, d := range r.Documents {
		docs[i] = object{
			"documentType": d.Type,
			"fileName":     trim(d.FileName),
			"fileContent":  base64.StdEncoding.EncodeToString(d.Content),
		}
	}
	return object{"domainName": trim(r.DomainName), "documents": docs}
}

func (r DomainDocumentsUploadRequest) Validate() error {
	var v validator
	v.required("domainName", r.DomainName)
	v.requiredList("documents", len(r.Documents))
	total := 0
	for i, d := range r.Documents {
		field := fmt.Sprintf("documents[%d]", i)
		_, known := documentTypeNames[d.Type]
		v.check(known, field+".type", "is not a known DocumentType")
		v.required(field+".fileName", d.FileName)
		ext := strings.ToLower(filepath.Ext(trim(d.FileName)))
		v.check(d.FileName == "" || documentExtensions[ext], field+".fileName", "must be a PDF, JPG, JPEG, PNG, DOC, DOCX or ODT file")
		v.check(len(d.Content) > 0, field+".content", "is empty")
		v.check(len(d.Content) <= MaxDocumentSize, field+".content", "is larger than 10 MB")
		total += len(d.Content)
	}
	v.check(total <= MaxDocumentUploadSize, "documents", "are larger than 50 MB together; upload them in several requests")
	return v.err()
}

// Documents returns one page of a domain's registry documents.
func (s *DomainService) Documents(ctx context.Context, req DomainDocumentsListRequest) (*Page[Document], error) {
	return call[Page[Document]](ctx, s.client, req)
}

// UploadDocuments uploads registry documents for a domain:
//
//	doc, err := dna.DocumentFromFile(dna.DocumentTypeIdentityCard, "id.pdf")
//	...
//	res, err := client.Domains.UploadDocuments(ctx, "example.com.tr", doc)
func (s *DomainService) UploadDocuments(ctx context.Context, domainName string, documents ...DocumentUpload) (*DocumentUploadResult, error) {
	return call[DocumentUploadResult](ctx, s.client, DomainDocumentsUploadRequest{DomainName: domainName, Documents: documents})
}
