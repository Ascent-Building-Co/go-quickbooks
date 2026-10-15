package quickbooks

import (
	"context"
	"net/http"
	"net/url"
)

// TaxCodeService provides access to the TaxCode-related endpoints of the
// QuickBooks Online API.
type TaxCodeService struct {
	companyService
}

func (c *CompanyClient) TaxCodes() *TaxCodeService {
	return &TaxCodeService{companyService: companyService{company: c}}
}

// TaxRateDetail specifies a single tax rate reference within a TaxRateList,
// including how and in what order the rate is applied.
type TaxRateDetail struct {
	// Required
	TaxRateRef *ReferenceType `json:"TaxRateRef,omitempty"`

	// Optional
	TaxTypeApplicable string `json:"TaxTypeApplicable,omitempty"`
	TaxOrder          int    `json:"TaxOrder,omitempty"`
}

// TaxRateList is a collection of TaxRateDetail entries associated with a
// TaxCode for either sales or purchase transactions.
type TaxRateList struct {
	TaxRateDetail []TaxRateDetail `json:"TaxRateDetail,omitempty"`
}

// TaxCode represents a QuickBooks Online TaxCode entity, which groups one or
// more tax rates applied to sales and purchase transactions.
//
// TaxCode is read-only via the QBO API — CREATE and UPDATE are not supported.
// Use QUERY or READ to retrieve tax codes.
//
// Fields marked "Read-only" are set by the server and ignored if sent in a
// write request.
type TaxCode struct {
	IDValue
	SyncTokenValue

	MetaDataValue

	// Required
	Name string `json:"Name,omitempty"`

	// Conditionally required
	SalesTaxRateList    *TaxRateList `json:"SalesTaxRateList,omitempty"`
	PurchaseTaxRateList *TaxRateList `json:"PurchaseTaxRateList,omitempty"`

	// Optional
	TaxGroup    *bool  `json:"TaxGroup,omitempty"`
	Taxable     *bool  `json:"Taxable,omitempty"`
	Active      *bool  `json:"Active,omitempty"`
	Description string `json:"Description,omitempty"`
	Hidden      *bool  `json:"Hidden,omitempty"`

	// Read-only: set by the server.
	TaxCodeConfigType string `json:"TaxCodeConfigType,omitempty"`
}

func (TaxCode) EntityName() EntityName {
	return EntityTaxCode
}

type taxCodeEnvelope struct {
	TaxCode TaxCode `json:"TaxCode"`
}

type taxCodeQueryEnvelope struct {
	QueryResponse QueryResponse `json:"QueryResponse"`
}

func (s *TaxCodeService) Get(ctx context.Context, id string, opts ...RequestOption) (*TaxCode, error) {
	var out taxCodeEnvelope
	if err := s.doJSON(ctx, http.MethodGet, "taxcode/"+id, nil, nil, &out, opts...); err != nil {
		return nil, err
	}
	return &out.TaxCode, nil
}

func (s *TaxCodeService) Query(ctx context.Context, query string, opts ...QueryOption) ([]TaxCode, error) {
	query = applyQueryOptions(query, opts...)

	var out taxCodeQueryEnvelope
	if err := s.doJSON(ctx, http.MethodGet, "query", url.Values{"query": []string{query}}, nil, &out); err != nil {
		return nil, err
	}

	raw, err := out.QueryResponse.Entities(EntityTaxCode)
	if err != nil {
		return nil, err
	}

	return decodeEntitySlice[TaxCode](raw)
}

func (s *TaxCodeService) List(ctx context.Context, opts ...QueryOption) ([]TaxCode, error) {
	return s.Query(ctx, "SELECT * FROM TaxCode", opts...)
}

func (s *TaxCodeService) ListPage(ctx context.Context, page, pageSize int, opts ...QueryOption) ([]TaxCode, error) {
	pageOpts := append([]QueryOption{WithPage(page, pageSize)}, opts...)
	return s.List(ctx, pageOpts...)
}

func (s *TaxCodeService) ExtractBatch(resp *BatchResponse) ([]TaxCode, error) {
	return ExtractBatchEntities[TaxCode](resp, EntityTaxCode)
}

func (s *TaxCodeService) ExtractBatchItems(items []BatchItemResponse) ([]TaxCode, error) {
	return ExtractBatchItemEntities[TaxCode](items, EntityTaxCode)
}

func (s *TaxCodeService) BatchQueryItem(id, query string, opts ...QueryOption) BatchItemRequest {
	return NewBatchQuery(id, query, opts...)
}
