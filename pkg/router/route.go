package router

import (
	"dcard-backend-2024/pkg/bootstrap"
	"dcard-backend-2024/pkg/controller"
	"dcard-backend-2024/pkg/middleware"
	"dcard-backend-2024/pkg/model"
)

type Services struct {
	UserService  model.UserService
	EventService model.EventService
	AsynqService model.AsynqNotificationService
}

func RegisterRoutes(app *bootstrap.Application, services *Services) {
	// Register Global Middleware
	cors := middleware.CORSMiddleware()
	app.Engine.Use(cors)

	// Register Event Routes
	eventController := controller.NewEventController(services.EventService, services.AsynqService)
	RegisterEventRouter(app, eventController)
}
