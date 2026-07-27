package quickbooks

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
)

func TestItemServiceGet(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/company/123/item/37", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_, err := io.WriteString(w, `{"Item":{"Id":"37","SyncToken":"0","domain":"QBO","status":"Active","Name":"Office Supplies","Type":"Inventory","FullyQualifiedName":"Office Supplies"}}`)
		require.NoError(t, err)
	}))
	defer server.Close()

	client, err := NewClient(WithHTTPClient(server.Client()), WithBaseURL(server.URL+"/v3/company"))
	require.NoError(t, err)

	item, err := client.Company("123", oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-token"})).Items().Get(context.Background(), "37")
	require.NoError(t, err)
	assert.Equal(t, "37", item.ID())
	assert.Equal(t, EntityItem, item.EntityName())
	assert.Equal(t, "Active", item.CDCStatus())
	assert.Equal(t, "QBO", item.CDCDomain())
	assert.Equal(t, "Office Supplies", item.Name)
	assert.Equal(t, InventoryItemType, item.Type)
}

func TestItemServiceQueryUsesQueryOptions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/company/123/query", r.URL.Path)
		assert.Equal(t, "SELECT * FROM Item STARTPOSITION 11 MAXRESULTS 10", r.URL.Query().Get("query"))
		w.Header().Set("Content-Type", "application/json")
		_, err := io.WriteString(w, `{"QueryResponse":{"Item":[{"Id":"37","SyncToken":"0"},{"Id":"38","SyncToken":"1"}],"startPosition":6,"maxResults":10}}`)
		require.NoError(t, err)
	}))
	defer server.Close()

	client, err := NewClient(WithHTTPClient(server.Client()), WithBaseURL(server.URL+"/v3/company"))
	require.NoError(t, err)

	items, err := client.Company("123", oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-token"})).Items().ListPage(context.Background(), 2, 10)
	require.NoError(t, err)
	require.Len(t, items, 2)
	assert.Equal(t, "37", items[0].ID())
	assert.Equal(t, "38", items[1].ID())
}

func TestItemJSONDecodesInventoryItem(t *testing.T) {
	const fixture = `{
		"Item": {
			"Id": "37",
			"SyncToken": "0",
			"domain": "QBO",
			"status": "Active",
			"MetaData": {
				"CreateTime": "2015-04-22T11:03:23-07:00",
				"LastUpdatedTime": "2015-04-22T11:03:24-07:00"
			},
			"Name": "Office Supplies",
			"FullyQualifiedName": "Office Supplies",
			"Type": "Inventory",
			"TrackQtyOnHand": true,
			"QtyOnHand": 10,
			"InvStartDate": "2013-02-19",
			"UnitPrice": 25,
			"PurchaseCost": 35,
			"Description": "This is the sales description.",
			"PurchaseDesc": "This is the purchasing description.",
			"Taxable": true,
			"Active": true,
			"Level": 0,
			"IncomeAccountRef": {"value": "79", "name": "Sales of Product Income"},
			"AssetAccountRef": {"value": "81", "name": "Inventory Asset"},
			"ExpenseAccountRef": {"value": "80", "name": "Cost of Goods Sold"}
		}
	}`

	var envelope struct {
		Item Item `json:"Item"`
	}
	require.NoError(t, json.Unmarshal([]byte(fixture), &envelope))

	item := envelope.Item
	assert.Equal(t, "37", item.ID())
	assert.Equal(t, "0", item.SyncToken())
	assert.Equal(t, "QBO", item.CDCDomain())
	assert.Equal(t, "Active", item.CDCStatus())

	require.NotNil(t, item.MetaData)
	require.NotNil(t, item.MetaData.CreateTime)

	assert.Equal(t, "Office Supplies", item.Name)
	assert.Equal(t, "Office Supplies", item.FullyQualifiedName)
	assert.Equal(t, InventoryItemType, item.Type)
	assert.Equal(t, 0, item.Level)

	require.NotNil(t, item.TrackQtyOnHand)
	assert.True(t, *item.TrackQtyOnHand)
	assert.Equal(t, "10", item.QtyOnHand.String())

	require.NotNil(t, item.InvStartDate)
	assert.Equal(t, "2013-02-19", item.InvStartDate.String())

	assert.Equal(t, "25", item.UnitPrice.String())
	assert.Equal(t, "35", item.PurchaseCost.String())
	assert.Equal(t, "This is the sales description.", item.Description)
	assert.Equal(t, "This is the purchasing description.", item.PurchaseDesc)

	require.NotNil(t, item.Taxable)
	assert.True(t, *item.Taxable)
	require.NotNil(t, item.Active)
	assert.True(t, *item.Active)

	require.NotNil(t, item.IncomeAccountRef)
	assert.Equal(t, "79", item.IncomeAccountRef.Value)
	require.NotNil(t, item.AssetAccountRef)
	assert.Equal(t, "81", item.AssetAccountRef.Value)
	require.NotNil(t, item.ExpenseAccountRef)
	assert.Equal(t, "80", item.ExpenseAccountRef.Value)
}

func TestItemJSONDecodesServiceItem(t *testing.T) {
	const fixture = `{
		"Item": {
			"Id": "7",
			"SyncToken": "0",
			"domain": "QBO",
			"status": "Active",
			"Name": "Installation",
			"FullyQualifiedName": "Installation",
			"Type": "Service",
			"UnitPrice": 50,
			"Description": "Installation services",
			"Taxable": false,
			"Active": true,
			"IncomeAccountRef": {"value": "1", "name": "Services"}
		}
	}`

	var envelope struct {
		Item Item `json:"Item"`
	}
	require.NoError(t, json.Unmarshal([]byte(fixture), &envelope))

	item := envelope.Item
	assert.Equal(t, "7", item.ID())
	assert.Equal(t, ServiceItemType, item.Type)
	assert.Equal(t, "50", item.UnitPrice.String())
	require.NotNil(t, item.IncomeAccountRef)
	assert.Equal(t, "1", item.IncomeAccountRef.Value)
	// No inventory fields
	assert.Empty(t, item.QtyOnHand.String())
	assert.Nil(t, item.AssetAccountRef)
}

func TestItemJSONDecodesGroupItem(t *testing.T) {
	const fixture = `{
		"Item": {
			"Id": "49",
			"SyncToken": "1",
			"domain": "QBO",
			"status": "Active",
			"Name": "Deluxe Fountain",
			"FullyQualifiedName": "Deluxe Fountain",
			"Type": "Group",
			"Sku": "234",
			"PrintGroupedItems": true,
			"Active": true,
			"ItemGroupDetail": {
				"ItemGroupLine": [
					{"Qty": 1, "ItemRef": {"value": "11", "name": "Pump", "type": "Inventory"}},
					{"Qty": 2, "ItemRef": {"value": "8", "name": "Lighting", "type": "Service"}}
				]
			}
		}
	}`

	var envelope struct {
		Item Item `json:"Item"`
	}
	require.NoError(t, json.Unmarshal([]byte(fixture), &envelope))

	item := envelope.Item
	assert.Equal(t, "49", item.ID())
	assert.Equal(t, GroupItemType, item.Type)
	assert.Equal(t, "234", item.Sku)
	require.NotNil(t, item.PrintGroupedItems)
	assert.True(t, *item.PrintGroupedItems)

	require.NotNil(t, item.ItemGroupDetail)
	require.Len(t, item.ItemGroupDetail.ItemGroupLine, 2)
	assert.Equal(t, "1", item.ItemGroupDetail.ItemGroupLine[0].Qty.String())
	assert.Equal(t, "11", item.ItemGroupDetail.ItemGroupLine[0].ItemRef.Value)
	assert.Equal(t, "Inventory", item.ItemGroupDetail.ItemGroupLine[0].ItemRef.Type)
	assert.Equal(t, "2", item.ItemGroupDetail.ItemGroupLine[1].Qty.String())
	assert.Equal(t, "8", item.ItemGroupDetail.ItemGroupLine[1].ItemRef.Value)
	assert.Equal(t, "Service", item.ItemGroupDetail.ItemGroupLine[1].ItemRef.Type)
}

func TestItemJSONDecodesCategoryItem(t *testing.T) {
	const fixture = `{
		"Item": {
			"Id": "30",
			"SyncToken": "0",
			"domain": "QBO",
			"status": "Active",
			"Name": "Cedar",
			"FullyQualifiedName": "Trees:Cedar",
			"Type": "Category",
			"SubItem": true,
			"Level": 1,
			"Active": true,
			"ParentRef": {"value": "29", "name": "Trees"}
		}
	}`

	var envelope struct {
		Item Item `json:"Item"`
	}
	require.NoError(t, json.Unmarshal([]byte(fixture), &envelope))

	item := envelope.Item
	assert.Equal(t, "30", item.ID())
	assert.Equal(t, CategoryItemType, item.Type)
	assert.Equal(t, "Trees:Cedar", item.FullyQualifiedName)
	assert.Equal(t, 1, item.Level)
	require.NotNil(t, item.SubItem)
	assert.True(t, *item.SubItem)
	require.NotNil(t, item.ParentRef)
	assert.Equal(t, "29", item.ParentRef.Value)
}

func TestItemRequestsMarshalWritablePayloads(t *testing.T) {
	create := Item{
		Name:              "Garden Supplies",
		Type:              InventoryItemType,
		TrackQtyOnHand:    boolPtr(true),
		QtyOnHand:         json.Number("10"),
		InvStartDate:      &Date{Time: time.Date(2015, 1, 1, 0, 0, 0, 0, time.UTC)},
		IncomeAccountRef:  &ReferenceType{Value: "79"},
		AssetAccountRef:   &ReferenceType{Value: "81"},
		ExpenseAccountRef: &ReferenceType{Value: "80"},
	}

	body, err := json.Marshal(create)
	require.NoError(t, err)
	assert.Contains(t, string(body), `"Name":"Garden Supplies"`)
	assert.Contains(t, string(body), `"Type":"Inventory"`)
	assert.Contains(t, string(body), `"TrackQtyOnHand":true`)
	assert.NotContains(t, string(body), `"sparse"`)
	assert.NotContains(t, string(body), `"FullyQualifiedName"`)
}

func TestItemCDCExtraction(t *testing.T) {
	changedSince := time.Date(2026, time.March, 10, 12, 0, 0, 0, time.UTC)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/company/123/cdc", r.URL.Path)
		assert.Equal(t, string(EntityItem), r.URL.Query().Get("entities"))
		assert.Equal(t, changedSince.Format(time.RFC3339), r.URL.Query().Get("changedSince"))
		w.Header().Set("Content-Type", "application/json")
		_, err := io.WriteString(w, `{"CDCResponse":[{"QueryResponse":[{"Item":[{"Id":"37","SyncToken":"1","domain":"QBO","status":"Deleted"}],"startPosition":1,"maxResults":1}]}],"time":"2026-03-10T12:00:00Z"}`)
		require.NoError(t, err)
	}))
	defer server.Close()

	client, err := NewClient(WithHTTPClient(server.Client()), WithBaseURL(server.URL+"/v3/company"))
	require.NoError(t, err)

	items, err := client.Company("123", oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-token"})).Items().ChangeDataCapture(context.Background(), changedSince)
	require.NoError(t, err)
	require.Len(t, items, 1)
	assert.Equal(t, "37", items[0].ID())
	assert.Equal(t, "Deleted", items[0].CDCStatus())
}

// Compile-time interface compliance checks for Item and ItemService.
var (
	_ Entity    = Item{}
	_ CDCEntity = Item{}

	_ Getter[Item]          = (*ItemService)(nil)
	_ Querier[Item]         = (*ItemService)(nil)
	_ Lister[Item]          = (*ItemService)(nil)
	_ Pager[Item]           = (*ItemService)(nil)
	_ Creator[Item]         = (*ItemService)(nil)
	_ Updater[Item]         = (*ItemService)(nil)
	_ SparseUpdater[Item]   = (*ItemService)(nil)
	_ EntityCDCReader[Item] = (*ItemService)(nil)
)

func TestItemServiceUpdateSparseInjectsSparseFlag(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/company/123/item", r.URL.Path)
		var payload map[string]interface{}
		require.NoError(t, json.NewDecoder(r.Body).Decode(&payload))
		assert.Equal(t, true, payload["sparse"])
		w.Header().Set("Content-Type", "application/json")
		_, err := io.WriteString(w, `{"Item":{"Id":"1","SyncToken":"2"}}`)
		require.NoError(t, err)
	}))
	defer server.Close()

	client, err := NewClient(WithHTTPClient(server.Client()), WithBaseURL(server.URL+"/v3/company"))
	require.NoError(t, err)

	item := Item{}
	item.Id = "1"
	item.SyncTokenValue = SyncTokenValue{"1"}

	_, err = client.Company("123", oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-token"})).Items().UpdateSparse(context.Background(), item)
	require.NoError(t, err)
}
