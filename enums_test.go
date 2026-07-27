package quickbooks

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// assertUniqueNonEmpty verifies that the given slice contains no duplicate or
// empty values. T is constrained to ~string so the helper works with all
// string-backed enum types declared in this package.
func assertUniqueNonEmpty[T ~string](t *testing.T, values []T, label string) {
	t.Helper()
	seen := make(map[T]struct{}, len(values))
	for _, v := range values {
		assert.NotEmpty(t, string(v), "%s contained an empty value", label)
		_, dup := seen[v]
		assert.Falsef(t, dup, "%s contained duplicate value %q", label, v)
		seen[v] = struct{}{}
	}
}

// assertUniqueNonZero verifies that the given slice contains no duplicate or
// zero values. Used for int-backed enums where the zero value is reserved for
// "unset".
func assertUniqueNonZero[T ~int](t *testing.T, values []T, label string) {
	t.Helper()
	seen := make(map[T]struct{}, len(values))
	for _, v := range values {
		assert.NotZerof(t, int(v), "%s contained a zero value", label)
		_, dup := seen[v]
		assert.Falsef(t, dup, "%s contained duplicate value %d", label, v)
		seen[v] = struct{}{}
	}
}

func TestEnumValuesReturnExpectedSets(t *testing.T) {
	assert.ElementsMatch(t,
		[]CustomFieldTypeEnum{
			CustomFieldTypeString, CustomFieldTypeBoolean,
			CustomFieldTypeNumber, CustomFieldTypeDate,
		},
		CustomFieldTypeValues(),
	)

	assert.ElementsMatch(t,
		[]PrintStatusEnum{NotSetStatus, NeedToPrintStatus, PrintCompleteStatus},
		PrintStatusValues(),
	)

	assert.ElementsMatch(t,
		[]EmailStatusEnum{EmailNotSetStatus, EmailNeedToSend, EmailSent},
		EmailStatusValues(),
	)

	assert.ElementsMatch(t,
		[]GlobalTaxCalculationEnum{
			TaxExcludedGlobalTaxCalculation,
			TaxInclusiveGlobalTaxCalculation,
			NotApplicableGlobalTaxCalculation,
		},
		GlobalTaxCalculationValues(),
	)

	assert.ElementsMatch(t,
		[]PostingTypeEnum{DebitPostingType, CreditPostingType},
		PostingTypeValues(),
	)

	assert.ElementsMatch(t,
		[]ItemTypeEnum{
			InventoryItemType, ServiceItemType, NonInventoryItemType,
			GroupItemType, CategoryItemType,
		},
		ItemTypeValues(),
	)

	assert.ElementsMatch(t,
		[]BillableStatusEnum{
			BillableBillableStatus, NotBillableBillableStatus, HasBeenBilledBillableStatus,
		},
		BillableStatusValues(),
	)

	assert.ElementsMatch(t,
		[]NameOfEnum{NameOfVendor, NameOfEmployee},
		NameOfValues(),
	)

	assert.ElementsMatch(t,
		[]TermTypeEnum{TermTypeStandard, TermTypeDateDriven},
		TermTypeValues(),
	)

	assert.ElementsMatch(t,
		[]ReportBasisEnum{ReportBasisAccrual, ReportBasisCash},
		ReportBasisValues(),
	)

	assert.ElementsMatch(t,
		[]ETransactionEnabledStatusEnum{ETransactionEnabled, ETransactionNotApplicable},
		ETransactionEnabledStatusValues(),
	)

	assert.ElementsMatch(t,
		[]AccountClassificationEnum{
			AccountClassificationAsset,
			AccountClassificationEquity,
			AccountClassificationExpense,
			AccountClassificationLiability,
			AccountClassificationRevenue,
		},
		AccountClassificationValues(),
	)

	assert.ElementsMatch(t,
		[]BillPaymentTypeEnum{CheckBillPaymentType, CreditCardBillPaymentType},
		BillPaymentTypeValues(),
	)

	assert.ElementsMatch(t,
		[]PurchasePaymentTypeEnum{
			CashPurchasePaymentType,
			CheckPurchasePaymentType,
			CreditCardPurchasePaymentType,
		},
		PurchasePaymentTypeValues(),
	)

	assert.ElementsMatch(t,
		[]EstimateTxnStatusEnum{
			EstimatePending, EstimateAccepted, EstimateRejected,
			EstimateClosed, EstimateConverted,
		},
		EstimateTxnStatusValues(),
	)

	assert.ElementsMatch(t,
		[]AttachableCategoryEnum{
			ContactPhotoAttachableCategory,
			DocumentAttachableCategory,
			ImageAttachableCategory,
			ReceiptAttachableCategory,
			SignatureAttachableCategory,
			SoundAttachableCategory,
			OtherAttachableCategory,
		},
		AttachableCategoryValues(),
	)

	assert.Equal(t,
		[]TaxExemptionReasonEnum{
			TaxExemptionReasonFederalGovernment,
			TaxExemptionReasonStateGovernment,
			TaxExemptionReasonLocalGovernment,
			TaxExemptionReasonTribalGovernment,
			TaxExemptionReasonCharitableOrganization,
			TaxExemptionReasonReligiousOrganization,
			TaxExemptionReasonEducationalOrganization,
			TaxExemptionReasonHospital,
			TaxExemptionReasonResale,
			TaxExemptionReasonDirectPayPermit,
			TaxExemptionReasonMultiplePointsOfUse,
			TaxExemptionReasonDirectMail,
			TaxExemptionReasonAgriculturalProduction,
			TaxExemptionReasonIndustrialProduction,
			TaxExemptionReasonForeignDiplomat,
		},
		TaxExemptionReasonValues(),
	)
}

func TestEnumValuesAreUniqueAndNonEmpty(t *testing.T) {
	assertUniqueNonEmpty(t, CustomFieldTypeValues(), "CustomFieldTypeValues")
	assertUniqueNonEmpty(t, PrintStatusValues(), "PrintStatusValues")
	assertUniqueNonEmpty(t, EmailStatusValues(), "EmailStatusValues")
	assertUniqueNonEmpty(t, GlobalTaxCalculationValues(), "GlobalTaxCalculationValues")
	assertUniqueNonEmpty(t, PostingTypeValues(), "PostingTypeValues")
	assertUniqueNonEmpty(t, ItemTypeValues(), "ItemTypeValues")
	assertUniqueNonEmpty(t, BillableStatusValues(), "BillableStatusValues")
	assertUniqueNonEmpty(t, AccountTypeValues(), "AccountTypeValues")
	assertUniqueNonEmpty(t, AccountClassificationValues(), "AccountClassificationValues")
	assertUniqueNonEmpty(t, NameOfValues(), "NameOfValues")
	assertUniqueNonEmpty(t, TermTypeValues(), "TermTypeValues")
	assertUniqueNonEmpty(t, ReportBasisValues(), "ReportBasisValues")
	assertUniqueNonEmpty(t, WorkWeekStartDateValues(), "WorkWeekStartDateValues")
	assertUniqueNonEmpty(t, MonthValues(), "MonthValues")
	assertUniqueNonEmpty(t, ETransactionEnabledStatusValues(), "ETransactionEnabledStatusValues")
	assertUniqueNonEmpty(t, AttachableCategoryValues(), "AttachableCategoryValues")
	assertUniqueNonEmpty(t, BillPaymentTypeValues(), "BillPaymentTypeValues")
	assertUniqueNonEmpty(t, PurchasePaymentTypeValues(), "PurchasePaymentTypeValues")
	assertUniqueNonEmpty(t, EstimateTxnStatusValues(), "EstimateTxnStatusValues")
	assertUniqueNonEmpty(t, AccountSubTypeValues(), "AccountSubTypeValues")
	assertUniqueNonZero(t, TaxExemptionReasonValues(), "TaxExemptionReasonValues")
}

func TestTaxExemptionReasonLabels(t *testing.T) {
	tests := []struct {
		value TaxExemptionReasonEnum
		label string
	}{
		{TaxExemptionReasonFederalGovernment, "Federal government"},
		{TaxExemptionReasonStateGovernment, "State government"},
		{TaxExemptionReasonLocalGovernment, "Local government"},
		{TaxExemptionReasonTribalGovernment, "Tribal government"},
		{TaxExemptionReasonCharitableOrganization, "Charitable organization"},
		{TaxExemptionReasonReligiousOrganization, "Religious organization"},
		{TaxExemptionReasonEducationalOrganization, "Educational organization"},
		{TaxExemptionReasonHospital, "Hospital"},
		{TaxExemptionReasonResale, "Resale"},
		{TaxExemptionReasonDirectPayPermit, "Direct pay permit"},
		{TaxExemptionReasonMultiplePointsOfUse, "Multiple points of use"},
		{TaxExemptionReasonDirectMail, "Direct mail"},
		{TaxExemptionReasonAgriculturalProduction, "Agricultural production"},
		{TaxExemptionReasonIndustrialProduction, "Industrial production / manufacturing"},
		{TaxExemptionReasonForeignDiplomat, "Foreign diplomat"},
	}

	for _, tt := range tests {
		t.Run(tt.label, func(t *testing.T) {
			assert.Equal(t, tt.label, tt.value.Label())
		})
	}

	// Every value returned by Values() must have a non-empty label.
	for _, v := range TaxExemptionReasonValues() {
		assert.NotEmptyf(t, v.Label(), "TaxExemptionReasonEnum(%d) has empty label", v)
	}

	// Unknown values return "".
	assert.Empty(t, TaxExemptionReasonEnum(0).Label())
	assert.Empty(t, TaxExemptionReasonEnum(99).Label())
}

func TestTaxExemptionReasonValuesReturnsIndependentCopy(t *testing.T) {
	a := TaxExemptionReasonValues()
	require.NotEmpty(t, a)
	a[0] = TaxExemptionReasonEnum(99)
	b := TaxExemptionReasonValues()
	assert.Equal(t, TaxExemptionReasonFederalGovernment, b[0])
}

func TestMonthValuesInCalendarOrder(t *testing.T) {
	expected := []MonthEnum{
		MonthJanuary, MonthFebruary, MonthMarch, MonthApril,
		MonthMay, MonthJune, MonthJuly, MonthAugust,
		MonthSeptember, MonthOctober, MonthNovember, MonthDecember,
	}
	assert.Equal(t, expected, MonthValues())
}

func TestWorkWeekStartDateValuesInWeekOrder(t *testing.T) {
	expected := []WorkWeekStartDateEnum{
		WorkWeekStartSunday, WorkWeekStartMonday, WorkWeekStartTuesday,
		WorkWeekStartWednesday, WorkWeekStartThursday, WorkWeekStartFriday,
		WorkWeekStartSaturday,
	}
	assert.Equal(t, expected, WorkWeekStartDateValues())
}

func TestValuesHelpersReturnIndependentCopies(t *testing.T) {
	a := AccountTypeValues()
	require.NotEmpty(t, a)
	a[0] = AccountTypeEnum("mutated")
	b := AccountTypeValues()
	assert.Equal(t, AccountTypeBank, b[0])

	subs := AccountSubTypesFor(AccountTypeBank)
	require.NotEmpty(t, subs)
	subs[0] = AccountSubTypeEnum("mutated")
	again := AccountSubTypesFor(AccountTypeBank)
	assert.Equal(t, AccountSubTypeCashOnHand, again[0])
}

func TestAccountSubTypesForKnownTypes(t *testing.T) {
	tests := []struct {
		name        string
		accountType AccountTypeEnum
		mustContain []AccountSubTypeEnum
	}{
		{
			name:        "bank",
			accountType: AccountTypeBank,
			mustContain: []AccountSubTypeEnum{
				AccountSubTypeChecking, AccountSubTypeSavings,
				AccountSubTypeCashOnHand, AccountSubTypeMoneyMarket,
			},
		},
		{
			name:        "accounts payable",
			accountType: AccountTypeAccountsPayable,
			mustContain: []AccountSubTypeEnum{AccountSubTypeAccountsPayable},
		},
		{
			name:        "credit card",
			accountType: AccountTypeCreditCard,
			mustContain: []AccountSubTypeEnum{AccountSubTypeCreditCard},
		},
		{
			name:        "income",
			accountType: AccountTypeIncome,
			mustContain: []AccountSubTypeEnum{
				AccountSubTypeSalesOfProductIncome, AccountSubTypeServiceFeeIncome,
			},
		},
		{
			name:        "fixed asset",
			accountType: AccountTypeFixedAsset,
			mustContain: []AccountSubTypeEnum{
				AccountSubTypeFurnitureAndFixtures, AccountSubTypeBuildings,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := AccountSubTypesFor(tt.accountType)
			require.NotEmpty(t, got)
			for _, want := range tt.mustContain {
				assert.Containsf(t, got, want, "AccountSubTypesFor(%q) missing %q", tt.accountType, want)
			}
		})
	}
}

func TestAccountSubTypesForUnknownReturnsNil(t *testing.T) {
	got := AccountSubTypesFor(AccountTypeEnum("Bogus"))
	assert.Nil(t, got)
}

func TestAccountSubTypesForCoversAllAccountTypes(t *testing.T) {
	for _, parent := range AccountTypeValues() {
		got := AccountSubTypesFor(parent)
		assert.NotEmptyf(t, got, "AccountSubTypesFor(%q) returned empty slice", parent)
	}
}

func TestAccountClassificationsForKnownTypes(t *testing.T) {
	tests := []struct {
		accountType AccountTypeEnum
		want        AccountClassificationEnum
	}{
		{AccountTypeBank, AccountClassificationAsset},
		{AccountTypeOtherCurrentAsset, AccountClassificationAsset},
		{AccountTypeFixedAsset, AccountClassificationAsset},
		{AccountTypeOtherAsset, AccountClassificationAsset},
		{AccountTypeAccountsReceivable, AccountClassificationAsset},
		{AccountTypeEquity, AccountClassificationEquity},
		{AccountTypeExpense, AccountClassificationExpense},
		{AccountTypeOtherExpense, AccountClassificationExpense},
		{AccountTypeCostOfGoodsSold, AccountClassificationExpense},
		{AccountTypeAccountsPayable, AccountClassificationLiability},
		{AccountTypeCreditCard, AccountClassificationLiability},
		{AccountTypeLongTermLiability, AccountClassificationLiability},
		{AccountTypeOtherCurrentLiab, AccountClassificationLiability},
		{AccountTypeIncome, AccountClassificationRevenue},
		{AccountTypeOtherIncome, AccountClassificationRevenue},
	}

	for _, tt := range tests {
		t.Run(string(tt.accountType), func(t *testing.T) {
			got := AccountClassificationsFor(tt.accountType)
			require.Len(t, got, 1)
			assert.Equal(t, tt.want, got[0])
		})
	}
}

func TestAccountClassificationsForUnknownReturnsNil(t *testing.T) {
	got := AccountClassificationsFor(AccountTypeEnum("Bogus"))
	assert.Nil(t, got)
}

func TestAccountSubTypeValuesIsUnionOfPerTypeValues(t *testing.T) {
	got := AccountSubTypeValues()
	require.NotEmpty(t, got)

	gotSet := make(map[AccountSubTypeEnum]struct{}, len(got))
	for _, v := range got {
		gotSet[v] = struct{}{}
	}

	for _, parent := range AccountTypeValues() {
		for _, sub := range AccountSubTypesFor(parent) {
			_, ok := gotSet[sub]
			assert.Truef(t, ok, "AccountSubTypeValues missing %q (parent %q)", sub, parent)
		}
	}
}
