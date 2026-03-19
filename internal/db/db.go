package db

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math/rand"
	"regexp"
	"strings"
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

type TicketTranscript struct {
	ID            int       `json:"id"`
	TicketID      int       `json:"ticket_id"`
	ChannelID     string    `json:"channel_id"`
	GuildID       string    `json:"guild_id"`
	UserID        string    `json:"user_id"`
	TranscriptURL string    `json:"transcript_url"`
	CreatedAt     time.Time `json:"created_at"`
}

type Ticket struct {
	ID        int        `json:"id"`
	GuildID   string     `json:"guild_id"`
	UserID    string     `json:"user_id"`
	ChannelID string     `json:"channel_id"`
	Reason    string     `json:"reason"`
	Status    string     `json:"status"`
	CreatedAt time.Time  `json:"created_at"`
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

// Phase 30: Ticket Panels

type TicketPanel struct {
	GuildID   string
	ChannelID string
	MessageID string
}

func (db *DB) SetTicketPanel(ctx context.Context, guildID, channelID, messageID string) error {
	query := `
		INSERT INTO ticket_panels (guild_id, channel_id, message_id)
		VALUES ($1, $2, $3)
		ON CONFLICT (guild_id) DO UPDATE SET channel_id = EXCLUDED.channel_id, message_id = EXCLUDED.message_id, created_at = CURRENT_TIMESTAMP
	`
	_, err := db.Pool.Exec(ctx, query, guildID, channelID, messageID)
	return err
}

func (db *DB) GetTicketPanel(ctx context.Context, guildID string) (*TicketPanel, error) {
	query := `SELECT guild_id, channel_id, message_id FROM ticket_panels WHERE guild_id = $1`
	var panel TicketPanel
	err := db.Pool.QueryRow(ctx, query, guildID).Scan(&panel.GuildID, &panel.ChannelID, &panel.MessageID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &panel, nil
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

// RemoveWarning removes a specific warning by its ID.
func (db *DB) RemoveWarning(ctx context.Context, warningID int) error {
	query := `DELETE FROM warnings WHERE id = $1`
	_, err := db.Pool.Exec(ctx, query, warningID)
	return err
}

// ClearWarnings removes all warnings for a specific user in a guild.
func (db *DB) ClearWarnings(ctx context.Context, guildID, userID string) error {
	query := `DELETE FROM warnings WHERE guild_id = $1 AND user_id = $2`
	_, err := db.Pool.Exec(ctx, query, guildID, userID)
	return err
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
	return db.LogModActionComplex(ctx, guildID, userID, moderatorID, action, reason, nil, nil)
}

// LogModActionComplex records a moderation action with duration and evidence.
func (db *DB) LogModActionComplex(ctx context.Context, guildID, userID, moderatorID, action, reason string, duration *string, evidenceURL *string) error {
	query := `
		INSERT INTO mod_actions (guild_id, user_id, moderator_id, action, reason, duration, evidence_url)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	_, err := db.Pool.Exec(ctx, query, guildID, userID, moderatorID, action, reason, duration, evidenceURL)
	return err
}

// MarkModActionResolved sets resolved = true on a specific moderation action ID.
func (db *DB) MarkModActionResolved(ctx context.Context, actionID int) error {
	query := `
		UPDATE mod_actions
		SET resolved = true
		WHERE id = $1
	`
	_, err := db.Pool.Exec(ctx, query, actionID)
	return err
}

// GetActiveBans returns active ban actions.
func (db *DB) GetActiveBans(ctx context.Context, guildID string) ([]ModActionJoined, error) {
	// Bans don't automatically expire in discord via the bot (unless a duration feature is added to ban later),
	// but we'll fetch unresolved bans.
	query := `
		SELECT
			m.id, m.guild_id, m.action, m.reason, m.created_at, m.duration, m.resolved, m.evidence_url,
			tu.id as target_id, tu.username as target_username, tu.global_name as target_global_name, tu.avatar_url as target_avatar_url,
			mu.id as mod_id, mu.username as mod_username, mu.global_name as mod_global_name, mu.avatar_url as mod_avatar_url
		FROM mod_actions m
		JOIN users tu ON m.user_id = tu.id
		JOIN users mu ON m.moderator_id = mu.id
		WHERE m.guild_id = $1 AND m.action = 'ban' AND m.resolved = false
		ORDER BY m.created_at DESC
	`
	rows, err := db.Pool.Query(ctx, query, guildID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var actions []ModActionJoined
	for rows.Next() {
		var a ModActionJoined
		var t time.Time
		if err := rows.Scan(
			&a.ID, &a.GuildID, &a.Action, &a.Reason, &t, &a.Duration, &a.Resolved, &a.EvidenceURL,
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

// GetActiveMutes returns active mute actions.
func (db *DB) GetActiveMutes(ctx context.Context, guildID string) ([]ModActionJoined, error) {
	query := `
		SELECT
			m.id, m.guild_id, m.action, m.reason, m.created_at, m.duration, m.resolved, m.evidence_url,
			tu.id as target_id, tu.username as target_username, tu.global_name as target_global_name, tu.avatar_url as target_avatar_url,
			mu.id as mod_id, mu.username as mod_username, mu.global_name as mod_global_name, mu.avatar_url as mod_avatar_url
		FROM mod_actions m
		JOIN users tu ON m.user_id = tu.id
		JOIN users mu ON m.moderator_id = mu.id
		WHERE m.guild_id = $1 AND m.action = 'Mute' AND m.resolved = false
		ORDER BY m.created_at DESC
	`
	rows, err := db.Pool.Query(ctx, query, guildID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var actions []ModActionJoined
	for rows.Next() {
		var a ModActionJoined
		var t time.Time
		if err := rows.Scan(
			&a.ID, &a.GuildID, &a.Action, &a.Reason, &t, &a.Duration, &a.Resolved, &a.EvidenceURL,
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

// GetGuildPrefix returns the custom prefix for a guild, falling back to '!' if not set or found.
func (db *DB) GetGuildPrefix(ctx context.Context, guildID string) (string, error) {
	query := `SELECT prefix FROM guilds WHERE id = $1`
	var prefix string
	err := db.Pool.QueryRow(ctx, query, guildID).Scan(&prefix)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "!", nil
		}
		return "!", err
	}
	if prefix == "" {
		return "!", nil
	}
	return prefix, nil
}

// SetGuildPrefix sets the custom prefix for a guild.
func (db *DB) SetGuildPrefix(ctx context.Context, guildID, prefix string) error {
	query := `
		INSERT INTO guilds (id, prefix)
		VALUES ($1, $2)
		ON CONFLICT (id)
		DO UPDATE SET prefix = EXCLUDED.prefix
	`
	_, err := db.Pool.Exec(ctx, query, guildID, prefix)
	return err
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

// UserReputation represents a user's reputation score.
type UserReputation struct {
	GuildID string
	UserID  string
	Rep     int64
}

// UserEconomy represents a user's economy state in a guild.
type UserEconomy struct {
	GuildID        string
	UserID         string
	XP             int64
	Level          int
	Coins          int64
	LastDailyAt    *time.Time
	BackgroundURL  *string
	LastWorkAt     *time.Time
	LastCrimeAt    *time.Time
	LastRobAt      *time.Time
	Bank           int64
	LastInterestAt *time.Time
	JobID          *int
}

// GetUserEconomy retrieves the economy record for a user in a guild.
func (db *DB) GetUserEconomy(ctx context.Context, guildID, userID string) (*UserEconomy, error) {
	query := `
		SELECT guild_id, user_id, xp, level, coins, last_daily_at, background_url, last_work_at, last_crime_at, last_rob_at, bank, last_interest_at, job_id
		FROM user_economy
		WHERE guild_id = $1 AND user_id = $2
	`
	row := db.Pool.QueryRow(ctx, query, guildID, userID)
	var e UserEconomy
	err := row.Scan(&e.GuildID, &e.UserID, &e.XP, &e.Level, &e.Coins, &e.LastDailyAt, &e.BackgroundURL, &e.LastWorkAt, &e.LastCrimeAt, &e.LastRobAt, &e.Bank, &e.LastInterestAt, &e.JobID)
	if err != nil {
		return nil, err
	}
	return &e, nil
}

// UpdateRobActivity updates a user's last rob time.
func (db *DB) UpdateRobActivity(ctx context.Context, guildID, userID string) error {
	query := `
		UPDATE user_economy
		SET last_rob_at = NOW()
		WHERE guild_id = $1 AND user_id = $2
	`
	_, err := db.Pool.Exec(ctx, query, guildID, userID)
	return err
}

// RobCoins transfers coins between two users as a transaction.
func (db *DB) RobCoins(ctx context.Context, guildID, fromUserID, toUserID string, amount int64) error {
	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// Deduct from victim
	deductQuery := `
		UPDATE user_economy
		SET coins = GREATEST(0, coins - $1)
		WHERE guild_id = $2 AND user_id = $3 AND coins >= $1
	`
	cmdTag, err := tx.Exec(ctx, deductQuery, amount, guildID, fromUserID)
	if err != nil {
		return err
	}
	if cmdTag.RowsAffected() == 0 {
		return fmt.Errorf("insufficient funds for target user")
	}

	// Add to robber
	addQuery := `
		UPDATE user_economy
		SET coins = coins + $1
		WHERE guild_id = $2 AND user_id = $3
	`
	_, err = tx.Exec(ctx, addQuery, amount, guildID, toUserID)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

// UpdateWorkActivity updates a user's last work time.
func (db *DB) UpdateWorkActivity(ctx context.Context, guildID, userID string) error {
	query := `
		UPDATE user_economy
		SET last_work_at = NOW()
		WHERE guild_id = $1 AND user_id = $2
	`
	_, err := db.Pool.Exec(ctx, query, guildID, userID)
	return err
}

// UpdateCrimeActivity updates a user's last crime time.
func (db *DB) UpdateCrimeActivity(ctx context.Context, guildID, userID string) error {
	query := `
		UPDATE user_economy
		SET last_crime_at = NOW()
		WHERE guild_id = $1 AND user_id = $2
	`
	_, err := db.Pool.Exec(ctx, query, guildID, userID)
	return err
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

// GetTopUsersByCoins retrieves the top 10 users by coins in a guild.
func (db *DB) GetTopUsersByCoins(ctx context.Context, guildID string) ([]UserEconomy, error) {
	query := `
		SELECT guild_id, user_id, xp, level, coins, last_daily_at
		FROM user_economy
		WHERE guild_id = $1
		ORDER BY coins DESC
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

// GetTopUsersByXP retrieves the top 10 users by XP in a guild.

// GetTopLevelUsers retrieves the top 10 users by level and then XP in a guild.
func (db *DB) GetTopLevelUsers(ctx context.Context, guildID string) ([]UserEconomy, error) {
	query := `
		SELECT guild_id, user_id, xp, level, coins, last_daily_at, background_url, last_work_at, last_crime_at, last_rob_at, bank, last_interest_at, job_id
		FROM user_economy
		WHERE guild_id = $1
		ORDER BY level DESC, xp DESC
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
		if err := rows.Scan(&e.GuildID, &e.UserID, &e.XP, &e.Level, &e.Coins, &e.LastDailyAt, &e.BackgroundURL, &e.LastWorkAt, &e.LastCrimeAt, &e.LastRobAt, &e.Bank, &e.LastInterestAt, &e.JobID); err != nil {
			return nil, err
		}
		topUsers = append(topUsers, e)
	}

	return topUsers, rows.Err()
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
	Duration         *string `json:"duration"`
	Resolved         bool    `json:"resolved"`
	EvidenceURL      *string `json:"evidence_url"`
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
			m.id, m.guild_id, m.action, m.reason, m.created_at, m.duration, m.resolved, m.evidence_url,
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
			&a.ID, &a.GuildID, &a.Action, &a.Reason, &t, &a.Duration, &a.Resolved, &a.EvidenceURL,
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

// VoiceGeneratorConfig represents the configuration for voice generator.
type VoiceGeneratorConfig struct {
	GuildID             string  `json:"guild_id"`
	BaseChannelID       string  `json:"base_channel_id"`
	MaxChannels         int     `json:"max_channels"`
	AllowCustomNames    bool    `json:"allow_custom_names"`
	DefaultNameTemplate *string `json:"default_name_template"`
}

// SetVoiceGeneratorConfig updates the voice generator config for a guild.
func (db *DB) SetVoiceGeneratorConfig(ctx context.Context, guildID, baseChannelID string, maxChannels int, allowCustomNames bool, defaultNameTemplate *string) error {
	query := `
		INSERT INTO voice_generator_config (guild_id, base_channel_id, max_channels, allow_custom_names, default_name_template)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (guild_id) DO UPDATE SET base_channel_id = EXCLUDED.base_channel_id, max_channels = EXCLUDED.max_channels, allow_custom_names = EXCLUDED.allow_custom_names, default_name_template = EXCLUDED.default_name_template
	`
	_, err := db.Pool.Exec(ctx, query, guildID, baseChannelID, maxChannels, allowCustomNames, defaultNameTemplate)
	return err
}

// GetVoiceGeneratorConfig gets the voice generator config for a guild.
func (db *DB) GetVoiceGeneratorConfig(ctx context.Context, guildID string) (*VoiceGeneratorConfig, error) {
	query := `SELECT guild_id, base_channel_id, max_channels, allow_custom_names, default_name_template FROM voice_generator_config WHERE guild_id = $1`
	var config VoiceGeneratorConfig
	err := db.Pool.QueryRow(ctx, query, guildID).Scan(&config.GuildID, &config.BaseChannelID, &config.MaxChannels, &config.AllowCustomNames, &config.DefaultNameTemplate)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil // No config set
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
	EmojiID   string `json:"emoji_id"`
	IsCustom  bool   `json:"is_custom"`
	RoleID    string `json:"role_id"`
	GroupID   *int   `json:"group_id"`
}

// AddReactionRole adds a new reaction role mapping.
func (db *DB) AddReactionRole(ctx context.Context, messageID, emoji, emojiID string, isCustom bool, roleID string) error {
	query := `
		INSERT INTO reaction_roles (message_id, emoji, emoji_id, is_custom, role_id)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (message_id, emoji, emoji_id) DO UPDATE SET
			role_id = EXCLUDED.role_id,
			is_custom = EXCLUDED.is_custom
	`
	_, err := db.Pool.Exec(ctx, query, messageID, emoji, emojiID, isCustom, roleID)
	return err
}

// RemoveReactionRole removes a reaction role mapping.
func (db *DB) RemoveReactionRole(ctx context.Context, messageID, emoji, emojiID string) error {
	query := `
		DELETE FROM reaction_roles
		WHERE message_id = $1 AND emoji = $2 AND emoji_id = $3
	`
	_, err := db.Pool.Exec(ctx, query, messageID, emoji, emojiID)
	return err
}

// GetReactionRole retrieves a reaction role mapping.
func (db *DB) GetReactionRole(ctx context.Context, messageID, emoji, emojiID string) (*ReactionRole, error) {
	query := `
		SELECT message_id, emoji, emoji_id, is_custom, role_id, group_id
		FROM reaction_roles
		WHERE message_id = $1 AND emoji = $2 AND emoji_id = $3
	`
	var rr ReactionRole
	err := db.Pool.QueryRow(ctx, query, messageID, emoji, emojiID).Scan(&rr.MessageID, &rr.Emoji, &rr.EmojiID, &rr.IsCustom, &rr.RoleID, &rr.GroupID)
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

// GetPendingRemindersForGuild gets all undelivered reminders for a specific guild.
func (db *DB) GetPendingRemindersForGuild(ctx context.Context, guildID string) ([]Reminder, error) {
	start := time.Now()
	metrics.DBQueryLatency.WithLabelValues("GetPendingRemindersForGuild").Observe(time.Since(start).Seconds())

	query := `
		SELECT id, user_id, channel_id, guild_id, message, due_at, delivered, created_at
		FROM reminders
		WHERE guild_id = $1 AND delivered = FALSE
		ORDER BY due_at ASC
	`
	rows, err := db.Pool.Query(ctx, query, guildID)
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

// DeleteAllRemindersForUser deletes all reminders for a specific user.
func (db *DB) DeleteAllRemindersForUser(ctx context.Context, userID string) (int64, error) {
	start := time.Now()
	metrics.DBQueryLatency.WithLabelValues("DeleteAllRemindersForUser").Observe(time.Since(start).Seconds())

	query := `
		DELETE FROM reminders
		WHERE user_id = $1 AND delivered = FALSE
	`
	cmdTag, err := db.Pool.Exec(ctx, query, userID)
	if err != nil {
		return 0, err
	}
	return cmdTag.RowsAffected(), nil
}

// SnoozeReminder adds the given duration to the reminder's due time and marks it undelivered.
func (db *DB) SnoozeReminder(ctx context.Context, id int, d time.Duration) error {
	start := time.Now()
	metrics.DBQueryLatency.WithLabelValues("SnoozeReminder").Observe(time.Since(start).Seconds())

	// Set due_at to NOW() + duration to prevent immediate triggering of snoozed past-due reminders
	query := `
		UPDATE reminders
		SET due_at = NOW() + $2 * interval '1 microsecond', delivered = FALSE
		WHERE id = $1
	`
	_, err := db.Pool.Exec(ctx, query, id, d.Microseconds())
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

// AddLevelingBlacklist adds a role to the leveling blacklist for a guild.
func (db *DB) AddLevelingBlacklist(ctx context.Context, guildID, roleID string) error {
	start := time.Now()
	defer func() {
		metrics.DBQueryLatency.WithLabelValues("AddLevelingBlacklist").Observe(time.Since(start).Seconds())
	}()

	_, err := db.Pool.Exec(ctx, `
		INSERT INTO leveling_blacklist (guild_id, role_id)
		VALUES ($1, $2)
		ON CONFLICT (guild_id, role_id) DO NOTHING
	`, guildID, roleID)
	if err != nil {
		metrics.ErrorCounter.WithLabelValues("db_query").Inc()
		return fmt.Errorf("failed to add leveling blacklist: %w", err)
	}
	return nil
}

// RemoveLevelingBlacklist removes a role from the leveling blacklist for a guild.
func (db *DB) RemoveLevelingBlacklist(ctx context.Context, guildID, roleID string) error {
	start := time.Now()
	defer func() {
		metrics.DBQueryLatency.WithLabelValues("RemoveLevelingBlacklist").Observe(time.Since(start).Seconds())
	}()

	_, err := db.Pool.Exec(ctx, `
		DELETE FROM leveling_blacklist
		WHERE guild_id = $1 AND role_id = $2
	`, guildID, roleID)
	if err != nil {
		metrics.ErrorCounter.WithLabelValues("db_query").Inc()
		return fmt.Errorf("failed to remove leveling blacklist: %w", err)
	}
	return nil
}

// IsRoleBlacklisted checks if a specific role is in the leveling blacklist for a guild.
func (db *DB) IsRoleBlacklisted(ctx context.Context, guildID, roleID string) (bool, error) {
	start := time.Now()
	defer func() {
		metrics.DBQueryLatency.WithLabelValues("IsRoleBlacklisted").Observe(time.Since(start).Seconds())
	}()

	var count int
	err := db.Pool.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM leveling_blacklist
		WHERE guild_id = $1 AND role_id = $2
	`, guildID, roleID).Scan(&count)
	if err != nil {
		metrics.ErrorCounter.WithLabelValues("db_query").Inc()
		return false, fmt.Errorf("failed to check leveling blacklist: %w", err)
	}

	return count > 0, nil
}

// GetLevelingBlacklists retrieves all blacklisted roles for a guild.
func (db *DB) GetLevelingBlacklists(ctx context.Context, guildID string) ([]string, error) {
	start := time.Now()
	defer func() {
		metrics.DBQueryLatency.WithLabelValues("GetLevelingBlacklists").Observe(time.Since(start).Seconds())
	}()

	rows, err := db.Pool.Query(ctx, `
		SELECT role_id
		FROM leveling_blacklist
		WHERE guild_id = $1
	`, guildID)
	if err != nil {
		metrics.ErrorCounter.WithLabelValues("db_query").Inc()
		return nil, fmt.Errorf("failed to get leveling blacklists: %w", err)
	}
	defer rows.Close()

	var roleIDs []string
	for rows.Next() {
		var roleID string
		if err := rows.Scan(&roleID); err != nil {
			return nil, fmt.Errorf("failed to scan leveling blacklist row: %w", err)
		}
		roleIDs = append(roleIDs, roleID)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating over leveling blacklist rows: %w", err)
	}

	return roleIDs, nil
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
	ID          int            `json:"id"`
	GuildID     string         `json:"guild_id"`
	TriggerWord string         `json:"trigger_word"`
	Response    string         `json:"response"`
	IsRegex     bool           `json:"is_regex"`
	CompiledReg *regexp.Regexp `json:"-"`
	CreatedAt   time.Time      `json:"created_at"`
}

// AddAutoResponder adds a new auto-responder or updates an existing one for the same trigger.
func (db *DB) AddAutoResponder(ctx context.Context, guildID, triggerWord, response string, isRegex bool) error {
	start := time.Now()
	defer func() {
		metrics.DBQueryLatency.WithLabelValues("AddAutoResponder").Observe(time.Since(start).Seconds())
	}()

	query := `
		INSERT INTO auto_responders (guild_id, trigger_word, response, is_regex)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (guild_id, trigger_word)
		DO UPDATE SET response = EXCLUDED.response, is_regex = EXCLUDED.is_regex
	`
	_, err := db.Pool.Exec(ctx, query, guildID, triggerWord, response, isRegex)
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
		SELECT id, guild_id, trigger_word, response, is_regex, created_at
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
		err := rows.Scan(&r.ID, &r.GuildID, &r.TriggerWord, &r.Response, &r.IsRegex, &r.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan auto-responder: %w", err)
		}
		if r.IsRegex {
			re, err := regexp.Compile(r.TriggerWord)
			if err == nil {
				r.CompiledReg = re
			}
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
		SELECT id, guild_id, trigger_word, response, is_regex, created_at
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
		err := rows.Scan(&r.ID, &r.GuildID, &r.TriggerWord, &r.Response, &r.IsRegex, &r.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan auto-responder: %w", err)
		}
		if r.IsRegex {
			re, err := regexp.Compile(r.TriggerWord)
			if err == nil {
				r.CompiledReg = re
			}
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

// Todo represents a user's task.
type Todo struct {
	ID        int
	UserID    string
	Content   string
	Completed bool
	CreatedAt time.Time
}

// AddTodo adds a new todo for a user.
func (db *DB) AddTodo(ctx context.Context, userID, content string) error {
	query := `INSERT INTO todos (user_id, content) VALUES ($1, $2)`
	_, err := db.Pool.Exec(ctx, query, userID, content)
	return err
}

// GetTodos retrieves all pending and completed todos for a user.
func (db *DB) GetTodos(ctx context.Context, userID string) ([]Todo, error) {
	query := `SELECT id, user_id, content, completed, created_at FROM todos WHERE user_id = $1 ORDER BY created_at ASC`
	rows, err := db.Pool.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var todos []Todo
	for rows.Next() {
		var t Todo
		if err := rows.Scan(&t.ID, &t.UserID, &t.Content, &t.Completed, &t.CreatedAt); err != nil {
			return nil, err
		}
		todos = append(todos, t)
	}
	return todos, rows.Err()
}

// CompleteTodo marks a specific todo as completed.
func (db *DB) CompleteTodo(ctx context.Context, userID string, todoID int) error {
	query := `UPDATE todos SET completed = TRUE WHERE user_id = $1 AND id = $2`
	_, err := db.Pool.Exec(ctx, query, userID, todoID)
	return err
}

// RemoveTodo deletes a specific todo.
func (db *DB) RemoveTodo(ctx context.Context, userID string, todoID int) error {
	query := `DELETE FROM todos WHERE user_id = $1 AND id = $2`
	_, err := db.Pool.Exec(ctx, query, userID, todoID)
	return err
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

// UserNote represents a note added by a moderator to a user.
type UserNote struct {
	ID          int       `json:"id"`
	GuildID     string    `json:"guild_id"`
	UserID      string    `json:"user_id"`
	ModeratorID string    `json:"moderator_id"`
	Note        string    `json:"note"`
	CreatedAt   time.Time `json:"created_at"`
}

// AddNote adds a new note to a user.
func (db *DB) AddNote(ctx context.Context, guildID, userID, moderatorID, note string) error {
	start := time.Now()
	defer func() {
		metrics.DBQueryLatency.WithLabelValues("AddNote").Observe(time.Since(start).Seconds())
	}()
	query := `
		INSERT INTO user_notes (guild_id, user_id, moderator_id, note)
		VALUES ($1, $2, $3, $4)
	`
	_, err := db.Pool.Exec(ctx, query, guildID, userID, moderatorID, note)
	return err
}

// GetNotes retrieves all notes for a user in a guild.
func (db *DB) GetNotes(ctx context.Context, guildID, userID string) ([]UserNote, error) {
	start := time.Now()
	defer func() {
		metrics.DBQueryLatency.WithLabelValues("GetNotes").Observe(time.Since(start).Seconds())
	}()
	query := `
		SELECT id, guild_id, user_id, moderator_id, note, created_at
		FROM user_notes
		WHERE guild_id = $1 AND user_id = $2
		ORDER BY created_at DESC
	`
	rows, err := db.Pool.Query(ctx, query, guildID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var notes []UserNote
	for rows.Next() {
		var n UserNote
		if err := rows.Scan(&n.ID, &n.GuildID, &n.UserID, &n.ModeratorID, &n.Note, &n.CreatedAt); err != nil {
			return nil, err
		}
		notes = append(notes, n)
	}
	return notes, rows.Err()
}

// RemoveNote removes a specific note by ID in a guild.
func (db *DB) RemoveNote(ctx context.Context, guildID string, id int) error {
	start := time.Now()
	defer func() {
		metrics.DBQueryLatency.WithLabelValues("RemoveNote").Observe(time.Since(start).Seconds())
	}()
	query := `DELETE FROM user_notes WHERE guild_id = $1 AND id = $2`
	_, err := db.Pool.Exec(ctx, query, guildID, id)
	return err
}

// LevelRole represents a level role reward configuration in a guild.
type LevelRole struct {
	GuildID     string    `json:"guild_id"`
	Level       int       `json:"level"`
	RoleID      string    `json:"role_id"`
	CoinsReward int       `json:"coins_reward"`
	CustomMessage *string `json:"custom_message"`
	CreatedAt   time.Time `json:"created_at"`
}

// SetLevelRole adds or updates a role reward for a specific level in a guild.
func (db *DB) SetLevelRole(ctx context.Context, guildID string, level int, roleID string, coinsReward int) error {
	start := time.Now()
	defer func() {
		metrics.DBQueryLatency.WithLabelValues("SetLevelRole").Observe(time.Since(start).Seconds())
	}()

	query := `
		INSERT INTO level_roles (guild_id, level, role_id, coins_reward)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (guild_id, level) DO UPDATE SET
			role_id = EXCLUDED.role_id,
			coins_reward = EXCLUDED.coins_reward
	`
	_, err := db.Pool.Exec(ctx, query, guildID, level, roleID, coinsReward)
	return err
}

// RemoveLevelRole removes a role reward for a specific level in a guild.
func (db *DB) RemoveLevelRole(ctx context.Context, guildID string, level int) error {
	start := time.Now()
	defer func() {
		metrics.DBQueryLatency.WithLabelValues("RemoveLevelRole").Observe(time.Since(start).Seconds())
	}()

	query := `DELETE FROM level_roles WHERE guild_id = $1 AND level = $2`
	_, err := db.Pool.Exec(ctx, query, guildID, level)
	return err
}

// GetLevelRoles retrieves all configured level roles for a guild.
func (db *DB) GetLevelRoles(ctx context.Context, guildID string) ([]LevelRole, error) {
	start := time.Now()
	defer func() {
		metrics.DBQueryLatency.WithLabelValues("GetLevelRoles").Observe(time.Since(start).Seconds())
	}()

	query := `
		SELECT guild_id, level, role_id, coins_reward, custom_message, created_at
		FROM level_roles
		WHERE guild_id = $1
		ORDER BY level ASC
	`
	rows, err := db.Pool.Query(ctx, query, guildID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var roles []LevelRole
	for rows.Next() {
		var r LevelRole
		if err := rows.Scan(&r.GuildID, &r.Level, &r.RoleID, &r.CoinsReward, &r.CustomMessage, &r.CreatedAt); err != nil {
			return nil, err
		}
		roles = append(roles, r)
	}

	return roles, rows.Err()
}

// SetLevelRoleMessage sets a custom message for a specific level role reward.
func (db *DB) SetLevelRoleMessage(ctx context.Context, guildID string, level int, message string) error {
	start := time.Now()
	defer func() {
		metrics.DBQueryLatency.WithLabelValues("SetLevelRoleMessage").Observe(time.Since(start).Seconds())
	}()

	var nullMsg *string
	if message != "" {
		nullMsg = &message
	}

	query := `
		UPDATE level_roles
		SET custom_message = $1
		WHERE guild_id = $2 AND level = $3
	`
	res, err := db.Pool.Exec(ctx, query, nullMsg, guildID, level)
	if err != nil {
		return err
	}
	if res.RowsAffected() == 0 {
		return fmt.Errorf("level role not found for level %d", level)
	}
	return nil
}

// GetLevelRoleMessage retrieves the custom message for a specific level role reward.
func (db *DB) GetLevelRoleMessage(ctx context.Context, guildID string, level int) (*string, error) {
	start := time.Now()
	defer func() {
		metrics.DBQueryLatency.WithLabelValues("GetLevelRoleMessage").Observe(time.Since(start).Seconds())
	}()

	query := `SELECT custom_message FROM level_roles WHERE guild_id = $1 AND level = $2`
	var message *string
	err := db.Pool.QueryRow(ctx, query, guildID, level).Scan(&message)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil // No role configured for this level
		}
		return nil, err
	}

	return message, nil
}

// GetLevelRole retrieves the configured role reward for a specific level in a guild.
func (db *DB) GetLevelRole(ctx context.Context, guildID string, level int) (*string, int, *string, error) {
	start := time.Now()
	defer func() {
		metrics.DBQueryLatency.WithLabelValues("GetLevelRole").Observe(time.Since(start).Seconds())
	}()

	query := `SELECT role_id, coins_reward, custom_message FROM level_roles WHERE guild_id = $1 AND level = $2`
	var roleID string
	var coinsReward int
	var customMessage *string
	err := db.Pool.QueryRow(ctx, query, guildID, level).Scan(&roleID, &coinsReward, &customMessage)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, 0, nil, nil // No role configured for this level
		}
		return nil, 0, nil, err
	}
	return &roleID, coinsReward, customMessage, nil
}

// SetVoiceLogChannel sets the voice log channel for a guild.
func (db *DB) SetVoiceLogChannel(ctx context.Context, guildID string, channelID string) error {
	query := `
		INSERT INTO voice_log_config (guild_id, channel_id, updated_at)
		VALUES ($1, $2, CURRENT_TIMESTAMP)
		ON CONFLICT (guild_id) DO UPDATE
		SET channel_id = EXCLUDED.channel_id,
		    updated_at = CURRENT_TIMESTAMP
	`
	_, err := db.Pool.Exec(ctx, query, guildID, channelID)
	return err
}

// GetVoiceLogChannel retrieves the voice log channel for a guild.
func (db *DB) GetVoiceLogChannel(ctx context.Context, guildID string) (*string, error) {
	query := `
		SELECT channel_id
		FROM voice_log_config
		WHERE guild_id = $1
	`
	var channelID string
	err := db.Pool.QueryRow(ctx, query, guildID).Scan(&channelID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil // No config found
		}
		return nil, err
	}
	return &channelID, nil
}

// GetReputation retrieves a user's reputation points in a guild.

// GetTopReputationUsers retrieves the top 10 users by reputation in a guild.
func (db *DB) GetTopReputationUsers(ctx context.Context, guildID string) ([]UserReputation, error) {
	query := `
		SELECT guild_id, user_id, rep
		FROM reputation
		WHERE guild_id = $1
		ORDER BY rep DESC
		LIMIT 10
	`
	rows, err := db.Pool.Query(ctx, query, guildID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var topUsers []UserReputation
	for rows.Next() {
		var r UserReputation
		if err := rows.Scan(&r.GuildID, &r.UserID, &r.Rep); err != nil {
			return nil, err
		}
		topUsers = append(topUsers, r)
	}
	return topUsers, rows.Err()
}

// GetReputation retrieves a user's reputation points in a guild.
func (db *DB) GetReputation(ctx context.Context, guildID, userID string) (int64, error) {
	query := `
		SELECT rep
		FROM reputation
		WHERE guild_id = $1 AND user_id = $2
	`
	var rep int64
	err := db.Pool.QueryRow(ctx, query, guildID, userID).Scan(&rep)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, nil
		}
		return 0, err
	}
	return rep, nil
}

// CanGiveReputation checks if a user has given reputation in the last 24 hours.
// Returns a boolean indicating if they can give rep, and the duration until they can.
func (db *DB) CanGiveReputation(ctx context.Context, guildID, senderID string) (bool, time.Duration, error) {
	query := `
		SELECT given_at
		FROM reputation_log
		WHERE guild_id = $1 AND sender_id = $2
		ORDER BY given_at DESC
		LIMIT 1
	`
	var lastGiven time.Time
	err := db.Pool.QueryRow(ctx, query, guildID, senderID).Scan(&lastGiven)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return true, 0, nil
		}
		return false, 0, err
	}

	cooldown := 24 * time.Hour
	timeSinceLastGiven := time.Since(lastGiven)
	if timeSinceLastGiven >= cooldown {
		return true, 0, nil
	}

	return false, cooldown - timeSinceLastGiven, nil
}

// AddReputation adds 1 reputation point to a user and logs the transaction.
func (db *DB) AddReputation(ctx context.Context, guildID, senderID, receiverID string) error {
	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// Upsert reputation
	upsertQuery := `
		INSERT INTO reputation (guild_id, user_id, rep)
		VALUES ($1, $2, 1)
		ON CONFLICT (guild_id, user_id)
		DO UPDATE SET rep = reputation.rep + 1
	`
	_, err = tx.Exec(ctx, upsertQuery, guildID, receiverID)
	if err != nil {
		return err
	}

	// Log transaction
	logQuery := `
		INSERT INTO reputation_log (guild_id, sender_id, receiver_id)
		VALUES ($1, $2, $3)
	`
	_, err = tx.Exec(ctx, logQuery, guildID, senderID, receiverID)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

// UserProfile represents a user's customized profile in a specific guild
type UserProfile struct {
	GuildID   string
	UserID    string
	Bio       *string // using pointer to handle nullable columns natively
	Color     *string // using pointer to handle nullable columns natively
	Website   *string
	Github    *string
	Twitter   *string
	UpdatedAt time.Time
}

// SetProfileBio sets the bio for a user's profile
func (db *DB) SetProfileBio(ctx context.Context, guildID, userID, bio string) error {
	query := `
		INSERT INTO user_profiles (guild_id, user_id, bio, updated_at)
		VALUES ($1, $2, $3, CURRENT_TIMESTAMP)
		ON CONFLICT (guild_id, user_id)
		DO UPDATE SET bio = EXCLUDED.bio, updated_at = CURRENT_TIMESTAMP
	`
	_, err := db.Pool.Exec(ctx, query, guildID, userID, bio)
	if err != nil {
		return fmt.Errorf("failed to set profile bio: %w", err)
	}
	return nil
}

// SetProfileColor sets the hex color for a user's profile
func (db *DB) SetProfileColor(ctx context.Context, guildID, userID, color string) error {
	query := `
		INSERT INTO user_profiles (guild_id, user_id, color, updated_at)
		VALUES ($1, $2, $3, CURRENT_TIMESTAMP)
		ON CONFLICT (guild_id, user_id)
		DO UPDATE SET color = EXCLUDED.color, updated_at = CURRENT_TIMESTAMP
	`
	_, err := db.Pool.Exec(ctx, query, guildID, userID, color)
	if err != nil {
		return fmt.Errorf("failed to set profile color: %w", err)
	}
	return nil
}

// GetProfile retrieves a user's profile for a specific guild
func (db *DB) GetProfile(ctx context.Context, guildID, userID string) (*UserProfile, error) {
	query := `
		SELECT guild_id, user_id, bio, color, website, github, twitter, updated_at
		FROM user_profiles
		WHERE guild_id = $1 AND user_id = $2
	`
	var p UserProfile
	err := db.Pool.QueryRow(ctx, query, guildID, userID).Scan(
		&p.GuildID, &p.UserID, &p.Bio, &p.Color, &p.Website, &p.Github, &p.Twitter, &p.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil // No profile exists yet
		}
		return nil, fmt.Errorf("failed to get profile: %w", err)
	}
	return &p, nil
}

// ShopItem represents an item available for purchase in a guild's shop.
type ShopItem struct {
	ID          int       `json:"id"`
	GuildID     string    `json:"guild_id"`
	Name        string    `json:"name"`
	Description *string   `json:"description"`
	Price       int64     `json:"price"`
	RoleID      *string   `json:"role_id"`
	CreatedAt   time.Time `json:"created_at"`
}

// AddShopItem adds a new item to the guild's shop.
func (db *DB) AddShopItem(ctx context.Context, guildID, name string, description *string, price int64, roleID *string) error {
	query := `
		INSERT INTO shop_items (guild_id, name, description, price, role_id)
		VALUES ($1, $2, $3, $4, $5)
	`
	_, err := db.Pool.Exec(ctx, query, guildID, name, description, price, roleID)
	return err
}

// RemoveShopItem removes an item from the guild's shop by name.
func (db *DB) RemoveShopItem(ctx context.Context, guildID, name string) error {
	query := `DELETE FROM shop_items WHERE guild_id = $1 AND name = $2`
	_, err := db.Pool.Exec(ctx, query, guildID, name)
	return err
}

// GetShopItems retrieves all items available in a guild's shop.
func (db *DB) GetShopItems(ctx context.Context, guildID string) ([]ShopItem, error) {
	query := `
		SELECT id, guild_id, name, description, price, role_id, created_at
		FROM shop_items
		WHERE guild_id = $1
		ORDER BY id ASC
	`
	rows, err := db.Pool.Query(ctx, query, guildID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []ShopItem
	for rows.Next() {
		var item ShopItem
		if err := rows.Scan(&item.ID, &item.GuildID, &item.Name, &item.Description, &item.Price, &item.RoleID, &item.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

// GetShopItem retrieves a specific shop item by name in a guild.
func (db *DB) GetShopItem(ctx context.Context, guildID, name string) (*ShopItem, error) {
	query := `
		SELECT id, guild_id, name, description, price, role_id, created_at
		FROM shop_items
		WHERE guild_id = $1 AND name = $2
	`
	var item ShopItem
	err := db.Pool.QueryRow(ctx, query, guildID, name).Scan(
		&item.ID, &item.GuildID, &item.Name, &item.Description, &item.Price, &item.RoleID, &item.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &item, nil
}

// BuyItem processes the purchase of a shop item by a user.
func (db *DB) BuyItem(ctx context.Context, guildID, userID string, itemID int, price int64) error {
	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// Deduct coins
	updateCoinsQuery := `
		UPDATE user_economy
		SET coins = coins - $1
		WHERE guild_id = $2 AND user_id = $3 AND coins >= $1
	`
	cmdTag, err := tx.Exec(ctx, updateCoinsQuery, price, guildID, userID)
	if err != nil {
		return err
	}

	if cmdTag.RowsAffected() == 0 {
		return fmt.Errorf("insufficient funds or user economy record not found")
	}

	// Add item to inventory
	insertInventoryQuery := `
		INSERT INTO user_items (guild_id, user_id, item_id, quantity)
		VALUES ($1, $2, $3, 1)
		ON CONFLICT (guild_id, user_id, item_id)
		DO UPDATE SET quantity = user_items.quantity + 1, updated_at = CURRENT_TIMESTAMP
	`
	_, err = tx.Exec(ctx, insertInventoryQuery, guildID, userID, itemID)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

// UserItem represents an item owned by a user in the shop, with quantity.
type UserItem struct {
	ID          int       `json:"id"`
	GuildID     string    `json:"guild_id"`
	UserID      string    `json:"user_id"`
	ItemID      int       `json:"item_id"`
	Quantity    int       `json:"quantity"`
	ItemName    string    `json:"item_name"`
	Description *string   `json:"description"`
	RoleID      *string   `json:"role_id"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// AddUserItem adds a specific quantity of an item to a user's inventory.
func (db *DB) AddUserItem(ctx context.Context, guildID, userID string, itemID int, quantity int) error {
	query := `
		INSERT INTO user_items (guild_id, user_id, item_id, quantity)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (guild_id, user_id, item_id)
		DO UPDATE SET quantity = user_items.quantity + EXCLUDED.quantity, updated_at = CURRENT_TIMESTAMP
	`
	_, err := db.Pool.Exec(ctx, query, guildID, userID, itemID, quantity)
	return err
}

// RemoveUserItem removes a specific quantity of an item from a user's inventory.
func (db *DB) RemoveUserItem(ctx context.Context, guildID, userID string, itemID int, quantity int) error {
	query := `
		UPDATE user_items
		SET quantity = quantity - $1, updated_at = CURRENT_TIMESTAMP
		WHERE guild_id = $2 AND user_id = $3 AND item_id = $4 AND quantity >= $1
	`
	cmdTag, err := db.Pool.Exec(ctx, query, quantity, guildID, userID, itemID)
	if err != nil {
		return err
	}

	if cmdTag.RowsAffected() == 0 {
		return fmt.Errorf("insufficient item quantity or item not found")
	}

	return nil
}

// GetUserItems retrieves all items in a user's inventory for a specific guild, grouped by quantity.
func (db *DB) GetUserItems(ctx context.Context, guildID, userID string) ([]UserItem, error) {
	query := `
		SELECT ui.id, ui.guild_id, ui.user_id, ui.item_id, ui.quantity, si.name, si.description, si.role_id, ui.updated_at
		FROM user_items ui
		JOIN shop_items si ON ui.item_id = si.id
		WHERE ui.guild_id = $1 AND ui.user_id = $2 AND ui.quantity > 0
		ORDER BY ui.updated_at DESC
	`
	rows, err := db.Pool.Query(ctx, query, guildID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []UserItem
	for rows.Next() {
		var item UserItem
		if err := rows.Scan(
			&item.ID, &item.GuildID, &item.UserID, &item.ItemID, &item.Quantity,
			&item.ItemName, &item.Description, &item.RoleID, &item.UpdatedAt,
		); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

// BirthdayConfig represents a server's birthday configuration.
type BirthdayConfig struct {
	GuildID   string
	ChannelID string
}

// WarnAutomationRule represents a rule for automated punishments based on warning counts
type WarnAutomationRule struct {
	ID               int
	GuildID          string
	WarningThreshold int
	Action           string
	Duration         *string
}

// Birthday represents a user's birthday.
type Birthday struct {
	GuildID           string
	UserID            string
	BirthMonth        int
	BirthDay          int
	LastAnnouncedYear *int
}

// SetBirthdayChannel sets the channel where birthdays will be announced.
func (db *DB) SetBirthdayChannel(ctx context.Context, guildID, channelID string) error {
	query := `
		INSERT INTO birthday_config (guild_id, channel_id)
		VALUES ($1, $2)
		ON CONFLICT (guild_id) DO UPDATE SET channel_id = EXCLUDED.channel_id
	`
	_, err := db.Pool.Exec(ctx, query, guildID, channelID)
	return err
}

// GetBirthdayChannel gets the configured birthday channel for a guild.
func (db *DB) GetBirthdayChannel(ctx context.Context, guildID string) (string, error) {
	var channelID string
	err := db.Pool.QueryRow(ctx, "SELECT channel_id FROM birthday_config WHERE guild_id = $1", guildID).Scan(&channelID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	return channelID, err
}

// SetBirthday sets a user's birthday in a guild.
func (db *DB) SetBirthday(ctx context.Context, guildID, userID string, month, day int) error {
	query := `
		INSERT INTO birthdays (guild_id, user_id, birth_month, birth_day)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (guild_id, user_id) DO UPDATE SET birth_month = EXCLUDED.birth_month, birth_day = EXCLUDED.birth_day, last_announced_year = NULL
	`
	_, err := db.Pool.Exec(ctx, query, guildID, userID, month, day)
	return err
}

// RemoveBirthday removes a user's birthday in a guild.
func (db *DB) RemoveBirthday(ctx context.Context, guildID, userID string) error {
	query := "DELETE FROM birthdays WHERE guild_id = $1 AND user_id = $2"
	_, err := db.Pool.Exec(ctx, query, guildID, userID)
	return err
}

// GetBirthdays gets all birthdays for a given month and day across all guilds.
func (db *DB) GetBirthdays(ctx context.Context, month, day int) ([]Birthday, error) {
	query := `
		SELECT guild_id, user_id, birth_month, birth_day, last_announced_year
		FROM birthdays
		WHERE birth_month = $1 AND birth_day = $2
	`
	rows, err := db.Pool.Query(ctx, query, month, day)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var birthdays []Birthday
	for rows.Next() {
		var b Birthday
		if err := rows.Scan(&b.GuildID, &b.UserID, &b.BirthMonth, &b.BirthDay, &b.LastAnnouncedYear); err != nil {
			return nil, err
		}
		birthdays = append(birthdays, b)
	}
	return birthdays, nil
}

// GetDueBirthdays gets birthdays that need to be announced today (haven't been announced this year).
func (db *DB) GetDueBirthdays(ctx context.Context, month, day, year int) ([]Birthday, error) {
	query := `
		SELECT guild_id, user_id, birth_month, birth_day, last_announced_year
		FROM birthdays
		WHERE birth_month = $1 AND birth_day = $2 AND (last_announced_year IS NULL OR last_announced_year != $3)
	`
	rows, err := db.Pool.Query(ctx, query, month, day, year)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var birthdays []Birthday
	for rows.Next() {
		var b Birthday
		if err := rows.Scan(&b.GuildID, &b.UserID, &b.BirthMonth, &b.BirthDay, &b.LastAnnouncedYear); err != nil {
			return nil, err
		}
		birthdays = append(birthdays, b)
	}
	return birthdays, nil
}

// MarkBirthdayAnnounced marks a birthday as announced for the current year.
func (db *DB) MarkBirthdayAnnounced(ctx context.Context, guildID, userID string, year int) error {
	query := "UPDATE birthdays SET last_announced_year = $1 WHERE guild_id = $2 AND user_id = $3"
	_, err := db.Pool.Exec(ctx, query, year, guildID, userID)
	return err
}

// GetGuildBirthdays gets all birthdays for a specific guild.
func (db *DB) GetGuildBirthdays(ctx context.Context, guildID string) ([]Birthday, error) {
	query := `
		SELECT guild_id, user_id, birth_month, birth_day, last_announced_year
		FROM birthdays
		WHERE guild_id = $1
		ORDER BY birth_month, birth_day
	`
	rows, err := db.Pool.Query(ctx, query, guildID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var birthdays []Birthday
	for rows.Next() {
		var b Birthday
		if err := rows.Scan(&b.GuildID, &b.UserID, &b.BirthMonth, &b.BirthDay, &b.LastAnnouncedYear); err != nil {
			return nil, err
		}
		birthdays = append(birthdays, b)
	}
	return birthdays, nil
}

// Phase 29: Temporary Voice Channels

type TempVoiceConfig struct {
	GuildID          string
	CategoryID       string
	TriggerChannelID string
}

func (db *DB) SetTempVoiceConfig(ctx context.Context, guildID, categoryID, triggerChannelID string) error {
	query := `
		INSERT INTO temp_voice_config (guild_id, category_id, trigger_channel_id)
		VALUES ($1, $2, $3)
		ON CONFLICT (guild_id) DO UPDATE SET category_id = EXCLUDED.category_id, trigger_channel_id = EXCLUDED.trigger_channel_id
	`
	_, err := db.Pool.Exec(ctx, query, guildID, categoryID, triggerChannelID)
	return err
}

func (db *DB) GetTempVoiceConfig(ctx context.Context, guildID string) (*TempVoiceConfig, error) {
	query := `SELECT guild_id, category_id, trigger_channel_id FROM temp_voice_config WHERE guild_id = $1`
	var config TempVoiceConfig
	err := db.Pool.QueryRow(ctx, query, guildID).Scan(&config.GuildID, &config.CategoryID, &config.TriggerChannelID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil // No config set
		}
		return nil, err
	}
	return &config, nil
}

type TempVoiceChannel struct {
	GuildID   string
	UserID    string
	ChannelID string
}

func (db *DB) CreateTempVoiceChannel(ctx context.Context, guildID, userID, channelID string) error {
	query := `
		INSERT INTO temp_voice_channels (guild_id, user_id, channel_id)
		VALUES ($1, $2, $3)
	`
	_, err := db.Pool.Exec(ctx, query, guildID, userID, channelID)
	return err
}

func (db *DB) GetTempVoiceChannel(ctx context.Context, channelID string) (*TempVoiceChannel, error) {
	query := `SELECT guild_id, user_id, channel_id FROM temp_voice_channels WHERE channel_id = $1`
	var channel TempVoiceChannel
	err := db.Pool.QueryRow(ctx, query, channelID).Scan(&channel.GuildID, &channel.UserID, &channel.ChannelID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &channel, nil
}

func (db *DB) DeleteTempVoiceChannel(ctx context.Context, channelID string) error {
	query := `DELETE FROM temp_voice_channels WHERE channel_id = $1`
	_, err := db.Pool.Exec(ctx, query, channelID)
	return err
}

// Marriage represents a marriage proposal or active marriage.
type Marriage struct {
	ID           int       `json:"id"`
	GuildID      string    `json:"guild_id"`
	User1ID      string    `json:"user1_id"`
	User2ID      string    `json:"user2_id"`
	Status       string    `json:"status"`
	JointBank    bool      `json:"joint_bank"`
	JointBalance int64     `json:"joint_balance"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// ProposeMarriage creates a new marriage proposal.
func (db *DB) ProposeMarriage(ctx context.Context, guildID, proposerID, proposeeID string) error {
	start := time.Now()
	defer func() {
		metrics.DBQueryLatency.WithLabelValues("ProposeMarriage").Observe(time.Since(start).Seconds())
	}()

	// Ensure neither user is already in a marriage/proposal in this guild
	var count int
	err := db.Pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM marriages
		WHERE guild_id = $1 AND (user1_id = $2 OR user2_id = $2 OR user1_id = $3 OR user2_id = $3)
	`, guildID, proposerID, proposeeID).Scan(&count)

	if err != nil {
		return err
	}
	if count > 0 {
		return fmt.Errorf("one or both users are already married or have a pending proposal")
	}

	_, err = db.Pool.Exec(ctx, `
		INSERT INTO marriages (guild_id, user1_id, user2_id, status)
		VALUES ($1, $2, $3, 'pending')
	`, guildID, proposerID, proposeeID)
	return err
}

// AcceptMarriage updates a proposal status to accepted.
func (db *DB) AcceptMarriage(ctx context.Context, guildID, proposeeID, proposerID string) error {
	start := time.Now()
	defer func() {
		metrics.DBQueryLatency.WithLabelValues("AcceptMarriage").Observe(time.Since(start).Seconds())
	}()

	cmd, err := db.Pool.Exec(ctx, `
		UPDATE marriages
		SET status = 'accepted', updated_at = CURRENT_TIMESTAMP
		WHERE guild_id = $1 AND user1_id = $2 AND user2_id = $3 AND status = 'pending'
	`, guildID, proposerID, proposeeID)

	if err != nil {
		return err
	}
	if cmd.RowsAffected() == 0 {
		return fmt.Errorf("no pending proposal found from that user")
	}
	return nil
}

// Divorce removes a marriage or pending proposal.
func (db *DB) Divorce(ctx context.Context, guildID, userID string) error {
	start := time.Now()
	defer func() {
		metrics.DBQueryLatency.WithLabelValues("Divorce").Observe(time.Since(start).Seconds())
	}()

	cmd, err := db.Pool.Exec(ctx, `
		DELETE FROM marriages
		WHERE guild_id = $1 AND (user1_id = $2 OR user2_id = $2)
	`, guildID, userID)

	if err != nil {
		return err
	}
	if cmd.RowsAffected() == 0 {
		return fmt.Errorf("you are not married or have no pending proposals")
	}
	return nil
}

// GetMarriage returns the active marriage or proposal for a user.
func (db *DB) GetMarriage(ctx context.Context, guildID, userID string) (*Marriage, error) {
	start := time.Now()
	defer func() {
		metrics.DBQueryLatency.WithLabelValues("GetMarriage").Observe(time.Since(start).Seconds())
	}()

	var m Marriage
	err := db.Pool.QueryRow(ctx, `
		SELECT id, guild_id, user1_id, user2_id, status, joint_bank, joint_balance, created_at, updated_at
		FROM marriages
		WHERE guild_id = $1 AND (user1_id = $2 OR user2_id = $2)
	`, guildID, userID).Scan(&m.ID, &m.GuildID, &m.User1ID, &m.User2ID, &m.Status, &m.JointBank, &m.JointBalance, &m.CreatedAt, &m.UpdatedAt)

	if err != nil {
		return nil, err
	}
	return &m, nil
}

// SetJointBank enables or disables the joint bank for a marriage.
func (db *DB) SetJointBank(ctx context.Context, guildID, userID string, enable bool) error {
	start := time.Now()
	defer func() {
		metrics.DBQueryLatency.WithLabelValues("SetJointBank").Observe(time.Since(start).Seconds())
	}()

	cmd, err := db.Pool.Exec(ctx, `
		UPDATE marriages
		SET joint_bank = $1, updated_at = NOW()
		WHERE guild_id = $2 AND (user1_id = $3 OR user2_id = $3) AND status = 'married'
	`, enable, guildID, userID)

	if err != nil {
		return err
	}
	if cmd.RowsAffected() == 0 {
		return fmt.Errorf("no active marriage found")
	}
	return nil
}

// DepositJoint transfers coins from a user's wallet to their joint bank.
func (db *DB) DepositJoint(ctx context.Context, guildID, userID string, amount int64) error {
	start := time.Now()
	defer func() {
		metrics.DBQueryLatency.WithLabelValues("DepositJoint").Observe(time.Since(start).Seconds())
	}()

	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// Deduct from user
	deductQuery := `
		UPDATE user_economy
		SET coins = coins - $3
		WHERE guild_id = $1 AND user_id = $2 AND coins >= $3
	`
	cmd, err := tx.Exec(ctx, deductQuery, guildID, userID, amount)
	if err != nil {
		return err
	}
	if cmd.RowsAffected() == 0 {
		return fmt.Errorf("insufficient funds")
	}

	// Add to joint bank
	addQuery := `
		UPDATE marriages
		SET joint_balance = joint_balance + $3, updated_at = NOW()
		WHERE guild_id = $1 AND (user1_id = $2 OR user2_id = $2) AND status = 'married' AND joint_bank = true
	`
	cmd, err = tx.Exec(ctx, addQuery, guildID, userID, amount)
	if err != nil {
		return err
	}
	if cmd.RowsAffected() == 0 {
		return fmt.Errorf("no active marriage with joint bank enabled found")
	}

	return tx.Commit(ctx)
}

// WithdrawJoint transfers coins from a joint bank to a user's wallet.
func (db *DB) WithdrawJoint(ctx context.Context, guildID, userID string, amount int64) error {
	start := time.Now()
	defer func() {
		metrics.DBQueryLatency.WithLabelValues("WithdrawJoint").Observe(time.Since(start).Seconds())
	}()

	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// Deduct from joint bank
	deductQuery := `
		UPDATE marriages
		SET joint_balance = joint_balance - $3, updated_at = NOW()
		WHERE guild_id = $1 AND (user1_id = $2 OR user2_id = $2) AND status = 'married' AND joint_bank = true AND joint_balance >= $3
	`
	cmd, err := tx.Exec(ctx, deductQuery, guildID, userID, amount)
	if err != nil {
		return err
	}
	if cmd.RowsAffected() == 0 {
		return fmt.Errorf("insufficient funds in joint bank or no active marriage with joint bank enabled found")
	}

	// Add to user
	addQuery := `
		INSERT INTO user_economy (guild_id, user_id, coins)
		VALUES ($1, $2, $3)
		ON CONFLICT (guild_id, user_id) DO UPDATE SET
			coins = user_economy.coins + EXCLUDED.coins
	`
	_, err = tx.Exec(ctx, addQuery, guildID, userID, amount)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

// GetJointBalance retrieves the current joint bank balance for a marriage.
func (db *DB) GetJointBalance(ctx context.Context, guildID, userID string) (int64, error) {
	start := time.Now()
	defer func() {
		metrics.DBQueryLatency.WithLabelValues("GetJointBalance").Observe(time.Since(start).Seconds())
	}()

	var balance int64
	err := db.Pool.QueryRow(ctx, `
		SELECT joint_balance
		FROM marriages
		WHERE guild_id = $1 AND (user1_id = $2 OR user2_id = $2) AND status = 'married'
	`, guildID, userID).Scan(&balance)

	if err != nil {
		return 0, err
	}
	return balance, nil
}

// CountingConfig represents the configuration and state of a counting channel for a guild.
type CountingConfig struct {
	GuildID       string
	ChannelID     string
	CurrentNumber int
	LastUserID    *string
	UpdatedAt     time.Time
}

// SetCountingChannel sets the designated counting channel for a guild.
func (db *DB) SetCountingChannel(ctx context.Context, guildID, channelID string) error {
	query := `
		INSERT INTO counting_config (guild_id, channel_id, current_number, updated_at)
		VALUES ($1, $2, 0, CURRENT_TIMESTAMP)
		ON CONFLICT (guild_id) DO UPDATE SET
			channel_id = EXCLUDED.channel_id,
			current_number = 0,
			last_user_id = NULL,
			updated_at = CURRENT_TIMESTAMP
	`
	_, err := db.Pool.Exec(ctx, query, guildID, channelID)
	return err
}

// GetCountingChannel retrieves the counting channel configuration for a guild.
func (db *DB) GetCountingChannel(ctx context.Context, guildID string) (*CountingConfig, error) {
	query := `
		SELECT guild_id, channel_id, current_number, last_user_id, updated_at
		FROM counting_config
		WHERE guild_id = $1
	`
	var config CountingConfig
	err := db.Pool.QueryRow(ctx, query, guildID).Scan(
		&config.GuildID, &config.ChannelID, &config.CurrentNumber, &config.LastUserID, &config.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil // Return nil, nil if config is not found
		}
		return nil, err
	}
	return &config, nil
}

// UpdateCountingNumber updates the current number and the last user who counted.
func (db *DB) UpdateCountingNumber(ctx context.Context, guildID string, number int, userID string) error {
	query := `
		UPDATE counting_config
		SET current_number = $1, last_user_id = $2, updated_at = CURRENT_TIMESTAMP
		WHERE guild_id = $3
	`
	_, err := db.Pool.Exec(ctx, query, number, userID, guildID)
	return err
}

// ResetCountingNumber resets the counting number back to 0.
func (db *DB) ResetCountingNumber(ctx context.Context, guildID string) error {
	query := `
		UPDATE counting_config
		SET current_number = 0, last_user_id = NULL, updated_at = CURRENT_TIMESTAMP
		WHERE guild_id = $1
	`
	_, err := db.Pool.Exec(ctx, query, guildID)
	return err
}

// TriviaScore represents a user's trivia score in a guild.
type TriviaScore struct {
	GuildID string
	UserID  string
	Score   int
}

// AddTriviaScore increments a user's trivia score by 1.
func (db *DB) AddTriviaScore(ctx context.Context, guildID, userID string) error {
	query := `
		INSERT INTO trivia_scores (guild_id, user_id, score)
		VALUES ($1, $2, 1)
		ON CONFLICT (guild_id, user_id) DO UPDATE SET
			score = trivia_scores.score + 1,
			updated_at = CURRENT_TIMESTAMP
	`
	_, err := db.Pool.Exec(ctx, query, guildID, userID)
	return err
}

// GetTriviaLeaderboard returns the top 10 trivia players in a guild.
func (db *DB) GetTriviaLeaderboard(ctx context.Context, guildID string) ([]TriviaScore, error) {
	query := `
		SELECT guild_id, user_id, score
		FROM trivia_scores
		WHERE guild_id = $1
		ORDER BY score DESC
		LIMIT 10
	`
	rows, err := db.Pool.Query(ctx, query, guildID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var scores []TriviaScore
	for rows.Next() {
		var s TriviaScore
		if err := rows.Scan(&s.GuildID, &s.UserID, &s.Score); err != nil {
			return nil, err
		}
		scores = append(scores, s)
	}
	return scores, rows.Err()
}

// AddCoins awards a specific amount of coins to a user.
func (db *DB) AddCoins(ctx context.Context, guildID, userID string, amount int) error {
	query := `
		INSERT INTO user_economy (guild_id, user_id, coins, last_daily_at)
		VALUES ($1, $2, $3, NOW())
		ON CONFLICT (guild_id, user_id) DO UPDATE SET
			coins = user_economy.coins + EXCLUDED.coins
	`
	_, err := db.Pool.Exec(ctx, query, guildID, userID, amount)
	return err
}

// RemoveCoins subtracts a specific amount of coins from a user.
func (db *DB) RemoveCoins(ctx context.Context, guildID, userID string, amount int) error {
	query := `
		UPDATE user_economy
		SET coins = GREATEST(0, coins - $3)
		WHERE guild_id = $1 AND user_id = $2
	`
	_, err := db.Pool.Exec(ctx, query, guildID, userID, amount)
	return err
}

// Transfer represents a coin transfer between two users.
type Transfer struct {
	ID         int
	GuildID    string
	SenderID   string
	ReceiverID string
	Amount     int64
	CreatedAt  time.Time
}

// TransferCoins safely transfers coins between two users in a transaction.
func (db *DB) TransferCoins(ctx context.Context, guildID, senderID, receiverID string, amount int64) error {
	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) // Safe to call even if committed

	// Deduct from sender
	deductQuery := `
		UPDATE user_economy
		SET coins = coins - $3
		WHERE guild_id = $1 AND user_id = $2 AND coins >= $3
	`
	tag, err := tx.Exec(ctx, deductQuery, guildID, senderID, amount)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("insufficient funds or user not found")
	}

	// Add to receiver
	addQuery := `
		INSERT INTO user_economy (guild_id, user_id, coins, last_daily_at)
		VALUES ($1, $2, $3, '1970-01-01 00:00:00')
		ON CONFLICT (guild_id, user_id) DO UPDATE SET
			coins = user_economy.coins + EXCLUDED.coins
	`
	_, err = tx.Exec(ctx, addQuery, guildID, receiverID, amount)
	if err != nil {
		return err
	}

	// Log transfer
	logQuery := `
		INSERT INTO transfers (guild_id, sender_id, receiver_id, amount)
		VALUES ($1, $2, $3, $4)
	`
	_, err = tx.Exec(ctx, logQuery, guildID, senderID, receiverID, amount)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

// GetTransfers returns the recent coin transfers for a user.
func (db *DB) GetTransfers(ctx context.Context, guildID, userID string, limit int) ([]*Transfer, error) {
	query := `
		SELECT id, guild_id, sender_id, receiver_id, amount, created_at
		FROM transfers
		WHERE guild_id = $1 AND (sender_id = $2 OR receiver_id = $2)
		ORDER BY created_at DESC
		LIMIT $3
	`
	rows, err := db.Pool.Query(ctx, query, guildID, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var transfers []*Transfer
	for rows.Next() {
		t := &Transfer{}
		if err := rows.Scan(&t.ID, &t.GuildID, &t.SenderID, &t.ReceiverID, &t.Amount, &t.CreatedAt); err != nil {
			return nil, err
		}
		transfers = append(transfers, t)
	}
	return transfers, rows.Err()
}

// GamblingStats represents a user's gambling statistics.
type GamblingStats struct {
	GuildID     string
	UserID      string
	CoinsWon    int64
	CoinsLost   int64
	GamesPlayed int
	GamesWon    int
	GamesLost   int
}

// UpdateGamblingStats updates a user's gambling stats.
func (db *DB) UpdateGamblingStats(ctx context.Context, guildID, userID string, amountWon, amountLost int) error {
	wonInt := 0
	lostInt := 0
	if amountWon > 0 {
		wonInt = 1
	} else if amountLost > 0 {
		lostInt = 1
	}

	query := `
		INSERT INTO gambling_stats (guild_id, user_id, coins_won, coins_lost, games_played, games_won, games_lost)
		VALUES ($1, $2, $3, $4, 1, $5, $6)
		ON CONFLICT (guild_id, user_id) DO UPDATE SET
			coins_won = gambling_stats.coins_won + EXCLUDED.coins_won,
			coins_lost = gambling_stats.coins_lost + EXCLUDED.coins_lost,
			games_played = gambling_stats.games_played + 1,
			games_won = gambling_stats.games_won + EXCLUDED.games_won,
			games_lost = gambling_stats.games_lost + EXCLUDED.games_lost
	`
	_, err := db.Pool.Exec(ctx, query, guildID, userID, amountWon, amountLost, wonInt, lostInt)
	return err
}

// GetGamblingStats retrieves a user's gambling stats.
func (db *DB) GetGamblingStats(ctx context.Context, guildID, userID string) (*GamblingStats, error) {
	query := `
		SELECT guild_id, user_id, coins_won, coins_lost, games_played, games_won, games_lost
		FROM gambling_stats
		WHERE guild_id = $1 AND user_id = $2
	`
	row := db.Pool.QueryRow(ctx, query, guildID, userID)
	var stats GamblingStats
	err := row.Scan(&stats.GuildID, &stats.UserID, &stats.CoinsWon, &stats.CoinsLost, &stats.GamesPlayed, &stats.GamesWon, &stats.GamesLost)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &stats, nil
}

// CustomCommand represents a custom command for a guild.
type CustomCommand struct {
	ID        int       `json:"id"`
	GuildID   string    `json:"guild_id"`
	Name      string    `json:"name"`
	Response  string    `json:"response"`
	CreatedAt time.Time `json:"created_at"`
}

type Snipe struct {
	ChannelID      string
	MessageContent string
	AuthorID       string
	Timestamp      time.Time
}

type EditSnipe struct {
	ChannelID  string
	OldContent string
	NewContent string
	AuthorID   string
	Timestamp  time.Time
}

// AddCustomCommand creates or updates a custom command.
func (db *DB) AddCustomCommand(ctx context.Context, guildID, name, response string) error {
	query := `
		INSERT INTO custom_commands (guild_id, name, response)
		VALUES ($1, $2, $3)
		ON CONFLICT (guild_id, name) DO UPDATE SET
			response = EXCLUDED.response
	`
	_, err := db.Pool.Exec(ctx, query, guildID, name, response)
	return err
}

// RemoveCustomCommand deletes a custom command.
func (db *DB) RemoveCustomCommand(ctx context.Context, guildID, name string) error {
	query := `DELETE FROM custom_commands WHERE guild_id = $1 AND name = $2`
	_, err := db.Pool.Exec(ctx, query, guildID, name)
	return err
}

// ListCustomCommands returns all custom commands for a guild.
func (db *DB) ListCustomCommands(ctx context.Context, guildID string) ([]CustomCommand, error) {
	query := `
		SELECT id, guild_id, name, response, created_at
		FROM custom_commands
		WHERE guild_id = $1
		ORDER BY name ASC
	`
	rows, err := db.Pool.Query(ctx, query, guildID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var commands []CustomCommand
	for rows.Next() {
		var c CustomCommand
		if err := rows.Scan(&c.ID, &c.GuildID, &c.Name, &c.Response, &c.CreatedAt); err != nil {
			return nil, err
		}
		commands = append(commands, c)
	}
	return commands, rows.Err()
}

// GetCustomCommand gets a specific custom command by name.
func (db *DB) GetCustomCommand(ctx context.Context, guildID, name string) (*CustomCommand, error) {
	query := `
		SELECT id, guild_id, name, response, created_at
		FROM custom_commands
		WHERE guild_id = $1 AND name = $2
	`
	var c CustomCommand
	err := db.Pool.QueryRow(ctx, query, guildID, name).Scan(&c.ID, &c.GuildID, &c.Name, &c.Response, &c.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &c, nil
}

// AddSnipe saves a deleted message to the snipes table.
func (db *DB) AddSnipe(ctx context.Context, channelID, content, authorID string) error {
	query := `
		INSERT INTO snipes (channel_id, message_content, author_id)
		VALUES ($1, $2, $3)
		ON CONFLICT (channel_id) DO UPDATE
		SET message_content = EXCLUDED.message_content,
		    author_id = EXCLUDED.author_id,
		    timestamp = CURRENT_TIMESTAMP
	`
	_, err := db.Pool.Exec(ctx, query, channelID, content, authorID)
	return err
}

// GetSnipe retrieves the most recently deleted message for a channel.
func (db *DB) GetSnipe(ctx context.Context, channelID string) (*Snipe, error) {
	query := `
		SELECT channel_id, message_content, author_id, timestamp
		FROM snipes
		WHERE channel_id = $1
	`
	var s Snipe
	err := db.Pool.QueryRow(ctx, query, channelID).Scan(&s.ChannelID, &s.MessageContent, &s.AuthorID, &s.Timestamp)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &s, nil
}

// AddEditSnipe saves an edited message's old and new content.
func (db *DB) AddEditSnipe(ctx context.Context, channelID, oldContent, newContent, authorID string) error {
	query := `
		INSERT INTO edit_snipes (channel_id, old_content, new_content, author_id)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (channel_id) DO UPDATE
		SET old_content = EXCLUDED.old_content,
		    new_content = EXCLUDED.new_content,
		    author_id = EXCLUDED.author_id,
		    timestamp = CURRENT_TIMESTAMP
	`
	_, err := db.Pool.Exec(ctx, query, channelID, oldContent, newContent, authorID)
	return err
}

// GetEditSnipe retrieves the most recently edited message for a channel.
func (db *DB) GetEditSnipe(ctx context.Context, channelID string) (*EditSnipe, error) {
	query := `
		SELECT channel_id, old_content, new_content, author_id, timestamp
		FROM edit_snipes
		WHERE channel_id = $1
	`
	var es EditSnipe
	err := db.Pool.QueryRow(ctx, query, channelID).Scan(&es.ChannelID, &es.OldContent, &es.NewContent, &es.AuthorID, &es.Timestamp)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &es, nil
}

// SetConfessionChannel sets the confession channel for a guild.
func (db *DB) SetConfessionChannel(ctx context.Context, guildID string, channelID string) error {
	query := `
		INSERT INTO confession_config (guild_id, channel_id)
		VALUES ($1, $2)
		ON CONFLICT (guild_id) DO UPDATE SET channel_id = $2
	`
	_, err := db.Pool.Exec(ctx, query, guildID, channelID)
	return err
}

// GetConfessionChannel gets the confession channel for a guild.
func (db *DB) GetConfessionChannel(ctx context.Context, guildID string) (string, error) {
	query := `SELECT channel_id FROM confession_config WHERE guild_id = $1`
	var channelID string
	err := db.Pool.QueryRow(ctx, query, guildID).Scan(&channelID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", nil
		}
		return "", err
	}
	return channelID, nil
}

// RoleMenu represents a role menu message.
type RoleMenu struct {
	MessageID string
	GuildID   string
	ChannelID string
}

// RoleMenuOption represents a single option in a role menu.
type RoleMenuOption struct {
	MessageID   string
	RoleID      string
	Emoji       string
	Label       string
	Description string
}

// CreateRoleMenu creates a new role menu.
func (db *DB) CreateRoleMenu(ctx context.Context, messageID, guildID, channelID string) error {
	query := `
		INSERT INTO role_menus (message_id, guild_id, channel_id)
		VALUES ($1, $2, $3)
		ON CONFLICT (message_id) DO NOTHING
	`
	_, err := db.Pool.Exec(ctx, query, messageID, guildID, channelID)
	return err
}

// AddRoleMenuOption adds an option to an existing role menu.
func (db *DB) AddRoleMenuOption(ctx context.Context, messageID, roleID, emoji, label, description string) error {
	query := `
		INSERT INTO role_menu_options (message_id, role_id, emoji, label, description)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (message_id, role_id) DO UPDATE
		SET emoji = EXCLUDED.emoji, label = EXCLUDED.label, description = EXCLUDED.description
	`
	_, err := db.Pool.Exec(ctx, query, messageID, roleID, emoji, label, description)
	return err
}

// GetRoleMenu gets all options for a specific role menu message.
func (db *DB) GetRoleMenu(ctx context.Context, messageID string) ([]RoleMenuOption, error) {
	query := `
		SELECT message_id, role_id, emoji, label, description
		FROM role_menu_options
		WHERE message_id = $1
	`
	rows, err := db.Pool.Query(ctx, query, messageID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var options []RoleMenuOption
	for rows.Next() {
		var o RoleMenuOption
		var desc *string
		if err := rows.Scan(&o.MessageID, &o.RoleID, &o.Emoji, &o.Label, &desc); err != nil {
			return nil, err
		}
		if desc != nil {
			o.Description = *desc
		}
		options = append(options, o)
	}
	return options, rows.Err()
}

// Quote represents a saved quote.
type Quote struct {
	ID        int       `json:"id"`
	GuildID   string    `json:"guild_id"`
	UserID    string    `json:"user_id"`
	AuthorID  string    `json:"author_id"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

// AddQuote adds a new quote to the database.
func (db *DB) AddQuote(ctx context.Context, guildID, userID, authorID, content string) (int, error) {
	query := `
		INSERT INTO quotes (guild_id, user_id, author_id, content)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`
	var id int
	err := db.Pool.QueryRow(ctx, query, guildID, userID, authorID, content).Scan(&id)
	return id, err
}

// GetQuote retrieves a specific quote by ID and Guild ID.
func (db *DB) GetQuote(ctx context.Context, id int, guildID string) (*Quote, error) {
	query := `
		SELECT id, guild_id, user_id, author_id, content, created_at
		FROM quotes
		WHERE id = $1 AND guild_id = $2
	`
	q := &Quote{}
	err := db.Pool.QueryRow(ctx, query, id, guildID).Scan(
		&q.ID, &q.GuildID, &q.UserID, &q.AuthorID, &q.Content, &q.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil // Not found
		}
		return nil, err
	}
	return q, nil
}

// GetRandomQuote retrieves a random quote for a guild.
func (db *DB) GetRandomQuote(ctx context.Context, guildID string) (*Quote, error) {
	query := `
		SELECT id, guild_id, user_id, author_id, content, created_at
		FROM quotes
		WHERE guild_id = $1
		ORDER BY RANDOM()
		LIMIT 1
	`
	q := &Quote{}
	err := db.Pool.QueryRow(ctx, query, guildID).Scan(
		&q.ID, &q.GuildID, &q.UserID, &q.AuthorID, &q.Content, &q.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil // No quotes found
		}
		return nil, err
	}
	return q, nil
}

// DeleteQuote deletes a quote by ID and Guild ID.
func (db *DB) DeleteQuote(ctx context.Context, id int, guildID string) error {
	query := `
		DELETE FROM quotes
		WHERE id = $1 AND guild_id = $2
	`
	_, err := db.Pool.Exec(ctx, query, id, guildID)
	return err
}

type CustomRole struct {
	ID      int
	GuildID string
	UserID  string
	RoleID  string
	Name    string
	Color   int
	IconURL string
}

func (db *DB) CreateCustomRole(ctx context.Context, guildID, userID, roleID, name string, color int, iconURL string) error {
	query := `
		INSERT INTO custom_roles (guild_id, user_id, role_id, name, color, icon_url)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (guild_id, user_id) DO UPDATE SET
			role_id = EXCLUDED.role_id,
			name = EXCLUDED.name,
			color = EXCLUDED.color,
			icon_url = EXCLUDED.icon_url
	`
	_, err := db.Pool.Exec(ctx, query, guildID, userID, roleID, name, color, iconURL)
	return err
}

func (db *DB) GetCustomRole(ctx context.Context, guildID, userID string) (*CustomRole, error) {
	query := `
		SELECT id, guild_id, user_id, role_id, name, color, icon_url
		FROM custom_roles
		WHERE guild_id = $1 AND user_id = $2
	`
	role := &CustomRole{}
	err := db.Pool.QueryRow(ctx, query, guildID, userID).Scan(
		&role.ID, &role.GuildID, &role.UserID, &role.RoleID, &role.Name, &role.Color, &role.IconURL,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return role, nil
}

func (db *DB) UpdateCustomRole(ctx context.Context, guildID, userID, name string, color int, iconURL string) error {
	query := `
		UPDATE custom_roles
		SET name = $3, color = $4, icon_url = $5
		WHERE guild_id = $1 AND user_id = $2
	`
	_, err := db.Pool.Exec(ctx, query, guildID, userID, name, color, iconURL)
	return err
}

func (db *DB) DeleteCustomRole(ctx context.Context, guildID, userID string) error {
	query := `
		DELETE FROM custom_roles
		WHERE guild_id = $1 AND user_id = $2
	`
	_, err := db.Pool.Exec(ctx, query, guildID, userID)
	return err
}

// SetMusicChannel sets the music channel for a guild.
func (db *DB) SetMusicChannel(ctx context.Context, guildID, channelID string) error {
	query := `
		INSERT INTO music_config (guild_id, music_channel_id)
		VALUES ($1, $2)
		ON CONFLICT (guild_id) DO UPDATE
		SET music_channel_id = $2, updated_at = CURRENT_TIMESTAMP
	`
	_, err := db.Pool.Exec(ctx, query, guildID, channelID)
	return err
}

// GetMusicChannel retrieves the music channel ID for a guild.
func (db *DB) GetMusicChannel(ctx context.Context, guildID string) (string, error) {
	query := `
		SELECT music_channel_id FROM music_config
		WHERE guild_id = $1
	`
	var channelID string
	err := db.Pool.QueryRow(ctx, query, guildID).Scan(&channelID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", nil // No channel configured
		}
		return "", err
	}
	return channelID, nil
}

// SetReportChannel configures the channel where reports will be sent for a guild.
func (db *DB) SetReportChannel(ctx context.Context, guildID, channelID string) error {
	query := `
		INSERT INTO report_config (guild_id, report_channel_id)
		VALUES ($1, $2)
		ON CONFLICT (guild_id) DO UPDATE
		SET report_channel_id = $2, updated_at = CURRENT_TIMESTAMP
	`
	_, err := db.Pool.Exec(ctx, query, guildID, channelID)
	return err
}

// GetReportChannel retrieves the report channel ID for a guild.
func (db *DB) GetReportChannel(ctx context.Context, guildID string) (string, error) {
	query := `
		SELECT report_channel_id FROM report_config
		WHERE guild_id = $1
	`
	var channelID string
	err := db.Pool.QueryRow(ctx, query, guildID).Scan(&channelID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", nil // No channel configured
		}
		return "", err
	}
	return channelID, nil
}

// CreateReport creates a new report.
func (db *DB) CreateReport(ctx context.Context, guildID, authorID, targetID, reason string) (int, error) {
	query := `
		INSERT INTO reports (guild_id, author_id, target_id, reason)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`
	var id int
	err := db.Pool.QueryRow(ctx, query, guildID, authorID, targetID, reason).Scan(&id)
	return id, err
}

// SetWelcomeImage configures the welcome image URL for a guild.
func (db *DB) SetWelcomeImage(ctx context.Context, guildID, imageURL string) error {
	query := `
		INSERT INTO welcome_images (guild_id, image_url)
		VALUES ($1, $2)
		ON CONFLICT (guild_id) DO UPDATE
		SET image_url = $2, updated_at = CURRENT_TIMESTAMP
	`
	_, err := db.Pool.Exec(ctx, query, guildID, imageURL)
	return err
}

// GetWelcomeImage retrieves the welcome image URL for a guild.
func (db *DB) GetWelcomeImage(ctx context.Context, guildID string) (string, error) {
	query := `
		SELECT image_url FROM welcome_images
		WHERE guild_id = $1
	`
	var imageURL string
	err := db.Pool.QueryRow(ctx, query, guildID).Scan(&imageURL)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", nil // No image configured
		}
		return "", err
	}
	return imageURL, nil
}

// SetBackgroundURL sets the background URL for a user's rank profile.
func (db *DB) SetBackgroundURL(ctx context.Context, guildID, userID, backgroundURL string) error {
	query := `
		INSERT INTO user_economy (guild_id, user_id, background_url)
		VALUES ($1, $2, $3)
		ON CONFLICT (guild_id, user_id) DO UPDATE SET
			background_url = EXCLUDED.background_url
	`
	_, err := db.Pool.Exec(ctx, query, guildID, userID, backgroundURL)
	return err
}

// SetAutoRole sets the auto role ID for a guild in the autorole_config table.
func (db *DB) SetAutoRole(ctx context.Context, guildID, roleID string) error {
	query := `
		INSERT INTO autorole_config (guild_id, role_id)
		VALUES ($1, $2)
		ON CONFLICT (guild_id) DO UPDATE SET
			role_id = EXCLUDED.role_id,
			updated_at = NOW()
	`
	_, err := db.Pool.Exec(ctx, query, guildID, roleID)
	return err
}

// GetAutoRole gets the auto role ID for a guild from the autorole_config table.
func (db *DB) GetAutoRole(ctx context.Context, guildID string) (string, error) {
	query := `SELECT role_id FROM autorole_config WHERE guild_id = $1`
	var roleID string
	err := db.Pool.QueryRow(ctx, query, guildID).Scan(&roleID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", nil // No auto-role configured
		}
		return "", err
	}
	return roleID, nil
}

// AddMediaChannel adds a channel to the media_channels table.
func (db *DB) AddMediaChannel(ctx context.Context, guildID, channelID string) error {
	query := `
		INSERT INTO media_channels (guild_id, channel_id)
		VALUES ($1, $2)
		ON CONFLICT (guild_id, channel_id) DO NOTHING
	`
	_, err := db.Pool.Exec(ctx, query, guildID, channelID)
	return err
}

// RemoveMediaChannel removes a channel from the media_channels table.
func (db *DB) RemoveMediaChannel(ctx context.Context, guildID, channelID string) error {
	query := `DELETE FROM media_channels WHERE guild_id = $1 AND channel_id = $2`
	_, err := db.Pool.Exec(ctx, query, guildID, channelID)
	return err
}

// ListMediaChannels gets all media channels for a guild.
func (db *DB) ListMediaChannels(ctx context.Context, guildID string) ([]string, error) {
	query := `SELECT channel_id FROM media_channels WHERE guild_id = $1`
	rows, err := db.Pool.Query(ctx, query, guildID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var channels []string
	for rows.Next() {
		var channelID string
		if err := rows.Scan(&channelID); err != nil {
			return nil, err
		}
		channels = append(channels, channelID)
	}

	return channels, rows.Err()
}

// IsMediaChannel checks if a channel is configured as a media channel.
func (db *DB) IsMediaChannel(ctx context.Context, guildID, channelID string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM media_channels WHERE guild_id = $1 AND channel_id = $2)`
	var exists bool
	err := db.Pool.QueryRow(ctx, query, guildID, channelID).Scan(&exists)
	return exists, err
}

// --- Badges ---

type Badge struct {
	ID          int
	Name        string
	Emoji       string
	Description string
}

type UserBadge struct {
	UserID    string
	BadgeID   int
	AwardedAt time.Time
}

func (db *DB) CreateBadge(name, emoji, description string) error {
	query := `INSERT INTO available_badges (name, emoji, description) VALUES ($1, $2, $3)`
	_, err := db.Pool.Exec(context.Background(), query, name, emoji, description)
	return err
}

func (db *DB) GetAllBadges() ([]*Badge, error) {
	query := `SELECT id, name, emoji, description FROM available_badges ORDER BY id`
	rows, err := db.Pool.Query(context.Background(), query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var badges []*Badge
	for rows.Next() {
		b := &Badge{}
		if err := rows.Scan(&b.ID, &b.Name, &b.Emoji, &b.Description); err != nil {
			return nil, err
		}
		badges = append(badges, b)
	}
	return badges, nil
}

// Mute represents a temporary user timeout.
type Mute struct {
	ID          int       `json:"id"`
	GuildID     string    `json:"guild_id"`
	UserID      string    `json:"user_id"`
	ModeratorID string    `json:"moderator_id"`
	Reason      string    `json:"reason"`
	ExpiresAt   time.Time `json:"expires_at"`
	CreatedAt   time.Time `json:"created_at"`
}

// AddMute adds a new mute record to the database.
func (db *DB) AddMute(ctx context.Context, guildID, userID, moderatorID, reason string, expiresAt time.Time) error {
	query := `
		INSERT INTO mutes (guild_id, user_id, moderator_id, reason, expires_at)
		VALUES ($1, $2, $3, $4, $5)
	`
	_, err := db.Pool.Exec(ctx, query, guildID, userID, moderatorID, reason, expiresAt)
	return err
}

// GetMute fetches the active mute for a user in a guild.
func (db *DB) GetMute(ctx context.Context, guildID, userID string) (*Mute, error) {
	query := `
		SELECT id, guild_id, user_id, moderator_id, reason, expires_at, created_at
		FROM mutes
		WHERE guild_id = $1 AND user_id = $2 AND expires_at > NOW()
		ORDER BY expires_at DESC
		LIMIT 1
	`
	m := &Mute{}
	err := db.Pool.QueryRow(ctx, query, guildID, userID).Scan(
		&m.ID, &m.GuildID, &m.UserID, &m.ModeratorID, &m.Reason, &m.ExpiresAt, &m.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return m, nil
}

// RemoveMute removes an active mute record.
func (db *DB) RemoveMute(ctx context.Context, guildID, userID string) error {
	query := `
		DELETE FROM mutes
		WHERE guild_id = $1 AND user_id = $2
	`
	_, err := db.Pool.Exec(ctx, query, guildID, userID)
	return err
}

// GetExpiredMutes returns all mutes that have expired.
func (db *DB) GetExpiredMutes(ctx context.Context) ([]*Mute, error) {
	query := `
		SELECT id, guild_id, user_id, moderator_id, reason, expires_at, created_at
		FROM mutes
		WHERE expires_at <= NOW()
	`
	rows, err := db.Pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var mutes []*Mute
	for rows.Next() {
		m := &Mute{}
		if err := rows.Scan(&m.ID, &m.GuildID, &m.UserID, &m.ModeratorID, &m.Reason, &m.ExpiresAt, &m.CreatedAt); err != nil {
			return nil, err
		}
		mutes = append(mutes, m)
	}
	return mutes, nil
}

func (db *DB) AwardBadge(userID string, badgeID int) error {
	query := `INSERT INTO user_badges (user_id, badge_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`
	_, err := db.Pool.Exec(context.Background(), query, userID, badgeID)
	return err
}

func (db *DB) RemoveBadge(userID string, badgeID int) error {
	query := `DELETE FROM user_badges WHERE user_id = $1 AND badge_id = $2`
	_, err := db.Pool.Exec(context.Background(), query, userID, badgeID)
	return err
}

func (db *DB) GetUserBadges(userID string) ([]*Badge, error) {
	query := `
		SELECT ab.id, ab.name, ab.emoji, ab.description
		FROM available_badges ab
		JOIN user_badges ub ON ab.id = ub.badge_id
		WHERE ub.user_id = $1
		ORDER BY ub.awarded_at DESC
	`
	rows, err := db.Pool.Query(context.Background(), query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var badges []*Badge
	for rows.Next() {
		b := &Badge{}
		if err := rows.Scan(&b.ID, &b.Name, &b.Emoji, &b.Description); err != nil {
			return nil, err
		}
		badges = append(badges, b)
	}
	return badges, nil
}

// DepositCoins transfers coins from a user's wallet to their bank.
func (db *DB) DepositCoins(ctx context.Context, guildID, userID string, amount int64) error {
	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// Deduct from wallet
	deductQuery := `
		UPDATE user_economy
		SET coins = GREATEST(0, coins - $1)
		WHERE guild_id = $2 AND user_id = $3 AND coins >= $1
	`
	cmdTag, err := tx.Exec(ctx, deductQuery, amount, guildID, userID)
	if err != nil {
		return err
	}
	if cmdTag.RowsAffected() == 0 {
		return fmt.Errorf("insufficient coins in wallet")
	}

	// Add to bank
	addQuery := `
		UPDATE user_economy
		SET bank = bank + $1
		WHERE guild_id = $2 AND user_id = $3
	`
	_, err = tx.Exec(ctx, addQuery, amount, guildID, userID)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

// WithdrawCoins transfers coins from a user's bank to their wallet.
func (db *DB) WithdrawCoins(ctx context.Context, guildID, userID string, amount int64) error {
	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// Deduct from bank
	deductQuery := `
		UPDATE user_economy
		SET bank = GREATEST(0, bank - $1)
		WHERE guild_id = $2 AND user_id = $3 AND bank >= $1
	`
	cmdTag, err := tx.Exec(ctx, deductQuery, amount, guildID, userID)
	if err != nil {
		return err
	}
	if cmdTag.RowsAffected() == 0 {
		return fmt.Errorf("insufficient coins in bank")
	}

	// Add to wallet
	addQuery := `
		UPDATE user_economy
		SET coins = coins + $1
		WHERE guild_id = $2 AND user_id = $3
	`
	_, err = tx.Exec(ctx, addQuery, amount, guildID, userID)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

// ApplyInterest applies 1% interest to all eligible bank accounts.
func (db *DB) ApplyInterest(ctx context.Context) (int64, error) {
	query := `
		UPDATE user_economy
		SET bank = bank + GREATEST(1, CAST(bank * 0.01 AS BIGINT)),
		    last_interest_at = NOW()
		WHERE bank > 0 AND (last_interest_at IS NULL OR last_interest_at < NOW() - INTERVAL '1 day')
	`
	cmdTag, err := db.Pool.Exec(ctx, query)
	return cmdTag.RowsAffected(), err
}

// UserPet represents a user's pet.
type UserPet struct {
	ID           int
	GuildID      string
	UserID       string
	Name         string
	Type         string
	Hunger       int
	Happiness    int
	LastFedAt    time.Time
	LastPlayedAt time.Time
	CreatedAt    time.Time
}

// AdoptPet adopts a new pet for a user.
func (db *DB) AdoptPet(ctx context.Context, guildID, userID, name, petType string) error {
	query := `
		INSERT INTO user_pets (guild_id, user_id, name, type, hunger, happiness)
		VALUES ($1, $2, $3, $4, 50, 50)
	`
	_, err := db.Pool.Exec(ctx, query, guildID, userID, name, petType)
	return err
}

// GetPet gets a user's pet.
func (db *DB) GetPet(ctx context.Context, guildID, userID string) (*UserPet, error) {
	query := `
		SELECT id, guild_id, user_id, name, type, hunger, happiness, last_fed_at, last_played_at, created_at
		FROM user_pets
		WHERE guild_id = $1 AND user_id = $2
	`
	row := db.Pool.QueryRow(ctx, query, guildID, userID)
	var p UserPet
	err := row.Scan(&p.ID, &p.GuildID, &p.UserID, &p.Name, &p.Type, &p.Hunger, &p.Happiness, &p.LastFedAt, &p.LastPlayedAt, &p.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// FeedPet feeds a user's pet, decreasing hunger and increasing happiness.
func (db *DB) FeedPet(ctx context.Context, guildID, userID string) error {
	query := `
		UPDATE user_pets
		SET hunger = GREATEST(0, hunger - 30),
		    happiness = LEAST(100, happiness + 10),
		    last_fed_at = NOW()
		WHERE guild_id = $1 AND user_id = $2
	`
	_, err := db.Pool.Exec(ctx, query, guildID, userID)
	return err
}

// PlayPet plays with a user's pet, increasing happiness and hunger.
func (db *DB) PlayPet(ctx context.Context, guildID, userID string) error {
	query := `
		UPDATE user_pets
		SET happiness = LEAST(100, happiness + 30),
		    hunger = LEAST(100, hunger + 10),
		    last_played_at = NOW()
		WHERE guild_id = $1 AND user_id = $2
	`
	_, err := db.Pool.Exec(ctx, query, guildID, userID)
	return err
}

// UpdateAllPetStats simulates time passing for all pets by increasing hunger and decreasing happiness.
func (db *DB) UpdateAllPetStats(ctx context.Context) error {
	query := `
		UPDATE user_pets
		SET hunger = LEAST(100, hunger + 5),
		    happiness = GREATEST(0, happiness - 5)
	`
	_, err := db.Pool.Exec(ctx, query)
	return err
}

// Job represents an available job in a guild.
type Job struct {
	ID            int
	GuildID       string
	Name          string
	Description   string
	Salary        int
	RequiredLevel int
}

// CreateJob creates a new job for a guild.
func (db *DB) CreateJob(ctx context.Context, guildID, name, description string, salary, requiredLevel int) error {
	query := `
		INSERT INTO available_jobs (guild_id, name, description, salary, required_level)
		VALUES ($1, $2, $3, $4, $5)
	`
	_, err := db.Pool.Exec(ctx, query, guildID, name, description, salary, requiredLevel)
	return err
}

// GetJobs retrieves all available jobs for a guild.
func (db *DB) GetJobs(ctx context.Context, guildID string) ([]Job, error) {
	query := `
		SELECT id, guild_id, name, description, salary, required_level
		FROM available_jobs
		WHERE guild_id = $1
		ORDER BY required_level ASC
	`
	rows, err := db.Pool.Query(ctx, query, guildID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobs []Job
	for rows.Next() {
		var j Job
		if err := rows.Scan(&j.ID, &j.GuildID, &j.Name, &j.Description, &j.Salary, &j.RequiredLevel); err != nil {
			return nil, err
		}
		jobs = append(jobs, j)
	}

	return jobs, rows.Err()
}

// GetJob retrieves a specific job by ID.
func (db *DB) GetJob(ctx context.Context, jobID int) (*Job, error) {
	query := `
		SELECT id, guild_id, name, description, salary, required_level
		FROM available_jobs
		WHERE id = $1
	`
	row := db.Pool.QueryRow(ctx, query, jobID)
	var j Job
	err := row.Scan(&j.ID, &j.GuildID, &j.Name, &j.Description, &j.Salary, &j.RequiredLevel)
	if err != nil {
		return nil, err
	}
	return &j, nil
}

// SetUserJob assigns a job to a user.
func (db *DB) SetUserJob(ctx context.Context, guildID, userID string, jobID int) error {
	query := `
		UPDATE user_economy
		SET job_id = $3
		WHERE guild_id = $1 AND user_id = $2
	`
	_, err := db.Pool.Exec(ctx, query, guildID, userID, jobID)
	return err
}

// RemoveUserJob removes the user's current job.
func (db *DB) RemoveUserJob(ctx context.Context, guildID, userID string) error {
	query := `
		UPDATE user_economy
		SET job_id = NULL
		WHERE guild_id = $1 AND user_id = $2
	`
	_, err := db.Pool.Exec(ctx, query, guildID, userID)
	return err
}

// AutoThreadConfig represents an auto-thread configuration
type AutoThreadConfig struct {
	GuildID            string `json:"guild_id"`
	ChannelID          string `json:"channel_id"`
	ThreadNameTemplate string `json:"thread_name_template"`
}

// SetAutoThread sets the auto-thread configuration for a channel
func (db *DB) SetAutoThread(ctx context.Context, guildID, channelID, template string) error {
	query := `
		INSERT INTO auto_threads_config (guild_id, channel_id, thread_name_template)
		VALUES ($1, $2, $3)
		ON CONFLICT (guild_id, channel_id) DO UPDATE
		SET thread_name_template = $3
	`
	_, err := db.Pool.Exec(ctx, query, guildID, channelID, template)
	return err
}

// GetAutoThread gets the auto-thread configuration for a channel
func (db *DB) GetAutoThread(ctx context.Context, guildID, channelID string) (*AutoThreadConfig, error) {
	query := `SELECT guild_id, channel_id, thread_name_template FROM auto_threads_config WHERE guild_id = $1 AND channel_id = $2`
	var config AutoThreadConfig
	err := db.Pool.QueryRow(ctx, query, guildID, channelID).Scan(&config.GuildID, &config.ChannelID, &config.ThreadNameTemplate)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &config, nil
}

// RemoveAutoThread removes the auto-thread configuration for a channel
func (db *DB) RemoveAutoThread(ctx context.Context, guildID, channelID string) error {
	query := `DELETE FROM auto_threads_config WHERE guild_id = $1 AND channel_id = $2`
	_, err := db.Pool.Exec(ctx, query, guildID, channelID)
	return err
}

// SetVoiceJoinTime records the time a user joined a voice channel.
func (db *DB) SetVoiceJoinTime(ctx context.Context, guildID, userID string) error {
	query := `
		INSERT INTO voice_xp (guild_id, user_id, join_time)
		VALUES ($1, $2, NOW())
		ON CONFLICT (guild_id, user_id) DO UPDATE SET join_time = NOW()
	`
	_, err := db.Pool.Exec(ctx, query, guildID, userID)
	return err
}

// GetVoiceJoinTime retrieves the time a user joined a voice channel.
func (db *DB) GetVoiceJoinTime(ctx context.Context, guildID, userID string) (*time.Time, error) {
	query := `SELECT join_time FROM voice_xp WHERE guild_id = $1 AND user_id = $2`
	var joinTime time.Time
	err := db.Pool.QueryRow(ctx, query, guildID, userID).Scan(&joinTime)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &joinTime, nil
}

// RemoveVoiceJoinTime deletes the join time record for a user.
func (db *DB) RemoveVoiceJoinTime(ctx context.Context, guildID, userID string) error {
	query := `DELETE FROM voice_xp WHERE guild_id = $1 AND user_id = $2`
	_, err := db.Pool.Exec(ctx, query, guildID, userID)
	return err
}

// Bookmark represents a saved message for a user.
type Bookmark struct {
	ID        int       `json:"id"`
	UserID    string    `json:"user_id"`
	MessageID string    `json:"message_id"`
	ChannelID string    `json:"channel_id"`
	GuildID   string    `json:"guild_id"`
	Note      *string   `json:"note"`
	CreatedAt time.Time `json:"created_at"`
}

// AddBookmark saves a message to the user's bookmarks.
func (db *DB) AddBookmark(ctx context.Context, userID, messageID, channelID, guildID string, note *string) error {
	query := `
		INSERT INTO bookmarks (user_id, message_id, channel_id, guild_id, note)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (user_id, message_id) DO NOTHING
	`
	tag, err := db.Pool.Exec(ctx, query, userID, messageID, channelID, guildID, note)
	if err != nil {
		return fmt.Errorf("failed to add bookmark: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("bookmark already exists")
	}
	return nil
}

// RemoveBookmark removes a saved message from the user's bookmarks.
func (db *DB) RemoveBookmark(ctx context.Context, userID, messageID string) error {
	query := `DELETE FROM bookmarks WHERE user_id = $1 AND message_id = $2`
	tag, err := db.Pool.Exec(ctx, query, userID, messageID)
	if err != nil {
		return fmt.Errorf("failed to remove bookmark: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("bookmark not found")
	}
	return nil
}

// GetBookmarks retrieves a user's bookmarked messages.
func (db *DB) GetBookmarks(ctx context.Context, userID string) ([]Bookmark, error) {
	query := `
		SELECT id, user_id, message_id, channel_id, guild_id, note, created_at
		FROM bookmarks
		WHERE user_id = $1
		ORDER BY created_at DESC
	`
	rows, err := db.Pool.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query bookmarks: %w", err)
	}
	defer rows.Close()

	var bookmarks []Bookmark
	for rows.Next() {
		var b Bookmark
		if err := rows.Scan(&b.ID, &b.UserID, &b.MessageID, &b.ChannelID, &b.GuildID, &b.Note, &b.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan bookmark: %w", err)
		}
		bookmarks = append(bookmarks, b)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}
	return bookmarks, nil
}

// SetUserTimezone updates or inserts a user's timezone
func (db *DB) SetUserTimezone(ctx context.Context, userID, timezone string) error {
	query := `
		INSERT INTO user_timezones (user_id, timezone)
		VALUES ($1, $2)
		ON CONFLICT (user_id) DO UPDATE
		SET timezone = EXCLUDED.timezone;
	`
	_, err := db.Pool.Exec(ctx, query, userID, timezone)
	if err != nil {
		return fmt.Errorf("failed to set user timezone: %w", err)
	}
	return nil
}

// GetUserTimezone retrieves a user's timezone
func (db *DB) GetUserTimezone(ctx context.Context, userID string) (string, error) {
	query := `
		SELECT timezone
		FROM user_timezones
		WHERE user_id = $1
	`
	var timezone string
	err := db.Pool.QueryRow(ctx, query, userID).Scan(&timezone)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", nil // Returns empty string if not found
		}
		return "", fmt.Errorf("failed to get user timezone: %w", err)
	}
	return timezone, nil
}

// SaveUserRoles saves a user's roles for a specific guild.
func (db *DB) SaveUserRoles(ctx context.Context, userID, guildID string, roleIDs []string) error {
	query := `
		INSERT INTO user_roles (user_id, guild_id, role_ids)
		VALUES ($1, $2, $3)
		ON CONFLICT (user_id, guild_id) DO UPDATE
		SET role_ids = EXCLUDED.role_ids
	`
	_, err := db.Pool.Exec(ctx, query, userID, guildID, roleIDs)
	if err != nil {
		return fmt.Errorf("error saving user roles: %w", err)
	}
	return nil
}

// GetUserRoles gets a user's roles for a specific guild.
func (db *DB) GetUserRoles(ctx context.Context, userID, guildID string) ([]string, error) {
	query := `SELECT role_ids FROM user_roles WHERE user_id = $1 AND guild_id = $2`
	var roleIDs []string
	err := db.Pool.QueryRow(ctx, query, userID, guildID).Scan(&roleIDs)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil // No roles saved
		}
		return nil, fmt.Errorf("error getting user roles: %w", err)
	}
	return roleIDs, nil
}

// MusicQueue represents a queued song in a guild's music system.
type MusicQueue struct {
	ID        int
	GuildID   string
	UserID    string
	Query     string
	Position  int
	CreatedAt time.Time
}

// PlayMusic adds a new song query to the end of the guild's music queue.
func (db *DB) PlayMusic(ctx context.Context, guildID, userID, query string) (MusicQueue, error) {
	var q MusicQueue
	err := db.Pool.QueryRow(ctx, `
		INSERT INTO music_queue (guild_id, user_id, query, position)
		VALUES ($1, $2, $3, COALESCE((SELECT MAX(position) FROM music_queue WHERE guild_id = $1), 0) + 1)
		RETURNING id, guild_id, user_id, query, position, created_at
	`, guildID, userID, query).Scan(&q.ID, &q.GuildID, &q.UserID, &q.Query, &q.Position, &q.CreatedAt)
	return q, err
}

// SkipMusic removes the first song from the queue and returns it.
func (db *DB) SkipMusic(ctx context.Context, guildID string) (*MusicQueue, error) {
	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var q MusicQueue
	err = tx.QueryRow(ctx, `
		DELETE FROM music_queue
		WHERE id = (
			SELECT id FROM music_queue
			WHERE guild_id = $1
			ORDER BY position ASC
			LIMIT 1
		)
		RETURNING id, guild_id, user_id, query, position, created_at
	`, guildID).Scan(&q.ID, &q.GuildID, &q.UserID, &q.Query, &q.Position, &q.CreatedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil // Queue is empty
		}
		return nil, err
	}

	err = tx.Commit(ctx)
	return &q, err
}

// StopMusic clears the entire queue for the given guild.
func (db *DB) StopMusic(ctx context.Context, guildID string) error {
	_, err := db.Pool.Exec(ctx, "DELETE FROM music_queue WHERE guild_id = $1", guildID)
	return err
}

// GetQueue retrieves all songs currently in the queue for a given guild, ordered by position.
func (db *DB) GetQueue(ctx context.Context, guildID string) ([]MusicQueue, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, guild_id, user_id, query, position, created_at
		FROM music_queue
		WHERE guild_id = $1
		ORDER BY position ASC
	`, guildID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var queue []MusicQueue
	for rows.Next() {
		var q MusicQueue
		if err := rows.Scan(&q.ID, &q.GuildID, &q.UserID, &q.Query, &q.Position, &q.CreatedAt); err != nil {
			return nil, err
		}
		queue = append(queue, q)
	}
	return queue, rows.Err()
}

// AddFact inserts a new fact for a guild.
func (db *DB) AddFact(ctx context.Context, guildID string, authorID string, text string) (int, error) {
	var id int
	err := db.Pool.QueryRow(ctx,
		"INSERT INTO facts (guild_id, author_id, text) VALUES ($1, $2, $3) RETURNING id",
		guildID, authorID, text,
	).Scan(&id)
	return id, err
}

// GetRandomFact returns a random fact for a guild.
func (db *DB) GetRandomFact(ctx context.Context, guildID string) (*int, *string, *string, error) {
	var id int
	var text, authorID string
	err := db.Pool.QueryRow(ctx,
		"SELECT id, text, author_id FROM facts WHERE guild_id = $1 ORDER BY RANDOM() LIMIT 1",
		guildID,
	).Scan(&id, &text, &authorID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil, nil, nil
		}
		return nil, nil, nil, err
	}
	return &id, &text, &authorID, nil
}

// DeleteFact deletes a fact by ID and guild ID.
func (db *DB) DeleteFact(ctx context.Context, guildID string, factID int) error {
	tag, err := db.Pool.Exec(ctx,
		"DELETE FROM facts WHERE guild_id = $1 AND id = $2",
		guildID, factID,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return errors.New("fact not found or access denied")
	}
	return nil
}

// SetProfileLinks sets the social links for a user's profile
func (db *DB) SetProfileLinks(ctx context.Context, guildID, userID string, website, github, twitter *string) error {
	query := `
		INSERT INTO user_profiles (guild_id, user_id, website, github, twitter, updated_at)
		VALUES ($1, $2, $3, $4, $5, CURRENT_TIMESTAMP)
		ON CONFLICT (guild_id, user_id)
		DO UPDATE SET website = COALESCE(EXCLUDED.website, user_profiles.website),
					  github = COALESCE(EXCLUDED.github, user_profiles.github),
					  twitter = COALESCE(EXCLUDED.twitter, user_profiles.twitter),
					  updated_at = CURRENT_TIMESTAMP
	`
	_, err := db.Pool.Exec(ctx, query, guildID, userID, website, github, twitter)
	if err != nil {
		return fmt.Errorf("failed to set profile links: %w", err)
	}
	return nil
}

// Highlight represents a highlighted message in a server.
type Highlight struct {
	ID        int        `json:"id"`
	GuildID   string     `json:"guild_id"`
	MessageID string     `json:"message_id"`
	ChannelID string     `json:"channel_id"`
	AuthorID  string     `json:"author_id"`
	AddedBy   string     `json:"added_by"`
	CreatedAt *time.Time `json:"created_at"`
}

// AddHighlight adds a new highlight to the database.
func (db *DB) AddHighlight(ctx context.Context, guildID, messageID, channelID, authorID, addedBy string) error {
	start := time.Now()
	metrics.DBQueryLatency.WithLabelValues("AddHighlight").Observe(time.Since(start).Seconds())

	query := `
		INSERT INTO highlights (guild_id, message_id, channel_id, author_id, added_by, created_at)
		VALUES ($1, $2, $3, $4, $5, NOW())
		ON CONFLICT (guild_id, message_id) DO NOTHING
	`
	_, err := db.Pool.Exec(ctx, query, guildID, messageID, channelID, authorID, addedBy)
	return err
}

// GetHighlights retrieves all highlights for a guild, ordered by newest first.
func (db *DB) GetHighlights(ctx context.Context, guildID string) ([]*Highlight, error) {
	start := time.Now()
	metrics.DBQueryLatency.WithLabelValues("GetHighlights").Observe(time.Since(start).Seconds())

	query := `
		SELECT id, guild_id, message_id, channel_id, author_id, added_by, created_at
		FROM highlights
		WHERE guild_id = $1
		ORDER BY created_at DESC
	`
	rows, err := db.Pool.Query(ctx, query, guildID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var highlights []*Highlight
	for rows.Next() {
		var h Highlight
		if err := rows.Scan(&h.ID, &h.GuildID, &h.MessageID, &h.ChannelID, &h.AuthorID, &h.AddedBy, &h.CreatedAt); err != nil {
			return nil, err
		}
		highlights = append(highlights, &h)
	}
	return highlights, nil
}

// RemoveHighlight deletes a highlight by its ID and Guild ID.
func (db *DB) RemoveHighlight(ctx context.Context, id int, guildID string) error {
	start := time.Now()
	metrics.DBQueryLatency.WithLabelValues("RemoveHighlight").Observe(time.Since(start).Seconds())

	query := `
		DELETE FROM highlights
		WHERE id = $1 AND guild_id = $2
	`
	tag, err := db.Pool.Exec(ctx, query, id, guildID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("highlight not found")
	}
	return nil
}

// SetNicknameTemplate updates or inserts a nickname template for a guild.
func (db *DB) SetNicknameTemplate(ctx context.Context, guildID, template string) error {
	query := `
		INSERT INTO nickname_config (guild_id, template)
		VALUES ($1, $2)
		ON CONFLICT (guild_id) DO UPDATE SET
			template = EXCLUDED.template,
			updated_at = CURRENT_TIMESTAMP
	`
	_, err := db.Pool.Exec(ctx, query, guildID, template)
	if err != nil {
		return fmt.Errorf("failed to set nickname template: %w", err)
	}
	return nil
}

// GetNicknameTemplate retrieves the nickname template for a guild.
func (db *DB) GetNicknameTemplate(ctx context.Context, guildID string) (*string, error) {
	query := `SELECT template FROM nickname_config WHERE guild_id = $1`
	var template string
	err := db.Pool.QueryRow(ctx, query, guildID).Scan(&template)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil // Return nil, nil if no template is found
		}
		return nil, fmt.Errorf("failed to get nickname template: %w", err)
	}
	return &template, nil
}

// UserConfig represents the user's advanced configuration settings.
type UserConfig struct {
	UserID          string `json:"user_id"`
	DNDMode         bool   `json:"dnd_mode"`
	DMNotifications bool   `json:"dm_notifications"`
}

// SetUserConfig upserts a user's configuration, preserving existing values for unprovided fields.
func (db *DB) SetUserConfig(ctx context.Context, userID string, dndMode *bool, dmNotifications *bool) error {
	query := `
		INSERT INTO user_config (user_id, dnd_mode, dm_notifications)
		VALUES ($1, COALESCE($2, FALSE), COALESCE($3, TRUE))
		ON CONFLICT (user_id) DO UPDATE SET
			dnd_mode = COALESCE($2, user_config.dnd_mode),
			dm_notifications = COALESCE($3, user_config.dm_notifications)
	`
	_, err := db.Pool.Exec(ctx, query, userID, dndMode, dmNotifications)
	if err != nil {
		return fmt.Errorf("failed to set user config: %w", err)
	}
	return nil
}

// GetUserConfig retrieves a user's configuration, returning default values if none exists.
func (db *DB) GetUserConfig(ctx context.Context, userID string) (*UserConfig, error) {
	query := `SELECT user_id, dnd_mode, dm_notifications FROM user_config WHERE user_id = $1`
	var config UserConfig
	err := db.Pool.QueryRow(ctx, query, userID).Scan(&config.UserID, &config.DNDMode, &config.DMNotifications)
	if err == pgx.ErrNoRows {
		// Return defaults if not found
		return &UserConfig{
			UserID:          userID,
			DNDMode:         false,
			DMNotifications: true,
		}, nil
	} else if err != nil {
		return nil, fmt.Errorf("failed to get user config: %w", err)
	}
	return &config, nil
}

type ActiveBan struct {
	ID        int
	UserID    string
	GuildID   string
	UnbanAt   time.Time
	CreatedAt time.Time
}

func (db *DB) AddTempBan(userID, guildID string, unbanAt time.Time) error {
	query := `
		INSERT INTO active_bans (user_id, guild_id, unban_at)
		VALUES ($1, $2, $3)
		ON CONFLICT (user_id, guild_id) DO UPDATE SET unban_at = EXCLUDED.unban_at
	`
	_, err := db.Pool.Exec(context.Background(), query, userID, guildID, unbanAt)
	return err
}

func (db *DB) GetActiveTempBans() ([]*ActiveBan, error) {
	query := `
		SELECT id, user_id, guild_id, unban_at, created_at
		FROM active_bans
		WHERE unban_at <= NOW()
	`
	rows, err := db.Pool.Query(context.Background(), query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var bans []*ActiveBan
	for rows.Next() {
		var b ActiveBan
		if err := rows.Scan(&b.ID, &b.UserID, &b.GuildID, &b.UnbanAt, &b.CreatedAt); err != nil {
			return nil, err
		}
		bans = append(bans, &b)
	}
	return bans, nil
}

func (db *DB) RemoveTempBan(userID, guildID string) error {
	query := `DELETE FROM active_bans WHERE user_id = $1 AND guild_id = $2`
	_, err := db.Pool.Exec(context.Background(), query, userID, guildID)
	return err
}

func (db *DB) MarkAllUserModActionsResolved(ctx context.Context, guildID, userID, action string) error {
	query := `
		UPDATE mod_actions
		SET resolved = true
		WHERE guild_id = $1 AND user_id = $2 AND action = $3 AND resolved = false
	`
	_, err := db.Pool.Exec(ctx, query, guildID, userID, action)
	return err
}

// SetCommandCooldown sets or updates a user's command cooldown duration.
func (db *DB) SetCommandCooldown(ctx context.Context, userID, command string, duration time.Duration) error {
	query := `
		INSERT INTO command_cooldowns (user_id, command, expires_at)
		VALUES ($1, $2, $3)
		ON CONFLICT (user_id, command)
		DO UPDATE SET expires_at = EXCLUDED.expires_at;
	`
	expiresAt := time.Now().Add(duration)
	_, err := db.Pool.Exec(ctx, query, userID, command, expiresAt)
	return err
}

// GetCommandCooldown retrieves a user's current command cooldown expiration time, if any.
// If the user has no active cooldown for the command, it returns a zero time.Time and nil error.
func (db *DB) GetCommandCooldown(ctx context.Context, userID, command string) (time.Time, error) {
	query := `
		SELECT expires_at FROM command_cooldowns
		WHERE user_id = $1 AND command = $2;
	`
	var expiresAt time.Time
	err := db.Pool.QueryRow(ctx, query, userID, command).Scan(&expiresAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return time.Time{}, nil // Not on cooldown
		}
		return time.Time{}, err
	}
	return expiresAt, nil
}

// AntiSpamConfig represents a guild's anti-spam configuration.
type AntiSpamConfig struct {
	GuildID      string
	MessageLimit int
	TimeWindow   int
	MuteDuration string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// SetAntiSpamConfig sets or updates the anti-spam configuration for a guild.
func (db *DB) SetAntiSpamConfig(ctx context.Context, guildID string, messageLimit, timeWindow int, muteDuration string) error {
	query := `
		INSERT INTO anti_spam_config (guild_id, message_limit, time_window, mute_duration, updated_at)
		VALUES ($1, $2, $3, $4, NOW())
		ON CONFLICT (guild_id) DO UPDATE
		SET message_limit = EXCLUDED.message_limit,
		    time_window = EXCLUDED.time_window,
		    mute_duration = EXCLUDED.mute_duration,
		    updated_at = NOW()
	`
	_, err := db.Pool.Exec(ctx, query, guildID, messageLimit, timeWindow, muteDuration)
	return err
}

// GetAntiSpamConfig gets the anti-spam configuration for a guild.
func (db *DB) GetAntiSpamConfig(ctx context.Context, guildID string) (*AntiSpamConfig, error) {
	query := `
		SELECT guild_id, message_limit, time_window, mute_duration, created_at, updated_at
		FROM anti_spam_config
		WHERE guild_id = $1
	`
	c := &AntiSpamConfig{}
	err := db.Pool.QueryRow(ctx, query, guildID).Scan(
		&c.GuildID, &c.MessageLimit, &c.TimeWindow, &c.MuteDuration, &c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil // Return nil, nil if no config exists
		}
		return nil, err
	}
	return c, nil
}

// AdvancedLogConfig represents the configuration for advanced event logging in a guild.
type AdvancedLogConfig struct {
	GuildID   string
	Events    string
	ChannelID string
}

// SetAdvancedLogConfig sets or updates the advanced logging configuration for a guild.
func (db *DB) SetAdvancedLogConfig(ctx context.Context, guildID, events, channelID string) error {
	query := `
		INSERT INTO advanced_log_config (guild_id, events, channel_id)
		VALUES ($1, $2, $3)
		ON CONFLICT (guild_id)
		DO UPDATE SET events = EXCLUDED.events, channel_id = EXCLUDED.channel_id
	`
	_, err := db.Pool.Exec(ctx, query, guildID, events, channelID)
	return err
}

// GetAdvancedLogConfig retrieves the advanced logging configuration for a guild.
func (db *DB) GetAdvancedLogConfig(ctx context.Context, guildID string) (*AdvancedLogConfig, error) {
	query := `SELECT guild_id, events, channel_id FROM advanced_log_config WHERE guild_id = $1`
	var config AdvancedLogConfig
	err := db.Pool.QueryRow(ctx, query, guildID).Scan(&config.GuildID, &config.Events, &config.ChannelID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &config, nil
}

// SetDynamicVoiceConfig updates the dynamic voice config for a guild.
func (db *DB) SetDynamicVoiceConfig(ctx context.Context, guildID, categoryID, triggerChannelID string) error {
	query := `
		INSERT INTO dynamic_voice_config (guild_id, category_id, trigger_channel_id)
		VALUES ($1, $2, $3)
		ON CONFLICT (guild_id) DO UPDATE SET category_id = EXCLUDED.category_id, trigger_channel_id = EXCLUDED.trigger_channel_id
	`
	_, err := db.Pool.Exec(ctx, query, guildID, categoryID, triggerChannelID)
	return err
}

// DynamicVoiceConfig represents the configuration for dynamic voice channels.
type DynamicVoiceConfig struct {
	GuildID          string `json:"guild_id"`
	CategoryID       string `json:"category_id"`
	TriggerChannelID string `json:"trigger_channel_id"`
}

// GetDynamicVoiceConfig gets the dynamic voice config for a guild.
func (db *DB) GetDynamicVoiceConfig(ctx context.Context, guildID string) (*DynamicVoiceConfig, error) {
	query := `SELECT guild_id, category_id, trigger_channel_id FROM dynamic_voice_config WHERE guild_id = $1`
	var config DynamicVoiceConfig
	err := db.Pool.QueryRow(ctx, query, guildID).Scan(&config.GuildID, &config.CategoryID, &config.TriggerChannelID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil // No config set
		}
		return nil, err
	}
	return &config, nil
}

// ReactionRoleGroup represents a group of reaction roles
type ReactionRoleGroup struct {
	ID          int
	GuildID     string
	Name        string
	IsExclusive bool
	MaxRoles    int
}

// CreateReactionRoleGroup creates a new reaction role group
func (db *DB) CreateReactionRoleGroup(ctx context.Context, group *ReactionRoleGroup) error {
	query := `
		INSERT INTO reaction_role_groups (guild_id, name, is_exclusive, max_roles)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`
	err := db.Pool.QueryRow(ctx, query, group.GuildID, group.Name, group.IsExclusive, group.MaxRoles).Scan(&group.ID)
	if err != nil {
		return err
	}
	return nil
}

// GetReactionRoleGroups gets all reaction role groups for a guild
func (db *DB) GetReactionRoleGroups(ctx context.Context, guildID string) ([]ReactionRoleGroup, error) {
	query := `
		SELECT id, guild_id, name, is_exclusive, max_roles
		FROM reaction_role_groups
		WHERE guild_id = $1
	`
	rows, err := db.Pool.Query(ctx, query, guildID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var groups []ReactionRoleGroup
	for rows.Next() {
		var g ReactionRoleGroup
		if err := rows.Scan(&g.ID, &g.GuildID, &g.Name, &g.IsExclusive, &g.MaxRoles); err != nil {
			return nil, err
		}
		groups = append(groups, g)
	}
	return groups, nil
}

// AssignRoleToGroup assigns a reaction role to a group
func (db *DB) AssignRoleToGroup(ctx context.Context, messageID, emoji, emojiID string, groupID int) error {
	query := `
		UPDATE reaction_roles
		SET group_id = $1
		WHERE message_id = $2 AND emoji = $3 AND emoji_id = $4
	`
	_, err := db.Pool.Exec(ctx, query, groupID, messageID, emoji, emojiID)
	return err
}

// GetGroupRoles gets all role IDs that belong to a specific reaction role group
func (db *DB) GetGroupRoles(ctx context.Context, groupID int) ([]string, error) {
	query := `
		SELECT role_id
		FROM reaction_roles
		WHERE group_id = $1
	`
	rows, err := db.Pool.Query(ctx, query, groupID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var roleIDs []string
	for rows.Next() {
		var roleID string
		if err := rows.Scan(&roleID); err != nil {
			return nil, err
		}
		roleIDs = append(roleIDs, roleID)
	}
	return roleIDs, nil
}

// GetReactionRoleGroup gets a reaction role group by ID
func (db *DB) GetReactionRoleGroup(ctx context.Context, id int) (*ReactionRoleGroup, error) {
	query := `
		SELECT id, guild_id, name, is_exclusive, max_roles
		FROM reaction_role_groups
		WHERE id = $1
	`
	var g ReactionRoleGroup
	err := db.Pool.QueryRow(ctx, query, id).Scan(&g.ID, &g.GuildID, &g.Name, &g.IsExclusive, &g.MaxRoles)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &g, nil
}

func (db *DB) SaveTicketTranscript(ctx context.Context, ticketID int, channelID, guildID, userID, transcriptURL string) error {
	query := `
		INSERT INTO ticket_transcripts (ticket_id, channel_id, guild_id, user_id, transcript_url)
		VALUES ($1, $2, $3, $4, $5)
	`
	_, err := db.Pool.Exec(ctx, query, ticketID, channelID, guildID, userID, transcriptURL)
	return err
}

func (db *DB) GetTicketTranscripts(ctx context.Context, guildID, userID string) ([]TicketTranscript, error) {
	query := `
		SELECT id, ticket_id, channel_id, guild_id, user_id, transcript_url, created_at
		FROM ticket_transcripts
		WHERE guild_id = $1 AND user_id = $2
		ORDER BY created_at DESC
	`
	rows, err := db.Pool.Query(ctx, query, guildID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var transcripts []TicketTranscript
	for rows.Next() {
		var t TicketTranscript
		if err := rows.Scan(&t.ID, &t.TicketID, &t.ChannelID, &t.GuildID, &t.UserID, &t.TranscriptURL, &t.CreatedAt); err != nil {
			return nil, err
		}
		transcripts = append(transcripts, t)
	}
	return transcripts, nil
}

type ThreadConfig struct {
	GuildID             string
	AutoArchiveDuration int
}

func (db *DB) SetThreadConfig(ctx context.Context, guildID string, duration int) error {
	query := `
		INSERT INTO thread_config (guild_id, auto_archive_duration)
		VALUES ($1, $2)
		ON CONFLICT (guild_id) DO UPDATE
		SET auto_archive_duration = EXCLUDED.auto_archive_duration;
	`
	_, err := db.Pool.Exec(ctx, query, guildID, duration)
	return err
}

func (db *DB) GetThreadConfig(ctx context.Context, guildID string) (*ThreadConfig, error) {
	query := `SELECT guild_id, auto_archive_duration FROM thread_config WHERE guild_id = $1`
	var config ThreadConfig
	err := db.Pool.QueryRow(ctx, query, guildID).Scan(&config.GuildID, &config.AutoArchiveDuration)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &config, nil
}

// AddAutoPublishChannel adds a channel to the auto-publish configuration for a guild.
func (db *DB) AddAutoPublishChannel(ctx context.Context, guildID, channelID string) error {
	start := time.Now()
	query := `
		INSERT INTO auto_publish_config (guild_id, channel_id)
		VALUES ($1, $2)
		ON CONFLICT (guild_id, channel_id) DO NOTHING
	`
	_, err := db.Pool.Exec(ctx, query, guildID, channelID)
	if err != nil {
		metrics.ErrorCounter.WithLabelValues("db_query").Inc()
		return err
	}
	metrics.DBQueryLatency.WithLabelValues("AddAutoPublishChannel").Observe(time.Since(start).Seconds())
	return nil
}

// RemoveAutoPublishChannel removes a channel from the auto-publish configuration for a guild.
func (db *DB) RemoveAutoPublishChannel(ctx context.Context, guildID, channelID string) error {
	start := time.Now()
	query := `
		DELETE FROM auto_publish_config
		WHERE guild_id = $1 AND channel_id = $2
	`
	_, err := db.Pool.Exec(ctx, query, guildID, channelID)
	if err != nil {
		metrics.ErrorCounter.WithLabelValues("db_query").Inc()
		return err
	}
	metrics.DBQueryLatency.WithLabelValues("RemoveAutoPublishChannel").Observe(time.Since(start).Seconds())
	return nil
}

// IsAutoPublishChannel checks if a channel is configured for auto-publishing.
func (db *DB) IsAutoPublishChannel(ctx context.Context, guildID, channelID string) (bool, error) {
	start := time.Now()
	query := `
		SELECT EXISTS(
			SELECT 1 FROM auto_publish_config
			WHERE guild_id = $1 AND channel_id = $2
		)
	`
	var exists bool
	err := db.Pool.QueryRow(ctx, query, guildID, channelID).Scan(&exists)
	if err != nil {
		metrics.ErrorCounter.WithLabelValues("db_query").Inc()
		return false, err
	}
	metrics.DBQueryLatency.WithLabelValues("IsAutoPublishChannel").Observe(time.Since(start).Seconds())
	return exists, nil
}

// SaveStickyRole saves a sticky role for a user.
func (db *DB) SaveStickyRole(ctx context.Context, guildID, userID, roleID string) error {
	query := `
		INSERT INTO sticky_roles (guild_id, user_id, role_id)
		VALUES ($1, $2, $3)
		ON CONFLICT (guild_id, user_id, role_id) DO NOTHING
	`
	_, err := db.Pool.Exec(ctx, query, guildID, userID, roleID)
	return err
}

// GetStickyRoles returns all sticky roles for a user.
func (db *DB) GetStickyRoles(ctx context.Context, guildID, userID string) ([]string, error) {
	query := `
		SELECT role_id FROM sticky_roles
		WHERE guild_id = $1 AND user_id = $2
	`
	rows, err := db.Pool.Query(ctx, query, guildID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var roles []string
	for rows.Next() {
		var roleID string
		if err := rows.Scan(&roleID); err != nil {
			return nil, err
		}
		roles = append(roles, roleID)
	}
	return roles, rows.Err()
}

// RemoveStickyRole removes a sticky role for a user.
func (db *DB) RemoveStickyRole(ctx context.Context, guildID, userID, roleID string) error {
	query := `
		DELETE FROM sticky_roles
		WHERE guild_id = $1 AND user_id = $2 AND role_id = $3
	`
	_, err := db.Pool.Exec(ctx, query, guildID, userID, roleID)
	return err
}

// ReactionMenu represents a reaction role menu.
type ReactionMenu struct {
	MessageID string
	GuildID   string
	ChannelID string
}

// ReactionMenuItem represents an item in a reaction role menu.
type ReactionMenuItem struct {
	MessageID string
	Emoji     string
	RoleID    string
}

// CreateReactionMenu saves a new reaction menu to the database.
func (db *DB) CreateReactionMenu(ctx context.Context, messageID, guildID, channelID string) error {
	start := time.Now()
	metrics.DBQueryLatency.WithLabelValues("CreateReactionMenu").Observe(time.Since(start).Seconds())

	query := `
		INSERT INTO reaction_menus (message_id, guild_id, channel_id)
		VALUES ($1, $2, $3)
		ON CONFLICT (message_id) DO NOTHING
	`
	_, err := db.Pool.Exec(ctx, query, messageID, guildID, channelID)
	if err != nil {
		metrics.ErrorCounter.WithLabelValues("db_query").Inc()
		slog.Error("Failed to create reaction menu", "error", err)
	}
	return err
}

// AddReactionMenuItem adds a new item to a reaction menu.
func (db *DB) AddReactionMenuItem(ctx context.Context, messageID, emoji, roleID string) error {
	start := time.Now()
	metrics.DBQueryLatency.WithLabelValues("AddReactionMenuItem").Observe(time.Since(start).Seconds())

	query := `
		INSERT INTO reaction_menu_items (message_id, emoji, role_id)
		VALUES ($1, $2, $3)
		ON CONFLICT (message_id, emoji) DO UPDATE SET role_id = EXCLUDED.role_id
	`
	_, err := db.Pool.Exec(ctx, query, messageID, emoji, roleID)
	if err != nil {
		metrics.ErrorCounter.WithLabelValues("db_query").Inc()
		slog.Error("Failed to add reaction menu item", "error", err)
	}
	return err
}

// GetReactionMenuItems retrieves all items for a reaction menu.
func (db *DB) GetReactionMenuItems(ctx context.Context, messageID string) ([]*ReactionMenuItem, error) {
	start := time.Now()
	metrics.DBQueryLatency.WithLabelValues("GetReactionMenuItems").Observe(time.Since(start).Seconds())

	query := `
		SELECT message_id, emoji, role_id
		FROM reaction_menu_items
		WHERE message_id = $1
	`
	rows, err := db.Pool.Query(ctx, query, messageID)
	if err != nil {
		metrics.ErrorCounter.WithLabelValues("db_query").Inc()
		slog.Error("Failed to get reaction menu items", "error", err)
		return nil, err
	}
	defer rows.Close()

	var items []*ReactionMenuItem
	for rows.Next() {
		item := &ReactionMenuItem{}
		if err := rows.Scan(&item.MessageID, &item.Emoji, &item.RoleID); err != nil {
			slog.Error("Failed to scan reaction menu item", "error", err)
			continue
		}
		items = append(items, item)
	}
	return items, nil
}

// GetReactionMenu retrieves a reaction menu by its message ID.
func (db *DB) GetReactionMenu(ctx context.Context, messageID string) (*ReactionMenu, error) {
	start := time.Now()
	metrics.DBQueryLatency.WithLabelValues("GetReactionMenu").Observe(time.Since(start).Seconds())

	query := `
		SELECT message_id, guild_id, channel_id
		FROM reaction_menus
		WHERE message_id = $1
	`
	menu := &ReactionMenu{}
	err := db.Pool.QueryRow(ctx, query, messageID).Scan(&menu.MessageID, &menu.GuildID, &menu.ChannelID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		metrics.ErrorCounter.WithLabelValues("db_query").Inc()
		slog.Error("Failed to get reaction menu", "error", err)
		return nil, err
	}
	return menu, nil
}

// SetWelcomeMessage configures or updates the welcome message and channel for a guild.
func (db *DB) SetWelcomeMessage(ctx context.Context, guildID, channelID, message string) error {
	query := `
		INSERT INTO welcome_messages (guild_id, channel_id, message)
		VALUES ($1, $2, $3)
		ON CONFLICT (guild_id) DO UPDATE
		SET channel_id = EXCLUDED.channel_id, message = EXCLUDED.message
	`
	_, err := db.Pool.Exec(ctx, query, guildID, channelID, message)
	if err != nil {
		return fmt.Errorf("failed to set welcome message: %w", err)
	}
	return nil
}

// SetGoodbyeMessage sets or updates the goodbye channel and message for a guild.
func (db *DB) SetGoodbyeMessage(ctx context.Context, guildID, channelID, message string) error {
	query := `
		INSERT INTO goodbye_messages (guild_id, channel_id, message)
		VALUES ($1, $2, $3)
		ON CONFLICT (guild_id) DO UPDATE
		SET channel_id = EXCLUDED.channel_id, message = EXCLUDED.message
	`
	_, err := db.Pool.Exec(ctx, query, guildID, channelID, message)
	if err != nil {
		return fmt.Errorf("failed to set goodbye message: %w", err)
	}
	return nil
}

// GetGoodbyeMessage retrieves the configured goodbye channel ID and message for a guild.
func (db *DB) GetGoodbyeMessage(ctx context.Context, guildID string) (string, string, error) {
	query := `SELECT channel_id, message FROM goodbye_messages WHERE guild_id = $1`
	var channelID, message string
	err := db.Pool.QueryRow(ctx, query, guildID).Scan(&channelID, &message)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", "", nil
		}
		return "", "", fmt.Errorf("failed to get goodbye message: %w", err)
	}
	return channelID, message, nil
}

// RemoveGoodbyeMessage removes the configured goodbye message for a guild.
func (db *DB) RemoveGoodbyeMessage(ctx context.Context, guildID string) error {
	query := `DELETE FROM goodbye_messages WHERE guild_id = $1`
	_, err := db.Pool.Exec(ctx, query, guildID)
	if err != nil {
		return fmt.Errorf("failed to remove goodbye message: %w", err)
	}
	return nil
}

// GetWelcomeMessage retrieves the configured welcome channel ID and message for a guild.
func (db *DB) GetWelcomeMessage(ctx context.Context, guildID string) (string, string, error) {
	query := `SELECT channel_id, message FROM welcome_messages WHERE guild_id = $1`
	var channelID, message string
	err := db.Pool.QueryRow(ctx, query, guildID).Scan(&channelID, &message)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", "", nil
		}
		return "", "", fmt.Errorf("failed to get welcome message: %w", err)
	}
	return channelID, message, nil
}

// RemoveWelcomeMessage disables the welcome message for a guild.
func (db *DB) RemoveWelcomeMessage(ctx context.Context, guildID string) error {
	query := `DELETE FROM welcome_messages WHERE guild_id = $1`
	_, err := db.Pool.Exec(ctx, query, guildID)
	if err != nil {
		return fmt.Errorf("failed to remove welcome message: %w", err)
	}
	return nil
}

// SetMemberCountChannel sets or updates the member count channel config for a guild.
func (db *DB) SetMemberCountChannel(ctx context.Context, guildID, channelID string) error {
	query := `
		INSERT INTO member_count_config (guild_id, channel_id)
		VALUES ($1, $2)
		ON CONFLICT (guild_id) DO UPDATE SET channel_id = EXCLUDED.channel_id
	`
	_, err := db.Pool.Exec(ctx, query, guildID, channelID)
	return err
}

// GetMemberCountChannel retrieves the member count channel config for a guild.
func (db *DB) GetMemberCountChannel(ctx context.Context, guildID string) (string, error) {
	query := `SELECT channel_id FROM member_count_config WHERE guild_id = $1`
	var channelID string
	err := db.Pool.QueryRow(ctx, query, guildID).Scan(&channelID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", nil
		}
		return "", err
	}
	return channelID, nil
}

// RemoveMemberCountChannel removes the member count channel config for a guild.
func (db *DB) RemoveMemberCountChannel(ctx context.Context, guildID string) error {
	query := `DELETE FROM member_count_config WHERE guild_id = $1`
	_, err := db.Pool.Exec(ctx, query, guildID)
	return err
}

// MemberCountConfig represents a member count channel config.
type MemberCountConfig struct {
	GuildID   string
	ChannelID string
}

// GetAllMemberCountChannels retrieves all member count channel configs.
func (db *DB) GetAllMemberCountChannels(ctx context.Context) ([]MemberCountConfig, error) {
	query := `SELECT guild_id, channel_id FROM member_count_config`
	rows, err := db.Pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var configs []MemberCountConfig
	for rows.Next() {
		var c MemberCountConfig
		if err := rows.Scan(&c.GuildID, &c.ChannelID); err != nil {
			return nil, err
		}
		configs = append(configs, c)
	}
	return configs, nil
}

// TempRole represents a temporary role assigned to a user.
type TempRole struct {
	ID        int
	GuildID   string
	UserID    string
	RoleID    string
	ExpiresAt time.Time
}

// AddTempRole adds a new temporary role assignment or updates an existing one.
func (db *DB) AddTempRole(ctx context.Context, guildID, userID, roleID string, expiresAt time.Time) error {
	query := `
		INSERT INTO temp_roles (guild_id, user_id, role_id, expires_at)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (guild_id, user_id, role_id) DO UPDATE
		SET expires_at = EXCLUDED.expires_at
	`
	_, err := db.Pool.Exec(ctx, query, guildID, userID, roleID, expiresAt)
	return err
}

// GetExpiredTempRoles retrieves temporary roles that have expired.
func (db *DB) GetExpiredTempRoles(ctx context.Context) ([]TempRole, error) {
	query := `
		SELECT id, guild_id, user_id, role_id, expires_at
		FROM temp_roles
		WHERE expires_at <= NOW()
	`
	rows, err := db.Pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var roles []TempRole
	for rows.Next() {
		var r TempRole
		if err := rows.Scan(&r.ID, &r.GuildID, &r.UserID, &r.RoleID, &r.ExpiresAt); err != nil {
			return nil, err
		}
		roles = append(roles, r)
	}

	return roles, rows.Err()
}

// RemoveTempRole removes a temporary role by its ID.
func (db *DB) RemoveTempRole(ctx context.Context, id int) error {
	query := `DELETE FROM temp_roles WHERE id = $1`
	_, err := db.Pool.Exec(ctx, query, id)
	return err
}

// RemoveTempRoleByGuildUserRole removes a temporary role matching the guild, user, and role.
func (db *DB) RemoveTempRoleByGuildUserRole(ctx context.Context, guildID, userID, roleID string) error {
	query := `DELETE FROM temp_roles WHERE guild_id = $1 AND user_id = $2 AND role_id = $3`
	_, err := db.Pool.Exec(ctx, query, guildID, userID, roleID)
	return err
}

// SetWelcomeDM inserts or updates the welcome DM message for a guild and enables it.
func (db *DB) SetWelcomeDM(ctx context.Context, guildID, message string) error {
	start := time.Now()
	defer func() {
		metrics.DBQueryLatency.WithLabelValues("SetWelcomeDM").Observe(time.Since(start).Seconds())
	}()

	query := `
		INSERT INTO welcome_dm_config (guild_id, message, is_enabled)
		VALUES ($1, $2, true)
		ON CONFLICT (guild_id) DO UPDATE SET message = EXCLUDED.message, is_enabled = true
	`
	_, err := db.Pool.Exec(ctx, query, guildID, message)
	if err != nil {
		slog.Error("Failed to set welcome DM message", "error", err, "guild_id", guildID)
		return err
	}
	return nil
}

// GetWelcomeDM retrieves the welcome DM message and enabled status for a guild.
func (db *DB) GetWelcomeDM(ctx context.Context, guildID string) (string, bool, error) {
	start := time.Now()
	defer func() {
		metrics.DBQueryLatency.WithLabelValues("GetWelcomeDM").Observe(time.Since(start).Seconds())
	}()

	var message string
	var isEnabled bool
	query := `SELECT message, is_enabled FROM welcome_dm_config WHERE guild_id = $1`
	err := db.Pool.QueryRow(ctx, query, guildID).Scan(&message, &isEnabled)
	if err != nil {
		if err == pgx.ErrNoRows {
			return "", false, nil
		}
		slog.Error("Failed to get welcome DM message", "error", err, "guild_id", guildID)
		return "", false, err
	}
	return message, isEnabled, nil
}

// ToggleWelcomeDM enables or disables the welcome DM for a guild.
func (db *DB) ToggleWelcomeDM(ctx context.Context, guildID string, enabled bool) error {
	start := time.Now()
	defer func() {
		metrics.DBQueryLatency.WithLabelValues("ToggleWelcomeDM").Observe(time.Since(start).Seconds())
	}()

	query := `
		UPDATE welcome_dm_config SET is_enabled = $2
		WHERE guild_id = $1
	`
	_, err := db.Pool.Exec(ctx, query, guildID, enabled)
	if err != nil {
		slog.Error("Failed to toggle welcome DM", "error", err, "guild_id", guildID, "enabled", enabled)
		return err
	}
	return nil
}

// AddLevelingChannelBlacklist adds a channel to the leveling blacklist for a guild.
func (db *DB) AddLevelingChannelBlacklist(ctx context.Context, guildID, channelID string) error {
	start := time.Now()
	defer func() {
		metrics.DBQueryLatency.WithLabelValues("AddLevelingChannelBlacklist").Observe(time.Since(start).Seconds())
	}()

	query := `
		INSERT INTO leveling_channel_blacklist (guild_id, channel_id)
		VALUES ($1, $2)
		ON CONFLICT (guild_id, channel_id) DO NOTHING
	`
	_, err := db.Pool.Exec(ctx, query, guildID, channelID)
	if err != nil {
		metrics.ErrorCounter.WithLabelValues("db_query").Inc()
	}
	return err
}

// RemoveLevelingChannelBlacklist removes a channel from the leveling blacklist for a guild.
func (db *DB) RemoveLevelingChannelBlacklist(ctx context.Context, guildID, channelID string) error {
	start := time.Now()
	defer func() {
		metrics.DBQueryLatency.WithLabelValues("RemoveLevelingChannelBlacklist").Observe(time.Since(start).Seconds())
	}()

	query := `
		DELETE FROM leveling_channel_blacklist
		WHERE guild_id = $1 AND channel_id = $2
	`
	_, err := db.Pool.Exec(ctx, query, guildID, channelID)
	if err != nil {
		metrics.ErrorCounter.WithLabelValues("db_query").Inc()
	}
	return err
}

// AddLevelMultiplier sets a multiplier for a role in a guild.
func (db *DB) AddLevelMultiplier(ctx context.Context, guildID, roleID string, multiplier float64) error {
	start := time.Now()
	defer func() {
		metrics.DBQueryLatency.WithLabelValues("AddLevelMultiplier").Observe(time.Since(start).Seconds())
	}()

	query := `
		INSERT INTO leveling_multipliers (guild_id, role_id, multiplier)
		VALUES ($1, $2, $3)
		ON CONFLICT (guild_id, role_id) DO UPDATE
		SET multiplier = EXCLUDED.multiplier
	`
	_, err := db.Pool.Exec(ctx, query, guildID, roleID, multiplier)
	if err != nil {
		metrics.ErrorCounter.WithLabelValues("db_query").Inc()
	}
	return err
}

// RemoveLevelMultiplier removes a multiplier for a role in a guild.
func (db *DB) RemoveLevelMultiplier(ctx context.Context, guildID, roleID string) error {
	start := time.Now()
	defer func() {
		metrics.DBQueryLatency.WithLabelValues("RemoveLevelMultiplier").Observe(time.Since(start).Seconds())
	}()

	query := `
		DELETE FROM leveling_multipliers
		WHERE guild_id = $1 AND role_id = $2
	`
	_, err := db.Pool.Exec(ctx, query, guildID, roleID)
	if err != nil {
		metrics.ErrorCounter.WithLabelValues("db_query").Inc()
	}
	return err
}

// GetLevelMultipliers retrieves all multipliers for a guild.
func (db *DB) GetLevelMultipliers(ctx context.Context, guildID string) (map[string]float64, error) {
	start := time.Now()
	defer func() {
		metrics.DBQueryLatency.WithLabelValues("GetLevelMultipliers").Observe(time.Since(start).Seconds())
	}()

	query := `
		SELECT role_id, multiplier
		FROM leveling_multipliers
		WHERE guild_id = $1
	`
	rows, err := db.Pool.Query(ctx, query, guildID)
	if err != nil {
		metrics.ErrorCounter.WithLabelValues("db_query").Inc()
		return nil, err
	}
	defer rows.Close()

	multipliers := make(map[string]float64)
	for rows.Next() {
		var roleID string
		var multiplier float64
		if err := rows.Scan(&roleID, &multiplier); err != nil {
			metrics.ErrorCounter.WithLabelValues("db_query").Inc()
			return nil, err
		}
		multipliers[roleID] = multiplier
	}
	return multipliers, nil
}

// GetLevelingChannelBlacklists retrieves all blacklisted channels for a guild.
func (db *DB) GetLevelingChannelBlacklists(ctx context.Context, guildID string) ([]string, error) {
	start := time.Now()
	defer func() {
		metrics.DBQueryLatency.WithLabelValues("GetLevelingChannelBlacklists").Observe(time.Since(start).Seconds())
	}()

	query := `
		SELECT channel_id
		FROM leveling_channel_blacklist
		WHERE guild_id = $1
	`
	rows, err := db.Pool.Query(ctx, query, guildID)
	if err != nil {
		metrics.ErrorCounter.WithLabelValues("db_query").Inc()
		return nil, err
	}
	defer rows.Close()

	var blacklists []string
	for rows.Next() {
		var channelID string
		if err := rows.Scan(&channelID); err != nil {
			metrics.ErrorCounter.WithLabelValues("db_query").Inc()
			return nil, err
		}
		blacklists = append(blacklists, channelID)
	}

	if err := rows.Err(); err != nil {
		metrics.ErrorCounter.WithLabelValues("db_query").Inc()
		return nil, err
	}

	return blacklists, nil
}

// AddWarnAutomationRule adds or updates an automated warning punishment rule for a guild
func (db *DB) AddWarnAutomationRule(ctx context.Context, guildID string, threshold int, action string, duration *string) error {
	query := `
		INSERT INTO warn_automation_config (guild_id, warning_threshold, action, duration)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (guild_id, warning_threshold)
		DO UPDATE SET action = EXCLUDED.action, duration = EXCLUDED.duration
	`
	_, err := db.Pool.Exec(ctx, query, guildID, threshold, action, duration)
	return err
}

// RemoveWarnAutomationRule removes an automated warning punishment rule
func (db *DB) RemoveWarnAutomationRule(ctx context.Context, guildID string, threshold int) error {
	query := `DELETE FROM warn_automation_config WHERE guild_id = $1 AND warning_threshold = $2`
	_, err := db.Pool.Exec(ctx, query, guildID, threshold)
	return err
}

// GetWarnAutomationRules retrieves all automated warning rules for a given guild
func (db *DB) GetWarnAutomationRules(ctx context.Context, guildID string) ([]WarnAutomationRule, error) {
	query := `SELECT id, guild_id, warning_threshold, action, duration FROM warn_automation_config WHERE guild_id = $1 ORDER BY warning_threshold ASC`
	rows, err := db.Pool.Query(ctx, query, guildID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rules []WarnAutomationRule
	for rows.Next() {
		var rule WarnAutomationRule
		if err := rows.Scan(&rule.ID, &rule.GuildID, &rule.WarningThreshold, &rule.Action, &rule.Duration); err != nil {
			return nil, err
		}
		rules = append(rules, rule)
	}

	return rules, nil
}

// AddSnippet saves a message snippet
func (db *DB) AddSnippet(ctx context.Context, guildID, name, content string) error {
	query := `
		INSERT INTO message_snippets (guild_id, name, content)
		VALUES ($1, $2, $3)
		ON CONFLICT (guild_id, name) DO UPDATE SET content = EXCLUDED.content
	`
	_, err := db.Pool.Exec(ctx, query, guildID, name, content)
	return err
}

// RemoveSnippet deletes a message snippet
func (db *DB) RemoveSnippet(ctx context.Context, guildID, name string) error {
	query := `DELETE FROM message_snippets WHERE guild_id = $1 AND name = $2`
	_, err := db.Pool.Exec(ctx, query, guildID, name)
	return err
}

// GetSnippet retrieves a message snippet by name
func (db *DB) GetSnippet(ctx context.Context, guildID, name string) (string, error) {
	query := `SELECT content FROM message_snippets WHERE guild_id = $1 AND name = $2`
	var content string
	err := db.Pool.QueryRow(ctx, query, guildID, name).Scan(&content)
	if err == pgx.ErrNoRows {
		return "", nil
	}
	return content, err
}

// ListSnippets retrieves all message snippets for a guild
func (db *DB) ListSnippets(ctx context.Context, guildID string) ([]string, error) {
	query := `SELECT name FROM message_snippets WHERE guild_id = $1 ORDER BY name`
	rows, err := db.Pool.Query(ctx, query, guildID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var snippets []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		snippets = append(snippets, name)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return snippets, nil
}

// SetTranslationConfig sets the default translation language for a guild.
func (db *DB) SetTranslationConfig(ctx context.Context, guildID, language string) error {
	query := `
		INSERT INTO translation_config (guild_id, default_language)
		VALUES ($1, $2)
		ON CONFLICT (guild_id) DO UPDATE
		SET default_language = EXCLUDED.default_language
	`
	_, err := db.Pool.Exec(ctx, query, guildID, language)
	if err != nil {
		metrics.ErrorCounter.WithLabelValues("db_query").Inc()
		return fmt.Errorf("error setting translation config: %w", err)
	}
	return nil
}

// GetTranslationConfig gets the default translation language for a guild.
func (db *DB) GetTranslationConfig(ctx context.Context, guildID string) (string, error) {
	query := `SELECT default_language FROM translation_config WHERE guild_id = $1`
	var language string
	err := db.Pool.QueryRow(ctx, query, guildID).Scan(&language)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", nil
		}
		metrics.ErrorCounter.WithLabelValues("db_query").Inc()
		return "", fmt.Errorf("error getting translation config: %w", err)
	}
	return language, nil
}

// SetThreadAutomation sets the thread automation config for a specific channel
func (db *DB) SetThreadAutomation(ctx context.Context, guildID, channelID string, autoJoin bool) error {
	query := `
		INSERT INTO thread_automation_config (guild_id, channel_id, auto_join)
		VALUES ($1, $2, $3)
		ON CONFLICT (guild_id, channel_id) DO UPDATE
		SET auto_join = EXCLUDED.auto_join
	`
	_, err := db.Pool.Exec(ctx, query, guildID, channelID, autoJoin)
	return err
}

// GetThreadAutomation retrieves the thread automation config for a specific channel
func (db *DB) GetThreadAutomation(ctx context.Context, guildID, channelID string) (bool, error) {
	query := `SELECT auto_join FROM thread_automation_config WHERE guild_id = $1 AND channel_id = $2`
	var autoJoin bool
	err := db.Pool.QueryRow(ctx, query, guildID, channelID).Scan(&autoJoin)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	return autoJoin, nil
}

// RemoveThreadAutomation removes the thread automation config for a specific channel
func (db *DB) RemoveThreadAutomation(ctx context.Context, guildID, channelID string) error {
	query := `DELETE FROM thread_automation_config WHERE guild_id = $1 AND channel_id = $2`
	_, err := db.Pool.Exec(ctx, query, guildID, channelID)
	return err
}

// AddForwardingRule adds a new message forwarding rule.
func (db *DB) AddForwardingRule(ctx context.Context, guildID, sourceChannelID, targetChannelID string) error {
	start := time.Now()
	query := `
		INSERT INTO forwarding_config (guild_id, source_channel_id, target_channel_id)
		VALUES ($1, $2, $3)
		ON CONFLICT (guild_id, source_channel_id, target_channel_id) DO NOTHING
	`
	_, err := db.Pool.Exec(ctx, query, guildID, sourceChannelID, targetChannelID)
	if err != nil {
		metrics.ErrorCounter.WithLabelValues("db_query").Inc()
		return err
	}
	metrics.DBQueryLatency.WithLabelValues("AddForwardingRule").Observe(time.Since(start).Seconds())
	return nil
}

// RemoveForwardingRule removes a message forwarding rule.
func (db *DB) RemoveForwardingRule(ctx context.Context, guildID, sourceChannelID, targetChannelID string) error {
	start := time.Now()
	query := `
		DELETE FROM forwarding_config
		WHERE guild_id = $1 AND source_channel_id = $2 AND target_channel_id = $3
	`
	_, err := db.Pool.Exec(ctx, query, guildID, sourceChannelID, targetChannelID)
	if err != nil {
		metrics.ErrorCounter.WithLabelValues("db_query").Inc()
		return err
	}
	metrics.DBQueryLatency.WithLabelValues("RemoveForwardingRule").Observe(time.Since(start).Seconds())
	return nil
}

// GetForwardingRules returns a list of target channel IDs for a given source channel.
func (db *DB) GetForwardingRules(ctx context.Context, guildID, sourceChannelID string) ([]string, error) {
	start := time.Now()
	query := `
		SELECT target_channel_id
		FROM forwarding_config
		WHERE guild_id = $1 AND source_channel_id = $2
	`
	rows, err := db.Pool.Query(ctx, query, guildID, sourceChannelID)
	if err != nil {
		metrics.ErrorCounter.WithLabelValues("db_query").Inc()
		return nil, err
	}
	defer rows.Close()

	var targets []string
	for rows.Next() {
		var targetID string
		if err := rows.Scan(&targetID); err != nil {
			metrics.ErrorCounter.WithLabelValues("db_query").Inc()
			return nil, err
		}
		targets = append(targets, targetID)
	}
	metrics.DBQueryLatency.WithLabelValues("GetForwardingRules").Observe(time.Since(start).Seconds())
	return targets, nil
}

func (db *DB) SetAutoDelete(ctx context.Context, guildID, channelID string, deleteAfter int) error {
	query := `
		INSERT INTO auto_delete_config (guild_id, channel_id, delete_after)
		VALUES ($1, $2, $3)
		ON CONFLICT (guild_id, channel_id) DO UPDATE
		SET delete_after = EXCLUDED.delete_after
	`
	_, err := db.Pool.Exec(ctx, query, guildID, channelID, deleteAfter)
	if err != nil {
		return fmt.Errorf("error setting auto_delete config: %w", err)
	}
	return nil
}

func (db *DB) GetAutoDelete(ctx context.Context, guildID, channelID string) (int, error) {
	query := `
		SELECT delete_after FROM auto_delete_config
		WHERE guild_id = $1 AND channel_id = $2
	`
	var deleteAfter int
	err := db.Pool.QueryRow(ctx, query, guildID, channelID).Scan(&deleteAfter)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, nil
		}
		return 0, fmt.Errorf("error getting auto_delete config: %w", err)
	}
	return deleteAfter, nil
}

func (db *DB) RemoveAutoDelete(ctx context.Context, guildID, channelID string) error {
	query := `DELETE FROM auto_delete_config WHERE guild_id = $1 AND channel_id = $2`
	_, err := db.Pool.Exec(ctx, query, guildID, channelID)
	if err != nil {
		return fmt.Errorf("error removing auto_delete config: %w", err)
	}
	return nil
}

type KeywordNotification struct {
	UserID  string
	Keyword string
}

func (db *DB) AddKeywordNotification(ctx context.Context, userID, guildID, keyword string) error {
	_, err := db.Pool.Exec(ctx,
		`INSERT INTO keyword_notifications (user_id, guild_id, keyword) VALUES ($1, $2, $3)`,
		userID, guildID, keyword)
	return err
}

func (db *DB) RemoveKeywordNotification(ctx context.Context, userID, guildID, keyword string) error {
	_, err := db.Pool.Exec(ctx,
		`DELETE FROM keyword_notifications WHERE user_id = $1 AND guild_id = $2 AND keyword = $3`,
		userID, guildID, keyword)
	return err
}

func (db *DB) GetKeywordNotifications(ctx context.Context, guildID string) ([]KeywordNotification, error) {
	rows, err := db.Pool.Query(ctx,
		`SELECT user_id, keyword FROM keyword_notifications WHERE guild_id = $1`,
		guildID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var notifs []KeywordNotification
	for rows.Next() {
		var n KeywordNotification
		if err := rows.Scan(&n.UserID, &n.Keyword); err != nil {
			return nil, err
		}
		notifs = append(notifs, n)
	}
	return notifs, nil
}

type ReactionTrigger struct {
	ID      int
	GuildID string
	Keyword string
	Emoji   string
}

func (db *DB) AddReactionTrigger(ctx context.Context, guildID, keyword, emoji string) error {
	_, err := db.Pool.Exec(ctx,
		`INSERT INTO reaction_triggers (guild_id, keyword, emoji) VALUES ($1, $2, $3)
		ON CONFLICT (guild_id, keyword) DO UPDATE SET emoji = EXCLUDED.emoji`,
		guildID, keyword, emoji)
	return err
}

func (db *DB) RemoveReactionTrigger(ctx context.Context, guildID, keyword string) error {
	_, err := db.Pool.Exec(ctx,
		`DELETE FROM reaction_triggers WHERE guild_id = $1 AND keyword = $2`,
		guildID, keyword)
	return err
}

func (db *DB) GetReactionTriggers(ctx context.Context, guildID string) ([]ReactionTrigger, error) {
	rows, err := db.Pool.Query(ctx,
		`SELECT id, guild_id, keyword, emoji FROM reaction_triggers WHERE guild_id = $1`,
		guildID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var triggers []ReactionTrigger
	for rows.Next() {
		var t ReactionTrigger
		if err := rows.Scan(&t.ID, &t.GuildID, &t.Keyword, &t.Emoji); err != nil {
			return nil, err
		}
		triggers = append(triggers, t)
	}
	return triggers, nil
}

// TempNickname represents a temporary nickname assigned to a user.
type TempNickname struct {
	ID               int
	GuildID          string
	UserID           string
	OriginalNickname string
	ExpiresAt        time.Time
}

// SetTempNickname sets a temporary nickname for a user, or updates an existing one.
func (db *DB) SetTempNickname(ctx context.Context, guildID, userID, originalNickname string, expiresAt time.Time) error {
	query := `
		INSERT INTO temp_nicknames (guild_id, user_id, original_nickname, expires_at)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (guild_id, user_id)
		DO UPDATE SET expires_at = EXCLUDED.expires_at
	`
	_, err := db.Pool.Exec(ctx, query, guildID, userID, originalNickname, expiresAt)
	return err
}

// GetExpiredTempNicknames retrieves temporary nicknames that have expired.
func (db *DB) GetExpiredTempNicknames(ctx context.Context) ([]TempNickname, error) {
	query := `
		SELECT id, guild_id, user_id, original_nickname, expires_at
		FROM temp_nicknames
		WHERE expires_at <= NOW()
	`
	rows, err := db.Pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var expired []TempNickname
	for rows.Next() {
		var tn TempNickname
		if err := rows.Scan(&tn.ID, &tn.GuildID, &tn.UserID, &tn.OriginalNickname, &tn.ExpiresAt); err != nil {
			return nil, err
		}
		expired = append(expired, tn)
	}
	return expired, rows.Err()
}

// RemoveTempNickname removes a temporary nickname record.
func (db *DB) RemoveTempNickname(ctx context.Context, id int) error {
	query := `DELETE FROM temp_nicknames WHERE id = $1`
	_, err := db.Pool.Exec(ctx, query, id)
	return err
}

// RemoveTempNicknameByGuildUser removes a temporary nickname record by guild and user ID.
func (db *DB) RemoveTempNicknameByGuildUser(ctx context.Context, guildID, userID string) error {
	query := `DELETE FROM temp_nicknames WHERE guild_id = $1 AND user_id = $2`
	_, err := db.Pool.Exec(ctx, query, guildID, userID)
	return err
}

// GetTempNicknameByGuildUser retrieves a temporary nickname record by guild and user ID.
func (db *DB) GetTempNicknameByGuildUser(ctx context.Context, guildID, userID string) (*TempNickname, error) {
	query := `
		SELECT id, guild_id, user_id, original_nickname, expires_at
		FROM temp_nicknames
		WHERE guild_id = $1 AND user_id = $2
	`
	var tn TempNickname
	err := db.Pool.QueryRow(ctx, query, guildID, userID).Scan(&tn.ID, &tn.GuildID, &tn.UserID, &tn.OriginalNickname, &tn.ExpiresAt)
	if err != nil {
		return nil, err
	}
	return &tn, nil
}

// Stock represents a market stock.
type Stock struct {
	Symbol       string
	CurrentPrice int
	History      []int
}

// UserStock represents a user's stock holdings.
type UserStock struct {
	GuildID         string
	UserID          string
	Symbol          string
	Quantity        int
	AverageBuyPrice float64
}

// GetStocks retrieves all available stocks.
func (db *DB) GetStocks(ctx context.Context) ([]Stock, error) {
	query := `SELECT symbol, current_price, history FROM stocks`
	rows, err := db.Pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stocks []Stock
	for rows.Next() {
		var s Stock
		var histBytes []byte
		if err := rows.Scan(&s.Symbol, &s.CurrentPrice, &histBytes); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(histBytes, &s.History); err != nil {
			// fallback if json parsing fails
			s.History = []int{}
		}
		stocks = append(stocks, s)
	}
	return stocks, rows.Err()
}

// GetStock retrieves a specific stock by symbol.
func (db *DB) GetStock(ctx context.Context, symbol string) (*Stock, error) {
	query := `SELECT symbol, current_price, history FROM stocks WHERE symbol = $1`
	var s Stock
	var histBytes []byte
	err := db.Pool.QueryRow(ctx, query, strings.ToUpper(symbol)).Scan(&s.Symbol, &s.CurrentPrice, &histBytes)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	if err := json.Unmarshal(histBytes, &s.History); err != nil {
		s.History = []int{}
	}
	return &s, nil
}

// GetUserStocks retrieves all stocks owned by a user.
func (db *DB) GetUserStocks(ctx context.Context, guildID, userID string) ([]UserStock, error) {
	query := `SELECT guild_id, user_id, symbol, quantity, average_buy_price FROM user_stocks WHERE guild_id = $1 AND user_id = $2 AND quantity > 0`
	rows, err := db.Pool.Query(ctx, query, guildID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stocks []UserStock
	for rows.Next() {
		var s UserStock
		if err := rows.Scan(&s.GuildID, &s.UserID, &s.Symbol, &s.Quantity, &s.AverageBuyPrice); err != nil {
			return nil, err
		}
		stocks = append(stocks, s)
	}
	return stocks, rows.Err()
}

// BuyStock attempts to purchase shares of a stock for a user.
func (db *DB) BuyStock(ctx context.Context, guildID, userID, symbol string, quantity int) error {
	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	sym := strings.ToUpper(symbol)

	// Get current stock price
	var currentPrice int
	err = tx.QueryRow(ctx, `SELECT current_price FROM stocks WHERE symbol = $1`, sym).Scan(&currentPrice)
	if err != nil {
		if err == pgx.ErrNoRows {
			return fmt.Errorf("stock %s not found", sym)
		}
		return err
	}

	totalCost := currentPrice * quantity

	// Deduct coins from user
	res, err := tx.Exec(ctx, `
		UPDATE user_economy
		SET coins = coins - $1
		WHERE guild_id = $2 AND user_id = $3 AND coins >= $1
	`, totalCost, guildID, userID)
	if err != nil {
		return err
	}
	if res.RowsAffected() == 0 {
		return fmt.Errorf("insufficient coins")
	}

	// Calculate new average buy price and insert/update
	var currentQuantity int
	var currentAvg float64
	err = tx.QueryRow(ctx, `SELECT quantity, average_buy_price FROM user_stocks WHERE guild_id = $1 AND user_id = $2 AND symbol = $3 FOR UPDATE`, guildID, userID, sym).Scan(&currentQuantity, &currentAvg)
	if err != nil && err != pgx.ErrNoRows {
		return err
	}

	newQuantity := currentQuantity + quantity
	var newAvg float64
	if currentQuantity == 0 {
		newAvg = float64(currentPrice)
	} else {
		newAvg = ((currentAvg * float64(currentQuantity)) + float64(totalCost)) / float64(newQuantity)
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO user_stocks (guild_id, user_id, symbol, quantity, average_buy_price)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (guild_id, user_id, symbol) DO UPDATE SET
			quantity = EXCLUDED.quantity,
			average_buy_price = EXCLUDED.average_buy_price
	`, guildID, userID, sym, newQuantity, newAvg)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

// SellStock attempts to sell shares of a stock for a user.
func (db *DB) SellStock(ctx context.Context, guildID, userID, symbol string, quantity int) error {
	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	sym := strings.ToUpper(symbol)

	// Check user holdings
	var currentQuantity int
	err = tx.QueryRow(ctx, `SELECT quantity FROM user_stocks WHERE guild_id = $1 AND user_id = $2 AND symbol = $3 FOR UPDATE`, guildID, userID, sym).Scan(&currentQuantity)
	if err != nil {
		if err == pgx.ErrNoRows {
			return fmt.Errorf("you do not own any shares of %s in this server", sym)
		}
		return err
	}

	if currentQuantity < quantity {
		return fmt.Errorf("you only own %d shares of %s", currentQuantity, sym)
	}

	// Get current stock price
	var currentPrice int
	err = tx.QueryRow(ctx, `SELECT current_price FROM stocks WHERE symbol = $1`, sym).Scan(&currentPrice)
	if err != nil {
		return err
	}

	totalValue := currentPrice * quantity

	// Deduct shares
	if currentQuantity == quantity {
		_, err = tx.Exec(ctx, `DELETE FROM user_stocks WHERE guild_id = $1 AND user_id = $2 AND symbol = $3`, guildID, userID, sym)
	} else {
		_, err = tx.Exec(ctx, `UPDATE user_stocks SET quantity = quantity - $1 WHERE guild_id = $2 AND user_id = $3 AND symbol = $4`, quantity, guildID, userID, sym)
	}
	if err != nil {
		return err
	}

	// Add coins to user
	_, err = tx.Exec(ctx, `
		INSERT INTO user_economy (guild_id, user_id, coins)
		VALUES ($1, $2, $3)
		ON CONFLICT (guild_id, user_id) DO UPDATE SET coins = user_economy.coins + EXCLUDED.coins
	`, guildID, userID, totalValue)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

// UpdateStockPrices simulates market fluctuations for all stocks.
func (db *DB) UpdateStockPrices(ctx context.Context) error {
	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	query := `SELECT symbol, current_price, history FROM stocks`
	rows, err := tx.Query(ctx, query)
	if err != nil {
		return err
	}

	type updateData struct {
		symbol string
		price  int
		hist   []byte
	}
	var updates []updateData

	for rows.Next() {
		var sym string
		var price int
		var histBytes []byte
		if err := rows.Scan(&sym, &price, &histBytes); err != nil {
			rows.Close()
			return err
		}

		var hist []int
		if err := json.Unmarshal(histBytes, &hist); err != nil {
			hist = []int{}
		}

		// Calculate new price (fluctuate by -10% to +10%)
		changePercent := (rand.Float64() * 0.2) - 0.1
		changeAmt := int(float64(price) * changePercent)

		// Add some random noise (-5 to +5)
		noise := rand.Intn(11) - 5
		newPrice := price + changeAmt + noise

		// Ensure price doesn't drop below 1
		if newPrice < 1 {
			newPrice = 1
		}

		// Update history (keep last 10)
		hist = append(hist, newPrice)
		if len(hist) > 10 {
			hist = hist[len(hist)-10:]
		}

		newHistBytes, _ := json.Marshal(hist)
		updates = append(updates, updateData{sym, newPrice, newHistBytes})
	}
	rows.Close()

	for _, u := range updates {
		_, err := tx.Exec(ctx, `UPDATE stocks SET current_price = $1, history = $2 WHERE symbol = $3`, u.price, u.hist, u.symbol)
		if err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

// SetJoinLeaveLog saves or updates the join/leave log configuration for a guild.
func (db *DB) SetJoinLeaveLog(ctx context.Context, guildID, channelID string, logJoins, logLeaves bool) error {
	query := `
		INSERT INTO join_leave_log_config (guild_id, channel_id, log_joins, log_leaves)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (guild_id) DO UPDATE
		SET channel_id = EXCLUDED.channel_id,
		    log_joins = EXCLUDED.log_joins,
		    log_leaves = EXCLUDED.log_leaves
	`
	_, err := db.Pool.Exec(ctx, query, guildID, channelID, logJoins, logLeaves)
	if err != nil {
		metrics.ErrorCounter.WithLabelValues("db_query").Inc()
	}
	return err
}

// GetJoinLeaveLog retrieves the join/leave log configuration for a guild.
func (db *DB) GetJoinLeaveLog(ctx context.Context, guildID string) (channelID string, logJoins bool, logLeaves bool, err error) {
	query := `
		SELECT channel_id, log_joins, log_leaves
		FROM join_leave_log_config
		WHERE guild_id = $1
	`
	err = db.Pool.QueryRow(ctx, query, guildID).Scan(&channelID, &logJoins, &logLeaves)
	if err == pgx.ErrNoRows {
		return "", false, false, nil
	}
	if err != nil {
		metrics.ErrorCounter.WithLabelValues("db_query").Inc()
	}
	return channelID, logJoins, logLeaves, err
}

// RemoveJoinLeaveLog deletes the join/leave log configuration for a guild.
func (db *DB) RemoveJoinLeaveLog(ctx context.Context, guildID string) error {
	query := `DELETE FROM join_leave_log_config WHERE guild_id = $1`
	_, err := db.Pool.Exec(ctx, query, guildID)
	if err != nil {
		metrics.ErrorCounter.WithLabelValues("db_query").Inc()
	}
	return err
}

// AutoReactConfig represents an auto-react configuration for a channel.
type AutoReactConfig struct {
	ID        int       `json:"id"`
	GuildID   string    `json:"guild_id"`
	ChannelID string    `json:"channel_id"`
	Emojis    string    `json:"emojis"`
	CreatedAt time.Time `json:"created_at"`
}

// AddAutoReact adds or updates an auto-react configuration for a channel.
func (db *DB) AddAutoReact(ctx context.Context, guildID, channelID, emojis string) error {
	start := time.Now()
	defer func() {
		metrics.DBQueryLatency.WithLabelValues("AddAutoReact").Observe(time.Since(start).Seconds())
	}()

	query := `
		INSERT INTO auto_react_config (guild_id, channel_id, emojis)
		VALUES ($1, $2, $3)
		ON CONFLICT (guild_id, channel_id) DO UPDATE SET
			emojis = EXCLUDED.emojis
	`
	_, err := db.Pool.Exec(ctx, query, guildID, channelID, emojis)
	return err
}

// RemoveAutoReact removes an auto-react configuration for a channel.
func (db *DB) RemoveAutoReact(ctx context.Context, guildID, channelID string) error {
	start := time.Now()
	defer func() {
		metrics.DBQueryLatency.WithLabelValues("RemoveAutoReact").Observe(time.Since(start).Seconds())
	}()

	query := `DELETE FROM auto_react_config WHERE guild_id = $1 AND channel_id = $2`
	_, err := db.Pool.Exec(ctx, query, guildID, channelID)
	return err
}

// GetAutoReactChannels retrieves all auto-react configurations for a guild.
func (db *DB) GetAutoReactChannels(ctx context.Context, guildID string) ([]AutoReactConfig, error) {
	start := time.Now()
	defer func() {
		metrics.DBQueryLatency.WithLabelValues("GetAutoReactChannels").Observe(time.Since(start).Seconds())
	}()

	query := `SELECT id, guild_id, channel_id, emojis, created_at FROM auto_react_config WHERE guild_id = $1`
	rows, err := db.Pool.Query(ctx, query, guildID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var configs []AutoReactConfig
	for rows.Next() {
		var c AutoReactConfig
		if err := rows.Scan(&c.ID, &c.GuildID, &c.ChannelID, &c.Emojis, &c.CreatedAt); err != nil {
			return nil, err
		}
		configs = append(configs, c)
	}
	return configs, rows.Err()
}

// AddWelcomeRole adds a welcome role.
func (db *DB) AddWelcomeRole(ctx context.Context, guildID, roleID string) error {
	query := `
		INSERT INTO welcome_roles (guild_id, role_id)
		VALUES ($1, $2)
		ON CONFLICT (guild_id, role_id) DO NOTHING
	`
	_, err := db.Pool.Exec(ctx, query, guildID, roleID)
	return err
}

// RemoveWelcomeRole removes a welcome role.
func (db *DB) RemoveWelcomeRole(ctx context.Context, guildID, roleID string) error {
	query := `
		DELETE FROM welcome_roles
		WHERE guild_id = $1 AND role_id = $2
	`
	_, err := db.Pool.Exec(ctx, query, guildID, roleID)
	return err
}

// GetWelcomeRoles returns all welcome roles for a guild.
func (db *DB) GetWelcomeRoles(ctx context.Context, guildID string) ([]string, error) {
	query := `
		SELECT role_id
		FROM welcome_roles
		WHERE guild_id = $1
	`
	rows, err := db.Pool.Query(ctx, query, guildID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var roles []string
	for rows.Next() {
		var roleID string
		if err := rows.Scan(&roleID); err != nil {
			return nil, err
		}
		roles = append(roles, roleID)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return roles, nil
}
