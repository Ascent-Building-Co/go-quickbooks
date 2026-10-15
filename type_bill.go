package quickbooks

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"time"
)

type BillService struct {
	companyService
}

func (c *CompanyClient) Bills() *BillService {
	return &BillService{companyService: companyService{company: c}}
}

// Bill represents a QuickBooks Online Bill entity (an accounts-payable
// transaction recording an amount owed to a vendor).
//
// Fields marked "Read-only" are set by the server and ignored if sent in a
// write request.
type Bill struct {
	IDValue
	SyncTokenValue
	CDCFields

	MetaDataValue

	// Required
	VendorRef *ReferenceType `json:"VendorRef,omitempty"`
	Line      []BillLine     `json:"Line,omitempty"`

	// Conditionally required
	CurrencyRef          *ReferenceType           `json:"CurrencyRef,omitempty"`
	GlobalTaxCalculation GlobalTaxCalculationEnum `json:"GlobalTaxCalculation,omitempty"`

	// Optional
	TxnDate                 *Date            `json:"TxnDate,omitempty"`
	APAccountRef            *ReferenceType   `json:"APAccountRef,omitempty"`
	SalesTermRef            *ReferenceType   `json:"SalesTermRef,omitempty"`
	LinkedTxn               []LinkedTxn      `json:"LinkedTxn,omitempty"`
	TotalAmt                json.Number      `json:"TotalAmt,omitempty"`
	TransactionLocationType string           `json:"TransactionLocationType,omitempty"` // France locale only.
	DueDate                 *Date            `json:"DueDate,omitempty"`
	DocNumber               string           `json:"DocNumber,omitempty"`
	PrivateNote             string           `json:"PrivateNote,omitempty"`
	TxnTaxDetail            *TxnTaxDetail    `json:"TxnTaxDetail,omitempty"`
	ExchangeRate            json.Number      `json:"ExchangeRate,omitempty"`
	DepartmentRef           *ReferenceType   `json:"DepartmentRef,omitempty"`
	IncludeInAnnualTPAR     *bool            `json:"IncludeInAnnualTPAR,omitempty"`
	VendorAddr              *PhysicalAddress `json:"VendorAddr,omitempty"`

	// Read-only: set by the server.
	HomeBalance  json.Number    `json:"HomeBalance,omitempty"`
	RecurDataRef *ReferenceType `json:"RecurDataRef,omitempty"`
	Balance      json.Number    `json:"Balance,omitempty"`
}

func (Bill) EntityName() EntityName {
	return EntityBill
}

// BillLine represents a single line item on a Bill. The valid detail types for
// Bill are ItemBasedExpenseLine and AccountBasedExpenseLine.
//
// ProjectRef is not part of the v75 schema model but appears in API payloads
// when the line is associated with a QuickBooks project.
type BillLine struct {
	IDValue

	LineNum                       json.Number                    `json:"LineNum,omitempty"`
	Description                   string                         `json:"Description,omitempty"`
	Amount                        json.Number                    `json:"Amount,omitempty"`
	DetailType                    LineDetailType                 `json:"DetailType,omitempty"`
	LinkedTxn                     []LinkedTxn                    `json:"LinkedTxn,omitempty"`
	ProjectRef                    *ReferenceType                 `json:"ProjectRef,omitempty"`
	ItemBasedExpenseLineDetail    *ItemBasedExpenseLineDetail    `json:"ItemBasedExpenseLineDetail,omitempty"`
	AccountBasedExpenseLineDetail *AccountBasedExpenseLineDetail `json:"AccountBasedExpenseLineDetail,omitempty"`
}

type billEnvelope struct {
	Bill Bill `json:"Bill"`
}

type billQueryEnvelope struct {
	QueryResponse QueryResponse `json:"QueryResponse"`
}

func (s *BillService) Get(ctx context.Context, id string, opts ...RequestOption) (*Bill, error) {
	var out billEnvelope
	if err := s.doJSON(ctx, http.MethodGet, "bill/"+id, nil, nil, &out, opts...); err != nil {
		return nil, err
	}
	return &out.Bill, nil
}

func (s *BillService) Query(ctx context.Context, query string, opts ...QueryOption) ([]Bill, error) {
	query = applyQueryOptions(query, opts...)

	var out billQueryEnvelope
	if err := s.doJSON(ctx, http.MethodGet, "query", url.Values{"query": []string{query}}, nil, &out); err != nil {
		return nil, err
	}

	raw, err := out.QueryResponse.Entities(EntityBill)
	if err != nil {
		return nil, err
	}

	return decodeEntitySlice[Bill](raw)
}

func (s *BillService) List(ctx context.Context, opts ...QueryOption) ([]Bill, error) {
	return s.Query(ctx, "SELECT * FROM Bill", opts...)
}

func (s *BillService) ListPage(ctx context.Context, page, pageSize int, opts ...QueryOption) ([]Bill, error) {
	pageOpts := append([]QueryOption{WithPage(page, pageSize)}, opts...)
	return s.List(ctx, pageOpts...)
}

func (s *BillService) Create(ctx context.Context, bill Bill, opts ...RequestOption) (*Bill, error) {
	var out billEnvelope
	if err := s.doJSON(ctx, http.MethodPost, "bill", nil, bill, &out, opts...); err != nil {
		return nil, err
	}
	return &out.Bill, nil
}

func (s *BillService) Update(ctx context.Context, bill Bill, opts ...RequestOption) (*Bill, error) {
	var out billEnvelope
	if err := s.doJSON(ctx, http.MethodPost, "bill", nil, bill, &out, opts...); err != nil {
		return nil, err
	}
	return &out.Bill, nil
}

func (s *BillService) UpdateSparse(ctx context.Context, bill Bill, opts ...RequestOption) (*Bill, error) {
	payload := struct {
		Bill
		Sparse bool `json:"sparse"`
	}{bill, true}
	var out billEnvelope
	if err := s.doJSON(ctx, http.MethodPost, "bill", nil, payload, &out, opts...); err != nil {
		return nil, err
	}
	return &out.Bill, nil
}

func (s *BillService) Delete(ctx context.Context, req DeleteRequest, opts ...RequestOption) (*Bill, error) {
	var out billEnvelope
	query := url.Values{"operation": []string{"delete"}}
	if err := s.doJSON(ctx, http.MethodPost, "bill", query, req, &out, opts...); err != nil {
		return nil, err
	}
	return &out.Bill, nil
}

func (s *BillService) ChangeDataCapture(ctx context.Context, changedSince time.Time, opts ...RequestOption) ([]Bill, error) {
	resp, err := s.company.ChangeDataCapture(ctx, changedSince, []EntityName{EntityBill}, opts...)
	if err != nil {
		return nil, err
	}
	return s.ExtractCDC(resp)
}

func (s *BillService) ExtractCDC(resp *CDCResponse) ([]Bill, error) {
	return ExtractCDCEntities[Bill](resp, EntityBill)
}

func (s *BillService) ExtractBatch(resp *BatchResponse) ([]Bill, error) {
	return ExtractBatchEntities[Bill](resp, EntityBill)
}

func (s *BillService) ExtractBatchItems(items []BatchItemResponse) ([]Bill, error) {
	return ExtractBatchItemEntities[Bill](items, EntityBill)
}

func (s *BillService) BatchCreateItem(id string, bill Bill) BatchItemRequest {
	return NewBatchEntity(id, BatchOperationCreate, EntityBill, bill)
}

func (s *BillService) BatchUpdateItem(id string, bill Bill) BatchItemRequest {
	return NewBatchEntity(id, BatchOperationUpdate, EntityBill, bill)
}

func (s *BillService) BatchDeleteItem(id string, req DeleteRequest) BatchItemRequest {
	return NewBatchEntity(id, BatchOperationDelete, EntityBill, req)
}

func (s *BillService) BatchQueryItem(id, query string, opts ...QueryOption) BatchItemRequest {
	return NewBatchQuery(id, query, opts...)
}
