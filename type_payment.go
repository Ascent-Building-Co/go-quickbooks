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

// PaymentCreditCard holds the credit card processing details on a Payment.
// Unlike BillPaymentCreditCard, it does not include a CCAccountRef — the
// account is implicit from the payment context. Both fields are read-only,
// set by the payment processor.
type PaymentCreditCard struct {
	CreditChargeInfo     *CreditChargeInfo     `json:"CreditChargeInfo,omitempty"`
	CreditChargeResponse *CreditChargeResponse `json:"CreditChargeResponse,omitempty"`
}

type PaymentService struct {
	companyService
}

func (c *CompanyClient) Payments() *PaymentService {
	return &PaymentService{companyService: companyService{company: c}}
}

// Payment represents a QuickBooks Online Payment entity, recording a customer
// payment that can be applied against one or more outstanding invoices.
//
// Fields marked "Read-only" are set by the server and ignored if sent in a
// write request.
type Payment struct {
	IDValue
	SyncTokenValue
	CDCFields

	MetaDataValue

	// Required
	TotalAmt    json.Number    `json:"TotalAmt,omitempty"`
	CustomerRef *ReferenceType `json:"CustomerRef,omitempty"`

	// Conditionally required
	CurrencyRef *ReferenceType `json:"CurrencyRef,omitempty"`
	ProjectRef  *ReferenceType `json:"ProjectRef,omitempty"`

	// Optional
	PrivateNote             string             `json:"PrivateNote,omitempty"`
	PaymentMethodRef        *ReferenceType     `json:"PaymentMethodRef,omitempty"`
	UnappliedAmt            json.Number        `json:"UnappliedAmt,omitempty"`
	DepositToAccountRef     *ReferenceType     `json:"DepositToAccountRef,omitempty"`
	ExchangeRate            json.Number        `json:"ExchangeRate,omitempty"`
	Line                    []PaymentLine      `json:"Line,omitempty"`
	TxnSource               string             `json:"TxnSource,omitempty"`
	PaymentRefNum           string             `json:"PaymentRefNum,omitempty"`
	TxnDate                 *Date              `json:"TxnDate,omitempty"`
	CreditCardPayment       *PaymentCreditCard `json:"CreditCardPayment,omitempty"`
	TransactionLocationType string             `json:"TransactionLocationType,omitempty"` // France locale only.
	ProcessPayment          *bool              `json:"ProcessPayment,omitempty"`

	// Read-only: set by the server.
	TaxExemptionRef *ReferenceType `json:"TaxExemptionRef,omitempty"`
}

func (Payment) EntityName() EntityName {
	return EntityPayment
}

// PaymentLine represents a single line on a Payment, linking the payment
// amount to a specific transaction. Payment lines are always of the
// LineLinkedTxn variant — Amount and LinkedTxn are the only meaningful fields.
type PaymentLine struct {
	Amount    json.Number `json:"Amount,omitempty"`
	LinkedTxn []LinkedTxn `json:"LinkedTxn,omitempty"`
}

type paymentEnvelope struct {
	Payment Payment `json:"Payment"`
}

type paymentQueryEnvelope struct {
	QueryResponse QueryResponse `json:"QueryResponse"`
}

func (s *PaymentService) Get(ctx context.Context, id string, opts ...RequestOption) (*Payment, error) {
	var out paymentEnvelope
	if err := s.doJSON(ctx, http.MethodGet, "payment/"+id, nil, nil, &out, opts...); err != nil {
		return nil, err
	}
	return &out.Payment, nil
}

func (s *PaymentService) Query(ctx context.Context, query string, opts ...QueryOption) ([]Payment, error) {
	query = applyQueryOptions(query, opts...)

	var out paymentQueryEnvelope
	if err := s.doJSON(ctx, http.MethodGet, "query", url.Values{"query": []string{query}}, nil, &out); err != nil {
		return nil, err
	}

	raw, err := out.QueryResponse.Entities(EntityPayment)
	if err != nil {
		return nil, err
	}

	return decodeEntitySlice[Payment](raw)
}

func (s *PaymentService) List(ctx context.Context, opts ...QueryOption) ([]Payment, error) {
	return s.Query(ctx, "SELECT * FROM Payment", opts...)
}

func (s *PaymentService) ListPage(ctx context.Context, page, pageSize int, opts ...QueryOption) ([]Payment, error) {
	pageOpts := append([]QueryOption{WithPage(page, pageSize)}, opts...)
	return s.List(ctx, pageOpts...)
}

func (s *PaymentService) Create(ctx context.Context, payment Payment, opts ...RequestOption) (*Payment, error) {
	var out paymentEnvelope
	if err := s.doJSON(ctx, http.MethodPost, "payment", nil, payment, &out, opts...); err != nil {
		return nil, err
	}
	return &out.Payment, nil
}

func (s *PaymentService) Update(ctx context.Context, payment Payment, opts ...RequestOption) (*Payment, error) {
	var out paymentEnvelope
	if err := s.doJSON(ctx, http.MethodPost, "payment", nil, payment, &out, opts...); err != nil {
		return nil, err
	}
	return &out.Payment, nil
}

func (s *PaymentService) UpdateSparse(ctx context.Context, payment Payment, opts ...RequestOption) (*Payment, error) {
	payload := struct {
		Payment
		Sparse bool `json:"sparse"`
	}{payment, true}
	var out paymentEnvelope
	if err := s.doJSON(ctx, http.MethodPost, "payment", nil, payload, &out, opts...); err != nil {
		return nil, err
	}
	return &out.Payment, nil
}

func (s *PaymentService) Delete(ctx context.Context, req DeleteRequest, opts ...RequestOption) (*Payment, error) {
	var out paymentEnvelope
	query := url.Values{"operation": []string{"delete"}}
	if err := s.doJSON(ctx, http.MethodPost, "payment", query, req, &out, opts...); err != nil {
		return nil, err
	}
	return &out.Payment, nil
}

// Void voids the Payment identified by req. QuickBooks requires a sparse
// update with include=void: the payload must contain Id, SyncToken, and
// "sparse":true.
func (s *PaymentService) Void(ctx context.Context, req VoidRequest, opts ...RequestOption) (*Payment, error) {
	payload := struct {
		VoidRequest
		Sparse bool `json:"sparse"`
	}{req, true}
	var out paymentEnvelope
	query := url.Values{"operation": []string{"update"}, "include": []string{"void"}}
	if err := s.doJSON(ctx, http.MethodPost, "payment", query, payload, &out, opts...); err != nil {
		return nil, err
	}
	return &out.Payment, nil
}

func (s *PaymentService) Send(ctx context.Context, id, email string, opts ...RequestOption) error {
	addr, err := mail.ParseAddress(email)
	if err != nil {
		return err
	}

	query := url.Values{"sendTo": []string{addr.Address}}

	return s.doJSON(ctx, http.MethodPost, "payment/"+id+"/send", query, nil, nil, opts...)
}

func (s *PaymentService) GetPDF(ctx context.Context, id string, opts ...RequestOption) ([]byte, error) {
	return s.doBytes(ctx, http.MethodGet, "payment/"+id+"/pdf", nil, "application/pdf", opts...)
}

func (s *PaymentService) StreamPDF(ctx context.Context, id string, opts ...RequestOption) (io.ReadCloser, error) {
	return s.doStream(ctx, http.MethodGet, "payment/"+id+"/pdf", nil, "application/pdf", opts...)
}

func (s *PaymentService) ChangeDataCapture(ctx context.Context, changedSince time.Time, opts ...RequestOption) ([]Payment, error) {
	resp, err := s.company.ChangeDataCapture(ctx, changedSince, []EntityName{EntityPayment}, opts...)
	if err != nil {
		return nil, err
	}
	return s.ExtractCDC(resp)
}

func (s *PaymentService) ExtractCDC(resp *CDCResponse) ([]Payment, error) {
	return ExtractCDCEntities[Payment](resp, EntityPayment)
}

func (s *PaymentService) ExtractBatch(resp *BatchResponse) ([]Payment, error) {
	return ExtractBatchEntities[Payment](resp, EntityPayment)
}

func (s *PaymentService) ExtractBatchItems(items []BatchItemResponse) ([]Payment, error) {
	return ExtractBatchItemEntities[Payment](items, EntityPayment)
}

func (s *PaymentService) BatchCreateItem(id string, payment Payment) BatchItemRequest {
	return NewBatchEntity(id, BatchOperationCreate, EntityPayment, payment)
}

func (s *PaymentService) BatchUpdateItem(id string, payment Payment) BatchItemRequest {
	return NewBatchEntity(id, BatchOperationUpdate, EntityPayment, payment)
}

func (s *PaymentService) BatchDeleteItem(id string, req DeleteRequest) BatchItemRequest {
	return NewBatchEntity(id, BatchOperationDelete, EntityPayment, req)
}

func (s *PaymentService) BatchVoidItem(id string, req VoidRequest) BatchItemRequest {
	payload := struct {
		VoidRequest
		Sparse bool `json:"sparse"`
	}{req, true}
	return NewBatchVoidEntity(id, EntityPayment, payload)
}

func (s *PaymentService) BatchQueryItem(id, query string, opts ...QueryOption) BatchItemRequest {
	return NewBatchQuery(id, query, opts...)
}
