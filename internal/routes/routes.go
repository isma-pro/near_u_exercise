package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/ismaelucky94/near_u_exercise/internal/handlers"
)

// Register wires all application routes to the provided gin Engine.
func Register(r *gin.Engine) {
	health := handlers.NewHealthHandler()
	user := handlers.NewUserHandler()

	r.GET("/health", health.Check)

	api := r.Group("/api/v1")
	{
		api.GET("/users", user.List)
		api.POST("/users", user.Create)
		api.GET("/users/:id", user.Get)
		api.PUT("/users/:id", user.Update)
		api.DELETE("/users/:id", user.Delete)
	}
}
