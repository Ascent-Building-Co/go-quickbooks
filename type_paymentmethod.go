package quickbooks

import (
	"context"
	"net/http"
	"net/url"
	"time"
)

// PaymentMethodService provides access to the PaymentMethod-related endpoints
// of the QuickBooks Online API.
type PaymentMethodService struct {
	companyService
}

func (c *CompanyClient) PaymentMethods() *PaymentMethodService {
	return &PaymentMethodService{companyService: companyService{company: c}}
}

// PaymentMethod represents a QuickBooks Online PaymentMethod entity, which
// defines how a customer pays (e.g. cash, check, credit card).
//
// The Type field is constrained to "CREDIT_CARD" or "NON_CREDIT_CARD" by the
// API, though it is modelled as a plain string.
//
// Fields marked "Read-only" are set by the server and ignored if sent in a
// write request.
type PaymentMethod struct {
	IDValue
	SyncTokenValue
	CDCFields

	MetaDataValue

	// Required
	Name string `json:"Name,omitempty"`

	// Optional
	Active *bool  `json:"Active,omitempty"`
	Type   string `json:"Type,omitempty"`
}

func (PaymentMethod) EntityName() EntityName {
	return EntityPaymentMethod
}

type paymentMethodEnvelope struct {
	PaymentMethod PaymentMethod `json:"PaymentMethod"`
}

type paymentMethodQueryEnvelope struct {
	QueryResponse QueryResponse `json:"QueryResponse"`
}

func (s *PaymentMethodService) Get(ctx context.Context, id string, opts ...RequestOption) (*PaymentMethod, error) {
	var out paymentMethodEnvelope
	if err := s.doJSON(ctx, http.MethodGet, "paymentmethod/"+id, nil, nil, &out, opts...); err != nil {
		return nil, err
	}
	return &out.PaymentMethod, nil
}

func (s *PaymentMethodService) Query(ctx context.Context, query string, opts ...QueryOption) ([]PaymentMethod, error) {
	query = applyQueryOptions(query, opts...)

	var out paymentMethodQueryEnvelope
	if err := s.doJSON(ctx, http.MethodGet, "query", url.Values{"query": []string{query}}, nil, &out); err != nil {
		return nil, err
	}

	raw, err := out.QueryResponse.Entities(EntityPaymentMethod)
	if err != nil {
		return nil, err
	}

	return decodeEntitySlice[PaymentMethod](raw)
}

func (s *PaymentMethodService) List(ctx context.Context, opts ...QueryOption) ([]PaymentMethod, error) {
	return s.Query(ctx, "SELECT * FROM PaymentMethod", opts...)
}

func (s *PaymentMethodService) ListPage(ctx context.Context, page, pageSize int, opts ...QueryOption) ([]PaymentMethod, error) {
	pageOpts := append([]QueryOption{WithPage(page, pageSize)}, opts...)
	return s.List(ctx, pageOpts...)
}

func (s *PaymentMethodService) Create(ctx context.Context, pm PaymentMethod, opts ...RequestOption) (*PaymentMethod, error) {
	var out paymentMethodEnvelope
	if err := s.doJSON(ctx, http.MethodPost, "paymentmethod", nil, pm, &out, opts...); err != nil {
		return nil, err
	}
	return &out.PaymentMethod, nil
}

func (s *PaymentMethodService) Update(ctx context.Context, pm PaymentMethod, opts ...RequestOption) (*PaymentMethod, error) {
	var out paymentMethodEnvelope
	if err := s.doJSON(ctx, http.MethodPost, "paymentmethod", nil, pm, &out, opts...); err != nil {
		return nil, err
	}
	return &out.PaymentMethod, nil
}

func (s *PaymentMethodService) UpdateSparse(ctx context.Context, pm PaymentMethod, opts ...RequestOption) (*PaymentMethod, error) {
	payload := struct {
		PaymentMethod
		Sparse bool `json:"sparse"`
	}{pm, true}
	var out paymentMethodEnvelope
	if err := s.doJSON(ctx, http.MethodPost, "paymentmethod", nil, payload, &out, opts...); err != nil {
		return nil, err
	}
	return &out.PaymentMethod, nil
}

func (s *PaymentMethodService) ChangeDataCapture(ctx context.Context, changedSince time.Time, opts ...RequestOption) ([]PaymentMethod, error) {
	resp, err := s.company.ChangeDataCapture(ctx, changedSince, []EntityName{EntityPaymentMethod}, opts...)
	if err != nil {
		return nil, err
	}
	return s.ExtractCDC(resp)
}

func (s *PaymentMethodService) ExtractCDC(resp *CDCResponse) ([]PaymentMethod, error) {
	return ExtractCDCEntities[PaymentMethod](resp, EntityPaymentMethod)
}

func (s *PaymentMethodService) ExtractBatch(resp *BatchResponse) ([]PaymentMethod, error) {
	return ExtractBatchEntities[PaymentMethod](resp, EntityPaymentMethod)
}

func (s *PaymentMethodService) ExtractBatchItems(items []BatchItemResponse) ([]PaymentMethod, error) {
	return ExtractBatchItemEntities[PaymentMethod](items, EntityPaymentMethod)
}

func (s *PaymentMethodService) BatchCreateItem(id string, pm PaymentMethod) BatchItemRequest {
	return NewBatchEntity(id, BatchOperationCreate, EntityPaymentMethod, pm)
}

func (s *PaymentMethodService) BatchUpdateItem(id string, pm PaymentMethod) BatchItemRequest {
	return NewBatchEntity(id, BatchOperationUpdate, EntityPaymentMethod, pm)
}

func (s *PaymentMethodService) BatchQueryItem(id, query string, opts ...QueryOption) BatchItemRequest {
	return NewBatchQuery(id, query, opts...)
}
