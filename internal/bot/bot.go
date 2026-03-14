package bot

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"math/rand"
	"strings"
	"sync"
	"time"

	"julescord/internal/bot/commands"
	"julescord/internal/config"
	"julescord/internal/db"

	"github.com/bwmarrin/discordgo"
)

// Bot manages the Discord connection.
type Bot struct {
	Session        *discordgo.Session
	Config         *config.Config
	Registry       *commands.Registry
	DB             *db.DB
	xpCooldown     sync.Map // map[string]time.Time (key: guildID_channelID_userID)
	AutoResponders sync.Map // map[string][]*db.AutoResponder (key: guildID)
}

// New initializes a new bot instance.
func New(cfg *config.Config, database *db.DB) (*Bot, error) {
	if cfg.DiscordToken == "" {
		return nil, fmt.Errorf("discord token is required")
	}

	session, err := discordgo.New("Bot " + cfg.DiscordToken)
	if err != nil {
		return nil, fmt.Errorf("error creating Discord session: %w", err)
	}

	registry := commands.NewRegistry()
	registry.Add(commands.Ping())
	registry.Add(commands.About())
	registry.Add(commands.Stats(database))
	registry.Add(commands.Help(registry))
	registry.Add(commands.Warn(database))
	registry.Add(commands.Warnings(database))
	registry.Add(commands.Kick(database))
	registry.Add(commands.Ban(database))
	registry.Add(commands.Purge(database))
	registry.Add(commands.Rank(database))
	registry.Add(commands.Leaderboard(database))
	registry.Add(commands.Daily(database))
	registry.Add(commands.Coins(database))
	registry.Add(commands.Config(database))
	registry.Add(commands.ReactionRole(database))
	registry.Add(commands.Schedule(database))
	registry.Add(commands.Changelog())
	registry.Add(commands.Remind(database))
	registry.Add(commands.Ticket(database))
	registry.Add(commands.Tag(database))
	registry.Add(commands.Giveaway(database))
	registry.Add(commands.AFKCommand(database))

	bot := &Bot{
		Session:  session,
		Config:   cfg,
		Registry: registry,
		DB:       database,
	}

	registry.Add(commands.AutoResponder(database, bot))
	registry.Add(commands.Starboard(database))
	registry.Add(commands.NewStickyCommand(bot))
	registry.Add(commands.Poll(database))

	// Load auto-responders into memory cache
	if database != nil {
		allResponders, err := database.ListAllAutoResponders(context.Background())
		if err != nil {
			slog.Error("Failed to load auto-responders into cache", "error", err)
		} else {
			grouped := make(map[string][]*db.AutoResponder)
			for _, r := range allResponders {
				grouped[r.GuildID] = append(grouped[r.GuildID], r)
			}
			for guildID, responders := range grouped {
				bot.AutoResponders.Store(guildID, responders)
			}
			slog.Info("Successfully loaded auto-responders into cache")
		}
	}

	// Register ready handler
	bot.Session.AddHandler(bot.readyHandler)

	// Register guild create handler
	bot.Session.AddHandler(bot.guildCreateHandler)

	// Register interaction handler
	bot.Session.AddHandler(bot.interactionCreateHandler)

	// Register message create handler
	bot.Session.AddHandler(bot.messageCreateHandler)

	// Register guild member add handler
	bot.Session.AddHandler(bot.guildMemberAddHandler)

	// Register message reaction add handler
	bot.Session.AddHandler(bot.messageReactionAddHandler)

	// Register message reaction remove handler
	bot.Session.AddHandler(bot.messageReactionRemoveHandler)

	// Set intentions
	bot.Session.Identify.Intents = discordgo.IntentsGuilds | discordgo.IntentsGuildMessages | discordgo.IntentsGuildMembers | discordgo.IntentsGuildMessageReactions

	return bot, nil
}

// UpdateAutoResponderCache updates the in-memory cache for a specific guild
func (b *Bot) UpdateAutoResponderCache(guildID string) {
	if b.DB == nil {
		return
	}

	responders, err := b.DB.ListAutoResponders(context.Background(), guildID)
	if err != nil {
		slog.Error("Failed to update auto-responder cache for guild", "guild_id", guildID, "error", err)
		return
	}

	b.AutoResponders.Store(guildID, responders)
}

// Start opens the connection to Discord.
func (b *Bot) Start() error {
	slog.Info("Starting Discord bot...")
	err := b.Session.Open()
	if err != nil {
		return fmt.Errorf("error opening Discord connection: %w", err)
	}

	slog.Info("Discord bot started successfully.")
	return nil
}

// Stop closes the connection to Discord gracefully.
func (b *Bot) Stop() error {
	slog.Info("Stopping Discord bot...")
	err := b.Session.Close()
	if err != nil {
		return fmt.Errorf("error closing Discord connection: %w", err)
	}

	slog.Info("Discord bot stopped gracefully.")
	return nil
}

// readyHandler triggers when the bot connects to Discord.
func (b *Bot) readyHandler(s *discordgo.Session, event *discordgo.Ready) {
	slog.Info(fmt.Sprintf("Bot is ready! Logged in as %s#%s", event.User.Username, event.User.Discriminator))

	// Register commands with Discord when ready
	err := b.Registry.RegisterWithDiscord(s, b.Config.DiscordClientID, "")
	if err != nil {
		slog.Error("Error registering commands", "error", err)
	}

	// Start bot status rotation
	go b.rotateStatus()

	// Start scheduled announcements checker
	go b.checkScheduledAnnouncements()

	// Start reminder delivery checker
	go b.checkReminders()

	// Start giveaway checker
	go b.checkGiveaways()
}

// checkScheduledAnnouncements checks for pending announcements and sends them.
func (b *Bot) checkScheduledAnnouncements() {
	if b.DB == nil {
		return
	}

	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		announcements, err := b.DB.GetPendingAnnouncements(context.Background())
		if err != nil {
			slog.Error("Failed to get pending announcements", "error", err)
		} else {
			for _, a := range announcements {
				_, err := b.Session.ChannelMessageSend(a.ChannelID, a.Message)
				if err != nil {
					slog.Error("Failed to send scheduled announcement %d to channel %s", "arg1", a.ID, "arg2", a.ChannelID, "error", err)
				}

				// Mark as sent regardless of success to avoid spamming errors if channel is deleted/bot lacks permissions
				err = b.DB.MarkAnnouncementSent(context.Background(), a.ID)
				if err != nil {
					slog.Error("Failed to mark announcement %d as sent", "arg1", a.ID, "error", err)
				}
			}
		}

		<-ticker.C
	}
}

// rotateStatus updates the bot's custom status periodically.
func (b *Bot) rotateStatus() {
	statuses := []string{
		"Building myself...",
		"Reading AGENTS.md...",
		"Running go build ./...",
		"Checking pull requests...",
		"Connecting to PostgreSQL...",
		"Watching the dashboard...",
	}

	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		// Pick a random status
		status := statuses[rand.Intn(len(statuses))]

		err := b.Session.UpdateGameStatus(0, status)
		if err != nil {
			slog.Error("Failed to update bot status", "error", err)
		}

		<-ticker.C
	}
}

// guildCreateHandler is called when the bot joins a new guild or a guild becomes available.
func (b *Bot) guildCreateHandler(s *discordgo.Session, event *discordgo.GuildCreate) {
	if b.DB == nil {
		return
	}

	err := b.DB.UpsertGuild(context.Background(), event.Guild.ID)
	if err != nil {
		slog.Error("Failed to upsert guild %s", "arg1", event.Guild.ID, "error", err)
	} else {
		slog.Info(fmt.Sprintf("Guild registered/upserted: %s", event.Guild.ID))
	}
}

// interactionCreateHandler handles all slash commands
func (b *Bot) interactionCreateHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if b.DB != nil && i.Type == discordgo.InteractionApplicationCommand {
		var user *discordgo.User
		if i.Member != nil {
			user = i.Member.User
		} else {
			user = i.User // Fallback for DMs
		}

		if user != nil {
			// Track user
			err := b.DB.UpsertUser(context.Background(), user.ID, user.Username, user.GlobalName, user.AvatarURL(""))
			if err != nil {
				slog.Error("Failed to upsert user %s", "arg1", user.ID, "error", err)
			}

			// Log command execution
			commandName := i.ApplicationCommandData().Name
			err = b.DB.LogCommand(context.Background(), commandName, user.ID, i.GuildID)
			if err != nil {
				slog.Error("Failed to log command execution for %s", "arg1", commandName, "error", err)
			}
		}
	}

	b.Registry.Dispatch(s, i)
}

// messageCreateHandler is called every time a new message is created
func (b *Bot) messageCreateHandler(s *discordgo.Session, m *discordgo.MessageCreate) {

	// Ignore all messages created by the bot itself or other bots
	if m.Author.ID == s.State.User.ID || m.Author.Bot {
		return
	}

	// Sticky Messages System
	if b.DB != nil && m.GuildID != "" {
		// Check for sticky message in channel
		sticky, err := b.DB.GetSticky(context.Background(), m.ChannelID)
		if err == nil && sticky != nil {
			if sticky.LastMessageID != "" {
				_ = s.ChannelMessageDelete(m.ChannelID, sticky.LastMessageID)
			}
			newMsg, err := s.ChannelMessageSend(m.ChannelID, sticky.MessageText)
			if err == nil {
				_ = b.DB.UpdateStickyMessageID(context.Background(), m.ChannelID, newMsg.ID)
			}
		}
	}

	// AFK System
	if b.DB != nil && m.GuildID != "" {
		// Check if author was AFK
		reason, _, err := b.DB.GetAFK(context.Background(), m.Author.ID, m.GuildID)
		if err == nil && reason != "" {
			err = b.DB.RemoveAFK(context.Background(), m.Author.ID, m.GuildID)
			if err == nil {
				_, _ = s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Welcome back %s! I've removed your AFK status.", m.Author.Mention()))
			}
		}

		// Check if mentioned users are AFK
		for _, mention := range m.Mentions {
			if mention.Bot || mention.ID == m.Author.ID {
				continue
			}
			mentionReason, createdAt, err := b.DB.GetAFK(context.Background(), mention.ID, m.GuildID)
			if err == nil && mentionReason != "" {
				duration := time.Since(createdAt).Round(time.Minute)
				msg := fmt.Sprintf("%s is currently AFK: %s", mention.Username, mentionReason)
				if duration > 0 {
					msg += fmt.Sprintf(" - %s ago", duration.String())
				}
				_, _ = s.ChannelMessageSend(m.ChannelID, msg)
			}
		}
	}

	// XP System
	if b.DB != nil && m.GuildID != "" {
		cooldownKey := fmt.Sprintf("%s_%s_%s", m.GuildID, m.ChannelID, m.Author.ID)
		now := time.Now()

		var onCooldown bool
		if lastXpTimeAny, ok := b.xpCooldown.Load(cooldownKey); ok {
			lastXpTime := lastXpTimeAny.(time.Time)
			if now.Sub(lastXpTime) < time.Minute {
				onCooldown = true
			}
		}

		// Check Auto-Responders from cache
		if respondersAny, ok := b.AutoResponders.Load(m.GuildID); ok {
			responders := respondersAny.([]*db.AutoResponder)
			contentLower := strings.ToLower(strings.TrimSpace(m.Content))
			for _, r := range responders {
				if strings.Contains(contentLower, r.TriggerWord) {
					_, err := s.ChannelMessageSend(m.ChannelID, r.Response)
					if err != nil {
						slog.Error("Failed to send auto-responder message", "error", err)
					}
					// Only trigger one auto-responder per message
					break
				}
			}
		}

		if !onCooldown {
			// Award XP (e.g., random 15-25 XP)
			amount := rand.Intn(11) + 15

			// Ensure user exists first
			err := b.DB.UpsertUser(context.Background(), m.Author.ID, m.Author.Username, m.Author.GlobalName, m.Author.AvatarURL(""))
			if err != nil {
				slog.Error("Failed to upsert user %s for XP", "arg1", m.Author.ID, "error", err)
			} else {
				// Add XP
				newXP, err := b.DB.AddXP(context.Background(), m.GuildID, m.Author.ID, amount)
				if err != nil {
					slog.Error("Failed to add XP to user %s", "arg1", m.Author.ID, "error", err)
				} else {
					// Update cooldown
					b.xpCooldown.Store(cooldownKey, now)

					// Calculate new level: Level = floor(sqrt(XP) / 10)
					// (Level 1 = 100 XP, Level 2 = 400 XP, Level 3 = 900 XP, etc.)
					newLevel := int(math.Floor(math.Sqrt(float64(newXP)) / 10.0))

					// Fetch current economy state to get the previous level
					oldEcon, err := b.DB.GetUserEconomy(context.Background(), m.GuildID, m.Author.ID)
					oldLevel := 0
					if err == nil && oldEcon != nil {
						oldLevel = oldEcon.Level
					}

					if newLevel > oldLevel {
						// Update level in DB
						err := b.DB.SetLevel(context.Background(), m.GuildID, m.Author.ID, newLevel)
						if err != nil {
							slog.Error("Failed to update level for user %s", "arg1", m.Author.ID, "error", err)
						} else {
							// Announce level up
							msg := fmt.Sprintf("🎉 Congratulations <@%s>, you just advanced to **Level %d**!", m.Author.ID, newLevel)
							_, err = s.ChannelMessageSend(m.ChannelID, msg)
							if err != nil {
								slog.Error("Failed to send level up message", "error", err)
							}
						}
					}
				}
			}
		}
	}
}

// messageReactionAddHandler is called when a user adds a reaction to a message
func (b *Bot) messageReactionAddHandler(s *discordgo.Session, r *discordgo.MessageReactionAdd) {

	// Starboard check
	b.handleStarboardReaction(s, r.GuildID, r.ChannelID, r.MessageID, r.Emoji.Name)

	if b.DB == nil || r.GuildID == "" || r.UserID == s.State.User.ID {
		return
	}

	emojiName := r.Emoji.Name
	if r.Emoji.ID != "" {
		emojiName = fmt.Sprintf("%s:%s", r.Emoji.Name, r.Emoji.ID)
	}

	// Check for giveaway entry
	if emojiName == "🎉" {
		g, err := b.DB.GetGiveawayByMessage(context.Background(), r.MessageID)
		if err == nil && g != nil && !g.Ended {
			err = b.DB.AddGiveawayEntrant(context.Background(), g.ID, r.UserID)
			if err != nil {
				slog.Error("Failed to add giveaway entrant", "error", err)
			}
		}
	}

	rr, err := b.DB.GetReactionRole(context.Background(), r.MessageID, emojiName)
	if err != nil {
		slog.Error("Failed to get reaction role config", "error", err)
		return
	}

	if rr != nil {
		err = s.GuildMemberRoleAdd(r.GuildID, r.UserID, rr.RoleID)
		if err != nil {
			slog.Error("Failed to add role %s to user %s via reaction", "arg1", rr.RoleID, "arg2", r.UserID, "error", err)
		}
	}
}

// messageReactionRemoveHandler is called when a user removes a reaction from a message
func (b *Bot) messageReactionRemoveHandler(s *discordgo.Session, r *discordgo.MessageReactionRemove) {

	// Starboard check
	b.handleStarboardReaction(s, r.GuildID, r.ChannelID, r.MessageID, r.Emoji.Name)

	if b.DB == nil || r.GuildID == "" || r.UserID == s.State.User.ID {
		return
	}

	emojiName := r.Emoji.Name
	if r.Emoji.ID != "" {
		emojiName = fmt.Sprintf("%s:%s", r.Emoji.Name, r.Emoji.ID)
	}

	rr, err := b.DB.GetReactionRole(context.Background(), r.MessageID, emojiName)
	if err != nil {
		slog.Error("Failed to get reaction role config", "error", err)
		return
	}

	if rr != nil {
		err = s.GuildMemberRoleRemove(r.GuildID, r.UserID, rr.RoleID)
		if err != nil {
			slog.Error("Failed to remove role %s from user %s via reaction", "arg1", rr.RoleID, "arg2", r.UserID, "error", err)
		}
	}
}

// guildMemberAddHandler is called every time a new member joins a guild
func (b *Bot) guildMemberAddHandler(s *discordgo.Session, m *discordgo.GuildMemberAdd) {
	if b.DB == nil {
		return
	}

	config, err := b.DB.GetGuildConfig(context.Background(), m.GuildID)
	if err != nil {
		slog.Error("Failed to get guild config for welcome message/auto-role", "error", err)
		return
	}

	if config.WelcomeChannelID != nil && *config.WelcomeChannelID != "" {
		welcomeMsg := fmt.Sprintf("Welcome to the server, <@%s>! We are glad to have you here.", m.User.ID)
		_, err := s.ChannelMessageSend(*config.WelcomeChannelID, welcomeMsg)
		if err != nil {
			slog.Error("Failed to send welcome message", "error", err)
		}
	}

	if config.AutoRoleID != nil && *config.AutoRoleID != "" {
		err := s.GuildMemberRoleAdd(m.GuildID, m.User.ID, *config.AutoRoleID)
		if err != nil {
			slog.Error("Failed to assign auto-role to user %s in guild %s", "arg1", m.User.ID, "arg2", m.GuildID, "error", err)
		}
	}
}

// checkReminders checks for pending reminders and sends them.
func (b *Bot) checkReminders() {
	if b.DB == nil {
		return
	}

	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		reminders, err := b.DB.GetDueReminders(context.Background())
		if err != nil {
			slog.Error("Failed to get due reminders", "error", err)
		} else {
			for _, r := range reminders {
				msg := fmt.Sprintf("⏰ <@%s>, here is your reminder: **%s**", r.UserID, r.Message)
				_, err := b.Session.ChannelMessageSend(r.ChannelID, msg)
				if err != nil {
					slog.Error("Failed to send reminder", "id", r.ID, "channel", r.ChannelID, "error", err)
				}

				err = b.DB.MarkReminderDelivered(context.Background(), r.ID)
				if err != nil {
					slog.Error("Failed to mark reminder as delivered", "id", r.ID, "error", err)
				}
			}
		}

		<-ticker.C
	}
}

// handleStarboardReaction processes reactions to check if they should be added/updated on the starboard.
func (b *Bot) handleStarboardReaction(s *discordgo.Session, guildID, channelID, messageID, emojiName string) {
	if b.DB == nil || guildID == "" || emojiName != "⭐" {
		return
	}

	config, err := b.DB.GetStarboardConfig(context.Background(), guildID)
	if err != nil || config == nil {
		// Not configured or error
		return
	}

	// Prevent starboard-ception: don't track reactions on messages inside the starboard channel itself
	if channelID == config.ChannelID {
		return
	}

	// Fetch the message to count reactions
	msg, err := s.ChannelMessage(channelID, messageID)
	if err != nil {
		slog.Error("Failed to fetch message for starboard", "error", err, "message_id", messageID)
		return
	}

	// Calculate total star reactions
	var starCount int
	for _, reaction := range msg.Reactions {
		if reaction.Emoji.Name == "⭐" {
			starCount = reaction.Count
			break
		}
	}

	// Determine action based on threshold
	sbMsg, err := b.DB.GetStarboardMessage(context.Background(), messageID)
	if err != nil {
		slog.Error("Failed to fetch starboard message config", "error", err, "message_id", messageID)
		return
	}

	// Create embed for the starboard
	authorName := msg.Author.Username
	if msg.Author.GlobalName != "" {
		authorName = msg.Author.GlobalName
	}

	// Default embed color (gold for stars)
	embedColor := 0xFFD700

	// Create the embed with the message content
	embed := &discordgo.MessageEmbed{
		Author: &discordgo.MessageEmbedAuthor{
			Name:    authorName,
			IconURL: msg.Author.AvatarURL(""),
		},
		Description: msg.Content,
		Color:       embedColor,
		Timestamp:   msg.Timestamp.Format(time.RFC3339),
	}

	// Add link to original message
	embed.Description += fmt.Sprintf("\n\n[Jump to Message](https://discord.com/channels/%s/%s/%s)", guildID, channelID, messageID)

	// Add first attachment if exists
	if len(msg.Attachments) > 0 {
		embed.Image = &discordgo.MessageEmbedImage{
			URL: msg.Attachments[0].URL,
		}
	}

	starText := fmt.Sprintf("⭐ **%d** | <#%s>", starCount, channelID)

	if starCount >= config.MinStars {
		if sbMsg == nil || sbMsg.StarboardMessageID == "" {
			// Threshold reached for the first time, send new message to starboard
			sentMsg, err := s.ChannelMessageSendComplex(config.ChannelID, &discordgo.MessageSend{
				Content: starText,
				Embeds:  []*discordgo.MessageEmbed{embed},
			})
			if err != nil {
				slog.Error("Failed to send message to starboard channel", "error", err, "channel_id", config.ChannelID)
				return
			}

			// Upsert to database
			err = b.DB.UpsertStarboardMessage(context.Background(), messageID, guildID, channelID, sentMsg.ID, starCount)
			if err != nil {
				slog.Error("Failed to upsert starboard message to database", "error", err, "message_id", messageID)
			}
		} else {
			// Update existing starboard message
			_, err := s.ChannelMessageEditComplex(&discordgo.MessageEdit{
				Channel: config.ChannelID,
				ID:      sbMsg.StarboardMessageID,
				Content: &starText,
				Embeds:  &[]*discordgo.MessageEmbed{embed},
			})
			if err != nil {
				slog.Error("Failed to update starboard message", "error", err, "starboard_message_id", sbMsg.StarboardMessageID)
				return
			}

			// Update count in database
			err = b.DB.UpsertStarboardMessage(context.Background(), messageID, guildID, channelID, sbMsg.StarboardMessageID, starCount)
			if err != nil {
				slog.Error("Failed to update starboard message count in database", "error", err, "message_id", messageID)
			}
		}
	} else if sbMsg != nil && sbMsg.StarboardMessageID != "" {
		// Fell below threshold, delete from starboard
		err := s.ChannelMessageDelete(config.ChannelID, sbMsg.StarboardMessageID)
		if err != nil {
			slog.Error("Failed to delete starboard message", "error", err, "starboard_message_id", sbMsg.StarboardMessageID)
		}

		// Remove mapping in database or mark as 0 stars
		err = b.DB.UpsertStarboardMessage(context.Background(), messageID, guildID, channelID, "", starCount)
		if err != nil {
			slog.Error("Failed to remove starboard message mapping from database", "error", err, "message_id", messageID)
		}
	}
}

// checkGiveaways checks for ended giveaways and picks winners.
func (b *Bot) checkGiveaways() {
	if b.DB == nil {
		return
	}

	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		giveaways, err := b.DB.GetActiveGiveaways(context.Background())
		if err != nil {
			slog.Error("Failed to get active giveaways", "error", err)
		} else {
			for _, g := range giveaways {
				// Pick winners and announce
				commands.PickGiveawayWinners(b.Session, b.DB, g)

				// Mark as ended
				err = b.DB.EndGiveaway(context.Background(), g.MessageID)
				if err != nil {
					slog.Error("Failed to mark giveaway as ended", "error", err, "message_id", g.MessageID)
				}
			}
		}

		<-ticker.C
	}
}

// GetDB returns the database instance for commands to use.
func (b *Bot) GetDB() *db.DB {
	return b.DB
}
