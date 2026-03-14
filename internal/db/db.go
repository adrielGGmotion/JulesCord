package db

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"julescord/internal/metrics"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DB wraps the pgxpool connection pool.
type DB struct {
	Pool *pgxpool.Pool
}

// New establishes a connection pool to the database.
func New(databaseURL string) (*DB, error) {
	if databaseURL == "" {
		return nil, fmt.Errorf("database URL is required")
	}

	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("error parsing database URL: %w", err)
	}

	pool, err := pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		return nil, fmt.Errorf("error creating database pool: %w", err)
	}

	// Ping to ensure connection is valid
	if err := pool.Ping(context.Background()); err != nil {
		return nil, fmt.Errorf("error connecting to database: %w", err)
	}

	slog.Info("Connected to PostgreSQL database successfully.")

	return &DB{Pool: pool}, nil
}

// Close gracefully closes the database connection pool.
func (db *DB) Close() {
	if db.Pool != nil {
		db.Pool.Close()
		slog.Info("Database connection closed gracefully.")
	}
}

// RunMigrations executes database migrations from the migrations folder.
func RunMigrations(databaseURL string) error {
	slog.Info("Running database migrations...")

	m, err := migrate.New("file://migrations", databaseURL)
	if err != nil {
		return fmt.Errorf("could not create migrate instance: %w", err)
	}
	defer m.Close()

	err = m.Up()
	if err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			slog.Info("Database is already up to date.")
			return nil
		}
		return fmt.Errorf("could not run up migrations: %w", err)
	}

	slog.Info("Database migrations applied successfully.")
	return nil
}

// UpsertGuild inserts a new guild or ignores if it already exists.
func (db *DB) UpsertGuild(ctx context.Context, id string) error {
	query := `
		INSERT INTO guilds (id)
		VALUES ($1)
		ON CONFLICT (id) DO NOTHING
	`
	_, err := db.Pool.Exec(ctx, query, id)
	return err
}

// Ticket represents a support ticket in a guild.
type Ticket struct {
	ID        int       `json:"id"`
	GuildID   string    `json:"guild_id"`
	UserID    string    `json:"user_id"`
	ChannelID string    `json:"channel_id"`
	Reason    string    `json:"reason"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	ClosedAt  *time.Time `json:"closed_at"`
}

// CreateTicket creates a new ticket in the database.
func (db *DB) CreateTicket(ctx context.Context, guildID, userID, channelID, reason string) error {
	start := time.Now()
	metrics.DBQueryLatency.WithLabelValues("CreateTicket").Observe(time.Since(start).Seconds())

	query := `
		INSERT INTO tickets (guild_id, user_id, channel_id, reason, status, created_at)
		VALUES ($1, $2, $3, $4, 'open', NOW())
	`
	_, err := db.Pool.Exec(ctx, query, guildID, userID, channelID, reason)
	return err
}

// GetTicketByChannel retrieves a ticket by its channel ID.
func (db *DB) GetTicketByChannel(ctx context.Context, channelID string) (*Ticket, error) {
	start := time.Now()
	metrics.DBQueryLatency.WithLabelValues("GetTicketByChannel").Observe(time.Since(start).Seconds())

	query := `
		SELECT id, guild_id, user_id, channel_id, reason, status, created_at, closed_at
		FROM tickets
		WHERE channel_id = $1
	`
	var t Ticket
	err := db.Pool.QueryRow(ctx, query, channelID).Scan(
		&t.ID, &t.GuildID, &t.UserID, &t.ChannelID, &t.Reason, &t.Status, &t.CreatedAt, &t.ClosedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil // Return nil if no ticket is found
		}
		return nil, err
	}
	return &t, nil
}

// CloseTicket marks a ticket as closed.
func (db *DB) CloseTicket(ctx context.Context, channelID string) error {
	start := time.Now()
	metrics.DBQueryLatency.WithLabelValues("CloseTicket").Observe(time.Since(start).Seconds())

	query := `
		UPDATE tickets
		SET status = 'closed', closed_at = NOW()
		WHERE channel_id = $1
	`
	_, err := db.Pool.Exec(ctx, query, channelID)
	return err
}

// UpsertUser inserts a new user or updates their info if they exist.
func (db *DB) UpsertUser(ctx context.Context, id, username, globalName, avatarURL string) error {
	query := `
		INSERT INTO users (id, username, global_name, avatar_url)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (id) DO UPDATE SET
			username = EXCLUDED.username,
			global_name = EXCLUDED.global_name,
			avatar_url = EXCLUDED.avatar_url,
			updated_at = NOW()
	`
	_, err := db.Pool.Exec(ctx, query, id, username, globalName, avatarURL)
	return err
}

// LogCommand records a command execution in the database.
// guildID is optional, pass an empty string if command was executed in DMs.
func (db *DB) LogCommand(ctx context.Context, commandName, userID, guildID string) error {
	var gID *string
	if guildID != "" {
		gID = &guildID
	}

	query := `
		INSERT INTO command_log (command_name, user_id, guild_id)
		VALUES ($1, $2, $3)
	`
	_, err := db.Pool.Exec(ctx, query, commandName, userID, gID)
	return err
}

// AddWarning inserts a new warning for a user in a guild.
func (db *DB) AddWarning(ctx context.Context, guildID, userID, moderatorID, reason string) error {
	query := `
		INSERT INTO warnings (guild_id, user_id, moderator_id, reason)
		VALUES ($1, $2, $3, $4)
	`
	_, err := db.Pool.Exec(ctx, query, guildID, userID, moderatorID, reason)
	return err
}

// Warning represents a single warning record.
type Warning struct {
	ID          int
	GuildID     string
	UserID      string
	ModeratorID string
	Reason      string
	CreatedAt   string // Can parse as time.Time if needed, but string is fine for formatting
}

// GetWarnings retrieves all warnings for a specific user in a guild.
func (db *DB) GetWarnings(ctx context.Context, guildID, userID string) ([]Warning, error) {
	query := `
		SELECT id, guild_id, user_id, moderator_id, reason, created_at
		FROM warnings
		WHERE guild_id = $1 AND user_id = $2
		ORDER BY created_at DESC
	`
	rows, err := db.Pool.Query(ctx, query, guildID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var warnings []Warning
	for rows.Next() {
		var w Warning
		var t time.Time
		if err := rows.Scan(&w.ID, &w.GuildID, &w.UserID, &w.ModeratorID, &w.Reason, &t); err != nil {
			return nil, err
		}
		w.CreatedAt = t.Format(time.RFC1123)
		warnings = append(warnings, w)
	}

	return warnings, rows.Err()
}

// LogModAction records a moderation action.
func (db *DB) LogModAction(ctx context.Context, guildID, userID, moderatorID, action, reason string) error {
	query := `
		INSERT INTO mod_actions (guild_id, user_id, moderator_id, action, reason)
		VALUES ($1, $2, $3, $4, $5)
	`
	_, err := db.Pool.Exec(ctx, query, guildID, userID, moderatorID, action, reason)
	return err
}

// GetStats returns the total number of guilds, users, and commands executed.
func (db *DB) GetStats(ctx context.Context) (guildCount, userCount, commandCount int64, err error) {
	start := time.Now()
	defer func() {
		metrics.DBQueryLatency.WithLabelValues("GetStats").Observe(time.Since(start).Seconds())
	}()

	err = db.Pool.QueryRow(ctx, "SELECT COUNT(*) FROM guilds").Scan(&guildCount)
	if err != nil {
		return
	}

	err = db.Pool.QueryRow(ctx, "SELECT COUNT(*) FROM users").Scan(&userCount)
	if err != nil {
		return
	}

	err = db.Pool.QueryRow(ctx, "SELECT COUNT(*) FROM command_log").Scan(&commandCount)
	return
}

// Guild represents a single guild record.
type Guild struct {
	ID       string `json:"id"`
	JoinedAt string `json:"joined_at"`
}

// GetGuilds returns a list of all guilds the bot is in.
func (db *DB) GetGuilds(ctx context.Context) ([]Guild, error) {
	start := time.Now()
	defer func() {
		metrics.DBQueryLatency.WithLabelValues("GetGuilds").Observe(time.Since(start).Seconds())
	}()

	query := `
		SELECT id, joined_at
		FROM guilds
		ORDER BY joined_at DESC
	`
	rows, err := db.Pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var guilds []Guild
	for rows.Next() {
		var g Guild
		var t time.Time
		if err := rows.Scan(&g.ID, &t); err != nil {
			return nil, err
		}
		g.JoinedAt = t.Format(time.RFC3339)
		guilds = append(guilds, g)
	}

	return guilds, rows.Err()
}

// UserEconomy represents a user's economy state in a guild.
type UserEconomy struct {
	GuildID     string
	UserID      string
	XP          int64
	Level       int
	Coins       int64
	LastDailyAt *time.Time
}

// GetUserEconomy retrieves the economy record for a user in a guild.
func (db *DB) GetUserEconomy(ctx context.Context, guildID, userID string) (*UserEconomy, error) {
	query := `
		SELECT guild_id, user_id, xp, level, coins, last_daily_at
		FROM user_economy
		WHERE guild_id = $1 AND user_id = $2
	`
	row := db.Pool.QueryRow(ctx, query, guildID, userID)
	var e UserEconomy
	err := row.Scan(&e.GuildID, &e.UserID, &e.XP, &e.Level, &e.Coins, &e.LastDailyAt)
	if err != nil {
		return nil, err
	}
	return &e, nil
}

// AddXP adds XP to a user's economy record.
func (db *DB) AddXP(ctx context.Context, guildID, userID string, amount int) (newXP int64, err error) {
	query := `
		INSERT INTO user_economy (guild_id, user_id, xp, level)
		VALUES ($1, $2, $3, 0)
		ON CONFLICT (guild_id, user_id) DO UPDATE SET
			xp = user_economy.xp + EXCLUDED.xp
		RETURNING xp
	`
	err = db.Pool.QueryRow(ctx, query, guildID, userID, amount).Scan(&newXP)
	return newXP, err
}

// SetLevel updates a user's level in a guild.
func (db *DB) SetLevel(ctx context.Context, guildID, userID string, level int) error {
	query := `
		UPDATE user_economy
		SET level = $1
		WHERE guild_id = $2 AND user_id = $3
	`
	_, err := db.Pool.Exec(ctx, query, level, guildID, userID)
	return err
}

// GetRank returns a user's rank based on XP within a guild.
func (db *DB) GetRank(ctx context.Context, guildID, userID string) (int, error) {
	query := `
		SELECT rank FROM (
			SELECT user_id, RANK() OVER (ORDER BY xp DESC) as rank
			FROM user_economy
			WHERE guild_id = $1
		) ranked_users
		WHERE user_id = $2
	`
	var rank int
	err := db.Pool.QueryRow(ctx, query, guildID, userID).Scan(&rank)
	return rank, err
}

// GetTopUsersByXP retrieves the top 10 users by XP in a guild.
func (db *DB) GetTopUsersByXP(ctx context.Context, guildID string) ([]UserEconomy, error) {
	query := `
		SELECT guild_id, user_id, xp, level, coins, last_daily_at
		FROM user_economy
		WHERE guild_id = $1
		ORDER BY xp DESC
		LIMIT 10
	`
	rows, err := db.Pool.Query(ctx, query, guildID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var topUsers []UserEconomy
	for rows.Next() {
		var e UserEconomy
		if err := rows.Scan(&e.GuildID, &e.UserID, &e.XP, &e.Level, &e.Coins, &e.LastDailyAt); err != nil {
			return nil, err
		}
		topUsers = append(topUsers, e)
	}

	return topUsers, rows.Err()
}

// ClaimDaily awards a daily coin amount to a user and updates their last_daily_at timestamp.
func (db *DB) ClaimDaily(ctx context.Context, guildID, userID string, amount int) (newCoins int64, err error) {
	// Need to handle both upsert and update
	query := `
		INSERT INTO user_economy (guild_id, user_id, coins, last_daily_at)
		VALUES ($1, $2, $3, NOW())
		ON CONFLICT (guild_id, user_id) DO UPDATE SET
			coins = user_economy.coins + EXCLUDED.coins,
			last_daily_at = EXCLUDED.last_daily_at
		RETURNING coins
	`
	err = db.Pool.QueryRow(ctx, query, guildID, userID, amount).Scan(&newCoins)
	return newCoins, err
}

// UserWithEconomy represents a user joined with their aggregated economy data.
type UserWithEconomy struct {
	ID         string  `json:"id"`
	Username   string  `json:"username"`
	GlobalName *string `json:"global_name"`
	AvatarURL  *string `json:"avatar_url"`
	TotalXP    int64   `json:"total_xp"`
	MaxLevel   int     `json:"max_level"`
}

// GetUsersWithEconomy returns all users with their aggregated XP and level.
func (db *DB) GetUsersWithEconomy(ctx context.Context) ([]UserWithEconomy, error) {
	start := time.Now()
	defer func() {
		metrics.DBQueryLatency.WithLabelValues("GetUsersWithEconomy").Observe(time.Since(start).Seconds())
	}()

	query := `
		SELECT
			u.id, u.username, u.global_name, u.avatar_url,
			COALESCE(SUM(e.xp), 0) as total_xp,
			COALESCE(MAX(e.level), 0) as max_level
		FROM users u
		LEFT JOIN user_economy e ON u.id = e.user_id
		GROUP BY u.id, u.username, u.global_name, u.avatar_url
		ORDER BY total_xp DESC
	`
	rows, err := db.Pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []UserWithEconomy
	for rows.Next() {
		var u UserWithEconomy
		if err := rows.Scan(&u.ID, &u.Username, &u.GlobalName, &u.AvatarURL, &u.TotalXP, &u.MaxLevel); err != nil {
			return nil, err
		}
		users = append(users, u)
	}

	return users, rows.Err()
}

// ModActionJoined represents a moderation action joined with user and moderator details.
type ModActionJoined struct {
	ID               int     `json:"id"`
	GuildID          string  `json:"guild_id"`
	Action           string  `json:"action"`
	Reason           string  `json:"reason"`
	CreatedAt        string  `json:"created_at"`
	TargetID         string  `json:"target_id"`
	TargetUsername   string  `json:"target_username"`
	TargetGlobalName *string `json:"target_global_name"`
	TargetAvatarURL  *string `json:"target_avatar_url"`
	ModID            string  `json:"mod_id"`
	ModUsername      string  `json:"mod_username"`
	ModGlobalName    *string `json:"mod_global_name"`
	ModAvatarURL     *string `json:"mod_avatar_url"`
}

// GetGuildLogChannel retrieves the configured log channel ID for a guild.
func (db *DB) GetGuildLogChannel(ctx context.Context, guildID string) (string, error) {
	query := `SELECT log_channel_id FROM guild_config WHERE guild_id = $1`
	var logChannelID *string
	err := db.Pool.QueryRow(ctx, query, guildID).Scan(&logChannelID)
	if err != nil {
		return "", err
	}
	if logChannelID == nil {
		return "", nil
	}
	return *logChannelID, nil
}

// SetGuildLogChannel updates or inserts the log channel ID for a guild.
func (db *DB) SetGuildLogChannel(ctx context.Context, guildID, logChannelID string) error {
	query := `
		INSERT INTO guild_config (guild_id, log_channel_id)
		VALUES ($1, $2)
		ON CONFLICT (guild_id) DO UPDATE SET
			log_channel_id = EXCLUDED.log_channel_id,
			updated_at = NOW()
	`
	_, err := db.Pool.Exec(ctx, query, guildID, logChannelID)
	return err
}

// CommandUsage represents the usage count for a specific command.
type CommandUsage struct {
	CommandName string `json:"name"`
	Count       int64  `json:"count"`
}

// GetCommandUsageStats returns the top 10 most used commands.
func (db *DB) GetCommandUsageStats(ctx context.Context) ([]CommandUsage, error) {
	start := time.Now()
	defer func() {
		metrics.DBQueryLatency.WithLabelValues("GetCommandUsageStats").Observe(time.Since(start).Seconds())
	}()

	query := `
		SELECT command_name, COUNT(*) as count
		FROM command_log
		GROUP BY command_name
		ORDER BY count DESC
		LIMIT 10
	`
	rows, err := db.Pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stats []CommandUsage
	for rows.Next() {
		var u CommandUsage
		if err := rows.Scan(&u.CommandName, &u.Count); err != nil {
			return nil, err
		}
		stats = append(stats, u)
	}

	return stats, rows.Err()
}

// GetModActions returns a list of all moderation actions with user details.
func (db *DB) GetModActions(ctx context.Context) ([]ModActionJoined, error) {
	start := time.Now()
	defer func() {
		metrics.DBQueryLatency.WithLabelValues("GetModActions").Observe(time.Since(start).Seconds())
	}()

	query := `
		SELECT
			m.id, m.guild_id, m.action, m.reason, m.created_at,
			tu.id as target_id, tu.username as target_username, tu.global_name as target_global_name, tu.avatar_url as target_avatar_url,
			mu.id as mod_id, mu.username as mod_username, mu.global_name as mod_global_name, mu.avatar_url as mod_avatar_url
		FROM mod_actions m
		JOIN users tu ON m.user_id = tu.id
		JOIN users mu ON m.moderator_id = mu.id
		ORDER BY m.created_at DESC
	`
	rows, err := db.Pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var actions []ModActionJoined
	for rows.Next() {
		var a ModActionJoined
		var t time.Time
		if err := rows.Scan(
			&a.ID, &a.GuildID, &a.Action, &a.Reason, &t,
			&a.TargetID, &a.TargetUsername, &a.TargetGlobalName, &a.TargetAvatarURL,
			&a.ModID, &a.ModUsername, &a.ModGlobalName, &a.ModAvatarURL,
		); err != nil {
			return nil, err
		}
		a.CreatedAt = t.Format(time.RFC3339)
		actions = append(actions, a)
	}

	return actions, rows.Err()
}

// GuildConfig represents the configuration for a single guild.
type GuildConfig struct {
	GuildID          string  `json:"guild_id"`
	LogChannelID     *string `json:"log_channel_id"`
	WelcomeChannelID *string `json:"welcome_channel_id"`
	ModRoleID        *string `json:"mod_role_id"`
	AutoRoleID       *string `json:"auto_role_id"`
}

// GetGuildConfig retrieves the entire configuration for a guild.
func (db *DB) GetGuildConfig(ctx context.Context, guildID string) (*GuildConfig, error) {
	query := `
		SELECT guild_id, log_channel_id, welcome_channel_id, mod_role_id, auto_role_id
		FROM guild_config
		WHERE guild_id = $1
	`
	var config GuildConfig
	err := db.Pool.QueryRow(ctx, query, guildID).Scan(
		&config.GuildID,
		&config.LogChannelID,
		&config.WelcomeChannelID,
		&config.ModRoleID,
		&config.AutoRoleID,
	)
	if err != nil {
		// If no row is found, return an empty config with just the GuildID
		if errors.Is(err, pgx.ErrNoRows) {
			return &GuildConfig{GuildID: guildID}, nil
		}
		return nil, err
	}
	return &config, nil
}

// SetGuildWelcomeChannel updates or inserts the welcome channel ID for a guild.
func (db *DB) SetGuildWelcomeChannel(ctx context.Context, guildID, welcomeChannelID string) error {
	query := `
		INSERT INTO guild_config (guild_id, welcome_channel_id)
		VALUES ($1, $2)
		ON CONFLICT (guild_id) DO UPDATE SET
			welcome_channel_id = EXCLUDED.welcome_channel_id,
			updated_at = NOW()
	`
	_, err := db.Pool.Exec(ctx, query, guildID, welcomeChannelID)
	return err
}

// SetGuildModRole updates or inserts the mod role ID for a guild.
func (db *DB) SetGuildModRole(ctx context.Context, guildID, modRoleID string) error {
	query := `
		INSERT INTO guild_config (guild_id, mod_role_id)
		VALUES ($1, $2)
		ON CONFLICT (guild_id) DO UPDATE SET
			mod_role_id = EXCLUDED.mod_role_id,
			updated_at = NOW()
	`
	_, err := db.Pool.Exec(ctx, query, guildID, modRoleID)
	return err
}

// UpdateGuildConfig updates the entire configuration for a guild.
func (db *DB) UpdateGuildConfig(ctx context.Context, guildID string, config GuildConfig) error {
	query := `
		INSERT INTO guild_config (guild_id, log_channel_id, welcome_channel_id, mod_role_id, auto_role_id)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (guild_id) DO UPDATE SET
			log_channel_id = EXCLUDED.log_channel_id,
			welcome_channel_id = EXCLUDED.welcome_channel_id,
			mod_role_id = EXCLUDED.mod_role_id,
			auto_role_id = EXCLUDED.auto_role_id,
			updated_at = NOW()
	`
	_, err := db.Pool.Exec(ctx, query, guildID, config.LogChannelID, config.WelcomeChannelID, config.ModRoleID, config.AutoRoleID)
	return err
}

// SetGuildAutoRole updates or inserts the auto role ID for a guild.
func (db *DB) SetGuildAutoRole(ctx context.Context, guildID, autoRoleID string) error {
	query := `
		INSERT INTO guild_config (guild_id, auto_role_id)
		VALUES ($1, $2)
		ON CONFLICT (guild_id) DO UPDATE SET
			auto_role_id = EXCLUDED.auto_role_id,
			updated_at = NOW()
	`
	_, err := db.Pool.Exec(ctx, query, guildID, autoRoleID)
	return err
}

// ReactionRole represents a reaction role mapping.
type ReactionRole struct {
	MessageID string `json:"message_id"`
	Emoji     string `json:"emoji"`
	RoleID    string `json:"role_id"`
}

// AddReactionRole adds a new reaction role mapping.
func (db *DB) AddReactionRole(ctx context.Context, messageID, emoji, roleID string) error {
	query := `
		INSERT INTO reaction_roles (message_id, emoji, role_id)
		VALUES ($1, $2, $3)
		ON CONFLICT (message_id, emoji) DO UPDATE SET
			role_id = EXCLUDED.role_id
	`
	_, err := db.Pool.Exec(ctx, query, messageID, emoji, roleID)
	return err
}

// RemoveReactionRole removes a reaction role mapping.
func (db *DB) RemoveReactionRole(ctx context.Context, messageID, emoji string) error {
	query := `
		DELETE FROM reaction_roles
		WHERE message_id = $1 AND emoji = $2
	`
	_, err := db.Pool.Exec(ctx, query, messageID, emoji)
	return err
}

// GetReactionRole retrieves a reaction role mapping.
func (db *DB) GetReactionRole(ctx context.Context, messageID, emoji string) (*ReactionRole, error) {
	query := `
		SELECT message_id, emoji, role_id
		FROM reaction_roles
		WHERE message_id = $1 AND emoji = $2
	`
	var rr ReactionRole
	err := db.Pool.QueryRow(ctx, query, messageID, emoji).Scan(&rr.MessageID, &rr.Emoji, &rr.RoleID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil // Return nil if no mapping is found
		}
		return nil, err
	}
	return &rr, nil
}

// ScheduledAnnouncement represents a scheduled announcement.
type ScheduledAnnouncement struct {
	ID        int       `json:"id"`
	GuildID   string    `json:"guild_id"`
	ChannelID string    `json:"channel_id"`
	Message   string    `json:"message"`
	SendAt    time.Time `json:"send_at"`
	Sent      bool      `json:"sent"`
}

// CreateScheduledAnnouncement creates a new scheduled announcement.
func (db *DB) CreateScheduledAnnouncement(ctx context.Context, guildID, channelID, message string, sendAt time.Time) error {
	query := `
		INSERT INTO scheduled_announcements (guild_id, channel_id, message, send_at)
		VALUES ($1, $2, $3, $4)
	`
	_, err := db.Pool.Exec(ctx, query, guildID, channelID, message, sendAt)
	return err
}

// GetPendingAnnouncements retrieves all pending announcements that are ready to be sent.
func (db *DB) GetPendingAnnouncements(ctx context.Context) ([]ScheduledAnnouncement, error) {
	query := `
		SELECT id, guild_id, channel_id, message, send_at, sent
		FROM scheduled_announcements
		WHERE sent = false AND send_at <= NOW()
	`
	rows, err := db.Pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var announcements []ScheduledAnnouncement
	for rows.Next() {
		var a ScheduledAnnouncement
		if err := rows.Scan(&a.ID, &a.GuildID, &a.ChannelID, &a.Message, &a.SendAt, &a.Sent); err != nil {
			return nil, err
		}
		announcements = append(announcements, a)
	}
	return announcements, rows.Err()
}

// MarkAnnouncementSent marks a scheduled announcement as sent.
func (db *DB) MarkAnnouncementSent(ctx context.Context, id int) error {
	query := `
		UPDATE scheduled_announcements
		SET sent = true
		WHERE id = $1
	`
	_, err := db.Pool.Exec(ctx, query, id)
	return err
}

// Reminder represents a scheduled user reminder.
type Reminder struct {
	ID        int       `json:"id"`
	UserID    string    `json:"user_id"`
	ChannelID string    `json:"channel_id"`
	GuildID   *string   `json:"guild_id"`
	Message   string    `json:"message"`
	DueAt     time.Time `json:"due_at"`
	Delivered bool      `json:"delivered"`
	CreatedAt time.Time `json:"created_at"`
}

// AddReminder adds a new reminder to the database.
func (db *DB) AddReminder(ctx context.Context, userID, channelID string, guildID *string, message string, dueAt time.Time) error {
	start := time.Now()
	metrics.DBQueryLatency.WithLabelValues("AddReminder").Observe(time.Since(start).Seconds())

	query := `
		INSERT INTO reminders (user_id, channel_id, guild_id, message, due_at)
		VALUES ($1, $2, $3, $4, $5)
	`
	_, err := db.Pool.Exec(ctx, query, userID, channelID, guildID, message, dueAt)
	return err
}

// GetPendingReminders gets all undelivered reminders for a specific user.
func (db *DB) GetPendingReminders(ctx context.Context, userID string) ([]Reminder, error) {
	start := time.Now()
	metrics.DBQueryLatency.WithLabelValues("GetPendingReminders").Observe(time.Since(start).Seconds())

	query := `
		SELECT id, user_id, channel_id, guild_id, message, due_at, delivered, created_at
		FROM reminders
		WHERE user_id = $1 AND delivered = FALSE
		ORDER BY due_at ASC
		LIMIT 10
	`
	rows, err := db.Pool.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var reminders []Reminder
	for rows.Next() {
		var r Reminder
		err := rows.Scan(&r.ID, &r.UserID, &r.ChannelID, &r.GuildID, &r.Message, &r.DueAt, &r.Delivered, &r.CreatedAt)
		if err != nil {
			return nil, err
		}
		reminders = append(reminders, r)
	}
	return reminders, nil
}

// GetDueReminders gets all undelivered reminders that are past their due time.
func (db *DB) GetDueReminders(ctx context.Context) ([]Reminder, error) {
	start := time.Now()
	metrics.DBQueryLatency.WithLabelValues("GetDueReminders").Observe(time.Since(start).Seconds())

	query := `
		SELECT id, user_id, channel_id, guild_id, message, due_at, delivered, created_at
		FROM reminders
		WHERE due_at <= NOW() AND delivered = FALSE
	`
	rows, err := db.Pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var reminders []Reminder
	for rows.Next() {
		var r Reminder
		err := rows.Scan(&r.ID, &r.UserID, &r.ChannelID, &r.GuildID, &r.Message, &r.DueAt, &r.Delivered, &r.CreatedAt)
		if err != nil {
			return nil, err
		}
		reminders = append(reminders, r)
	}
	return reminders, nil
}

// DeleteReminder deletes a reminder by ID for a specific user.
func (db *DB) DeleteReminder(ctx context.Context, id int, userID string) (bool, error) {
	start := time.Now()
	metrics.DBQueryLatency.WithLabelValues("DeleteReminder").Observe(time.Since(start).Seconds())

	query := `
		DELETE FROM reminders
		WHERE id = $1 AND user_id = $2
	`
	cmdTag, err := db.Pool.Exec(ctx, query, id, userID)
	if err != nil {
		return false, err
	}
	return cmdTag.RowsAffected() > 0, nil
}

// MarkReminderDelivered marks a reminder as delivered.
func (db *DB) MarkReminderDelivered(ctx context.Context, id int) error {
	start := time.Now()
	metrics.DBQueryLatency.WithLabelValues("MarkReminderDelivered").Observe(time.Since(start).Seconds())

	query := `
		UPDATE reminders
		SET delivered = TRUE
		WHERE id = $1
	`
	_, err := db.Pool.Exec(ctx, query, id)
	return err
}

// Tag represents a custom text response in a guild.
type Tag struct {
	ID        int       `json:"id"`
	GuildID   string    `json:"guild_id"`
	Name      string    `json:"name"`
	Content   string    `json:"content"`
	AuthorID  string    `json:"author_id"`
	CreatedAt time.Time `json:"created_at"`
}

// CreateTag creates a new tag in the database.
func (db *DB) CreateTag(ctx context.Context, guildID, name, content, authorID string) error {
	start := time.Now()
	defer func() {
		metrics.DBQueryLatency.WithLabelValues("CreateTag").Observe(time.Since(start).Seconds())
	}()

	query := `
		INSERT INTO tags (guild_id, name, content, author_id)
		VALUES ($1, $2, $3, $4)
	`
	_, err := db.Pool.Exec(ctx, query, guildID, name, content, authorID)
	if err != nil {
		slog.Error("Failed to create tag", "error", err, "guild", guildID, "name", name)
		return err
	}
	return nil
}

// GetTag retrieves a tag by name in a specific guild.
func (db *DB) GetTag(ctx context.Context, guildID, name string) (*Tag, error) {
	start := time.Now()
	defer func() {
		metrics.DBQueryLatency.WithLabelValues("GetTag").Observe(time.Since(start).Seconds())
	}()

	query := `
		SELECT id, guild_id, name, content, author_id, created_at
		FROM tags
		WHERE guild_id = $1 AND name = $2
	`
	tag := &Tag{}
	err := db.Pool.QueryRow(ctx, query, guildID, name).Scan(
		&tag.ID, &tag.GuildID, &tag.Name, &tag.Content, &tag.AuthorID, &tag.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil // Return nil, nil if tag is not found
		}
		slog.Error("Failed to get tag", "error", err, "guild", guildID, "name", name)
		return nil, err
	}
	return tag, nil
}

// DeleteTag deletes a tag by name in a specific guild.
func (db *DB) DeleteTag(ctx context.Context, guildID, name string) error {
	start := time.Now()
	defer func() {
		metrics.DBQueryLatency.WithLabelValues("DeleteTag").Observe(time.Since(start).Seconds())
	}()

	query := `
		DELETE FROM tags
		WHERE guild_id = $1 AND name = $2
	`
	tag, err := db.Pool.Exec(ctx, query, guildID, name)
	if err != nil {
		slog.Error("Failed to delete tag", "error", err, "guild", guildID, "name", name)
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

// ListTags lists all tags in a specific guild.
func (db *DB) ListTags(ctx context.Context, guildID string) ([]*Tag, error) {
	start := time.Now()
	defer func() {
		metrics.DBQueryLatency.WithLabelValues("ListTags").Observe(time.Since(start).Seconds())
	}()

	query := `
		SELECT id, guild_id, name, content, author_id, created_at
		FROM tags
		WHERE guild_id = $1
		ORDER BY name ASC
	`
	rows, err := db.Pool.Query(ctx, query, guildID)
	if err != nil {
		slog.Error("Failed to list tags", "error", err, "guild", guildID)
		return nil, err
	}
	defer rows.Close()

	var tags []*Tag
	for rows.Next() {
		tag := &Tag{}
		if err := rows.Scan(&tag.ID, &tag.GuildID, &tag.Name, &tag.Content, &tag.AuthorID, &tag.CreatedAt); err != nil {
			slog.Error("Failed to scan tag", "error", err)
			return nil, err
		}
		tags = append(tags, tag)
	}

	if err := rows.Err(); err != nil {
		slog.Error("Error iterating tags", "error", err)
		return nil, err
	}

	return tags, nil
}

// AutoResponder represents an auto-responder configuration in a guild.
type AutoResponder struct {
	ID          int       `json:"id"`
	GuildID     string    `json:"guild_id"`
	TriggerWord string    `json:"trigger_word"`
	Response    string    `json:"response"`
	CreatedAt   time.Time `json:"created_at"`
}

// AddAutoResponder adds a new auto-responder or updates an existing one for the same trigger.
func (db *DB) AddAutoResponder(ctx context.Context, guildID, triggerWord, response string) error {
	start := time.Now()
	defer func() {
		metrics.DBQueryLatency.WithLabelValues("AddAutoResponder").Observe(time.Since(start).Seconds())
	}()

	query := `
		INSERT INTO auto_responders (guild_id, trigger_word, response)
		VALUES ($1, $2, $3)
		ON CONFLICT (guild_id, trigger_word)
		DO UPDATE SET response = EXCLUDED.response
	`
	_, err := db.Pool.Exec(ctx, query, guildID, triggerWord, response)
	if err != nil {
		return fmt.Errorf("failed to add auto-responder: %w", err)
	}

	return nil
}

// RemoveAutoResponder removes an auto-responder by trigger word in a guild.
func (db *DB) RemoveAutoResponder(ctx context.Context, guildID, triggerWord string) error {
	start := time.Now()
	defer func() {
		metrics.DBQueryLatency.WithLabelValues("RemoveAutoResponder").Observe(time.Since(start).Seconds())
	}()

	query := `DELETE FROM auto_responders WHERE guild_id = $1 AND trigger_word = $2`
	_, err := db.Pool.Exec(ctx, query, guildID, triggerWord)
	if err != nil {
		return fmt.Errorf("failed to remove auto-responder: %w", err)
	}

	return nil
}

// ListAllAutoResponders returns all auto-responders across all guilds.
func (db *DB) ListAllAutoResponders(ctx context.Context) ([]*AutoResponder, error) {
	start := time.Now()
	defer func() {
		metrics.DBQueryLatency.WithLabelValues("ListAllAutoResponders").Observe(time.Since(start).Seconds())
	}()

	query := `
		SELECT id, guild_id, trigger_word, response, created_at
		FROM auto_responders
		ORDER BY id DESC
	`
	rows, err := db.Pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list all auto-responders: %w", err)
	}
	defer rows.Close()

	var responders []*AutoResponder
	for rows.Next() {
		r := &AutoResponder{}
		err := rows.Scan(&r.ID, &r.GuildID, &r.TriggerWord, &r.Response, &r.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan auto-responder: %w", err)
		}
		responders = append(responders, r)
	}

	return responders, nil
}

// ListAutoResponders returns all auto-responders for a specific guild.
func (db *DB) ListAutoResponders(ctx context.Context, guildID string) ([]*AutoResponder, error) {
	start := time.Now()
	defer func() {
		metrics.DBQueryLatency.WithLabelValues("ListAutoResponders").Observe(time.Since(start).Seconds())
	}()

	query := `
		SELECT id, guild_id, trigger_word, response, created_at
		FROM auto_responders
		WHERE guild_id = $1
		ORDER BY id DESC
	`
	rows, err := db.Pool.Query(ctx, query, guildID)
	if err != nil {
		return nil, fmt.Errorf("failed to list auto-responders: %w", err)
	}
	defer rows.Close()

	var responders []*AutoResponder
	for rows.Next() {
		r := &AutoResponder{}
		err := rows.Scan(&r.ID, &r.GuildID, &r.TriggerWord, &r.Response, &r.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan auto-responder: %w", err)
		}
		responders = append(responders, r)
	}

	return responders, nil
}

// StarboardConfig represents the starboard configuration for a guild.
type StarboardConfig struct {
	GuildID   string    `json:"guild_id"`
	ChannelID string    `json:"channel_id"`
	MinStars  int       `json:"min_stars"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// StarboardMessage represents a message that has been posted to the starboard.
type StarboardMessage struct {
	MessageID          string    `json:"message_id"`
	GuildID            string    `json:"guild_id"`
	ChannelID          string    `json:"channel_id"`
	StarboardMessageID string    `json:"starboard_message_id"`
	Stars              int       `json:"stars"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

// SetStarboardConfig sets the starboard configuration for a guild.
func (db *DB) SetStarboardConfig(ctx context.Context, guildID, channelID string, minStars int) error {
	start := time.Now()
	defer func() {
		metrics.DBQueryLatency.WithLabelValues("SetStarboardConfig").Observe(time.Since(start).Seconds())
	}()

	query := `
		INSERT INTO starboard_config (guild_id, channel_id, min_stars, updated_at)
		VALUES ($1, $2, $3, NOW())
		ON CONFLICT (guild_id)
		DO UPDATE SET channel_id = EXCLUDED.channel_id, min_stars = EXCLUDED.min_stars, updated_at = NOW()
	`
	_, err := db.Pool.Exec(ctx, query, guildID, channelID, minStars)
	if err != nil {
		slog.Error("Failed to set starboard config", "error", err, "guild_id", guildID)
		return err
	}
	return nil
}

// GetStarboardConfig retrieves the starboard configuration for a guild.
func (db *DB) GetStarboardConfig(ctx context.Context, guildID string) (*StarboardConfig, error) {
	start := time.Now()
	defer func() {
		metrics.DBQueryLatency.WithLabelValues("GetStarboardConfig").Observe(time.Since(start).Seconds())
	}()

	query := `
		SELECT guild_id, channel_id, min_stars, created_at, updated_at
		FROM starboard_config
		WHERE guild_id = $1
	`
	config := &StarboardConfig{}
	err := db.Pool.QueryRow(ctx, query, guildID).Scan(
		&config.GuildID, &config.ChannelID, &config.MinStars, &config.CreatedAt, &config.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil // Return nil, nil if config is not found
		}
		slog.Error("Failed to get starboard config", "error", err, "guild_id", guildID)
		return nil, err
	}
	return config, nil
}

// GetStarboardMessage retrieves a starboard message by its original message ID.
func (db *DB) GetStarboardMessage(ctx context.Context, messageID string) (*StarboardMessage, error) {
	start := time.Now()
	defer func() {
		metrics.DBQueryLatency.WithLabelValues("GetStarboardMessage").Observe(time.Since(start).Seconds())
	}()

	query := `
		SELECT message_id, guild_id, channel_id, starboard_message_id, stars, created_at, updated_at
		FROM starboard_messages
		WHERE message_id = $1
	`
	msg := &StarboardMessage{}
	err := db.Pool.QueryRow(ctx, query, messageID).Scan(
		&msg.MessageID, &msg.GuildID, &msg.ChannelID, &msg.StarboardMessageID, &msg.Stars, &msg.CreatedAt, &msg.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil // Return nil, nil if message is not found
		}
		slog.Error("Failed to get starboard message", "error", err, "message_id", messageID)
		return nil, err
	}
	return msg, nil
}

// UpsertStarboardMessage upserts a starboard message record.
func (db *DB) UpsertStarboardMessage(ctx context.Context, messageID, guildID, channelID, starboardMessageID string, stars int) error {
	start := time.Now()
	defer func() {
		metrics.DBQueryLatency.WithLabelValues("UpsertStarboardMessage").Observe(time.Since(start).Seconds())
	}()

	query := `
		INSERT INTO starboard_messages (message_id, guild_id, channel_id, starboard_message_id, stars, updated_at)
		VALUES ($1, $2, $3, $4, $5, NOW())
		ON CONFLICT (message_id)
		DO UPDATE SET starboard_message_id = EXCLUDED.starboard_message_id, stars = EXCLUDED.stars, updated_at = NOW()
	`
	_, err := db.Pool.Exec(ctx, query, messageID, guildID, channelID, starboardMessageID, stars)
	if err != nil {
		slog.Error("Failed to upsert starboard message", "error", err, "message_id", messageID)
		return err
	}
	return nil
}

// Giveaway represents a giveaway record in the database.
type Giveaway struct {
	ID          int       `json:"id"`
	GuildID     string    `json:"guild_id"`
	ChannelID   string    `json:"channel_id"`
	MessageID   string    `json:"message_id"`
	Prize       string    `json:"prize"`
	WinnerCount int       `json:"winner_count"`
	EndAt       time.Time `json:"end_at"`
	Ended       bool      `json:"ended"`
	CreatedAt   time.Time `json:"created_at"`
}

// CreateGiveaway inserts a new giveaway into the database.
func (db *DB) CreateGiveaway(ctx context.Context, guildID, channelID, messageID, prize string, winnerCount int, endAt time.Time) error {
	start := time.Now()
	defer func() {
		metrics.DBQueryLatency.WithLabelValues("CreateGiveaway").Observe(time.Since(start).Seconds())
	}()

	query := `
		INSERT INTO giveaways (guild_id, channel_id, message_id, prize, winner_count, end_at, ended, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, false, NOW())
	`
	_, err := db.Pool.Exec(ctx, query, guildID, channelID, messageID, prize, winnerCount, endAt)
	if err != nil {
		slog.Error("Failed to create giveaway", "error", err, "guild_id", guildID, "message_id", messageID)
		return err
	}
	return nil
}

// GetActiveGiveaways retrieves all giveaways that have ended=false and their end_at time is in the past.
func (db *DB) GetActiveGiveaways(ctx context.Context) ([]*Giveaway, error) {
	start := time.Now()
	defer func() {
		metrics.DBQueryLatency.WithLabelValues("GetActiveGiveaways").Observe(time.Since(start).Seconds())
	}()

	query := `
		SELECT id, guild_id, channel_id, message_id, prize, winner_count, end_at, ended, created_at
		FROM giveaways
		WHERE ended = false AND end_at <= NOW()
	`
	rows, err := db.Pool.Query(ctx, query)
	if err != nil {
		slog.Error("Failed to get active giveaways", "error", err)
		return nil, err
	}
	defer rows.Close()

	var giveaways []*Giveaway
	for rows.Next() {
		g := &Giveaway{}
		err := rows.Scan(
			&g.ID, &g.GuildID, &g.ChannelID, &g.MessageID, &g.Prize, &g.WinnerCount, &g.EndAt, &g.Ended, &g.CreatedAt,
		)
		if err != nil {
			slog.Error("Failed to scan giveaway", "error", err)
			return nil, err
		}
		giveaways = append(giveaways, g)
	}
	return giveaways, nil
}

// EndGiveaway marks a giveaway as ended.
func (db *DB) EndGiveaway(ctx context.Context, messageID string) error {
	start := time.Now()
	defer func() {
		metrics.DBQueryLatency.WithLabelValues("EndGiveaway").Observe(time.Since(start).Seconds())
	}()

	query := `
		UPDATE giveaways
		SET ended = true
		WHERE message_id = $1
	`
	_, err := db.Pool.Exec(ctx, query, messageID)
	if err != nil {
		slog.Error("Failed to end giveaway", "error", err, "message_id", messageID)
		return err
	}
	return nil
}

// GetGiveawayByMessage retrieves a giveaway by its message ID.
func (db *DB) GetGiveawayByMessage(ctx context.Context, messageID string) (*Giveaway, error) {
	start := time.Now()
	defer func() {
		metrics.DBQueryLatency.WithLabelValues("GetGiveawayByMessage").Observe(time.Since(start).Seconds())
	}()

	query := `
		SELECT id, guild_id, channel_id, message_id, prize, winner_count, end_at, ended, created_at
		FROM giveaways
		WHERE message_id = $1
	`
	g := &Giveaway{}
	err := db.Pool.QueryRow(ctx, query, messageID).Scan(
		&g.ID, &g.GuildID, &g.ChannelID, &g.MessageID, &g.Prize, &g.WinnerCount, &g.EndAt, &g.Ended, &g.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil // Return nil, nil if giveaway is not found
		}
		slog.Error("Failed to get giveaway by message", "error", err, "message_id", messageID)
		return nil, err
	}
	return g, nil
}

// AddGiveawayEntrant adds a user to a giveaway.
func (db *DB) AddGiveawayEntrant(ctx context.Context, giveawayID int, userID string) error {
	start := time.Now()
	defer func() {
		metrics.DBQueryLatency.WithLabelValues("AddGiveawayEntrant").Observe(time.Since(start).Seconds())
	}()

	query := `
		INSERT INTO giveaway_entrants (giveaway_id, user_id, created_at)
		VALUES ($1, $2, NOW())
		ON CONFLICT ON CONSTRAINT giveaway_entrants_unique DO NOTHING
	`
	_, err := db.Pool.Exec(ctx, query, giveawayID, userID)
	if err != nil {
		slog.Error("Failed to add giveaway entrant", "error", err, "giveaway_id", giveawayID, "user_id", userID)
		return err
	}
	return nil
}

// GetGiveawayEntrants retrieves all entrants for a giveaway.
func (db *DB) GetGiveawayEntrants(ctx context.Context, giveawayID int) ([]string, error) {
	start := time.Now()
	defer func() {
		metrics.DBQueryLatency.WithLabelValues("GetGiveawayEntrants").Observe(time.Since(start).Seconds())
	}()

	query := `
		SELECT user_id
		FROM giveaway_entrants
		WHERE giveaway_id = $1
	`
	rows, err := db.Pool.Query(ctx, query, giveawayID)
	if err != nil {
		slog.Error("Failed to get giveaway entrants", "error", err, "giveaway_id", giveawayID)
		return nil, err
	}
	defer rows.Close()

	var entrants []string
	for rows.Next() {
		var userID string
		err := rows.Scan(&userID)
		if err != nil {
			slog.Error("Failed to scan giveaway entrant", "error", err)
			return nil, err
		}
		entrants = append(entrants, userID)
	}
	return entrants, nil
}

// SetAFK sets a user as AFK in a guild.
func (db *DB) SetAFK(ctx context.Context, userID, guildID, reason string) error {
	query := `
		INSERT INTO afk_users (user_id, guild_id, reason, created_at)
		VALUES ($1, $2, $3, CURRENT_TIMESTAMP)
		ON CONFLICT (user_id, guild_id) DO UPDATE
		SET reason = EXCLUDED.reason, created_at = CURRENT_TIMESTAMP
	`
	_, err := db.Pool.Exec(ctx, query, userID, guildID, reason)
	if err != nil {
		slog.Error("Failed to set AFK status", "error", err, "user_id", userID, "guild_id", guildID)
		return fmt.Errorf("failed to set AFK status: %w", err)
	}
	return nil
}

// RemoveAFK removes a user's AFK status in a guild.
func (db *DB) RemoveAFK(ctx context.Context, userID, guildID string) error {
	query := `DELETE FROM afk_users WHERE user_id = $1 AND guild_id = $2`
	_, err := db.Pool.Exec(ctx, query, userID, guildID)
	if err != nil {
		slog.Error("Failed to remove AFK status", "error", err, "user_id", userID, "guild_id", guildID)
		return fmt.Errorf("failed to remove AFK status: %w", err)
	}
	return nil
}

// GetAFK gets a user's AFK status in a guild.
func (db *DB) GetAFK(ctx context.Context, userID, guildID string) (string, time.Time, error) {
	var reason string
	var createdAt time.Time
	query := `SELECT reason, created_at FROM afk_users WHERE user_id = $1 AND guild_id = $2`
	err := db.Pool.QueryRow(ctx, query, userID, guildID).Scan(&reason, &createdAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", time.Time{}, nil
		}
		slog.Error("Failed to get AFK status", "error", err, "user_id", userID, "guild_id", guildID)
		return "", time.Time{}, fmt.Errorf("failed to get AFK status: %w", err)
	}
	return reason, createdAt, nil
}

// Poll represents a poll in a guild.
type Poll struct {
	ID        int
	GuildID   string
	ChannelID string
	MessageID string
	CreatorID string
	Question  string
	Options   []string
	IsClosed  bool
	CreatedAt time.Time
}

// CreatePoll inserts a new poll into the database.
func (db *DB) CreatePoll(ctx context.Context, p *Poll) error {
	start := time.Now()
	metrics.DBQueryLatency.WithLabelValues("CreatePoll").Observe(time.Since(start).Seconds())

	query := `
		INSERT INTO polls (guild_id, channel_id, message_id, creator_id, question, options)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	_, err := db.Pool.Exec(ctx, query, p.GuildID, p.ChannelID, p.MessageID, p.CreatorID, p.Question, p.Options)
	if err != nil {
		slog.Error("Failed to create poll", "error", err, "guild_id", p.GuildID)
		return err
	}
	return nil
}

// GetPoll retrieves a poll by its message ID.
func (db *DB) GetPoll(ctx context.Context, messageID string) (*Poll, error) {
	start := time.Now()
	metrics.DBQueryLatency.WithLabelValues("GetPoll").Observe(time.Since(start).Seconds())

	query := `
		SELECT id, guild_id, channel_id, message_id, creator_id, question, options, is_closed, created_at
		FROM polls
		WHERE message_id = $1
	`
	var p Poll
	err := db.Pool.QueryRow(ctx, query, messageID).Scan(
		&p.ID, &p.GuildID, &p.ChannelID, &p.MessageID, &p.CreatorID, &p.Question, &p.Options, &p.IsClosed, &p.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil // Return nil, nil if poll is not found
		}
		slog.Error("Failed to get poll by message", "error", err, "message_id", messageID)
		return nil, err
	}
	return &p, nil
}

// ClosePoll marks a poll as closed.
func (db *DB) ClosePoll(ctx context.Context, messageID string) error {
	start := time.Now()
	metrics.DBQueryLatency.WithLabelValues("ClosePoll").Observe(time.Since(start).Seconds())

	query := `
		UPDATE polls
		SET is_closed = true
		WHERE message_id = $1
	`
	_, err := db.Pool.Exec(ctx, query, messageID)
	if err != nil {
		slog.Error("Failed to close poll", "error", err, "message_id", messageID)
		return err
	}
	return nil
}

// StickyMessage represents a sticky message in a channel.
type StickyMessage struct {
	ChannelID     string
	GuildID       string
	MessageText   string
	LastMessageID string
}

// SetSticky creates or updates a sticky message for a channel.
func (db *DB) SetSticky(ctx context.Context, channelID, guildID, messageText string) error {
	start := time.Now()
	metrics.DBQueryLatency.WithLabelValues("SetSticky").Observe(time.Since(start).Seconds())

	query := `
		INSERT INTO sticky_messages (channel_id, guild_id, message_text)
		VALUES ($1, $2, $3)
		ON CONFLICT (channel_id) DO UPDATE
		SET message_text = EXCLUDED.message_text, last_message_id = NULL, updated_at = CURRENT_TIMESTAMP
	`
	_, err := db.Pool.Exec(ctx, query, channelID, guildID, messageText)
	if err != nil {
		slog.Error("Failed to set sticky message", "error", err, "channel_id", channelID)
	}
	return err
}

// RemoveSticky removes a sticky message from a channel.
func (db *DB) RemoveSticky(ctx context.Context, channelID string) error {
	start := time.Now()
	metrics.DBQueryLatency.WithLabelValues("RemoveSticky").Observe(time.Since(start).Seconds())

	query := `DELETE FROM sticky_messages WHERE channel_id = $1`
	_, err := db.Pool.Exec(ctx, query, channelID)
	if err != nil {
		slog.Error("Failed to remove sticky message", "error", err, "channel_id", channelID)
	}
	return err
}

// GetSticky retrieves the sticky message configuration for a channel.
func (db *DB) GetSticky(ctx context.Context, channelID string) (*StickyMessage, error) {
	start := time.Now()
	metrics.DBQueryLatency.WithLabelValues("GetSticky").Observe(time.Since(start).Seconds())

	query := `SELECT channel_id, guild_id, message_text, last_message_id FROM sticky_messages WHERE channel_id = $1`
	var sticky StickyMessage
	var lastMessageID *string
	err := db.Pool.QueryRow(ctx, query, channelID).Scan(&sticky.ChannelID, &sticky.GuildID, &sticky.MessageText, &lastMessageID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil // Not an error, just no sticky message
		}
		slog.Error("Failed to get sticky message", "error", err, "channel_id", channelID)
		return nil, err
	}
	if lastMessageID != nil {
		sticky.LastMessageID = *lastMessageID
	}
	return &sticky, nil
}

// UpdateStickyMessageID updates the last message ID of a sticky message.
func (db *DB) UpdateStickyMessageID(ctx context.Context, channelID, messageID string) error {
	start := time.Now()
	metrics.DBQueryLatency.WithLabelValues("UpdateStickyMessageID").Observe(time.Since(start).Seconds())

	query := `UPDATE sticky_messages SET last_message_id = $1, updated_at = CURRENT_TIMESTAMP WHERE channel_id = $2`
	_, err := db.Pool.Exec(ctx, query, messageID, channelID)
	if err != nil {
		slog.Error("Failed to update sticky message ID", "error", err, "channel_id", channelID)
	}
	return err
}

// SuggestionConfig represents the configuration for suggestions in a guild
type SuggestionConfig struct {
	GuildID             string
	SuggestionChannelID string
}

// Suggestion represents a single user suggestion
type Suggestion struct {
	ID        int
	GuildID   string
	UserID    string
	MessageID string
	Content   string
	Status    string
}

// SetSuggestionChannel configures the channel where suggestions will be posted
func (db *DB) SetSuggestionChannel(ctx context.Context, guildID, channelID string) error {
	query := `
		INSERT INTO suggestion_config (guild_id, suggestion_channel_id)
		VALUES ($1, $2)
		ON CONFLICT (guild_id) DO UPDATE
		SET suggestion_channel_id = $2, updated_at = CURRENT_TIMESTAMP
	`
	_, err := db.Pool.Exec(ctx, query, guildID, channelID)
	return err
}

// GetSuggestionChannel gets the configured suggestion channel for a guild
func (db *DB) GetSuggestionChannel(ctx context.Context, guildID string) (string, error) {
	query := `SELECT suggestion_channel_id FROM suggestion_config WHERE guild_id = $1`
	var channelID string
	err := db.Pool.QueryRow(ctx, query, guildID).Scan(&channelID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", nil // No config set
		}
		return "", err
	}
	return channelID, nil
}

// VerificationConfig represents the verification configuration for a guild.
type VerificationConfig struct {
	GuildID string
	RoleID  string
}

// SetVerificationConfig sets the verification configuration for a guild.
func (db *DB) SetVerificationConfig(ctx context.Context, guildID, roleID string) error {
	query := `
		INSERT INTO verification_config (guild_id, role_id)
		VALUES ($1, $2)
		ON CONFLICT (guild_id) DO UPDATE
		SET role_id = EXCLUDED.role_id, updated_at = CURRENT_TIMESTAMP
	`
	_, err := db.Pool.Exec(ctx, query, guildID, roleID)
	return err
}

// GetVerificationConfig gets the verification configuration for a guild.
func (db *DB) GetVerificationConfig(ctx context.Context, guildID string) (*VerificationConfig, error) {
	query := `SELECT role_id FROM verification_config WHERE guild_id = $1`
	var roleID string
	err := db.Pool.QueryRow(ctx, query, guildID).Scan(&roleID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil // No config set
		}
		return nil, err
	}
	return &VerificationConfig{GuildID: guildID, RoleID: roleID}, nil
}

// AutomodConfig represents the automod configuration for a guild.
type AutomodConfig struct {
	GuildID       string
	LogChannelID  string
	FilterLinks   bool
	FilterInvites bool
}

// SetAutomodConfig upserts the automod configuration for a guild.
func (db *DB) SetAutomodConfig(ctx context.Context, guildID, logChannelID string, filterLinks, filterInvites bool) error {
	query := `
		INSERT INTO automod_config (guild_id, log_channel_id, filter_links, filter_invites)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (guild_id) DO UPDATE
		SET log_channel_id = $2, filter_links = $3, filter_invites = $4, updated_at = CURRENT_TIMESTAMP
	`
	_, err := db.Pool.Exec(ctx, query, guildID, logChannelID, filterLinks, filterInvites)
	return err
}

// GetAutomodConfig retrieves the automod configuration for a guild.
func (db *DB) GetAutomodConfig(ctx context.Context, guildID string) (*AutomodConfig, error) {
	query := `SELECT log_channel_id, filter_links, filter_invites FROM automod_config WHERE guild_id = $1`
	var config AutomodConfig
	config.GuildID = guildID

	err := db.Pool.QueryRow(ctx, query, guildID).Scan(&config.LogChannelID, &config.FilterLinks, &config.FilterInvites)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil // Not configured
		}
		return nil, err
	}

	return &config, nil
}

// AddAutomodWord adds a restricted word to the automod configuration for a guild.
func (db *DB) AddAutomodWord(ctx context.Context, guildID, word string) error {
	query := `
		INSERT INTO automod_words (guild_id, word)
		VALUES ($1, $2)
		ON CONFLICT (guild_id, word) DO NOTHING
	`
	_, err := db.Pool.Exec(ctx, query, guildID, word)
	return err
}

// RemoveAutomodWord removes a restricted word from the automod configuration for a guild.
func (db *DB) RemoveAutomodWord(ctx context.Context, guildID, word string) error {
	query := `DELETE FROM automod_words WHERE guild_id = $1 AND word = $2`
	_, err := db.Pool.Exec(ctx, query, guildID, word)
	return err
}

// GetAutomodWords retrieves all restricted words for a guild.
func (db *DB) GetAutomodWords(ctx context.Context, guildID string) ([]string, error) {
	query := `SELECT word FROM automod_words WHERE guild_id = $1`
	rows, err := db.Pool.Query(ctx, query, guildID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var words []string
	for rows.Next() {
		var word string
		if err := rows.Scan(&word); err != nil {
			return nil, err
		}
		words = append(words, word)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return words, nil
}

// CreateSuggestion creates a new suggestion and returns its ID
func (db *DB) CreateSuggestion(ctx context.Context, guildID, userID, messageID, content string) (int, error) {
	query := `
		INSERT INTO suggestions (guild_id, user_id, message_id, content, status)
		VALUES ($1, $2, $3, $4, 'pending')
		RETURNING id
	`
	var id int
	err := db.Pool.QueryRow(ctx, query, guildID, userID, messageID, content).Scan(&id)
	return id, err
}

// GetSuggestionByID gets a suggestion by its ID
func (db *DB) GetSuggestionByID(ctx context.Context, suggestionID int) (*Suggestion, error) {
	query := `
		SELECT id, guild_id, user_id, message_id, content, status
		FROM suggestions
		WHERE id = $1
	`
	var s Suggestion
	err := db.Pool.QueryRow(ctx, query, suggestionID).Scan(&s.ID, &s.GuildID, &s.UserID, &s.MessageID, &s.Content, &s.Status)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil // Not found
		}
		return nil, err
	}
	return &s, nil
}

// UpdateSuggestionStatus updates the status of a suggestion
func (db *DB) UpdateSuggestionStatus(ctx context.Context, suggestionID int, status string) error {
	query := `
		UPDATE suggestions
		SET status = $1, updated_at = CURRENT_TIMESTAMP
		WHERE id = $2
	`
	_, err := db.Pool.Exec(ctx, query, status, suggestionID)
	return err
}

// SetServerLogChannel sets the server log channel for a guild
func (db *DB) SetServerLogChannel(ctx context.Context, guildID, channelID string) error {
	query := `
		INSERT INTO server_log_config (guild_id, channel_id)
		VALUES ($1, $2)
		ON CONFLICT (guild_id) DO UPDATE
		SET channel_id = $2, updated_at = CURRENT_TIMESTAMP
	`
	_, err := db.Pool.Exec(ctx, query, guildID, channelID)
	return err
}

// GetServerLogChannel gets the configured server log channel for a guild
func (db *DB) GetServerLogChannel(ctx context.Context, guildID string) (string, error) {
	query := `SELECT channel_id FROM server_log_config WHERE guild_id = $1`
	var channelID string
	err := db.Pool.QueryRow(ctx, query, guildID).Scan(&channelID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", nil // No config set
		}
		return "", err
	}
	return channelID, nil
}
