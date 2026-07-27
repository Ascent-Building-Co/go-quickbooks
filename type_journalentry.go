package quickbooks

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"time"
)

// JournalEntryEntity identifies the customer or vendor associated with a
// journal entry line.
type JournalEntryEntity struct {
	Type      string         `json:"Type,omitempty"`
	EntityRef *ReferenceType `json:"EntityRef,omitempty"`
}

// JournalEntryLineDetail holds the detail fields for a JournalEntryLine.
// AccountRef and PostingType are required. JournalCodeRef is required for
// France locales only; include it when targeting French-locale companies.
type JournalEntryLineDetail struct {
	AccountRef      *ReferenceType      `json:"AccountRef,omitempty"`
	PostingType     PostingTypeEnum     `json:"PostingType,omitempty"`
	Entity          *JournalEntryEntity `json:"Entity,omitempty"`
	ClassRef        *ReferenceType      `json:"ClassRef,omitempty"`
	DepartmentRef   *ReferenceType      `json:"DepartmentRef,omitempty"`
	TaxCodeRef      *ReferenceType      `json:"TaxCodeRef,omitempty"`
	JournalCodeRef  *ReferenceType      `json:"JournalCodeRef,omitempty"`
	BillableStatus  BillableStatusEnum  `json:"BillableStatus,omitempty"`
	TaxApplicableOn string              `json:"TaxApplicableOn,omitempty"`
	TaxInclusiveAmt json.Number         `json:"TaxInclusiveAmt,omitempty"`

	// Read-only: set by the server.
	TaxAmount json.Number `json:"TaxAmount,omitempty"`
}

type JournalEntryService struct {
	companyService
}

func (c *CompanyClient) JournalEntries() *JournalEntryService {
	return &JournalEntryService{companyService: companyService{company: c}}
}

// JournalEntry represents a QuickBooks Online JournalEntry entity, recording
// a manual accounting entry with debit and credit lines.
//
// Fields marked "Read-only" are set by the server and ignored if sent in a
// write request.
type JournalEntry struct {
	IDValue
	SyncTokenValue
	CDCFields

	MetaData *ModificationMetaData `json:"MetaData,omitempty"`

	// Required
	Line []JournalEntryLine `json:"Line,omitempty"`

	// Conditionally required
	CurrencyRef          *ReferenceType           `json:"CurrencyRef,omitempty"`
	GlobalTaxCalculation GlobalTaxCalculationEnum `json:"GlobalTaxCalculation,omitempty"`

	// Optional
	DocNumber               string         `json:"DocNumber,omitempty"`
	PrivateNote             string         `json:"PrivateNote,omitempty"`
	TxnDate                 *Date          `json:"TxnDate,omitempty"`
	ExchangeRate            json.Number    `json:"ExchangeRate,omitempty"`
	TaxRateRef              *ReferenceType `json:"TaxRateRef,omitempty"`
	TransactionLocationType string         `json:"TransactionLocationType,omitempty"`
	TxnTaxDetail            *TxnTaxDetail  `json:"TxnTaxDetail,omitempty"`
	Adjustment              *bool          `json:"Adjustment,omitempty"`

	// Read-only: set by the server.
	TotalAmt     json.Number    `json:"TotalAmt,omitempty"`
	HomeTotalAmt json.Number    `json:"HomeTotalAmt,omitempty"`
	RecurDataRef *ReferenceType `json:"RecurDataRef,omitempty"`
}

func (JournalEntry) EntityName() EntityName {
	return EntityJournalEntry
}

// JournalEntryLine represents a single line on a JournalEntry. The valid
// detail types are JournalEntryLineDetail (for debit/credit accounting lines)
// and DescriptionOnly (for memo-only lines).
//
// ProjectRef is not part of the v75 schema model but appears in API payloads
// when the line is associated with a QuickBooks project.
type JournalEntryLine struct {
	IDValue

	LineNum                json.Number             `json:"LineNum,omitempty"`
	Description            string                  `json:"Description,omitempty"`
	Amount                 json.Number             `json:"Amount,omitempty"`
	DetailType             LineDetailType          `json:"DetailType,omitempty"`
	ProjectRef             *ReferenceType          `json:"ProjectRef,omitempty"`
	JournalEntryLineDetail *JournalEntryLineDetail `json:"JournalEntryLineDetail,omitempty"`
	DescriptionLineDetail  *DescriptionLineDetail  `json:"DescriptionLineDetail,omitempty"`
}

type journalEntryEnvelope struct {
	JournalEntry JournalEntry `json:"JournalEntry"`
}

type journalEntryQueryEnvelope struct {
	QueryResponse QueryResponse `json:"QueryResponse"`
}

func (s *JournalEntryService) Get(ctx context.Context, id string, opts ...RequestOption) (*JournalEntry, error) {
	var out journalEntryEnvelope
	if err := s.doJSON(ctx, http.MethodGet, "journalentry/"+id, nil, nil, &out, opts...); err != nil {
		return nil, err
	}
	return &out.JournalEntry, nil
}

func (s *JournalEntryService) Query(ctx context.Context, query string, opts ...QueryOption) ([]JournalEntry, error) {
	query = applyQueryOptions(query, opts...)

	var out journalEntryQueryEnvelope
	if err := s.doJSON(ctx, http.MethodGet, "query", url.Values{"query": []string{query}}, nil, &out); err != nil {
		return nil, err
	}

	raw, err := out.QueryResponse.Entities(EntityJournalEntry)
	if err != nil {
		return nil, err
	}

	return decodeEntitySlice[JournalEntry](raw)
}

func (s *JournalEntryService) List(ctx context.Context, opts ...QueryOption) ([]JournalEntry, error) {
	return s.Query(ctx, "SELECT * FROM JournalEntry", opts...)
}

func (s *JournalEntryService) ListPage(ctx context.Context, page, pageSize int, opts ...QueryOption) ([]JournalEntry, error) {
	pageOpts := append([]QueryOption{WithPage(page, pageSize)}, opts...)
	return s.List(ctx, pageOpts...)
}

func (s *JournalEntryService) Create(ctx context.Context, je JournalEntry, opts ...RequestOption) (*JournalEntry, error) {
	var out journalEntryEnvelope
	if err := s.doJSON(ctx, http.MethodPost, "journalentry", nil, je, &out, opts...); err != nil {
		return nil, err
	}
	return &out.JournalEntry, nil
}

func (s *JournalEntryService) Update(ctx context.Context, je JournalEntry, opts ...RequestOption) (*JournalEntry, error) {
	var out journalEntryEnvelope
	if err := s.doJSON(ctx, http.MethodPost, "journalentry", nil, je, &out, opts...); err != nil {
		return nil, err
	}
	return &out.JournalEntry, nil
}

func (s *JournalEntryService) UpdateSparse(ctx context.Context, je JournalEntry, opts ...RequestOption) (*JournalEntry, error) {
	payload := struct {
		JournalEntry
		Sparse bool `json:"sparse"`
	}{je, true}
	var out journalEntryEnvelope
	if err := s.doJSON(ctx, http.MethodPost, "journalentry", nil, payload, &out, opts...); err != nil {
		return nil, err
	}
	return &out.JournalEntry, nil
}

func (s *JournalEntryService) Delete(ctx context.Context, req DeleteRequest, opts ...RequestOption) (*JournalEntry, error) {
	var out journalEntryEnvelope
	query := url.Values{"operation": []string{"delete"}}
	if err := s.doJSON(ctx, http.MethodPost, "journalentry", query, req, &out, opts...); err != nil {
		return nil, err
	}
	return &out.JournalEntry, nil
}

func (s *JournalEntryService) ChangeDataCapture(ctx context.Context, changedSince time.Time, opts ...RequestOption) ([]JournalEntry, error) {
	resp, err := s.company.ChangeDataCapture(ctx, changedSince, []EntityName{EntityJournalEntry}, opts...)
	if err != nil {
		return nil, err
	}
	return s.ExtractCDC(resp)
}

func (s *JournalEntryService) ExtractCDC(resp *CDCResponse) ([]JournalEntry, error) {
	return ExtractCDCEntities[JournalEntry](resp, EntityJournalEntry)
}

func (s *JournalEntryService) ExtractBatch(resp *BatchResponse) ([]JournalEntry, error) {
	return ExtractBatchEntities[JournalEntry](resp, EntityJournalEntry)
}

func (s *JournalEntryService) ExtractBatchItems(items []BatchItemResponse) ([]JournalEntry, error) {
	return ExtractBatchItemEntities[JournalEntry](items, EntityJournalEntry)
}

func (s *JournalEntryService) BatchCreateItem(id string, je JournalEntry) BatchItemRequest {
	return NewBatchEntity(id, BatchOperationCreate, EntityJournalEntry, je)
}

func (s *JournalEntryService) BatchUpdateItem(id string, je JournalEntry) BatchItemRequest {
	return NewBatchEntity(id, BatchOperationUpdate, EntityJournalEntry, je)
}

func (s *JournalEntryService) BatchDeleteItem(id string, req DeleteRequest) BatchItemRequest {
	return NewBatchEntity(id, BatchOperationDelete, EntityJournalEntry, req)
}

func (s *JournalEntryService) BatchQueryItem(id, query string, opts ...QueryOption) BatchItemRequest {
	return NewBatchQuery(id, query, opts...)
}
