package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/ismaelucky94/near_u_exercise/internal/handlers"
	"github.com/ismaelucky94/near_u_exercise/internal/services"
)

// Dependencies bundles the services needed by the HTTP layer.
type Dependencies struct {
	Orders   *services.OrderService
	Accounts *services.AccountService
	Pricing  *services.PricingService
	Ready    func() bool
}

// Register wires all application routes to the provided gin Engine.
func Register(r *gin.Engine, deps Dependencies) {
	health := handlers.NewHealthHandler(deps.Ready)
	orders := handlers.NewOrdersHandler(deps.Orders)
	accounts := handlers.NewAccountsHandler(deps.Accounts)
	funds := handlers.NewFundsHandler(deps.Pricing)

	r.GET("/healthz", health.Liveness)
	r.GET("/readyz", health.Readiness)

	r.POST("/orders", orders.PlaceOrder)
	r.GET("/orders", orders.ListOrders)
	r.GET("/orders/:id", orders.GetOrder)
	r.POST("/orders/:id/cancel", orders.CancelOrder)
	r.GET("/orders/:id/events", orders.GetEvents)

	r.GET("/accounts/:id", accounts.GetAccount)

	r.POST("/funds/:id/navs", funds.PublishNAV)
}
