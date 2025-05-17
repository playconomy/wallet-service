package main

import (
	"context"
	"log"

	"github.com/playconomy/wallet-service/docs"
	"github.com/playconomy/wallet-service/internal/module"
	
	"go.uber.org/fx"
)

//	@title			Wallet Service API
//	@version		1.0
//	@description	This is a wallet service API for managing platform tokens
//	@termsOfService	http://swagger.io/terms/

//	@contact.name	API Support
//	@contact.url	http://www.example.com/support
//	@contact.email	support@example.com

//	@license.name	Apache 2.0
//	@license.url	http://www.apache.org/licenses/LICENSE-2.0.html

//	@host		localhost:3000
//	@BasePath	/
//	@schemes	http https

//	@securityDefinitions.apikey	ApiKeyAuth
//	@in							header
//	@name						X-User-Id
//	@description				User ID for authentication

//	@securityDefinitions.apikey	ApiEmailAuth
//	@in							header
//	@name						X-User-Email
//	@description				User email for authentication

//	@securityDefinitions.apikey	ApiRoleAuth
//	@in							header
//	@name						X-User-Role
//	@description				User role for authentication

func main() {
	// Programmatically set swagger info
	docs.SwaggerInfo.Title = "Wallet Service API"
	docs.SwaggerInfo.Description = "This is a wallet service API for managing platform tokens"
	docs.SwaggerInfo.Version = "1.0"
	docs.SwaggerInfo.Host = "localhost:3000"
	docs.SwaggerInfo.BasePath = "/"
	docs.SwaggerInfo.Schemes = []string{"http", "https"}

	app := fx.New(
		module.Module,
	)

	if err := app.Start(context.Background()); err != nil {
		log.Fatalf("Failed to start application: %v", err)
	}

	<-app.Done()

	if err := app.Stop(context.Background()); err != nil {
		log.Fatalf("Failed to stop application: %v", err)
	}
}
