package quickbooks

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"time"
)

type EmployeeService struct {
	companyService
}

func (c *CompanyClient) Employees() *EmployeeService {
	return &EmployeeService{companyService: companyService{company: c}}
}

// Employee represents a QuickBooks Online Employee entity.
//
// Fields marked "Read-only" are set by the server and ignored if sent in a
// write request.
//
// Gender accepts "Male" or "Female". To clear the value, omit it on a full
// update (it will be set to null server-side).
type Employee struct {
	IDValue
	SyncTokenValue
	CDCFields

	MetaData *ModificationMetaData `json:"MetaData,omitempty"`

	// Conditionally required
	PrimaryAddr *PhysicalAddress `json:"PrimaryAddr,omitempty"`

	// Optional
	DisplayName      string           `json:"DisplayName,omitempty"`
	Title            string           `json:"Title,omitempty"`
	GivenName        string           `json:"GivenName,omitempty"`
	MiddleName       string           `json:"MiddleName,omitempty"`
	FamilyName       string           `json:"FamilyName,omitempty"`
	Suffix           string           `json:"Suffix,omitempty"`
	PrintOnCheckName string           `json:"PrintOnCheckName,omitempty"`
	EmployeeNumber   string           `json:"EmployeeNumber,omitempty"`
	SSN              string           `json:"SSN,omitempty"`
	Gender           string           `json:"Gender,omitempty"`
	BirthDate        *Date            `json:"BirthDate,omitempty"`
	HiredDate        *Date            `json:"HiredDate,omitempty"`
	ReleasedDate     *Date            `json:"ReleasedDate,omitempty"`
	Active           *bool            `json:"Active,omitempty"`
	BillableTime     *bool            `json:"BillableTime,omitempty"`
	Organization     *bool            `json:"Organization,omitempty"`
	PrimaryPhone     *TelephoneNumber `json:"PrimaryPhone,omitempty"`
	Mobile           *TelephoneNumber `json:"Mobile,omitempty"`
	PrimaryEmailAddr *EmailAddress    `json:"PrimaryEmailAddr,omitempty"`
	CostRate         json.Number      `json:"CostRate,omitempty"`
	BillRate         json.Number      `json:"BillRate,omitempty"`

	// Read-only: set by the server.
	// V4IDPseudonym is an internal employee reference number used by QuickBooks
	// for integration purposes. It is not intended for display or business logic.
	V4IDPseudonym string `json:"V4IDPseudonym,omitempty"`
}

func (Employee) EntityName() EntityName {
	return EntityEmployee
}

type employeeEnvelope struct {
	Employee Employee `json:"Employee"`
}

type employeeQueryEnvelope struct {
	QueryResponse QueryResponse `json:"QueryResponse"`
}

func (s *EmployeeService) Get(ctx context.Context, id string, opts ...RequestOption) (*Employee, error) {
	var out employeeEnvelope
	if err := s.doJSON(ctx, http.MethodGet, "employee/"+id, nil, nil, &out, opts...); err != nil {
		return nil, err
	}
	return &out.Employee, nil
}

func (s *EmployeeService) Query(ctx context.Context, query string, opts ...QueryOption) ([]Employee, error) {
	query = applyQueryOptions(query, opts...)

	var out employeeQueryEnvelope
	if err := s.doJSON(ctx, http.MethodGet, "query", url.Values{"query": []string{query}}, nil, &out); err != nil {
		return nil, err
	}

	raw, err := out.QueryResponse.Entities(EntityEmployee)
	if err != nil {
		return nil, err
	}

	return decodeEntitySlice[Employee](raw)
}

func (s *EmployeeService) List(ctx context.Context, opts ...QueryOption) ([]Employee, error) {
	return s.Query(ctx, "SELECT * FROM Employee", opts...)
}

func (s *EmployeeService) ListPage(ctx context.Context, page, pageSize int, opts ...QueryOption) ([]Employee, error) {
	pageOpts := append([]QueryOption{WithPage(page, pageSize)}, opts...)
	return s.List(ctx, pageOpts...)
}

func (s *EmployeeService) Create(ctx context.Context, employee Employee, opts ...RequestOption) (*Employee, error) {
	var out employeeEnvelope
	if err := s.doJSON(ctx, http.MethodPost, "employee", nil, employee, &out, opts...); err != nil {
		return nil, err
	}
	return &out.Employee, nil
}

func (s *EmployeeService) Update(ctx context.Context, employee Employee, opts ...RequestOption) (*Employee, error) {
	var out employeeEnvelope
	if err := s.doJSON(ctx, http.MethodPost, "employee", nil, employee, &out, opts...); err != nil {
		return nil, err
	}
	return &out.Employee, nil
}

func (s *EmployeeService) UpdateSparse(ctx context.Context, employee Employee, opts ...RequestOption) (*Employee, error) {
	payload := struct {
		Employee
		Sparse bool `json:"sparse"`
	}{employee, true}
	var out employeeEnvelope
	if err := s.doJSON(ctx, http.MethodPost, "employee", nil, payload, &out, opts...); err != nil {
		return nil, err
	}
	return &out.Employee, nil
}

func (s *EmployeeService) ChangeDataCapture(ctx context.Context, changedSince time.Time, opts ...RequestOption) ([]Employee, error) {
	resp, err := s.company.ChangeDataCapture(ctx, changedSince, []EntityName{EntityEmployee}, opts...)
	if err != nil {
		return nil, err
	}
	return s.ExtractCDC(resp)
}

func (s *EmployeeService) ExtractCDC(resp *CDCResponse) ([]Employee, error) {
	return ExtractCDCEntities[Employee](resp, EntityEmployee)
}

func (s *EmployeeService) ExtractBatch(resp *BatchResponse) ([]Employee, error) {
	return ExtractBatchEntities[Employee](resp, EntityEmployee)
}

func (s *EmployeeService) ExtractBatchItems(items []BatchItemResponse) ([]Employee, error) {
	return ExtractBatchItemEntities[Employee](items, EntityEmployee)
}

func (s *EmployeeService) BatchCreateItem(id string, employee Employee) BatchItemRequest {
	return NewBatchEntity(id, BatchOperationCreate, EntityEmployee, employee)
}

func (s *EmployeeService) BatchUpdateItem(id string, employee Employee) BatchItemRequest {
	return NewBatchEntity(id, BatchOperationUpdate, EntityEmployee, employee)
}

func (s *EmployeeService) BatchQueryItem(id, query string, opts ...QueryOption) BatchItemRequest {
	return NewBatchQuery(id, query, opts...)
}
