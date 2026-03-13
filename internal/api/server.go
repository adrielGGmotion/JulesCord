package api

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"julescord/internal/config"
)

// Server handles the Gin REST API for the dashboard.
type Server struct {
	Engine *gin.Engine
	Server *http.Server
	Config *config.Config
}

// New initializes a new API server.
func New(cfg *config.Config) *Server {
	// Use release mode in production-like setups, adjust as needed.
	gin.SetMode(gin.ReleaseMode)

	r := gin.Default()

	s := &Server{
		Engine: r,
		Config: cfg,
	}

	s.registerRoutes()

	return s
}

func (s *Server) registerRoutes() {
	s.Engine.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "ok",
		})
	})

	s.Engine.GET("/api/status", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "online",
			"bot":    "JulesCord",
		})
	})
}

// Start begins listening and serving HTTP traffic.
func (s *Server) Start() error {
	addr := ":" + s.Config.APIPort
	s.Server = &http.Server{
		Addr:    addr,
		Handler: s.Engine,
	}

	log.Printf("Starting API server on %s", addr)
	if err := s.Server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

// Stop gracefully shuts down the server.
func (s *Server) Stop() error {
	if s.Server == nil {
		return nil
	}

	log.Println("Stopping API server...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := s.Server.Shutdown(ctx); err != nil {
		return err
	}

	log.Println("API server stopped gracefully.")
	return nil
}
