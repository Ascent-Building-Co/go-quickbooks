package quickbooks

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"time"
)

type CustomerService struct {
	companyService
}

func (c *CompanyClient) Customers() *CustomerService {
	return &CustomerService{companyService: companyService{company: c}}
}

// Customer represents a QuickBooks Online Customer entity. Customers can be
// individuals, companies, or sub-customers (jobs) associated with a parent.
//
// At least one of the conditionally required name fields must be provided to
// form a unique DisplayName.
//
// Fields marked "Read-only" are set by the server and ignored if sent in a
// write request.
type Customer struct {
	IDValue
	SyncTokenValue
	CDCFields

	MetaDataValue

	// Conditionally required — at least one combination must produce a unique DisplayName.
	DisplayName string `json:"DisplayName,omitempty"`
	Title       string `json:"Title,omitempty"`
	GivenName   string `json:"GivenName,omitempty"`
	MiddleName  string `json:"MiddleName,omitempty"`
	Suffix      string `json:"Suffix,omitempty"`
	FamilyName  string `json:"FamilyName,omitempty"`

	// Optional
	CompanyName             string                 `json:"CompanyName,omitempty"`
	PrintOnCheckName        string                 `json:"PrintOnCheckName,omitempty"`
	Active                  *bool                  `json:"Active,omitempty"`
	PrimaryPhone            *TelephoneNumber       `json:"PrimaryPhone,omitempty"`
	AlternatePhone          *TelephoneNumber       `json:"AlternatePhone,omitempty"`
	Mobile                  *TelephoneNumber       `json:"Mobile,omitempty"`
	Fax                     *TelephoneNumber       `json:"Fax,omitempty"`
	PrimaryEmailAddr        *EmailAddress          `json:"PrimaryEmailAddr,omitempty"`
	WebAddr                 *WebsiteAddress        `json:"WebAddr,omitempty"`
	BillAddr                *PhysicalAddress       `json:"BillAddr,omitempty"`
	ShipAddr                *PhysicalAddress       `json:"ShipAddr,omitempty"`
	Notes                   string                 `json:"Notes,omitempty"`
	Taxable                 *bool                  `json:"Taxable,omitempty"`
	Balance                 json.Number            `json:"Balance,omitempty"`
	BalanceWithJobs         json.Number            `json:"BalanceWithJobs,omitempty"`
	OpenBalanceDate         *Date                  `json:"OpenBalanceDate,omitempty"`
	CurrencyRef             *ReferenceType         `json:"CurrencyRef,omitempty"`
	PreferredDeliveryMethod string                 `json:"PreferredDeliveryMethod,omitempty"`
	PaymentMethodRef        *ReferenceType         `json:"PaymentMethodRef,omitempty"`
	SalesTermRef            *ReferenceType         `json:"SalesTermRef,omitempty"`
	ARAccountRef            *ReferenceType         `json:"ARAccountRef,omitempty"` // France locale only.
	DefaultTaxCodeRef       *ReferenceType         `json:"DefaultTaxCodeRef,omitempty"`
	CustomerTypeRef         *ReferenceType         `json:"CustomerTypeRef,omitempty"`
	ParentRef               *ReferenceType         `json:"ParentRef,omitempty"`
	BillWithParent          *bool                  `json:"BillWithParent,omitempty"`
	Job                     *bool                  `json:"Job,omitempty"`
	Source                  string                 `json:"Source,omitempty"`
	ResaleNum               string                 `json:"ResaleNum,omitempty"`
	PrimaryTaxIdentifier    string                 `json:"PrimaryTaxIdentifier,omitempty"`
	SecondaryTaxIdentifier  string                 `json:"SecondaryTaxIdentifier,omitempty"`
	BusinessNumber          string                 `json:"BusinessNumber,omitempty"`      // India locale only.
	GSTIN                   string                 `json:"GSTIN,omitempty"`               // India locale only.
	GSTRegistrationType     string                 `json:"GSTRegistrationType,omitempty"` // India locale only.
	TaxExemptionReasonId    TaxExemptionReasonEnum `json:"TaxExemptionReasonId,omitempty"`

	// Read-only: set by the server.
	FullyQualifiedName string `json:"FullyQualifiedName,omitempty"`
	Level              int    `json:"Level,omitempty"`
	IsProject          *bool  `json:"IsProject,omitempty"`
}

func (Customer) EntityName() EntityName {
	return EntityCustomer
}

type customerEnvelope struct {
	Customer Customer `json:"Customer"`
}

type customerQueryEnvelope struct {
	QueryResponse QueryResponse `json:"QueryResponse"`
}

func (s *CustomerService) Get(ctx context.Context, id string, opts ...RequestOption) (*Customer, error) {
	var out customerEnvelope
	if err := s.doJSON(ctx, http.MethodGet, "customer/"+id, nil, nil, &out, opts...); err != nil {
		return nil, err
	}
	return &out.Customer, nil
}

func (s *CustomerService) Query(ctx context.Context, query string, opts ...QueryOption) ([]Customer, error) {
	query = applyQueryOptions(query, opts...)

	var out customerQueryEnvelope
	if err := s.doJSON(ctx, http.MethodGet, "query", url.Values{"query": []string{query}}, nil, &out); err != nil {
		return nil, err
	}

	raw, err := out.QueryResponse.Entities(EntityCustomer)
	if err != nil {
		return nil, err
	}

	return decodeEntitySlice[Customer](raw)
}

func (s *CustomerService) List(ctx context.Context, opts ...QueryOption) ([]Customer, error) {
	return s.Query(ctx, "SELECT * FROM Customer", opts...)
}

func (s *CustomerService) ListPage(ctx context.Context, page, pageSize int, opts ...QueryOption) ([]Customer, error) {
	pageOpts := append([]QueryOption{WithPage(page, pageSize)}, opts...)
	return s.List(ctx, pageOpts...)
}

func (s *CustomerService) Create(ctx context.Context, customer Customer, opts ...RequestOption) (*Customer, error) {
	var out customerEnvelope
	if err := s.doJSON(ctx, http.MethodPost, "customer", nil, customer, &out, opts...); err != nil {
		return nil, err
	}
	return &out.Customer, nil
}

func (s *CustomerService) Update(ctx context.Context, customer Customer, opts ...RequestOption) (*Customer, error) {
	var out customerEnvelope
	if err := s.doJSON(ctx, http.MethodPost, "customer", nil, customer, &out, opts...); err != nil {
		return nil, err
	}
	return &out.Customer, nil
}

func (s *CustomerService) UpdateSparse(ctx context.Context, customer Customer, opts ...RequestOption) (*Customer, error) {
	payload := struct {
		Customer
		Sparse bool `json:"sparse"`
	}{customer, true}
	var out customerEnvelope
	if err := s.doJSON(ctx, http.MethodPost, "customer", nil, payload, &out, opts...); err != nil {
		return nil, err
	}
	return &out.Customer, nil
}

func (s *CustomerService) ChangeDataCapture(ctx context.Context, changedSince time.Time, opts ...RequestOption) ([]Customer, error) {
	resp, err := s.company.ChangeDataCapture(ctx, changedSince, []EntityName{EntityCustomer}, opts...)
	if err != nil {
		return nil, err
	}
	return s.ExtractCDC(resp)
}

func (s *CustomerService) ExtractCDC(resp *CDCResponse) ([]Customer, error) {
	return ExtractCDCEntities[Customer](resp, EntityCustomer)
}

func (s *CustomerService) ExtractBatch(resp *BatchResponse) ([]Customer, error) {
	return ExtractBatchEntities[Customer](resp, EntityCustomer)
}

func (s *CustomerService) ExtractBatchItems(items []BatchItemResponse) ([]Customer, error) {
	return ExtractBatchItemEntities[Customer](items, EntityCustomer)
}

func (s *CustomerService) BatchCreateItem(id string, customer Customer) BatchItemRequest {
	return NewBatchEntity(id, BatchOperationCreate, EntityCustomer, customer)
}

func (s *CustomerService) BatchUpdateItem(id string, customer Customer) BatchItemRequest {
	return NewBatchEntity(id, BatchOperationUpdate, EntityCustomer, customer)
}

func (s *CustomerService) BatchQueryItem(id, query string, opts ...QueryOption) BatchItemRequest {
	return NewBatchQuery(id, query, opts...)
}
