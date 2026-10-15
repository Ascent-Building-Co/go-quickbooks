package quickbooks

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/mail"
	"net/url"
	"time"
)

type CreditMemoService struct {
	companyService
}

func (c *CompanyClient) CreditMemos() *CreditMemoService {
	return &CreditMemoService{companyService: companyService{company: c}}
}

// CreditMemo represents a QuickBooks Online CreditMemo entity, recording a
// credit issued to a customer that can be applied against future invoices.
//
// Fields marked "Read-only" are set by the server and ignored if sent in a
// write request.
type CreditMemo struct {
	IDValue
	SyncTokenValue
	CDCFields

	MetaDataValue
	CustomField []CustomField `json:"CustomField,omitempty"`

	// Required
	CustomerRef *ReferenceType   `json:"CustomerRef,omitempty"`
	Line        []CreditMemoLine `json:"Line,omitempty"`

	// Conditionally required
	CurrencyRef          *ReferenceType           `json:"CurrencyRef,omitempty"`
	GlobalTaxCalculation GlobalTaxCalculationEnum `json:"GlobalTaxCalculation,omitempty"`
	ProjectRef           *ReferenceType           `json:"ProjectRef,omitempty"`
	BillEmail            *EmailAddress            `json:"BillEmail,omitempty"`

	// Optional
	TxnDate                 *Date            `json:"TxnDate,omitempty"`
	ClassRef                *ReferenceType   `json:"ClassRef,omitempty"`
	PrintStatus             PrintStatusEnum  `json:"PrintStatus,omitempty"`
	SalesTermRef            *ReferenceType   `json:"SalesTermRef,omitempty"`
	TotalAmt                json.Number      `json:"TotalAmt,omitempty"`
	InvoiceRef              *ReferenceType   `json:"InvoiceRef,omitempty"`              // India locale only.
	TransactionLocationType string           `json:"TransactionLocationType,omitempty"` // France locale only.
	ApplyTaxAfterDiscount   *bool            `json:"ApplyTaxAfterDiscount,omitempty"`
	DocNumber               string           `json:"DocNumber,omitempty"`
	PrivateNote             string           `json:"PrivateNote,omitempty"`
	CustomerMemo            *MemoRef         `json:"CustomerMemo,omitempty"`
	TxnTaxDetail            *TxnTaxDetail    `json:"TxnTaxDetail,omitempty"`
	PaymentMethodRef        *ReferenceType   `json:"PaymentMethodRef,omitempty"`
	ExchangeRate            json.Number      `json:"ExchangeRate,omitempty"`
	ShipAddr                *PhysicalAddress `json:"ShipAddr,omitempty"`
	DepartmentRef           *ReferenceType   `json:"DepartmentRef,omitempty"`
	EmailStatus             EmailStatusEnum  `json:"EmailStatus,omitempty"`
	BillAddr                *PhysicalAddress `json:"BillAddr,omitempty"`

	// Read-only: set by the server.
	HomeBalance     json.Number    `json:"HomeBalance,omitempty"`
	RemainingCredit json.Number    `json:"RemainingCredit,omitempty"`
	RecurDataRef    *ReferenceType `json:"RecurDataRef,omitempty"`
	TaxExemptionRef *ReferenceType `json:"TaxExemptionRef,omitempty"`
	Balance         json.Number    `json:"Balance,omitempty"`
	HomeTotalAmt    json.Number    `json:"HomeTotalAmt,omitempty"`
}

func (CreditMemo) EntityName() EntityName {
	return EntityCreditMemo
}

// CreditMemoLine represents a single line item on a CreditMemo. The valid
// detail types are SalesItemLine, DescriptionOnlyLine, DiscountLine,
// GroupLine, and SubTotalLine — identical to Invoice.
type CreditMemoLine struct {
	IDValue

	LineNum               json.Number                `json:"LineNum,omitempty"`
	Description           string                     `json:"Description,omitempty"`
	Amount                json.Number                `json:"Amount,omitempty"`
	DetailType            LineDetailType             `json:"DetailType,omitempty"`
	LinkedTxn             []LinkedTxn                `json:"LinkedTxn,omitempty"`
	SalesItemLineDetail   *SalesItemLineDetail       `json:"SalesItemLineDetail,omitempty"`
	DescriptionLineDetail *DescriptionLineDetail     `json:"DescriptionLineDetail,omitempty"`
	DiscountLineDetail    *DiscountLineDetail        `json:"DiscountLineDetail,omitempty"`
	GroupLineDetail       *CreditMemoGroupLineDetail `json:"GroupLineDetail,omitempty"`
	SubTotalLineDetail    *SubTotalLineDetail        `json:"SubTotalLineDetail,omitempty"`
}

// CreditMemoGroupLineDetail is the group line detail type for CreditMemo. It
// uses []CreditMemoLine for its nested lines so that only CreditMemo-valid
// detail types are accessible within a group.
type CreditMemoGroupLineDetail struct {
	GroupItemRef *ReferenceType   `json:"GroupItemRef,omitempty"`
	Quantity     json.Number      `json:"Quantity,omitempty"`
	Line         []CreditMemoLine `json:"Line,omitempty"`
}

type creditMemoEnvelope struct {
	CreditMemo CreditMemo `json:"CreditMemo"`
}

type creditMemoQueryEnvelope struct {
	QueryResponse QueryResponse `json:"QueryResponse"`
}

func (s *CreditMemoService) Get(ctx context.Context, id string, opts ...RequestOption) (*CreditMemo, error) {
	var out creditMemoEnvelope
	if err := s.doJSON(ctx, http.MethodGet, "creditmemo/"+id, nil, nil, &out, opts...); err != nil {
		return nil, err
	}
	return &out.CreditMemo, nil
}

func (s *CreditMemoService) Query(ctx context.Context, query string, opts ...QueryOption) ([]CreditMemo, error) {
	query = applyQueryOptions(query, opts...)

	var out creditMemoQueryEnvelope
	if err := s.doJSON(ctx, http.MethodGet, "query", url.Values{"query": []string{query}}, nil, &out); err != nil {
		return nil, err
	}

	raw, err := out.QueryResponse.Entities(EntityCreditMemo)
	if err != nil {
		return nil, err
	}

	return decodeEntitySlice[CreditMemo](raw)
}

func (s *CreditMemoService) List(ctx context.Context, opts ...QueryOption) ([]CreditMemo, error) {
	return s.Query(ctx, "SELECT * FROM CreditMemo", opts...)
}

func (s *CreditMemoService) ListPage(ctx context.Context, page, pageSize int, opts ...QueryOption) ([]CreditMemo, error) {
	pageOpts := append([]QueryOption{WithPage(page, pageSize)}, opts...)
	return s.List(ctx, pageOpts...)
}

func (s *CreditMemoService) Create(ctx context.Context, cm CreditMemo, opts ...RequestOption) (*CreditMemo, error) {
	var out creditMemoEnvelope
	if err := s.doJSON(ctx, http.MethodPost, "creditmemo", nil, cm, &out, opts...); err != nil {
		return nil, err
	}
	return &out.CreditMemo, nil
}

func (s *CreditMemoService) Update(ctx context.Context, cm CreditMemo, opts ...RequestOption) (*CreditMemo, error) {
	var out creditMemoEnvelope
	if err := s.doJSON(ctx, http.MethodPost, "creditmemo", nil, cm, &out, opts...); err != nil {
		return nil, err
	}
	return &out.CreditMemo, nil
}

func (s *CreditMemoService) UpdateSparse(ctx context.Context, cm CreditMemo, opts ...RequestOption) (*CreditMemo, error) {
	payload := struct {
		CreditMemo
		Sparse bool `json:"sparse"`
	}{cm, true}
	var out creditMemoEnvelope
	if err := s.doJSON(ctx, http.MethodPost, "creditmemo", nil, payload, &out, opts...); err != nil {
		return nil, err
	}
	return &out.CreditMemo, nil
}

func (s *CreditMemoService) Delete(ctx context.Context, req DeleteRequest, opts ...RequestOption) (*CreditMemo, error) {
	var out creditMemoEnvelope
	query := url.Values{"operation": []string{"delete"}}
	if err := s.doJSON(ctx, http.MethodPost, "creditmemo", query, req, &out, opts...); err != nil {
		return nil, err
	}
	return &out.CreditMemo, nil
}

func (s *CreditMemoService) GetPDF(ctx context.Context, id string, opts ...RequestOption) ([]byte, error) {
	return s.doBytes(ctx, http.MethodGet, "creditmemo/"+id+"/pdf", nil, "application/pdf", opts...)
}

func (s *CreditMemoService) StreamPDF(ctx context.Context, id string, opts ...RequestOption) (io.ReadCloser, error) {
	return s.doStream(ctx, http.MethodGet, "creditmemo/"+id+"/pdf", nil, "application/pdf", opts...)
}

func (s *CreditMemoService) Send(ctx context.Context, id, email string, opts ...RequestOption) error {
	addr, err := mail.ParseAddress(email)
	if err != nil {
		return err
	}

	query := url.Values{"sendTo": []string{addr.Address}}

	return s.doJSON(ctx, http.MethodPost, "creditmemo/"+id+"/send", query, nil, nil, opts...)
}

func (s *CreditMemoService) ChangeDataCapture(ctx context.Context, changedSince time.Time, opts ...RequestOption) ([]CreditMemo, error) {
	resp, err := s.company.ChangeDataCapture(ctx, changedSince, []EntityName{EntityCreditMemo}, opts...)
	if err != nil {
		return nil, err
	}
	return s.ExtractCDC(resp)
}

func (s *CreditMemoService) ExtractCDC(resp *CDCResponse) ([]CreditMemo, error) {
	return ExtractCDCEntities[CreditMemo](resp, EntityCreditMemo)
}

func (s *CreditMemoService) ExtractBatch(resp *BatchResponse) ([]CreditMemo, error) {
	return ExtractBatchEntities[CreditMemo](resp, EntityCreditMemo)
}

func (s *CreditMemoService) ExtractBatchItems(items []BatchItemResponse) ([]CreditMemo, error) {
	return ExtractBatchItemEntities[CreditMemo](items, EntityCreditMemo)
}

func (s *CreditMemoService) BatchCreateItem(id string, cm CreditMemo) BatchItemRequest {
	return NewBatchEntity(id, BatchOperationCreate, EntityCreditMemo, cm)
}

func (s *CreditMemoService) BatchUpdateItem(id string, cm CreditMemo) BatchItemRequest {
	return NewBatchEntity(id, BatchOperationUpdate, EntityCreditMemo, cm)
}

func (s *CreditMemoService) BatchDeleteItem(id string, req DeleteRequest) BatchItemRequest {
	return NewBatchEntity(id, BatchOperationDelete, EntityCreditMemo, req)
}

func (s *CreditMemoService) BatchQueryItem(id, query string, opts ...QueryOption) BatchItemRequest {
	return NewBatchQuery(id, query, opts...)
}
