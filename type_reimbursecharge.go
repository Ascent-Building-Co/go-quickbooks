package quickbooks

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"time"
)

// ReimburseChargeService provides access to the ReimburseCharge-related
// endpoints of the QuickBooks Online API.
type ReimburseChargeService struct {
	companyService
}

func (c *CompanyClient) ReimburseCharges() *ReimburseChargeService {
	return &ReimburseChargeService{companyService: companyService{company: c}}
}

// ReimburseLineDetail holds the detail fields for a ReimburseChargeLine.
// All fields are optional per the v75 API schema.
type ReimburseLineDetail struct {
	ItemRef        *ReferenceType `json:"ItemRef,omitempty"`
	ItemAccountRef *ReferenceType `json:"ItemAccountRef,omitempty"`
	ClassRef       *ReferenceType `json:"ClassRef,omitempty"`
	TaxCodeRef     *ReferenceType `json:"TaxCodeRef,omitempty"`
	MarkupInfo     *MarkupInfo    `json:"MarkupInfo,omitempty"`
	Qty            json.Number    `json:"Qty,omitempty"`
	UnitPrice      json.Number    `json:"UnitPrice,omitempty"`
}

// ReimburseChargeLine represents a single line on a ReimburseCharge.
// DetailType must be set to ReimburseLineDetailType and ReimburseLineDetail
// is required.
type ReimburseChargeLine struct {
	IDValue
	DetailType          LineDetailType       `json:"DetailType,omitempty"`
	ReimburseLineDetail *ReimburseLineDetail `json:"ReimburseLineDetail,omitempty"`
	LinkedTxn           []LinkedTxn          `json:"LinkedTxn,omitempty"`
	LineNum             json.Number          `json:"LineNum,omitempty"`
	Amount              json.Number          `json:"Amount,omitempty"`
	Description         string               `json:"Description,omitempty"`
}

// ReimburseCharge represents a QuickBooks Online ReimburseCharge entity,
// which records a reimbursable expense charge against a customer that can
// later be linked to an invoice.
//
// ReimburseCharge is read-only via the QBO API — CREATE and UPDATE are not
// supported. Use QUERY or READ to retrieve reimbursable charges.
//
// Fields marked "Read-only" are set by the server and ignored if sent in a
// write request.
type ReimburseCharge struct {
	IDValue
	SyncTokenValue
	CDCFields

	MetaDataValue

	// Required
	CustomerRef *ReferenceType        `json:"CustomerRef,omitempty"`
	Amount      json.Number           `json:"Amount,omitempty"`
	Line        []ReimburseChargeLine `json:"Line,omitempty"`

	// Conditionally required
	CurrencyRef *ReferenceType `json:"CurrencyRef,omitempty"`

	// Optional
	PrivateNote  string      `json:"PrivateNote,omitempty"`
	LinkedTxn    []LinkedTxn `json:"LinkedTxn,omitempty"`
	ExchangeRate json.Number `json:"ExchangeRate,omitempty"`
	TxnDate      *Date       `json:"TxnDate,omitempty"`

	// Read-only: set by the server.
	HasBeenInvoiced *bool       `json:"HasBeenInvoiced,omitempty"`
	HomeTotalAmt    json.Number `json:"HomeTotalAmt,omitempty"`
}

func (ReimburseCharge) EntityName() EntityName {
	return EntityReimburseCharge
}

type reimburseChargeEnvelope struct {
	ReimburseCharge ReimburseCharge `json:"ReimburseCharge"`
}

type reimburseChargeQueryEnvelope struct {
	QueryResponse QueryResponse `json:"QueryResponse"`
}

func (s *ReimburseChargeService) Get(ctx context.Context, id string, opts ...RequestOption) (*ReimburseCharge, error) {
	var out reimburseChargeEnvelope
	if err := s.doJSON(ctx, http.MethodGet, "reimbursecharge/"+id, nil, nil, &out, opts...); err != nil {
		return nil, err
	}
	return &out.ReimburseCharge, nil
}

func (s *ReimburseChargeService) Query(ctx context.Context, query string, opts ...QueryOption) ([]ReimburseCharge, error) {
	query = applyQueryOptions(query, opts...)

	var out reimburseChargeQueryEnvelope
	if err := s.doJSON(ctx, http.MethodGet, "query", url.Values{"query": []string{query}}, nil, &out); err != nil {
		return nil, err
	}

	raw, err := out.QueryResponse.Entities(EntityReimburseCharge)
	if err != nil {
		return nil, err
	}

	return decodeEntitySlice[ReimburseCharge](raw)
}

func (s *ReimburseChargeService) List(ctx context.Context, opts ...QueryOption) ([]ReimburseCharge, error) {
	return s.Query(ctx, "SELECT * FROM ReimburseCharge", opts...)
}

func (s *ReimburseChargeService) ListPage(ctx context.Context, page, pageSize int, opts ...QueryOption) ([]ReimburseCharge, error) {
	pageOpts := append([]QueryOption{WithPage(page, pageSize)}, opts...)
	return s.List(ctx, pageOpts...)
}

func (s *ReimburseChargeService) ChangeDataCapture(ctx context.Context, changedSince time.Time, opts ...RequestOption) ([]ReimburseCharge, error) {
	resp, err := s.company.ChangeDataCapture(ctx, changedSince, []EntityName{EntityReimburseCharge}, opts...)
	if err != nil {
		return nil, err
	}
	return s.ExtractCDC(resp)
}

func (s *ReimburseChargeService) ExtractCDC(resp *CDCResponse) ([]ReimburseCharge, error) {
	return ExtractCDCEntities[ReimburseCharge](resp, EntityReimburseCharge)
}

func (s *ReimburseChargeService) ExtractBatch(resp *BatchResponse) ([]ReimburseCharge, error) {
	return ExtractBatchEntities[ReimburseCharge](resp, EntityReimburseCharge)
}

func (s *ReimburseChargeService) ExtractBatchItems(items []BatchItemResponse) ([]ReimburseCharge, error) {
	return ExtractBatchItemEntities[ReimburseCharge](items, EntityReimburseCharge)
}

func (s *ReimburseChargeService) BatchQueryItem(id, query string, opts ...QueryOption) BatchItemRequest {
	return NewBatchQuery(id, query, opts...)
}
