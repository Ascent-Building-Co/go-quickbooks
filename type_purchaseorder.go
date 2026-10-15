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

// PurchaseOrderService provides access to the PurchaseOrder-related endpoints
// of the QuickBooks Online API.
type PurchaseOrderService struct {
	companyService
}

func (c *CompanyClient) PurchaseOrders() *PurchaseOrderService {
	return &PurchaseOrderService{companyService: companyService{company: c}}
}

// PurchaseOrder represents a QuickBooks Online PurchaseOrder entity, a
// non-posting transaction that records a request to purchase goods or
// services from a vendor.
//
// Fields marked "Read-only" are set by the server and ignored if sent in a
// write request.
type PurchaseOrder struct {
	IDValue
	SyncTokenValue
	CDCFields

	MetaDataValue
	CustomField []CustomField `json:"CustomField,omitempty"`

	// Required
	APAccountRef *ReferenceType      `json:"APAccountRef,omitempty"`
	VendorRef    *ReferenceType      `json:"VendorRef,omitempty"`
	Line         []PurchaseOrderLine `json:"Line,omitempty"`

	// Conditionally required
	CurrencyRef          *ReferenceType           `json:"CurrencyRef,omitempty"`
	GlobalTaxCalculation GlobalTaxCalculationEnum `json:"GlobalTaxCalculation,omitempty"`

	// Optional
	TxnDate                 *Date            `json:"TxnDate,omitempty"`
	POEmail                 *EmailAddress    `json:"POEmail,omitempty"`
	ClassRef                *ReferenceType   `json:"ClassRef,omitempty"`
	SalesTermRef            *ReferenceType   `json:"SalesTermRef,omitempty"`
	LinkedTxn               []LinkedTxn      `json:"LinkedTxn,omitempty"`
	Memo                    string           `json:"Memo,omitempty"`
	POStatus                string           `json:"POStatus,omitempty"`
	DueDate                 *Date            `json:"DueDate,omitempty"`
	DocNumber               string           `json:"DocNumber,omitempty"`
	PrivateNote             string           `json:"PrivateNote,omitempty"`
	ShipMethodRef           *ReferenceType   `json:"ShipMethodRef,omitempty"`
	TxnTaxDetail            *TxnTaxDetail    `json:"TxnTaxDetail,omitempty"`
	ShipTo                  *ReferenceType   `json:"ShipTo,omitempty"`
	ExchangeRate            json.Number      `json:"ExchangeRate,omitempty"`
	ShipAddr                *PhysicalAddress `json:"ShipAddr,omitempty"`
	VendorAddr              *PhysicalAddress `json:"VendorAddr,omitempty"`
	EmailStatus             EmailStatusEnum  `json:"EmailStatus,omitempty"`
	TransactionLocationType string           `json:"TransactionLocationType,omitempty"` // France locale only.

	// Read-only: set by the server.
	TotalAmt     json.Number    `json:"TotalAmt,omitempty"`
	RecurDataRef *ReferenceType `json:"RecurDataRef,omitempty"`
}

func (PurchaseOrder) EntityName() EntityName {
	return EntityPurchaseOrder
}

// PurchaseOrderLine represents a single line item on a PurchaseOrder. The
// valid detail types are ItemBasedExpenseLine and AccountBasedExpenseLine.
//
// Note: the QBO API defines a PurchaseOrderItemLineDetail model with
// PO-specific fields (ManuallyClosed, OpenQty, ManPartNum) that are
// serialised under the "ItemBasedExpenseLineDetail" JSON key. These fields
// are not yet modelled here; they will be added in a future iteration.
type PurchaseOrderLine struct {
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

type purchaseOrderEnvelope struct {
	PurchaseOrder PurchaseOrder `json:"PurchaseOrder"`
}

type purchaseOrderQueryEnvelope struct {
	QueryResponse QueryResponse `json:"QueryResponse"`
}

func (s *PurchaseOrderService) Get(ctx context.Context, id string, opts ...RequestOption) (*PurchaseOrder, error) {
	var out purchaseOrderEnvelope
	if err := s.doJSON(ctx, http.MethodGet, "purchaseorder/"+id, nil, nil, &out, opts...); err != nil {
		return nil, err
	}
	return &out.PurchaseOrder, nil
}

func (s *PurchaseOrderService) Query(ctx context.Context, query string, opts ...QueryOption) ([]PurchaseOrder, error) {
	query = applyQueryOptions(query, opts...)

	var out purchaseOrderQueryEnvelope
	if err := s.doJSON(ctx, http.MethodGet, "query", url.Values{"query": []string{query}}, nil, &out); err != nil {
		return nil, err
	}

	raw, err := out.QueryResponse.Entities(EntityPurchaseOrder)
	if err != nil {
		return nil, err
	}

	return decodeEntitySlice[PurchaseOrder](raw)
}

func (s *PurchaseOrderService) List(ctx context.Context, opts ...QueryOption) ([]PurchaseOrder, error) {
	return s.Query(ctx, "SELECT * FROM PurchaseOrder", opts...)
}

func (s *PurchaseOrderService) ListPage(ctx context.Context, page, pageSize int, opts ...QueryOption) ([]PurchaseOrder, error) {
	pageOpts := append([]QueryOption{WithPage(page, pageSize)}, opts...)
	return s.List(ctx, pageOpts...)
}

func (s *PurchaseOrderService) Create(ctx context.Context, po PurchaseOrder, opts ...RequestOption) (*PurchaseOrder, error) {
	var out purchaseOrderEnvelope
	if err := s.doJSON(ctx, http.MethodPost, "purchaseorder", nil, po, &out, opts...); err != nil {
		return nil, err
	}
	return &out.PurchaseOrder, nil
}

func (s *PurchaseOrderService) Update(ctx context.Context, po PurchaseOrder, opts ...RequestOption) (*PurchaseOrder, error) {
	var out purchaseOrderEnvelope
	if err := s.doJSON(ctx, http.MethodPost, "purchaseorder", nil, po, &out, opts...); err != nil {
		return nil, err
	}
	return &out.PurchaseOrder, nil
}

func (s *PurchaseOrderService) Delete(ctx context.Context, req DeleteRequest, opts ...RequestOption) (*PurchaseOrder, error) {
	var out purchaseOrderEnvelope
	query := url.Values{"operation": []string{"delete"}}
	if err := s.doJSON(ctx, http.MethodPost, "purchaseorder", query, req, &out, opts...); err != nil {
		return nil, err
	}
	return &out.PurchaseOrder, nil
}

func (s *PurchaseOrderService) Send(ctx context.Context, id, email string, opts ...RequestOption) error {
	addr, err := mail.ParseAddress(email)
	if err != nil {
		return err
	}

	query := url.Values{"sendTo": []string{addr.Address}}

	return s.doJSON(ctx, http.MethodPost, "purchaseorder/"+id+"/send", query, nil, nil, opts...)
}

func (s *PurchaseOrderService) GetPDF(ctx context.Context, id string, opts ...RequestOption) ([]byte, error) {
	return s.doBytes(ctx, http.MethodGet, "purchaseorder/"+id+"/pdf", nil, "application/pdf", opts...)
}

func (s *PurchaseOrderService) StreamPDF(ctx context.Context, id string, opts ...RequestOption) (io.ReadCloser, error) {
	return s.doStream(ctx, http.MethodGet, "purchaseorder/"+id+"/pdf", nil, "application/pdf", opts...)
}

func (s *PurchaseOrderService) ChangeDataCapture(ctx context.Context, changedSince time.Time, opts ...RequestOption) ([]PurchaseOrder, error) {
	resp, err := s.company.ChangeDataCapture(ctx, changedSince, []EntityName{EntityPurchaseOrder}, opts...)
	if err != nil {
		return nil, err
	}
	return s.ExtractCDC(resp)
}

func (s *PurchaseOrderService) ExtractCDC(resp *CDCResponse) ([]PurchaseOrder, error) {
	return ExtractCDCEntities[PurchaseOrder](resp, EntityPurchaseOrder)
}

func (s *PurchaseOrderService) ExtractBatch(resp *BatchResponse) ([]PurchaseOrder, error) {
	return ExtractBatchEntities[PurchaseOrder](resp, EntityPurchaseOrder)
}

func (s *PurchaseOrderService) ExtractBatchItems(items []BatchItemResponse) ([]PurchaseOrder, error) {
	return ExtractBatchItemEntities[PurchaseOrder](items, EntityPurchaseOrder)
}

func (s *PurchaseOrderService) BatchCreateItem(id string, po PurchaseOrder) BatchItemRequest {
	return NewBatchEntity(id, BatchOperationCreate, EntityPurchaseOrder, po)
}

func (s *PurchaseOrderService) BatchUpdateItem(id string, po PurchaseOrder) BatchItemRequest {
	return NewBatchEntity(id, BatchOperationUpdate, EntityPurchaseOrder, po)
}

func (s *PurchaseOrderService) BatchDeleteItem(id string, req DeleteRequest) BatchItemRequest {
	return NewBatchEntity(id, BatchOperationDelete, EntityPurchaseOrder, req)
}

func (s *PurchaseOrderService) BatchQueryItem(id, query string, opts ...QueryOption) BatchItemRequest {
	return NewBatchQuery(id, query, opts...)
}
