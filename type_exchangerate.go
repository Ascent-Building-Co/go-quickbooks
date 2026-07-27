package quickbooks

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
)

// ExchangeRateService provides access to the ExchangeRate-related endpoints
// of the QuickBooks Online API.
type ExchangeRateService struct {
	companyService
}

func (c *CompanyClient) ExchangeRates() *ExchangeRateService {
	return &ExchangeRateService{companyService: companyService{company: c}}
}

// ExchangeRate represents a QuickBooks Online ExchangeRate entity, which
// provides the exchange rate between a source currency and a target currency
// (defaulting to the home currency) on a given date.
//
// ExchangeRate does not have an Id field — it is identified by the
// combination of SourceCurrencyCode and AsOfDate. As such, it does not
// satisfy the Entity interface.
//
// Supported operations are Read (via query params), Query, and Update.
// Create and Delete are not supported.
type ExchangeRate struct {
	SyncTokenValue

	MetaData    *ModificationMetaData `json:"MetaData,omitempty"`
	CustomField []CustomField         `json:"CustomField,omitempty"`

	// Required for update
	SourceCurrencyCode string      `json:"SourceCurrencyCode,omitempty"`
	AsOfDate           *Date       `json:"AsOfDate,omitempty"`
	Rate               json.Number `json:"Rate,omitempty"`

	// Optional
	TargetCurrencyCode string `json:"TargetCurrencyCode,omitempty"`
}

func (ExchangeRate) EntityName() EntityName {
	return EntityExchangeRate
}

type exchangeRateEnvelope struct {
	ExchangeRate ExchangeRate `json:"ExchangeRate"`
}

type exchangeRateQueryEnvelope struct {
	QueryResponse QueryResponse `json:"QueryResponse"`
}

// Get retrieves an exchange rate by source currency code and optional date.
// If asOfDate is nil, the server returns the rate for the current date.
// This uses query parameters rather than a path-based ID because
// ExchangeRate has no Id field.
func (s *ExchangeRateService) Get(ctx context.Context, sourceCurrencyCode string, asOfDate *Date, opts ...RequestOption) (*ExchangeRate, error) {
	query := url.Values{"sourcecurrencycode": []string{sourceCurrencyCode}}
	if asOfDate != nil && !asOfDate.Time.IsZero() {
		query.Set("asofdate", asOfDate.String())
	}
	var out exchangeRateEnvelope
	if err := s.doJSON(ctx, http.MethodGet, "exchangerate", query, nil, &out, opts...); err != nil {
		return nil, err
	}
	return &out.ExchangeRate, nil
}

func (s *ExchangeRateService) Query(ctx context.Context, query string, opts ...QueryOption) ([]ExchangeRate, error) {
	query = applyQueryOptions(query, opts...)

	var out exchangeRateQueryEnvelope
	if err := s.doJSON(ctx, http.MethodGet, "query", url.Values{"query": []string{query}}, nil, &out); err != nil {
		return nil, err
	}

	raw, err := out.QueryResponse.Entities(EntityExchangeRate)
	if err != nil {
		return nil, err
	}

	return decodeEntitySlice[ExchangeRate](raw)
}

func (s *ExchangeRateService) List(ctx context.Context, opts ...QueryOption) ([]ExchangeRate, error) {
	return s.Query(ctx, "SELECT * FROM ExchangeRate", opts...)
}

func (s *ExchangeRateService) ListPage(ctx context.Context, page, pageSize int, opts ...QueryOption) ([]ExchangeRate, error) {
	pageOpts := append([]QueryOption{WithPage(page, pageSize)}, opts...)
	return s.List(ctx, pageOpts...)
}

func (s *ExchangeRateService) Update(ctx context.Context, rate ExchangeRate, opts ...RequestOption) (*ExchangeRate, error) {
	var out exchangeRateEnvelope
	if err := s.doJSON(ctx, http.MethodPost, "exchangerate", nil, rate, &out, opts...); err != nil {
		return nil, err
	}
	return &out.ExchangeRate, nil
}

func (s *ExchangeRateService) ExtractBatch(resp *BatchResponse) ([]ExchangeRate, error) {
	return ExtractBatchEntities[ExchangeRate](resp, EntityExchangeRate)
}

func (s *ExchangeRateService) ExtractBatchItems(items []BatchItemResponse) ([]ExchangeRate, error) {
	return ExtractBatchItemEntities[ExchangeRate](items, EntityExchangeRate)
}

func (s *ExchangeRateService) BatchUpdateItem(id string, rate ExchangeRate) BatchItemRequest {
	return NewBatchEntity(id, BatchOperationUpdate, EntityExchangeRate, rate)
}

func (s *ExchangeRateService) BatchQueryItem(id, query string, opts ...QueryOption) BatchItemRequest {
	return NewBatchQuery(id, query, opts...)
}
