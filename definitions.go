package quickbooks

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

const (
	DateLayout     = "2006-01-02"
	DateTimeLayout = "2006-01-02T15:04:05-07:00"
)

type CustomFieldTypeEnum string

const (
	CustomFieldTypeString  CustomFieldTypeEnum = "StringType"
	CustomFieldTypeBoolean CustomFieldTypeEnum = "BooleanType"
	CustomFieldTypeNumber  CustomFieldTypeEnum = "NumberType"
	CustomFieldTypeDate    CustomFieldTypeEnum = "DateType"
)

type PrintStatusEnum string

const (
	NotSetStatus        PrintStatusEnum = "NotSet"
	NeedToPrintStatus   PrintStatusEnum = "NeedToPrint"
	PrintCompleteStatus PrintStatusEnum = "PrintComplete"
)

type EmailStatusEnum string

const (
	EmailNotSetStatus EmailStatusEnum = "NotSet"
	EmailNeedToSend   EmailStatusEnum = "NeedToSend"
	EmailSent         EmailStatusEnum = "EmailSent"
)

type GlobalTaxCalculationEnum string

const (
	TaxExcludedGlobalTaxCalculation   GlobalTaxCalculationEnum = "TaxExcluded"
	TaxInclusiveGlobalTaxCalculation  GlobalTaxCalculationEnum = "TaxInclusive"
	NotApplicableGlobalTaxCalculation GlobalTaxCalculationEnum = "NotApplicable"
)

type LineDetailType string

const (
	PaymentLineDetailType             LineDetailType = "PaymentLineDetail"
	DiscountLineDetailType            LineDetailType = "DiscountLineDetail"
	TaxLineDetailType                 LineDetailType = "TaxLineDetail"
	SalesItemLineDetailType           LineDetailType = "SalesItemLineDetail"
	DescriptionOnlyLineDetailType     LineDetailType = "DescriptionOnly"
	SubTotalLineDetailType            LineDetailType = "SubTotalLineDetail"
	ItemBasedExpenseLineDetailType    LineDetailType = "ItemBasedExpenseLineDetail"
	AccountBasedExpenseLineDetailType LineDetailType = "AccountBasedExpenseLineDetail"
	DepositLineDetailType             LineDetailType = "DepositLineDetail"
	JournalEntryLineDetailType        LineDetailType = "JournalEntryLineDetail"
	ReimburseLineDetailType           LineDetailType = "ReimburseLineDetail"
	GroupLineDetailType               LineDetailType = "GroupLineDetail"
)

type PostingTypeEnum string

const (
	DebitPostingType  PostingTypeEnum = "Debit"
	CreditPostingType PostingTypeEnum = "Credit"
)

// ItemTypeEnum classifies how an item is used in transactions.
// Group items (bundles) cannot be created via the API — they can only be read
// and updated. Category items require minorversion=4 and are only available on
// companies that have enabled the Categories feature.
type ItemTypeEnum string

const (
	InventoryItemType    ItemTypeEnum = "Inventory"
	ServiceItemType      ItemTypeEnum = "Service"
	NonInventoryItemType ItemTypeEnum = "NonInventory"
	GroupItemType        ItemTypeEnum = "Group"
	CategoryItemType     ItemTypeEnum = "Category"
)

// ItemCategoryTypeEnum classifies an item as a product or service.
// This field is applicable for France locale companies only, available at
// minorversion=3, and is read-only after the item is created.
type ItemCategoryTypeEnum string

const (
	ProductItemCategoryType ItemCategoryTypeEnum = "Product"
	ServiceItemCategoryType ItemCategoryTypeEnum = "Service"
)

type BillableStatusEnum string

const (
	BillableBillableStatus      BillableStatusEnum = "Billable"
	NotBillableBillableStatus   BillableStatusEnum = "NotBillable"
	HasBeenBilledBillableStatus BillableStatusEnum = "HasBeenBilled"
)

type AccountTypeEnum string

const (
	AccountTypeBank               AccountTypeEnum = "Bank"
	AccountTypeOtherCurrentAsset  AccountTypeEnum = "Other Current Asset"
	AccountTypeFixedAsset         AccountTypeEnum = "Fixed Asset"
	AccountTypeOtherAsset         AccountTypeEnum = "Other Asset"
	AccountTypeAccountsReceivable AccountTypeEnum = "Accounts Receivable"
	AccountTypeEquity             AccountTypeEnum = "Equity"
	AccountTypeExpense            AccountTypeEnum = "Expense"
	AccountTypeOtherExpense       AccountTypeEnum = "Other Expense"
	AccountTypeCostOfGoodsSold    AccountTypeEnum = "Cost of Goods Sold"
	AccountTypeAccountsPayable    AccountTypeEnum = "Accounts Payable"
	AccountTypeCreditCard         AccountTypeEnum = "Credit Card"
	AccountTypeLongTermLiability  AccountTypeEnum = "Long Term Liability"
	AccountTypeOtherCurrentLiab   AccountTypeEnum = "Other Current Liability"
	AccountTypeIncome             AccountTypeEnum = "Income"
	AccountTypeOtherIncome        AccountTypeEnum = "Other Income"
)

type NameOfEnum string

const (
	NameOfVendor   NameOfEnum = "Vendor"
	NameOfEmployee NameOfEnum = "Employee"
)

type TermTypeEnum string

const (
	TermTypeStandard   TermTypeEnum = "STANDARD"
	TermTypeDateDriven TermTypeEnum = "DATE_DRIVEN"
)

type Date struct {
	time.Time
}

func (d *Date) UnmarshalJSON(data []byte) error {
	parsed, err := parseTime(data, DateLayout, DateTimeLayout, time.RFC3339Nano)
	if err != nil {
		return err
	}
	d.Time = parsed
	return nil
}

func (d Date) MarshalJSON() ([]byte, error) {
	if d.Time.IsZero() {
		return []byte("null"), nil
	}
	return json.Marshal(d.Time.Format(DateLayout))
}

func (d Date) String() string {
	if d.Time.IsZero() {
		return ""
	}
	return d.Time.Format(DateLayout)
}

type DateTime struct {
	time.Time
}

func (dt *DateTime) UnmarshalJSON(data []byte) error {
	parsed, err := parseTime(data, time.RFC3339Nano, DateTimeLayout, DateLayout)
	if err != nil {
		return err
	}
	dt.Time = parsed
	return nil
}

func (dt DateTime) MarshalJSON() ([]byte, error) {
	if dt.Time.IsZero() {
		return []byte("null"), nil
	}
	return json.Marshal(dt.Time.Format(DateTimeLayout))
}

func parseTime(data []byte, layouts ...string) (time.Time, error) {
	data = bytes.TrimSpace(data)
	if bytes.Equal(data, []byte("null")) {
		return time.Time{}, nil
	}

	var value string
	if err := json.Unmarshal(data, &value); err != nil {
		return time.Time{}, err
	}
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}, nil
	}

	for _, layout := range layouts {
		parsed, err := time.Parse(layout, value)
		if err == nil {
			return parsed, nil
		}
	}

	return time.Time{}, fmt.Errorf("unsupported time format %q", value)
}

type ReferenceType struct {
	Value string `json:"value,omitempty"`
	Name  string `json:"name,omitempty"`
	Type  string `json:"type,omitempty"`
}

// TelephoneNumber holds a free-form phone number string.
type TelephoneNumber struct {
	FreeFormNumber string `json:"FreeFormNumber,omitempty"`
}

// WebsiteAddress holds a URI for a web address.
type WebsiteAddress struct {
	URI string `json:"URI,omitempty"`
}

// ContactInfo holds a typed contact entry, used for additional contact
// information on entities such as Vendor.
type ContactInfo struct {
	Type      string           `json:"Type,omitempty"`
	Telephone *TelephoneNumber `json:"Telephone,omitempty"`
}

// NameValue holds an arbitrary key-value pair, used in CompanyInfo and
// Preferences for settings not covered by the standard field set.
type NameValue struct {
	Name  string `json:"Name,omitempty"`
	Value string `json:"Value,omitempty"`
}

// EmailMessageType holds the subject and message body for a transaction email
// template, used within EmailMessagesPrefs on Preferences.
type EmailMessageType struct {
	Subject string `json:"Subject,omitempty"`
	Message string `json:"Message,omitempty"`
}

type ReportBasisEnum string

const (
	ReportBasisAccrual ReportBasisEnum = "Accrual"
	ReportBasisCash    ReportBasisEnum = "Cash"
)

type WorkWeekStartDateEnum string

const (
	WorkWeekStartSunday    WorkWeekStartDateEnum = "Sunday"
	WorkWeekStartMonday    WorkWeekStartDateEnum = "Monday"
	WorkWeekStartTuesday   WorkWeekStartDateEnum = "Tuesday"
	WorkWeekStartWednesday WorkWeekStartDateEnum = "Wednesday"
	WorkWeekStartThursday  WorkWeekStartDateEnum = "Thursday"
	WorkWeekStartFriday    WorkWeekStartDateEnum = "Friday"
	WorkWeekStartSaturday  WorkWeekStartDateEnum = "Saturday"
)

type MonthEnum string

const (
	MonthJanuary   MonthEnum = "January"
	MonthFebruary  MonthEnum = "February"
	MonthMarch     MonthEnum = "March"
	MonthApril     MonthEnum = "April"
	MonthMay       MonthEnum = "May"
	MonthJune      MonthEnum = "June"
	MonthJuly      MonthEnum = "July"
	MonthAugust    MonthEnum = "August"
	MonthSeptember MonthEnum = "September"
	MonthOctober   MonthEnum = "October"
	MonthNovember  MonthEnum = "November"
	MonthDecember  MonthEnum = "December"
)

type ModificationMetaData struct {
	CreateTime        *DateTime      `json:"CreateTime,omitempty"`
	LastUpdatedTime   *DateTime      `json:"LastUpdatedTime,omitempty"`
	LastModifiedByRef *ReferenceType `json:"LastModifiedByRef,omitempty"`
}

type CustomField struct {
	DefinitionID string              `json:"DefinitionId,omitempty"`
	Name         string              `json:"Name,omitempty"`
	Type         CustomFieldTypeEnum `json:"Type,omitempty"`
	BooleanValue *bool               `json:"BooleanValue,omitempty"`
	DateValue    *Date               `json:"DateValue,omitempty"`
	NumberValue  json.Number         `json:"NumberValue,omitempty"`
	StringValue  string              `json:"StringValue,omitempty"`
}

type MemoRef struct {
	ID    string `json:"id,omitempty"`
	Value string `json:"value,omitempty"`
}

type EmailAddress struct {
	ID      string `json:"Id,omitempty"`
	Address string `json:"Address,omitempty"`
	Default *bool  `json:"Default,omitempty"`
	Tag     string `json:"Tag,omitempty"`
}

type PhysicalAddress struct {
	ID                     string `json:"Id,omitempty"`
	Line1                  string `json:"Line1,omitempty"`
	Line2                  string `json:"Line2,omitempty"`
	Line3                  string `json:"Line3,omitempty"`
	Line4                  string `json:"Line4,omitempty"`
	Line5                  string `json:"Line5,omitempty"`
	City                   string `json:"City,omitempty"`
	Country                string `json:"Country,omitempty"`
	CountryCode            string `json:"CountryCode,omitempty"`
	County                 string `json:"County,omitempty"`
	CountrySubDivisionCode string `json:"CountrySubDivisionCode,omitempty"`
	PostalCode             string `json:"PostalCode,omitempty"`
	PostalCodeSuffix       string `json:"PostalCodeSuffix,omitempty"`
	Lat                    string `json:"Lat,omitempty"`
	Long                   string `json:"Long,omitempty"`
	Tag                    string `json:"Tag,omitempty"`
	Note                   string `json:"Note,omitempty"`
}

type LinkedTxn struct {
	TxnID     string `json:"TxnId,omitempty"`
	TxnType   string `json:"TxnType,omitempty"`
	TxnLineID string `json:"TxnLineId,omitempty"`
}

type MarkupInfo struct {
	Percent                json.Number    `json:"Percent,omitempty"`
	PercentBased           *bool          `json:"PercentBased,omitempty"`
	Value                  json.Number    `json:"Value,omitempty"`
	PriceLevelRef          *ReferenceType `json:"PriceLevelRef,omitempty"`
	MarkUpIncomeAccountRef *ReferenceType `json:"MarkUpIncomeAccountRef,omitempty"`
}

// SalesItemLineDetail holds the detail fields for a SalesItemLine line type.
// All fields are optional per the v75 API schema.
type SalesItemLineDetail struct {
	ItemRef              *ReferenceType `json:"ItemRef,omitempty"`
	ClassRef             *ReferenceType `json:"ClassRef,omitempty"`
	TaxCodeRef           *ReferenceType `json:"TaxCodeRef,omitempty"`
	MarkupInfo           *MarkupInfo    `json:"MarkupInfo,omitempty"`
	ItemAccountRef       *ReferenceType `json:"ItemAccountRef,omitempty"`
	ServiceDate          *Date          `json:"ServiceDate,omitempty"`
	TaxClassificationRef *ReferenceType `json:"TaxClassificationRef,omitempty"`
	Qty                  json.Number    `json:"Qty,omitempty"`
	UnitPrice            json.Number    `json:"UnitPrice,omitempty"`
	TaxInclusiveAmt      json.Number    `json:"TaxInclusiveAmt,omitempty"`
	DiscountRate         json.Number    `json:"DiscountRate,omitempty"`
	DiscountAmt          json.Number    `json:"DiscountAmt,omitempty"`
}

type DescriptionLineDetail struct {
	ServiceDate *Date          `json:"ServiceDate,omitempty"`
	TaxCodeRef  *ReferenceType `json:"TaxCodeRef,omitempty"`
}

// DiscountLineDetail holds the detail fields for a DiscountLine line type.
// All fields are optional per the v75 API schema.
type DiscountLineDetail struct {
	DiscountRef        *ReferenceType `json:"DiscountRef,omitempty"`
	PercentBased       *bool          `json:"PercentBased,omitempty"`
	DiscountPercent    json.Number    `json:"DiscountPercent,omitempty"`
	DiscountAccountRef *ReferenceType `json:"DiscountAccountRef,omitempty"`
	ClassRef           *ReferenceType `json:"ClassRef,omitempty"`
	TaxCodeRef         *ReferenceType `json:"TaxCodeRef,omitempty"`
}

type SubTotalLineDetail struct {
	ItemRef *ReferenceType `json:"ItemRef,omitempty"`
}

// ItemBasedExpenseLineDetail holds the detail fields for an item-based expense
// line on a Bill. All fields are optional per the v75 API schema.
type ItemBasedExpenseLineDetail struct {
	ItemRef         *ReferenceType     `json:"ItemRef,omitempty"`
	ClassRef        *ReferenceType     `json:"ClassRef,omitempty"`
	CustomerRef     *ReferenceType     `json:"CustomerRef,omitempty"`
	PriceLevelRef   *ReferenceType     `json:"PriceLevelRef,omitempty"`
	TaxCodeRef      *ReferenceType     `json:"TaxCodeRef,omitempty"`
	MarkupInfo      *MarkupInfo        `json:"MarkupInfo,omitempty"`
	BillableStatus  BillableStatusEnum `json:"BillableStatus,omitempty"`
	Qty             json.Number        `json:"Qty,omitempty"`
	UnitPrice       json.Number        `json:"UnitPrice,omitempty"`
	TaxInclusiveAmt json.Number        `json:"TaxInclusiveAmt,omitempty"`
}

// AccountBasedExpenseLineDetail holds the detail fields for an account-based
// expense line on a Bill. AccountRef is required; all other fields are optional.
type AccountBasedExpenseLineDetail struct {
	AccountRef      *ReferenceType     `json:"AccountRef,omitempty"`
	ClassRef        *ReferenceType     `json:"ClassRef,omitempty"`
	CustomerRef     *ReferenceType     `json:"CustomerRef,omitempty"`
	TaxCodeRef      *ReferenceType     `json:"TaxCodeRef,omitempty"`
	MarkupInfo      *MarkupInfo        `json:"MarkupInfo,omitempty"`
	BillableStatus  BillableStatusEnum `json:"BillableStatus,omitempty"`
	TaxAmount       json.Number        `json:"TaxAmount,omitempty"`
	TaxInclusiveAmt json.Number        `json:"TaxInclusiveAmt,omitempty"`
}

// CreditChargeInfo holds the charge details submitted with a credit card
// payment. All fields are optional.
type CreditChargeInfo struct {
	NameOnAcct     string      `json:"NameOnAcct,omitempty"`
	CcExpiryMonth  int         `json:"CcExpiryMonth,omitempty"`
	CcExpiryYear   int         `json:"CcExpiryYear,omitempty"`
	BillAddrStreet string      `json:"BillAddrStreet,omitempty"`
	PostalCode     string      `json:"PostalCode,omitempty"`
	Amount         json.Number `json:"Amount,omitempty"`
	Type           string      `json:"Type,omitempty"`
	ProcessPayment *bool       `json:"ProcessPayment,omitempty"`
}

// CreditChargeResponse holds the authorisation result returned by the payment
// processor for a credit card payment. All fields are read-only and optional.
type CreditChargeResponse struct {
	Status               string    `json:"Status,omitempty"`
	AuthCode             string    `json:"AuthCode,omitempty"`
	CCTransId            string    `json:"CCTransId,omitempty"`
	TxnAuthorizationTime *DateTime `json:"TxnAuthorizationTime,omitempty"`
}

type TaxLineDetail struct {
	TaxRateRef          *ReferenceType `json:"TaxRateRef,omitempty"`
	PercentBased        *bool          `json:"PercentBased,omitempty"`
	TaxPercent          json.Number    `json:"TaxPercent,omitempty"`
	NetAmountTaxable    json.Number    `json:"NetAmountTaxable,omitempty"`
	TaxInclusiveAmount  json.Number    `json:"TaxInclusiveAmount,omitempty"`
	OverrideDeltaAmount json.Number    `json:"OverrideDeltaAmount,omitempty"`
}

// TaxLine represents a single tax line within a TxnTaxDetail. The only valid
// detail type for a tax line is TaxLineDetail.
type TaxLine struct {
	Amount        json.Number    `json:"Amount,omitempty"`
	DetailType    LineDetailType `json:"DetailType,omitempty"`
	TaxLineDetail *TaxLineDetail `json:"TaxLineDetail,omitempty"`
}

type TxnTaxDetail struct {
	TxnTaxCodeRef *ReferenceType `json:"TxnTaxCodeRef,omitempty"`
	TotalTax      json.Number    `json:"TotalTax,omitempty"`
	TaxLine       []TaxLine      `json:"TaxLine,omitempty"`
}

// CustomExtensionAssociatedValue holds a single key/value pair within a
// CustomExtension, referencing a Dimensions DefinitionID (Key) and a Dimension
// ValueID (Value). Used with the Intuit Enterprise Suite Dimensions feature.
type CustomExtensionAssociatedValue struct {
	Key   string `json:"Key,omitempty"`
	Value string `json:"Value,omitempty"`
}

// CustomExtension associates additional attributes with an entity. The
// ExtensionType field specifies the kind of extension; the currently supported
// value is "DIMENSION". AssociatedValues contains key/value pairs that
// reference Dimensions definitions. This type is part of the Intuit Enterprise
// Suite feature set.
type CustomExtension struct {
	ExtensionType    string                           `json:"ExtensionType,omitempty"`
	AssociatedValues []CustomExtensionAssociatedValue `json:"AssociatedValues,omitempty"`
}

// DeleteRequest carries the identity fields required to delete a QBO entity.
// Both Id and SyncToken are required by the API.
type DeleteRequest struct {
	IDValue
	SyncTokenValue
}

// VoidRequest carries the identity fields required to void a QBO entity.
// Both Id and SyncToken are required by the API.
type VoidRequest struct {
	IDValue
	SyncTokenValue
}
