package authroutev1

import (
	"github.com/gin-gonic/gin"
	apperrors "kego.com/application/appErrors"
	"kego.com/application/controllers"
	"kego.com/application/controllers/dto"
	"kego.com/application/interfaces"
	middlewares "kego.com/infrastructure/middleware"
)


func UserRouter(router *gin.RouterGroup) {
	userRouter := router.Group("/user")
	{
		userRouter.GET("/profile", middlewares.AuthenticationMiddleware(false), func(ctx *gin.Context) {
			appContext, _ := ctx.MustGet("AppContext").(*interfaces.ApplicationContext[any])
			controllers.FetchUserProfile(appContext)
		})

		userRouter.PATCH("/profile/update", middlewares.AuthenticationMiddleware(false), func(ctx *gin.Context) {
			appContextAny, _ := ctx.MustGet("AppContext").(*interfaces.ApplicationContext[any])
			var body dto.UpdateUserDTO
			if err := ctx.ShouldBindJSON(&body); err != nil {
				apperrors.ErrorProcessingPayload(ctx)
				return
			}
			appContext := interfaces.ApplicationContext[dto.UpdateUserDTO]{
				Keys: appContextAny.Keys,
				Body: &body,
				Ctx: appContextAny.Ctx,
			}
			controllers.UpdateUserProfile(&appContext)
		})

		userRouter.PATCH("/profile/payment-tag", middlewares.AuthenticationMiddleware(false), func(ctx *gin.Context) {
			appContextAny, _ := ctx.MustGet("AppContext").(*interfaces.ApplicationContext[any])
			var body dto.SetPaymentTagDTO
			if err := ctx.ShouldBindJSON(&body); err != nil {
				apperrors.ErrorProcessingPayload(ctx)
				return
			}
			appContext := interfaces.ApplicationContext[dto.SetPaymentTagDTO]{
				Keys: appContextAny.Keys,
				Body: &body,
				Ctx: appContextAny.Ctx,
			}
			controllers.SetPaymentTag(&appContext)
		})
	}
}
