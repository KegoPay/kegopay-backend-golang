package constants

const (
	SUPPORT_EMAIL string = "support@kego.com"
	BUSINESS_WALLET_LIMIT int = 11
	MAX_TRANSACTION_PIN_TRIES  int = 3
	INTERNATIONAL_TRANSACTION_FEE_RATE float32 = 0.01
	INTERNATIONAL_PROCESSOR_FEE_RATE float32 = 0.005
	LOCAL_TRANSACTION_FEE_VAT float32 = 0.075
	LOCAL_TRANSACTION_FEE_RATE float32 = 0.5
	LOCAL_PROCESSOR_FEE_LT_5000 float32 = 10.00
	LOCAL_PROCESSOR_FEE_LT_50000 float32 = 25.00
	LOCAL_PROCESSOR_FEE_GT_50000 float32 = 50.00
)