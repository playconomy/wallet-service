// Package router provides HTTP routing functionality
package router

import (
	"github.com/gofiber/fiber/v2"
)

// RouterInterface defines the interface for application routing
type RouterInterface interface {
	// Setup configures all routes and middlewares for the application
	Setup(app *fiber.App)
	
	// RegisterMiddlewares registers global middlewares for the application
	RegisterMiddlewares(app *fiber.App)
	
	// RegisterRoutes registers all application routes
	RegisterRoutes(app *fiber.App)
}
