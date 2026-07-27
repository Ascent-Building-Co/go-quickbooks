package quickbooks

// This file defines additional general-purpose enum types and value catalog
// helpers for user-facing dropdowns, filters, and validation. The catalog is
// intentionally explicit (no reflection) and limited to enums that are
// general — locale-specific values (e.g. France TransactionLocationType,
// India ServiceType) are not included.
//
// All Values() helpers return a copy of an internal canonical slice so callers
// cannot mutate the package's data. Helpers for dependent enums
// (e.g. AccountSubTypesFor, AccountClassificationsFor) accept the parent enum
// value and return only the values valid for that parent.

// AccountClassificationEnum classifies an Account into one of the high-level
// QBO classifications. This value is server-derived from AccountType and is
// returned on Account responses but is not user-selectable when creating or
// updating an Account.
type AccountClassificationEnum string

const (
	AccountClassificationAsset     AccountClassificationEnum = "Asset"
	AccountClassificationEquity    AccountClassificationEnum = "Equity"
	AccountClassificationExpense   AccountClassificationEnum = "Expense"
	AccountClassificationLiability AccountClassificationEnum = "Liability"
	AccountClassificationRevenue   AccountClassificationEnum = "Revenue"
)

// ETransactionEnabledStatusEnum reflects whether eTransaction features are
// enabled on a SalesFormsPrefs configuration.
type ETransactionEnabledStatusEnum string

const (
	ETransactionEnabled       ETransactionEnabledStatusEnum = "Enabled"
	ETransactionNotApplicable ETransactionEnabledStatusEnum = "NotApplicable"
)

// TaxExemptionReasonEnum identifies why a Customer is exempt from sales tax.
// The underlying type is int because the QBO wire format is an integer ID;
// human-readable descriptions are exposed via Label.
type TaxExemptionReasonEnum int

const (
	TaxExemptionReasonFederalGovernment       TaxExemptionReasonEnum = 1
	TaxExemptionReasonStateGovernment         TaxExemptionReasonEnum = 2
	TaxExemptionReasonLocalGovernment         TaxExemptionReasonEnum = 3
	TaxExemptionReasonTribalGovernment        TaxExemptionReasonEnum = 4
	TaxExemptionReasonCharitableOrganization  TaxExemptionReasonEnum = 5
	TaxExemptionReasonReligiousOrganization   TaxExemptionReasonEnum = 6
	TaxExemptionReasonEducationalOrganization TaxExemptionReasonEnum = 7
	TaxExemptionReasonHospital                TaxExemptionReasonEnum = 8
	TaxExemptionReasonResale                  TaxExemptionReasonEnum = 9
	TaxExemptionReasonDirectPayPermit         TaxExemptionReasonEnum = 10
	TaxExemptionReasonMultiplePointsOfUse     TaxExemptionReasonEnum = 11
	TaxExemptionReasonDirectMail              TaxExemptionReasonEnum = 12
	TaxExemptionReasonAgriculturalProduction  TaxExemptionReasonEnum = 13
	TaxExemptionReasonIndustrialProduction    TaxExemptionReasonEnum = 14
	TaxExemptionReasonForeignDiplomat         TaxExemptionReasonEnum = 15
)

// AccountSubTypeEnum is the detailed sub-classification of an Account. The set
// of valid sub-types is constrained by the Account's AccountType — see
// AccountSubTypesFor for the parent → child lookup.
type AccountSubTypeEnum string

const (
	// Bank
	AccountSubTypeCashOnHand       AccountSubTypeEnum = "CashOnHand"
	AccountSubTypeChecking         AccountSubTypeEnum = "Checking"
	AccountSubTypeMoneyMarket      AccountSubTypeEnum = "MoneyMarket"
	AccountSubTypeRentsHeldInTrust AccountSubTypeEnum = "RentsHeldInTrust"
	AccountSubTypeSavings          AccountSubTypeEnum = "Savings"
	AccountSubTypeTrustAccounts    AccountSubTypeEnum = "TrustAccounts"
	AccountSubTypeBankOther        AccountSubTypeEnum = "OtherEarmarkedBankAccounts"

	// Other Current Asset
	AccountSubTypeAllowanceForBadDebts                        AccountSubTypeEnum = "AllowanceForBadDebts"
	AccountSubTypeDevelopmentCosts                            AccountSubTypeEnum = "DevelopmentCosts"
	AccountSubTypeEmployeeCashAdvances                        AccountSubTypeEnum = "EmployeeCashAdvances"
	AccountSubTypeOtherCurrentAssets                          AccountSubTypeEnum = "OtherCurrentAssets"
	AccountSubTypeInventory                                   AccountSubTypeEnum = "Inventory"
	AccountSubTypeInvestmentMortgageRealEstateLoans           AccountSubTypeEnum = "Investment_MortgageRealEstateLoans"
	AccountSubTypeInvestmentOther                             AccountSubTypeEnum = "Investment_Other"
	AccountSubTypeInvestmentTaxExemptSecurities               AccountSubTypeEnum = "Investment_TaxExemptSecurities"
	AccountSubTypeInvestmentUSGovernmentObligations           AccountSubTypeEnum = "Investment_USGovernmentObligations"
	AccountSubTypeLoansToOfficers                             AccountSubTypeEnum = "LoansToOfficers"
	AccountSubTypeLoansToOthers                               AccountSubTypeEnum = "LoansToOthers"
	AccountSubTypeLoansToStockholders                         AccountSubTypeEnum = "LoansToStockholders"
	AccountSubTypePrepaidExpenses                             AccountSubTypeEnum = "PrepaidExpenses"
	AccountSubTypeRetainage                                   AccountSubTypeEnum = "Retainage"
	AccountSubTypeUndepositedFunds                            AccountSubTypeEnum = "UndepositedFunds"
	AccountSubTypeAssetsAvailableForSale                      AccountSubTypeEnum = "AssetsAvailableForSale"
	AccountSubTypeBalWithGovtAuthorities                      AccountSubTypeEnum = "BalWithGovtAuthorities"
	AccountSubTypeCalledUpShareCapitalNotPaid                 AccountSubTypeEnum = "CalledUpShareCapitalNotPaid"
	AccountSubTypeExpenditureAuthorisationsAndLettersOfCredit AccountSubTypeEnum = "ExpenditureAuthorisationsAndLettersOfCredit"
	AccountSubTypeGlobalTaxDeferred                           AccountSubTypeEnum = "GlobalTaxDeferred"
	AccountSubTypeGlobalTaxRefund                             AccountSubTypeEnum = "GlobalTaxRefund"
	AccountSubTypeInternalTransfers                           AccountSubTypeEnum = "InternalTransfers"
	AccountSubTypeOtherConsumables                            AccountSubTypeEnum = "OtherConsumables"
	AccountSubTypeProvisionsCurrentAssets                     AccountSubTypeEnum = "ProvisionsCurrentAssets"
	AccountSubTypeShortTermInvestmentsInRelatedParties        AccountSubTypeEnum = "ShortTermInvestmentsInRelatedParties"
	AccountSubTypeShortTermLoansAndAdvancesToRelatedParties   AccountSubTypeEnum = "ShortTermLoansAndAdvancesToRelatedParties"
	AccountSubTypeTradeAndOtherReceivables                    AccountSubTypeEnum = "TradeAndOtherReceivables"

	// Fixed Asset
	AccountSubTypeAccumulatedDepletion                     AccountSubTypeEnum = "AccumulatedDepletion"
	AccountSubTypeAccumulatedDepreciation                  AccountSubTypeEnum = "AccumulatedDepreciation"
	AccountSubTypeDepletableAssets                         AccountSubTypeEnum = "DepletableAssets"
	AccountSubTypeFixedAssetComputers                      AccountSubTypeEnum = "FixedAssetComputers"
	AccountSubTypeFixedAssetCopiers                        AccountSubTypeEnum = "FixedAssetCopiers"
	AccountSubTypeFixedAssetFurniture                      AccountSubTypeEnum = "FixedAssetFurniture"
	AccountSubTypeFixedAssetPhone                          AccountSubTypeEnum = "FixedAssetPhone"
	AccountSubTypeFixedAssetPhotoVideo                     AccountSubTypeEnum = "FixedAssetPhotoVideo"
	AccountSubTypeFixedAssetSoftware                       AccountSubTypeEnum = "FixedAssetSoftware"
	AccountSubTypeFixedAssetOtherToolsEquipment            AccountSubTypeEnum = "FixedAssetOtherToolsEquipment"
	AccountSubTypeFurnitureAndFixtures                     AccountSubTypeEnum = "FurnitureAndFixtures"
	AccountSubTypeBuildings                                AccountSubTypeEnum = "Buildings"
	AccountSubTypeIntangibleAssets                         AccountSubTypeEnum = "IntangibleAssets"
	AccountSubTypeLand                                     AccountSubTypeEnum = "Land"
	AccountSubTypeLeaseholdImprovements                    AccountSubTypeEnum = "LeaseholdImprovements"
	AccountSubTypeMachineryAndEquipment                    AccountSubTypeEnum = "MachineryAndEquipment"
	AccountSubTypeOtherFixedAssets                         AccountSubTypeEnum = "OtherFixedAssets"
	AccountSubTypeVehicles                                 AccountSubTypeEnum = "Vehicles"
	AccountSubTypeAssetsInCourseOfConstruction             AccountSubTypeEnum = "AssetsInCourseOfConstruction"
	AccountSubTypeCapitalWip                               AccountSubTypeEnum = "CapitalWip"
	AccountSubTypeCumulativeDepreciationOnIntangibleAssets AccountSubTypeEnum = "CumulativeDepreciationOnIntangibleAssets"
	AccountSubTypeIntangibleAssetsUnderDevelopment         AccountSubTypeEnum = "IntangibleAssetsUnderDevelopment"
	AccountSubTypeLandAsset                                AccountSubTypeEnum = "LandAsset"
	AccountSubTypeNonCurrentAssets                         AccountSubTypeEnum = "NonCurrentAssets"
	AccountSubTypeParticipatingInterests                   AccountSubTypeEnum = "ParticipatingInterests"
	AccountSubTypeProvisionsFixedAssets                    AccountSubTypeEnum = "ProvisionsFixedAssets"

	// Other Asset
	AccountSubTypeLeaseBuyout                              AccountSubTypeEnum = "LeaseBuyout"
	AccountSubTypeOtherLongTermAssets                      AccountSubTypeEnum = "OtherLongTermAssets"
	AccountSubTypeSecurityDeposits                         AccountSubTypeEnum = "SecurityDeposits"
	AccountSubTypeAccumulatedAmortization                  AccountSubTypeEnum = "AccumulatedAmortization"
	AccountSubTypeAccumulatedAmortizationOfOtherAssets     AccountSubTypeEnum = "AccumulatedAmortizationOfOtherAssets"
	AccountSubTypeGoodwill                                 AccountSubTypeEnum = "Goodwill"
	AccountSubTypeIntangibleAssetsOther                    AccountSubTypeEnum = "IntangibleAssetsOther"
	AccountSubTypeLicenses                                 AccountSubTypeEnum = "Licenses"
	AccountSubTypeOrganizationalCosts                      AccountSubTypeEnum = "OrganizationalCosts"
	AccountSubTypeAssetsHeldForSale                        AccountSubTypeEnum = "AssetsHeldForSale"
	AccountSubTypeAvailableForSaleFinancialAssets          AccountSubTypeEnum = "AvailableForSaleFinancialAssets"
	AccountSubTypeDeferredTax                              AccountSubTypeEnum = "DeferredTax"
	AccountSubTypeInvestments                              AccountSubTypeEnum = "Investments"
	AccountSubTypeLongTermInvestments                      AccountSubTypeEnum = "LongTermInvestments"
	AccountSubTypeLongTermLoansAndAdvancesToRelatedParties AccountSubTypeEnum = "LongTermLoansAndAdvancesToRelatedParties"
	AccountSubTypeOtherIntangibleAssets                    AccountSubTypeEnum = "OtherIntangibleAssets"
	AccountSubTypeOtherLongTermInvestments                 AccountSubTypeEnum = "OtherLongTermInvestments"
	AccountSubTypeOtherLongTermLoansAndAdvances            AccountSubTypeEnum = "OtherLongTermLoansAndAdvances"
	AccountSubTypePrepaymentsAndAccruedIncome              AccountSubTypeEnum = "PrepaymentsAndAccruedIncome"
	AccountSubTypeProvisionsNonCurrentAssets               AccountSubTypeEnum = "ProvisionsNonCurrentAssets"

	// Accounts Receivable
	AccountSubTypeAccountsReceivable AccountSubTypeEnum = "AccountsReceivable"

	// Equity
	AccountSubTypeOpeningBalanceEquity                  AccountSubTypeEnum = "OpeningBalanceEquity"
	AccountSubTypePartnersEquity                        AccountSubTypeEnum = "PartnersEquity"
	AccountSubTypeRetainedEarnings                      AccountSubTypeEnum = "RetainedEarnings"
	AccountSubTypeAccumulatedAdjustment                 AccountSubTypeEnum = "AccumulatedAdjustment"
	AccountSubTypeOwnersEquity                          AccountSubTypeEnum = "OwnersEquity"
	AccountSubTypePaidInCapitalOrSurplus                AccountSubTypeEnum = "PaidInCapitalOrSurplus"
	AccountSubTypePartnerContributions                  AccountSubTypeEnum = "PartnerContributions"
	AccountSubTypePartnerDistributions                  AccountSubTypeEnum = "PartnerDistributions"
	AccountSubTypePreferredStock                        AccountSubTypeEnum = "PreferredStock"
	AccountSubTypeCommonStock                           AccountSubTypeEnum = "CommonStock"
	AccountSubTypeTreasuryStock                         AccountSubTypeEnum = "TreasuryStock"
	AccountSubTypeEstimatedTaxes                        AccountSubTypeEnum = "EstimatedTaxes"
	AccountSubTypeHealthcare                            AccountSubTypeEnum = "Healthcare"
	AccountSubTypePersonalIncome                        AccountSubTypeEnum = "PersonalIncome"
	AccountSubTypePersonalExpense                       AccountSubTypeEnum = "PersonalExpense"
	AccountSubTypeAccumulatedOtherComprehensiveIncome   AccountSubTypeEnum = "AccumulatedOtherComprehensiveIncome"
	AccountSubTypeCalledUpShareCapital                  AccountSubTypeEnum = "CalledUpShareCapital"
	AccountSubTypeCapitalReserves                       AccountSubTypeEnum = "CapitalReserves"
	AccountSubTypeDividendDisbursed                     AccountSubTypeEnum = "DividendDisbursed"
	AccountSubTypeEquityInEarningsOfSubsiduaries        AccountSubTypeEnum = "EquityInEarningsOfSubsiduaries"
	AccountSubTypeFunds                                 AccountSubTypeEnum = "Funds"
	AccountSubTypeInvestmentGrants                      AccountSubTypeEnum = "InvestmentGrants"
	AccountSubTypeMoneyReceivedAgainstShareWarrants     AccountSubTypeEnum = "MoneyReceivedAgainstShareWarrants"
	AccountSubTypeOtherFreeReserves                     AccountSubTypeEnum = "OtherFreeReserves"
	AccountSubTypeShareApplicationMoneyPendingAllotment AccountSubTypeEnum = "ShareApplicationMoneyPendingAllotment"
	AccountSubTypeShareCapital                          AccountSubTypeEnum = "ShareCapital"
	AccountSubTypeUnrealisedLossOnSecuritiesNetOfTax    AccountSubTypeEnum = "UnrealisedLossOnSecuritiesNetOfTax"

	// Expense
	AccountSubTypeAdvertisingPromotional                AccountSubTypeEnum = "AdvertisingPromotional"
	AccountSubTypeBadDebts                              AccountSubTypeEnum = "BadDebts"
	AccountSubTypeBankCharges                           AccountSubTypeEnum = "BankCharges"
	AccountSubTypeCharitableContributions               AccountSubTypeEnum = "CharitableContributions"
	AccountSubTypeCommissionsAndFees                    AccountSubTypeEnum = "CommissionsAndFees"
	AccountSubTypeEntertainment                         AccountSubTypeEnum = "Entertainment"
	AccountSubTypeEntertainmentMeals                    AccountSubTypeEnum = "EntertainmentMeals"
	AccountSubTypeEquipmentRental                       AccountSubTypeEnum = "EquipmentRental"
	AccountSubTypeFinanceCosts                          AccountSubTypeEnum = "FinanceCosts"
	AccountSubTypeGlobalTaxExpense                      AccountSubTypeEnum = "GlobalTaxExpense"
	AccountSubTypeInsurance                             AccountSubTypeEnum = "Insurance"
	AccountSubTypeInterestPaid                          AccountSubTypeEnum = "InterestPaid"
	AccountSubTypeLegalProfessionalFees                 AccountSubTypeEnum = "LegalProfessionalFees"
	AccountSubTypeOfficeExpenses                        AccountSubTypeEnum = "OfficeExpenses"
	AccountSubTypeOfficeGeneralAdministrativeExpenses   AccountSubTypeEnum = "OfficeGeneralAdministrativeExpenses"
	AccountSubTypeOtherBusinessExpenses                 AccountSubTypeEnum = "OtherBusinessExpenses"
	AccountSubTypeOtherMiscellaneousExpense             AccountSubTypeEnum = "OtherMiscellaneousExpense"
	AccountSubTypePromotionalMeals                      AccountSubTypeEnum = "PromotionalMeals"
	AccountSubTypeRentOrLeaseOfBuildings                AccountSubTypeEnum = "RentOrLeaseOfBuildings"
	AccountSubTypeRepairMaintenance                     AccountSubTypeEnum = "RepairMaintenance"
	AccountSubTypeShippingFreightDelivery               AccountSubTypeEnum = "ShippingFreightDelivery"
	AccountSubTypeSuppliesMaterials                     AccountSubTypeEnum = "SuppliesMaterials"
	AccountSubTypeTravel                                AccountSubTypeEnum = "Travel"
	AccountSubTypeTravelMeals                           AccountSubTypeEnum = "TravelMeals"
	AccountSubTypeAuto                                  AccountSubTypeEnum = "Auto"
	AccountSubTypeCostOfLabor                           AccountSubTypeEnum = "CostOfLabor"
	AccountSubTypeDuesSubscriptions                     AccountSubTypeEnum = "DuesSubscriptions"
	AccountSubTypePayrollExpenses                       AccountSubTypeEnum = "PayrollExpenses"
	AccountSubTypeTaxesPaid                             AccountSubTypeEnum = "TaxesPaid"
	AccountSubTypeUnappliedCashBillPaymentExpense       AccountSubTypeEnum = "UnappliedCashBillPaymentExpense"
	AccountSubTypeUtilities                             AccountSubTypeEnum = "Utilities"
	AccountSubTypeAmortizationExpense                   AccountSubTypeEnum = "AmortizationExpense"
	AccountSubTypeAppropriationsToDepreciation          AccountSubTypeEnum = "AppropriationsToDepreciation"
	AccountSubTypeBorrowingCost                         AccountSubTypeEnum = "BorrowingCost"
	AccountSubTypeDistributionCosts                     AccountSubTypeEnum = "DistributionCosts"
	AccountSubTypeExternalServices                      AccountSubTypeEnum = "ExternalServices"
	AccountSubTypeExtraordinaryCharges                  AccountSubTypeEnum = "ExtraordinaryCharges"
	AccountSubTypeIncomeTaxExpense                      AccountSubTypeEnum = "IncomeTaxExpense"
	AccountSubTypeLossOnDiscontinuedOperationsNetOfTax  AccountSubTypeEnum = "LossOnDiscontinuedOperationsNetOfTax"
	AccountSubTypeManagementCompensation                AccountSubTypeEnum = "ManagementCompensation"
	AccountSubTypeOtherCurrentOperatingCharges          AccountSubTypeEnum = "OtherCurrentOperatingCharges"
	AccountSubTypeOtherExternalServices                 AccountSubTypeEnum = "OtherExternalServices"
	AccountSubTypeOtherRentalCosts                      AccountSubTypeEnum = "OtherRentalCosts"
	AccountSubTypeOtherSellingExpenses                  AccountSubTypeEnum = "OtherSellingExpenses"
	AccountSubTypeProjectStudiesSurveysAssessments      AccountSubTypeEnum = "ProjectStudiesSurveysAssessments"
	AccountSubTypePurchasesRebates                      AccountSubTypeEnum = "PurchasesRebates"
	AccountSubTypeShippingAndDeliveryExpense            AccountSubTypeEnum = "ShippingAndDeliveryExpense"
	AccountSubTypeStaffCosts                            AccountSubTypeEnum = "StaffCosts"
	AccountSubTypeSundry                                AccountSubTypeEnum = "Sundry"
	AccountSubTypeTravelExpensesGeneralAndAdminExpenses AccountSubTypeEnum = "TravelExpensesGeneralAndAdminExpenses"
	AccountSubTypeTravelExpensesSellingExpense          AccountSubTypeEnum = "TravelExpensesSellingExpense"

	// Other Expense
	AccountSubTypeDepreciation                   AccountSubTypeEnum = "Depreciation"
	AccountSubTypeExchangeGainOrLoss             AccountSubTypeEnum = "ExchangeGainOrLoss"
	AccountSubTypeOtherMiscellaneousExpenseSub   AccountSubTypeEnum = "OtherMiscellaneousExpenseSub"
	AccountSubTypePenaltiesSettlements           AccountSubTypeEnum = "PenaltiesSettlements"
	AccountSubTypeAmortization                   AccountSubTypeEnum = "Amortization"
	AccountSubTypeGasAndFuel                     AccountSubTypeEnum = "GasAndFuel"
	AccountSubTypeHomeOffice                     AccountSubTypeEnum = "HomeOffice"
	AccountSubTypeHomeownerRentalInsurance       AccountSubTypeEnum = "HomeownerRentalInsurance"
	AccountSubTypeOtherHomeOfficeExpenses        AccountSubTypeEnum = "OtherHomeOfficeExpenses"
	AccountSubTypeMortgageInterestHomeOffice     AccountSubTypeEnum = "MortgageInterestHomeOffice"
	AccountSubTypeRentAndLeaseHomeOffice         AccountSubTypeEnum = "RentAndLeaseHomeOffice"
	AccountSubTypeRepairsAndMaintainceHomeOffice AccountSubTypeEnum = "RepairsAndMaintainceHomeOffice"
	AccountSubTypeParkingAndTolls                AccountSubTypeEnum = "ParkingAndTolls"
	AccountSubTypeVehicle                        AccountSubTypeEnum = "Vehicle"
	AccountSubTypeVehicleInsurance               AccountSubTypeEnum = "VehicleInsurance"
	AccountSubTypeVehicleLease                   AccountSubTypeEnum = "VehicleLease"
	AccountSubTypeVehicleLoanInterest            AccountSubTypeEnum = "VehicleLoanInterest"
	AccountSubTypeVehicleRepairs                 AccountSubTypeEnum = "VehicleRepairs"
	AccountSubTypeOtherVehicleExpenses           AccountSubTypeEnum = "OtherVehicleExpenses"
	AccountSubTypeUtilitiesHomeOffice            AccountSubTypeEnum = "UtilitiesHomeOffice"
	AccountSubTypeWashAndRoadServices            AccountSubTypeEnum = "WashAndRoadServices"
	AccountSubTypeDeferredTaxExpense             AccountSubTypeEnum = "DeferredTaxExpense"
	AccountSubTypeDepletion                      AccountSubTypeEnum = "Depletion"
	AccountSubTypeExceptionalItems               AccountSubTypeEnum = "ExceptionalItems"
	AccountSubTypeExtraordinaryItems             AccountSubTypeEnum = "ExtraordinaryItems"
	AccountSubTypeIncomeTaxOtherExpense          AccountSubTypeEnum = "IncomeTaxOtherExpense"
	AccountSubTypeMatCredit                      AccountSubTypeEnum = "MatCredit"
	AccountSubTypePriorPeriodItems               AccountSubTypeEnum = "PriorPeriodItems"
	AccountSubTypeTaxRoundoffGainOrLoss          AccountSubTypeEnum = "TaxRoundoffGainOrLoss"

	// Cost of Goods Sold
	AccountSubTypeEquipmentRentalCos         AccountSubTypeEnum = "EquipmentRentalCos"
	AccountSubTypeOtherCostsOfServiceCos     AccountSubTypeEnum = "OtherCostsOfServiceCos"
	AccountSubTypeShippingFreightDeliveryCos AccountSubTypeEnum = "ShippingFreightDeliveryCos"
	AccountSubTypeSuppliesMaterialsCogs      AccountSubTypeEnum = "SuppliesMaterialsCogs"
	AccountSubTypeCostOfLaborCos             AccountSubTypeEnum = "CostOfLaborCos"
	AccountSubTypeCostOfSales                AccountSubTypeEnum = "CostOfSales"
	AccountSubTypeFreightAndDeliveryCos      AccountSubTypeEnum = "FreightAndDeliveryCos"

	// Accounts Payable
	AccountSubTypeAccountsPayable                              AccountSubTypeEnum = "AccountsPayable"
	AccountSubTypeOutstandingDuesMicroSmallEnterprise          AccountSubTypeEnum = "OutstandingDuesMicroSmallEnterprise"
	AccountSubTypeOutstandingDuesOtherThanMicroSmallEnterprise AccountSubTypeEnum = "OutstandingDuesOtherThanMicroSmallEnterprise"

	// Credit Card
	AccountSubTypeCreditCard AccountSubTypeEnum = "CreditCard"

	// Long Term Liability
	AccountSubTypeNotesPayable                             AccountSubTypeEnum = "NotesPayable"
	AccountSubTypeOtherLongTermLiabilities                 AccountSubTypeEnum = "OtherLongTermLiabilities"
	AccountSubTypeShareholderNotesPayable                  AccountSubTypeEnum = "ShareholderNotesPayable"
	AccountSubTypeAccruedLongLermLiabilities               AccountSubTypeEnum = "AccruedLongLermLiabilities"
	AccountSubTypeAccruedVacationPayable                   AccountSubTypeEnum = "AccruedVacationPayable"
	AccountSubTypeBankLoans                                AccountSubTypeEnum = "BankLoans"
	AccountSubTypeDebtsRelatedToParticipatingInterests     AccountSubTypeEnum = "DebtsRelatedToParticipatingInterests"
	AccountSubTypeDeferredTaxLiabilities                   AccountSubTypeEnum = "DeferredTaxLiabilities"
	AccountSubTypeGovernmentAndOtherPublicAuthorities      AccountSubTypeEnum = "GovernmentAndOtherPublicAuthorities"
	AccountSubTypeGroupAndAssociates                       AccountSubTypeEnum = "GroupAndAssociates"
	AccountSubTypeLiabilitiesRelatedToAssetsHeldForSale    AccountSubTypeEnum = "LiabilitiesRelatedToAssetsHeldForSale"
	AccountSubTypeLongTermBorrowings                       AccountSubTypeEnum = "LongTermBorrowings"
	AccountSubTypeLongTermDebit                            AccountSubTypeEnum = "LongTermDebit"
	AccountSubTypeLongTermEmployeeBenefitObligations       AccountSubTypeEnum = "LongTermEmployeeBenefitObligations"
	AccountSubTypeObligationsUnderFinanceLeases            AccountSubTypeEnum = "ObligationsUnderFinanceLeases"
	AccountSubTypeOtherLongTermProvisions                  AccountSubTypeEnum = "OtherLongTermProvisions"
	AccountSubTypeProvisionForLiabilities                  AccountSubTypeEnum = "ProvisionForLiabilities"
	AccountSubTypeProvisionsNonCurrentLiabilities          AccountSubTypeEnum = "ProvisionsNonCurrentLiabilities"
	AccountSubTypeStaffAndRelatedLongTermLiabilityAccounts AccountSubTypeEnum = "StaffAndRelatedLongTermLiabilityAccounts"

	// Other Current Liability
	AccountSubTypeDirectDepositPayable                          AccountSubTypeEnum = "DirectDepositPayable"
	AccountSubTypeFederalIncomeTaxPayable                       AccountSubTypeEnum = "FederalIncomeTaxPayable"
	AccountSubTypeInsurancePayable                              AccountSubTypeEnum = "InsurancePayable"
	AccountSubTypeLineOfCredit                                  AccountSubTypeEnum = "LineOfCredit"
	AccountSubTypeLoanPayable                                   AccountSubTypeEnum = "LoanPayable"
	AccountSubTypeGlobalTaxPayable                              AccountSubTypeEnum = "GlobalTaxPayable"
	AccountSubTypeGlobalTaxSuspense                             AccountSubTypeEnum = "GlobalTaxSuspense"
	AccountSubTypeOtherCurrentLiabilities                       AccountSubTypeEnum = "OtherCurrentLiabilities"
	AccountSubTypePayrollClearing                               AccountSubTypeEnum = "PayrollClearing"
	AccountSubTypePayrollTaxPayable                             AccountSubTypeEnum = "PayrollTaxPayable"
	AccountSubTypePrepaidExpensesPayable                        AccountSubTypeEnum = "PrepaidExpensesPayable"
	AccountSubTypeRentsInTrustLiability                         AccountSubTypeEnum = "RentsInTrustLiability"
	AccountSubTypeTrustAccountsLiabilities                      AccountSubTypeEnum = "TrustAccountsLiabilities"
	AccountSubTypeSalesTaxPayable                               AccountSubTypeEnum = "SalesTaxPayable"
	AccountSubTypeStateLocalIncomeTaxPayable                    AccountSubTypeEnum = "StateLocalIncomeTaxPayable"
	AccountSubTypeAccrualsAndDeferredIncome                     AccountSubTypeEnum = "AccrualsAndDeferredIncome"
	AccountSubTypeAccruedLiabilities                            AccountSubTypeEnum = "AccruedLiabilities"
	AccountSubTypeCurrentLiabilities                            AccountSubTypeEnum = "CurrentLiabilities"
	AccountSubTypeCurrentPortionEmployeeBenefitsObligations     AccountSubTypeEnum = "CurrentPortionEmployeeBenefitsObligations"
	AccountSubTypeCurrentPortionOfObligationsUnderFinanceLeases AccountSubTypeEnum = "CurrentPortionOfObligationsUnderFinanceLeases"
	AccountSubTypeCurrentTaxLiability                           AccountSubTypeEnum = "CurrentTaxLiability"
	AccountSubTypeDividendsPayable                              AccountSubTypeEnum = "DividendsPayable"
	AccountSubTypeDutiesAndTaxes                                AccountSubTypeEnum = "DutiesAndTaxes"
	AccountSubTypeInterestPayables                              AccountSubTypeEnum = "InterestPayables"
	AccountSubTypeProvisionForWarrantyObligations               AccountSubTypeEnum = "ProvisionForWarrantyObligations"
	AccountSubTypeProvisionsCurrentLiabilities                  AccountSubTypeEnum = "ProvisionsCurrentLiabilities"
	AccountSubTypeShortTermBorrowings                           AccountSubTypeEnum = "ShortTermBorrowings"
	AccountSubTypeSocialSecurityAgencies                        AccountSubTypeEnum = "SocialSecurityAgencies"
	AccountSubTypeStaffAndRelatedLiabilityAccounts              AccountSubTypeEnum = "StaffAndRelatedLiabilityAccounts"
	AccountSubTypeSundryDebtorsAndCreditors                     AccountSubTypeEnum = "SundryDebtorsAndCreditors"
	AccountSubTypeTradeAndOtherPayables                         AccountSubTypeEnum = "TradeAndOtherPayables"

	// Income
	AccountSubTypeNonProfitIncome             AccountSubTypeEnum = "NonProfitIncome"
	AccountSubTypeOtherPrimaryIncome          AccountSubTypeEnum = "OtherPrimaryIncome"
	AccountSubTypeSalesOfProductIncome        AccountSubTypeEnum = "SalesOfProductIncome"
	AccountSubTypeServiceFeeIncome            AccountSubTypeEnum = "ServiceFeeIncome"
	AccountSubTypeDiscountsRefundsGiven       AccountSubTypeEnum = "DiscountsRefundsGiven"
	AccountSubTypeUnappliedCashPaymentIncome  AccountSubTypeEnum = "UnappliedCashPaymentIncome"
	AccountSubTypeCashReceiptIncome           AccountSubTypeEnum = "CashReceiptIncome"
	AccountSubTypeOperatingGrants             AccountSubTypeEnum = "OperatingGrants"
	AccountSubTypeOtherCurrentOperatingIncome AccountSubTypeEnum = "OtherCurrentOperatingIncome"
	AccountSubTypeOwnWorkCapitalized          AccountSubTypeEnum = "OwnWorkCapitalized"
	AccountSubTypeRevenueGeneral              AccountSubTypeEnum = "RevenueGeneral"
	AccountSubTypeSalesRetail                 AccountSubTypeEnum = "SalesRetail"
	AccountSubTypeSalesWholesale              AccountSubTypeEnum = "SalesWholesale"
	AccountSubTypeSavingsByTaxScheme          AccountSubTypeEnum = "SavingsByTaxScheme"

	// Other Income
	AccountSubTypeDividendIncome                       AccountSubTypeEnum = "DividendIncome"
	AccountSubTypeInterestEarned                       AccountSubTypeEnum = "InterestEarned"
	AccountSubTypeOtherInvestmentIncome                AccountSubTypeEnum = "OtherInvestmentIncome"
	AccountSubTypeOtherMiscellaneousIncome             AccountSubTypeEnum = "OtherMiscellaneousIncome"
	AccountSubTypeTaxExemptInterest                    AccountSubTypeEnum = "TaxExemptInterest"
	AccountSubTypeGainLossOnSaleOfFixedAssets          AccountSubTypeEnum = "GainLossOnSaleOfFixedAssets"
	AccountSubTypeGainLossOnSaleOfInvestments          AccountSubTypeEnum = "GainLossOnSaleOfInvestments"
	AccountSubTypeLossOnDisposalOfAssets               AccountSubTypeEnum = "LossOnDisposalOfAssets"
	AccountSubTypeOtherOperatingIncome                 AccountSubTypeEnum = "OtherOperatingIncome"
	AccountSubTypeUnrealisedLossOnSecuritiesNetOfTaxes AccountSubTypeEnum = "UnrealisedLossOnSecuritiesNetOfTaxes"
)

// Canonical value lists. Each Values() helper returns a fresh copy so callers
// cannot mutate the package's internal slice.

var customFieldTypeValues = []CustomFieldTypeEnum{
	CustomFieldTypeString,
	CustomFieldTypeBoolean,
	CustomFieldTypeNumber,
	CustomFieldTypeDate,
}

var printStatusValues = []PrintStatusEnum{
	NotSetStatus,
	NeedToPrintStatus,
	PrintCompleteStatus,
}

var emailStatusValues = []EmailStatusEnum{
	EmailNotSetStatus,
	EmailNeedToSend,
	EmailSent,
}

var globalTaxCalculationValues = []GlobalTaxCalculationEnum{
	TaxExcludedGlobalTaxCalculation,
	TaxInclusiveGlobalTaxCalculation,
	NotApplicableGlobalTaxCalculation,
}

var postingTypeValues = []PostingTypeEnum{
	DebitPostingType,
	CreditPostingType,
}

var itemTypeValues = []ItemTypeEnum{
	InventoryItemType,
	ServiceItemType,
	NonInventoryItemType,
	GroupItemType,
	CategoryItemType,
}

var billableStatusValues = []BillableStatusEnum{
	BillableBillableStatus,
	NotBillableBillableStatus,
	HasBeenBilledBillableStatus,
}

var accountTypeValues = []AccountTypeEnum{
	AccountTypeBank,
	AccountTypeOtherCurrentAsset,
	AccountTypeFixedAsset,
	AccountTypeOtherAsset,
	AccountTypeAccountsReceivable,
	AccountTypeEquity,
	AccountTypeExpense,
	AccountTypeOtherExpense,
	AccountTypeCostOfGoodsSold,
	AccountTypeAccountsPayable,
	AccountTypeCreditCard,
	AccountTypeLongTermLiability,
	AccountTypeOtherCurrentLiab,
	AccountTypeIncome,
	AccountTypeOtherIncome,
}

var accountClassificationValues = []AccountClassificationEnum{
	AccountClassificationAsset,
	AccountClassificationEquity,
	AccountClassificationExpense,
	AccountClassificationLiability,
	AccountClassificationRevenue,
}

var nameOfValues = []NameOfEnum{
	NameOfVendor,
	NameOfEmployee,
}

var termTypeValues = []TermTypeEnum{
	TermTypeStandard,
	TermTypeDateDriven,
}

var reportBasisValues = []ReportBasisEnum{
	ReportBasisAccrual,
	ReportBasisCash,
}

var workWeekStartDateValues = []WorkWeekStartDateEnum{
	WorkWeekStartSunday,
	WorkWeekStartMonday,
	WorkWeekStartTuesday,
	WorkWeekStartWednesday,
	WorkWeekStartThursday,
	WorkWeekStartFriday,
	WorkWeekStartSaturday,
}

var monthValues = []MonthEnum{
	MonthJanuary,
	MonthFebruary,
	MonthMarch,
	MonthApril,
	MonthMay,
	MonthJune,
	MonthJuly,
	MonthAugust,
	MonthSeptember,
	MonthOctober,
	MonthNovember,
	MonthDecember,
}

var eTransactionEnabledStatusValues = []ETransactionEnabledStatusEnum{
	ETransactionEnabled,
	ETransactionNotApplicable,
}

var taxExemptionReasonValues = []TaxExemptionReasonEnum{
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
}

// taxExemptionReasonLabels maps each TaxExemptionReasonEnum value to the
// human-readable description published by QBO. Values not present in this
// map produce an empty Label.
var taxExemptionReasonLabels = map[TaxExemptionReasonEnum]string{
	TaxExemptionReasonFederalGovernment:       "Federal government",
	TaxExemptionReasonStateGovernment:         "State government",
	TaxExemptionReasonLocalGovernment:         "Local government",
	TaxExemptionReasonTribalGovernment:        "Tribal government",
	TaxExemptionReasonCharitableOrganization:  "Charitable organization",
	TaxExemptionReasonReligiousOrganization:   "Religious organization",
	TaxExemptionReasonEducationalOrganization: "Educational organization",
	TaxExemptionReasonHospital:                "Hospital",
	TaxExemptionReasonResale:                  "Resale",
	TaxExemptionReasonDirectPayPermit:         "Direct pay permit",
	TaxExemptionReasonMultiplePointsOfUse:     "Multiple points of use",
	TaxExemptionReasonDirectMail:              "Direct mail",
	TaxExemptionReasonAgriculturalProduction:  "Agricultural production",
	TaxExemptionReasonIndustrialProduction:    "Industrial production / manufacturing",
	TaxExemptionReasonForeignDiplomat:         "Foreign diplomat",
}

var attachableCategoryValues = []AttachableCategoryEnum{
	ContactPhotoAttachableCategory,
	DocumentAttachableCategory,
	ImageAttachableCategory,
	ReceiptAttachableCategory,
	SignatureAttachableCategory,
	SoundAttachableCategory,
	OtherAttachableCategory,
}

var billPaymentTypeValues = []BillPaymentTypeEnum{
	CheckBillPaymentType,
	CreditCardBillPaymentType,
}

var purchasePaymentTypeValues = []PurchasePaymentTypeEnum{
	CashPurchasePaymentType,
	CheckPurchasePaymentType,
	CreditCardPurchasePaymentType,
}

var estimateTxnStatusValues = []EstimateTxnStatusEnum{
	EstimatePending,
	EstimateAccepted,
	EstimateRejected,
	EstimateClosed,
	EstimateConverted,
}

// Values helpers — return a copy of the canonical slice for the given enum.

// CustomFieldTypeValues returns all valid CustomFieldTypeEnum values.
func CustomFieldTypeValues() []CustomFieldTypeEnum {
	return append([]CustomFieldTypeEnum(nil), customFieldTypeValues...)
}

// PrintStatusValues returns all valid PrintStatusEnum values.
func PrintStatusValues() []PrintStatusEnum {
	return append([]PrintStatusEnum(nil), printStatusValues...)
}

// EmailStatusValues returns all valid EmailStatusEnum values.
func EmailStatusValues() []EmailStatusEnum {
	return append([]EmailStatusEnum(nil), emailStatusValues...)
}

// GlobalTaxCalculationValues returns all valid GlobalTaxCalculationEnum values.
func GlobalTaxCalculationValues() []GlobalTaxCalculationEnum {
	return append([]GlobalTaxCalculationEnum(nil), globalTaxCalculationValues...)
}

// PostingTypeValues returns all valid PostingTypeEnum values.
func PostingTypeValues() []PostingTypeEnum {
	return append([]PostingTypeEnum(nil), postingTypeValues...)
}

// ItemTypeValues returns all valid ItemTypeEnum values.
func ItemTypeValues() []ItemTypeEnum {
	return append([]ItemTypeEnum(nil), itemTypeValues...)
}

// BillableStatusValues returns all valid BillableStatusEnum values.
func BillableStatusValues() []BillableStatusEnum {
	return append([]BillableStatusEnum(nil), billableStatusValues...)
}

// AccountTypeValues returns all valid AccountTypeEnum values supported by QBO.
// The "Non-Posting" account type is intentionally excluded as QBO does not
// support it.
func AccountTypeValues() []AccountTypeEnum {
	return append([]AccountTypeEnum(nil), accountTypeValues...)
}

// AccountClassificationValues returns all valid AccountClassificationEnum
// values. Note that AccountClassification is server-derived from AccountType
// and is not user-selectable on create or update.
func AccountClassificationValues() []AccountClassificationEnum {
	return append([]AccountClassificationEnum(nil), accountClassificationValues...)
}

// NameOfValues returns all valid NameOfEnum values used by TimeActivity.
func NameOfValues() []NameOfEnum {
	return append([]NameOfEnum(nil), nameOfValues...)
}

// TermTypeValues returns all valid TermTypeEnum values.
func TermTypeValues() []TermTypeEnum {
	return append([]TermTypeEnum(nil), termTypeValues...)
}

// ReportBasisValues returns all valid ReportBasisEnum values.
func ReportBasisValues() []ReportBasisEnum {
	return append([]ReportBasisEnum(nil), reportBasisValues...)
}

// WorkWeekStartDateValues returns all valid WorkWeekStartDateEnum values.
func WorkWeekStartDateValues() []WorkWeekStartDateEnum {
	return append([]WorkWeekStartDateEnum(nil), workWeekStartDateValues...)
}

// MonthValues returns all valid MonthEnum values, in calendar order.
func MonthValues() []MonthEnum {
	return append([]MonthEnum(nil), monthValues...)
}

// ETransactionEnabledStatusValues returns all valid
// ETransactionEnabledStatusEnum values.
func ETransactionEnabledStatusValues() []ETransactionEnabledStatusEnum {
	return append([]ETransactionEnabledStatusEnum(nil), eTransactionEnabledStatusValues...)
}

// TaxExemptionReasonValues returns all valid TaxExemptionReasonEnum values in
// the order published by QBO (1..15).
func TaxExemptionReasonValues() []TaxExemptionReasonEnum {
	return append([]TaxExemptionReasonEnum(nil), taxExemptionReasonValues...)
}

// Label returns the QBO-published human-readable description for the reason,
// for example "Resale". Label returns "" if the value is not a known
// TaxExemptionReasonEnum.
func (r TaxExemptionReasonEnum) Label() string {
	return taxExemptionReasonLabels[r]
}

// AttachableCategoryValues returns all valid AttachableCategoryEnum values.
func AttachableCategoryValues() []AttachableCategoryEnum {
	return append([]AttachableCategoryEnum(nil), attachableCategoryValues...)
}

// BillPaymentTypeValues returns all valid BillPaymentTypeEnum values.
func BillPaymentTypeValues() []BillPaymentTypeEnum {
	return append([]BillPaymentTypeEnum(nil), billPaymentTypeValues...)
}

// PurchasePaymentTypeValues returns all valid PurchasePaymentTypeEnum values.
func PurchasePaymentTypeValues() []PurchasePaymentTypeEnum {
	return append([]PurchasePaymentTypeEnum(nil), purchasePaymentTypeValues...)
}

// EstimateTxnStatusValues returns all valid EstimateTxnStatusEnum values.
func EstimateTxnStatusValues() []EstimateTxnStatusEnum {
	return append([]EstimateTxnStatusEnum(nil), estimateTxnStatusValues...)
}

// AccountSubTypeValues returns every AccountSubTypeEnum value across all
// AccountTypes, in a stable order. For dependent dropdowns, prefer
// AccountSubTypesFor.
func AccountSubTypeValues() []AccountSubTypeEnum {
	out := make([]AccountSubTypeEnum, 0, 256)
	seen := make(map[AccountSubTypeEnum]struct{}, 256)
	for _, parent := range accountTypeValues {
		for _, sub := range accountSubTypesByType[parent] {
			if _, ok := seen[sub]; ok {
				continue
			}
			seen[sub] = struct{}{}
			out = append(out, sub)
		}
	}
	return out
}

// accountSubTypesByType maps each AccountTypeEnum to the AccountSubTypeEnum
// values valid for it. The mapping reflects the QBO Accounting API and is the
// source of truth for AccountSubTypesFor.
var accountSubTypesByType = map[AccountTypeEnum][]AccountSubTypeEnum{
	AccountTypeBank: {
		AccountSubTypeCashOnHand,
		AccountSubTypeChecking,
		AccountSubTypeMoneyMarket,
		AccountSubTypeRentsHeldInTrust,
		AccountSubTypeSavings,
		AccountSubTypeTrustAccounts,
		AccountSubTypeBankOther,
	},
	AccountTypeOtherCurrentAsset: {
		AccountSubTypeAllowanceForBadDebts,
		AccountSubTypeDevelopmentCosts,
		AccountSubTypeEmployeeCashAdvances,
		AccountSubTypeOtherCurrentAssets,
		AccountSubTypeInventory,
		AccountSubTypeInvestmentMortgageRealEstateLoans,
		AccountSubTypeInvestmentOther,
		AccountSubTypeInvestmentTaxExemptSecurities,
		AccountSubTypeInvestmentUSGovernmentObligations,
		AccountSubTypeLoansToOfficers,
		AccountSubTypeLoansToOthers,
		AccountSubTypeLoansToStockholders,
		AccountSubTypePrepaidExpenses,
		AccountSubTypeRetainage,
		AccountSubTypeUndepositedFunds,
		AccountSubTypeAssetsAvailableForSale,
		AccountSubTypeBalWithGovtAuthorities,
		AccountSubTypeCalledUpShareCapitalNotPaid,
		AccountSubTypeExpenditureAuthorisationsAndLettersOfCredit,
		AccountSubTypeGlobalTaxDeferred,
		AccountSubTypeGlobalTaxRefund,
		AccountSubTypeInternalTransfers,
		AccountSubTypeOtherConsumables,
		AccountSubTypeProvisionsCurrentAssets,
		AccountSubTypeShortTermInvestmentsInRelatedParties,
		AccountSubTypeShortTermLoansAndAdvancesToRelatedParties,
		AccountSubTypeTradeAndOtherReceivables,
	},
	AccountTypeFixedAsset: {
		AccountSubTypeAccumulatedDepletion,
		AccountSubTypeAccumulatedDepreciation,
		AccountSubTypeDepletableAssets,
		AccountSubTypeFixedAssetComputers,
		AccountSubTypeFixedAssetCopiers,
		AccountSubTypeFixedAssetFurniture,
		AccountSubTypeFixedAssetPhone,
		AccountSubTypeFixedAssetPhotoVideo,
		AccountSubTypeFixedAssetSoftware,
		AccountSubTypeFixedAssetOtherToolsEquipment,
		AccountSubTypeFurnitureAndFixtures,
		AccountSubTypeBuildings,
		AccountSubTypeIntangibleAssets,
		AccountSubTypeLand,
		AccountSubTypeLeaseholdImprovements,
		AccountSubTypeMachineryAndEquipment,
		AccountSubTypeOtherFixedAssets,
		AccountSubTypeVehicles,
		AccountSubTypeAssetsInCourseOfConstruction,
		AccountSubTypeCapitalWip,
		AccountSubTypeCumulativeDepreciationOnIntangibleAssets,
		AccountSubTypeIntangibleAssetsUnderDevelopment,
		AccountSubTypeLandAsset,
		AccountSubTypeNonCurrentAssets,
		AccountSubTypeParticipatingInterests,
		AccountSubTypeProvisionsFixedAssets,
	},
	AccountTypeOtherAsset: {
		AccountSubTypeLeaseBuyout,
		AccountSubTypeOtherLongTermAssets,
		AccountSubTypeSecurityDeposits,
		AccountSubTypeAccumulatedAmortization,
		AccountSubTypeAccumulatedAmortizationOfOtherAssets,
		AccountSubTypeGoodwill,
		AccountSubTypeIntangibleAssetsOther,
		AccountSubTypeLicenses,
		AccountSubTypeOrganizationalCosts,
		AccountSubTypeAssetsHeldForSale,
		AccountSubTypeAvailableForSaleFinancialAssets,
		AccountSubTypeDeferredTax,
		AccountSubTypeInvestments,
		AccountSubTypeLongTermInvestments,
		AccountSubTypeLongTermLoansAndAdvancesToRelatedParties,
		AccountSubTypeOtherIntangibleAssets,
		AccountSubTypeOtherLongTermInvestments,
		AccountSubTypeOtherLongTermLoansAndAdvances,
		AccountSubTypePrepaymentsAndAccruedIncome,
		AccountSubTypeProvisionsNonCurrentAssets,
	},
	AccountTypeAccountsReceivable: {
		AccountSubTypeAccountsReceivable,
	},
	AccountTypeEquity: {
		AccountSubTypeOpeningBalanceEquity,
		AccountSubTypePartnersEquity,
		AccountSubTypeRetainedEarnings,
		AccountSubTypeAccumulatedAdjustment,
		AccountSubTypeOwnersEquity,
		AccountSubTypePaidInCapitalOrSurplus,
		AccountSubTypePartnerContributions,
		AccountSubTypePartnerDistributions,
		AccountSubTypePreferredStock,
		AccountSubTypeCommonStock,
		AccountSubTypeTreasuryStock,
		AccountSubTypeEstimatedTaxes,
		AccountSubTypeHealthcare,
		AccountSubTypePersonalIncome,
		AccountSubTypePersonalExpense,
		AccountSubTypeAccumulatedOtherComprehensiveIncome,
		AccountSubTypeCalledUpShareCapital,
		AccountSubTypeCapitalReserves,
		AccountSubTypeDividendDisbursed,
		AccountSubTypeEquityInEarningsOfSubsiduaries,
		AccountSubTypeFunds,
		AccountSubTypeInvestmentGrants,
		AccountSubTypeMoneyReceivedAgainstShareWarrants,
		AccountSubTypeOtherFreeReserves,
		AccountSubTypeShareApplicationMoneyPendingAllotment,
		AccountSubTypeShareCapital,
		AccountSubTypeUnrealisedLossOnSecuritiesNetOfTax,
	},
	AccountTypeExpense: {
		AccountSubTypeAdvertisingPromotional,
		AccountSubTypeBadDebts,
		AccountSubTypeBankCharges,
		AccountSubTypeCharitableContributions,
		AccountSubTypeCommissionsAndFees,
		AccountSubTypeEntertainment,
		AccountSubTypeEntertainmentMeals,
		AccountSubTypeEquipmentRental,
		AccountSubTypeFinanceCosts,
		AccountSubTypeGlobalTaxExpense,
		AccountSubTypeInsurance,
		AccountSubTypeInterestPaid,
		AccountSubTypeLegalProfessionalFees,
		AccountSubTypeOfficeExpenses,
		AccountSubTypeOfficeGeneralAdministrativeExpenses,
		AccountSubTypeOtherBusinessExpenses,
		AccountSubTypeOtherMiscellaneousExpense,
		AccountSubTypePromotionalMeals,
		AccountSubTypeRentOrLeaseOfBuildings,
		AccountSubTypeRepairMaintenance,
		AccountSubTypeShippingFreightDelivery,
		AccountSubTypeSuppliesMaterials,
		AccountSubTypeTravel,
		AccountSubTypeTravelMeals,
		AccountSubTypeAuto,
		AccountSubTypeCostOfLabor,
		AccountSubTypeDuesSubscriptions,
		AccountSubTypePayrollExpenses,
		AccountSubTypeTaxesPaid,
		AccountSubTypeUnappliedCashBillPaymentExpense,
		AccountSubTypeUtilities,
		AccountSubTypeAmortizationExpense,
		AccountSubTypeAppropriationsToDepreciation,
		AccountSubTypeBorrowingCost,
		AccountSubTypeDistributionCosts,
		AccountSubTypeExternalServices,
		AccountSubTypeExtraordinaryCharges,
		AccountSubTypeIncomeTaxExpense,
		AccountSubTypeLossOnDiscontinuedOperationsNetOfTax,
		AccountSubTypeManagementCompensation,
		AccountSubTypeOtherCurrentOperatingCharges,
		AccountSubTypeOtherExternalServices,
		AccountSubTypeOtherRentalCosts,
		AccountSubTypeOtherSellingExpenses,
		AccountSubTypeProjectStudiesSurveysAssessments,
		AccountSubTypePurchasesRebates,
		AccountSubTypeShippingAndDeliveryExpense,
		AccountSubTypeStaffCosts,
		AccountSubTypeSundry,
		AccountSubTypeTravelExpensesGeneralAndAdminExpenses,
		AccountSubTypeTravelExpensesSellingExpense,
	},
	AccountTypeOtherExpense: {
		AccountSubTypeDepreciation,
		AccountSubTypeExchangeGainOrLoss,
		AccountSubTypeOtherMiscellaneousExpenseSub,
		AccountSubTypePenaltiesSettlements,
		AccountSubTypeAmortization,
		AccountSubTypeGasAndFuel,
		AccountSubTypeHomeOffice,
		AccountSubTypeHomeownerRentalInsurance,
		AccountSubTypeOtherHomeOfficeExpenses,
		AccountSubTypeMortgageInterestHomeOffice,
		AccountSubTypeRentAndLeaseHomeOffice,
		AccountSubTypeRepairsAndMaintainceHomeOffice,
		AccountSubTypeParkingAndTolls,
		AccountSubTypeVehicle,
		AccountSubTypeVehicleInsurance,
		AccountSubTypeVehicleLease,
		AccountSubTypeVehicleLoanInterest,
		AccountSubTypeVehicleRepairs,
		AccountSubTypeOtherVehicleExpenses,
		AccountSubTypeUtilitiesHomeOffice,
		AccountSubTypeWashAndRoadServices,
		AccountSubTypeDeferredTaxExpense,
		AccountSubTypeDepletion,
		AccountSubTypeExceptionalItems,
		AccountSubTypeExtraordinaryItems,
		AccountSubTypeIncomeTaxOtherExpense,
		AccountSubTypeMatCredit,
		AccountSubTypePriorPeriodItems,
		AccountSubTypeTaxRoundoffGainOrLoss,
	},
	AccountTypeCostOfGoodsSold: {
		AccountSubTypeEquipmentRentalCos,
		AccountSubTypeOtherCostsOfServiceCos,
		AccountSubTypeShippingFreightDeliveryCos,
		AccountSubTypeSuppliesMaterialsCogs,
		AccountSubTypeCostOfLaborCos,
		AccountSubTypeCostOfSales,
		AccountSubTypeFreightAndDeliveryCos,
	},
	AccountTypeAccountsPayable: {
		AccountSubTypeAccountsPayable,
		AccountSubTypeOutstandingDuesMicroSmallEnterprise,
		AccountSubTypeOutstandingDuesOtherThanMicroSmallEnterprise,
	},
	AccountTypeCreditCard: {
		AccountSubTypeCreditCard,
	},
	AccountTypeLongTermLiability: {
		AccountSubTypeNotesPayable,
		AccountSubTypeOtherLongTermLiabilities,
		AccountSubTypeShareholderNotesPayable,
		AccountSubTypeAccruedLongLermLiabilities,
		AccountSubTypeAccruedVacationPayable,
		AccountSubTypeBankLoans,
		AccountSubTypeDebtsRelatedToParticipatingInterests,
		AccountSubTypeDeferredTaxLiabilities,
		AccountSubTypeGovernmentAndOtherPublicAuthorities,
		AccountSubTypeGroupAndAssociates,
		AccountSubTypeLiabilitiesRelatedToAssetsHeldForSale,
		AccountSubTypeLongTermBorrowings,
		AccountSubTypeLongTermDebit,
		AccountSubTypeLongTermEmployeeBenefitObligations,
		AccountSubTypeObligationsUnderFinanceLeases,
		AccountSubTypeOtherLongTermProvisions,
		AccountSubTypeProvisionForLiabilities,
		AccountSubTypeProvisionsNonCurrentLiabilities,
		AccountSubTypeStaffAndRelatedLongTermLiabilityAccounts,
	},
	AccountTypeOtherCurrentLiab: {
		AccountSubTypeDirectDepositPayable,
		AccountSubTypeFederalIncomeTaxPayable,
		AccountSubTypeInsurancePayable,
		AccountSubTypeLineOfCredit,
		AccountSubTypeLoanPayable,
		AccountSubTypeGlobalTaxPayable,
		AccountSubTypeGlobalTaxSuspense,
		AccountSubTypeOtherCurrentLiabilities,
		AccountSubTypePayrollClearing,
		AccountSubTypePayrollTaxPayable,
		AccountSubTypePrepaidExpensesPayable,
		AccountSubTypeRentsInTrustLiability,
		AccountSubTypeTrustAccountsLiabilities,
		AccountSubTypeSalesTaxPayable,
		AccountSubTypeStateLocalIncomeTaxPayable,
		AccountSubTypeAccrualsAndDeferredIncome,
		AccountSubTypeAccruedLiabilities,
		AccountSubTypeCurrentLiabilities,
		AccountSubTypeCurrentPortionEmployeeBenefitsObligations,
		AccountSubTypeCurrentPortionOfObligationsUnderFinanceLeases,
		AccountSubTypeCurrentTaxLiability,
		AccountSubTypeDividendsPayable,
		AccountSubTypeDutiesAndTaxes,
		AccountSubTypeInterestPayables,
		AccountSubTypeProvisionForWarrantyObligations,
		AccountSubTypeProvisionsCurrentLiabilities,
		AccountSubTypeShortTermBorrowings,
		AccountSubTypeSocialSecurityAgencies,
		AccountSubTypeStaffAndRelatedLiabilityAccounts,
		AccountSubTypeSundryDebtorsAndCreditors,
		AccountSubTypeTradeAndOtherPayables,
	},
	AccountTypeIncome: {
		AccountSubTypeNonProfitIncome,
		AccountSubTypeOtherPrimaryIncome,
		AccountSubTypeSalesOfProductIncome,
		AccountSubTypeServiceFeeIncome,
		AccountSubTypeDiscountsRefundsGiven,
		AccountSubTypeUnappliedCashPaymentIncome,
		AccountSubTypeCashReceiptIncome,
		AccountSubTypeOperatingGrants,
		AccountSubTypeOtherCurrentOperatingIncome,
		AccountSubTypeOwnWorkCapitalized,
		AccountSubTypeRevenueGeneral,
		AccountSubTypeSalesRetail,
		AccountSubTypeSalesWholesale,
		AccountSubTypeSavingsByTaxScheme,
	},
	AccountTypeOtherIncome: {
		AccountSubTypeDividendIncome,
		AccountSubTypeInterestEarned,
		AccountSubTypeOtherInvestmentIncome,
		AccountSubTypeOtherMiscellaneousIncome,
		AccountSubTypeTaxExemptInterest,
		AccountSubTypeGainLossOnSaleOfFixedAssets,
		AccountSubTypeGainLossOnSaleOfInvestments,
		AccountSubTypeLossOnDisposalOfAssets,
		AccountSubTypeOtherOperatingIncome,
		AccountSubTypeUnrealisedLossOnSecuritiesNetOfTaxes,
	},
}

// AccountSubTypesFor returns the AccountSubTypeEnum values valid for the given
// AccountTypeEnum. The returned slice is a fresh copy. If the AccountType is
// unrecognised, AccountSubTypesFor returns nil.
func AccountSubTypesFor(t AccountTypeEnum) []AccountSubTypeEnum {
	subs, ok := accountSubTypesByType[t]
	if !ok {
		return nil
	}
	return append([]AccountSubTypeEnum(nil), subs...)
}

// accountClassificationsByType maps each AccountTypeEnum to its single
// canonical AccountClassificationEnum. QBO derives Classification from
// AccountType, so this mapping is one-to-one rather than one-to-many. It is
// exposed via AccountClassificationsFor for symmetry with AccountSubTypesFor
// and to support filter UIs.
var accountClassificationsByType = map[AccountTypeEnum]AccountClassificationEnum{
	AccountTypeBank:               AccountClassificationAsset,
	AccountTypeOtherCurrentAsset:  AccountClassificationAsset,
	AccountTypeFixedAsset:         AccountClassificationAsset,
	AccountTypeOtherAsset:         AccountClassificationAsset,
	AccountTypeAccountsReceivable: AccountClassificationAsset,
	AccountTypeEquity:             AccountClassificationEquity,
	AccountTypeExpense:            AccountClassificationExpense,
	AccountTypeOtherExpense:       AccountClassificationExpense,
	AccountTypeCostOfGoodsSold:    AccountClassificationExpense,
	AccountTypeAccountsPayable:    AccountClassificationLiability,
	AccountTypeCreditCard:         AccountClassificationLiability,
	AccountTypeLongTermLiability:  AccountClassificationLiability,
	AccountTypeOtherCurrentLiab:   AccountClassificationLiability,
	AccountTypeIncome:             AccountClassificationRevenue,
	AccountTypeOtherIncome:        AccountClassificationRevenue,
}

// AccountClassificationsFor returns the AccountClassificationEnum values
// associated with the given AccountTypeEnum. In QBO the relationship is
// one-to-one, so the returned slice contains at most one element. If the
// AccountType is unrecognised, AccountClassificationsFor returns nil.
func AccountClassificationsFor(t AccountTypeEnum) []AccountClassificationEnum {
	c, ok := accountClassificationsByType[t]
	if !ok {
		return nil
	}
	return []AccountClassificationEnum{c}
}
