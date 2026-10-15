package quickbooks

import (
	"context"
	"net/http"
	"net/url"
)

// TaxAgencyService provides access to the TaxAgency-related endpoints of the
// QuickBooks Online API.
type TaxAgencyService struct {
	companyService
}

func (c *CompanyClient) TaxAgencies() *TaxAgencyService {
	return &TaxAgencyService{companyService: companyService{company: c}}
}

// TaxAgency represents a QuickBooks Online TaxAgency entity, which models a
// government tax authority to which taxes are remitted.
//
// TaxAgency does not support UPDATE or DELETE via the QBO API. The SyncToken
// field is present in read responses but serves no write purpose for this
// entity.
//
// Fields marked "Read-only" are set by the server and ignored if sent in a
// write request.
type TaxAgency struct {
	IDValue
	SyncTokenValue

	MetaDataValue

	// Required
	DisplayName string `json:"DisplayName,omitempty"`

	// Optional
	TaxTrackedOnSales     *bool  `json:"TaxTrackedOnSales,omitempty"`
	TaxTrackedOnPurchases *bool  `json:"TaxTrackedOnPurchases,omitempty"`
	LastFileDate          *Date  `json:"LastFileDate,omitempty"`
	TaxRegistrationNumber string `json:"TaxRegistrationNumber,omitempty"`

	// Read-only: set by the server.
	TaxAgencyConfig string `json:"TaxAgencyConfig,omitempty"`
}

func (TaxAgency) EntityName() EntityName {
	return EntityTaxAgency
}

type taxAgencyEnvelope struct {
	TaxAgency TaxAgency `json:"TaxAgency"`
}

type taxAgencyQueryEnvelope struct {
	QueryResponse QueryResponse `json:"QueryResponse"`
}

func (s *TaxAgencyService) Get(ctx context.Context, id string, opts ...RequestOption) (*TaxAgency, error) {
	var out taxAgencyEnvelope
	if err := s.doJSON(ctx, http.MethodGet, "taxagency/"+id, nil, nil, &out, opts...); err != nil {
		return nil, err
	}
	return &out.TaxAgency, nil
}

func (s *TaxAgencyService) Query(ctx context.Context, query string, opts ...QueryOption) ([]TaxAgency, error) {
	query = applyQueryOptions(query, opts...)

	var out taxAgencyQueryEnvelope
	if err := s.doJSON(ctx, http.MethodGet, "query", url.Values{"query": []string{query}}, nil, &out); err != nil {
		return nil, err
	}

	raw, err := out.QueryResponse.Entities(EntityTaxAgency)
	if err != nil {
		return nil, err
	}

	return decodeEntitySlice[TaxAgency](raw)
}

func (s *TaxAgencyService) List(ctx context.Context, opts ...QueryOption) ([]TaxAgency, error) {
	return s.Query(ctx, "SELECT * FROM TaxAgency", opts...)
}

func (s *TaxAgencyService) ListPage(ctx context.Context, page, pageSize int, opts ...QueryOption) ([]TaxAgency, error) {
	pageOpts := append([]QueryOption{WithPage(page, pageSize)}, opts...)
	return s.List(ctx, pageOpts...)
}

func (s *TaxAgencyService) Create(ctx context.Context, ta TaxAgency, opts ...RequestOption) (*TaxAgency, error) {
	var out taxAgencyEnvelope
	if err := s.doJSON(ctx, http.MethodPost, "taxagency", nil, ta, &out, opts...); err != nil {
		return nil, err
	}
	return &out.TaxAgency, nil
}

func (s *TaxAgencyService) ExtractBatch(resp *BatchResponse) ([]TaxAgency, error) {
	return ExtractBatchEntities[TaxAgency](resp, EntityTaxAgency)
}

func (s *TaxAgencyService) ExtractBatchItems(items []BatchItemResponse) ([]TaxAgency, error) {
	return ExtractBatchItemEntities[TaxAgency](items, EntityTaxAgency)
}

func (s *TaxAgencyService) BatchCreateItem(id string, ta TaxAgency) BatchItemRequest {
	return NewBatchEntity(id, BatchOperationCreate, EntityTaxAgency, ta)
}

func (s *TaxAgencyService) BatchQueryItem(id, query string, opts ...QueryOption) BatchItemRequest {
	return NewBatchQuery(id, query, opts...)
}
