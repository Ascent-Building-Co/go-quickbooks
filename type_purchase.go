package quickbooks

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"time"
)

type PurchasePaymentTypeEnum string

const (
	CashPurchasePaymentType       PurchasePaymentTypeEnum = "Cash"
	CheckPurchasePaymentType      PurchasePaymentTypeEnum = "Check"
	CreditCardPurchasePaymentType PurchasePaymentTypeEnum = "CreditCard"
)

type PurchaseService struct {
	companyService
}

func (c *CompanyClient) Purchases() *PurchaseService {
	return &PurchaseService{companyService: companyService{company: c}}
}

// Purchase represents a QuickBooks Online Purchase entity, recording an
// expense paid by cash, check, or credit card.
//
// Fields marked "Read-only" are set by the server and ignored if sent in a
// write request.
//
// The PurchaseEx field returned by the API contains legacy XML metadata and
// is intentionally omitted from this struct; it is silently ignored on decode.
type Purchase struct {
	IDValue
	SyncTokenValue
	CDCFields

	MetaDataValue

	// Required
	AccountRef  *ReferenceType          `json:"AccountRef,omitempty"`
	PaymentType PurchasePaymentTypeEnum `json:"PaymentType,omitempty"`
	Line        []PurchaseLine          `json:"Line,omitempty"`

	// Conditionally required
	CurrencyRef *ReferenceType `json:"CurrencyRef,omitempty"`

	// Optional
	TxnDate                 *Date                    `json:"TxnDate,omitempty"`
	PrintStatus             PrintStatusEnum          `json:"PrintStatus,omitempty"`
	RemitToAddr             *PhysicalAddress         `json:"RemitToAddr,omitempty"`
	TxnSource               string                   `json:"TxnSource,omitempty"`
	LinkedTxn               []LinkedTxn              `json:"LinkedTxn,omitempty"`
	GlobalTaxCalculation    GlobalTaxCalculationEnum `json:"GlobalTaxCalculation,omitempty"`
	TransactionLocationType string                   `json:"TransactionLocationType,omitempty"`
	DocNumber               string                   `json:"DocNumber,omitempty"`
	PrivateNote             string                   `json:"PrivateNote,omitempty"`
	Credit                  *bool                    `json:"Credit,omitempty"`
	TxnTaxDetail            *TxnTaxDetail            `json:"TxnTaxDetail,omitempty"`
	PaymentMethodRef        *ReferenceType           `json:"PaymentMethodRef,omitempty"`
	ExchangeRate            json.Number              `json:"ExchangeRate,omitempty"`
	DepartmentRef           *ReferenceType           `json:"DepartmentRef,omitempty"`
	EntityRef               *ReferenceType           `json:"EntityRef,omitempty"`
	IncludeInAnnualTPAR     *bool                    `json:"IncludeInAnnualTPAR,omitempty"`

	// Read-only: set by the server.
	TotalAmt     json.Number    `json:"TotalAmt,omitempty"`
	RecurDataRef *ReferenceType `json:"RecurDataRef,omitempty"`
}

func (Purchase) EntityName() EntityName {
	return EntityPurchase
}

// PurchaseLine represents a single line item on a Purchase. The valid detail
// types are ItemBasedExpenseLine and AccountBasedExpenseLine, reusing the
// shared detail structs from definitions.go.
//
// ProjectRef is not part of the v75 schema model but appears in API payloads
// when the line is associated with a QuickBooks project.
type PurchaseLine struct {
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

type purchaseEnvelope struct {
	Purchase Purchase `json:"Purchase"`
}

type purchaseQueryEnvelope struct {
	QueryResponse QueryResponse `json:"QueryResponse"`
}

func (s *PurchaseService) Get(ctx context.Context, id string, opts ...RequestOption) (*Purchase, error) {
	var out purchaseEnvelope
	if err := s.doJSON(ctx, http.MethodGet, "purchase/"+id, nil, nil, &out, opts...); err != nil {
		return nil, err
	}
	return &out.Purchase, nil
}

func (s *PurchaseService) Query(ctx context.Context, query string, opts ...QueryOption) ([]Purchase, error) {
	query = applyQueryOptions(query, opts...)

	var out purchaseQueryEnvelope
	if err := s.doJSON(ctx, http.MethodGet, "query", url.Values{"query": []string{query}}, nil, &out); err != nil {
		return nil, err
	}

	raw, err := out.QueryResponse.Entities(EntityPurchase)
	if err != nil {
		return nil, err
	}

	return decodeEntitySlice[Purchase](raw)
}

func (s *PurchaseService) List(ctx context.Context, opts ...QueryOption) ([]Purchase, error) {
	return s.Query(ctx, "SELECT * FROM Purchase", opts...)
}

func (s *PurchaseService) ListPage(ctx context.Context, page, pageSize int, opts ...QueryOption) ([]Purchase, error) {
	pageOpts := append([]QueryOption{WithPage(page, pageSize)}, opts...)
	return s.List(ctx, pageOpts...)
}

func (s *PurchaseService) Create(ctx context.Context, purchase Purchase, opts ...RequestOption) (*Purchase, error) {
	var out purchaseEnvelope
	if err := s.doJSON(ctx, http.MethodPost, "purchase", nil, purchase, &out, opts...); err != nil {
		return nil, err
	}
	return &out.Purchase, nil
}

func (s *PurchaseService) Update(ctx context.Context, purchase Purchase, opts ...RequestOption) (*Purchase, error) {
	var out purchaseEnvelope
	if err := s.doJSON(ctx, http.MethodPost, "purchase", nil, purchase, &out, opts...); err != nil {
		return nil, err
	}
	return &out.Purchase, nil
}

func (s *PurchaseService) UpdateSparse(ctx context.Context, purchase Purchase, opts ...RequestOption) (*Purchase, error) {
	payload := struct {
		Purchase
		Sparse bool `json:"sparse"`
	}{purchase, true}
	var out purchaseEnvelope
	if err := s.doJSON(ctx, http.MethodPost, "purchase", nil, payload, &out, opts...); err != nil {
		return nil, err
	}
	return &out.Purchase, nil
}

func (s *PurchaseService) Delete(ctx context.Context, req DeleteRequest, opts ...RequestOption) (*Purchase, error) {
	var out purchaseEnvelope
	query := url.Values{"operation": []string{"delete"}}
	if err := s.doJSON(ctx, http.MethodPost, "purchase", query, req, &out, opts...); err != nil {
		return nil, err
	}
	return &out.Purchase, nil
}

func (s *PurchaseService) ChangeDataCapture(ctx context.Context, changedSince time.Time, opts ...RequestOption) ([]Purchase, error) {
	resp, err := s.company.ChangeDataCapture(ctx, changedSince, []EntityName{EntityPurchase}, opts...)
	if err != nil {
		return nil, err
	}
	return s.ExtractCDC(resp)
}

func (s *PurchaseService) ExtractCDC(resp *CDCResponse) ([]Purchase, error) {
	return ExtractCDCEntities[Purchase](resp, EntityPurchase)
}

func (s *PurchaseService) ExtractBatch(resp *BatchResponse) ([]Purchase, error) {
	return ExtractBatchEntities[Purchase](resp, EntityPurchase)
}

func (s *PurchaseService) ExtractBatchItems(items []BatchItemResponse) ([]Purchase, error) {
	return ExtractBatchItemEntities[Purchase](items, EntityPurchase)
}

func (s *PurchaseService) BatchCreateItem(id string, purchase Purchase) BatchItemRequest {
	return NewBatchEntity(id, BatchOperationCreate, EntityPurchase, purchase)
}

func (s *PurchaseService) BatchUpdateItem(id string, purchase Purchase) BatchItemRequest {
	return NewBatchEntity(id, BatchOperationUpdate, EntityPurchase, purchase)
}

func (s *PurchaseService) BatchDeleteItem(id string, req DeleteRequest) BatchItemRequest {
	return NewBatchEntity(id, BatchOperationDelete, EntityPurchase, req)
}

func (s *PurchaseService) BatchQueryItem(id, query string, opts ...QueryOption) BatchItemRequest {
	return NewBatchQuery(id, query, opts...)
}
