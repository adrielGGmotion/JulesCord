package bot

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"math/rand"
	"strconv"
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
	registry.Add(commands.Rep(database))
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
	registry.Add(commands.VoiceLog(database))
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
	registry.Add(commands.NewSuggestCommand(bot))
	registry.Add(commands.ServerLog(database))
	registry.Add(commands.Automod(database))
	registry.Add(commands.Verification(database))
	registry.Add(commands.NoteCommand(database))
	registry.Add(commands.LevelRole(database))
	registry.Add(commands.NewProfileCommand(database))
	registry.Add(commands.Shop(database))
	registry.Add(commands.NewMarryCommand(database))
	registry.Add(commands.Inventory(database))
	registry.Add(commands.Birthday(database))
	registry.Add(commands.TempVoice(database))
	registry.Add(commands.NewCountingCommand(database))
	registry.Add(commands.NewTriviaCommand(database))
	registry.Add(commands.CustomCommand(database))
	registry.Add(commands.Snipe(database))
	registry.Add(commands.EditSnipe(database))
	registry.Add(commands.Gamble(database))
	registry.Add(commands.Confession(database))
	registry.Add(commands.Confess(database))
	registry.Add(commands.Todo(database))
	registry.Add(commands.RoleMenu(database))
	registry.Add(commands.Modmail(database))

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

	// Register message update handler
	bot.Session.AddHandler(bot.messageUpdateHandler)

	// Register message delete handler
	bot.Session.AddHandler(bot.messageDeleteHandler)
	bot.Session.AddHandler(bot.voiceStateUpdateHandler)

	// Set intentions
	bot.Session.Identify.Intents = discordgo.IntentsGuilds | discordgo.IntentsGuildMessages | discordgo.IntentsGuildMembers | discordgo.IntentsGuildMessageReactions | discordgo.IntentsMessageContent | discordgo.IntentsGuildVoiceStates

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
	go b.checkBirthdays()
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
	if b.DB != nil && i.Type == discordgo.InteractionMessageComponent {
			if i.MessageComponentData().CustomID == "role_menu_select" {
				selectedRoles := i.MessageComponentData().Values
				_, err := s.GuildMember(i.GuildID, i.Member.User.ID)
				if err != nil {
					slog.Error("Failed to fetch guild member for role menu", "error", err)
					return
				}

				// Get the role menu options from DB to know which roles are part of this menu
				menuOptions, err := b.DB.GetRoleMenu(context.Background(), i.Message.ID)
				if err != nil {
					slog.Error("Failed to fetch role menu options", "error", err)
					return
				}

				// Create a map of available roles in this menu for O(1) lookups
				availableRoles := make(map[string]bool)
				for _, opt := range menuOptions {
					availableRoles[opt.RoleID] = true
				}

				// Create a map of the roles the user selected
				selectedMap := make(map[string]bool)
				for _, r := range selectedRoles {
					selectedMap[r] = true
				}

				// Apply roles
				for roleID := range availableRoles {
					if selectedMap[roleID] {
						// Role was selected, ensure user has it
						_ = s.GuildMemberRoleAdd(i.GuildID, i.Member.User.ID, roleID)
					} else {
						// Role was NOT selected, but it's part of the menu. Ensure user DOES NOT have it
						_ = s.GuildMemberRoleRemove(i.GuildID, i.Member.User.ID, roleID)
					}
				}

				_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: "Your roles have been updated!",
						Flags:   discordgo.MessageFlagsEphemeral,
					},
				})
				return
			}

		if i.MessageComponentData().CustomID == "ticket_panel_button" {
			commands.HandleCreateTicket(s, i, b.DB, []*discordgo.ApplicationCommandInteractionDataOption{
				{
					Name:  "reason",
					Type:  discordgo.ApplicationCommandOptionString,
					Value: "Panel Support Ticket",
				},
			})
			return
		}

		if strings.HasPrefix(i.MessageComponentData().CustomID, "trivia_answer_") {
			customID := i.MessageComponentData().CustomID
			parts := strings.Split(customID, "_")
			if len(parts) >= 5 {
				isCorrectStr := parts[2]
				isCorrect := isCorrectStr == "1"

				var components []discordgo.MessageComponent
				// Reconstruct components to disable them
				if len(i.Message.Components) > 0 {
					if actionRow, ok := i.Message.Components[0].(*discordgo.ActionsRow); ok {
						var newActionRowComponents []discordgo.MessageComponent
						for _, comp := range actionRow.Components {
							if button, ok := comp.(*discordgo.Button); ok {
								button.Disabled = true

								// Highlight correct answer
								btnParts := strings.Split(button.CustomID, "_")
								if len(btnParts) >= 5 && btnParts[2] == "1" {
									button.Style = discordgo.SuccessButton
								} else if button.CustomID == customID && !isCorrect {
									// Highlight chosen wrong answer
									button.Style = discordgo.DangerButton
								} else {
									button.Style = discordgo.SecondaryButton
								}
								newActionRowComponents = append(newActionRowComponents, button)
							}
						}
						components = append(components, &discordgo.ActionsRow{
							Components: newActionRowComponents,
						})
					}
				}

				embed := i.Message.Embeds[0]
				var content string
				if isCorrect {
					content = fmt.Sprintf("🎉 <@%s> got it right and won **10 coins** and **1 trivia point**!", i.Member.User.ID)
					embed.Color = 0x2ecc71 // Green
					if b.DB != nil {
						_ = b.DB.AddTriviaScore(context.Background(), i.GuildID, i.Member.User.ID)
						_ = b.DB.AddCoins(context.Background(), i.GuildID, i.Member.User.ID, 10)
					}
				} else {
					content = fmt.Sprintf("❌ <@%s> got it wrong!", i.Member.User.ID)
					embed.Color = 0xe74c3c // Red
				}

				_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseUpdateMessage,
					Data: &discordgo.InteractionResponseData{
						Content:    content,
						Embeds:     []*discordgo.MessageEmbed{embed},
						Components: components,
					},
				})
			}
			return
		}

		if i.MessageComponentData().CustomID == "verify_button" {
			config, err := b.DB.GetVerificationConfig(context.Background(), i.GuildID)
			if err != nil || config == nil {
				slog.Error("Failed to get verification config", "error", err, "guild_id", i.GuildID)
				_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: "Verification system is not configured correctly. Please contact an administrator.",
						Flags:   discordgo.MessageFlagsEphemeral,
					},
				})
				return
			}

			err = s.GuildMemberRoleAdd(i.GuildID, i.Member.User.ID, config.RoleID)
			if err != nil {
				slog.Error("Failed to assign verified role", "error", err, "user_id", i.Member.User.ID)
				_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: "Failed to assign the verified role. Please contact an administrator.",
						Flags:   discordgo.MessageFlagsEphemeral,
					},
				})
				return
			}

			_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "You have been successfully verified! Enjoy the server.",
					Flags:   discordgo.MessageFlagsEphemeral,
				},
			})
			return
		}
	}

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

	// --- ModMail System ---
	// Check if DM
	if m.GuildID == "" {
		b.handleModmailDM(s, m)
		return
	}

	// Check if it is a mod replying in a modmail thread channel
	if b.DB != nil {
		thread, err := b.DB.GetModmailThreadByChannel(context.Background(), m.ChannelID)
		if err == nil && thread != nil && thread.IsOpen {
			b.handleModmailReply(s, m, thread)
			return
		}
	}

	// Auto-Moderation System
	if b.checkAutomod(s, m.GuildID, m.ChannelID, m.ID, m.Content, m.Author.ID, m.Author.String(), m.Author.AvatarURL("")) {
		return
	}

	// Counting System
	if b.DB != nil && m.GuildID != "" {
		config, err := b.DB.GetCountingChannel(context.Background(), m.GuildID)
		if err == nil && config != nil && config.ChannelID == m.ChannelID {
			number, err := strconv.Atoi(strings.TrimSpace(m.Content))
			if err == nil {
				expectedNumber := config.CurrentNumber + 1
				if number == expectedNumber {
					if config.LastUserID == nil || *config.LastUserID != m.Author.ID {
						// Correct number and not the same user
						b.DB.UpdateCountingNumber(context.Background(), m.GuildID, number, m.Author.ID)
						s.MessageReactionAdd(m.ChannelID, m.ID, "✅")
					} else {
						// Same user counted twice
						b.DB.ResetCountingNumber(context.Background(), m.GuildID)
						s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("❌ <@%s> RUINED IT at %d! You can't count twice in a row. Start over at 1.", m.Author.ID, config.CurrentNumber))
						s.MessageReactionAdd(m.ChannelID, m.ID, "❌")
					}
				} else {
					// Wrong number
					b.DB.ResetCountingNumber(context.Background(), m.GuildID)
					s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("❌ <@%s> RUINED IT at %d! The next number was %d. Start over at 1.", m.Author.ID, config.CurrentNumber, expectedNumber))
					s.MessageReactionAdd(m.ChannelID, m.ID, "❌")
				}
			} else {
				// Not a number; optionally delete message (ignoring for now)
			}
		}
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

		// Custom Commands System
		// Check if message starts with "!"
		contentLowerStr := strings.ToLower(strings.TrimSpace(m.Content))
		if b.DB != nil && strings.HasPrefix(contentLowerStr, "!") {
			parts := strings.Fields(contentLowerStr)
			if len(parts) > 0 {
				cmdName := strings.TrimPrefix(parts[0], "!")
				if cmdName != "" {
					cmd, err := b.DB.GetCustomCommand(context.Background(), m.GuildID, cmdName)
					if err == nil && cmd != nil {
						_, err := s.ChannelMessageSend(m.ChannelID, cmd.Response)
						if err != nil {
							slog.Error("Failed to send custom command response", "error", err)
						}
						// If a custom command matched, we probably don't want to process auto-responders or XP
						return
					}
				}
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
							// Check for level role reward
							roleID, roleErr := b.DB.GetLevelRole(context.Background(), m.GuildID, newLevel)
							if roleErr != nil {
								slog.Error("Failed to check for level role", "error", roleErr, "guild_id", m.GuildID, "level", newLevel)
							} else if roleID != nil && *roleID != "" {
								grantErr := s.GuildMemberRoleAdd(m.GuildID, m.Author.ID, *roleID)
								if grantErr != nil {
									slog.Error("Failed to grant level role", "error", grantErr, "guild_id", m.GuildID, "user_id", m.Author.ID, "role_id", *roleID)
								} else {
									slog.Info("Granted level role", "guild_id", m.GuildID, "user_id", m.Author.ID, "role_id", *roleID, "level", newLevel)
								}
							}

							// Announce level up
							msg := fmt.Sprintf("🎉 Congratulations <@%s>, you just advanced to **Level %d**!", m.Author.ID, newLevel)
							if roleID != nil && *roleID != "" {
								msg += fmt.Sprintf(" You've been awarded the <@&%s> role!", *roleID)
							}
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

// messageUpdateHandler is called when a message is updated
func (b *Bot) messageUpdateHandler(s *discordgo.Session, m *discordgo.MessageUpdate) {
	if b.DB == nil || m.GuildID == "" {
		return
	}

	// Avoid logging bot messages or system messages (author might be nil in partial update)
	if m.Author != nil && (m.Author.ID == s.State.User.ID || m.Author.Bot) {
		return
	}

	// Message updates can be partial, so we check if there's actually content updated
	if m.BeforeUpdate != nil && m.Content == m.BeforeUpdate.Content {
		return
	}

	// For partial updates without before cache where the content didn't change (e.g. embed loads)
	// It's safer to ignore if we don't have the old content to compare against and it's missing author info
	if m.BeforeUpdate == nil && m.Author == nil {
		return
	}

	// Auto-Moderation System
	authorID := ""
	authorName := ""
	authorAvatarURL := ""
	if m.Author != nil {
		authorID = m.Author.ID
		authorName = m.Author.String()
		authorAvatarURL = m.Author.AvatarURL("")
	} else if m.BeforeUpdate != nil && m.BeforeUpdate.Author != nil {
		authorID = m.BeforeUpdate.Author.ID
		authorName = m.BeforeUpdate.Author.String()
		authorAvatarURL = m.BeforeUpdate.Author.AvatarURL("")
	}

	if b.checkAutomod(s, m.GuildID, m.ChannelID, m.ID, m.Content, authorID, authorName, authorAvatarURL) {
		return
	}

	logChannelID, err := b.DB.GetServerLogChannel(context.Background(), m.GuildID)
	if err != nil || logChannelID == "" {
		return
	}

	// Try to get author info safely
	if m.Author != nil {
		authorName = m.Author.Username
		if m.Author.GlobalName != "" {
			authorName = m.Author.GlobalName
		}
		authorID = m.Author.ID
	} else if m.BeforeUpdate != nil && m.BeforeUpdate.Author != nil {
		authorName = m.BeforeUpdate.Author.Username
		if m.BeforeUpdate.Author.GlobalName != "" {
			authorName = m.BeforeUpdate.Author.GlobalName
		}
		authorID = m.BeforeUpdate.Author.ID
	} else {
		// Can't identify author, skip
		return
	}

	beforeContent := "*(Content not available in cache)*"
	if m.BeforeUpdate != nil {
		beforeContent = m.BeforeUpdate.Content
		if beforeContent == "" {
			beforeContent = "*(Empty message or only attachment)*"
		}
	}

	afterContent := m.Content
	if afterContent == "" {
		afterContent = "*(Empty message or only attachment)*"
	}

	if b.DB != nil && beforeContent != "*(Content not available in cache)*" && beforeContent != "*(Empty message or only attachment)*" && afterContent != "*(Empty message or only attachment)*" && authorID != "" {
		b.DB.AddEditSnipe(context.Background(), m.ChannelID, beforeContent, afterContent, authorID)
	}

	embed := &discordgo.MessageEmbed{
		Title:       "Message Edited",
		Description: fmt.Sprintf("**Message by <@%s> edited in <#%s>**\n\n[Jump to Message](https://discord.com/channels/%s/%s/%s)", authorID, m.ChannelID, m.GuildID, m.ChannelID, m.ID),
		Color:       0xFFA500, // Orange
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "Before",
				Value:  beforeContent,
				Inline: false,
			},
			{
				Name:   "After",
				Value:  afterContent,
				Inline: false,
			},
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: fmt.Sprintf("User ID: %s | Message ID: %s", authorID, m.ID),
		},
		Timestamp: time.Now().Format(time.RFC3339),
	}

	if m.Author != nil {
		embed.Author = &discordgo.MessageEmbedAuthor{
			Name:    authorName,
			IconURL: m.Author.AvatarURL(""),
		}
	}

	_, err = s.ChannelMessageSendEmbed(logChannelID, embed)
	if err != nil {
		slog.Error("Failed to send message update log", "error", err, "channel_id", logChannelID)
	}
}

// messageDeleteHandler is called when a message is deleted
func (b *Bot) messageDeleteHandler(s *discordgo.Session, m *discordgo.MessageDelete) {
	if b.DB == nil || m.GuildID == "" {
		return
	}

	logChannelID, err := b.DB.GetServerLogChannel(context.Background(), m.GuildID)
	if err != nil || logChannelID == "" {
		return
	}

	var authorName, authorID, content string
	var authorAvatarURL string

	if m.BeforeDelete != nil && m.BeforeDelete.Author != nil {
		// Ignore bot deletions if we have the info
		if m.BeforeDelete.Author.ID == s.State.User.ID || m.BeforeDelete.Author.Bot {
			return
		}
		authorName = m.BeforeDelete.Author.Username
		if m.BeforeDelete.Author.GlobalName != "" {
			authorName = m.BeforeDelete.Author.GlobalName
		}
		authorID = m.BeforeDelete.Author.ID
		authorAvatarURL = m.BeforeDelete.Author.AvatarURL("")
		content = m.BeforeDelete.Content
		if content == "" {
			content = "*(Empty message or only attachment)*"
		}
	} else {
		// Without message cache, we don't know who sent it or what it was
		authorName = "Unknown User"
		authorID = "Unknown"
		content = "*(Message content not available in cache)*"
	}

	if b.DB != nil && content != "" && content != "*(Message content not available in cache)*" && content != "*(Empty message or only attachment)*" && authorID != "" && authorID != "Unknown" {
		b.DB.AddSnipe(context.Background(), m.ChannelID, content, authorID)
	}

	embed := &discordgo.MessageEmbed{
		Title:       "Message Deleted",
		Description: fmt.Sprintf("**Message sent by <@%s> deleted in <#%s>**\n\n%s", authorID, m.ChannelID, content),
		Color:       0xFF0000, // Red
		Footer: &discordgo.MessageEmbedFooter{
			Text: fmt.Sprintf("User ID: %s | Message ID: %s", authorID, m.ID),
		},
		Timestamp: time.Now().Format(time.RFC3339),
	}

	if authorAvatarURL != "" {
		embed.Author = &discordgo.MessageEmbedAuthor{
			Name:    authorName,
			IconURL: authorAvatarURL,
		}
	} else {
		embed.Author = &discordgo.MessageEmbedAuthor{
			Name: authorName,
		}
	}

	_, err = s.ChannelMessageSendEmbed(logChannelID, embed)
	if err != nil {
		slog.Error("Failed to send message delete log", "error", err, "channel_id", logChannelID)
	}
}

// GetDB returns the database instance for commands to use.
func (b *Bot) GetDB() *db.DB {
	return b.DB
}

// checkAutomod verifies if a message violates automod rules.
// Returns true if the message was deleted.
func (b *Bot) checkAutomod(s *discordgo.Session, guildID, channelID, messageID, content, authorID, authorName, authorIconURL string) bool {
	if b.DB == nil || guildID == "" {
		return false
	}

	config, err := b.DB.GetAutomodConfig(context.Background(), guildID)
	if err != nil || config == nil {
		return false
	}

	contentLower := strings.ToLower(content)
	violation := ""

	// Check links
	if config.FilterLinks && (strings.Contains(contentLower, "http://") || strings.Contains(contentLower, "https://")) {
		violation = "Unauthorized Link"
	}

	// Check invites
	if violation == "" && config.FilterInvites && (strings.Contains(contentLower, "discord.gg/") || strings.Contains(contentLower, "discord.com/invite/")) {
		violation = "Discord Invite"
	}

	// Check bad words
	if violation == "" {
		words, err := b.DB.GetAutomodWords(context.Background(), guildID)
		if err == nil {
			for _, word := range words {
				if strings.Contains(contentLower, strings.ToLower(word)) {
					violation = fmt.Sprintf("Restricted Word (`%s`)", word)
					break
				}
			}
		}
	}

	if violation != "" {
		err := s.ChannelMessageDelete(channelID, messageID)
		if err != nil {
			slog.Error("Failed to delete automod message", "message_id", messageID, "error", err)
			return false
		}

		if config.LogChannelID != "" {
			embed := &discordgo.MessageEmbed{
				Title:       "Auto-Moderation Action",
				Description: fmt.Sprintf("**Message deleted in <#%s>**\n\n**Reason:** %s\n**Content:** %s", channelID, violation, content),
				Color:       0xFF0000,
				Author: &discordgo.MessageEmbedAuthor{
					Name:    authorName,
					IconURL: authorIconURL,
				},
				Footer: &discordgo.MessageEmbedFooter{
					Text: fmt.Sprintf("User ID: %s", authorID),
				},
				Timestamp: time.Now().Format(time.RFC3339),
			}
			_, _ = s.ChannelMessageSendEmbed(config.LogChannelID, embed)
		}
		return true
	}

	return false
}

// voiceStateUpdateHandler tracks voice joins, leaves, and moves and logs them.
func (b *Bot) voiceStateUpdateHandler(s *discordgo.Session, v *discordgo.VoiceStateUpdate) {
	if b.DB == nil || v.GuildID == "" {
		return
	}

	// Filter out state updates that don't involve channel changes (like mute/deafen)
	// We only care if the ChannelID changed.
	// v.BeforeUpdate can be nil if the user just joined a voice channel.
	var oldChannelID string
	if v.BeforeUpdate != nil {
		oldChannelID = v.BeforeUpdate.ChannelID
	}

	newChannelID := v.ChannelID

	// If the channel ID didn't change, it's a mute/deafen or similar state update
	if oldChannelID == newChannelID {
		return
	}

	// Fetch user details. Try v.Member.User first, fallback to state cache/API
	var user *discordgo.User
	if v.Member != nil && v.Member.User != nil {
		user = v.Member.User
	} else {
		// Fallback (might cause API call if not cached, but necessary if Member is nil)
		member, err := s.GuildMember(v.GuildID, v.UserID)
		if err == nil && member != nil && member.User != nil {
			user = member.User
		}
	}

	// Temporary Voice Channels Logic
	if newChannelID != "" {
		tempConfig, err := b.DB.GetTempVoiceConfig(context.Background(), v.GuildID)
		if err == nil && tempConfig != nil && tempConfig.TriggerChannelID == newChannelID {
			// User joined the trigger channel, create a new temp channel
			var name string
			if user != nil {
				name = user.Username + "'s Channel"
			} else {
				name = "Temp Channel"
			}

			channelData := discordgo.GuildChannelCreateData{
				Name:     name,
				Type:     discordgo.ChannelTypeGuildVoice,
				ParentID: tempConfig.CategoryID,
			}

			createdChannel, err := s.GuildChannelCreateComplex(v.GuildID, channelData)
			if err != nil {
				slog.Error("Failed to create temporary voice channel", "error", err)
			} else {
				// Move the user to the new channel
				err = s.GuildMemberMove(v.GuildID, v.UserID, &createdChannel.ID)
				if err != nil {
					slog.Error("Failed to move user to temporary voice channel", "error", err)
					// Clean up the channel if we can't move the user to it
					_, _ = s.ChannelDelete(createdChannel.ID)
				} else {
					// Save to database
					err = b.DB.CreateTempVoiceChannel(context.Background(), v.GuildID, v.UserID, createdChannel.ID)
					if err != nil {
						slog.Error("Failed to save temporary voice channel to DB", "error", err)
					}
				}
			}
		}
	}

	if oldChannelID != "" {
		// Check if the old channel was a temporary voice channel
		tempChannel, err := b.DB.GetTempVoiceChannel(context.Background(), oldChannelID)
		if err == nil && tempChannel != nil {
			// Check if the channel is now empty
			guild, err := s.State.Guild(v.GuildID)
			if err == nil {
				isEmpty := true
				for _, vs := range guild.VoiceStates {
					if vs.ChannelID == oldChannelID {
						isEmpty = false
						break
					}
				}

				if isEmpty {
					// Delete the channel from Discord
					_, err := s.ChannelDelete(oldChannelID)
					if err != nil {
						slog.Error("Failed to delete temporary voice channel", "error", err)
					}
					// Delete from DB regardless of discord deletion success to prevent zombie records
					_ = b.DB.DeleteTempVoiceChannel(context.Background(), oldChannelID)
				}
			}
		}
	}

	channelIDStr, err := b.DB.GetVoiceLogChannel(context.Background(), v.GuildID)
	if err != nil {
		slog.Error("Failed to get voice log config", "guild_id", v.GuildID, "error", err)
		return
	}

	if channelIDStr == nil {
		return // Voice logging not configured for this guild
	}

	if user == nil {
		slog.Warn("Could not determine user for voice state update", "user_id", v.UserID)
		return
	}

	var title string
	var description string
	var color int

	if oldChannelID == "" && newChannelID != "" {
		// Joined
		title = "🎙️ Voice Join"
		description = fmt.Sprintf("<@%s> joined voice channel <#%s>", v.UserID, newChannelID)
		color = 0x00FF00 // Green
	} else if oldChannelID != "" && newChannelID == "" {
		// Left
		title = "🎙️ Voice Leave"
		description = fmt.Sprintf("<@%s> left voice channel <#%s>", v.UserID, oldChannelID)
		color = 0xFF0000 // Red
	} else if oldChannelID != "" && newChannelID != "" {
		// Moved
		title = "🎙️ Voice Move"
		description = fmt.Sprintf("<@%s> moved from <#%s> to <#%s>", v.UserID, oldChannelID, newChannelID)
		color = 0xFFA500 // Orange
	} else {
		return
	}

	embed := &discordgo.MessageEmbed{
		Title:       title,
		Description: description,
		Color:       color,
		Author: &discordgo.MessageEmbedAuthor{
			Name:    user.String(),
			IconURL: user.AvatarURL(""),
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: fmt.Sprintf("User ID: %s", v.UserID),
		},
	}

	_, err = s.ChannelMessageSendEmbed(*channelIDStr, embed)
	if err != nil {
		slog.Error("Failed to send voice log embed", "channel_id", *channelIDStr, "error", err)
	}
}

// checkBirthdays runs daily (or hourly to catch all timezones in a real app, but minutely for this loop context)
// to announce birthdays for the current day.
func (b *Bot) checkBirthdays() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		if b.DB == nil {
			continue
		}

		now := time.Now()
		month := int(now.Month())
		day := now.Day()
		year := now.Year()

		birthdays, err := b.DB.GetDueBirthdays(context.Background(), month, day, year)
		if err != nil {
			slog.Error("Failed to fetch due birthdays", "error", err)
			continue
		}

		for _, bday := range birthdays {
			channelID, err := b.DB.GetBirthdayChannel(context.Background(), bday.GuildID)
			if err == nil && channelID != "" {
				// Announce birthday
				embed := &discordgo.MessageEmbed{
					Title:       "🎉 Happy Birthday! 🎉",
					Description: fmt.Sprintf("Wishing a very happy birthday to <@%s>! 🎂🎈", bday.UserID),
					Color:       0xFF1493, // Deep Pink
				}

				_, err = b.Session.ChannelMessageSendEmbed(channelID, embed)
				if err != nil {
					slog.Error("Failed to send birthday announcement", "error", err)
				}
			}

			// Mark as announced for the year
			err = b.DB.MarkBirthdayAnnounced(context.Background(), bday.GuildID, bday.UserID, year)
			if err != nil {
				slog.Error("Failed to mark birthday as announced", "error", err)
			}
		}
	}
}

// handleModmailDM processes direct messages sent to the bot, routing them to the appropriate ModMail thread.
func (b *Bot) handleModmailDM(s *discordgo.Session, m *discordgo.MessageCreate) {
	if b.DB == nil {
		return
	}

	thread, err := b.DB.GetOpenModmailThreadByUser(context.Background(), m.Author.ID)
	if err != nil {
		slog.Error("Failed to check for open modmail thread", "error", err)
		return
	}

	if thread != nil {
		// Existing thread found, forward message
		b.forwardModmailMessageToThread(s, m, thread)
		return
	}

	// Try to find a guild where both user and bot are present, and ModMail is configured
	// For simplicity in this implementation, we will check all guilds the bot is in.
	var targetGuildID string
	var config *db.ModmailConfig

	// Fetch guilds from state (might be incomplete if bot is in many guilds, but sufficient for this scale)
	for _, g := range s.State.Guilds {
		// Only check guilds where the user is known to be in state to avoid API spam
		member, err := s.State.Member(g.ID, m.Author.ID)
		if err == nil && member != nil {
			// User is in this guild, check if modmail is configured
			cfg, err := b.DB.GetModmailConfig(context.Background(), g.ID)
			if err == nil && cfg != nil && cfg.CategoryID != "" {
				targetGuildID = g.ID
				config = cfg
				break
			}
		}
	}

	if targetGuildID == "" || config == nil {
		s.ChannelMessageSend(m.ChannelID, "I couldn't find a server where ModMail is configured that we share.")
		return
	}

	// Sanitize username for channel name (lowercase, no spaces, no special chars)
	sanitizedName := strings.ToLower(m.Author.Username)
	sanitizedName = strings.ReplaceAll(sanitizedName, " ", "-")
	sanitizedName = strings.ReplaceAll(sanitizedName, ".", "")
	sanitizedName = strings.ReplaceAll(sanitizedName, "#", "")

	// Create new thread channel
	channelName := fmt.Sprintf("modmail-%s", sanitizedName)
	newChannel, err := s.GuildChannelCreateComplex(targetGuildID, discordgo.GuildChannelCreateData{
		Name:     channelName,
		Type:     discordgo.ChannelTypeGuildText,
		ParentID: config.CategoryID,
	})

	if err != nil {
		s.ChannelMessageSend(m.ChannelID, "Failed to create ModMail thread channel on the server.")
		slog.Error("Failed to create modmail channel", "error", err)
		return
	}

	// Save to DB
	err = b.DB.CreateModmailThread(context.Background(), targetGuildID, m.Author.ID, newChannel.ID)
	if err != nil {
		s.ChannelMessageSend(m.ChannelID, "Failed to register ModMail thread.")
		slog.Error("Failed to save modmail thread", "error", err)
		return
	}

	// Fetch thread with ID
	thread, _ = b.DB.GetOpenModmailThreadByUser(context.Background(), m.Author.ID)

	// Send initial message to the thread
	s.ChannelMessageSendEmbed(newChannel.ID, &discordgo.MessageEmbed{
		Title:       "New ModMail Thread",
		Description: fmt.Sprintf("User: <@%s> (%s)", m.Author.ID, m.Author.Username),
		Color:       0x00FF00,
	})

	// Forward the actual message
	if thread != nil {
		b.forwardModmailMessageToThread(s, m, thread)
		s.ChannelMessageSend(m.ChannelID, "Message sent to the server moderators.")
	}
}

func (b *Bot) forwardModmailMessageToThread(s *discordgo.Session, m *discordgo.MessageCreate, thread *db.ModmailThread) {
	content := m.Content
	if content == "" && len(m.Attachments) == 0 {
		return
	}

	embed := &discordgo.MessageEmbed{
		Author: &discordgo.MessageEmbedAuthor{
			Name:    m.Author.String(),
			IconURL: m.Author.AvatarURL(""),
		},
		Description: content,
		Color:       0x0000FF, // Blue for user messages
		Footer: &discordgo.MessageEmbedFooter{
			Text: fmt.Sprintf("User ID: %s", m.Author.ID),
		},
	}

	if len(m.Attachments) > 0 {
		embed.Image = &discordgo.MessageEmbedImage{
			URL: m.Attachments[0].URL,
		}
	}

	s.ChannelMessageSendEmbed(thread.ChannelID, embed)
}

func (b *Bot) handleModmailReply(s *discordgo.Session, m *discordgo.MessageCreate, thread *db.ModmailThread) {
	// Don't forward commands
	if strings.HasPrefix(m.Content, "/") || strings.HasPrefix(m.Content, "!") {
		return
	}

	dmChannel, err := s.UserChannelCreate(thread.UserID)
	if err != nil {
		s.ChannelMessageSend(m.ChannelID, "Failed to open DM with the user. They may have DMs disabled.")
		return
	}

	content := m.Content
	embed := &discordgo.MessageEmbed{
		Author: &discordgo.MessageEmbedAuthor{
			Name:    m.Author.String(),
			IconURL: m.Author.AvatarURL(""),
		},
		Description: content,
		Color:       0xFF0000, // Red for mod replies
		Footer: &discordgo.MessageEmbedFooter{
			Text: "Moderator Reply",
		},
	}

	if len(m.Attachments) > 0 {
		embed.Image = &discordgo.MessageEmbedImage{
			URL: m.Attachments[0].URL,
		}
	}

	_, err = s.ChannelMessageSendEmbed(dmChannel.ID, embed)
	if err != nil {
		s.ChannelMessageSend(m.ChannelID, "Failed to send message to user. They may have blocked the bot or disabled DMs.")
	} else {
		// Confirm sent in the thread channel
		s.MessageReactionAdd(m.ChannelID, m.ID, "✅")
	}
}
