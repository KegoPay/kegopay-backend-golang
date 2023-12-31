package apperrors

import (
	"errors"
	"fmt"
	"net/http"

	"kego.com/infrastructure/logger"
	"kego.com/infrastructure/logger/metrics"
	server_response "kego.com/infrastructure/serverResponse"
)

func NotFoundError(ctx interface{}, message string){
	server_response.Responder.Respond(ctx, http.StatusNotFound, message, nil, nil)
}

func ValidationFailedError(ctx interface{}, errMessages *[]error){
	server_response.Responder.Respond(ctx, http.StatusUnprocessableEntity, "Payload validation failed 🙄", nil, *errMessages)
}

func EntityAlreadyExistsError(ctx interface{}, message string){
	server_response.Responder.Respond(ctx, http.StatusConflict, message, nil, nil)
}

func AuthenticationError(ctx interface{}, message string){
	server_response.Responder.Respond(ctx, http.StatusUnauthorized, message, nil, nil)
}

func ExternalDependencyError(ctx interface{}, serviceName string, statusCode string, err error) {
	logger.Error(err, logger.LoggerOptions{
		Key: fmt.Sprintf("error with %s. status code %s", serviceName, statusCode),
	})
	server_response.Responder.Respond(ctx, http.StatusServiceUnavailable,
		"Omo! Our service is temporarily down 😢. Our team is working to fix it. Please check back later.", nil, nil)
}

func ErrorProcessingPayload(ctx interface{}){
	server_response.Responder.Respond(ctx, http.StatusBadRequest, "Abnormal payload passed 🤨", nil, nil)
}

func FatalServerError(ctx interface{}){
	metrics.MetricMonitor.ReportError(ctx, errors.New("fatal server error"))
	server_response.Responder.Respond(ctx, http.StatusInternalServerError,
		"Omo! Our service is temporarily down 😢. Our team is working to fix it. Please check back later.", nil, nil)
}

func UnknownError(ctx interface{}){
	metrics.MetricMonitor.ReportError(ctx, errors.New("an unknown error occured"))
	server_response.Responder.Respond(ctx, http.StatusBadRequest,
		"Omo! Something went wrong somewhere 😭. Please check back later.", nil, nil)
}

func CustomError(ctx interface{}, msg string){
	server_response.Responder.Respond(ctx, http.StatusBadRequest, msg, nil, nil)
}

func UnsupportedAppVersion(ctx interface{}){
	server_response.Responder.Respond(ctx, http.StatusBadRequest,
		"Uh oh! Seems you're using an old version of the app. 🤦🏻‍♂️\n Upgrade to the latest version to continue enjoying our blazing fast services! 🚀", nil, nil)
}

func UnsupportedUserAgent(ctx interface{}){
	metrics.MetricMonitor.ReportError(ctx, errors.New("unspported user agent"))
	server_response.Responder.Respond(ctx, http.StatusBadRequest,
		"Unsupported user agent 👮🏻‍♂️", nil, nil)
}

func ClientError(ctx interface{}, msg string, errs []error){
	server_response.Responder.Respond(ctx, http.StatusBadRequest, msg, nil, errs)
}
