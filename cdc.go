package quickbooks

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"golang.org/x/oauth2"
)

// CDCMaxResults is the maximum number of objects a single CDC request will
// return. If a CDC response's MaxResults equals this value, there may be
// additional changes; shorten the look-back window and re-query to ensure
// full data is returned.
const CDCMaxResults = 1000

// QueryResponse is the parsed body of a QBO query or CDC response. It holds
// pagination metadata and the raw entity payloads keyed by EntityName.
// Use Entities to retrieve the raw JSON for a specific entity type, then
// decode it with json.Unmarshal or a typed ExtractBatch / ExtractCDC helper.
type QueryResponse struct {
	StartPosition int
	MaxResults    int
	TotalCount    int
	entities      map[EntityName]json.RawMessage
}

func (r *QueryResponse) UnmarshalJSON(data []byte) error {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	r.entities = make(map[EntityName]json.RawMessage)
	for key, value := range raw {
		switch key {
		case "startPosition":
			n, err := decodeInt(value)
			if err != nil {
				return fmt.Errorf("failed to decode startPosition: %w", err)
			}
			r.StartPosition = n
		case "maxResults":
			n, err := decodeInt(value)
			if err != nil {
				return fmt.Errorf("failed to decode maxResults: %w", err)
			}
			r.MaxResults = n
		case "totalCount":
			n, err := decodeInt(value)
			if err != nil {
				return fmt.Errorf("failed to decode totalCount: %w", err)
			}
			r.TotalCount = n
		default:
			r.entities[EntityName(key)] = value
		}
	}

	return nil
}

func (r QueryResponse) EntityNames() []EntityName {
	names := make([]EntityName, 0, len(r.entities))
	for name := range r.entities {
		names = append(names, name)
	}
	sort.Slice(names, func(i, j int) bool { return names[i] < names[j] })
	return names
}

func (r QueryResponse) Entities(name EntityName) (json.RawMessage, error) {
	if r.entities == nil {
		return nil, nil
	}

	return r.entities[name], nil
}

type CDCResponse struct {
	CDCResponse []CDCQueryResponse `json:"CDCResponse"`
	Time        string             `json:"time"`
}

type CDCQueryResponse struct {
	QueryResponse []QueryResponse `json:"QueryResponse"`
}

func (c *Client) ChangeDataCapture(ctx context.Context, realmID string, tokenSource oauth2.TokenSource, changedSince time.Time, entities []EntityName, opts ...RequestOption) (*CDCResponse, error) {
	if changedSince.IsZero() {
		return nil, fmt.Errorf("changed since is required")
	}
	if len(entities) == 0 {
		return nil, fmt.Errorf("at least one entity is required")
	}

	values := url.Values{}
	values.Set("changedSince", changedSince.Format(time.RFC3339))

	names := make([]string, 0, len(entities))
	for _, entity := range entities {
		if !IsCDCEntity(entity) {
			return nil, fmt.Errorf("entity %q does not support change data capture", entity)
		}
		names = append(names, string(entity))
	}
	values.Set("entities", strings.Join(names, ","))

	var out CDCResponse
	if err := c.doJSON(ctx, http.MethodGet, realmID, "cdc", tokenSource, values, nil, &out, opts...); err != nil {
		return nil, err
	}

	return &out, nil
}

func (c *CompanyClient) ChangeDataCapture(ctx context.Context, changedSince time.Time, entities []EntityName, opts ...RequestOption) (*CDCResponse, error) {
	return c.client.ChangeDataCapture(ctx, c.realmID, c.tokenSource, changedSince, entities, opts...)
}

// ExtractCDCEntities collects all entities of the given type from a
// CDCResponse across all inner QueryResponse entries. Deleted entities are
// included — check CDCEntity.CDCStatus() == "Deleted" to identify them.
// Returns nil without error when resp is nil.
func ExtractCDCEntities[T any](resp *CDCResponse, entityName EntityName) ([]T, error) {
	if resp == nil {
		return nil, nil
	}

	var entities []T
	for _, item := range resp.CDCResponse {
		for _, qr := range item.QueryResponse {
			raw, err := qr.Entities(entityName)
			if err != nil {
				return nil, err
			}
			decoded, err := decodeEntitySlice[T](raw)
			if err != nil {
				return nil, err
			}
			entities = append(entities, decoded...)
		}
	}

	return entities, nil
}

func decodeEntitySlice[T any](raw json.RawMessage) ([]T, error) {
	if len(raw) == 0 {
		return nil, nil
	}

	var slice []T
	if err := json.Unmarshal(raw, &slice); err == nil {
		return slice, nil
	}

	var single T
	if err := json.Unmarshal(raw, &single); err == nil {
		return []T{single}, nil
	}

	return nil, fmt.Errorf("failed to decode entity payload")
}

func decodeInt(data []byte) (int, error) {
	var n int
	if err := json.Unmarshal(data, &n); err == nil {
		return n, nil
	}

	var text string
	if err := json.Unmarshal(data, &text); err != nil {
		return 0, err
	}

	n, err := strconv.Atoi(text)
	if err != nil {
		return 0, err
	}

	return n, nil
}
