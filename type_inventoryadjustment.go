package quickbooks

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"time"
)

// InventoryAdjustmentService provides access to the InventoryAdjustment-related
// endpoints of the QuickBooks Online API.
//
// InventoryAdjustment is available for QuickBooks Online Plus and Advanced
// companies in the US and requires minorversion 70 or higher.
type InventoryAdjustmentService struct {
	companyService
}

func (c *CompanyClient) InventoryAdjustments() *InventoryAdjustmentService {
	return &InventoryAdjustmentService{companyService: companyService{company: c}}
}

// InventoryAdjustment represents a QuickBooks Online InventoryAdjustment
// entity, which changes the quantity on hand of inventory items for reasons
// such as damage, stock write-off, shrinkage, or expiration. The underlying
// accounting and valuation changes are handled automatically by QuickBooks.
//
// Fields marked "Read-only" are set by the server and ignored if sent in a
// write request.
type InventoryAdjustment struct {
	IDValue
	SyncTokenValue
	CDCFields

	MetaDataValue

	// Required
	AdjustAccountRef *ReferenceType            `json:"AdjustAccountRef,omitempty"`
	Line             []InventoryAdjustmentLine `json:"Line,omitempty"`
	TxnDate          *Date                     `json:"TxnDate,omitempty"`   // Server date is used if omitted.
	DocNumber        string                    `json:"DocNumber,omitempty"` // Reference number for the transaction. Required for create. Example: "27".

	// Optional
	PrivateNote string `json:"PrivateNote,omitempty"` // Max 4000 characters.
}

func (InventoryAdjustment) EntityName() EntityName {
	return EntityInventoryAdjustment
}

// InventoryAdjustmentLine represents a single line item on an
// InventoryAdjustment. DetailType must be set to ItemAdjustmentLineDetailType
// and ItemAdjustmentLineDetail must be populated; it is the only detail type
// valid for this entity.
type InventoryAdjustmentLine struct {
	IDValue

	DetailType               LineDetailType            `json:"DetailType,omitempty"`
	ItemAdjustmentLineDetail *ItemAdjustmentLineDetail `json:"ItemAdjustmentLineDetail,omitempty"`
}

// ItemAdjustmentLineDetail holds the detail fields for an
// InventoryAdjustmentLine.
type ItemAdjustmentLineDetail struct {
	// Required
	QtyDiff json.Number `json:"QtyDiff,omitempty"` // Signed change in quantity: positive increments, negative decrements.

	// Optional
	ItemRef *ReferenceType `json:"ItemRef,omitempty"`
}

type inventoryAdjustmentEnvelope struct {
	InventoryAdjustment InventoryAdjustment `json:"InventoryAdjustment"`
}

type inventoryAdjustmentQueryEnvelope struct {
	QueryResponse QueryResponse `json:"QueryResponse"`
}

func (s *InventoryAdjustmentService) Get(ctx context.Context, id string, opts ...RequestOption) (*InventoryAdjustment, error) {
	var out inventoryAdjustmentEnvelope
	if err := s.doJSON(ctx, http.MethodGet, "inventoryadjustment/"+id, nil, nil, &out, opts...); err != nil {
		return nil, err
	}
	return &out.InventoryAdjustment, nil
}

func (s *InventoryAdjustmentService) Query(ctx context.Context, query string, opts ...QueryOption) ([]InventoryAdjustment, error) {
	query = applyQueryOptions(query, opts...)

	var out inventoryAdjustmentQueryEnvelope
	if err := s.doJSON(ctx, http.MethodGet, "query", url.Values{"query": []string{query}}, nil, &out); err != nil {
		return nil, err
	}

	raw, err := out.QueryResponse.Entities(EntityInventoryAdjustment)
	if err != nil {
		return nil, err
	}

	return decodeEntitySlice[InventoryAdjustment](raw)
}

func (s *InventoryAdjustmentService) List(ctx context.Context, opts ...QueryOption) ([]InventoryAdjustment, error) {
	return s.Query(ctx, "SELECT * FROM InventoryAdjustment", opts...)
}

func (s *InventoryAdjustmentService) ListPage(ctx context.Context, page, pageSize int, opts ...QueryOption) ([]InventoryAdjustment, error) {
	pageOpts := append([]QueryOption{WithPage(page, pageSize)}, opts...)
	return s.List(ctx, pageOpts...)
}

func (s *InventoryAdjustmentService) Create(ctx context.Context, adjustment InventoryAdjustment, opts ...RequestOption) (*InventoryAdjustment, error) {
	var out inventoryAdjustmentEnvelope
	if err := s.doJSON(ctx, http.MethodPost, "inventoryadjustment", nil, adjustment, &out, opts...); err != nil {
		return nil, err
	}
	return &out.InventoryAdjustment, nil
}

// UpdateSparse updates the inventory adjustment with a sparse payload. The API
// only supports sparse updates for this entity, so there is no full Update
// method.
func (s *InventoryAdjustmentService) UpdateSparse(ctx context.Context, adjustment InventoryAdjustment, opts ...RequestOption) (*InventoryAdjustment, error) {
	payload := struct {
		InventoryAdjustment
		Sparse bool `json:"sparse"`
	}{adjustment, true}
	var out inventoryAdjustmentEnvelope
	if err := s.doJSON(ctx, http.MethodPost, "inventoryadjustment", nil, payload, &out, opts...); err != nil {
		return nil, err
	}
	return &out.InventoryAdjustment, nil
}

func (s *InventoryAdjustmentService) Delete(ctx context.Context, req DeleteRequest, opts ...RequestOption) (*InventoryAdjustment, error) {
	var out inventoryAdjustmentEnvelope
	query := url.Values{"operation": []string{"delete"}}
	if err := s.doJSON(ctx, http.MethodPost, "inventoryadjustment", query, req, &out, opts...); err != nil {
		return nil, err
	}
	return &out.InventoryAdjustment, nil
}

func (s *InventoryAdjustmentService) ChangeDataCapture(ctx context.Context, changedSince time.Time, opts ...RequestOption) ([]InventoryAdjustment, error) {
	resp, err := s.company.ChangeDataCapture(ctx, changedSince, []EntityName{EntityInventoryAdjustment}, opts...)
	if err != nil {
		return nil, err
	}
	return s.ExtractCDC(resp)
}

func (s *InventoryAdjustmentService) ExtractCDC(resp *CDCResponse) ([]InventoryAdjustment, error) {
	return ExtractCDCEntities[InventoryAdjustment](resp, EntityInventoryAdjustment)
}

func (s *InventoryAdjustmentService) ExtractBatch(resp *BatchResponse) ([]InventoryAdjustment, error) {
	return ExtractBatchEntities[InventoryAdjustment](resp, EntityInventoryAdjustment)
}

func (s *InventoryAdjustmentService) ExtractBatchItems(items []BatchItemResponse) ([]InventoryAdjustment, error) {
	return ExtractBatchItemEntities[InventoryAdjustment](items, EntityInventoryAdjustment)
}

func (s *InventoryAdjustmentService) BatchCreateItem(id string, adjustment InventoryAdjustment) BatchItemRequest {
	return NewBatchEntity(id, BatchOperationCreate, EntityInventoryAdjustment, adjustment)
}

// BatchUpdateItem returns a batch item that sparse-updates the inventory
// adjustment. The API only supports sparse updates for this entity, so the
// payload always carries "sparse":true.
func (s *InventoryAdjustmentService) BatchUpdateItem(id string, adjustment InventoryAdjustment) BatchItemRequest {
	payload := struct {
		InventoryAdjustment
		Sparse bool `json:"sparse"`
	}{adjustment, true}
	return NewBatchEntity(id, BatchOperationUpdate, EntityInventoryAdjustment, payload)
}

func (s *InventoryAdjustmentService) BatchDeleteItem(id string, req DeleteRequest) BatchItemRequest {
	return NewBatchEntity(id, BatchOperationDelete, EntityInventoryAdjustment, req)
}

func (s *InventoryAdjustmentService) BatchQueryItem(id, query string, opts ...QueryOption) BatchItemRequest {
	return NewBatchQuery(id, query, opts...)
}
