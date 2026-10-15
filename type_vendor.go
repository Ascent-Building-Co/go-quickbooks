package quickbooks

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"time"
)

// VendorPaymentBankDetail holds the bank account details for direct payment
// to a vendor. All fields are required when VendorPaymentBankDetail is present
// in a request; BankAccountNumber is masked in responses.
type VendorPaymentBankDetail struct {
	BankAccountName      string `json:"BankAccountName,omitempty"`
	BankBranchIdentifier string `json:"BankBranchIdentifier,omitempty"`
	BankAccountNumber    string `json:"BankAccountNumber,omitempty"`
	StatementText        string `json:"StatementText,omitempty"`
}

// VendorService provides access to the Vendor-related endpoints of the
// QuickBooks Online API.
type VendorService struct {
	companyService
}

func (c *CompanyClient) Vendors() *VendorService {
	return &VendorService{companyService: companyService{company: c}}
}

// Vendor represents a QuickBooks Online Vendor entity — a person or
// organisation from whom the business purchases goods or services.
//
// At least one of the conditionally required name fields must be provided to
// form a unique DisplayName. DisplayName must be unique across all Vendor,
// Customer, and Employee objects.
//
// Fields marked "Read-only" are set by the server and ignored if sent in a
// write request.
type Vendor struct {
	IDValue
	SyncTokenValue
	CDCFields

	MetaDataValue

	// Conditionally required — DisplayName OR at least one name component required.
	DisplayName string `json:"DisplayName,omitempty"`
	Title       string `json:"Title,omitempty"`
	GivenName   string `json:"GivenName,omitempty"`
	MiddleName  string `json:"MiddleName,omitempty"`
	Suffix      string `json:"Suffix,omitempty"`
	FamilyName  string `json:"FamilyName,omitempty"`

	// Optional
	CompanyName             string                   `json:"CompanyName,omitempty"`
	PrintOnCheckName        string                   `json:"PrintOnCheckName,omitempty"`
	Active                  *bool                    `json:"Active,omitempty"`
	PrimaryPhone            *TelephoneNumber         `json:"PrimaryPhone,omitempty"`
	AlternatePhone          *TelephoneNumber         `json:"AlternatePhone,omitempty"`
	Mobile                  *TelephoneNumber         `json:"Mobile,omitempty"`
	Fax                     *TelephoneNumber         `json:"Fax,omitempty"`
	PrimaryEmailAddr        *EmailAddress            `json:"PrimaryEmailAddr,omitempty"`
	WebAddr                 *WebsiteAddress          `json:"WebAddr,omitempty"`
	BillAddr                *PhysicalAddress         `json:"BillAddr,omitempty"`
	OtherContactInfo        *ContactInfo             `json:"OtherContactInfo,omitempty"`
	APAccountRef            *ReferenceType           `json:"APAccountRef,omitempty"` // France locale only.
	TermRef                 *ReferenceType           `json:"TermRef,omitempty"`
	CurrencyRef             *ReferenceType           `json:"CurrencyRef,omitempty"`
	AcctNum                 string                   `json:"AcctNum,omitempty"`
	TaxIdentifier           string                   `json:"TaxIdentifier,omitempty"`
	BusinessNumber          string                   `json:"BusinessNumber,omitempty"`      // India locale only.
	GSTIN                   string                   `json:"GSTIN,omitempty"`               // India locale only.
	GSTRegistrationType     string                   `json:"GSTRegistrationType,omitempty"` // India locale only.
	VendorPaymentBankDetail *VendorPaymentBankDetail `json:"VendorPaymentBankDetail,omitempty"`
	Vendor1099              *bool                    `json:"Vendor1099,omitempty"`
	T4AEligible             *bool                    `json:"T4AEligible,omitempty"`
	T5018Eligible           *bool                    `json:"T5018Eligible,omitempty"`
	HasTPAR                 *bool                    `json:"HasTPAR,omitempty"`
	TaxReportingBasis       string                   `json:"TaxReportingBasis,omitempty"` // France locale only.
	CostRate                json.Number              `json:"CostRate,omitempty"`
	BillRate                json.Number              `json:"BillRate,omitempty"`
	Source                  string                   `json:"Source,omitempty"`

	// Read-only: set by the server.
	Balance json.Number `json:"Balance,omitempty"`
}

func (Vendor) EntityName() EntityName {
	return EntityVendor
}

type vendorEnvelope struct {
	Vendor Vendor `json:"Vendor"`
}

type vendorQueryEnvelope struct {
	QueryResponse QueryResponse `json:"QueryResponse"`
}

func (s *VendorService) Get(ctx context.Context, id string, opts ...RequestOption) (*Vendor, error) {
	var out vendorEnvelope
	if err := s.doJSON(ctx, http.MethodGet, "vendor/"+id, nil, nil, &out, opts...); err != nil {
		return nil, err
	}
	return &out.Vendor, nil
}

func (s *VendorService) Query(ctx context.Context, query string, opts ...QueryOption) ([]Vendor, error) {
	query = applyQueryOptions(query, opts...)

	var out vendorQueryEnvelope
	if err := s.doJSON(ctx, http.MethodGet, "query", url.Values{"query": []string{query}}, nil, &out); err != nil {
		return nil, err
	}

	raw, err := out.QueryResponse.Entities(EntityVendor)
	if err != nil {
		return nil, err
	}

	return decodeEntitySlice[Vendor](raw)
}

func (s *VendorService) List(ctx context.Context, opts ...QueryOption) ([]Vendor, error) {
	return s.Query(ctx, "SELECT * FROM Vendor", opts...)
}

func (s *VendorService) ListPage(ctx context.Context, page, pageSize int, opts ...QueryOption) ([]Vendor, error) {
	pageOpts := append([]QueryOption{WithPage(page, pageSize)}, opts...)
	return s.List(ctx, pageOpts...)
}

func (s *VendorService) Create(ctx context.Context, vendor Vendor, opts ...RequestOption) (*Vendor, error) {
	var out vendorEnvelope
	if err := s.doJSON(ctx, http.MethodPost, "vendor", nil, vendor, &out, opts...); err != nil {
		return nil, err
	}
	return &out.Vendor, nil
}

func (s *VendorService) Update(ctx context.Context, vendor Vendor, opts ...RequestOption) (*Vendor, error) {
	var out vendorEnvelope
	if err := s.doJSON(ctx, http.MethodPost, "vendor", nil, vendor, &out, opts...); err != nil {
		return nil, err
	}
	return &out.Vendor, nil
}

func (s *VendorService) UpdateSparse(ctx context.Context, vendor Vendor, opts ...RequestOption) (*Vendor, error) {
	payload := struct {
		Vendor
		Sparse bool `json:"sparse"`
	}{vendor, true}
	var out vendorEnvelope
	if err := s.doJSON(ctx, http.MethodPost, "vendor", nil, payload, &out, opts...); err != nil {
		return nil, err
	}
	return &out.Vendor, nil
}

func (s *VendorService) ChangeDataCapture(ctx context.Context, changedSince time.Time, opts ...RequestOption) ([]Vendor, error) {
	resp, err := s.company.ChangeDataCapture(ctx, changedSince, []EntityName{EntityVendor}, opts...)
	if err != nil {
		return nil, err
	}
	return s.ExtractCDC(resp)
}

func (s *VendorService) ExtractCDC(resp *CDCResponse) ([]Vendor, error) {
	return ExtractCDCEntities[Vendor](resp, EntityVendor)
}

func (s *VendorService) ExtractBatch(resp *BatchResponse) ([]Vendor, error) {
	return ExtractBatchEntities[Vendor](resp, EntityVendor)
}

func (s *VendorService) ExtractBatchItems(items []BatchItemResponse) ([]Vendor, error) {
	return ExtractBatchItemEntities[Vendor](items, EntityVendor)
}

func (s *VendorService) BatchCreateItem(id string, vendor Vendor) BatchItemRequest {
	return NewBatchEntity(id, BatchOperationCreate, EntityVendor, vendor)
}

func (s *VendorService) BatchUpdateItem(id string, vendor Vendor) BatchItemRequest {
	return NewBatchEntity(id, BatchOperationUpdate, EntityVendor, vendor)
}

func (s *VendorService) BatchQueryItem(id, query string, opts ...QueryOption) BatchItemRequest {
	return NewBatchQuery(id, query, opts...)
}
