package api

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"julescord/internal/config"
	"julescord/internal/db"
	"julescord/internal/metrics"

	"github.com/bwmarrin/discordgo"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
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
	Session *discordgo.Session
	StartAt time.Time
}

// New initializes a new API server.
func New(cfg *config.Config, database *db.DB, session *discordgo.Session) *Server {
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
		Session: session,
		StartAt: time.Now(),
	}

	s.registerRoutes()

	return s
}

func (s *Server) registerRoutes() {
	s.Engine.GET("/health", func(c *gin.Context) {
		dbConnected := false
		if s.DB != nil && s.DB.Pool != nil {
			err := s.DB.Pool.Ping(c.Request.Context())
			if err == nil {
				dbConnected = true
			}
		}

		discordLatency := "unknown"
		if s.Session != nil {
			latency := s.Session.HeartbeatLatency()
			if latency > 0 {
				discordLatency = fmt.Sprintf("%dms", latency.Milliseconds())
			}
		}

		status := "ok"
		statusCode := http.StatusOK
		if !dbConnected {
			status = "degraded"
		}

		c.JSON(statusCode, gin.H{
			"status":          status,
			"db_connected":    dbConnected,
			"discord_latency": discordLatency,
		})
	})

	s.Engine.GET("/api/status", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "online",
			"bot":    "JulesCord",
		})
	})

	s.Engine.GET("/metrics", gin.WrapH(promhttp.Handler()))

	s.Engine.GET("/api/dashboard-metrics", func(c *gin.Context) {
		mfs, err := prometheus.DefaultGatherer.Gather()
		if err != nil {
			metrics.ErrorCounter.WithLabelValues("prometheus_gather").Inc()
			slog.Error("Failed to gather metrics", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch dashboard metrics"})
			return
		}

		type CommandMetric struct {
			Command    string  `json:"command"`
			Executions float64 `json:"executions"`
			Latency    float64 `json:"latency"`
			LatencySum float64 `json:"-"`
			LatencyCnt uint64  `json:"-"`
		}

		type DBLatencyMetric struct {
			Query      string  `json:"query"`
			Latency    float64 `json:"latency"`
			LatencySum float64 `json:"-"`
			LatencyCnt uint64  `json:"-"`
		}

		type ErrorMetric struct {
			Type  string  `json:"type"`
			Count float64 `json:"count"`
		}

		cmds := make(map[string]*CommandMetric)
		var dbLatency []DBLatencyMetric
		var errorRates []ErrorMetric

		for _, mf := range mfs {
			if *mf.Name == "julescord_command_executions_total" {
				for _, m := range mf.Metric {
					var cmdName string
					for _, lp := range m.Label {
						if *lp.Name == "command" {
							cmdName = *lp.Value
							break
						}
					}
					if cmdName != "" {
						if _, ok := cmds[cmdName]; !ok {
							cmds[cmdName] = &CommandMetric{Command: cmdName}
						}
						cmds[cmdName].Executions = *m.Counter.Value
					}
				}
			} else if *mf.Name == "julescord_command_latency_seconds" {
				for _, m := range mf.Metric {
					var cmdName string
					for _, lp := range m.Label {
						if *lp.Name == "command" {
							cmdName = *lp.Value
							break
						}
					}
					if cmdName != "" {
						if _, ok := cmds[cmdName]; !ok {
							cmds[cmdName] = &CommandMetric{Command: cmdName}
						}
						if m.Histogram != nil {
							cmds[cmdName].LatencySum = *m.Histogram.SampleSum
							cmds[cmdName].LatencyCnt = *m.Histogram.SampleCount
							if cmds[cmdName].LatencyCnt > 0 {
								// Calculate avg latency in ms
								cmds[cmdName].Latency = (cmds[cmdName].LatencySum / float64(cmds[cmdName].LatencyCnt)) * 1000.0
							}
						}
					}
				}
			} else if *mf.Name == "julescord_db_query_latency_seconds" {
				for _, m := range mf.Metric {
					var queryName string
					for _, lp := range m.Label {
						if *lp.Name == "query" {
							queryName = *lp.Value
							break
						}
					}
					if queryName != "" && m.Histogram != nil {
						sum := *m.Histogram.SampleSum
						count := *m.Histogram.SampleCount
						var avgMs float64
						if count > 0 {
							avgMs = (sum / float64(count)) * 1000.0
						}
						dbLatency = append(dbLatency, DBLatencyMetric{
							Query:      queryName,
							Latency:    avgMs,
							LatencySum: sum,
							LatencyCnt: count,
						})
					}
				}
			} else if *mf.Name == "julescord_errors_total" {
				for _, m := range mf.Metric {
					var errType string
					for _, lp := range m.Label {
						if *lp.Name == "type" {
							errType = *lp.Value
							break
						}
					}
					if errType != "" && m.Counter != nil {
						errorRates = append(errorRates, ErrorMetric{
							Type:  errType,
							Count: *m.Counter.Value,
						})
					}
				}
			}
		}

		var commandsList []CommandMetric
		for _, cmd := range cmds {
			commandsList = append(commandsList, *cmd)
		}

		c.JSON(http.StatusOK, gin.H{
			"commands":   commandsList,
			"dbLatency":  dbLatency,
			"errorRates": errorRates,
		})
	})

	s.Engine.GET("/api/stats", func(c *gin.Context) {
		if s.DB == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Database not available"})
			return
		}

		guilds, users, cmds, err := s.DB.GetStats(c.Request.Context())
		if err != nil {
			metrics.ErrorCounter.WithLabelValues("db_get_stats").Inc()
			slog.Error("Error fetching stats", "error", err)
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
			metrics.ErrorCounter.WithLabelValues("db_get_guilds").Inc()
			slog.Error("Error fetching guilds", "error", err)
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
			metrics.ErrorCounter.WithLabelValues("db_get_users").Inc()
			slog.Error("Error fetching users", "error", err)
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
			metrics.ErrorCounter.WithLabelValues("db_get_mod_actions").Inc()
			slog.Error("Error fetching mod actions", "error", err)
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
			metrics.ErrorCounter.WithLabelValues("db_get_guild_config").Inc()
			slog.Error("Error fetching guild config", "error", err)
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
			metrics.ErrorCounter.WithLabelValues("db_get_guild_config_update").Inc()
			slog.Error("Error fetching guild config for update", "error", err)
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
			metrics.ErrorCounter.WithLabelValues("db_update_guild_config").Inc()
			slog.Error("Error updating guild config", "error", err)
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
			metrics.ErrorCounter.WithLabelValues("db_get_command_stats").Inc()
			slog.Error("Error fetching command stats", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch command stats"})
			return
		}

		c.JSON(http.StatusOK, stats)
	})

	s.Engine.GET("/ws", func(c *gin.Context) {
		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			slog.Error("Failed to upgrade websocket", "error", err)
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
					slog.Error("WebSocket send error", "error", err)
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

	slog.Info(fmt.Sprintf("Starting API server on %s", addr))
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

	slog.Info("Stopping API server...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := s.Server.Shutdown(ctx); err != nil {
		return err
	}

	slog.Info("API server stopped gracefully.")
	return nil
}
