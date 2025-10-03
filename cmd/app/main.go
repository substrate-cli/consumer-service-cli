package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"

	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/substrate-cli/consumer-service-cli/cmd/app/connections"
	"github.com/substrate-cli/consumer-service-cli/internal/utils"
)

func main() {
	router := gin.Default()
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.SetOutput(os.Stdout)
	origins := utils.GetSafeOrigins()
	parts := strings.Split(origins, ",")
	safeOrigins := make([]string, 0, len(parts))
	for _, o := range parts {
		safeOrigins = append(safeOrigins, strings.TrimSpace(o))
	}

	router.Use(cors.New(cors.Config{
		AllowOrigins:     safeOrigins,
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
	}))

	connections.InitRedis()

	router.GET("/", func(ctx *gin.Context) {
		ctx.String(http.StatusOK, "substrate-consumer-service, release - 1.0.0")
	})

	log.Println("****** Getting substrate ready for stream ******")
	srv := &http.Server{
		Addr:    ":" + utils.GetAppPort(),
		Handler: router,
	}

	stopChan := make(chan struct{})
	startHeartbeat(stopChan)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)

	go func() {
		connections.StartConsumer()
	}()

	go func() {
		log.Println("server running on port http://localhost:" + utils.GetAppPort())
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

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

func findSubstring(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
