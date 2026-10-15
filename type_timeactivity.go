package quickbooks

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
)

// TimeActivityService provides access to the TimeActivity-related endpoints of
// the QuickBooks Online API.
type TimeActivityService struct {
	companyService
}

func (c *CompanyClient) TimeActivities() *TimeActivityService {
	return &TimeActivityService{companyService: companyService{company: c}}
}

// TimeActivity represents a QuickBooks Online TimeActivity entity, which
// records time worked by an employee or vendor against a customer or project.
//
// The NameOf field controls which actor reference is required:
//   - NameOfEmployee: EmployeeRef must be set.
//   - NameOfVendor: VendorRef must be set.
//
// Time can be specified either as a StartTime/EndTime pair or as Hours/Minutes.
// BreakHours and BreakMinutes apply only when StartTime/EndTime are used.
//
// Fields marked "Read-only" are set by the server and ignored if sent in a
// write request.
type TimeActivity struct {
	IDValue
	SyncTokenValue

	MetaDataValue

	// Required
	NameOf NameOfEnum `json:"NameOf,omitempty"`

	// Conditionally required
	TxnDate      *Date          `json:"TxnDate,omitempty"`
	EmployeeRef  *ReferenceType `json:"EmployeeRef,omitempty"`
	VendorRef    *ReferenceType `json:"VendorRef,omitempty"`
	CustomerRef  *ReferenceType `json:"CustomerRef,omitempty"`
	ProjectRef   *ReferenceType `json:"ProjectRef,omitempty"`
	HourlyRate   json.Number    `json:"HourlyRate,omitempty"`
	StartTime    *DateTime      `json:"StartTime,omitempty"`
	EndTime      *DateTime      `json:"EndTime,omitempty"`
	Hours        int            `json:"Hours,omitempty"`
	Minutes      int            `json:"Minutes,omitempty"`
	BreakHours   int            `json:"BreakHours,omitempty"`
	BreakMinutes int            `json:"BreakMinutes,omitempty"`

	// Optional
	ClassRef                *ReferenceType     `json:"ClassRef,omitempty"`
	ItemRef                 *ReferenceType     `json:"ItemRef,omitempty"`
	PayrollItemRef          *ReferenceType     `json:"PayrollItemRef,omitempty"`
	DepartmentRef           *ReferenceType     `json:"DepartmentRef,omitempty"`
	BillableStatus          BillableStatusEnum `json:"BillableStatus,omitempty"`
	CostRate                json.Number        `json:"CostRate,omitempty"`
	Taxable                 *bool              `json:"Taxable,omitempty"`
	Description             string             `json:"Description,omitempty"`
	TransactionLocationType string             `json:"TransactionLocationType,omitempty"` // France locale only.
}

func (TimeActivity) EntityName() EntityName {
	return EntityTimeActivity
}

type timeActivityEnvelope struct {
	TimeActivity TimeActivity `json:"TimeActivity"`
}

type timeActivityQueryEnvelope struct {
	QueryResponse QueryResponse `json:"QueryResponse"`
}

func (s *TimeActivityService) Get(ctx context.Context, id string, opts ...RequestOption) (*TimeActivity, error) {
	var out timeActivityEnvelope
	if err := s.doJSON(ctx, http.MethodGet, "timeactivity/"+id, nil, nil, &out, opts...); err != nil {
		return nil, err
	}
	return &out.TimeActivity, nil
}

func (s *TimeActivityService) Query(ctx context.Context, query string, opts ...QueryOption) ([]TimeActivity, error) {
	query = applyQueryOptions(query, opts...)

	var out timeActivityQueryEnvelope
	if err := s.doJSON(ctx, http.MethodGet, "query", url.Values{"query": []string{query}}, nil, &out); err != nil {
		return nil, err
	}

	raw, err := out.QueryResponse.Entities(EntityTimeActivity)
	if err != nil {
		return nil, err
	}

	return decodeEntitySlice[TimeActivity](raw)
}

func (s *TimeActivityService) List(ctx context.Context, opts ...QueryOption) ([]TimeActivity, error) {
	return s.Query(ctx, "SELECT * FROM TimeActivity", opts...)
}

func (s *TimeActivityService) ListPage(ctx context.Context, page, pageSize int, opts ...QueryOption) ([]TimeActivity, error) {
	pageOpts := append([]QueryOption{WithPage(page, pageSize)}, opts...)
	return s.List(ctx, pageOpts...)
}

func (s *TimeActivityService) Create(ctx context.Context, ta TimeActivity, opts ...RequestOption) (*TimeActivity, error) {
	var out timeActivityEnvelope
	if err := s.doJSON(ctx, http.MethodPost, "timeactivity", nil, ta, &out, opts...); err != nil {
		return nil, err
	}
	return &out.TimeActivity, nil
}

func (s *TimeActivityService) Update(ctx context.Context, ta TimeActivity, opts ...RequestOption) (*TimeActivity, error) {
	var out timeActivityEnvelope
	if err := s.doJSON(ctx, http.MethodPost, "timeactivity", nil, ta, &out, opts...); err != nil {
		return nil, err
	}
	return &out.TimeActivity, nil
}

func (s *TimeActivityService) UpdateSparse(ctx context.Context, ta TimeActivity, opts ...RequestOption) (*TimeActivity, error) {
	payload := struct {
		TimeActivity
		Sparse bool `json:"sparse"`
	}{ta, true}
	var out timeActivityEnvelope
	if err := s.doJSON(ctx, http.MethodPost, "timeactivity", nil, payload, &out, opts...); err != nil {
		return nil, err
	}
	return &out.TimeActivity, nil
}

func (s *TimeActivityService) Delete(ctx context.Context, req DeleteRequest, opts ...RequestOption) (*TimeActivity, error) {
	payload := struct {
		IDValue
		SyncTokenValue
	}{req.IDValue, req.SyncTokenValue}

	var out timeActivityEnvelope
	if err := s.doJSON(ctx, http.MethodPost, "timeactivity", url.Values{"operation": []string{"delete"}}, payload, &out, opts...); err != nil {
		return nil, err
	}
	return &out.TimeActivity, nil
}

func (s *TimeActivityService) ExtractBatch(resp *BatchResponse) ([]TimeActivity, error) {
	return ExtractBatchEntities[TimeActivity](resp, EntityTimeActivity)
}

func (s *TimeActivityService) ExtractBatchItems(items []BatchItemResponse) ([]TimeActivity, error) {
	return ExtractBatchItemEntities[TimeActivity](items, EntityTimeActivity)
}

func (s *TimeActivityService) BatchCreateItem(id string, ta TimeActivity) BatchItemRequest {
	return NewBatchEntity(id, BatchOperationCreate, EntityTimeActivity, ta)
}

func (s *TimeActivityService) BatchUpdateItem(id string, ta TimeActivity) BatchItemRequest {
	return NewBatchEntity(id, BatchOperationUpdate, EntityTimeActivity, ta)
}

func (s *TimeActivityService) BatchDeleteItem(id string, req DeleteRequest) BatchItemRequest {
	payload := struct {
		IDValue
		SyncTokenValue
	}{req.IDValue, req.SyncTokenValue}
	return NewBatchEntity(id, BatchOperationDelete, EntityTimeActivity, payload)
}

func (s *TimeActivityService) BatchQueryItem(id, query string, opts ...QueryOption) BatchItemRequest {
	return NewBatchQuery(id, query, opts...)
}
