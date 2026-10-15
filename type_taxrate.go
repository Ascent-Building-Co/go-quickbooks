package quickbooks

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
)

// TaxRateService provides access to the TaxRate-related endpoints of the
// QuickBooks Online API.
type TaxRateService struct {
	companyService
}

func (c *CompanyClient) TaxRates() *TaxRateService {
	return &TaxRateService{companyService: companyService{company: c}}
}

// EffectiveTaxRate holds a single entry in the effective tax rate history for a
// TaxRate, capturing the rate value and the date range during which it applies.
type EffectiveTaxRate struct {
	RateValue     json.Number `json:"RateValue,omitempty"`
	EffectiveDate *DateTime   `json:"EffectiveDate,omitempty"`
	EndDate       *DateTime   `json:"EndDate,omitempty"`
}

// TaxRate represents a QuickBooks Online TaxRate entity, which defines a
// specific percentage charged by a tax agency.
//
// TaxRate is read-only via the QBO API — CREATE and UPDATE are not supported.
// Use QUERY or READ to retrieve tax rates.
//
// RateValue is typed as a string by the QBO API schema.
type TaxRate struct {
	IDValue
	SyncTokenValue

	MetaDataValue

	// Optional
	Name             string             `json:"Name,omitempty"`
	RateValue        json.Number        `json:"RateValue,omitempty"`
	AgencyRef        *ReferenceType     `json:"AgencyRef,omitempty"`
	SpecialTaxType   string             `json:"SpecialTaxType,omitempty"`
	EffectiveTaxRate []EffectiveTaxRate `json:"EffectiveTaxRate,omitempty"`
	DisplayType      string             `json:"DisplayType,omitempty"`
	TaxReturnLineRef *ReferenceType     `json:"TaxReturnLineRef,omitempty"`
	Active           *bool              `json:"Active,omitempty"`
	OriginalTaxRate  string             `json:"OriginalTaxRate,omitempty"`
	Description      string             `json:"Description,omitempty"`
}

func (TaxRate) EntityName() EntityName {
	return EntityTaxRate
}

type taxRateEnvelope struct {
	TaxRate TaxRate `json:"TaxRate"`
}

type taxRateQueryEnvelope struct {
	QueryResponse QueryResponse `json:"QueryResponse"`
}

func (s *TaxRateService) Get(ctx context.Context, id string, opts ...RequestOption) (*TaxRate, error) {
	var out taxRateEnvelope
	if err := s.doJSON(ctx, http.MethodGet, "taxrate/"+id, nil, nil, &out, opts...); err != nil {
		return nil, err
	}
	return &out.TaxRate, nil
}

func (s *TaxRateService) Query(ctx context.Context, query string, opts ...QueryOption) ([]TaxRate, error) {
	query = applyQueryOptions(query, opts...)

	var out taxRateQueryEnvelope
	if err := s.doJSON(ctx, http.MethodGet, "query", url.Values{"query": []string{query}}, nil, &out); err != nil {
		return nil, err
	}

	raw, err := out.QueryResponse.Entities(EntityTaxRate)
	if err != nil {
		return nil, err
	}

	return decodeEntitySlice[TaxRate](raw)
}

func (s *TaxRateService) List(ctx context.Context, opts ...QueryOption) ([]TaxRate, error) {
	return s.Query(ctx, "SELECT * FROM TaxRate", opts...)
}

func (s *TaxRateService) ListPage(ctx context.Context, page, pageSize int, opts ...QueryOption) ([]TaxRate, error) {
	pageOpts := append([]QueryOption{WithPage(page, pageSize)}, opts...)
	return s.List(ctx, pageOpts...)
}

func (s *TaxRateService) ExtractBatch(resp *BatchResponse) ([]TaxRate, error) {
	return ExtractBatchEntities[TaxRate](resp, EntityTaxRate)
}

func (s *TaxRateService) ExtractBatchItems(items []BatchItemResponse) ([]TaxRate, error) {
	return ExtractBatchItemEntities[TaxRate](items, EntityTaxRate)
}

func (s *TaxRateService) BatchQueryItem(id, query string, opts ...QueryOption) BatchItemRequest {
	return NewBatchQuery(id, query, opts...)
}
