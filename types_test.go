package quickbooks

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLastUpdatedReadsMetaData(t *testing.T) {
	var inv Invoice
	require.NoError(t, json.Unmarshal([]byte(`{"Id":"17","SyncToken":"3","MetaData":{"CreateTime":"2026-10-01T09:00:00-07:00","LastUpdatedTime":"2026-10-14T15:04:05-07:00"}}`), &inv))

	want := time.Date(2026, 10, 14, 15, 4, 5, 0, time.FixedZone("", -7*60*60))
	assert.True(t, inv.LastUpdated().Equal(want), "LastUpdated = %v, want %v", inv.LastUpdated(), want)
	// The field stays where it was.
	require.NotNil(t, inv.MetaData)
	assert.True(t, inv.MetaData.CreateTime.Equal(time.Date(2026, 10, 1, 9, 0, 0, 0, time.FixedZone("", -7*60*60))))
}

// A record deleted since a CDC call's changedSince carries the time of its
// deletion.
func TestLastUpdatedOfADeletedCDCRecord(t *testing.T) {
	var inv Invoice
	require.NoError(t, json.Unmarshal([]byte(`{"domain":"QBO","status":"Deleted","Id":"17","MetaData":{"LastUpdatedTime":"2026-10-14T15:04:05-07:00"}}`), &inv))
	assert.Equal(t, "Deleted", inv.CDCStatus())
	assert.False(t, inv.LastUpdated().IsZero())
}

func TestLastUpdatedWithoutMetaDataIsZero(t *testing.T) {
	assert.True(t, Invoice{}.LastUpdated().IsZero())

	var inv Invoice
	require.NoError(t, json.Unmarshal([]byte(`{"Id":"17","MetaData":{}}`), &inv))
	assert.True(t, inv.LastUpdated().IsZero())
}

// Every entity says when QBO last changed it.
var _ = []Entity{Invoice{}, Bill{}, Item{}, TaxCode{}, TimeActivity{}, Customer{}}
