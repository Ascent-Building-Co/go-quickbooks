package quickbooks

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"time"
)

type AccountService struct {
	companyService
}

func (c *CompanyClient) Accounts() *AccountService {
	return &AccountService{companyService: companyService{company: c}}
}

// Account represents a QuickBooks Online Account entity.
//
// Fields marked "Read-only" are set by the server and ignored if sent in a
// write request.
type Account struct {
	IDValue
	SyncTokenValue
	CDCFields

	MetaDataValue

	// Required
	Name string `json:"Name,omitempty"`

	// Conditionally required
	AcctNum        string             `json:"AcctNum,omitempty"`
	AccountType    AccountTypeEnum    `json:"AccountType,omitempty"`
	AccountSubType AccountSubTypeEnum `json:"AccountSubType,omitempty"`
	TaxCodeRef     *ReferenceType     `json:"TaxCodeRef,omitempty"`

	// Optional
	ParentRef   *ReferenceType `json:"ParentRef,omitempty"`
	Description string         `json:"Description,omitempty"`
	Active      *bool          `json:"Active,omitempty"`
	CurrencyRef *ReferenceType `json:"CurrencyRef,omitempty"`

	// Read-only: set by the server.
	SubAccount                    *bool                     `json:"SubAccount,omitempty"`
	Classification                AccountClassificationEnum `json:"Classification,omitempty"`
	FullyQualifiedName            string                    `json:"FullyQualifiedName,omitempty"`
	TxnLocationType               string                    `json:"TxnLocationType,omitempty"` // France locale only.
	CurrentBalance                json.Number               `json:"CurrentBalance,omitempty"`
	CurrentBalanceWithSubAccounts json.Number               `json:"CurrentBalanceWithSubAccounts,omitempty"`
	AccountAlias                  string                    `json:"AccountAlias,omitempty"` // France locale only.
}

func (Account) EntityName() EntityName {
	return EntityAccount
}

type accountEnvelope struct {
	Account Account `json:"Account"`
}

type accountQueryEnvelope struct {
	QueryResponse QueryResponse `json:"QueryResponse"`
}

func (s *AccountService) Get(ctx context.Context, id string, opts ...RequestOption) (*Account, error) {
	var out accountEnvelope
	if err := s.doJSON(ctx, http.MethodGet, "account/"+id, nil, nil, &out, opts...); err != nil {
		return nil, err
	}
	return &out.Account, nil
}

func (s *AccountService) Query(ctx context.Context, query string, opts ...QueryOption) ([]Account, error) {
	query = applyQueryOptions(query, opts...)

	var out accountQueryEnvelope
	if err := s.doJSON(ctx, http.MethodGet, "query", url.Values{"query": []string{query}}, nil, &out); err != nil {
		return nil, err
	}

	raw, err := out.QueryResponse.Entities(EntityAccount)
	if err != nil {
		return nil, err
	}

	return decodeEntitySlice[Account](raw)
}

func (s *AccountService) List(ctx context.Context, opts ...QueryOption) ([]Account, error) {
	return s.Query(ctx, "SELECT * FROM Account", opts...)
}

func (s *AccountService) ListPage(ctx context.Context, page, pageSize int, opts ...QueryOption) ([]Account, error) {
	pageOpts := append([]QueryOption{WithPage(page, pageSize)}, opts...)
	return s.List(ctx, pageOpts...)
}

func (s *AccountService) Create(ctx context.Context, account Account, opts ...RequestOption) (*Account, error) {
	var out accountEnvelope
	if err := s.doJSON(ctx, http.MethodPost, "account", nil, account, &out, opts...); err != nil {
		return nil, err
	}
	return &out.Account, nil
}

func (s *AccountService) Update(ctx context.Context, account Account, opts ...RequestOption) (*Account, error) {
	var out accountEnvelope
	if err := s.doJSON(ctx, http.MethodPost, "account", nil, account, &out, opts...); err != nil {
		return nil, err
	}
	return &out.Account, nil
}

func (s *AccountService) UpdateSparse(ctx context.Context, account Account, opts ...RequestOption) (*Account, error) {
	payload := struct {
		Account
		Sparse bool `json:"sparse"`
	}{account, true}
	var out accountEnvelope
	if err := s.doJSON(ctx, http.MethodPost, "account", nil, payload, &out, opts...); err != nil {
		return nil, err
	}
	return &out.Account, nil
}

func (s *AccountService) ChangeDataCapture(ctx context.Context, changedSince time.Time, opts ...RequestOption) ([]Account, error) {
	resp, err := s.company.ChangeDataCapture(ctx, changedSince, []EntityName{EntityAccount}, opts...)
	if err != nil {
		return nil, err
	}
	return s.ExtractCDC(resp)
}

func (s *AccountService) ExtractCDC(resp *CDCResponse) ([]Account, error) {
	return ExtractCDCEntities[Account](resp, EntityAccount)
}

func (s *AccountService) ExtractBatch(resp *BatchResponse) ([]Account, error) {
	return ExtractBatchEntities[Account](resp, EntityAccount)
}

func (s *AccountService) ExtractBatchItems(items []BatchItemResponse) ([]Account, error) {
	return ExtractBatchItemEntities[Account](items, EntityAccount)
}

func (s *AccountService) BatchCreateItem(id string, account Account) BatchItemRequest {
	return NewBatchEntity(id, BatchOperationCreate, EntityAccount, account)
}

func (s *AccountService) BatchUpdateItem(id string, account Account) BatchItemRequest {
	return NewBatchEntity(id, BatchOperationUpdate, EntityAccount, account)
}

func (s *AccountService) BatchQueryItem(id, query string, opts ...QueryOption) BatchItemRequest {
	return NewBatchQuery(id, query, opts...)
}
