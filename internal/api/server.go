package api

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"julescord/internal/config"
	"julescord/internal/db"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for the dashboard
	},
}

// Server handles the Gin REST API for the dashboard.
type Server struct {
	Engine  *gin.Engine
	Server  *http.Server
	Config  *config.Config
	DB      *db.DB
	StartAt time.Time
}

// New initializes a new API server.
func New(cfg *config.Config, database *db.DB) *Server {
	// Use release mode in production-like setups, adjust as needed.
	gin.SetMode(gin.ReleaseMode)

	r := gin.Default()

	// Add CORS middleware
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:5173", "http://127.0.0.1:5173"}, // Vite default port
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	s := &Server{
		Engine:  r,
		Config:  cfg,
		DB:      database,
		StartAt: time.Now(),
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
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Database not available"})
			return
		}

		guilds, users, cmds, err := s.DB.GetStats(c.Request.Context())
		if err != nil {
			log.Printf("Error fetching stats: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch stats"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"guilds":       guilds,
			"users":        users,
			"commands_run": cmds,
			"uptime":       time.Since(s.StartAt).String(),
		})
	})

	s.Engine.GET("/api/guilds", func(c *gin.Context) {
		if s.DB == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Database not available"})
			return
		}

		guilds, err := s.DB.GetGuilds(c.Request.Context())
		if err != nil {
			log.Printf("Error fetching guilds: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch guilds"})
			return
		}

		c.JSON(http.StatusOK, guilds)
	})

	s.Engine.GET("/api/users", func(c *gin.Context) {
		if s.DB == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Database not available"})
			return
		}

		users, err := s.DB.GetUsersWithEconomy(c.Request.Context())
		if err != nil {
			log.Printf("Error fetching users: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch users"})
			return
		}

		c.JSON(http.StatusOK, users)
	})

	s.Engine.GET("/api/mod-actions", func(c *gin.Context) {
		if s.DB == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Database not available"})
			return
		}

		actions, err := s.DB.GetModActions(c.Request.Context())
		if err != nil {
			log.Printf("Error fetching mod actions: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch mod actions"})
			return
		}

		c.JSON(http.StatusOK, actions)
	})

	s.Engine.GET("/api/guilds/:id/config", func(c *gin.Context) {
		if s.DB == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Database not available"})
			return
		}

		guildID := c.Param("id")
		config, err := s.DB.GetGuildConfig(c.Request.Context(), guildID)
		if err != nil {
			log.Printf("Error fetching guild config: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch guild config"})
			return
		}

		c.JSON(http.StatusOK, config)
	})

	s.Engine.PATCH("/api/guilds/:id/config", func(c *gin.Context) {
		if s.DB == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Database not available"})
			return
		}

		guildID := c.Param("id")

		var req struct {
			LogChannelID     *string `json:"log_channel_id"`
			WelcomeChannelID *string `json:"welcome_channel_id"`
			ModRoleID        *string `json:"mod_role_id"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
			return
		}

		config, err := s.DB.GetGuildConfig(c.Request.Context(), guildID)
		if err != nil {
			log.Printf("Error fetching guild config for update: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch guild config"})
			return
		}

		if req.LogChannelID != nil {
			config.LogChannelID = req.LogChannelID
		}
		if req.WelcomeChannelID != nil {
			config.WelcomeChannelID = req.WelcomeChannelID
		}
		if req.ModRoleID != nil {
			config.ModRoleID = req.ModRoleID
		}

		if err := s.DB.UpdateGuildConfig(c.Request.Context(), guildID, *config); err != nil {
			log.Printf("Error updating guild config: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update guild config"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"status": "success"})
	})

	s.Engine.GET("/api/stats/commands", func(c *gin.Context) {
		if s.DB == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Database not available"})
			return
		}

		stats, err := s.DB.GetCommandUsageStats(c.Request.Context())
		if err != nil {
			log.Printf("Error fetching command stats: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch command stats"})
			return
		}

		c.JSON(http.StatusOK, stats)
	})

	s.Engine.GET("/ws", func(c *gin.Context) {
		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			log.Printf("Failed to upgrade websocket: %v", err)
			return
		}
		defer conn.Close()

		ctx, cancel := context.WithCancel(c.Request.Context())
		defer cancel()

		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		// Send initial stats immediately
		s.sendWebSocketStats(conn, ctx)

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := s.sendWebSocketStats(conn, ctx); err != nil {
					log.Printf("WebSocket send error: %v", err)
					return // close connection on error
				}
			}
		}
	})
}

func (s *Server) sendWebSocketStats(conn *websocket.Conn, ctx context.Context) error {
	if s.DB == nil {
		return nil
	}

	guilds, users, cmds, err := s.DB.GetStats(ctx)
	if err != nil {
		return err
	}

	data := gin.H{
		"guilds":       guilds,
		"users":        users,
		"commands_run": cmds,
		"uptime":       time.Since(s.StartAt).String(),
	}

	return conn.WriteJSON(data)
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
