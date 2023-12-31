package business

import (
	"context"
	"errors"
	"strings"

	"go.mongodb.org/mongo-driver/mongo"
	apperrors "kego.com/application/appErrors"
	"kego.com/application/repository"
	walletUsecases "kego.com/application/usecases/wallet"
	"kego.com/entities"
	"kego.com/infrastructure/logger"
	"kego.com/infrastructure/validator"
)

func CreateBusiness(ctx any, payload *entities.Business) (*entities.Business, *entities.Wallet, error) {
	validationErr := validator.ValidatorInstance.ValidateStruct(*payload)
	if validationErr != nil {
		apperrors.ValidationFailedError(ctx, validationErr)
		return nil, nil, errors.New("")
	}
	businessRepo := repository.BusinessRepo()
	var business *entities.Business
	var wallet *entities.Wallet
	var err error
	businessRepo.StartTransaction(func(sc mongo.Session, c context.Context) error {
		payload = payload.ParseModel().(*entities.Business)
		walletPayload := &entities.Wallet{
			BusinessID: &payload.ID,
			UserID: payload.UserID,
			BusinessName: &payload.Name,
			Frozen: false,
			Balance: 0,
			LedgerBalance: 0,
			Currency: "NGN",
			LockedFundsLog: []entities.LockedFunds{},
		}
		walletPayload = walletPayload.ParseModel().(*entities.Wallet)
		payload.WalletID = walletPayload.ID
		b, e := businessRepo.CreateOne(c, *payload)
		if e != nil {
			logger.Error(errors.New("error creating users business"), logger.LoggerOptions{
				Key: "error",
				Data: e,
			}, logger.LoggerOptions{
				Key: "payload",
				Data: payload,
			})
			err = e
			(sc).AbortTransaction(c)
			return e
		}
		business = b
		w, e := walletUsecases.CreateWallet(ctx, c, walletPayload)
		if e != nil {
			logger.Error(errors.New("error creating users business wallet"), logger.LoggerOptions{
				Key: "error",
				Data: e,
			}, logger.LoggerOptions{
				Key: "payload",
				Data: walletPayload,
			})
			err = e
			(sc).AbortTransaction(c)
			return e
		}
		wallet = w
		(sc).CommitTransaction(c)
		return nil
	})
	if err != nil {
		if strings.Contains(err.Error(), "already exists"){
			apperrors.EntityAlreadyExistsError(ctx, err.Error())
			return nil, nil, err
		}else {
			apperrors.ClientError(ctx, err.Error(), nil)
			return nil, nil, err
		}
	}
	return business, wallet, nil
}