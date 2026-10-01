package main

import (
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/ismaelucky94/near_u_exercise/internal/config"
	"github.com/ismaelucky94/near_u_exercise/internal/routes"
)

func main() {
	cfg := config.Load()

	if cfg.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.Default()

	routes.Register(r)

	if err := r.Run(":" + cfg.Port); err != nil {
		log.Printf("Server failed: %v", err)
		os.Exit(1)
	}
}
