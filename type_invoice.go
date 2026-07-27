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

type DeliveryTypeEnum string

const (
	EmailDeliveryTypeEnum      DeliveryTypeEnum = "Email"
	TradeshiftDeliveryTypeEnum DeliveryTypeEnum = "Tradeshift"
)

type DeliveryErrorTypeEnum string

const (
	MissingInfoDeliveryErrorTypeEnum   DeliveryErrorTypeEnum = "Missing Info"
	UndeliverableDeliveryErrorTypeEnum DeliveryErrorTypeEnum = "Undeliverable"
)

type TransactionDeliveryInfo struct {
	DeliveryType      DeliveryTypeEnum      `json:"DeliveryType,omitempty"`
	DeliveryTime      *DateTime             `json:"DeliveryTime,omitempty"`
	DeliveryErrorType DeliveryErrorTypeEnum `json:"DeliveryErrorType,omitempty"`
}

type InvoiceService struct {
	companyService
}

func (c *CompanyClient) Invoices() *InvoiceService {
	return &InvoiceService{companyService: companyService{company: c}}
}

// Invoice represents a QuickBooks Online Invoice entity.
//
// Fields marked "Read-only" are set by the server and ignored if sent in a
// write request.
type Invoice struct {
	IDValue
	SyncTokenValue
	CDCFields

	MetaData    *ModificationMetaData `json:"MetaData,omitempty"`
	CustomField []CustomField         `json:"CustomField,omitempty"`

	// Required
	CustomerRef *ReferenceType `json:"CustomerRef,omitempty"`
	Line        []InvoiceLine  `json:"Line,omitempty"`

	// Conditionally required
	ProjectRef           *ReferenceType           `json:"ProjectRef,omitempty"`
	CurrencyRef          *ReferenceType           `json:"CurrencyRef,omitempty"`
	ShipFromAddr         *PhysicalAddress         `json:"ShipFromAddr,omitempty"`
	GlobalTaxCalculation GlobalTaxCalculationEnum `json:"GlobalTaxCalculation,omitempty"`
	DocNumber            string                   `json:"DocNumber,omitempty"`
	BillEmail            *EmailAddress            `json:"BillEmail,omitempty"`

	// Optional
	TxnDate                      *Date            `json:"TxnDate,omitempty"`
	ShipDate                     *Date            `json:"ShipDate,omitempty"`
	TrackingNum                  string           `json:"TrackingNum,omitempty"`
	ClassRef                     *ReferenceType   `json:"ClassRef,omitempty"`
	PrintStatus                  PrintStatusEnum  `json:"PrintStatus,omitempty"`
	SalesTermRef                 *ReferenceType   `json:"SalesTermRef,omitempty"`
	TxnSource                    string           `json:"TxnSource,omitempty"`
	LinkedTxn                    []LinkedTxn      `json:"LinkedTxn,omitempty"`
	DepositToAccountRef          *ReferenceType   `json:"DepositToAccountRef,omitempty"`
	AllowOnlineACHPayment        *bool            `json:"AllowOnlineACHPayment,omitempty"`
	TransactionLocationType      string           `json:"TransactionLocationType,omitempty"`
	DueDate                      *Date            `json:"DueDate,omitempty"`
	PrivateNote                  string           `json:"PrivateNote,omitempty"`
	BillEmailCC                  *EmailAddress    `json:"BillEmailCc,omitempty"`
	CustomerMemo                 *MemoRef         `json:"CustomerMemo,omitempty"`
	EmailStatus                  EmailStatusEnum  `json:"EmailStatus,omitempty"`
	ExchangeRate                 json.Number      `json:"ExchangeRate,omitempty"`
	Deposit                      json.Number      `json:"Deposit,omitempty"`
	TxnTaxDetail                 *TxnTaxDetail    `json:"TxnTaxDetail,omitempty"`
	AllowOnlineCreditCardPayment *bool            `json:"AllowOnlineCreditCardPayment,omitempty"`
	ShipAddr                     *PhysicalAddress `json:"ShipAddr,omitempty"`
	DepartmentRef                *ReferenceType   `json:"DepartmentRef,omitempty"`
	BillEmailBCC                 *EmailAddress    `json:"BillEmailBcc,omitempty"`
	ShipMethodRef                *ReferenceType   `json:"ShipMethodRef,omitempty"`
	BillAddr                     *PhysicalAddress `json:"BillAddr,omitempty"`
	ApplyTaxAfterDiscount        *bool            `json:"ApplyTaxAfterDiscount,omitempty"`

	// Read-only: set by the server.
	HomeBalance        json.Number              `json:"HomeBalance,omitempty"`
	DeliveryInfo       *TransactionDeliveryInfo `json:"DeliveryInfo,omitempty"`
	TotalAmt           json.Number              `json:"TotalAmt,omitempty"`
	InvoiceLink        string                   `json:"InvoiceLink,omitempty"`
	RecurDataRef       *ReferenceType           `json:"RecurDataRef,omitempty"`
	TaxExemptionRef    *ReferenceType           `json:"TaxExemptionRef,omitempty"`
	Balance            json.Number              `json:"Balance,omitempty"`
	HomeTotalAmt       json.Number              `json:"HomeTotalAmt,omitempty"`
	FreeFormAddress    *bool                    `json:"FreeFormAddress,omitempty"`
	AllowOnlinePayment *bool                    `json:"AllowOnlinePayment,omitempty"`
	AllowIPNPayment    *bool                    `json:"AllowIPNPayment,omitempty"`
}

func (Invoice) EntityName() EntityName {
	return EntityInvoice
}

// InvoiceLine represents a single line item on an Invoice. The valid detail
// types for Invoice are SalesItemLine, DescriptionOnlyLine, DiscountLine,
// GroupLine, and SubTotalLine.
type InvoiceLine struct {
	IDValue

	LineNum               json.Number             `json:"LineNum,omitempty"`
	Description           string                  `json:"Description,omitempty"`
	Amount                json.Number             `json:"Amount,omitempty"`
	DetailType            LineDetailType          `json:"DetailType,omitempty"`
	LinkedTxn             []LinkedTxn             `json:"LinkedTxn,omitempty"`
	SalesItemLineDetail   *SalesItemLineDetail    `json:"SalesItemLineDetail,omitempty"`
	DescriptionLineDetail *DescriptionLineDetail  `json:"DescriptionLineDetail,omitempty"`
	DiscountLineDetail    *DiscountLineDetail     `json:"DiscountLineDetail,omitempty"`
	GroupLineDetail       *InvoiceGroupLineDetail `json:"GroupLineDetail,omitempty"`
	SubTotalLineDetail    *SubTotalLineDetail     `json:"SubTotalLineDetail,omitempty"`
}

// InvoiceGroupLineDetail is the group line detail type for Invoice. It uses
// []InvoiceLine for its nested lines so that only Invoice-valid detail types
// are accessible within a group.
type InvoiceGroupLineDetail struct {
	Line         []InvoiceLine  `json:"Line,omitempty"`
	Qty          json.Number    `json:"Quantity,omitempty"`
	ServiceDate  *Date          `json:"ServiceDate,omitempty"`
	GroupItemRef *ReferenceType `json:"GroupItemRef,omitempty"`
}

type invoiceEnvelope struct {
	Invoice Invoice `json:"Invoice"`
}

type invoiceQueryEnvelope struct {
	QueryResponse QueryResponse `json:"QueryResponse"`
}

func (s *InvoiceService) Get(ctx context.Context, id string, opts ...RequestOption) (*Invoice, error) {
	var out invoiceEnvelope
	if err := s.doJSON(ctx, http.MethodGet, "invoice/"+id, nil, nil, &out, opts...); err != nil {
		return nil, err
	}
	return &out.Invoice, nil
}

func (s *InvoiceService) Query(ctx context.Context, query string, opts ...QueryOption) ([]Invoice, error) {
	query = applyQueryOptions(query, opts...)

	var out invoiceQueryEnvelope
	if err := s.doJSON(ctx, http.MethodGet, "query", url.Values{"query": []string{query}}, nil, &out); err != nil {
		return nil, err
	}

	raw, err := out.QueryResponse.Entities(EntityInvoice)
	if err != nil {
		return nil, err
	}

	return decodeEntitySlice[Invoice](raw)
}

func (s *InvoiceService) List(ctx context.Context, opts ...QueryOption) ([]Invoice, error) {
	return s.Query(ctx, "SELECT * FROM Invoice", opts...)
}

func (s *InvoiceService) ListPage(ctx context.Context, page, pageSize int, opts ...QueryOption) ([]Invoice, error) {
	pageOpts := append([]QueryOption{WithPage(page, pageSize)}, opts...)
	return s.List(ctx, pageOpts...)
}

func (s *InvoiceService) Create(ctx context.Context, invoice Invoice, opts ...RequestOption) (*Invoice, error) {
	var out invoiceEnvelope
	if err := s.doJSON(ctx, http.MethodPost, "invoice", nil, invoice, &out, opts...); err != nil {
		return nil, err
	}
	return &out.Invoice, nil
}

func (s *InvoiceService) Update(ctx context.Context, invoice Invoice, opts ...RequestOption) (*Invoice, error) {
	var out invoiceEnvelope
	if err := s.doJSON(ctx, http.MethodPost, "invoice", nil, invoice, &out, opts...); err != nil {
		return nil, err
	}
	return &out.Invoice, nil
}

func (s *InvoiceService) UpdateSparse(ctx context.Context, invoice Invoice, opts ...RequestOption) (*Invoice, error) {
	payload := struct {
		Invoice
		Sparse bool `json:"sparse"`
	}{invoice, true}
	var out invoiceEnvelope
	if err := s.doJSON(ctx, http.MethodPost, "invoice", nil, payload, &out, opts...); err != nil {
		return nil, err
	}
	return &out.Invoice, nil
}

func (s *InvoiceService) Delete(ctx context.Context, req DeleteRequest, opts ...RequestOption) (*Invoice, error) {
	var out invoiceEnvelope
	query := url.Values{"operation": []string{"delete"}}
	if err := s.doJSON(ctx, http.MethodPost, "invoice", query, req, &out, opts...); err != nil {
		return nil, err
	}
	return &out.Invoice, nil
}

func (s *InvoiceService) Void(ctx context.Context, req VoidRequest, opts ...RequestOption) (*Invoice, error) {
	var out invoiceEnvelope
	query := url.Values{"operation": []string{"void"}}
	if err := s.doJSON(ctx, http.MethodPost, "invoice", query, req, &out, opts...); err != nil {
		return nil, err
	}
	return &out.Invoice, nil
}

func (s *InvoiceService) Send(ctx context.Context, id, email string, opts ...RequestOption) error {
	addr, err := mail.ParseAddress(email)
	if err != nil {
		return err
	}

	query := url.Values{"sendTo": []string{addr.Address}}

	return s.doJSON(ctx, http.MethodPost, "invoice/"+id+"/send", query, nil, nil, opts...)
}

func (s *InvoiceService) GetPDF(ctx context.Context, id string, opts ...RequestOption) ([]byte, error) {
	return s.doBytes(ctx, http.MethodGet, "invoice/"+id+"/pdf", nil, "application/pdf", opts...)
}

func (s *InvoiceService) StreamPDF(ctx context.Context, id string, opts ...RequestOption) (io.ReadCloser, error) {
	return s.doStream(ctx, http.MethodGet, "invoice/"+id+"/pdf", nil, "application/pdf", opts...)
}

func (s *InvoiceService) ChangeDataCapture(ctx context.Context, changedSince time.Time, opts ...RequestOption) ([]Invoice, error) {
	resp, err := s.company.ChangeDataCapture(ctx, changedSince, []EntityName{EntityInvoice}, opts...)
	if err != nil {
		return nil, err
	}
	return s.ExtractCDC(resp)
}

func (s *InvoiceService) ExtractCDC(resp *CDCResponse) ([]Invoice, error) {
	return ExtractCDCEntities[Invoice](resp, EntityInvoice)
}

func (s *InvoiceService) ExtractBatch(resp *BatchResponse) ([]Invoice, error) {
	return ExtractBatchEntities[Invoice](resp, EntityInvoice)
}

func (s *InvoiceService) ExtractBatchItems(items []BatchItemResponse) ([]Invoice, error) {
	return ExtractBatchItemEntities[Invoice](items, EntityInvoice)
}

func (s *InvoiceService) BatchCreateItem(id string, invoice Invoice) BatchItemRequest {
	return NewBatchEntity(id, BatchOperationCreate, EntityInvoice, invoice)
}

func (s *InvoiceService) BatchUpdateItem(id string, invoice Invoice) BatchItemRequest {
	return NewBatchEntity(id, BatchOperationUpdate, EntityInvoice, invoice)
}

func (s *InvoiceService) BatchDeleteItem(id string, req DeleteRequest) BatchItemRequest {
	return NewBatchEntity(id, BatchOperationDelete, EntityInvoice, req)
}

func (s *InvoiceService) BatchVoidItem(id string, req VoidRequest) BatchItemRequest {
	return NewBatchVoidEntity(id, EntityInvoice, req)
}

func (s *InvoiceService) BatchQueryItem(id, query string, opts ...QueryOption) BatchItemRequest {
	return NewBatchQuery(id, query, opts...)
}
