package controllers

import (
	"context"
	"fmt"
	"net/http"
	"os"

	apperrors "kego.com/application/appErrors"
	bankssupported "kego.com/application/banksSupported"
	"kego.com/application/controllers/dto"
	"kego.com/application/interfaces"
	"kego.com/application/repository"
	"kego.com/application/services"
	"kego.com/application/utils"
	"kego.com/entities"
	"kego.com/infrastructure/messaging/emails"
	pushnotification "kego.com/infrastructure/messaging/push_notifications"
	international_payment_processor "kego.com/infrastructure/payment_processor/chimoney"
	"kego.com/infrastructure/payment_processor/types"
	server_response "kego.com/infrastructure/serverResponse"
)

func InitiateBusinessInternationalPayment(ctx *interfaces.ApplicationContext[dto.SendPaymentDTO]){
	if ctx.Body.Amount < 1000 {
		apperrors.ClientError(ctx.Ctx, "You cannot send less than ₦10", nil)
		return
	}
	if ctx.Body.Amount >= 30000000000 {
		apperrors.ClientError(ctx.Ctx, "You cannot send more than ₦300,000,000 at a time", nil)
		return
	}
	businessID := ctx.GetStringParameter("businessID") 
	rates, statusCode, err := international_payment_processor.InternationalPaymentProcessor.GetExchangeRates(ctx.Body.DestinationCountryCode, ctx.Body.Amount)
	if err != nil {
		apperrors.ExternalDependencyError(ctx.Ctx, "chimoney", fmt.Sprintf("%d", statusCode), err)
		return
	}
	if statusCode != 200 {
		apperrors.UnknownError(ctx.Ctx)
		return
	}
	amountInNGN := utils.Float32ToUint64Currency((*rates)["convertedValue"])
	internationalProcessorFee, transactionFee  := utils.GetInternationalTransactionFee(amountInNGN)
	totalAmount := internationalProcessorFee + transactionFee + amountInNGN
	wallet , err := services.InitiatePreAuth(ctx.Ctx, businessID, ctx.GetStringContextData("UserID"), totalAmount, ctx.Body.Pin)
	if err != nil {
		return
	}
	err = services.LockFunds(ctx.Ctx, wallet, totalAmount, entities.ChimoneyDebitInternational)
	if err != nil {
		return
	}
	destinationCountry := utils.CountryCodeToCountryName(ctx.Body.DestinationCountryCode)
	if os.Getenv("GIN_MODE") != "release" {
		destinationCountry = utils.CountryCodeToCountryName("NG")
	}
	response := services.InitiateInternationalPayment(ctx.Ctx, &international_payment_processor.InternationalPaymentRequestPayload{
		DestinationCountry: destinationCountry,
		AccountNumber: ctx.Body.AccountNumber,
		BankCode: ctx.Body.BankCode,
		ValueInUSD: (*rates)["convertToUSD"],
	})
	if response == nil {
		return
	}
	transaction := entities.Transaction{
		TransactionReference: response.Chimoneys[0].ChiRef,
		MetaData: response.Chimoneys[0],
		AmountInUSD: utils.GetUInt64Pointer(utils.Float32ToUint64Currency(response.Chimoneys[0].ValueInUSD)),
		AmountInNGN: totalAmount,
		Fee: transactionFee,
		ProcessorFeeCurrency: "USD",
		ProcessorFee: internationalProcessorFee,
		Amount: ctx.Body.Amount,
		Currency: utils.CurrencyCodeToCurrencySymbol(ctx.Body.DestinationCountryCode),
		WalletID: wallet.ID,
		UserID: wallet.UserID,
		BusinessID: wallet.BusinessID,
		Description: func () string {
			if	ctx.Body.Description == nil {
				des := fmt.Sprintf("International transfer from %s %s to %s", ctx.GetStringContextData("FirstName"), ctx.GetStringContextData("LastName"), *ctx.Body.FullName)
				return des
			}
			return *ctx.Body.Description
		}(),
		Location: entities.Location{
			IPAddress: ctx.Body.IPAddress,
		},
		Intent: entities.ChimoneyDebitInternational,
		DeviceInfo: entities.DeviceInfo{
			IPAddress: ctx.Body.IPAddress,
			DeviceID: ctx.GetStringContextData("DeviceID"),
			UserAgent: ctx.GetStringContextData("UserAgent"),
		},
		Sender: entities.TransactionSender{
			BusinessName: *wallet.BusinessName,
			FirstName: ctx.GetStringContextData("FirstName"),
			LastName: ctx.GetStringContextData("LastName"),
			Email: ctx.GetStringContextData("Email"),
		},
		Recepient: entities.TransactionRecepient{
			Name: *ctx.Body.FullName,
			BankCode: ctx.Body.BankCode,
			AccountNumber: ctx.Body.AccountNumber,
			BranchCode: ctx.Body.BranchCode,
			Country: ctx.Body.DestinationCountryCode,
		},
	}
	trxRepository := repository.TransactionRepo()
	trx, err := trxRepository.CreateOne(context.TODO(), transaction)
	if err != nil {
		apperrors.FatalServerError(ctx.Ctx)
		return
	}
	pushnotification.PushNotificationService.PushOne(ctx.GetStringContextData("DeviceID"), "Your payment is on its way! 🚀",
		fmt.Sprintf("Your payment of %s%d to %s in %s is currently being processed.", utils.CurrencyCodeToCurrencySymbol(transaction.Currency), transaction.Amount, transaction.Recepient.Name, utils.CountryCodeToCountryName(transaction.Recepient.Country)))
	emails.EmailService.SendEmail(ctx.GetStringContextData("Email"), "Your payment is on its way! 🚀", "payment_sent", map[string]any{
		"FIRSTNAME": transaction.Sender.FirstName,
		"CURRENCY_CODE": utils.CurrencyCodeToCurrencySymbol(ctx.Body.DestinationCountryCode),
		"AMOUNT": utils.UInt64ToFloat32Currency(ctx.Body.Amount),
		"RECEPIENT_NAME": transaction.Recepient.Name,
		"RECEPIENT_COUNTRY": utils.CountryCodeToCountryName(transaction.Recepient.Country),
	})
	server_response.Responder.Respond(ctx.Ctx, http.StatusCreated, "Your payment is on its way! 🚀", trx, nil)
}

func InitiateBusinessLocalPayment(ctx *interfaces.ApplicationContext[dto.SendPaymentDTO]){
	if ctx.Body.Amount < 1000 {
		apperrors.ClientError(ctx.Ctx, "You cannot send less than ₦10", nil)
		return
	}
	if ctx.Body.Amount >= 30000000000 {
		apperrors.ClientError(ctx.Ctx, "You cannot send more than ₦300,000,000 at a time", nil)
		return
	}
	businessID := ctx.GetStringParameter("businessID") 
	localProcessorFee, polymerFee := utils.GetLocalTransactionFee(ctx.Body.Amount)
	totalAmount := ctx.Body.Amount + utils.Float32ToUint64Currency(localProcessorFee) + utils.Float32ToUint64Currency(polymerFee)
	wallet , err := services.InitiatePreAuth(ctx.Ctx, businessID, ctx.GetStringContextData("UserID"), totalAmount, ctx.Body.Pin)
	if err != nil {
		return
	}
	err = services.LockFunds(ctx.Ctx, wallet, totalAmount, entities.FlutterwaveDebitLocal)
	if err != nil {
		return
	}
	bankName := ""
	for _, bank := range bankssupported.SupportedLocalBanks {
		if bank.Code == ctx.Body.BankCode {
			bankName =  bank.Name
			break
		}
	}
	if bankName == "" {
		apperrors.NotFoundError(ctx.Ctx, "Selected bank is not currently supported")
		return
	}
	narration := func () string {
		if	ctx.Body.Description == nil {
			des := fmt.Sprintf("NGN Transfer from %s %s to %s", ctx.GetStringContextData("FirstName"), ctx.GetStringContextData("LastName"), bankName)
			return des
		}
		return *ctx.Body.Description
	}()
	reference := utils.GenerateUUIDString()
	response := services.InitiateLocalPayment(ctx.Ctx, &types.InitiateLocalTransferPayload{
		AccountNumber: ctx.Body.AccountNumber,
		AccountBank: ctx.Body.BankCode,
		Currency: "NGN",
		Amount: totalAmount,
		Narration: narration ,
		Reference: reference,
		DebitCurrency: "NGN",
		CallbackURL: "https://webhook.site/e5cf956e-63a9-4beb-8960-1b45f61fc42c",
	})
	if response == nil {
		return
	}
	transaction := entities.Transaction{
		TransactionReference: reference,
		MetaData: response,
		AmountInNGN: totalAmount,
		Fee: utils.Float32ToUint64Currency(polymerFee),
		ProcessorFeeCurrency: "NGN",
		ProcessorFee: utils.Float32ToUint64Currency(localProcessorFee),
		Amount: totalAmount,
		Currency: "NGN",
		WalletID: wallet.ID,
		UserID: wallet.UserID,
		BusinessID: wallet.BusinessID,
		Description: narration,
		Location: entities.Location{
			IPAddress: ctx.Body.IPAddress,
		},
		Intent: entities.FlutterwaveDebitLocal,
		DeviceInfo: entities.DeviceInfo{
			IPAddress: ctx.Body.IPAddress,
			DeviceID: ctx.GetStringContextData("DeviceID"),
			UserAgent: ctx.GetStringContextData("UserAgent"),
		},
		Sender: entities.TransactionSender{
			BusinessName: *wallet.BusinessName,
			FirstName: ctx.GetStringContextData("FirstName"),
			LastName: ctx.GetStringContextData("LastName"),
			Email: ctx.GetStringContextData("Email"),
		},
		Recepient: entities.TransactionRecepient{
			Name: response.FullName,
			BankCode: ctx.Body.BankCode,
			AccountNumber: ctx.Body.AccountNumber,
			BranchCode: ctx.Body.BranchCode,
			BankName: bankName,
			Country: "Nigeria",
		},
	}
	trxRepository := repository.TransactionRepo()
	trx, err := trxRepository.CreateOne(context.TODO(), transaction)
	if err != nil {
		apperrors.FatalServerError(ctx.Ctx)
		return
	}
	pushnotification.PushNotificationService.PushOne(ctx.GetStringContextData("DeviceID"), "Your payment is on its way! 🚀",
		fmt.Sprintf("Your payment of %s%d to %s in %s is currently being processed.", utils.CurrencyCodeToCurrencySymbol(transaction.Currency), transaction.Amount, transaction.Recepient.Name, utils.CountryCodeToCountryName(transaction.Recepient.Country)))
	emails.EmailService.SendEmail(ctx.GetStringContextData("Email"), "Your payment is on its way! 🚀", "payment_sent", map[string]any{
		"FIRSTNAME": transaction.Sender.FirstName,
		"CURRENCY_CODE": utils.CurrencyCodeToCurrencySymbol("NGN"),
		"AMOUNT": utils.UInt64ToFloat32Currency(ctx.Body.Amount),
		"RECEPIENT_NAME": transaction.Recepient.Name,
		"RECEPIENT_COUNTRY": utils.CountryCodeToCountryName(transaction.Recepient.Country),
	})
	server_response.Responder.Respond(ctx.Ctx, http.StatusCreated, "Your payment is on its way! 🚀", trx, nil)
}

func BusinessLocalPaymentFee(ctx *interfaces.ApplicationContext[dto.SendPaymentDTO]){
	if ctx.Body.Amount < 1000 {
		apperrors.ClientError(ctx.Ctx, "You cannot send less than ₦10", nil)
		return
	}
	if ctx.Body.Amount >= 30000000000 {
		apperrors.ClientError(ctx.Ctx, "You cannot send more than ₦300,000,000 at a time", nil)
		return
	}
	localProcessorFee, polymerFee := utils.GetLocalTransactionFee(ctx.Body.Amount)
	server_response.Responder.Respond(ctx.Ctx, http.StatusOK, "fee calculated", map[string]any{
		"processorFee": utils.Float32ToUint64Currency(localProcessorFee),
		"polymerFee": utils.Float32ToUint64Currency(polymerFee),
	}, nil)
}

func BusinessInternationalPaymentFee(ctx *interfaces.ApplicationContext[dto.SendPaymentDTO]){
	if ctx.Body.Amount < 1000 {
		apperrors.ClientError(ctx.Ctx, "You cannot send less than ₦10", nil)
		return
	}
	if ctx.Body.Amount >= 30000000000 {
		apperrors.ClientError(ctx.Ctx, "You cannot send more than ₦300,000,000 at a time", nil)
		return
	}
	internationalProcessorFee, polymerFee := utils.GetInternationalTransactionFee(ctx.Body.Amount)
	server_response.Responder.Respond(ctx.Ctx, http.StatusOK, "fee calculated", map[string]any{
		"processorFee": internationalProcessorFee,
		"polymerFee": polymerFee,
	}, nil)
}

func VerifyLocalAccountName(ctx *interfaces.ApplicationContext[dto.NameVerificationDTO]){
	bankCode := ""
	for _, bank := range bankssupported.SupportedLocalBanks {
		if bank.Name == ctx.Body.BankName {
			bankCode = bank.Code
			break
		}
	}
	if bankCode  == "" {
		apperrors.NotFoundError(ctx.Ctx, fmt.Sprintf("%s is not a supported bank on our platform yet.", ctx.Body.BankName))
		return
	}
	name := services.NameVerification(ctx.Ctx, ctx.Body.AccountNumber, bankCode)
	if name == nil {
		return
	}
	server_response.Responder.Respond(ctx.Ctx, http.StatusOK, "name verification complete", map[string]string{
		"name": *name,
	}, nil)
}