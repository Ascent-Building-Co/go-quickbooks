package quickbooks

import (
	"context"
	"net/http"
	"net/url"
	"time"
)

type DepartmentService struct {
	companyService
}

func (c *CompanyClient) Departments() *DepartmentService {
	return &DepartmentService{companyService: companyService{company: c}}
}

// Department represents a QuickBooks Online Department entity, used to
// organise transactions by business unit or location.
//
// Departments can be hierarchical: when SubDepartment is true, ParentRef is
// required and FullyQualifiedName will reflect the full parent path.
//
// Fields marked "Read-only" are set by the server and ignored if sent in a
// write request.
type Department struct {
	IDValue
	SyncTokenValue
	CDCFields

	MetaData *ModificationMetaData `json:"MetaData,omitempty"`

	// Required
	Name string `json:"Name,omitempty"`

	// Conditionally required
	ParentRef *ReferenceType `json:"ParentRef,omitempty"`

	// Optional
	Active *bool `json:"Active,omitempty"`

	// Read-only: set by the server.
	FullyQualifiedName string `json:"FullyQualifiedName,omitempty"`
	SubDepartment      *bool  `json:"SubDepartment,omitempty"`
}

func (Department) EntityName() EntityName {
	return EntityDepartment
}

type departmentEnvelope struct {
	Department Department `json:"Department"`
}

type departmentQueryEnvelope struct {
	QueryResponse QueryResponse `json:"QueryResponse"`
}

func (s *DepartmentService) Get(ctx context.Context, id string, opts ...RequestOption) (*Department, error) {
	var out departmentEnvelope
	if err := s.doJSON(ctx, http.MethodGet, "department/"+id, nil, nil, &out, opts...); err != nil {
		return nil, err
	}
	return &out.Department, nil
}

func (s *DepartmentService) Query(ctx context.Context, query string, opts ...QueryOption) ([]Department, error) {
	query = applyQueryOptions(query, opts...)

	var out departmentQueryEnvelope
	if err := s.doJSON(ctx, http.MethodGet, "query", url.Values{"query": []string{query}}, nil, &out); err != nil {
		return nil, err
	}

	raw, err := out.QueryResponse.Entities(EntityDepartment)
	if err != nil {
		return nil, err
	}

	return decodeEntitySlice[Department](raw)
}

func (s *DepartmentService) List(ctx context.Context, opts ...QueryOption) ([]Department, error) {
	return s.Query(ctx, "SELECT * FROM Department", opts...)
}

func (s *DepartmentService) ListPage(ctx context.Context, page, pageSize int, opts ...QueryOption) ([]Department, error) {
	pageOpts := append([]QueryOption{WithPage(page, pageSize)}, opts...)
	return s.List(ctx, pageOpts...)
}

func (s *DepartmentService) Create(ctx context.Context, dept Department, opts ...RequestOption) (*Department, error) {
	var out departmentEnvelope
	if err := s.doJSON(ctx, http.MethodPost, "department", nil, dept, &out, opts...); err != nil {
		return nil, err
	}
	return &out.Department, nil
}

func (s *DepartmentService) Update(ctx context.Context, dept Department, opts ...RequestOption) (*Department, error) {
	var out departmentEnvelope
	if err := s.doJSON(ctx, http.MethodPost, "department", nil, dept, &out, opts...); err != nil {
		return nil, err
	}
	return &out.Department, nil
}

func (s *DepartmentService) UpdateSparse(ctx context.Context, dept Department, opts ...RequestOption) (*Department, error) {
	payload := struct {
		Department
		Sparse bool `json:"sparse"`
	}{dept, true}
	var out departmentEnvelope
	if err := s.doJSON(ctx, http.MethodPost, "department", nil, payload, &out, opts...); err != nil {
		return nil, err
	}
	return &out.Department, nil
}

func (s *DepartmentService) ChangeDataCapture(ctx context.Context, changedSince time.Time, opts ...RequestOption) ([]Department, error) {
	resp, err := s.company.ChangeDataCapture(ctx, changedSince, []EntityName{EntityDepartment}, opts...)
	if err != nil {
		return nil, err
	}
	return s.ExtractCDC(resp)
}

func (s *DepartmentService) ExtractCDC(resp *CDCResponse) ([]Department, error) {
	return ExtractCDCEntities[Department](resp, EntityDepartment)
}

func (s *DepartmentService) ExtractBatch(resp *BatchResponse) ([]Department, error) {
	return ExtractBatchEntities[Department](resp, EntityDepartment)
}

func (s *DepartmentService) ExtractBatchItems(items []BatchItemResponse) ([]Department, error) {
	return ExtractBatchItemEntities[Department](items, EntityDepartment)
}

func (s *DepartmentService) BatchCreateItem(id string, dept Department) BatchItemRequest {
	return NewBatchEntity(id, BatchOperationCreate, EntityDepartment, dept)
}

func (s *DepartmentService) BatchUpdateItem(id string, dept Department) BatchItemRequest {
	return NewBatchEntity(id, BatchOperationUpdate, EntityDepartment, dept)
}

func (s *DepartmentService) BatchQueryItem(id, query string, opts ...QueryOption) BatchItemRequest {
	return NewBatchQuery(id, query, opts...)
}
