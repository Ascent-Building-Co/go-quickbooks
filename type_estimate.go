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

type EstimateTxnStatusEnum string

const (
	EstimatePending   EstimateTxnStatusEnum = "Pending"
	EstimateAccepted  EstimateTxnStatusEnum = "Accepted"
	EstimateRejected  EstimateTxnStatusEnum = "Rejected"
	EstimateClosed    EstimateTxnStatusEnum = "Closed"
	EstimateConverted EstimateTxnStatusEnum = "Converted"
)

type EstimateService struct {
	companyService
}

func (c *CompanyClient) Estimates() *EstimateService {
	return &EstimateService{companyService: companyService{company: c}}
}

// Estimate represents a QuickBooks Online Estimate entity, a proposal for
// goods or services that can be converted to an Invoice.
//
// Fields marked "Read-only" are set by the server and ignored if sent in a
// write request.
type Estimate struct {
	IDValue
	SyncTokenValue
	CDCFields

	MetaDataValue
	CustomField []CustomField `json:"CustomField,omitempty"`

	// Required
	CustomerRef *ReferenceType `json:"CustomerRef,omitempty"`

	// Conditionally required
	ShipFromAddr         *PhysicalAddress         `json:"ShipFromAddr,omitempty"`
	CurrencyRef          *ReferenceType           `json:"CurrencyRef,omitempty"`
	GlobalTaxCalculation GlobalTaxCalculationEnum `json:"GlobalTaxCalculation,omitempty"`
	ProjectRef           *ReferenceType           `json:"ProjectRef,omitempty"`
	BillEmail            *EmailAddress            `json:"BillEmail,omitempty"`

	// Optional
	TxnDate                 *Date                 `json:"TxnDate,omitempty"`
	ShipDate                *Date                 `json:"ShipDate,omitempty"`
	ClassRef                *ReferenceType        `json:"ClassRef,omitempty"`
	PrintStatus             PrintStatusEnum       `json:"PrintStatus,omitempty"`
	SalesTermRef            *ReferenceType        `json:"SalesTermRef,omitempty"`
	TxnStatus               EstimateTxnStatusEnum `json:"TxnStatus,omitempty"`
	LinkedTxn               []LinkedTxn           `json:"LinkedTxn,omitempty"`
	AcceptedDate            *Date                 `json:"AcceptedDate,omitempty"`
	ExpirationDate          *Date                 `json:"ExpirationDate,omitempty"`
	TransactionLocationType string                `json:"TransactionLocationType,omitempty"` // France locale only.
	DueDate                 *Date                 `json:"DueDate,omitempty"`
	DocNumber               string                `json:"DocNumber,omitempty"`
	PrivateNote             string                `json:"PrivateNote,omitempty"`
	Line                    []EstimateLine        `json:"Line,omitempty"`
	CustomerMemo            *MemoRef              `json:"CustomerMemo,omitempty"`
	EmailStatus             EmailStatusEnum       `json:"EmailStatus,omitempty"`
	TxnTaxDetail            *TxnTaxDetail         `json:"TxnTaxDetail,omitempty"`
	AcceptedBy              string                `json:"AcceptedBy,omitempty"`
	ExchangeRate            json.Number           `json:"ExchangeRate,omitempty"`
	ShipAddr                *PhysicalAddress      `json:"ShipAddr,omitempty"`
	DepartmentRef           *ReferenceType        `json:"DepartmentRef,omitempty"`
	ShipMethodRef           *ReferenceType        `json:"ShipMethodRef,omitempty"`
	BillAddr                *PhysicalAddress      `json:"BillAddr,omitempty"`
	ApplyTaxAfterDiscount   *bool                 `json:"ApplyTaxAfterDiscount,omitempty"`

	// Read-only: set by the server.
	TotalAmt        json.Number    `json:"TotalAmt,omitempty"`
	HomeTotalAmt    json.Number    `json:"HomeTotalAmt,omitempty"`
	RecurDataRef    *ReferenceType `json:"RecurDataRef,omitempty"`
	TaxExemptionRef *ReferenceType `json:"TaxExemptionRef,omitempty"`
	FreeFormAddress *bool          `json:"FreeFormAddress,omitempty"`
}

func (Estimate) EntityName() EntityName {
	return EntityEstimate
}

// EstimateLine represents a single line item on an Estimate. The valid detail
// types are SalesItemLine, DescriptionOnlyLine, DiscountLine, GroupLine, and
// SubTotalLine — identical to Invoice.
type EstimateLine struct {
	IDValue

	LineNum               json.Number              `json:"LineNum,omitempty"`
	Description           string                   `json:"Description,omitempty"`
	Amount                json.Number              `json:"Amount,omitempty"`
	DetailType            LineDetailType           `json:"DetailType,omitempty"`
	LinkedTxn             []LinkedTxn              `json:"LinkedTxn,omitempty"`
	SalesItemLineDetail   *SalesItemLineDetail     `json:"SalesItemLineDetail,omitempty"`
	DescriptionLineDetail *DescriptionLineDetail   `json:"DescriptionLineDetail,omitempty"`
	DiscountLineDetail    *DiscountLineDetail      `json:"DiscountLineDetail,omitempty"`
	GroupLineDetail       *EstimateGroupLineDetail `json:"GroupLineDetail,omitempty"`
	SubTotalLineDetail    *SubTotalLineDetail      `json:"SubTotalLineDetail,omitempty"`
}

// EstimateGroupLineDetail is the group line detail type for Estimate. It uses
// []EstimateLine for its nested lines so that only Estimate-valid detail types
// are accessible within a group.
type EstimateGroupLineDetail struct {
	GroupItemRef *ReferenceType `json:"GroupItemRef,omitempty"`
	Quantity     json.Number    `json:"Quantity,omitempty"`
	Line         []EstimateLine `json:"Line,omitempty"`
}

type estimateEnvelope struct {
	Estimate Estimate `json:"Estimate"`
}

type estimateQueryEnvelope struct {
	QueryResponse QueryResponse `json:"QueryResponse"`
}

func (s *EstimateService) Get(ctx context.Context, id string, opts ...RequestOption) (*Estimate, error) {
	var out estimateEnvelope
	if err := s.doJSON(ctx, http.MethodGet, "estimate/"+id, nil, nil, &out, opts...); err != nil {
		return nil, err
	}
	return &out.Estimate, nil
}

func (s *EstimateService) Query(ctx context.Context, query string, opts ...QueryOption) ([]Estimate, error) {
	query = applyQueryOptions(query, opts...)

	var out estimateQueryEnvelope
	if err := s.doJSON(ctx, http.MethodGet, "query", url.Values{"query": []string{query}}, nil, &out); err != nil {
		return nil, err
	}

	raw, err := out.QueryResponse.Entities(EntityEstimate)
	if err != nil {
		return nil, err
	}

	return decodeEntitySlice[Estimate](raw)
}

func (s *EstimateService) List(ctx context.Context, opts ...QueryOption) ([]Estimate, error) {
	return s.Query(ctx, "SELECT * FROM Estimate", opts...)
}

func (s *EstimateService) ListPage(ctx context.Context, page, pageSize int, opts ...QueryOption) ([]Estimate, error) {
	pageOpts := append([]QueryOption{WithPage(page, pageSize)}, opts...)
	return s.List(ctx, pageOpts...)
}

func (s *EstimateService) Create(ctx context.Context, estimate Estimate, opts ...RequestOption) (*Estimate, error) {
	var out estimateEnvelope
	if err := s.doJSON(ctx, http.MethodPost, "estimate", nil, estimate, &out, opts...); err != nil {
		return nil, err
	}
	return &out.Estimate, nil
}

func (s *EstimateService) Update(ctx context.Context, estimate Estimate, opts ...RequestOption) (*Estimate, error) {
	var out estimateEnvelope
	if err := s.doJSON(ctx, http.MethodPost, "estimate", nil, estimate, &out, opts...); err != nil {
		return nil, err
	}
	return &out.Estimate, nil
}

func (s *EstimateService) UpdateSparse(ctx context.Context, estimate Estimate, opts ...RequestOption) (*Estimate, error) {
	payload := struct {
		Estimate
		Sparse bool `json:"sparse"`
	}{estimate, true}
	var out estimateEnvelope
	if err := s.doJSON(ctx, http.MethodPost, "estimate", nil, payload, &out, opts...); err != nil {
		return nil, err
	}
	return &out.Estimate, nil
}

func (s *EstimateService) Delete(ctx context.Context, req DeleteRequest, opts ...RequestOption) (*Estimate, error) {
	var out estimateEnvelope
	query := url.Values{"operation": []string{"delete"}}
	if err := s.doJSON(ctx, http.MethodPost, "estimate", query, req, &out, opts...); err != nil {
		return nil, err
	}
	return &out.Estimate, nil
}

func (s *EstimateService) GetPDF(ctx context.Context, id string, opts ...RequestOption) ([]byte, error) {
	return s.doBytes(ctx, http.MethodGet, "estimate/"+id+"/pdf", nil, "application/pdf", opts...)
}

func (s *EstimateService) StreamPDF(ctx context.Context, id string, opts ...RequestOption) (io.ReadCloser, error) {
	return s.doStream(ctx, http.MethodGet, "estimate/"+id+"/pdf", nil, "application/pdf", opts...)
}

func (s *EstimateService) Send(ctx context.Context, id, email string, opts ...RequestOption) error {
	addr, err := mail.ParseAddress(email)
	if err != nil {
		return err
	}

	query := url.Values{"sendTo": []string{addr.Address}}

	return s.doJSON(ctx, http.MethodPost, "estimate/"+id+"/send", query, nil, nil, opts...)
}

func (s *EstimateService) ChangeDataCapture(ctx context.Context, changedSince time.Time, opts ...RequestOption) ([]Estimate, error) {
	resp, err := s.company.ChangeDataCapture(ctx, changedSince, []EntityName{EntityEstimate}, opts...)
	if err != nil {
		return nil, err
	}
	return s.ExtractCDC(resp)
}

func (s *EstimateService) ExtractCDC(resp *CDCResponse) ([]Estimate, error) {
	return ExtractCDCEntities[Estimate](resp, EntityEstimate)
}

func (s *EstimateService) ExtractBatch(resp *BatchResponse) ([]Estimate, error) {
	return ExtractBatchEntities[Estimate](resp, EntityEstimate)
}

func (s *EstimateService) ExtractBatchItems(items []BatchItemResponse) ([]Estimate, error) {
	return ExtractBatchItemEntities[Estimate](items, EntityEstimate)
}

func (s *EstimateService) BatchCreateItem(id string, estimate Estimate) BatchItemRequest {
	return NewBatchEntity(id, BatchOperationCreate, EntityEstimate, estimate)
}

func (s *EstimateService) BatchUpdateItem(id string, estimate Estimate) BatchItemRequest {
	return NewBatchEntity(id, BatchOperationUpdate, EntityEstimate, estimate)
}

func (s *EstimateService) BatchDeleteItem(id string, req DeleteRequest) BatchItemRequest {
	return NewBatchEntity(id, BatchOperationDelete, EntityEstimate, req)
}

func (s *EstimateService) BatchQueryItem(id, query string, opts ...QueryOption) BatchItemRequest {
	return NewBatchQuery(id, query, opts...)
}
