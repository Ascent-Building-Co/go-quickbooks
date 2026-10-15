package quickbooks

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"time"
)

// TermService provides access to the Term-related endpoints of the QuickBooks
// Online API.
type TermService struct {
	companyService
}

func (c *CompanyClient) Terms() *TermService {
	return &TermService{companyService: companyService{company: c}}
}

// Term represents a QuickBooks Online Term entity, which defines the payment
// terms offered to customers (e.g. "Net 30", "2% 10 Net 30").
//
// Two types of terms are supported, controlled by the Type field:
//   - STANDARD: payment is due within a fixed number of days (DueDays).
//   - DATE_DRIVEN: payment is due on a specific day of the month (DayOfMonthDue).
//
// DueDays and DayOfMonthDue are conditionally required: exactly one must be
// provided, depending on the Type.
//
// Fields marked "Read-only" are set by the server and ignored if sent in a
// write request.
type Term struct {
	IDValue
	SyncTokenValue
	CDCFields

	MetaDataValue

	// Required
	Name string `json:"Name,omitempty"`

	// Conditionally required
	DueDays       int `json:"DueDays,omitempty"`
	DayOfMonthDue int `json:"DayOfMonthDue,omitempty"`

	// Optional
	DiscountPercent    json.Number  `json:"DiscountPercent,omitempty"`
	DiscountDays       int          `json:"DiscountDays,omitempty"`
	DiscountDayOfMonth int          `json:"DiscountDayOfMonth,omitempty"`
	DueNextMonthDays   int          `json:"DueNextMonthDays,omitempty"`
	Active             *bool        `json:"Active,omitempty"`
	Type               TermTypeEnum `json:"Type,omitempty"`
}

func (Term) EntityName() EntityName {
	return EntityTerm
}

type termEnvelope struct {
	Term Term `json:"Term"`
}

type termQueryEnvelope struct {
	QueryResponse QueryResponse `json:"QueryResponse"`
}

func (s *TermService) Get(ctx context.Context, id string, opts ...RequestOption) (*Term, error) {
	var out termEnvelope
	if err := s.doJSON(ctx, http.MethodGet, "term/"+id, nil, nil, &out, opts...); err != nil {
		return nil, err
	}
	return &out.Term, nil
}

func (s *TermService) Query(ctx context.Context, query string, opts ...QueryOption) ([]Term, error) {
	query = applyQueryOptions(query, opts...)

	var out termQueryEnvelope
	if err := s.doJSON(ctx, http.MethodGet, "query", url.Values{"query": []string{query}}, nil, &out); err != nil {
		return nil, err
	}

	raw, err := out.QueryResponse.Entities(EntityTerm)
	if err != nil {
		return nil, err
	}

	return decodeEntitySlice[Term](raw)
}

func (s *TermService) List(ctx context.Context, opts ...QueryOption) ([]Term, error) {
	return s.Query(ctx, "SELECT * FROM Term", opts...)
}

func (s *TermService) ListPage(ctx context.Context, page, pageSize int, opts ...QueryOption) ([]Term, error) {
	pageOpts := append([]QueryOption{WithPage(page, pageSize)}, opts...)
	return s.List(ctx, pageOpts...)
}

func (s *TermService) Create(ctx context.Context, term Term, opts ...RequestOption) (*Term, error) {
	var out termEnvelope
	if err := s.doJSON(ctx, http.MethodPost, "term", nil, term, &out, opts...); err != nil {
		return nil, err
	}
	return &out.Term, nil
}

func (s *TermService) Update(ctx context.Context, term Term, opts ...RequestOption) (*Term, error) {
	var out termEnvelope
	if err := s.doJSON(ctx, http.MethodPost, "term", nil, term, &out, opts...); err != nil {
		return nil, err
	}
	return &out.Term, nil
}

func (s *TermService) UpdateSparse(ctx context.Context, term Term, opts ...RequestOption) (*Term, error) {
	payload := struct {
		Term
		Sparse bool `json:"sparse"`
	}{term, true}
	var out termEnvelope
	if err := s.doJSON(ctx, http.MethodPost, "term", nil, payload, &out, opts...); err != nil {
		return nil, err
	}
	return &out.Term, nil
}

func (s *TermService) ChangeDataCapture(ctx context.Context, changedSince time.Time, opts ...RequestOption) ([]Term, error) {
	resp, err := s.company.ChangeDataCapture(ctx, changedSince, []EntityName{EntityTerm}, opts...)
	if err != nil {
		return nil, err
	}
	return s.ExtractCDC(resp)
}

func (s *TermService) ExtractCDC(resp *CDCResponse) ([]Term, error) {
	return ExtractCDCEntities[Term](resp, EntityTerm)
}

func (s *TermService) ExtractBatch(resp *BatchResponse) ([]Term, error) {
	return ExtractBatchEntities[Term](resp, EntityTerm)
}

func (s *TermService) ExtractBatchItems(items []BatchItemResponse) ([]Term, error) {
	return ExtractBatchItemEntities[Term](items, EntityTerm)
}

func (s *TermService) BatchCreateItem(id string, term Term) BatchItemRequest {
	return NewBatchEntity(id, BatchOperationCreate, EntityTerm, term)
}

func (s *TermService) BatchUpdateItem(id string, term Term) BatchItemRequest {
	return NewBatchEntity(id, BatchOperationUpdate, EntityTerm, term)
}

func (s *TermService) BatchQueryItem(id, query string, opts ...QueryOption) BatchItemRequest {
	return NewBatchQuery(id, query, opts...)
}
