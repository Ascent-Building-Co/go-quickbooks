package quickbooks

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"time"
)

// DepositLineDetail holds the account-based detail fields for a deposit line.
// AccountRef is required; all other fields are optional.
type DepositLineDetail struct {
	AccountRef       *ReferenceType `json:"AccountRef,omitempty"`
	ClassRef         *ReferenceType `json:"ClassRef,omitempty"`
	PaymentMethodRef *ReferenceType `json:"PaymentMethodRef,omitempty"`
	TaxCodeRef       *ReferenceType `json:"TaxCodeRef,omitempty"`
	Entity           *ReferenceType `json:"Entity,omitempty"`
	CheckNum         string         `json:"CheckNum,omitempty"`
	TaxApplicableOn  string         `json:"TaxApplicableOn,omitempty"`
	TxnType          string         `json:"TxnType,omitempty"`
}

// CashBackInfo describes the cash-back portion of a Deposit transaction.
// AccountRef and Amount are required; Memo is optional.
type CashBackInfo struct {
	AccountRef *ReferenceType `json:"AccountRef,omitempty"`
	Amount     json.Number    `json:"Amount,omitempty"`
	Memo       string         `json:"Memo,omitempty"`
}

type DepositService struct {
	companyService
}

func (c *CompanyClient) Deposits() *DepositService {
	return &DepositService{companyService: companyService{company: c}}
}

// Deposit represents a QuickBooks Online Deposit entity, recording funds
// received and deposited into a bank or other asset account.
//
// Fields marked "Read-only" are set by the server and ignored if sent in a
// write request.
type Deposit struct {
	IDValue
	SyncTokenValue
	CDCFields

	MetaDataValue

	// Required
	DepositToAccountRef *ReferenceType `json:"DepositToAccountRef,omitempty"`
	Line                []DepositLine  `json:"Line,omitempty"`

	// Conditionally required
	CurrencyRef          *ReferenceType           `json:"CurrencyRef,omitempty"`
	GlobalTaxCalculation GlobalTaxCalculationEnum `json:"GlobalTaxCalculation,omitempty"`

	// Optional
	TxnDate                 *Date          `json:"TxnDate,omitempty"`
	PrivateNote             string         `json:"PrivateNote,omitempty"`
	ExchangeRate            json.Number    `json:"ExchangeRate,omitempty"`
	DepartmentRef           *ReferenceType `json:"DepartmentRef,omitempty"`
	TxnSource               string         `json:"TxnSource,omitempty"`
	CashBack                *CashBackInfo  `json:"CashBack,omitempty"`
	TransactionLocationType string         `json:"TransactionLocationType,omitempty"` // France locale only.
	TxnTaxDetail            *TxnTaxDetail  `json:"TxnTaxDetail,omitempty"`

	// Read-only: set by the server.
	TotalAmt     json.Number    `json:"TotalAmt,omitempty"`
	HomeTotalAmt json.Number    `json:"HomeTotalAmt,omitempty"`
	RecurDataRef *ReferenceType `json:"RecurDataRef,omitempty"`
}

func (Deposit) EntityName() EntityName {
	return EntityDeposit
}

// DepositLine represents a single line on a Deposit. The two valid variants
// are a direct account-based deposit (DetailType = DepositLineDetail) and a
// linked-transaction deposit (DetailType omitted, LinkedTxn populated).
//
// ProjectRef is not part of the v75 schema model but appears in API payloads
// when the line is associated with a QuickBooks project.
type DepositLine struct {
	IDValue

	LineNum           json.Number        `json:"LineNum,omitempty"`
	Description       string             `json:"Description,omitempty"`
	Amount            json.Number        `json:"Amount,omitempty"`
	DetailType        LineDetailType     `json:"DetailType,omitempty"`
	ProjectRef        *ReferenceType     `json:"ProjectRef,omitempty"`
	LinkedTxn         []LinkedTxn        `json:"LinkedTxn,omitempty"`
	DepositLineDetail *DepositLineDetail `json:"DepositLineDetail,omitempty"`
}

type depositEnvelope struct {
	Deposit Deposit `json:"Deposit"`
}

type depositQueryEnvelope struct {
	QueryResponse QueryResponse `json:"QueryResponse"`
}

func (s *DepositService) Get(ctx context.Context, id string, opts ...RequestOption) (*Deposit, error) {
	var out depositEnvelope
	if err := s.doJSON(ctx, http.MethodGet, "deposit/"+id, nil, nil, &out, opts...); err != nil {
		return nil, err
	}
	return &out.Deposit, nil
}

func (s *DepositService) Query(ctx context.Context, query string, opts ...QueryOption) ([]Deposit, error) {
	query = applyQueryOptions(query, opts...)

	var out depositQueryEnvelope
	if err := s.doJSON(ctx, http.MethodGet, "query", url.Values{"query": []string{query}}, nil, &out); err != nil {
		return nil, err
	}

	raw, err := out.QueryResponse.Entities(EntityDeposit)
	if err != nil {
		return nil, err
	}

	return decodeEntitySlice[Deposit](raw)
}

func (s *DepositService) List(ctx context.Context, opts ...QueryOption) ([]Deposit, error) {
	return s.Query(ctx, "SELECT * FROM Deposit", opts...)
}

func (s *DepositService) ListPage(ctx context.Context, page, pageSize int, opts ...QueryOption) ([]Deposit, error) {
	pageOpts := append([]QueryOption{WithPage(page, pageSize)}, opts...)
	return s.List(ctx, pageOpts...)
}

func (s *DepositService) Create(ctx context.Context, deposit Deposit, opts ...RequestOption) (*Deposit, error) {
	var out depositEnvelope
	if err := s.doJSON(ctx, http.MethodPost, "deposit", nil, deposit, &out, opts...); err != nil {
		return nil, err
	}
	return &out.Deposit, nil
}

func (s *DepositService) Update(ctx context.Context, deposit Deposit, opts ...RequestOption) (*Deposit, error) {
	var out depositEnvelope
	if err := s.doJSON(ctx, http.MethodPost, "deposit", nil, deposit, &out, opts...); err != nil {
		return nil, err
	}
	return &out.Deposit, nil
}

func (s *DepositService) UpdateSparse(ctx context.Context, deposit Deposit, opts ...RequestOption) (*Deposit, error) {
	payload := struct {
		Deposit
		Sparse bool `json:"sparse"`
	}{deposit, true}
	var out depositEnvelope
	if err := s.doJSON(ctx, http.MethodPost, "deposit", nil, payload, &out, opts...); err != nil {
		return nil, err
	}
	return &out.Deposit, nil
}

func (s *DepositService) Delete(ctx context.Context, req DeleteRequest, opts ...RequestOption) (*Deposit, error) {
	var out depositEnvelope
	query := url.Values{"operation": []string{"delete"}}
	if err := s.doJSON(ctx, http.MethodPost, "deposit", query, req, &out, opts...); err != nil {
		return nil, err
	}
	return &out.Deposit, nil
}

func (s *DepositService) ChangeDataCapture(ctx context.Context, changedSince time.Time, opts ...RequestOption) ([]Deposit, error) {
	resp, err := s.company.ChangeDataCapture(ctx, changedSince, []EntityName{EntityDeposit}, opts...)
	if err != nil {
		return nil, err
	}
	return s.ExtractCDC(resp)
}

func (s *DepositService) ExtractCDC(resp *CDCResponse) ([]Deposit, error) {
	return ExtractCDCEntities[Deposit](resp, EntityDeposit)
}

func (s *DepositService) ExtractBatch(resp *BatchResponse) ([]Deposit, error) {
	return ExtractBatchEntities[Deposit](resp, EntityDeposit)
}

func (s *DepositService) ExtractBatchItems(items []BatchItemResponse) ([]Deposit, error) {
	return ExtractBatchItemEntities[Deposit](items, EntityDeposit)
}

func (s *DepositService) BatchCreateItem(id string, deposit Deposit) BatchItemRequest {
	return NewBatchEntity(id, BatchOperationCreate, EntityDeposit, deposit)
}

func (s *DepositService) BatchUpdateItem(id string, deposit Deposit) BatchItemRequest {
	return NewBatchEntity(id, BatchOperationUpdate, EntityDeposit, deposit)
}

func (s *DepositService) BatchDeleteItem(id string, req DeleteRequest) BatchItemRequest {
	return NewBatchEntity(id, BatchOperationDelete, EntityDeposit, req)
}

func (s *DepositService) BatchQueryItem(id, query string, opts ...QueryOption) BatchItemRequest {
	return NewBatchQuery(id, query, opts...)
}
