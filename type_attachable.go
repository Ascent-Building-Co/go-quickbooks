package quickbooks

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"net/url"
	"strings"
	"time"
)

// AttachableCategoryEnum classifies the type of attachment. Values are
// case-sensitive.
type AttachableCategoryEnum string

const (
	ContactPhotoAttachableCategory AttachableCategoryEnum = "Contact Photo"
	DocumentAttachableCategory     AttachableCategoryEnum = "Document"
	ImageAttachableCategory        AttachableCategoryEnum = "Image"
	ReceiptAttachableCategory      AttachableCategoryEnum = "Receipt"
	SignatureAttachableCategory    AttachableCategoryEnum = "Signature"
	SoundAttachableCategory        AttachableCategoryEnum = "Sound"
	OtherAttachableCategory        AttachableCategoryEnum = "Other"
)

// AttachableRef links an Attachable to a specific transaction or entity.
type AttachableRef struct {
	EntityRef     *ReferenceType `json:"EntityRef,omitempty"`
	IncludeOnSend *bool          `json:"IncludeOnSend,omitempty"`
	LineInfo      string         `json:"LineInfo,omitempty"`
	NoRefOnly     *bool          `json:"NoRefOnly,omitempty"`
	Inactive      *bool          `json:"Inactive,omitempty"`
}

type AttachableService struct {
	companyService
}

func (c *CompanyClient) Attachables() *AttachableService {
	return &AttachableService{companyService: companyService{company: c}}
}

// Attachable represents a QuickBooks Online Attachable entity — either a
// file attachment or a standalone note optionally linked to a transaction.
//
// Two variants exist:
//   - Note attachment: set Note; FileName and file-specific read-only fields
//     are absent.
//   - File attachment: FileName and ContentType identify the file; read-only
//     fields (FileAccessUri, TempDownloadUri, Size, etc.) are populated by the
//     server after upload.
//
// For MVP, Create supports note attachments only (JSON POST). File attachments
// are created via Upload (multipart POST to /upload). Existing file attachments
// can be read, queried, updated (metadata), and deleted via the standard JSON
// operations.
//
// Fields marked "Read-only" are set by the server and ignored if sent in a
// write request.
type Attachable struct {
	IDValue
	SyncTokenValue
	CDCFields

	MetaDataValue

	// Conditionally required — one of Note or FileName must be present.
	Note     string `json:"Note,omitempty"`
	FileName string `json:"FileName,omitempty"`

	// Optional
	AttachableRef []AttachableRef        `json:"AttachableRef,omitempty"`
	Category      AttachableCategoryEnum `json:"Category,omitempty"`
	ContentType   AttachmentContentType  `json:"ContentType,omitempty"`
	Tag           string                 `json:"Tag,omitempty"`
	Lat           string                 `json:"Lat,omitempty"`
	Long          string                 `json:"Long,omitempty"`
	PlaceName     string                 `json:"PlaceName,omitempty"`

	// Read-only: set by the server.
	FileAccessUri            string      `json:"FileAccessUri,omitempty"`
	ThumbnailFileAccessUri   string      `json:"ThumbnailFileAccessUri,omitempty"`
	TempDownloadUri          string      `json:"TempDownloadUri,omitempty"`
	ThumbnailTempDownloadUri string      `json:"ThumbnailTempDownloadUri,omitempty"`
	Size                     json.Number `json:"Size,omitempty"`
}

func (Attachable) EntityName() EntityName {
	return EntityAttachable
}

type attachableEnvelope struct {
	Attachable Attachable `json:"Attachable"`
}

type attachableQueryEnvelope struct {
	QueryResponse QueryResponse `json:"QueryResponse"`
}

// attachableUploadEnvelope decodes the upload response which wraps results in
// an array: {"AttachableResponse":[{"Attachable":{...}}]}.
type attachableUploadEnvelope struct {
	AttachableResponse []attachableEnvelope `json:"AttachableResponse"`
}

func (s *AttachableService) Get(ctx context.Context, id string, opts ...RequestOption) (*Attachable, error) {
	var out attachableEnvelope
	if err := s.doJSON(ctx, http.MethodGet, "attachable/"+id, nil, nil, &out, opts...); err != nil {
		return nil, err
	}
	return &out.Attachable, nil
}

func (s *AttachableService) Query(ctx context.Context, query string, opts ...QueryOption) ([]Attachable, error) {
	query = applyQueryOptions(query, opts...)

	var out attachableQueryEnvelope
	if err := s.doJSON(ctx, http.MethodGet, "query", url.Values{"query": []string{query}}, nil, &out); err != nil {
		return nil, err
	}

	raw, err := out.QueryResponse.Entities(EntityAttachable)
	if err != nil {
		return nil, err
	}

	return decodeEntitySlice[Attachable](raw)
}

func (s *AttachableService) List(ctx context.Context, opts ...QueryOption) ([]Attachable, error) {
	return s.Query(ctx, "SELECT * FROM Attachable", opts...)
}

func (s *AttachableService) ListPage(ctx context.Context, page, pageSize int, opts ...QueryOption) ([]Attachable, error) {
	pageOpts := append([]QueryOption{WithPage(page, pageSize)}, opts...)
	return s.List(ctx, pageOpts...)
}

// Create creates a note-only Attachable via a JSON POST. To upload a file
// attachment, use Upload instead.
func (s *AttachableService) Create(ctx context.Context, a Attachable, opts ...RequestOption) (*Attachable, error) {
	var out attachableEnvelope
	if err := s.doJSON(ctx, http.MethodPost, "attachable", nil, a, &out, opts...); err != nil {
		return nil, err
	}
	return &out.Attachable, nil
}

func (s *AttachableService) Update(ctx context.Context, a Attachable, opts ...RequestOption) (*Attachable, error) {
	var out attachableEnvelope
	if err := s.doJSON(ctx, http.MethodPost, "attachable", nil, a, &out, opts...); err != nil {
		return nil, err
	}
	return &out.Attachable, nil
}

func (s *AttachableService) UpdateSparse(ctx context.Context, a Attachable, opts ...RequestOption) (*Attachable, error) {
	payload := struct {
		Attachable
		Sparse bool `json:"sparse"`
	}{a, true}
	var out attachableEnvelope
	if err := s.doJSON(ctx, http.MethodPost, "attachable", nil, payload, &out, opts...); err != nil {
		return nil, err
	}
	return &out.Attachable, nil
}

func (s *AttachableService) Delete(ctx context.Context, req DeleteRequest, opts ...RequestOption) (*Attachable, error) {
	var out attachableEnvelope
	query := url.Values{"operation": []string{"delete"}}
	if err := s.doJSON(ctx, http.MethodPost, "attachable", query, req, &out, opts...); err != nil {
		return nil, err
	}
	return &out.Attachable, nil
}

// Download retrieves a temporary pre-signed URL for the specified attachment.
// The URL expires after 15 minutes; the caller is responsible for fetching the
// file from the returned URL as a separate step outside this SDK.
func (s *AttachableService) Download(ctx context.Context, id string, opts ...RequestOption) (string, error) {
	raw, err := s.doBytes(ctx, http.MethodGet, "download/"+id, nil, "text/plain", opts...)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

// GetFile fetches the file content for the specified attachment as a buffered
// byte slice. It first calls Download to obtain the pre-signed URL, then
// fetches the file from that URL using the client's HTTP transport. The
// pre-signed URL expires after 15 minutes.
func (s *AttachableService) GetFile(ctx context.Context, id string, opts ...RequestOption) ([]byte, error) {
	downloadURL, err := s.Download(ctx, id, opts...)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, downloadURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create file request: %w", err)
	}

	resp, err := s.company.client.HTTPClient().Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch file: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read file response: %w", err)
	}
	return data, nil
}

// StreamFile fetches the file content for the specified attachment as a
// streaming io.ReadCloser. It first calls Download to obtain the pre-signed
// URL, then opens a streaming connection to that URL using the client's HTTP
// transport. The caller is responsible for closing the returned ReadCloser.
// The pre-signed URL expires after 15 minutes.
func (s *AttachableService) StreamFile(ctx context.Context, id string, opts ...RequestOption) (io.ReadCloser, error) {
	downloadURL, err := s.Download(ctx, id, opts...)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, downloadURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create file request: %w", err)
	}

	resp, err := s.company.client.HTTPClient().Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch file: %w", err)
	}

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		resp.Body.Close()
		return nil, fmt.Errorf("file fetch returned status %d", resp.StatusCode)
	}

	return resp.Body, nil
}

// UploadFile pairs an Attachable's metadata with the file content to upload.
// Content is streamed into the multipart request; it is read at most
// MaxAttachmentFileSize bytes.
type UploadFile struct {
	Meta    Attachable
	Content io.Reader
}

// Attachment upload size limits, enforced client-side so oversized uploads
// fail fast with a clear error instead of a slow round trip. They are
// variables (not constants) so integrators can adjust them if the API limit
// differs; set them once at startup. Defaults mirror the QuickBooks limits:
// 20 MiB per file and 100 MiB per request.
var (
	MaxAttachmentFileSize    int64 = 20 << 20
	MaxAttachmentRequestSize int64 = 100 << 20
)

// Upload uploads one or more files to QBO in a single multipart/form-data POST
// to /upload and returns the created Attachables in the same order.
//
// Each file's type is validated by its FileName extension against the approved
// set (see AttachmentTypeForFilename), and the metadata's ContentType is
// normalised to the canonical value for that extension. A file whose extension
// is not approved, a file larger than MaxAttachmentFileSize, or a combined
// request larger than MaxAttachmentRequestSize is rejected before the request
// is sent.
//
// An entry in the response without an Attachable (empty Id) yields an error
// carrying the raw response, so a QBO-side rejection is never silently treated
// as success.
func (s *AttachableService) Upload(ctx context.Context, files ...UploadFile) ([]Attachable, error) {
	if len(files) == 0 {
		return nil, fmt.Errorf("upload: no files provided")
	}

	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)

	var total int64
	for i, f := range files {
		ct, ok := AttachmentTypeForFilename(f.Meta.FileName)
		if !ok {
			return nil, fmt.Errorf("upload: file %q has an unsupported type; approved extensions: %s",
				f.Meta.FileName, strings.Join(ApprovedAttachmentExtensions(), ", "))
		}
		f.Meta.ContentType = ct

		n := i + 1

		// Part: file bytes, bounded by the per-file size limit.
		filePart, err := mw.CreateFormFile(fmt.Sprintf("file_content_%02d", n), f.Meta.FileName)
		if err != nil {
			return nil, fmt.Errorf("upload: create file part for %q: %w", f.Meta.FileName, err)
		}
		written, err := io.CopyN(filePart, f.Content, MaxAttachmentFileSize+1)
		if err != nil && err != io.EOF {
			return nil, fmt.Errorf("upload: read content for %q: %w", f.Meta.FileName, err)
		}
		if written > MaxAttachmentFileSize {
			return nil, fmt.Errorf("upload: file %q exceeds the %d-byte per-file limit", f.Meta.FileName, MaxAttachmentFileSize)
		}
		total += written
		if total > MaxAttachmentRequestSize {
			return nil, fmt.Errorf("upload: combined file size exceeds the %d-byte request limit", MaxAttachmentRequestSize)
		}

		// Part: JSON metadata, paired with the file by the numeric suffix.
		metaBytes, err := json.Marshal(f.Meta)
		if err != nil {
			return nil, fmt.Errorf("upload: marshal metadata for %q: %w", f.Meta.FileName, err)
		}
		metaHeader := make(textproto.MIMEHeader)
		metaHeader.Set("Content-Disposition", fmt.Sprintf(`form-data; name="file_metadata_%02d"`, n))
		metaHeader.Set("Content-Type", "application/json")
		metaPart, err := mw.CreatePart(metaHeader)
		if err != nil {
			return nil, fmt.Errorf("upload: create metadata part for %q: %w", f.Meta.FileName, err)
		}
		if _, err := metaPart.Write(metaBytes); err != nil {
			return nil, fmt.Errorf("upload: write metadata for %q: %w", f.Meta.FileName, err)
		}
	}

	if err := mw.Close(); err != nil {
		return nil, fmt.Errorf("upload: close multipart writer: %w", err)
	}

	raw, err := s.doMultipart(ctx, "upload", nil, &buf, mw.FormDataContentType())
	if err != nil {
		return nil, err
	}

	var out attachableUploadEnvelope
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("upload: decode response: %w", err)
	}
	if len(out.AttachableResponse) == 0 {
		return nil, fmt.Errorf("upload: response contained no attachable entries (raw: %s)", string(raw))
	}

	result := make([]Attachable, 0, len(out.AttachableResponse))
	for i, entry := range out.AttachableResponse {
		if entry.Attachable.Id == "" {
			return nil, fmt.Errorf("upload: response entry %d contained no attachable (raw: %s)", i, string(raw))
		}
		result = append(result, entry.Attachable)
	}
	return result, nil
}

func (s *AttachableService) ChangeDataCapture(ctx context.Context, changedSince time.Time, opts ...RequestOption) ([]Attachable, error) {
	resp, err := s.company.ChangeDataCapture(ctx, changedSince, []EntityName{EntityAttachable}, opts...)
	if err != nil {
		return nil, err
	}
	return s.ExtractCDC(resp)
}

func (s *AttachableService) ExtractCDC(resp *CDCResponse) ([]Attachable, error) {
	return ExtractCDCEntities[Attachable](resp, EntityAttachable)
}

func (s *AttachableService) ExtractBatch(resp *BatchResponse) ([]Attachable, error) {
	return ExtractBatchEntities[Attachable](resp, EntityAttachable)
}

func (s *AttachableService) ExtractBatchItems(items []BatchItemResponse) ([]Attachable, error) {
	return ExtractBatchItemEntities[Attachable](items, EntityAttachable)
}

func (s *AttachableService) BatchCreateItem(id string, a Attachable) BatchItemRequest {
	return NewBatchEntity(id, BatchOperationCreate, EntityAttachable, a)
}

func (s *AttachableService) BatchUpdateItem(id string, a Attachable) BatchItemRequest {
	return NewBatchEntity(id, BatchOperationUpdate, EntityAttachable, a)
}

func (s *AttachableService) BatchDeleteItem(id string, req DeleteRequest) BatchItemRequest {
	return NewBatchEntity(id, BatchOperationDelete, EntityAttachable, req)
}

func (s *AttachableService) BatchQueryItem(id, query string, opts ...QueryOption) BatchItemRequest {
	return NewBatchQuery(id, query, opts...)
}
