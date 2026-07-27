package quickbooks

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"time"
)

type TransferService struct {
	companyService
}

func (c *CompanyClient) Transfers() *TransferService {
	return &TransferService{companyService: companyService{company: c}}
}

// Transfer represents a QuickBooks Online Transfer entity, recording a
// movement of funds between two accounts.
//
// Fields marked "Read-only" are set by the server and ignored if sent in a
// write request.
type Transfer struct {
	IDValue
	SyncTokenValue
	CDCFields

	MetaData *ModificationMetaData `json:"MetaData,omitempty"`

	// Required
	FromAccountRef *ReferenceType `json:"FromAccountRef,omitempty"`
	ToAccountRef   *ReferenceType `json:"ToAccountRef,omitempty"`
	Amount         json.Number    `json:"Amount,omitempty"`

	// Optional
	PrivateNote             string `json:"PrivateNote,omitempty"`
	TxnDate                 *Date  `json:"TxnDate,omitempty"`
	TransactionLocationType string `json:"TransactionLocationType,omitempty"` // France locale only.

	// Read-only: set by the server.
	RecurDataRef *ReferenceType `json:"RecurDataRef,omitempty"`
}

func (Transfer) EntityName() EntityName {
	return EntityTransfer
}

type transferEnvelope struct {
	Transfer Transfer `json:"Transfer"`
}

type transferQueryEnvelope struct {
	QueryResponse QueryResponse `json:"QueryResponse"`
}

func (s *TransferService) Get(ctx context.Context, id string, opts ...RequestOption) (*Transfer, error) {
	var out transferEnvelope
	if err := s.doJSON(ctx, http.MethodGet, "transfer/"+id, nil, nil, &out, opts...); err != nil {
		return nil, err
	}
	return &out.Transfer, nil
}

func (s *TransferService) Query(ctx context.Context, query string, opts ...QueryOption) ([]Transfer, error) {
	query = applyQueryOptions(query, opts...)

	var out transferQueryEnvelope
	if err := s.doJSON(ctx, http.MethodGet, "query", url.Values{"query": []string{query}}, nil, &out); err != nil {
		return nil, err
	}

	raw, err := out.QueryResponse.Entities(EntityTransfer)
	if err != nil {
		return nil, err
	}

	return decodeEntitySlice[Transfer](raw)
}

func (s *TransferService) List(ctx context.Context, opts ...QueryOption) ([]Transfer, error) {
	return s.Query(ctx, "SELECT * FROM Transfer", opts...)
}

func (s *TransferService) ListPage(ctx context.Context, page, pageSize int, opts ...QueryOption) ([]Transfer, error) {
	pageOpts := append([]QueryOption{WithPage(page, pageSize)}, opts...)
	return s.List(ctx, pageOpts...)
}

func (s *TransferService) Create(ctx context.Context, transfer Transfer, opts ...RequestOption) (*Transfer, error) {
	var out transferEnvelope
	if err := s.doJSON(ctx, http.MethodPost, "transfer", nil, transfer, &out, opts...); err != nil {
		return nil, err
	}
	return &out.Transfer, nil
}

func (s *TransferService) Update(ctx context.Context, transfer Transfer, opts ...RequestOption) (*Transfer, error) {
	var out transferEnvelope
	if err := s.doJSON(ctx, http.MethodPost, "transfer", nil, transfer, &out, opts...); err != nil {
		return nil, err
	}
	return &out.Transfer, nil
}

func (s *TransferService) UpdateSparse(ctx context.Context, transfer Transfer, opts ...RequestOption) (*Transfer, error) {
	payload := struct {
		Transfer
		Sparse bool `json:"sparse"`
	}{transfer, true}
	var out transferEnvelope
	if err := s.doJSON(ctx, http.MethodPost, "transfer", nil, payload, &out, opts...); err != nil {
		return nil, err
	}
	return &out.Transfer, nil
}

func (s *TransferService) Delete(ctx context.Context, req DeleteRequest, opts ...RequestOption) (*Transfer, error) {
	var out transferEnvelope
	query := url.Values{"operation": []string{"delete"}}
	if err := s.doJSON(ctx, http.MethodPost, "transfer", query, req, &out, opts...); err != nil {
		return nil, err
	}
	return &out.Transfer, nil
}

func (s *TransferService) ChangeDataCapture(ctx context.Context, changedSince time.Time, opts ...RequestOption) ([]Transfer, error) {
	resp, err := s.company.ChangeDataCapture(ctx, changedSince, []EntityName{EntityTransfer}, opts...)
	if err != nil {
		return nil, err
	}
	return s.ExtractCDC(resp)
}

func (s *TransferService) ExtractCDC(resp *CDCResponse) ([]Transfer, error) {
	return ExtractCDCEntities[Transfer](resp, EntityTransfer)
}

func (s *TransferService) ExtractBatch(resp *BatchResponse) ([]Transfer, error) {
	return ExtractBatchEntities[Transfer](resp, EntityTransfer)
}

func (s *TransferService) ExtractBatchItems(items []BatchItemResponse) ([]Transfer, error) {
	return ExtractBatchItemEntities[Transfer](items, EntityTransfer)
}

func (s *TransferService) BatchCreateItem(id string, transfer Transfer) BatchItemRequest {
	return NewBatchEntity(id, BatchOperationCreate, EntityTransfer, transfer)
}

func (s *TransferService) BatchUpdateItem(id string, transfer Transfer) BatchItemRequest {
	return NewBatchEntity(id, BatchOperationUpdate, EntityTransfer, transfer)
}

func (s *TransferService) BatchDeleteItem(id string, req DeleteRequest) BatchItemRequest {
	return NewBatchEntity(id, BatchOperationDelete, EntityTransfer, req)
}

func (s *TransferService) BatchQueryItem(id, query string, opts ...QueryOption) BatchItemRequest {
	return NewBatchQuery(id, query, opts...)
}
