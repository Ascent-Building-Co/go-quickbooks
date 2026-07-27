package quickbooks

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"time"
)

type BillPaymentTypeEnum string

const (
	CheckBillPaymentType      BillPaymentTypeEnum = "Check"
	CreditCardBillPaymentType BillPaymentTypeEnum = "CreditCard"
)

// BillPaymentCheck holds the check payment details for a BillPayment.
// BankAccountRef is required when PayType is Check; PrintStatus is optional.
type BillPaymentCheck struct {
	BankAccountRef *ReferenceType  `json:"BankAccountRef,omitempty"`
	PrintStatus    PrintStatusEnum `json:"PrintStatus,omitempty"`
}

// BillPaymentCreditCard holds the credit card payment details for a
// BillPayment. CCAccountRef is required when PayType is CreditCard.
// CreditChargeInfo and CreditChargeResponse are read-only server-set fields.
type BillPaymentCreditCard struct {
	CCAccountRef         *ReferenceType        `json:"CCAccountRef,omitempty"`
	CreditChargeInfo     *CreditChargeInfo     `json:"CreditChargeInfo,omitempty"`
	CreditChargeResponse *CreditChargeResponse `json:"CreditChargeResponse,omitempty"`
}

type BillPaymentService struct {
	companyService
}

func (c *CompanyClient) BillPayments() *BillPaymentService {
	return &BillPaymentService{companyService: companyService{company: c}}
}

// BillPayment represents a QuickBooks Online BillPayment entity, recording a
// payment made against one or more outstanding Bills.
//
// PayType determines which of CheckPayment or CreditCardPayment is active.
// Fields marked "Read-only" are set by the server and ignored if sent in a
// write request.
type BillPayment struct {
	IDValue
	SyncTokenValue
	CDCFields

	MetaData *ModificationMetaData `json:"MetaData,omitempty"`

	// Required
	VendorRef *ReferenceType      `json:"VendorRef,omitempty"`
	Line      []BillPaymentLine   `json:"Line,omitempty"`
	TotalAmt  json.Number         `json:"TotalAmt,omitempty"`
	PayType   BillPaymentTypeEnum `json:"PayType,omitempty"`

	// Conditionally required
	CurrencyRef       *ReferenceType         `json:"CurrencyRef,omitempty"`
	CheckPayment      *BillPaymentCheck      `json:"CheckPayment,omitempty"`
	CreditCardPayment *BillPaymentCreditCard `json:"CreditCardPayment,omitempty"`

	// Optional
	DocNumber               string         `json:"DocNumber,omitempty"`
	PrivateNote             string         `json:"PrivateNote,omitempty"`
	TxnDate                 *Date          `json:"TxnDate,omitempty"`
	ExchangeRate            json.Number    `json:"ExchangeRate,omitempty"`
	APAccountRef            *ReferenceType `json:"APAccountRef,omitempty"`
	DepartmentRef           *ReferenceType `json:"DepartmentRef,omitempty"`
	TransactionLocationType string         `json:"TransactionLocationType,omitempty"` // France locale only.
	ProcessBillPayment      *bool          `json:"ProcessBillPayment,omitempty"`
}

func (BillPayment) EntityName() EntityName {
	return EntityBillPayment
}

// BillPaymentLine represents a single line on a BillPayment, linking the
// payment to a specific Bill transaction.
type BillPaymentLine struct {
	Amount    json.Number `json:"Amount,omitempty"`
	LinkedTxn []LinkedTxn `json:"LinkedTxn,omitempty"`
}

type billPaymentEnvelope struct {
	BillPayment BillPayment `json:"BillPayment"`
}

type billPaymentQueryEnvelope struct {
	QueryResponse QueryResponse `json:"QueryResponse"`
}

func (s *BillPaymentService) Get(ctx context.Context, id string, opts ...RequestOption) (*BillPayment, error) {
	var out billPaymentEnvelope
	if err := s.doJSON(ctx, http.MethodGet, "billpayment/"+id, nil, nil, &out, opts...); err != nil {
		return nil, err
	}
	return &out.BillPayment, nil
}

func (s *BillPaymentService) Query(ctx context.Context, query string, opts ...QueryOption) ([]BillPayment, error) {
	query = applyQueryOptions(query, opts...)

	var out billPaymentQueryEnvelope
	if err := s.doJSON(ctx, http.MethodGet, "query", url.Values{"query": []string{query}}, nil, &out); err != nil {
		return nil, err
	}

	raw, err := out.QueryResponse.Entities(EntityBillPayment)
	if err != nil {
		return nil, err
	}

	return decodeEntitySlice[BillPayment](raw)
}

func (s *BillPaymentService) List(ctx context.Context, opts ...QueryOption) ([]BillPayment, error) {
	return s.Query(ctx, "SELECT * FROM BillPayment", opts...)
}

func (s *BillPaymentService) ListPage(ctx context.Context, page, pageSize int, opts ...QueryOption) ([]BillPayment, error) {
	pageOpts := append([]QueryOption{WithPage(page, pageSize)}, opts...)
	return s.List(ctx, pageOpts...)
}

func (s *BillPaymentService) Create(ctx context.Context, bp BillPayment, opts ...RequestOption) (*BillPayment, error) {
	var out billPaymentEnvelope
	if err := s.doJSON(ctx, http.MethodPost, "billpayment", nil, bp, &out, opts...); err != nil {
		return nil, err
	}
	return &out.BillPayment, nil
}

func (s *BillPaymentService) Update(ctx context.Context, bp BillPayment, opts ...RequestOption) (*BillPayment, error) {
	var out billPaymentEnvelope
	if err := s.doJSON(ctx, http.MethodPost, "billpayment", nil, bp, &out, opts...); err != nil {
		return nil, err
	}
	return &out.BillPayment, nil
}

func (s *BillPaymentService) UpdateSparse(ctx context.Context, bp BillPayment, opts ...RequestOption) (*BillPayment, error) {
	payload := struct {
		BillPayment
		Sparse bool `json:"sparse"`
	}{bp, true}
	var out billPaymentEnvelope
	if err := s.doJSON(ctx, http.MethodPost, "billpayment", nil, payload, &out, opts...); err != nil {
		return nil, err
	}
	return &out.BillPayment, nil
}

func (s *BillPaymentService) Delete(ctx context.Context, req DeleteRequest, opts ...RequestOption) (*BillPayment, error) {
	var out billPaymentEnvelope
	query := url.Values{"operation": []string{"delete"}}
	if err := s.doJSON(ctx, http.MethodPost, "billpayment", query, req, &out, opts...); err != nil {
		return nil, err
	}
	return &out.BillPayment, nil
}

// Void voids the BillPayment identified by req. QuickBooks requires a sparse
// update with include=void: the payload must contain Id, SyncToken, and
// "sparse":true.
func (s *BillPaymentService) Void(ctx context.Context, req VoidRequest, opts ...RequestOption) (*BillPayment, error) {
	payload := struct {
		VoidRequest
		Sparse bool `json:"sparse"`
	}{req, true}
	var out billPaymentEnvelope
	query := url.Values{"operation": []string{"update"}, "include": []string{"void"}}
	if err := s.doJSON(ctx, http.MethodPost, "billpayment", query, payload, &out, opts...); err != nil {
		return nil, err
	}
	return &out.BillPayment, nil
}

func (s *BillPaymentService) ChangeDataCapture(ctx context.Context, changedSince time.Time, opts ...RequestOption) ([]BillPayment, error) {
	resp, err := s.company.ChangeDataCapture(ctx, changedSince, []EntityName{EntityBillPayment}, opts...)
	if err != nil {
		return nil, err
	}
	return s.ExtractCDC(resp)
}

func (s *BillPaymentService) ExtractCDC(resp *CDCResponse) ([]BillPayment, error) {
	return ExtractCDCEntities[BillPayment](resp, EntityBillPayment)
}

func (s *BillPaymentService) ExtractBatch(resp *BatchResponse) ([]BillPayment, error) {
	return ExtractBatchEntities[BillPayment](resp, EntityBillPayment)
}

func (s *BillPaymentService) ExtractBatchItems(items []BatchItemResponse) ([]BillPayment, error) {
	return ExtractBatchItemEntities[BillPayment](items, EntityBillPayment)
}

func (s *BillPaymentService) BatchCreateItem(id string, bp BillPayment) BatchItemRequest {
	return NewBatchEntity(id, BatchOperationCreate, EntityBillPayment, bp)
}

func (s *BillPaymentService) BatchUpdateItem(id string, bp BillPayment) BatchItemRequest {
	return NewBatchEntity(id, BatchOperationUpdate, EntityBillPayment, bp)
}

func (s *BillPaymentService) BatchDeleteItem(id string, req DeleteRequest) BatchItemRequest {
	return NewBatchEntity(id, BatchOperationDelete, EntityBillPayment, req)
}

func (s *BillPaymentService) BatchVoidItem(id string, req VoidRequest) BatchItemRequest {
	payload := struct {
		VoidRequest
		Sparse bool `json:"sparse"`
	}{req, true}
	return NewBatchVoidEntity(id, EntityBillPayment, payload)
}

func (s *BillPaymentService) BatchQueryItem(id, query string, opts ...QueryOption) BatchItemRequest {
	return NewBatchQuery(id, query, opts...)
}
