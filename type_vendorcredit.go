package quickbooks

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"time"
)

type VendorCreditService struct {
	companyService
}

func (c *CompanyClient) VendorCredits() *VendorCreditService {
	return &VendorCreditService{companyService: companyService{company: c}}
}

// VendorCredit represents a QuickBooks Online VendorCredit entity, recording
// a credit issued by a vendor that reduces amounts owed to that vendor.
//
// Fields marked "Read-only" are set by the server and ignored if sent in a
// write request.
//
// Balance is calculated by QuickBooks and reflects the remaining unapplied
// credit amount. It is functionally read-only despite being listed as Optional
// in the v75 schema.
type VendorCredit struct {
	IDValue
	SyncTokenValue
	CDCFields

	MetaData *ModificationMetaData `json:"MetaData,omitempty"`

	// Required
	VendorRef *ReferenceType     `json:"VendorRef,omitempty"`
	Line      []VendorCreditLine `json:"Line,omitempty"`

	// Conditionally required
	CurrencyRef          *ReferenceType           `json:"CurrencyRef,omitempty"`
	GlobalTaxCalculation GlobalTaxCalculationEnum `json:"GlobalTaxCalculation,omitempty"`

	// Optional
	DocNumber               string           `json:"DocNumber,omitempty"`
	PrivateNote             string           `json:"PrivateNote,omitempty"`
	LinkedTxn               []LinkedTxn      `json:"LinkedTxn,omitempty"`
	ExchangeRate            json.Number      `json:"ExchangeRate,omitempty"`
	APAccountRef            *ReferenceType   `json:"APAccountRef,omitempty"`
	DepartmentRef           *ReferenceType   `json:"DepartmentRef,omitempty"`
	TxnDate                 *Date            `json:"TxnDate,omitempty"`
	IncludeInAnnualTPAR     *bool            `json:"IncludeInAnnualTPAR,omitempty"`
	TransactionLocationType string           `json:"TransactionLocationType,omitempty"` // France locale only.
	Balance                 json.Number      `json:"Balance,omitempty"`
	VendorAddr              *PhysicalAddress `json:"VendorAddr,omitempty"`

	// Read-only: set by the server.
	TotalAmt     json.Number    `json:"TotalAmt,omitempty"`
	RecurDataRef *ReferenceType `json:"RecurDataRef,omitempty"`
}

func (VendorCredit) EntityName() EntityName {
	return EntityVendorCredit
}

// VendorCreditLine represents a single line item on a VendorCredit. The valid
// detail types are ItemBasedExpenseLine and AccountBasedExpenseLine, reusing
// the shared detail structs from definitions.go.
//
// ProjectRef is not part of the v75 schema model but appears in API payloads
// when the line is associated with a QuickBooks project.
type VendorCreditLine struct {
	IDValue

	LineNum                       json.Number                    `json:"LineNum,omitempty"`
	Description                   string                         `json:"Description,omitempty"`
	Amount                        json.Number                    `json:"Amount,omitempty"`
	DetailType                    LineDetailType                 `json:"DetailType,omitempty"`
	ProjectRef                    *ReferenceType                 `json:"ProjectRef,omitempty"`
	ItemBasedExpenseLineDetail    *ItemBasedExpenseLineDetail    `json:"ItemBasedExpenseLineDetail,omitempty"`
	AccountBasedExpenseLineDetail *AccountBasedExpenseLineDetail `json:"AccountBasedExpenseLineDetail,omitempty"`
}

type vendorCreditEnvelope struct {
	VendorCredit VendorCredit `json:"VendorCredit"`
}

type vendorCreditQueryEnvelope struct {
	QueryResponse QueryResponse `json:"QueryResponse"`
}

func (s *VendorCreditService) Get(ctx context.Context, id string, opts ...RequestOption) (*VendorCredit, error) {
	var out vendorCreditEnvelope
	if err := s.doJSON(ctx, http.MethodGet, "vendorcredit/"+id, nil, nil, &out, opts...); err != nil {
		return nil, err
	}
	return &out.VendorCredit, nil
}

func (s *VendorCreditService) Query(ctx context.Context, query string, opts ...QueryOption) ([]VendorCredit, error) {
	query = applyQueryOptions(query, opts...)

	var out vendorCreditQueryEnvelope
	if err := s.doJSON(ctx, http.MethodGet, "query", url.Values{"query": []string{query}}, nil, &out); err != nil {
		return nil, err
	}

	raw, err := out.QueryResponse.Entities(EntityVendorCredit)
	if err != nil {
		return nil, err
	}

	return decodeEntitySlice[VendorCredit](raw)
}

func (s *VendorCreditService) List(ctx context.Context, opts ...QueryOption) ([]VendorCredit, error) {
	return s.Query(ctx, "SELECT * FROM VendorCredit", opts...)
}

func (s *VendorCreditService) ListPage(ctx context.Context, page, pageSize int, opts ...QueryOption) ([]VendorCredit, error) {
	pageOpts := append([]QueryOption{WithPage(page, pageSize)}, opts...)
	return s.List(ctx, pageOpts...)
}

func (s *VendorCreditService) Create(ctx context.Context, vc VendorCredit, opts ...RequestOption) (*VendorCredit, error) {
	var out vendorCreditEnvelope
	if err := s.doJSON(ctx, http.MethodPost, "vendorcredit", nil, vc, &out, opts...); err != nil {
		return nil, err
	}
	return &out.VendorCredit, nil
}

func (s *VendorCreditService) Update(ctx context.Context, vc VendorCredit, opts ...RequestOption) (*VendorCredit, error) {
	var out vendorCreditEnvelope
	if err := s.doJSON(ctx, http.MethodPost, "vendorcredit", nil, vc, &out, opts...); err != nil {
		return nil, err
	}
	return &out.VendorCredit, nil
}

func (s *VendorCreditService) UpdateSparse(ctx context.Context, vc VendorCredit, opts ...RequestOption) (*VendorCredit, error) {
	payload := struct {
		VendorCredit
		Sparse bool `json:"sparse"`
	}{vc, true}
	var out vendorCreditEnvelope
	if err := s.doJSON(ctx, http.MethodPost, "vendorcredit", nil, payload, &out, opts...); err != nil {
		return nil, err
	}
	return &out.VendorCredit, nil
}

func (s *VendorCreditService) Delete(ctx context.Context, req DeleteRequest, opts ...RequestOption) (*VendorCredit, error) {
	var out vendorCreditEnvelope
	query := url.Values{"operation": []string{"delete"}}
	if err := s.doJSON(ctx, http.MethodPost, "vendorcredit", query, req, &out, opts...); err != nil {
		return nil, err
	}
	return &out.VendorCredit, nil
}

func (s *VendorCreditService) ChangeDataCapture(ctx context.Context, changedSince time.Time, opts ...RequestOption) ([]VendorCredit, error) {
	resp, err := s.company.ChangeDataCapture(ctx, changedSince, []EntityName{EntityVendorCredit}, opts...)
	if err != nil {
		return nil, err
	}
	return s.ExtractCDC(resp)
}

func (s *VendorCreditService) ExtractCDC(resp *CDCResponse) ([]VendorCredit, error) {
	return ExtractCDCEntities[VendorCredit](resp, EntityVendorCredit)
}

func (s *VendorCreditService) ExtractBatch(resp *BatchResponse) ([]VendorCredit, error) {
	return ExtractBatchEntities[VendorCredit](resp, EntityVendorCredit)
}

func (s *VendorCreditService) ExtractBatchItems(items []BatchItemResponse) ([]VendorCredit, error) {
	return ExtractBatchItemEntities[VendorCredit](items, EntityVendorCredit)
}

func (s *VendorCreditService) BatchCreateItem(id string, vc VendorCredit) BatchItemRequest {
	return NewBatchEntity(id, BatchOperationCreate, EntityVendorCredit, vc)
}

func (s *VendorCreditService) BatchUpdateItem(id string, vc VendorCredit) BatchItemRequest {
	return NewBatchEntity(id, BatchOperationUpdate, EntityVendorCredit, vc)
}

func (s *VendorCreditService) BatchDeleteItem(id string, req DeleteRequest) BatchItemRequest {
	return NewBatchEntity(id, BatchOperationDelete, EntityVendorCredit, req)
}

func (s *VendorCreditService) BatchQueryItem(id, query string, opts ...QueryOption) BatchItemRequest {
	return NewBatchQuery(id, query, opts...)
}
