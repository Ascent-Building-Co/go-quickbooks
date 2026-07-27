package quickbooks

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"golang.org/x/oauth2"
)

type BatchOperation string

type BatchOption string

const (
	BatchMaxItems                          = 30
	BatchRequestIDMaxLength                = 36
	BatchItemIDMaxLength                   = 10
	BatchOperationCreate    BatchOperation = "create"
	BatchOperationUpdate    BatchOperation = "update"
	BatchOperationDelete    BatchOperation = "delete"
	BatchOptionVoid         BatchOption    = "void"
)

// Entity abbreviation constants for use with BatchItemID. Each two-character
// abbreviation is unique across all supported QBO entities.
const (
	BatchEntityAbbrAccount         = "ac"
	BatchEntityAbbrAttachable      = "at"
	BatchEntityAbbrBill            = "bi"
	BatchEntityAbbrBillPayment     = "bp"
	BatchEntityAbbrClass           = "cl"
	BatchEntityAbbrCompanyCurrency = "cc"
	BatchEntityAbbrCompanyInfo     = "ci"
	BatchEntityAbbrCreditMemo      = "cm"
	BatchEntityAbbrCustomer        = "cu"
	BatchEntityAbbrCustomerType    = "ct"
	BatchEntityAbbrDepartment      = "de"
	BatchEntityAbbrDeposit         = "dp"
	BatchEntityAbbrEmployee        = "em"
	BatchEntityAbbrEstimate        = "es"
	BatchEntityAbbrExchangeRate    = "er"
	BatchEntityAbbrInvoice         = "in"
	BatchEntityAbbrItem            = "it"
	BatchEntityAbbrJournalEntry    = "je"
	BatchEntityAbbrPayment         = "pa"
	BatchEntityAbbrPaymentMethod   = "pm"
	BatchEntityAbbrPreferences     = "pr"
	BatchEntityAbbrPurchase        = "pu"
	BatchEntityAbbrPurchaseOrder   = "po"
	BatchEntityAbbrReimburseCharge = "rc"
	BatchEntityAbbrTaxAgency       = "ta"
	BatchEntityAbbrTaxCode         = "tc"
	BatchEntityAbbrTaxRate         = "tr"
	BatchEntityAbbrTerm            = "te"
	BatchEntityAbbrTimeActivity    = "ti"
	BatchEntityAbbrTransfer        = "tf"
	BatchEntityAbbrVendor          = "ve"
	BatchEntityAbbrVendorCredit    = "vc"
)

// Operation abbreviation constants for use with BatchItemID.
const (
	BatchOpAbbrCreate = "c"
	BatchOpAbbrUpdate = "u"
	BatchOpAbbrDelete = "d"
	BatchOpAbbrVoid   = "v"
	BatchOpAbbrQuery  = "q"
)

// batchEntityAbbrMap is the single source of truth mapping each EntityName to
// its abbreviation. EntityAbbr and EntityFromAbbr both derive from this map.
var batchEntityAbbrMap = map[EntityName]string{
	EntityAccount:         BatchEntityAbbrAccount,
	EntityAttachable:      BatchEntityAbbrAttachable,
	EntityBill:            BatchEntityAbbrBill,
	EntityBillPayment:     BatchEntityAbbrBillPayment,
	EntityClass:           BatchEntityAbbrClass,
	EntityCompanyCurrency: BatchEntityAbbrCompanyCurrency,
	EntityCompanyInfo:     BatchEntityAbbrCompanyInfo,
	EntityCreditMemo:      BatchEntityAbbrCreditMemo,
	EntityCustomer:        BatchEntityAbbrCustomer,
	EntityCustomerType:    BatchEntityAbbrCustomerType,
	EntityDepartment:      BatchEntityAbbrDepartment,
	EntityDeposit:         BatchEntityAbbrDeposit,
	EntityEmployee:        BatchEntityAbbrEmployee,
	EntityEstimate:        BatchEntityAbbrEstimate,
	EntityExchangeRate:    BatchEntityAbbrExchangeRate,
	EntityInvoice:         BatchEntityAbbrInvoice,
	EntityItem:            BatchEntityAbbrItem,
	EntityJournalEntry:    BatchEntityAbbrJournalEntry,
	EntityPayment:         BatchEntityAbbrPayment,
	EntityPaymentMethod:   BatchEntityAbbrPaymentMethod,
	EntityPreferences:     BatchEntityAbbrPreferences,
	EntityPurchase:        BatchEntityAbbrPurchase,
	EntityPurchaseOrder:   BatchEntityAbbrPurchaseOrder,
	EntityReimburseCharge: BatchEntityAbbrReimburseCharge,
	EntityTaxAgency:       BatchEntityAbbrTaxAgency,
	EntityTaxCode:         BatchEntityAbbrTaxCode,
	EntityTaxRate:         BatchEntityAbbrTaxRate,
	EntityTerm:            BatchEntityAbbrTerm,
	EntityTimeActivity:    BatchEntityAbbrTimeActivity,
	EntityTransfer:        BatchEntityAbbrTransfer,
	EntityVendor:          BatchEntityAbbrVendor,
	EntityVendorCredit:    BatchEntityAbbrVendorCredit,
}

// EntityAbbr returns the batch item ID abbreviation for the given EntityName.
// Returns ("", false) if the entity name is not recognised.
func EntityAbbr(entity EntityName) (string, bool) {
	abbr, ok := batchEntityAbbrMap[entity]
	return abbr, ok
}

// EntityFromAbbr returns the EntityName for the given batch item ID
// abbreviation. Returns ("", false) if the abbreviation is not recognised.
func EntityFromAbbr(abbr string) (EntityName, bool) {
	for entity, a := range batchEntityAbbrMap {
		if a == abbr {
			return entity, true
		}
	}
	return "", false
}

// BatchItemID builds a bId-safe identifier from an optional entity
// abbreviation, operation abbreviation, and sequence number. Any part may be
// omitted by passing a zero value ("" or 0); at least one part must be
// non-empty. The returned string is guaranteed to fit within
// BatchItemIDMaxLength (10 chars), and an error is returned if it would exceed
// that limit.
//
// The entity parameter is used as-is, so callers can supply a BatchEntityAbbr*
// constant, the result of EntityAbbr, or any custom abbreviation they choose.
//
// Format when all parts are present: <entity>-<op><seq>, e.g. "in-c1".
// The separator "-" is only included when an entity abbreviation is present
// and at least one of op or seq is also present.
func BatchItemID(entity, op string, seq int) (string, error) {
	var b strings.Builder

	if entity != "" {
		b.WriteString(entity)
	}

	if op != "" || seq > 0 {
		if b.Len() > 0 {
			b.WriteByte('-')
		}
		b.WriteString(op)
		if seq > 0 {
			b.WriteString(strconv.Itoa(seq))
		}
	}

	result := b.String()
	if result == "" {
		return "", fmt.Errorf("batch item id: at least one of entity, op, or seq must be non-empty")
	}
	if len(result) > BatchItemIDMaxLength {
		return "", fmt.Errorf("batch item id %q exceeds maximum length of %d characters", result, BatchItemIDMaxLength)
	}
	return result, nil
}

// BatchRequest is a collection of up to BatchMaxItems (30) batch item
// requests sent in a single API call. Build one with NewBatchRequest or by
// calling Add / AddQuery / AddEntity on an empty value, then pass it to
// CompanyClient.Batch.
type BatchRequest struct {
	BatchItemRequest []BatchItemRequest `json:"BatchItemRequest"`
}

func NewBatchRequest(items ...BatchItemRequest) *BatchRequest {
	return &BatchRequest{BatchItemRequest: append([]BatchItemRequest(nil), items...)}
}

func (r *BatchRequest) Add(item BatchItemRequest) {
	r.BatchItemRequest = append(r.BatchItemRequest, item)
}

func (r *BatchRequest) AddQuery(id, query string) {
	r.Add(NewBatchQuery(id, query))
}

func (r *BatchRequest) AddEntity(id string, operation BatchOperation, entityName EntityName, payload any) {
	r.Add(NewBatchEntity(id, operation, entityName, payload))
}

func (r *BatchRequest) AddVoidEntity(id string, entityName EntityName, payload any) {
	r.Add(NewBatchVoidEntity(id, entityName, payload))
}

func (r BatchRequest) Validate(intent *string) error {
	if len(r.BatchItemRequest) == 0 {
		return fmt.Errorf("batch request requires at least one item")
	}

	if len(r.BatchItemRequest) > BatchMaxItems {
		return fmt.Errorf("batch request contains too many items: max is %d, got %d", BatchMaxItems, len(r.BatchItemRequest))
	}

	bIDMaxLen := BatchItemIDMaxLength
	if intent != nil {
		if *intent == "" {
			bIDMaxLen = RequestIDMaxLength
		}
	}

	seen := make(map[string]struct{}, len(r.BatchItemRequest))
	for i, item := range r.BatchItemRequest {
		if err := item.Validate(bIDMaxLen); err != nil {
			return fmt.Errorf("batch item %d: %w", i+1, err)
		}

		if _, ok := seen[item.ID]; ok {
			return fmt.Errorf("batch item %d: duplicate batch item id %q", i+1, item.ID)
		}
		seen[item.ID] = struct{}{}
	}

	return nil
}

type BatchItemRequest struct {
	ID          string
	Operation   BatchOperation
	OptionsData BatchOption
	Query       string
	EntityName  EntityName
	Payload     any
}

func NewBatchQuery(id, query string, opts ...QueryOption) BatchItemRequest {
	return BatchItemRequest{ID: id, Query: applyQueryOptions(query, opts...)}
}

func NewBatchEntity(id string, operation BatchOperation, entityName EntityName, payload any) BatchItemRequest {
	return BatchItemRequest{ID: id, Operation: operation, EntityName: entityName, Payload: payload}
}

func NewBatchVoidEntity(id string, entityName EntityName, payload any) BatchItemRequest {
	return BatchItemRequest{ID: id, OptionsData: BatchOptionVoid, EntityName: entityName, Payload: payload}
}

func (r BatchItemRequest) Validate(maxID int) error {
	if strings.TrimSpace(r.ID) == "" {
		return fmt.Errorf("batch item id is required")
	}

	if len(r.ID) > maxID {
		return fmt.Errorf("batch item id %q exceeds maximum length of %d characters", r.ID, maxID)
	}

	if r.Query != "" {
		if r.Operation != "" || r.OptionsData != "" || r.EntityName != "" || r.Payload != nil {
			return fmt.Errorf("query batch item cannot include entity payload fields")
		}
		return nil
	}

	if r.EntityName == "" {
		return fmt.Errorf("batch entity name is required")
	}
	if r.Payload == nil {
		return fmt.Errorf("batch payload is required")
	}
	if (r.Operation == "") == (r.OptionsData == "") {
		return fmt.Errorf("batch item must specify exactly one of operation or optionsData")
	}

	return nil
}

func (r BatchItemRequest) MarshalJSON() ([]byte, error) {
	body := map[string]any{"bId": r.ID}

	if r.Query != "" {
		body["Query"] = r.Query
		return json.Marshal(body)
	}

	if r.OptionsData != "" {
		body["optionsData"] = string(r.OptionsData)
	} else {
		body["operation"] = string(r.Operation)
	}

	body[string(r.EntityName)] = r.Payload

	return json.Marshal(body)
}

type BatchResponse struct {
	BatchItemResponse []BatchItemResponse `json:"BatchItemResponse"`
	Time              string              `json:"time"`
}

type BatchItemResponse struct {
	ID            string
	Fault         *Fault
	QueryResponse *QueryResponse
	entities      map[EntityName]json.RawMessage
}

type BatchFault struct {
	ID      string
	Type    string
	Message string
	Code    string
	Detail  string
	Element string
}

type BatchError struct {
	Faults []BatchFault
}

func (e BatchError) Error() string {
	if len(e.Faults) == 0 {
		return "batch faults: none"
	}

	parts := make([]string, 0, len(e.Faults))
	for _, fault := range e.Faults {
		segment := ""
		if fault.ID != "" {
			segment = "bId=" + fault.ID + " "
		}
		if fault.Code != "" {
			segment += fault.Code
		}
		if fault.Element != "" {
			segment += "/" + fault.Element
		}
		if fault.Detail != "" {
			segment += ": " + fault.Detail
		} else if fault.Message != "" {
			segment += ": " + fault.Message
		}
		parts = append(parts, strings.TrimSpace(segment))
	}

	return "batch faults: " + strings.Join(parts, "; ")
}

func (r *BatchItemResponse) UnmarshalJSON(data []byte) error {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	r.entities = make(map[EntityName]json.RawMessage)
	for key, value := range raw {
		switch key {
		case "bId":
			if err := json.Unmarshal(value, &r.ID); err != nil {
				return fmt.Errorf("failed to decode batch item id: %w", err)
			}
		case "Fault":
			var fault Fault
			if err := json.Unmarshal(value, &fault); err != nil {
				return fmt.Errorf("failed to decode batch fault: %w", err)
			}
			r.Fault = &fault
		case "QueryResponse":
			var response QueryResponse
			if err := json.Unmarshal(value, &response); err != nil {
				return fmt.Errorf("failed to decode batch query response: %w", err)
			}
			r.QueryResponse = &response
		default:
			r.entities[EntityName(key)] = value
		}
	}

	return nil
}

func (r BatchItemResponse) EntityNames() []EntityName {
	names := make([]EntityName, 0, len(r.entities))
	for name := range r.entities {
		names = append(names, name)
	}
	sort.Slice(names, func(i, j int) bool { return names[i] < names[j] })
	return names
}

func (r BatchItemResponse) Entity(name EntityName) json.RawMessage {
	return r.entities[name]
}

func (r BatchItemResponse) BatchFaults() []BatchFault {
	if r.Fault == nil {
		return nil
	}

	faults := make([]BatchFault, 0, len(r.Fault.Error))
	for _, item := range r.Fault.Error {
		faults = append(faults, BatchFault{
			ID:      r.ID,
			Type:    r.Fault.Type,
			Message: item.Message,
			Code:    item.Code,
			Detail:  item.Detail,
			Element: item.Element,
		})
	}

	return faults
}

func (r BatchItemResponse) HasFaults() bool {
	return len(r.BatchFaults()) > 0
}

func (r *BatchResponse) Item(id string) *BatchItemResponse {
	if r == nil {
		return nil
	}

	for i := range r.BatchItemResponse {
		if r.BatchItemResponse[i].ID == id {
			return &r.BatchItemResponse[i]
		}
	}

	return nil
}

func (r *BatchResponse) Faults() []BatchFault {
	if r == nil {
		return nil
	}

	return CollectBatchFaults(r.BatchItemResponse)
}

func (r *BatchResponse) HasFaults() bool {
	return len(r.Faults()) > 0
}

func CollectBatchFaults(items []BatchItemResponse) []BatchFault {
	var faults []BatchFault
	for _, item := range items {
		faults = append(faults, item.BatchFaults()...)
	}

	return faults
}

// PartitionBatch splits batch response items into two slices: those without
// faults (ok) and those with faults (bad). Useful when partial success is
// acceptable and you want to process successful items while retrying or
// logging failures separately.
func PartitionBatch(items []BatchItemResponse) (ok []BatchItemResponse, bad []BatchItemResponse) {
	for _, item := range items {
		if item.HasFaults() {
			bad = append(bad, item)
			continue
		}
		ok = append(ok, item)
	}

	return ok, bad
}

// GroupBatchRequests splits a flat slice of BatchItemRequests into
// consecutive sub-slices of at most batchSize items each. If batchSize is
// zero or greater than BatchMaxItems it is clamped to BatchMaxItems.
// Returns nil when items is empty.
func GroupBatchRequests(items []BatchItemRequest, batchSize int) [][]BatchItemRequest {
	if batchSize <= 0 || batchSize > BatchMaxItems {
		batchSize = BatchMaxItems
	}
	if len(items) == 0 {
		return nil
	}

	groups := make([][]BatchItemRequest, 0, (len(items)+batchSize-1)/batchSize)
	for start := 0; start < len(items); start += batchSize {
		end := start + batchSize
		if end > len(items) {
			end = len(items)
		}
		groups = append(groups, items[start:end])
	}

	return groups
}

func (c *Client) Batch(ctx context.Context, realmID string, tokenSource oauth2.TokenSource, req *BatchRequest, opts ...RequestOption) (*BatchResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("batch request is required")
	}

	var cfg requestConfig
	for _, opt := range opts {
		opt.applyRequest(&cfg)
	}
	rpt, err := validateRequestID(true, cfg.requestID)
	if err != nil {
		return nil, err
	}

	if err := req.Validate(rpt); err != nil {
		return nil, err
	}

	var out BatchResponse
	if err := c.doJSON(ctx, http.MethodPost, realmID, "batch", tokenSource, nil, req, &out, opts...); err != nil {
		return nil, err
	}

	return &out, nil
}

func (c *CompanyClient) Batch(ctx context.Context, req *BatchRequest, opts ...RequestOption) (*BatchResponse, error) {
	return c.client.Batch(ctx, c.realmID, c.tokenSource, req, opts...)
}

// ExtractBatchItemEntities collects all entities of the given type from a
// slice of BatchItemResponse values, covering both entity mutation responses
// and query responses. This is useful when working with a subset of a
// BatchResponse — for example, extracting results from specific paginated
// query items rather than the entire response. Returns nil without error when
// items is nil or empty.
func ExtractBatchItemEntities[T any](items []BatchItemResponse, entityName EntityName) ([]T, error) {
	if len(items) == 0 {
		return nil, nil
	}

	var entities []T
	for _, item := range items {
		if raw := item.Entity(entityName); len(raw) > 0 {
			decoded, err := decodeEntitySlice[T](raw)
			if err != nil {
				return nil, err
			}
			entities = append(entities, decoded...)
		}

		if item.QueryResponse == nil {
			continue
		}

		decoded, err := item.QueryResponse.Entities(entityName)
		if err != nil {
			return nil, err
		}

		values, err := decodeEntitySlice[T](decoded)
		if err != nil {
			return nil, err
		}
		entities = append(entities, values...)
	}

	return entities, nil
}

// ExtractBatchEntities collects all entities of the given type from a
// BatchResponse, covering both entity mutation responses and query responses
// within the same batch. Returns nil without error when resp is nil.
func ExtractBatchEntities[T any](resp *BatchResponse, entityName EntityName) ([]T, error) {
	if resp == nil {
		return nil, nil
	}
	return ExtractBatchItemEntities[T](resp.BatchItemResponse, entityName)
}
