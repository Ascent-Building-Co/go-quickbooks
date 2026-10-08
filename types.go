package quickbooks

import (
	"context"
	"io"
	"net/url"
	"time"

	"golang.org/x/oauth2"
)

type EntityName string

const (
	EntityAccount             EntityName = "Account"
	EntityAttachable          EntityName = "Attachable"
	EntityBill                EntityName = "Bill"
	EntityBillPayment         EntityName = "BillPayment"
	EntityClass               EntityName = "Class"
	EntityCompanyCurrency     EntityName = "CompanyCurrency"
	EntityCompanyInfo         EntityName = "CompanyInfo"
	EntityCreditMemo          EntityName = "CreditMemo"
	EntityCustomer            EntityName = "Customer"
	EntityCustomerType        EntityName = "CustomerType"
	EntityDepartment          EntityName = "Department"
	EntityDeposit             EntityName = "Deposit"
	EntityEmployee            EntityName = "Employee"
	EntityEstimate            EntityName = "Estimate"
	EntityExchangeRate        EntityName = "ExchangeRate"
	EntityInventoryAdjustment EntityName = "InventoryAdjustment"
	EntityInvoice             EntityName = "Invoice"
	EntityItem                EntityName = "Item"
	EntityJournalEntry        EntityName = "JournalEntry"
	EntityPayment             EntityName = "Payment"
	EntityPaymentMethod       EntityName = "PaymentMethod"
	EntityPreferences         EntityName = "Preferences"
	EntityPurchase            EntityName = "Purchase"
	EntityPurchaseOrder       EntityName = "PurchaseOrder"
	EntityReimburseCharge     EntityName = "ReimburseCharge"
	EntityTaxAgency           EntityName = "TaxAgency"
	EntityTaxCode             EntityName = "TaxCode"
	EntityTaxRate             EntityName = "TaxRate"
	EntityTerm                EntityName = "Term"
	EntityTimeActivity        EntityName = "TimeActivity"
	EntityTransfer            EntityName = "Transfer"
	EntityVendor              EntityName = "Vendor"
	EntityVendorCredit        EntityName = "VendorCredit"
)

func EntityNames() []EntityName {
	return []EntityName{
		EntityAccount,
		EntityAttachable,
		EntityBill,
		EntityBillPayment,
		EntityClass,
		EntityCompanyCurrency,
		EntityCompanyInfo,
		EntityCreditMemo,
		EntityCustomer,
		EntityCustomerType,
		EntityDepartment,
		EntityDeposit,
		EntityEmployee,
		EntityEstimate,
		EntityExchangeRate,
		EntityInventoryAdjustment,
		EntityInvoice,
		EntityItem,
		EntityJournalEntry,
		EntityPayment,
		EntityPaymentMethod,
		EntityPreferences,
		EntityPurchase,
		EntityPurchaseOrder,
		EntityReimburseCharge,
		EntityTaxAgency,
		EntityTaxCode,
		EntityTaxRate,
		EntityTerm,
		EntityTimeActivity,
		EntityTransfer,
		EntityVendor,
		EntityVendorCredit,
	}
}

// CDCEntityNames returns the entities that support change data capture. QBO
// supports CDC for every API entity except JournalCode, TaxAgency, TaxCode,
// TaxRate, and TimeActivity.
func CDCEntityNames() []EntityName {
	return []EntityName{
		EntityAccount,
		EntityAttachable,
		EntityBill,
		EntityBillPayment,
		EntityClass,
		EntityCompanyCurrency,
		EntityCompanyInfo,
		EntityCreditMemo,
		EntityCustomer,
		EntityCustomerType,
		EntityDepartment,
		EntityDeposit,
		EntityEmployee,
		EntityEstimate,
		EntityExchangeRate,
		EntityInventoryAdjustment,
		EntityInvoice,
		EntityItem,
		EntityJournalEntry,
		EntityPayment,
		EntityPaymentMethod,
		EntityPreferences,
		EntityPurchase,
		EntityPurchaseOrder,
		EntityReimburseCharge,
		EntityTerm,
		EntityTransfer,
		EntityVendor,
		EntityVendorCredit,
	}
}

func IsCDCEntity(name EntityName) bool {
	names := CDCEntityNames()
	m := make(map[EntityName]struct{}, len(names))
	for _, n := range names {
		m[n] = struct{}{}
	}

	_, ok := m[name]

	return ok
}

type Identifiable interface {
	ID() string
}

type NamedEntity interface {
	EntityName() EntityName
}

type Syncronized interface {
	SyncToken() string
}

type Entity interface {
	Identifiable
	NamedEntity
	Syncronized
}

type CDCEntity interface {
	Entity
	CDCStatus() string
	CDCDomain() string
}

type IDValue struct {
	Id string `json:"Id,omitempty"`
}

func (v IDValue) ID() string {
	return v.Id
}

type SyncTokenValue struct {
	SyncTokenValue string `json:"SyncToken,omitempty"`
}

func (v SyncTokenValue) SyncToken() string {
	return v.SyncTokenValue
}

type CDCFields struct {
	Domain string `json:"domain,omitempty"`
	Status string `json:"status,omitempty"`
}

func (f CDCFields) CDCStatus() string {
	return f.Status
}

func (f CDCFields) CDCDomain() string {
	return f.Domain
}

type serviceBase interface {
	doJSON(ctx context.Context, method, realmID, path string, tokenSource oauth2.TokenSource, query url.Values, payload any, out any, opts ...RequestOption) error
}

type service struct {
	client *Client
}

func (s service) doJSON(ctx context.Context, method, realmID, path string, tokenSource oauth2.TokenSource, query url.Values, payload any, out any, opts ...RequestOption) error {
	return s.client.doJSON(ctx, method, realmID, path, tokenSource, query, payload, out, opts...)
}

type companyService struct {
	company *CompanyClient
}

func (s companyService) doJSON(ctx context.Context, method, path string, query url.Values, payload any, out any, opts ...RequestOption) error {
	return s.company.doJSON(ctx, method, path, query, payload, out, opts...)
}

func (s companyService) doBytes(ctx context.Context, method, path string, query url.Values, contentType string, opts ...RequestOption) ([]byte, error) {
	return s.company.doBytes(ctx, method, path, query, contentType, opts...)
}

func (s companyService) doStream(ctx context.Context, method, path string, query url.Values, contentType string, opts ...RequestOption) (io.ReadCloser, error) {
	return s.company.doStream(ctx, method, path, query, contentType, opts...)
}

func (s companyService) doMultipart(ctx context.Context, path string, query url.Values, body io.Reader, contentType string, opts ...RequestOption) ([]byte, error) {
	return s.company.doMultipart(ctx, path, query, body, contentType, opts...)
}

type RootGetter[T any] interface {
	Get(ctx context.Context, realmID string, tokenSource oauth2.TokenSource, id string, opts ...RequestOption) (*T, error)
}

type Getter[T any] interface {
	Get(ctx context.Context, id string, opts ...RequestOption) (*T, error)
}

type RootQuerier[T any] interface {
	Query(ctx context.Context, realmID string, tokenSource oauth2.TokenSource, query string, opts ...QueryOption) ([]T, error)
}

type Querier[T any] interface {
	Query(ctx context.Context, query string, opts ...QueryOption) ([]T, error)
}

type RootLister[T any] interface {
	List(ctx context.Context, realmID string, tokenSource oauth2.TokenSource, opts ...QueryOption) ([]T, error)
}

type Lister[T any] interface {
	List(ctx context.Context, opts ...QueryOption) ([]T, error)
}

type RootPager[T any] interface {
	ListPage(ctx context.Context, realmID string, tokenSource oauth2.TokenSource, page, pageSize int, opts ...QueryOption) ([]T, error)
}

type Pager[T any] interface {
	ListPage(ctx context.Context, page, pageSize int, opts ...QueryOption) ([]T, error)
}

type RootCreator[T any] interface {
	Create(ctx context.Context, realmID string, tokenSource oauth2.TokenSource, entity T, opts ...RequestOption) (*T, error)
}

type Creator[T any] interface {
	Create(ctx context.Context, entity T, opts ...RequestOption) (*T, error)
}

type RootUpdater[T any] interface {
	Update(ctx context.Context, realmID string, tokenSource oauth2.TokenSource, entity T, opts ...RequestOption) (*T, error)
}

type Updater[T any] interface {
	Update(ctx context.Context, entity T, opts ...RequestOption) (*T, error)
}

type RootSparseUpdater[T any] interface {
	UpdateSparse(ctx context.Context, realmID string, tokenSource oauth2.TokenSource, entity T, opts ...RequestOption) (*T, error)
}

type SparseUpdater[T any] interface {
	UpdateSparse(ctx context.Context, entity T, opts ...RequestOption) (*T, error)
}

type RootDeleter[T any] interface {
	Delete(ctx context.Context, realmID string, tokenSource oauth2.TokenSource, req DeleteRequest, opts ...RequestOption) (*T, error)
}

type Deleter[T any] interface {
	Delete(ctx context.Context, req DeleteRequest, opts ...RequestOption) (*T, error)
}

type RootVoider[T any] interface {
	Void(ctx context.Context, realmID string, tokenSource oauth2.TokenSource, req VoidRequest, opts ...RequestOption) (*T, error)
}

type Voider[T any] interface {
	Void(ctx context.Context, req VoidRequest, opts ...RequestOption) (*T, error)
}

type RootSender interface {
	Send(ctx context.Context, realmID string, tokenSource oauth2.TokenSource, entityID, email string, opts ...RequestOption) error
}

type Sender interface {
	Send(ctx context.Context, id, email string, opts ...RequestOption) error
}

type RootCDCReader interface {
	ChangeDataCapture(ctx context.Context, realmID string, tokenSource oauth2.TokenSource, changedSince time.Time, entities []EntityName, opts ...RequestOption) (*CDCResponse, error)
}

type CDCReader interface {
	ChangeDataCapture(ctx context.Context, changedSince time.Time, entities []EntityName, opts ...RequestOption) (*CDCResponse, error)
}

type RootPDFGetter interface {
	GetPDF(ctx context.Context, realmID string, tokenSource oauth2.TokenSource, entity EntityName, entityID string, opts ...RequestOption) ([]byte, error)
}

type PDFGetter interface {
	GetPDF(ctx context.Context, id string, opts ...RequestOption) ([]byte, error)
}

type RootPDFStreamer interface {
	StreamPDF(ctx context.Context, realmID string, tokenSource oauth2.TokenSource, entity EntityName, entityID string, opts ...RequestOption) (io.ReadCloser, error)
}

type PDFStreamer interface {
	StreamPDF(ctx context.Context, id string, opts ...RequestOption) (io.ReadCloser, error)
}

// FileGetter describes a service that can retrieve a file attachment as a
// buffered byte slice. The implementation is expected to resolve a pre-signed
// URL and fetch the file in a single call.
type FileGetter interface {
	GetFile(ctx context.Context, id string, opts ...RequestOption) ([]byte, error)
}

// FileStreamer describes a service that can retrieve a file attachment as a
// streaming io.ReadCloser. The caller is responsible for closing the returned
// ReadCloser.
type FileStreamer interface {
	StreamFile(ctx context.Context, id string, opts ...RequestOption) (io.ReadCloser, error)
}

// EntityCDCReader describes an entity service that supports change data
// capture. Unlike CDCReader, it is scoped to a single entity type and returns
// a typed slice rather than the raw CDCResponse.
type EntityCDCReader[T any] interface {
	ChangeDataCapture(ctx context.Context, changedSince time.Time, opts ...RequestOption) ([]T, error)
}
