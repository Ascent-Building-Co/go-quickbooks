package quickbooks

import (
	"context"
	"net/http"
	"net/url"
	"time"
)

// PreferencesService provides access to the Preferences-related endpoints of
// the QuickBooks Online API.
type PreferencesService struct {
	companyService
}

func (c *CompanyClient) Preferences() *PreferencesService {
	return &PreferencesService{companyService: companyService{company: c}}
}

// EmailMessagesPrefs holds the default email subject and message body
// templates for each transaction type that can be emailed to customers.
type EmailMessagesPrefs struct {
	InvoiceMessage      *EmailMessageType `json:"InvoiceMessage,omitempty"`
	EstimateMessage     *EmailMessageType `json:"EstimateMessage,omitempty"`
	SalesReceiptMessage *EmailMessageType `json:"SalesReceiptMessage,omitempty"`
	StatementMessage    *EmailMessageType `json:"StatementMessage,omitempty"`
}

// ProductAndServicesPrefs holds settings that control product and service
// item behaviour across transactions.
type ProductAndServicesPrefs struct {
	QuantityWithPriceAndRate *bool `json:"QuantityWithPriceAndRate,omitempty"`
	ForPurchase              *bool `json:"ForPurchase,omitempty"`
	QuantityOnHand           *bool `json:"QuantityOnHand,omitempty"`
	ForSales                 *bool `json:"ForSales,omitempty"`
}

// ReportPrefs holds company-wide reporting preferences.
type ReportPrefs struct {
	ReportBasis                ReportBasisEnum `json:"ReportBasis,omitempty"`
	CalcAgingReportFromTxnDate *bool           `json:"CalcAgingReportFromTxnDate,omitempty"`
}

// AccountingInfoPrefs holds accounting and fiscal year configuration.
type AccountingInfoPrefs struct {
	FirstMonthOfFiscalYear  MonthEnum `json:"FirstMonthOfFiscalYear,omitempty"`
	TaxYearMonth            MonthEnum `json:"TaxYearMonth,omitempty"`
	UseAccountNumbers       *bool     `json:"UseAccountNumbers,omitempty"`
	ClassTrackingPerTxn     *bool     `json:"ClassTrackingPerTxn,omitempty"`
	ClassTrackingPerTxnLine *bool     `json:"ClassTrackingPerTxnLine,omitempty"`
	TrackDepartments        *bool     `json:"TrackDepartments,omitempty"`
	TaxForm                 string    `json:"TaxForm,omitempty"`
	CustomerTerminology     string    `json:"CustomerTerminology,omitempty"`
	DepartmentTerminology   string    `json:"DepartmentTerminology,omitempty"`
	BookCloseDate           *Date     `json:"BookCloseDate,omitempty"`
}

// SalesFormsPrefs holds settings that control the behaviour and appearance
// of sales transactions such as invoices, estimates, and receipts.
type SalesFormsPrefs struct {
	CustomTxnNumbers           *bool                         `json:"CustomTxnNumbers,omitempty"`
	AllowShipping              *bool                         `json:"AllowShipping,omitempty"`
	AllowServiceDate           *bool                         `json:"AllowServiceDate,omitempty"`
	AllowEstimates             *bool                         `json:"AllowEstimates,omitempty"`
	AllowDiscount              *bool                         `json:"AllowDiscount,omitempty"`
	AllowDeposit               *bool                         `json:"AllowDeposit,omitempty"`
	AutoApplyPayments          *bool                         `json:"AutoApplyPayments,omitempty"`
	AutoApplyCredit            *bool                         `json:"AutoApplyCredit,omitempty"`
	UsingPriceLevels           *bool                         `json:"UsingPriceLevels,omitempty"`
	EmailCopyToCompany         *bool                         `json:"EmailCopyToCompany,omitempty"`
	IPNSupportEnabled          *bool                         `json:"IPNSupportEnabled,omitempty"`
	ETransactionPaymentEnabled *bool                         `json:"ETransactionPaymentEnabled,omitempty"`
	ETransactionAttachPDF      *bool                         `json:"ETransactionAttachPDF,omitempty"`
	ETransactionEnabledStatus  ETransactionEnabledStatusEnum `json:"ETransactionEnabledStatus,omitempty"`
	DefaultCustomerMessage     string                        `json:"DefaultCustomerMessage,omitempty"`
	DefaultTerms               *ReferenceType                `json:"DefaultTerms,omitempty"`
	DefaultDiscountAccount     string                        `json:"DefaultDiscountAccount,omitempty"`
	CustomField                []CustomField                 `json:"CustomField,omitempty"`
}

// VendorAndPurchasesPrefs holds settings that control vendor and purchase
// transaction behaviour.
type VendorAndPurchasesPrefs struct {
	BillableExpenseTracking *bool         `json:"BillableExpenseTracking,omitempty"`
	TrackingByCustomer      *bool         `json:"TrackingByCustomer,omitempty"`
	POCustomField           []CustomField `json:"POCustomField,omitempty"`
}

// TaxPrefs holds tax-related company preferences.
type TaxPrefs struct {
	TaxGroupCodeRef *ReferenceType `json:"TaxGroupCodeRef,omitempty"`
	UsingSalesTax   *bool          `json:"UsingSalesTax,omitempty"`
}

// OtherPrefs holds miscellaneous preferences as arbitrary name-value pairs.
type OtherPrefs struct {
	NameValue []NameValue `json:"NameValue,omitempty"`
}

// TimeTrackingPrefs holds settings that control time tracking behaviour.
type TimeTrackingPrefs struct {
	WorkWeekStartDate       WorkWeekStartDateEnum `json:"WorkWeekStartDate,omitempty"`
	MarkTimeEntriesBillable *bool                 `json:"MarkTimeEntriesBillable,omitempty"`
	ShowBillRateToAll       *bool                 `json:"ShowBillRateToAll,omitempty"`
	UseServices             *bool                 `json:"UseServices,omitempty"`
	BillCustomers           *bool                 `json:"BillCustomers,omitempty"`
}

// CurrencyPrefs holds multi-currency settings for the company.
type CurrencyPrefs struct {
	HomeCurrency         *ReferenceType `json:"HomeCurrency,omitempty"`
	MultiCurrencyEnabled *bool          `json:"MultiCurrencyEnabled,omitempty"`
}

// Preferences represents the QuickBooks Online Preferences entity, which holds
// company-wide settings across accounting, sales, purchasing, and reporting.
//
// Preferences does not support CREATE or DELETE. Use Get to read, and Update
// or UpdateSparse to modify.
//
// Get uses the fixed path "preferences" bound to the CompanyClient — no entity
// ID is required.
//
// All preference sub-objects are optional; the server returns defaults for any
// that are not explicitly set.
//
// Fields marked "Read-only" are set by the server and ignored if sent in a
// write request.
type Preferences struct {
	IDValue
	SyncTokenValue
	CDCFields

	MetaData *ModificationMetaData `json:"MetaData,omitempty"`

	// Optional — server-set defaults are returned for all sub-objects.
	EmailMessagesPrefs      *EmailMessagesPrefs      `json:"EmailMessagesPrefs,omitempty"`
	ProductAndServicesPrefs *ProductAndServicesPrefs `json:"ProductAndServicesPrefs,omitempty"`
	ReportPrefs             *ReportPrefs             `json:"ReportPrefs,omitempty"`
	AccountingInfoPrefs     *AccountingInfoPrefs     `json:"AccountingInfoPrefs,omitempty"`
	SalesFormsPrefs         *SalesFormsPrefs         `json:"SalesFormsPrefs,omitempty"`
	VendorAndPurchasesPrefs *VendorAndPurchasesPrefs `json:"VendorAndPurchasesPrefs,omitempty"`
	TaxPrefs                *TaxPrefs                `json:"TaxPrefs,omitempty"`
	OtherPrefs              *OtherPrefs              `json:"OtherPrefs,omitempty"`
	TimeTrackingPrefs       *TimeTrackingPrefs       `json:"TimeTrackingPrefs,omitempty"`
	CurrencyPrefs           *CurrencyPrefs           `json:"CurrencyPrefs,omitempty"`
}

func (Preferences) EntityName() EntityName {
	return EntityPreferences
}

type preferencesEnvelope struct {
	Preferences Preferences `json:"Preferences"`
}

type preferencesQueryEnvelope struct {
	QueryResponse QueryResponse `json:"QueryResponse"`
}

// Get retrieves the Preferences for the realm bound to this CompanyClient.
func (s *PreferencesService) Get(ctx context.Context, opts ...RequestOption) (*Preferences, error) {
	var out preferencesEnvelope
	if err := s.doJSON(ctx, http.MethodGet, "preferences", nil, nil, &out, opts...); err != nil {
		return nil, err
	}
	return &out.Preferences, nil
}

func (s *PreferencesService) Query(ctx context.Context, query string, opts ...QueryOption) ([]Preferences, error) {
	query = applyQueryOptions(query, opts...)

	var out preferencesQueryEnvelope
	if err := s.doJSON(ctx, http.MethodGet, "query", url.Values{"query": []string{query}}, nil, &out); err != nil {
		return nil, err
	}

	raw, err := out.QueryResponse.Entities(EntityPreferences)
	if err != nil {
		return nil, err
	}

	return decodeEntitySlice[Preferences](raw)
}

func (s *PreferencesService) List(ctx context.Context, opts ...QueryOption) ([]Preferences, error) {
	return s.Query(ctx, "SELECT * FROM Preferences", opts...)
}

func (s *PreferencesService) Update(ctx context.Context, prefs Preferences, opts ...RequestOption) (*Preferences, error) {
	var out preferencesEnvelope
	if err := s.doJSON(ctx, http.MethodPost, "preferences", nil, prefs, &out, opts...); err != nil {
		return nil, err
	}
	return &out.Preferences, nil
}

func (s *PreferencesService) UpdateSparse(ctx context.Context, prefs Preferences, opts ...RequestOption) (*Preferences, error) {
	payload := struct {
		Preferences
		Sparse bool `json:"sparse"`
	}{prefs, true}
	var out preferencesEnvelope
	if err := s.doJSON(ctx, http.MethodPost, "preferences", nil, payload, &out, opts...); err != nil {
		return nil, err
	}
	return &out.Preferences, nil
}

func (s *PreferencesService) ChangeDataCapture(ctx context.Context, changedSince time.Time, opts ...RequestOption) ([]Preferences, error) {
	resp, err := s.company.ChangeDataCapture(ctx, changedSince, []EntityName{EntityPreferences}, opts...)
	if err != nil {
		return nil, err
	}
	return s.ExtractCDC(resp)
}

func (s *PreferencesService) ExtractCDC(resp *CDCResponse) ([]Preferences, error) {
	return ExtractCDCEntities[Preferences](resp, EntityPreferences)
}

func (s *PreferencesService) ExtractBatch(resp *BatchResponse) ([]Preferences, error) {
	return ExtractBatchEntities[Preferences](resp, EntityPreferences)
}

func (s *PreferencesService) ExtractBatchItems(items []BatchItemResponse) ([]Preferences, error) {
	return ExtractBatchItemEntities[Preferences](items, EntityPreferences)
}

func (s *PreferencesService) BatchQueryItem(id, query string, opts ...QueryOption) BatchItemRequest {
	return NewBatchQuery(id, query, opts...)
}
