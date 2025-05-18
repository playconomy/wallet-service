package router

import (
	"github.com/playconomy/wallet-service/internal/server/handler"
	"github.com/playconomy/wallet-service/internal/server/middleware"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/swagger"
	"go.uber.org/fx"
)

// Module provides router dependencies
var Module = fx.Options(
	fx.Provide(NewRouter),
	// Provide interface implementation for dependency injection
	fx.Provide(func(r *Router) RouterInterface { return r }),
	fx.Invoke(SetupRoutes),
)

type Router struct {
	app           *fiber.App
	walletHandler handler.WalletHandlerInterface
}

// Compile-time verification that Router implements RouterInterface
var _ RouterInterface = (*Router)(nil)

func NewRouter(app *fiber.App, walletHandler handler.WalletHandlerInterface) *Router {
	return &Router{
		app:           app,
		walletHandler: walletHandler,
	}
}

func SetupRoutes(router RouterInterface) {
	// Setup the router
	router.Setup(router.(*Router).app)
}

// Setup configures all routes and middlewares for the application
func (r *Router) Setup(app *fiber.App) {
	r.RegisterMiddlewares(app)
	r.RegisterRoutes(app)
}

// RegisterMiddlewares registers global middlewares for the application
func (r *Router) RegisterMiddlewares(app *fiber.App) {
	// Global middlewares can be registered here
}

// RegisterRoutes registers all application routes
func (r *Router) RegisterRoutes(app *fiber.App) {
	// Swagger routes
	app.Get("/swagger/*", swagger.HandlerDefault)

	// Health check (unprotected)
	//	@Summary		Health check
	//	@Description	Check if the service is running
	//	@Tags			health
	//	@Produce		json
	//	@Success		200	{object}	map[string]string	"Health status"
	//	@Router			/health [get]
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok"})
	})

	// Create a group with auth middleware
	api := app.Group("/", middleware.AuthMiddleware())

	// Protected routes
	api.Get("/:user_id", r.walletHandler.GetWallet)
	api.Get("/:user_id/logs", r.walletHandler.GetWalletLogs)
	api.Post("/exchange", r.walletHandler.Exchange)
	api.Post("/spend", r.walletHandler.Spend)
}
