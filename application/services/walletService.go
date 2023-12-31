package services

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	apperrors "kego.com/application/appErrors"
	"kego.com/application/constants"
	"kego.com/application/repository"
	wallet_constants "kego.com/application/services/constants"
	"kego.com/application/utils"
	"kego.com/entities"
	"kego.com/infrastructure/cryptography"
	"kego.com/infrastructure/database/repository/cache"
	"kego.com/infrastructure/logger"
)


func GetWalletByBusinessID(ctx any, id string, userID string) (*entities.Wallet, error) {
	walletRepository := repository.WalletRepo()
	wallet, err := walletRepository.FindOneByFilter(map[string]interface{}{
		"businessID": id,
	})
	if err != nil {
		logger.Error(errors.New("error fetching a business wallet"), logger.LoggerOptions{
			Key: "error",
			Data: err,
		}, logger.LoggerOptions{
			Key: "businessID",
			Data: id,
		})
		apperrors.FatalServerError(ctx)
		return nil, err
	}
	if wallet == nil {
		err := fmt.Errorf("This wallet was not found. Please contact support on %s to help resolve this issue.", constants.SUPPORT_EMAIL)
		apperrors.NotFoundError(ctx, err.Error())
		return nil, err
	} 
	return wallet, nil
}

func GetWalletByUserID(ctx any, id string) (*entities.Wallet, error) {
	walletRepository := repository.WalletRepo()
	wallet, err := walletRepository.FindOneByFilter(map[string]interface{}{
		"userID": id,
	})
	if err != nil {
		logger.Error(errors.New("error fetching a userID wallet"), logger.LoggerOptions{
			Key: "error",
			Data: err,
		}, logger.LoggerOptions{
			Key: "userID",
			Data: id,
		})
		apperrors.FatalServerError(ctx)
		return nil, err
	}
	if wallet == nil {
		err := fmt.Errorf("This wallet was not found. Please contact support on %s to help resolve this issue.", constants.SUPPORT_EMAIL)
		apperrors.NotFoundError(ctx, err.Error())
		return nil, err
	} 
	return wallet, nil
}

func FreezeWallet(ctx any, walletID string, userID string, reason wallet_constants.FrozenAccountReason, time wallet_constants.FrozenAccountTime) (bool, error) {
	walletRepository := repository.WalletRepo()
	frozenWalletLogRepository := repository.FrozenWalletLogRepo()
	affected, err := walletRepository.UpdatePartialByID(walletID, map[string]any{
		"frozen": true,
	})
	if err != nil || affected == 0 {
		logger.Error(errors.New("could not freeze wallet"), logger.LoggerOptions{
			Key: "reason",
			Data: reason,
		}, logger.LoggerOptions{
			Key: "walletID",
			Data: walletID,
		},logger.LoggerOptions{
			Key: "userID",
			Data: userID,
		}, logger.LoggerOptions{
			Key: "error",
			Data: err,
		})
		apperrors.UnknownError(ctx)
		return false, err
	}
	_, err = frozenWalletLogRepository.CreateOne(nil, entities.FrozenWalletLog{
		Unfrozen: false,
		Reason: reason,
		WalletID: walletID,
		UserID: userID,
		Time: time,
	})
	if err != nil {
		logger.Error(errors.New("could not create frozen wallet log"), logger.LoggerOptions{
			Key: "reason",
			Data: reason,
		}, logger.LoggerOptions{
			Key: "walletID",
			Data: walletID,
		},logger.LoggerOptions{
			Key: "userID",
			Data: userID,
		}, logger.LoggerOptions{
			Key: "error",
			Data: err,
		})
		apperrors.UnknownError(ctx)
		return false, err
	}
	return true, nil
}

func verifyTransactionPinByUserID(ctx any, userID string, pin string) (bool, error){
	currentTries := cache.Cache.FindOne(fmt.Sprintf("%s-transaction-pin-tries", userID))
	if currentTries == nil {
		defaultTries := "0"
		currentTries = &defaultTries
	}
	currentTriesInt, err := strconv.Atoi(*currentTries)
	if err != nil {
		logger.Error(errors.New("error parsing users transaction pin current tries"), logger.LoggerOptions{
			Key: "error",
			Data: err,
		}, logger.LoggerOptions{
			Key: "key",
			Data: fmt.Sprintf("%s-transaction-pin-tries", userID),
		}, logger.LoggerOptions{
			Key: "data",
			Data: currentTries,
		})
		apperrors.FatalServerError(ctx)
		return false, err
	}
	if currentTriesInt == constants.MAX_TRANSACTION_PIN_TRIES {
		err = errors.New("You have exceeded the number of tries for your transaction pin and your account has been temporarily locked for 5 days.")
		apperrors.AuthenticationError(ctx, err.Error())
		return false, err
	}
	userRepository := repository.UserRepo()
	user, err := userRepository.FindByID(userID)
	if err != nil {
		logger.Error(errors.New("error fetching a user account"), logger.LoggerOptions{
			Key: "error",
			Data: err,
		}, logger.LoggerOptions{
			Key: "userID",
			Data: userID,
		})
		return false, err
	}
	if user == nil {
		err = fmt.Errorf("This user profile was not found. Please contact support on %s to help resolve this issue.", constants.SUPPORT_EMAIL)
		apperrors.NotFoundError(ctx, err.Error())
		return false, err
	} 
	pinMatch := cryptography.CryptoHahser.VerifyData(user.TransactionPin, pin)
	if !pinMatch {
		currentTriesInt =  currentTriesInt + 1
		cache.Cache.CreateEntry(fmt.Sprintf("%s-transaction-pin-tries", userID), fmt.Sprintf("%d", currentTriesInt), time.Hour * 24 * 5)
		err = errors.New("wrong pin")
		apperrors.NotFoundError(ctx, err.Error())
		return false, err
	}
	cache.Cache.CreateEntry(fmt.Sprintf("%s-transaction-pin-tries", userID), 0, 0)
	return pinMatch, nil
}

func verifyWalletBalance(ctx any, wallet *entities.Wallet, amount uint64) (bool, error) {
	if wallet.Frozen {
		err := fmt.Errorf("This wallet has been frozen and cannot carry out any transaction at the moment. Please contact support on %s to help resolve this issue.", constants.SUPPORT_EMAIL)
		apperrors.AuthenticationError(ctx, err.Error())
		return false, err
	}
	if wallet.Balance < amount {
		err := fmt.Errorf("Insufficient funds. Credit your account with at least %s%v to complete this transaction.", wallet.Currency, utils.UInt64ToFloat32Currency(amount))
		apperrors.ClientError(ctx, err.Error(), nil)
		return false, err
	}
	return true, nil
}

func InitiatePreAuth(ctx any, businessID string, userID string, amount uint64, pin string) (*entities.Wallet, error) {
	wallet, err := GetWalletByBusinessID(ctx, businessID, userID)
	if err != nil {
		return nil, err
	}
	success, err := verifyTransactionPinByUserID(ctx, userID, pin)
	if err != nil  || !success{
		return nil, err
	}
	success, err = verifyWalletBalance(ctx, wallet, amount)
	if err != nil  || !success {
		return nil, err
	}
	return wallet, nil
}

func LockFunds(ctx any, wallet *entities.Wallet, amount uint64, intent entities.TransactionIntent) error {
	lockedFundsLog := entities.LockedFunds{
		LockedFundsID: utils.GenerateUUIDString(),
		Amount: amount,
		LockedAt: time.Now(),
		Reason: intent,
	}
	walletRepository := repository.WalletRepo()
	affected, err := walletRepository.UpdateManyWithOperator(map[string]interface{}{
		"_id": wallet.ID,
	}, map[string]any{
		"$push": map[string]any{
			"lockedFundsLog": lockedFundsLog,
		},
		"$inc": map[string]any {
			"balance": int64(-amount),
		},
	})
	if err != nil || affected == 0 {
		logger.Error(errors.New("could not lock funds"), logger.LoggerOptions{
			Key: "intent",
			Data: intent,
		}, logger.LoggerOptions{
			Key: "wallet",
			Data: wallet,
		}, logger.LoggerOptions{
			Key: "error",
			Data: err,
		})
		apperrors.UnknownError(ctx)
		return err
	}
	return nil
}