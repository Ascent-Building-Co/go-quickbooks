package quickbooks

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// computeSignature is a test helper that produces a valid intuit-signature
// header value for the given payload and verifier token.
func computeSignature(token string, payload []byte) string {
	mac := hmac.New(sha256.New, []byte(token))
	mac.Write(payload)
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

func TestVerifyWebhookSignature(t *testing.T) {
	token := "test-verifier-token-abc123"
	payload := []byte(`[{"specversion":"1.0","type":"qbo.customer.created.v1"}]`)
	validSig := computeSignature(token, payload)

	t.Run("valid signature", func(t *testing.T) {
		err := VerifyWebhookSignature(token, payload, validSig)
		assert.NoError(t, err)
	})

	t.Run("empty verifier token", func(t *testing.T) {
		err := VerifyWebhookSignature("", payload, validSig)
		assert.ErrorIs(t, err, ErrWebhookVerifierTokenEmpty)
	})

	t.Run("empty signature", func(t *testing.T) {
		err := VerifyWebhookSignature(token, payload, "")
		assert.ErrorIs(t, err, ErrWebhookSignatureEmpty)
	})

	t.Run("malformed base64 signature", func(t *testing.T) {
		err := VerifyWebhookSignature(token, payload, "not-valid-base64!!!")
		assert.ErrorIs(t, err, ErrWebhookSignatureDecode)
	})

	t.Run("wrong signature", func(t *testing.T) {
		wrongSig := computeSignature("wrong-token", payload)
		err := VerifyWebhookSignature(token, payload, wrongSig)
		assert.ErrorIs(t, err, ErrWebhookSignatureMismatch)
	})

	t.Run("tampered payload", func(t *testing.T) {
		tampered := []byte(`[{"specversion":"1.0","type":"qbo.customer.deleted.v1"}]`)
		err := VerifyWebhookSignature(token, tampered, validSig)
		assert.ErrorIs(t, err, ErrWebhookSignatureMismatch)
	})
}

func TestParseWebhookNotification(t *testing.T) {
	t.Run("valid payload", func(t *testing.T) {
		payload := []byte(`[
			{
				"specversion": "1.0",
				"id": "88cd52aa-33b6-4351-9aa4-47572edbd068",
				"source": "intuit.dsnBgbseACLLRZNxo2dfc4evmEJdxde58xeeYcZliOU=",
				"type": "qbo.customer.created.v1",
				"datacontenttype": "application/json",
				"time": "2025-09-10T21:31:25.179Z",
				"intuitentityid": "1234",
				"intuitaccountid": "310687",
				"data": {}
			}
		]`)

		notification, err := ParseWebhookNotification(payload)
		require.NoError(t, err)
		require.Len(t, notification, 1)
		assert.Equal(t, "1.0", notification[0].Specversion)
		assert.Equal(t, "88cd52aa-33b6-4351-9aa4-47572edbd068", notification[0].ID)
		assert.Equal(t, "qbo.customer.created.v1", notification[0].Type)
		assert.Equal(t, "1234", notification[0].Intuitentityid)
		assert.Equal(t, "310687", notification[0].Intuitaccountid)
	})

	t.Run("multiple events", func(t *testing.T) {
		payload := []byte(`[
			{"specversion":"1.0","type":"qbo.customer.created.v1","intuitentityid":"1","intuitaccountid":"100"},
			{"specversion":"1.0","type":"qbo.invoice.updated.v1","intuitentityid":"2","intuitaccountid":"200"}
		]`)

		notification, err := ParseWebhookNotification(payload)
		require.NoError(t, err)
		assert.Len(t, notification, 2)
	})

	t.Run("invalid json", func(t *testing.T) {
		_, err := ParseWebhookNotification([]byte(`not json`))
		assert.Error(t, err)
	})
}

func TestWebhookItemEventType(t *testing.T) {
	tests := []struct {
		name     string
		typ      string
		wantET   EventType
		wantKnow bool
	}{
		{"created", "qbo.customer.created.v1", Created, true},
		{"deleted", "qbo.invoice.deleted.v1", Deleted, true},
		{"merged", "qbo.vendor.merged.v1", Merged, true},
		{"updated", "qbo.bill.updated.v1", Updated, true},
		{"void", "qbo.payment.void.v1", Void, true},
		{"emailed", "qbo.estimate.emailed.v1", Emailed, true},
		{"unknown event", "qbo.customer.archived.v1", "", false},
		{"malformed type - too few parts", "qbo.customer", "", false},
		{"malformed type - wrong prefix", "other.customer.created.v1", "", false},
		{"empty type", "", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			item := WebhookItem{Type: tt.typ}
			got, known := item.EventType()
			assert.Equal(t, tt.wantET, got)
			assert.Equal(t, tt.wantKnow, known)
		})
	}
}

func TestWebhookItemEntityName(t *testing.T) {
	tests := []struct {
		name     string
		typ      string
		wantEN   EntityName
		wantKnow bool
	}{
		{"customer", "qbo.customer.created.v1", EntityCustomer, true},
		{"invoice", "qbo.invoice.updated.v1", EntityInvoice, true},
		{"bill payment", "qbo.billpayment.created.v1", EntityBillPayment, true},
		{"journal entry", "qbo.journalentry.updated.v1", EntityJournalEntry, true},
		{"credit memo", "qbo.creditmemo.deleted.v1", EntityCreditMemo, true},
		{"time activity", "qbo.timeactivity.created.v1", EntityTimeActivity, true},
		{"vendor credit", "qbo.vendorcredit.updated.v1", EntityVendorCredit, true},
		{"unknown entity", "qbo.unknownentity.created.v1", "", false},
		{"malformed type", "invalid", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			item := WebhookItem{Type: tt.typ}
			got, known := item.EntityName()
			assert.Equal(t, tt.wantEN, got)
			assert.Equal(t, tt.wantKnow, known)
		})
	}
}

func TestWebhookItemRealmID(t *testing.T) {
	item := WebhookItem{Intuitaccountid: "310687"}
	assert.Equal(t, "310687", item.RealmID())
}

func TestWebhookItemEntityID(t *testing.T) {
	item := WebhookItem{Intuitentityid: "1234"}
	assert.Equal(t, "1234", item.EntityID())
}
