package db

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

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

	log.Println("Connected to PostgreSQL database successfully.")

	return &DB{Pool: pool}, nil
}

// Close gracefully closes the database connection pool.
func (db *DB) Close() {
	if db.Pool != nil {
		db.Pool.Close()
		log.Println("Database connection closed gracefully.")
	}
}

// RunMigrations executes database migrations from the migrations folder.
func RunMigrations(databaseURL string) error {
	log.Println("Running database migrations...")

	m, err := migrate.New("file://migrations", databaseURL)
	if err != nil {
		return fmt.Errorf("could not create migrate instance: %w", err)
	}
	defer m.Close()

	err = m.Up()
	if err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			log.Println("Database is already up to date.")
			return nil
		}
		return fmt.Errorf("could not run up migrations: %w", err)
	}

	log.Println("Database migrations applied successfully.")
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
