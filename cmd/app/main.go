package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	// "path/filepath"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/sshfz/consumer-service-substrate/cmd/app/connections"
	// "github.com/sshfz/consumer-service-substrate/internal/consumers"
	// "github.com/sshfz/consumer-service-substrate/cmd/app/mq"
	// "github.com/sshfz/consumer-service-substrate/internal/utils"
	// "github.com/sshfz/consumer-service-substrate/internal/db"
	// "github.com/sshfz/consumer-service-substrate/internal/helpers"
)

func main() {
	router := gin.Default()
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.SetOutput(os.Stdout)

	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:3000", "https://your-frontend.com"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
	}))

	//running redis server locally ----
	connections.InitRedis()

	router.GET("/", func(ctx *gin.Context) {
		ctx.String(http.StatusOK, "substrate-consumer-service, release - 1.0.0")
	})

	log.Println("****** Getting substrate ready for stream ******")
	srv := &http.Server{
		Addr:    ":8090",
		Handler: router,
	}

	stopChan := make(chan struct{})
	startHeartbeat(stopChan)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)

	go func() {
		connections.StartConsumer()
	}()

	go func() {
		log.Println("server running on port http://localhost: 8090")
		log.Println("Substrate stream prepared, ping a prompt to spin up application")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %s\n", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)

	<-quit
	log.Println("Shutting down server...")
	close(stopChan)

	// graceful shutdown ---
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown")
	}

	log.Println("server exiting gracefully.")
}
