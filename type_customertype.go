package quickbooks

import (
	"context"
	"net/http"
	"net/url"
	"time"
)

// CustomerTypeService provides access to the CustomerType-related endpoints
// of the QuickBooks Online API.
type CustomerTypeService struct {
	companyService
}

func (c *CompanyClient) CustomerTypes() *CustomerTypeService {
	return &CustomerTypeService{companyService: companyService{company: c}}
}

// CustomerType represents a QuickBooks Online CustomerType entity, which is a
// user-defined category that can be assigned to customers.
//
// CustomerType is read-only via the QBO API — CREATE and UPDATE are not
// supported. Use QUERY or READ to retrieve customer types.
type CustomerType struct {
	IDValue
	SyncTokenValue
	CDCFields

	MetaData *ModificationMetaData `json:"MetaData,omitempty"`

	// Optional
	Name   string `json:"Name,omitempty"`
	Active *bool  `json:"Active,omitempty"`
}

func (CustomerType) EntityName() EntityName {
	return EntityCustomerType
}

type customerTypeEnvelope struct {
	CustomerType CustomerType `json:"CustomerType"`
}

type customerTypeQueryEnvelope struct {
	QueryResponse QueryResponse `json:"QueryResponse"`
}

func (s *CustomerTypeService) Get(ctx context.Context, id string, opts ...RequestOption) (*CustomerType, error) {
	var out customerTypeEnvelope
	if err := s.doJSON(ctx, http.MethodGet, "customertype/"+id, nil, nil, &out, opts...); err != nil {
		return nil, err
	}
	return &out.CustomerType, nil
}

func (s *CustomerTypeService) Query(ctx context.Context, query string, opts ...QueryOption) ([]CustomerType, error) {
	query = applyQueryOptions(query, opts...)

	var out customerTypeQueryEnvelope
	if err := s.doJSON(ctx, http.MethodGet, "query", url.Values{"query": []string{query}}, nil, &out); err != nil {
		return nil, err
	}

	raw, err := out.QueryResponse.Entities(EntityCustomerType)
	if err != nil {
		return nil, err
	}

	return decodeEntitySlice[CustomerType](raw)
}

func (s *CustomerTypeService) List(ctx context.Context, opts ...QueryOption) ([]CustomerType, error) {
	return s.Query(ctx, "SELECT * FROM CustomerType", opts...)
}

func (s *CustomerTypeService) ListPage(ctx context.Context, page, pageSize int, opts ...QueryOption) ([]CustomerType, error) {
	pageOpts := append([]QueryOption{WithPage(page, pageSize)}, opts...)
	return s.List(ctx, pageOpts...)
}

func (s *CustomerTypeService) ChangeDataCapture(ctx context.Context, changedSince time.Time, opts ...RequestOption) ([]CustomerType, error) {
	resp, err := s.company.ChangeDataCapture(ctx, changedSince, []EntityName{EntityCustomerType}, opts...)
	if err != nil {
		return nil, err
	}
	return s.ExtractCDC(resp)
}

func (s *CustomerTypeService) ExtractCDC(resp *CDCResponse) ([]CustomerType, error) {
	return ExtractCDCEntities[CustomerType](resp, EntityCustomerType)
}

func (s *CustomerTypeService) ExtractBatch(resp *BatchResponse) ([]CustomerType, error) {
	return ExtractBatchEntities[CustomerType](resp, EntityCustomerType)
}

func (s *CustomerTypeService) ExtractBatchItems(items []BatchItemResponse) ([]CustomerType, error) {
	return ExtractBatchItemEntities[CustomerType](items, EntityCustomerType)
}

func (s *CustomerTypeService) BatchQueryItem(id, query string, opts ...QueryOption) BatchItemRequest {
	return NewBatchQuery(id, query, opts...)
}
