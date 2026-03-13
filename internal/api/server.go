package api

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"julescord/internal/config"
	"julescord/internal/db"
)

// Server handles the Gin REST API for the dashboard.
type Server struct {
	Engine *gin.Engine
	Server *http.Server
	Config *config.Config
	DB     *db.DB
	BootTime time.Time
}

// New initializes a new API server.
func New(cfg *config.Config, database *db.DB) *Server {
	// Use release mode in production-like setups, adjust as needed.
	gin.SetMode(gin.ReleaseMode)

	r := gin.Default()

	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"}, // Allowing all for local dev dashboard
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
	}))

	s := &Server{
		Engine:   r,
		Config:   cfg,
		DB:       database,
		BootTime: time.Now(),
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

	s.Engine.GET("/api/stats", func(c *gin.Context) {
		if s.DB == nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Database not connected"})
			return
		}

		guilds, users, cmds, err := s.DB.GetStats(c.Request.Context())
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch stats"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"guilds":   guilds,
			"users":    users,
			"commands": cmds,
			"uptime":   time.Since(s.BootTime).String(),
		})
	})

	s.Engine.GET("/api/guilds", func(c *gin.Context) {
		if s.DB == nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Database not connected"})
			return
		}

		guilds, err := s.DB.GetGuilds(c.Request.Context())
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch guilds"})
			return
		}

		c.JSON(http.StatusOK, guilds)
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
