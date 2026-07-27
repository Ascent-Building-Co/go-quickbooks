package quickbooks

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

// EventType represents the type of change that occurred on an entity.
type EventType string

const (
	Created EventType = "created"
	Deleted EventType = "deleted"
	Merged  EventType = "merged"
	Updated EventType = "updated"
	Void    EventType = "void"
	Emailed EventType = "emailed"
)

// knownEventTypes maps lowercase event strings to their EventType constant.
var knownEventTypes = map[string]EventType{
	"created": Created,
	"deleted": Deleted,
	"merged":  Merged,
	"updated": Updated,
	"void":    Void,
	"emailed": Emailed,
}

// webhookEntityNames maps the lowercase entity name as it appears in the
// webhook type field (e.g. "billpayment") to the SDK EntityName constant.
var webhookEntityNames = map[string]EntityName{
	"account":       EntityAccount,
	"bill":          EntityBill,
	"billpayment":   EntityBillPayment,
	"class":         EntityClass,
	"creditmemo":    EntityCreditMemo,
	"customer":      EntityCustomer,
	"department":    EntityDepartment,
	"deposit":       EntityDeposit,
	"employee":      EntityEmployee,
	"estimate":      EntityEstimate,
	"invoice":       EntityInvoice,
	"item":          EntityItem,
	"journalentry":  EntityJournalEntry,
	"payment":       EntityPayment,
	"paymentmethod": EntityPaymentMethod,
	"preferences":   EntityPreferences,
	"purchase":      EntityPurchase,
	"purchaseorder": EntityPurchaseOrder,
	"taxagency":     EntityTaxAgency,
	"term":          EntityTerm,
	"timeactivity":  EntityTimeActivity,
	"transfer":      EntityTransfer,
	"vendor":        EntityVendor,
	"vendorcredit":  EntityVendorCredit,
}

// WebhookItem represents a single CloudEvents v1.0 webhook event from
// QuickBooks Online.
type WebhookItem struct {
	Specversion     string         `json:"specversion"`
	ID              string         `json:"id"`
	Source          string         `json:"source"`
	Type            string         `json:"type"`
	Datacontenttype string         `json:"datacontenttype"`
	Time            time.Time      `json:"time"`
	Intuitentityid  string         `json:"intuitentityid"`
	Intuitaccountid string         `json:"intuitaccountid"`
	Data            map[string]any `json:"data"`
}

// WebhookNotification is a collection of webhook events received in a single
// notification payload from QuickBooks.
type WebhookNotification []WebhookItem

// Webhook signature verification errors.
var (
	ErrWebhookVerifierTokenEmpty = errors.New("webhook verifier token is empty")
	ErrWebhookSignatureEmpty     = errors.New("webhook signature header is empty")
	ErrWebhookSignatureDecode    = errors.New("webhook signature is not valid base64")
	ErrWebhookSignatureMismatch  = errors.New("webhook signature does not match payload")
)

// VerifyWebhookSignature validates that a webhook payload was sent by Intuit.
// It computes HMAC-SHA256 of the payload using the verifier token and compares
// it (constant-time) against the base64-encoded intuit-signature header value.
// Returns nil if the signature is valid.
func VerifyWebhookSignature(verifierToken string, payload []byte, signature string) error {
	if verifierToken == "" {
		return ErrWebhookVerifierTokenEmpty
	}
	if signature == "" {
		return ErrWebhookSignatureEmpty
	}

	sigBytes, err := base64.StdEncoding.DecodeString(signature)
	if err != nil {
		return ErrWebhookSignatureDecode
	}

	mac := hmac.New(sha256.New, []byte(verifierToken))
	mac.Write(payload)
	expected := mac.Sum(nil)

	if !hmac.Equal(sigBytes, expected) {
		return ErrWebhookSignatureMismatch
	}

	return nil
}

// ParseWebhookNotification unmarshals a raw webhook payload into a
// WebhookNotification ([]WebhookItem).
func ParseWebhookNotification(payload []byte) (WebhookNotification, error) {
	var notification WebhookNotification
	if err := json.Unmarshal(payload, &notification); err != nil {
		return nil, err
	}
	return notification, nil
}

// parseType splits the CloudEvents type field (e.g. "qbo.customer.created.v1")
// into its entity and event components. Returns ("", "", false) if the format
// is not recognized.
func parseType(typ string) (entity, event string, ok bool) {
	// Expected format: qbo.<entity>.<event>.v1
	parts := strings.Split(typ, ".")
	if len(parts) < 4 || parts[0] != "qbo" {
		return "", "", false
	}
	return parts[1], parts[2], true
}

// EventType parses the event type from the CloudEvents type field.
// The second return value indicates whether the event type is recognized
// by the SDK.
func (w WebhookItem) EventType() (EventType, bool) {
	_, event, ok := parseType(w.Type)
	if !ok {
		return "", false
	}
	et, known := knownEventTypes[event]
	return et, known
}

// EntityName parses the entity name from the CloudEvents type field.
// The second return value indicates whether the entity is recognized
// by the SDK.
func (w WebhookItem) EntityName() (EntityName, bool) {
	entity, _, ok := parseType(w.Type)
	if !ok {
		return "", false
	}
	en, known := webhookEntityNames[entity]
	return en, known
}

// ParsedTime returns the time field. Since Time is already a time.Time,
// this is a convenience accessor that matches the pre-existing API shape.
func (w WebhookItem) ParsedTime() (time.Time, error) {
	return w.Time, nil
}

// RealmID returns the QuickBooks company ID (realm ID) associated with this
// webhook event.
func (w WebhookItem) RealmID() string {
	return w.Intuitaccountid
}

// EntityID returns the ID of the entity that was changed.
func (w WebhookItem) EntityID() string {
	return w.Intuitentityid
}
