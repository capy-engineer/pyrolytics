package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"pyrolytics/config"
	"pyrolytics/internal/event_ingestor/application"
	"pyrolytics/internal/event_ingestor/domain"
	"pyrolytics/pkg/database"

	"github.com/gagliardetto/solana-go"
	"github.com/gin-gonic/gin"
)

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	db, err := database.New(&cfg.Database)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	dexSubscriber := application.NewDEXSubscriber(cfg)
	defer dexSubscriber.Close()

	programs := map[solana.PublicKey]string{
		domain.RaydiumAMMV4Devnet:   "Raydium-AMMV4",
		domain.RaydiumCPMMDevnet:    "Raydium-CPMM",
		domain.RaydiumCLMMDevnet:    "Raydium-CLMM",
		domain.RaydiumRoutingDevnet: "Raydium-Routing",
		// Note: Orca addresses might need to be updated for Devnet
		// OrcaWhirlpoolProgram: "Orca-Whirlpool",
	}

	for programID, name := range programs {
		if err := dexSubscriber.SubscribeToProgram(programID, name); err != nil {
			log.Printf("❌ Failed to subscribe to %s: %v", name, err)
		} else {
			log.Printf("✅ Successfully subscribed to %s", name)
		}
	}

	router := gin.Default()

	router.Use(gin.Recovery())
	router.Use(gin.Logger())

	router.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, "pong")
	})

	srv := &http.Server{
		Addr:    ":8080",
		Handler: router,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v\n", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}

	log.Println("Server exiting")
}
