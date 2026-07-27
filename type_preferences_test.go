package quickbooks

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
)

func TestPreferencesServiceGet(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/company/123/preferences", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_, err := io.WriteString(w, `{"Preferences":{"Id":"1","SyncToken":"6","ReportPrefs":{"ReportBasis":"Accrual","CalcAgingReportFromTxnDate":false},"AccountingInfoPrefs":{"FirstMonthOfFiscalYear":"January","UseAccountNumbers":true},"TimeTrackingPrefs":{"WorkWeekStartDate":"Monday","BillCustomers":true},"CurrencyPrefs":{"HomeCurrency":{"value":"USD"},"MultiCurrencyEnabled":false}}}`)
		require.NoError(t, err)
	}))
	defer server.Close()

	client, err := NewClient(WithHTTPClient(server.Client()), WithBaseURL(server.URL+"/v3/company"))
	require.NoError(t, err)

	prefs, err := client.Company("123", oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-token"})).Preferences().Get(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "1", prefs.ID())
	assert.Equal(t, "6", prefs.SyncToken())
	assert.Equal(t, EntityPreferences, prefs.EntityName())

	require.NotNil(t, prefs.ReportPrefs)
	assert.Equal(t, ReportBasisAccrual, prefs.ReportPrefs.ReportBasis)
	require.NotNil(t, prefs.ReportPrefs.CalcAgingReportFromTxnDate)
	assert.False(t, *prefs.ReportPrefs.CalcAgingReportFromTxnDate)

	require.NotNil(t, prefs.AccountingInfoPrefs)
	assert.Equal(t, MonthJanuary, prefs.AccountingInfoPrefs.FirstMonthOfFiscalYear)
	require.NotNil(t, prefs.AccountingInfoPrefs.UseAccountNumbers)
	assert.True(t, *prefs.AccountingInfoPrefs.UseAccountNumbers)

	require.NotNil(t, prefs.TimeTrackingPrefs)
	assert.Equal(t, WorkWeekStartMonday, prefs.TimeTrackingPrefs.WorkWeekStartDate)
	require.NotNil(t, prefs.TimeTrackingPrefs.BillCustomers)
	assert.True(t, *prefs.TimeTrackingPrefs.BillCustomers)

	require.NotNil(t, prefs.CurrencyPrefs)
	require.NotNil(t, prefs.CurrencyPrefs.HomeCurrency)
	assert.Equal(t, "USD", prefs.CurrencyPrefs.HomeCurrency.Value)
	require.NotNil(t, prefs.CurrencyPrefs.MultiCurrencyEnabled)
	assert.False(t, *prefs.CurrencyPrefs.MultiCurrencyEnabled)
}

func TestPreferencesServiceUpdateSparseInjectsSparseFlag(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/company/123/preferences", r.URL.Path)
		var payload map[string]interface{}
		require.NoError(t, json.NewDecoder(r.Body).Decode(&payload))
		assert.Equal(t, true, payload["sparse"])
		w.Header().Set("Content-Type", "application/json")
		_, err := io.WriteString(w, `{"Preferences":{"Id":"1","SyncToken":"7"}}`)
		require.NoError(t, err)
	}))
	defer server.Close()

	client, err := NewClient(WithHTTPClient(server.Client()), WithBaseURL(server.URL+"/v3/company"))
	require.NoError(t, err)

	prefs := Preferences{}
	prefs.Id = "1"
	prefs.SyncTokenValue = SyncTokenValue{"6"}
	prefs.SalesFormsPrefs = &SalesFormsPrefs{AutoApplyPayments: boolPtr(true)}

	_, err = client.Company("123", oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-token"})).Preferences().UpdateSparse(context.Background(), prefs)
	require.NoError(t, err)
}

func TestPreferencesJSONDecodesV75Fields(t *testing.T) {
	const fixture = `{
		"Preferences": {
			"Id": "1",
			"SyncToken": "6",
			"MetaData": {
				"CreateTime": "2017-10-25T01:05:43-07:00",
				"LastUpdatedTime": "2018-03-08T13:24:26-08:00"
			},
			"EmailMessagesPrefs": {
				"InvoiceMessage": {
					"Subject": "Invoice from Acme",
					"Message": "Your invoice is attached."
				},
				"EstimateMessage": {
					"Subject": "Estimate from Acme",
					"Message": "Please review the estimate."
				}
			},
			"ProductAndServicesPrefs": {
				"QuantityWithPriceAndRate": true,
				"ForPurchase": true,
				"QuantityOnHand": true,
				"ForSales": true
			},
			"ReportPrefs": {
				"ReportBasis": "Accrual",
				"CalcAgingReportFromTxnDate": false
			},
			"AccountingInfoPrefs": {
				"FirstMonthOfFiscalYear": "January",
				"TaxYearMonth": "January",
				"UseAccountNumbers": true,
				"ClassTrackingPerTxn": false,
				"ClassTrackingPerTxnLine": true,
				"TrackDepartments": true,
				"TaxForm": "6",
				"CustomerTerminology": "Customers",
				"DepartmentTerminology": "Location",
				"BookCloseDate": "2018-12-31"
			},
			"SalesFormsPrefs": {
				"CustomTxnNumbers": false,
				"AllowEstimates": true,
				"AllowDiscount": true,
				"AllowDeposit": true,
				"AutoApplyPayments": true,
				"AutoApplyCredit": true,
				"DefaultCustomerMessage": "Thank you!",
				"DefaultTerms": {"value": "3"},
				"ETransactionEnabledStatus": "NotApplicable"
			},
			"VendorAndPurchasesPrefs": {
				"BillableExpenseTracking": true,
				"TrackingByCustomer": true
			},
			"TaxPrefs": {
				"TaxGroupCodeRef": {"value": "2"},
				"UsingSalesTax": true
			},
			"OtherPrefs": {
				"NameValue": [
					{"Name": "SalesFormsPrefs.DefaultCustomerMessage", "Value": "Thank you!"}
				]
			},
			"TimeTrackingPrefs": {
				"WorkWeekStartDate": "Monday",
				"MarkTimeEntriesBillable": true,
				"ShowBillRateToAll": false,
				"UseServices": true,
				"BillCustomers": true
			},
			"CurrencyPrefs": {
				"HomeCurrency": {"value": "USD"},
				"MultiCurrencyEnabled": false
			}
		}
	}`

	var envelope struct {
		Preferences Preferences `json:"Preferences"`
	}
	require.NoError(t, json.Unmarshal([]byte(fixture), &envelope))

	p := envelope.Preferences
	assert.Equal(t, "1", p.ID())
	assert.Equal(t, "6", p.SyncToken())
	require.NotNil(t, p.MetaData)

	require.NotNil(t, p.EmailMessagesPrefs)
	require.NotNil(t, p.EmailMessagesPrefs.InvoiceMessage)
	assert.Equal(t, "Invoice from Acme", p.EmailMessagesPrefs.InvoiceMessage.Subject)
	assert.Equal(t, "Your invoice is attached.", p.EmailMessagesPrefs.InvoiceMessage.Message)
	require.NotNil(t, p.EmailMessagesPrefs.EstimateMessage)
	assert.Equal(t, "Estimate from Acme", p.EmailMessagesPrefs.EstimateMessage.Subject)

	require.NotNil(t, p.ProductAndServicesPrefs)
	require.NotNil(t, p.ProductAndServicesPrefs.ForSales)
	assert.True(t, *p.ProductAndServicesPrefs.ForSales)

	require.NotNil(t, p.ReportPrefs)
	assert.Equal(t, ReportBasisAccrual, p.ReportPrefs.ReportBasis)

	require.NotNil(t, p.AccountingInfoPrefs)
	assert.Equal(t, MonthJanuary, p.AccountingInfoPrefs.FirstMonthOfFiscalYear)
	assert.Equal(t, MonthJanuary, p.AccountingInfoPrefs.TaxYearMonth)
	require.NotNil(t, p.AccountingInfoPrefs.BookCloseDate)
	assert.Equal(t, "2018-12-31", p.AccountingInfoPrefs.BookCloseDate.String())

	require.NotNil(t, p.SalesFormsPrefs)
	require.NotNil(t, p.SalesFormsPrefs.AllowEstimates)
	assert.True(t, *p.SalesFormsPrefs.AllowEstimates)
	require.NotNil(t, p.SalesFormsPrefs.DefaultTerms)
	assert.Equal(t, "3", p.SalesFormsPrefs.DefaultTerms.Value)
	assert.Equal(t, ETransactionNotApplicable, p.SalesFormsPrefs.ETransactionEnabledStatus)

	require.NotNil(t, p.VendorAndPurchasesPrefs)
	require.NotNil(t, p.VendorAndPurchasesPrefs.BillableExpenseTracking)
	assert.True(t, *p.VendorAndPurchasesPrefs.BillableExpenseTracking)

	require.NotNil(t, p.TaxPrefs)
	require.NotNil(t, p.TaxPrefs.TaxGroupCodeRef)
	assert.Equal(t, "2", p.TaxPrefs.TaxGroupCodeRef.Value)
	require.NotNil(t, p.TaxPrefs.UsingSalesTax)
	assert.True(t, *p.TaxPrefs.UsingSalesTax)

	require.NotNil(t, p.OtherPrefs)
	require.Len(t, p.OtherPrefs.NameValue, 1)
	assert.Equal(t, "SalesFormsPrefs.DefaultCustomerMessage", p.OtherPrefs.NameValue[0].Name)

	require.NotNil(t, p.TimeTrackingPrefs)
	assert.Equal(t, WorkWeekStartMonday, p.TimeTrackingPrefs.WorkWeekStartDate)
	require.NotNil(t, p.TimeTrackingPrefs.MarkTimeEntriesBillable)
	assert.True(t, *p.TimeTrackingPrefs.MarkTimeEntriesBillable)

	require.NotNil(t, p.CurrencyPrefs)
	require.NotNil(t, p.CurrencyPrefs.HomeCurrency)
	assert.Equal(t, "USD", p.CurrencyPrefs.HomeCurrency.Value)
	require.NotNil(t, p.CurrencyPrefs.MultiCurrencyEnabled)
	assert.False(t, *p.CurrencyPrefs.MultiCurrencyEnabled)
}

// Compile-time interface compliance checks for Preferences and PreferencesService.
// Preferences.Get does not satisfy Getter[T] by design — the path is fixed
// and the entity ID is not accepted as a parameter.
var (
	_ Entity    = Preferences{}
	_ CDCEntity = Preferences{}

	_ Querier[Preferences]         = (*PreferencesService)(nil)
	_ Lister[Preferences]          = (*PreferencesService)(nil)
	_ Updater[Preferences]         = (*PreferencesService)(nil)
	_ SparseUpdater[Preferences]   = (*PreferencesService)(nil)
	_ EntityCDCReader[Preferences] = (*PreferencesService)(nil)
)
