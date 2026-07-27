package quickbooks

import (
	"context"
	"net/http"
	"net/url"
	"time"
)

// CompanyInfoService provides access to the CompanyInfo-related endpoints of
// the QuickBooks Online API.
type CompanyInfoService struct {
	companyService
}

func (c *CompanyClient) CompanyInfo() *CompanyInfoService {
	return &CompanyInfoService{companyService: companyService{company: c}}
}

// CompanyInfo represents the QuickBooks Online CompanyInfo entity, which holds
// company-level settings and metadata.
//
// CompanyInfo does not support CREATE or DELETE. Use Get to read, and Update
// or UpdateSparse to modify.
//
// Get uses the realmID bound to the CompanyClient — no entity ID is required.
//
// CompanyName and CompanyAddr are required on full Update; they are optional
// on read and sparse update.
//
// Fields marked "Read-only" are set by the server and ignored if sent in a
// write request.
type CompanyInfo struct {
	IDValue
	SyncTokenValue
	CDCFields

	MetaData *ModificationMetaData `json:"MetaData,omitempty"`

	// Required for update
	CompanyName string           `json:"CompanyName,omitempty"`
	CompanyAddr *PhysicalAddress `json:"CompanyAddr,omitempty"`

	// Optional
	LegalName                 string           `json:"LegalName,omitempty"`
	LegalAddr                 *PhysicalAddress `json:"LegalAddr,omitempty"`
	CustomerCommunicationAddr *PhysicalAddress `json:"CustomerCommunicationAddr,omitempty"`
	SupportedLanguages        string           `json:"SupportedLanguages,omitempty"`
	Country                   string           `json:"Country,omitempty"`
	FiscalYearStartMonth      MonthEnum        `json:"FiscalYearStartMonth,omitempty"`
	Email                     *EmailAddress    `json:"Email,omitempty"`
	WebAddr                   *WebsiteAddress  `json:"WebAddr,omitempty"`
	PrimaryPhone              *TelephoneNumber `json:"PrimaryPhone,omitempty"`
	EmployerId                string           `json:"EmployerId,omitempty"`
	NameValue                 []NameValue      `json:"NameValue,omitempty"`

	// Read-only: set by the server.
	CompanyStartDate *DateTime `json:"CompanyStartDate,omitempty"`
}

func (CompanyInfo) EntityName() EntityName {
	return EntityCompanyInfo
}

type companyInfoEnvelope struct {
	CompanyInfo CompanyInfo `json:"CompanyInfo"`
}

type companyInfoQueryEnvelope struct {
	QueryResponse QueryResponse `json:"QueryResponse"`
}

// Get retrieves the CompanyInfo for the realm bound to this CompanyClient.
// The realmID is used as the entity ID per the QBO API contract.
func (s *CompanyInfoService) Get(ctx context.Context, opts ...RequestOption) (*CompanyInfo, error) {
	var out companyInfoEnvelope
	if err := s.doJSON(ctx, http.MethodGet, "companyinfo/"+s.company.RealmID(), nil, nil, &out, opts...); err != nil {
		return nil, err
	}
	return &out.CompanyInfo, nil
}

func (s *CompanyInfoService) Query(ctx context.Context, query string, opts ...QueryOption) ([]CompanyInfo, error) {
	query = applyQueryOptions(query, opts...)

	var out companyInfoQueryEnvelope
	if err := s.doJSON(ctx, http.MethodGet, "query", url.Values{"query": []string{query}}, nil, &out); err != nil {
		return nil, err
	}

	raw, err := out.QueryResponse.Entities(EntityCompanyInfo)
	if err != nil {
		return nil, err
	}

	return decodeEntitySlice[CompanyInfo](raw)
}

func (s *CompanyInfoService) List(ctx context.Context, opts ...QueryOption) ([]CompanyInfo, error) {
	return s.Query(ctx, "SELECT * FROM CompanyInfo", opts...)
}

func (s *CompanyInfoService) Update(ctx context.Context, ci CompanyInfo, opts ...RequestOption) (*CompanyInfo, error) {
	var out companyInfoEnvelope
	if err := s.doJSON(ctx, http.MethodPost, "companyinfo", nil, ci, &out, opts...); err != nil {
		return nil, err
	}
	return &out.CompanyInfo, nil
}

func (s *CompanyInfoService) UpdateSparse(ctx context.Context, ci CompanyInfo, opts ...RequestOption) (*CompanyInfo, error) {
	payload := struct {
		CompanyInfo
		Sparse bool `json:"sparse"`
	}{ci, true}
	var out companyInfoEnvelope
	if err := s.doJSON(ctx, http.MethodPost, "companyinfo", nil, payload, &out, opts...); err != nil {
		return nil, err
	}
	return &out.CompanyInfo, nil
}

func (s *CompanyInfoService) ChangeDataCapture(ctx context.Context, changedSince time.Time, opts ...RequestOption) ([]CompanyInfo, error) {
	resp, err := s.company.ChangeDataCapture(ctx, changedSince, []EntityName{EntityCompanyInfo}, opts...)
	if err != nil {
		return nil, err
	}
	return s.ExtractCDC(resp)
}

func (s *CompanyInfoService) ExtractCDC(resp *CDCResponse) ([]CompanyInfo, error) {
	return ExtractCDCEntities[CompanyInfo](resp, EntityCompanyInfo)
}

func (s *CompanyInfoService) ExtractBatch(resp *BatchResponse) ([]CompanyInfo, error) {
	return ExtractBatchEntities[CompanyInfo](resp, EntityCompanyInfo)
}

func (s *CompanyInfoService) ExtractBatchItems(items []BatchItemResponse) ([]CompanyInfo, error) {
	return ExtractBatchItemEntities[CompanyInfo](items, EntityCompanyInfo)
}

func (s *CompanyInfoService) BatchQueryItem(id, query string, opts ...QueryOption) BatchItemRequest {
	return NewBatchQuery(id, query, opts...)
}
