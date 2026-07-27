package quickbooks

import (
	"context"
	"net/http"
	"net/url"
	"time"
)

// CompanyCurrencyService provides access to the CompanyCurrency-related
// endpoints of the QuickBooks Online API.
type CompanyCurrencyService struct {
	companyService
}

func (c *CompanyClient) CompanyCurrencies() *CompanyCurrencyService {
	return &CompanyCurrencyService{companyService: companyService{company: c}}
}

// CompanyCurrency represents a QuickBooks Online CompanyCurrency entity,
// defining a currency active in a multicurrency-enabled company. It
// corresponds to the Currency Center list in the QBO UI.
//
// CompanyCurrency does not support hard delete. To deactivate a currency,
// set Active to false via an update request.
//
// Fields marked "Read-only" are set by the server and ignored if sent in a
// write request.
type CompanyCurrency struct {
	IDValue
	SyncTokenValue
	CDCFields

	MetaData    *ModificationMetaData `json:"MetaData,omitempty"`
	CustomField []CustomField         `json:"CustomField,omitempty"`

	// Required
	Code string `json:"Code,omitempty"`

	// Optional
	Name   string `json:"Name,omitempty"`
	Active *bool  `json:"Active,omitempty"`
}

func (CompanyCurrency) EntityName() EntityName {
	return EntityCompanyCurrency
}

type companyCurrencyEnvelope struct {
	CompanyCurrency CompanyCurrency `json:"CompanyCurrency"`
}

type companyCurrencyQueryEnvelope struct {
	QueryResponse QueryResponse `json:"QueryResponse"`
}

func (s *CompanyCurrencyService) Get(ctx context.Context, id string, opts ...RequestOption) (*CompanyCurrency, error) {
	var out companyCurrencyEnvelope
	if err := s.doJSON(ctx, http.MethodGet, "companycurrency/"+id, nil, nil, &out, opts...); err != nil {
		return nil, err
	}
	return &out.CompanyCurrency, nil
}

func (s *CompanyCurrencyService) Query(ctx context.Context, query string, opts ...QueryOption) ([]CompanyCurrency, error) {
	query = applyQueryOptions(query, opts...)

	var out companyCurrencyQueryEnvelope
	if err := s.doJSON(ctx, http.MethodGet, "query", url.Values{"query": []string{query}}, nil, &out); err != nil {
		return nil, err
	}

	raw, err := out.QueryResponse.Entities(EntityCompanyCurrency)
	if err != nil {
		return nil, err
	}

	return decodeEntitySlice[CompanyCurrency](raw)
}

func (s *CompanyCurrencyService) List(ctx context.Context, opts ...QueryOption) ([]CompanyCurrency, error) {
	return s.Query(ctx, "SELECT * FROM CompanyCurrency", opts...)
}

func (s *CompanyCurrencyService) ListPage(ctx context.Context, page, pageSize int, opts ...QueryOption) ([]CompanyCurrency, error) {
	pageOpts := append([]QueryOption{WithPage(page, pageSize)}, opts...)
	return s.List(ctx, pageOpts...)
}

func (s *CompanyCurrencyService) Create(ctx context.Context, currency CompanyCurrency, opts ...RequestOption) (*CompanyCurrency, error) {
	var out companyCurrencyEnvelope
	if err := s.doJSON(ctx, http.MethodPost, "companycurrency", nil, currency, &out, opts...); err != nil {
		return nil, err
	}
	return &out.CompanyCurrency, nil
}

func (s *CompanyCurrencyService) Update(ctx context.Context, currency CompanyCurrency, opts ...RequestOption) (*CompanyCurrency, error) {
	var out companyCurrencyEnvelope
	if err := s.doJSON(ctx, http.MethodPost, "companycurrency", nil, currency, &out, opts...); err != nil {
		return nil, err
	}
	return &out.CompanyCurrency, nil
}

func (s *CompanyCurrencyService) ChangeDataCapture(ctx context.Context, changedSince time.Time, opts ...RequestOption) ([]CompanyCurrency, error) {
	resp, err := s.company.ChangeDataCapture(ctx, changedSince, []EntityName{EntityCompanyCurrency}, opts...)
	if err != nil {
		return nil, err
	}
	return s.ExtractCDC(resp)
}

func (s *CompanyCurrencyService) ExtractCDC(resp *CDCResponse) ([]CompanyCurrency, error) {
	return ExtractCDCEntities[CompanyCurrency](resp, EntityCompanyCurrency)
}

func (s *CompanyCurrencyService) ExtractBatch(resp *BatchResponse) ([]CompanyCurrency, error) {
	return ExtractBatchEntities[CompanyCurrency](resp, EntityCompanyCurrency)
}

func (s *CompanyCurrencyService) ExtractBatchItems(items []BatchItemResponse) ([]CompanyCurrency, error) {
	return ExtractBatchItemEntities[CompanyCurrency](items, EntityCompanyCurrency)
}

func (s *CompanyCurrencyService) BatchCreateItem(id string, currency CompanyCurrency) BatchItemRequest {
	return NewBatchEntity(id, BatchOperationCreate, EntityCompanyCurrency, currency)
}

func (s *CompanyCurrencyService) BatchUpdateItem(id string, currency CompanyCurrency) BatchItemRequest {
	return NewBatchEntity(id, BatchOperationUpdate, EntityCompanyCurrency, currency)
}

func (s *CompanyCurrencyService) BatchQueryItem(id, query string, opts ...QueryOption) BatchItemRequest {
	return NewBatchQuery(id, query, opts...)
}
