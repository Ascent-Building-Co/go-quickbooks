package quickbooks

import (
	"context"
	"net/http"
	"net/url"
	"time"
)

type ClassService struct {
	companyService
}

func (c *CompanyClient) Classes() *ClassService {
	return &ClassService{companyService: companyService{company: c}}
}

// Class represents a QuickBooks Online Class entity, used to categorise
// transactions across multiple segments of the business.
//
// Classes can be hierarchical: when SubClass is true, ParentRef is required
// and FullyQualifiedName will reflect the full parent path.
//
// Fields marked "Read-only" are set by the server and ignored if sent in a
// write request.
type Class struct {
	IDValue
	SyncTokenValue
	CDCFields

	MetaData *ModificationMetaData `json:"MetaData,omitempty"`

	// Required
	Name string `json:"Name,omitempty"`

	// Conditionally required
	ParentRef *ReferenceType `json:"ParentRef,omitempty"`

	// Optional
	SubClass *bool `json:"SubClass,omitempty"`
	Active   *bool `json:"Active,omitempty"`

	// Read-only: set by the server.
	FullyQualifiedName string `json:"FullyQualifiedName,omitempty"`
}

func (Class) EntityName() EntityName {
	return EntityClass
}

type classEnvelope struct {
	Class Class `json:"Class"`
}

type classQueryEnvelope struct {
	QueryResponse QueryResponse `json:"QueryResponse"`
}

func (s *ClassService) Get(ctx context.Context, id string, opts ...RequestOption) (*Class, error) {
	var out classEnvelope
	if err := s.doJSON(ctx, http.MethodGet, "class/"+id, nil, nil, &out, opts...); err != nil {
		return nil, err
	}
	return &out.Class, nil
}

func (s *ClassService) Query(ctx context.Context, query string, opts ...QueryOption) ([]Class, error) {
	query = applyQueryOptions(query, opts...)

	var out classQueryEnvelope
	if err := s.doJSON(ctx, http.MethodGet, "query", url.Values{"query": []string{query}}, nil, &out); err != nil {
		return nil, err
	}

	raw, err := out.QueryResponse.Entities(EntityClass)
	if err != nil {
		return nil, err
	}

	return decodeEntitySlice[Class](raw)
}

func (s *ClassService) List(ctx context.Context, opts ...QueryOption) ([]Class, error) {
	return s.Query(ctx, "SELECT * FROM Class", opts...)
}

func (s *ClassService) ListPage(ctx context.Context, page, pageSize int, opts ...QueryOption) ([]Class, error) {
	pageOpts := append([]QueryOption{WithPage(page, pageSize)}, opts...)
	return s.List(ctx, pageOpts...)
}

func (s *ClassService) Create(ctx context.Context, class Class, opts ...RequestOption) (*Class, error) {
	var out classEnvelope
	if err := s.doJSON(ctx, http.MethodPost, "class", nil, class, &out, opts...); err != nil {
		return nil, err
	}
	return &out.Class, nil
}

func (s *ClassService) Update(ctx context.Context, class Class, opts ...RequestOption) (*Class, error) {
	var out classEnvelope
	if err := s.doJSON(ctx, http.MethodPost, "class", nil, class, &out, opts...); err != nil {
		return nil, err
	}
	return &out.Class, nil
}

func (s *ClassService) UpdateSparse(ctx context.Context, class Class, opts ...RequestOption) (*Class, error) {
	payload := struct {
		Class
		Sparse bool `json:"sparse"`
	}{class, true}
	var out classEnvelope
	if err := s.doJSON(ctx, http.MethodPost, "class", nil, payload, &out, opts...); err != nil {
		return nil, err
	}
	return &out.Class, nil
}

func (s *ClassService) ChangeDataCapture(ctx context.Context, changedSince time.Time, opts ...RequestOption) ([]Class, error) {
	resp, err := s.company.ChangeDataCapture(ctx, changedSince, []EntityName{EntityClass}, opts...)
	if err != nil {
		return nil, err
	}
	return s.ExtractCDC(resp)
}

func (s *ClassService) ExtractCDC(resp *CDCResponse) ([]Class, error) {
	return ExtractCDCEntities[Class](resp, EntityClass)
}

func (s *ClassService) ExtractBatch(resp *BatchResponse) ([]Class, error) {
	return ExtractBatchEntities[Class](resp, EntityClass)
}

func (s *ClassService) ExtractBatchItems(items []BatchItemResponse) ([]Class, error) {
	return ExtractBatchItemEntities[Class](items, EntityClass)
}

func (s *ClassService) BatchCreateItem(id string, class Class) BatchItemRequest {
	return NewBatchEntity(id, BatchOperationCreate, EntityClass, class)
}

func (s *ClassService) BatchUpdateItem(id string, class Class) BatchItemRequest {
	return NewBatchEntity(id, BatchOperationUpdate, EntityClass, class)
}

func (s *ClassService) BatchQueryItem(id, query string, opts ...QueryOption) BatchItemRequest {
	return NewBatchQuery(id, query, opts...)
}
