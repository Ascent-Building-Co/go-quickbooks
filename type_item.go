package quickbooks

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"time"
)

// ItemGroupDetail holds the bundle line items for a Group (bundle) type Item.
type ItemGroupDetail struct {
	ItemGroupLine []ItemGroupLine `json:"ItemGroupLine,omitempty"`
}

// ItemGroupLine represents a single component line in a Group (bundle) item.
// ItemRef uses the standard ReferenceType which carries an optional Type field
// indicating the component item's type (e.g. "Inventory", "Service").
type ItemGroupLine struct {
	Qty     json.Number    `json:"Qty,omitempty"`
	ItemRef *ReferenceType `json:"ItemRef,omitempty"`
}

type ItemService struct {
	companyService
}

func (c *CompanyClient) Items() *ItemService {
	return &ItemService{companyService: companyService{company: c}}
}

// Item represents a QuickBooks Online Item entity — a product, service,
// category, or bundle that can appear on transactions.
//
// The Type field determines which other fields are required. See ItemTypeEnum
// for the full set of supported types.
//
// Group (bundle) items cannot be created via the API; they can only be read
// and updated.
//
// To propagate account changes to existing transactions on update, pass
// WithInclude(IncludeDoNotUpdateAccountOnTxns) to prevent propagation, or
// omit it to use the default behaviour.
//
// Fields marked "Read-only" are set by the server and ignored if sent in a
// write request.
type Item struct {
	IDValue
	SyncTokenValue
	CDCFields

	MetaData *ModificationMetaData `json:"MetaData,omitempty"`

	// Required
	Name string `json:"Name,omitempty"`

	// Conditionally required — varies by Type.
	// InvStartDate, QtyOnHand, and AssetAccountRef are required for Inventory items.
	// IncomeAccountRef is required for non-Category, non-Group items.
	Type             ItemTypeEnum   `json:"Type,omitempty"`
	InvStartDate     *Date          `json:"InvStartDate,omitempty"`
	QtyOnHand        json.Number    `json:"QtyOnHand,omitempty"`
	AssetAccountRef  *ReferenceType `json:"AssetAccountRef,omitempty"`
	IncomeAccountRef *ReferenceType `json:"IncomeAccountRef,omitempty"`

	// Optional
	ItemCategoryType    ItemCategoryTypeEnum `json:"ItemCategoryType,omitempty"` // France locale only. Read-only after creation.
	Sku                 string               `json:"Sku,omitempty"`
	Description         string               `json:"Description,omitempty"`
	PurchaseDesc        string               `json:"PurchaseDesc,omitempty"`
	UnitPrice           json.Number          `json:"UnitPrice,omitempty"`
	PurchaseCost        json.Number          `json:"PurchaseCost,omitempty"`
	Active              *bool                `json:"Active,omitempty"`
	Taxable             *bool                `json:"Taxable,omitempty"`
	TrackQtyOnHand      *bool                `json:"TrackQtyOnHand,omitempty"`
	SubItem             *bool                `json:"SubItem,omitempty"`
	ParentRef           *ReferenceType       `json:"ParentRef,omitempty"`
	ClassRef            *ReferenceType       `json:"ClassRef,omitempty"`
	SalesTaxCodeRef     *ReferenceType       `json:"SalesTaxCodeRef,omitempty"`
	PurchaseTaxCodeRef  *ReferenceType       `json:"PurchaseTaxCodeRef,omitempty"`
	PrefVendorRef       *ReferenceType       `json:"PrefVendorRef,omitempty"`
	ReorderPoint        json.Number          `json:"ReorderPoint,omitempty"`
	SalesTaxIncluded    *bool                `json:"SalesTaxIncluded,omitempty"`
	PurchaseTaxIncluded *bool                `json:"PurchaseTaxIncluded,omitempty"`
	ServiceType         string               `json:"ServiceType,omitempty"` // India locale only.
	Source              string               `json:"Source,omitempty"`
	UQCDisplayText      string               `json:"UQCDisplayText,omitempty"`    // India locale only.
	UQCId               string               `json:"UQCId,omitempty"`             // India locale only.
	AbatementRate       json.Number          `json:"AbatementRate,omitempty"`     // India locale only.
	ReverseChargeRate   json.Number          `json:"ReverseChargeRate,omitempty"` // India locale only.
	// PrintGroupedItems controls whether the component items of a Group are
	// printed on transactions. Only applicable for Group type items.
	PrintGroupedItems *bool `json:"PrintGroupedItems,omitempty"`
	// ItemGroupDetail holds the bundle lines for Group type items. Not part of
	// the v75 schema model but present in Group item responses.
	ItemGroupDetail *ItemGroupDetail `json:"ItemGroupDetail,omitempty"`
	// CustomExtensions associates additional attributes with the item.
	// Currently only supports the DIMENSION extension type. Intuit Enterprise Suite only.
	CustomExtensions []CustomExtension `json:"CustomExtensions,omitempty"`

	// Read-only: set by the server.
	FullyQualifiedName   string         `json:"FullyQualifiedName,omitempty"`
	Level                int            `json:"Level,omitempty"`
	ExpenseAccountRef    *ReferenceType `json:"ExpenseAccountRef,omitempty"`
	TaxClassificationRef *ReferenceType `json:"TaxClassificationRef,omitempty"`
}

func (Item) EntityName() EntityName {
	return EntityItem
}

type itemEnvelope struct {
	Item Item `json:"Item"`
}

type itemQueryEnvelope struct {
	QueryResponse QueryResponse `json:"QueryResponse"`
}

func (s *ItemService) Get(ctx context.Context, id string, opts ...RequestOption) (*Item, error) {
	var out itemEnvelope
	if err := s.doJSON(ctx, http.MethodGet, "item/"+id, nil, nil, &out, opts...); err != nil {
		return nil, err
	}
	return &out.Item, nil
}

func (s *ItemService) Query(ctx context.Context, query string, opts ...QueryOption) ([]Item, error) {
	query = applyQueryOptions(query, opts...)

	var out itemQueryEnvelope
	if err := s.doJSON(ctx, http.MethodGet, "query", url.Values{"query": []string{query}}, nil, &out); err != nil {
		return nil, err
	}

	raw, err := out.QueryResponse.Entities(EntityItem)
	if err != nil {
		return nil, err
	}

	return decodeEntitySlice[Item](raw)
}

func (s *ItemService) List(ctx context.Context, opts ...QueryOption) ([]Item, error) {
	return s.Query(ctx, "SELECT * FROM Item", opts...)
}

func (s *ItemService) ListPage(ctx context.Context, page, pageSize int, opts ...QueryOption) ([]Item, error) {
	pageOpts := append([]QueryOption{WithPage(page, pageSize)}, opts...)
	return s.List(ctx, pageOpts...)
}

func (s *ItemService) Create(ctx context.Context, item Item, opts ...RequestOption) (*Item, error) {
	var out itemEnvelope
	if err := s.doJSON(ctx, http.MethodPost, "item", nil, item, &out, opts...); err != nil {
		return nil, err
	}
	return &out.Item, nil
}

func (s *ItemService) Update(ctx context.Context, item Item, opts ...RequestOption) (*Item, error) {
	var out itemEnvelope
	if err := s.doJSON(ctx, http.MethodPost, "item", nil, item, &out, opts...); err != nil {
		return nil, err
	}
	return &out.Item, nil
}

func (s *ItemService) UpdateSparse(ctx context.Context, item Item, opts ...RequestOption) (*Item, error) {
	payload := struct {
		Item
		Sparse bool `json:"sparse"`
	}{item, true}
	var out itemEnvelope
	if err := s.doJSON(ctx, http.MethodPost, "item", nil, payload, &out, opts...); err != nil {
		return nil, err
	}
	return &out.Item, nil
}

func (s *ItemService) ChangeDataCapture(ctx context.Context, changedSince time.Time, opts ...RequestOption) ([]Item, error) {
	resp, err := s.company.ChangeDataCapture(ctx, changedSince, []EntityName{EntityItem}, opts...)
	if err != nil {
		return nil, err
	}
	return s.ExtractCDC(resp)
}

func (s *ItemService) ExtractCDC(resp *CDCResponse) ([]Item, error) {
	return ExtractCDCEntities[Item](resp, EntityItem)
}

func (s *ItemService) ExtractBatch(resp *BatchResponse) ([]Item, error) {
	return ExtractBatchEntities[Item](resp, EntityItem)
}

func (s *ItemService) ExtractBatchItems(items []BatchItemResponse) ([]Item, error) {
	return ExtractBatchItemEntities[Item](items, EntityItem)
}

func (s *ItemService) BatchCreateItem(id string, item Item) BatchItemRequest {
	return NewBatchEntity(id, BatchOperationCreate, EntityItem, item)
}

func (s *ItemService) BatchUpdateItem(id string, item Item) BatchItemRequest {
	return NewBatchEntity(id, BatchOperationUpdate, EntityItem, item)
}

func (s *ItemService) BatchQueryItem(id, query string, opts ...QueryOption) BatchItemRequest {
	return NewBatchQuery(id, query, opts...)
}
