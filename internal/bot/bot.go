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
	Session                *discordgo.Session
	Config                 *config.Config
	Registry               *commands.Registry
	DB                     *db.DB
	xpCooldown             sync.Map // map[string]time.Time (key: guildID_channelID_userID)
	AutoResponders         sync.Map // map[string][]*db.AutoResponder (key: guildID)
	antiSpamTracking       sync.Map // map[string][]time.Time (key: guildID_userID)
	generatedVoiceChannels sync.Map // map[string]bool (key: channelID)
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
	registry.Add(commands.WarnAutoMod(database))
	registry.Add(commands.Warnings(database))
	registry.Add(commands.Kick(database))
	registry.Add(commands.Ban(database))
	registry.Add(commands.Purge(database))
	registry.Add(commands.Rank(database))
	registry.Add(commands.Leaderboard(database))
	registry.Add(commands.Daily(database))
	registry.Add(commands.Coins(database))
	registry.Add(commands.Config(database))
	registry.Add(commands.Settings(database))
	registry.Add(commands.VoiceGen(database))
	registry.Add(commands.VoiceLog(database))
	registry.Add(commands.AutoPublish(database))
	registry.Add(commands.ReactionRole(database))
	registry.Add(commands.Schedule(database))
	registry.Add(commands.Changelog())
	registry.Add(commands.Remind(database))
	registry.Add(commands.Ticket(database))
	registry.Add(commands.Tag(database))
	registry.Add(commands.Giveaway(database))
	registry.Add(commands.AFKCommand(database))
	registry.Add(commands.Quote(database))
	registry.Add(commands.Userinfo(database))
	registry.Add(commands.Roleinfo(database))
	registry.Add(commands.Serverinfo(database))
	registry.Add(commands.Avatar(database))
	registry.Add(commands.BookmarkContext(database))
	registry.Add(commands.BookmarksSlash(database))
	registry.Add(commands.Timezone(database))
	registry.Add(commands.Thread(database))

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
	registry.Add(commands.DynamicVoice(database))
	registry.Add(commands.NewCountingCommand(database))
	registry.Add(commands.NewTriviaCommand(database))
	registry.Add(commands.CustomCommand(database))
	registry.Add(commands.NewMemberCountCommand(database))
	registry.Add(commands.Snipe(database))
	registry.Add(commands.EditSnipe(database))
	registry.Add(commands.Gamble(database))
	registry.Add(commands.Confession(database))
	registry.Add(commands.Confess(database))
	registry.Add(commands.Todo(database))
	registry.Add(commands.RoleMenu(database))
	registry.Add(commands.Music(database))
	registry.Add(commands.Play(database))
	registry.Add(commands.Skip(database))
	registry.Add(commands.Stop(database))
	registry.Add(commands.EightBall())
	registry.Add(commands.Roll())
	registry.Add(commands.RPS())
	registry.Add(commands.Report(database))
	registry.Add(commands.Welcome(database))
	registry.Add(commands.WelcomeDM(database))
	registry.Add(commands.Goodbye(database))
	registry.Add(commands.Autorole(database))
	registry.Add(commands.MediaChannel(database))
	registry.Add(commands.Badge(database))
	registry.Add(commands.Transfer(database))
	registry.Add(commands.Transfers(database))
	registry.Add(commands.Baltop(database))
	registry.Add(commands.Mute(database))
	registry.Add(commands.Unmute(database))
	registry.Add(commands.Work(database))
	registry.Add(commands.Crime(database))
	registry.Add(commands.Rob(database))
	registry.Add(commands.Use(database))
	registry.Add(commands.Bank(database))
	registry.Add(commands.Pet(database))
	registry.Add(commands.Job(database))
	registry.Add(commands.Prefix(database))
	registry.Add(commands.AutoThread(database))
	registry.Add(commands.RepLB(database))
	registry.Add(commands.LevelLB(database))
	registry.Add(commands.FactCommand(database))
	registry.Add(commands.MyRoleCommand(database))
	registry.Add(commands.HighlightCommand(database))
	registry.Add(commands.NickTemplate(database))
	registry.Add(commands.Unban(database))
	registry.Add(commands.ClearWarnings(database))
	registry.Add(commands.LevelBlacklist(database))
	registry.Add(commands.LevelChannelBlacklist(database))
	registry.Add(commands.Lock(database))
	registry.Add(commands.AntiSpam(database))
	registry.Add(commands.AdvancedLog(database))
	registry.Add(commands.Unlock(database))
	registry.Add(commands.Slowmode(database))
	registry.Add(commands.Cooldown(database))
	registry.Add(commands.ReactionGroup(database))
	registry.Add(commands.Role(database))
	registry.Add(commands.ReactionMenu(database))
	registry.Add(commands.StickyRole(database))
	registry.Add(commands.TempRole(database))
	registry.Add(commands.Snippet(database))
	registry.Add(commands.Translate(database))
	registry.Add(commands.ThreadAuto(database))
	registry.Add(commands.Keyword(database))
	registry.Add(commands.ReactionTrigger(database))
	registry.Add(commands.TempNick(database))

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

	// Register member remove handler
	bot.Session.AddHandler(bot.guildMemberRemoveHandler)
	bot.Session.AddHandler(bot.channelCreateHandler)
	bot.Session.AddHandler(bot.channelDeleteHandler)
	bot.Session.AddHandler(bot.guildRoleCreateHandler)
	bot.Session.AddHandler(bot.guildRoleDeleteHandler)
	bot.Session.AddHandler(bot.threadCreateHandler)

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

func (b *Bot) threadCreateHandler(s *discordgo.Session, t *discordgo.ThreadCreate) {
	if b.DB == nil {
		return
	}

	// We only want to join automatically when the thread is newly created
	if t.NewlyCreated {
		ctx := context.Background()
		autoJoin, err := b.DB.GetThreadAutomation(ctx, t.GuildID, t.ParentID)
		if err == nil && autoJoin {
			err = s.ThreadJoin(t.ID)
			if err != nil {
				slog.Error("Failed to auto-join thread", "thread_id", t.ID, "guild_id", t.GuildID, "error", err)
			} else {
				slog.Info("Automatically joined thread", "thread_id", t.ID, "guild_id", t.GuildID)
			}
		}
	}
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

	go b.memberCountLoop()

	// Start scheduled announcements checker
	go b.checkScheduledAnnouncements()

	// Start reminder delivery checker
	go b.checkReminders()

	// Start giveaway checker
	go b.checkGiveaways()
	go b.checkBirthdays()
	go b.checkTempBans()
	go b.checkTempRoles()
	go b.checkTempNicknames()

	// Start mute expiration checker
	go b.checkExpiredMutes()

	// Start daily interest application loop
	go b.applyInterestLoop()

	// Start pet stats loop
	go b.petStatsLoop()
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
		if strings.HasPrefix(i.MessageComponentData().CustomID, "snooze_") {
			parts := strings.Split(i.MessageComponentData().CustomID, "_")
			if len(parts) == 3 {
				userID := parts[1]
				reminderIDStr := parts[2]

				var actingUserID string
				if i.Member != nil {
					actingUserID = i.Member.User.ID
				} else {
					actingUserID = i.User.ID
				}

				if userID != actingUserID {
					_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
						Type: discordgo.InteractionResponseChannelMessageWithSource,
						Data: &discordgo.InteractionResponseData{
							Content: "You cannot snooze someone else's reminder.",
							Flags:   discordgo.MessageFlagsEphemeral,
						},
					})
					return
				}

				reminderID, err := strconv.Atoi(reminderIDStr)
				if err != nil {
					slog.Error("Failed to parse reminder ID", "error", err)
					return
				}

				err = b.DB.SnoozeReminder(context.Background(), reminderID, 10*time.Minute)
				if err != nil {
					slog.Error("Failed to snooze reminder", "id", reminderID, "error", err)
					return
				}

				_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseUpdateMessage,
					Data: &discordgo.InteractionResponseData{
						Content:    i.Message.Content + "\n\n*Snoozed for 10 minutes.*",
						Components: []discordgo.MessageComponent{}, // Remove the button
					},
				})
			}
			return
		}

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

			// Check custom command cooldown
			expiresAt, err := b.DB.GetCommandCooldown(context.Background(), user.ID, commandName)
			if err != nil {
				slog.Error("Failed to check command cooldown", "error", err)
			} else if !expiresAt.IsZero() && expiresAt.After(time.Now()) {
				durationStr := expiresAt.Sub(time.Now()).Round(time.Second).String()
				_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: fmt.Sprintf("You are on cooldown for `/%s`. Try again in %s.", commandName, durationStr),
						Flags:   discordgo.MessageFlagsEphemeral,
					},
				})
				return
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

	// Auto-Delete System
	if b.DB != nil && m.GuildID != "" {
		deleteAfter, err := b.DB.GetAutoDelete(context.Background(), m.GuildID, m.ChannelID)
		if err == nil && deleteAfter > 0 {
			go func(cID, mID string, wait int) {
				time.Sleep(time.Duration(wait) * time.Second)
				s.ChannelMessageDelete(cID, mID)
			}(m.ChannelID, m.ID, deleteAfter)
		}
	}

	// Message Forwarding System
	if b.DB != nil && m.GuildID != "" && m.Type != discordgo.MessageTypeReply {
		targets, err := b.DB.GetForwardingRules(context.Background(), m.GuildID, m.ChannelID)
		if err == nil && len(targets) > 0 {
			for _, targetID := range targets {
				embed := &discordgo.MessageEmbed{
					Author: &discordgo.MessageEmbedAuthor{
						Name:    m.Author.Username,
						IconURL: m.Author.AvatarURL(""),
					},
					Description: m.Content,
					Color:       0x00BFFF,
					Footer: &discordgo.MessageEmbedFooter{
						Text: fmt.Sprintf("Forwarded from #%s", m.ChannelID),
					},
				}
				if len(m.Attachments) > 0 {
					embed.Image = &discordgo.MessageEmbedImage{
						URL: m.Attachments[0].URL,
					}
				}
				go func(tID string, emb *discordgo.MessageEmbed) {
					_, err := s.ChannelMessageSendEmbed(tID, emb)
					if err != nil {
						slog.Error("Failed to forward message", "target", tID, "error", err)
					}
				}(targetID, embed)
			}
		}
	}

	// Reaction Triggers System
	if b.DB != nil && m.GuildID != "" {
		triggers, err := b.DB.GetReactionTriggers(context.Background(), m.GuildID)
		if err == nil && len(triggers) > 0 {
			msgContentLower := strings.ToLower(m.Content)
			for _, t := range triggers {
				if strings.Contains(msgContentLower, strings.ToLower(t.Keyword)) {
					emoji := t.Emoji
					// Handle custom emojis strictly as name:id or a:name:id
					if strings.HasPrefix(emoji, "<:") && strings.HasSuffix(emoji, ">") {
						emoji = strings.TrimPrefix(emoji, "<:")
						emoji = strings.TrimSuffix(emoji, ">")
					} else if strings.HasPrefix(emoji, "<a:") && strings.HasSuffix(emoji, ">") {
						emoji = strings.TrimPrefix(emoji, "<a:")
						emoji = strings.TrimSuffix(emoji, ">")
					}
					s.MessageReactionAdd(m.ChannelID, m.ID, emoji)
				}
			}
		}
	}

	// Keyword Notification System
	if b.DB != nil && m.GuildID != "" {
		notifs, err := b.DB.GetKeywordNotifications(context.Background(), m.GuildID)
		if err == nil && len(notifs) > 0 {
			msgContentLower := strings.ToLower(m.Content)
			notifiedUsers := make(map[string]bool)

			for _, n := range notifs {
				if n.UserID == m.Author.ID {
					continue // Don't notify the author
				}
				if notifiedUsers[n.UserID] {
					continue // Already notified for this message
				}

				if strings.Contains(msgContentLower, n.Keyword) {
					notifiedUsers[n.UserID] = true

					go func(userID, keyword string) {
						channel, err := s.UserChannelCreate(userID)
						if err != nil {
							return
						}

						link := fmt.Sprintf("https://discord.com/channels/%s/%s/%s", m.GuildID, m.ChannelID, m.ID)
						content := fmt.Sprintf("Your keyword **%s** was mentioned in <#%s> by **%s**.\n\n[Jump to message](%s)", keyword, m.ChannelID, m.Author.Username, link)

						s.ChannelMessageSend(channel.ID, content)
					}(n.UserID, n.Keyword)
				}
			}
		}
	}

	// Auto-Publish System
	if b.DB != nil && m.GuildID != "" {
		isAutoPublish, err := b.DB.IsAutoPublishChannel(context.Background(), m.GuildID, m.ChannelID)
		if err == nil && isAutoPublish {
			go func() {
				_, err := s.ChannelMessageCrosspost(m.ChannelID, m.ID)
				if err != nil {
					slog.Error("Failed to auto-publish message", "channel_id", m.ChannelID, "message_id", m.ID, "error", err)
				} else {
					slog.Info("Auto-published message", "channel_id", m.ChannelID, "message_id", m.ID)
				}
			}()
		}
	}

	// Auto-Moderation System
	if b.checkAutomod(s, m.GuildID, m.ChannelID, m.ID, m.Content, m.Author.ID, m.Author.String(), m.Author.AvatarURL("")) {
		return
	}

	// Thread Auto-Archive Duration Enforcement
	if b.DB != nil && m.GuildID != "" && m.ChannelID != "" {
		channel, err := s.State.Channel(m.ChannelID)
		if err != nil {
			channel, err = s.Channel(m.ChannelID)
		}

		if err == nil && (channel.Type == discordgo.ChannelTypeGuildPublicThread || channel.Type == discordgo.ChannelTypeGuildPrivateThread || channel.Type == discordgo.ChannelTypeGuildNewsThread) {
			cfg, cfgErr := b.DB.GetThreadConfig(context.Background(), m.GuildID)
			if cfgErr == nil && cfg != nil {
				if channel.ThreadMetadata != nil && channel.ThreadMetadata.AutoArchiveDuration != cfg.AutoArchiveDuration {
					_, editErr := s.ChannelEdit(m.ChannelID, &discordgo.ChannelEdit{
						AutoArchiveDuration: cfg.AutoArchiveDuration,
					})
					if editErr == nil {
						// Update state cache to prevent repeated API calls
						channel.ThreadMetadata.AutoArchiveDuration = cfg.AutoArchiveDuration
					}
				}
			}
		}
	}

	// Advanced Anti-Spam System
	if b.DB != nil && m.GuildID != "" {
		antiSpamCfg, err := b.DB.GetAntiSpamConfig(context.Background(), m.GuildID)
		if err == nil && antiSpamCfg != nil {
			trackKey := fmt.Sprintf("%s_%s", m.GuildID, m.Author.ID)
			now := time.Now()

			var timestamps []time.Time
			val, ok := b.antiSpamTracking.Load(trackKey)
			if ok {
				timestamps = val.([]time.Time)
			}

			// Filter out timestamps older than the window
			windowStart := now.Add(-time.Duration(antiSpamCfg.TimeWindow) * time.Second)
			var validTimestamps []time.Time
			for _, t := range timestamps {
				if t.After(windowStart) {
					validTimestamps = append(validTimestamps, t)
				}
			}

			validTimestamps = append(validTimestamps, now)

			if len(validTimestamps) > antiSpamCfg.MessageLimit {
				// Spam detected
				b.antiSpamTracking.Delete(trackKey) // Clear tracking to avoid repeated mutes

				muteDur, err := time.ParseDuration(antiSpamCfg.MuteDuration)
				if err == nil {
					until := now.Add(muteDur)

					// Apply mute via Discord API
					err = s.GuildMemberTimeout(m.GuildID, m.Author.ID, &until)
					if err == nil {
						// Log mute in database
						_ = b.DB.AddMute(context.Background(), m.GuildID, m.Author.ID, s.State.User.ID, "Auto-Mute: Spamming", until)

						s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("⚠️ <@%s> has been muted for **%s** for spamming.", m.Author.ID, antiSpamCfg.MuteDuration))

						// Optionally post to mod log
						logChan, err := b.DB.GetGuildLogChannel(context.Background(), m.GuildID)
						if err == nil && logChan != "" {
							embed := &discordgo.MessageEmbed{
								Title:       "User Auto-Muted (Anti-Spam)",
								Color:       0xFFA500,
								Description: fmt.Sprintf("**User:** <@%s>\n**Reason:** Spamming in <#%s>\n**Duration:** %s", m.Author.ID, m.ChannelID, antiSpamCfg.MuteDuration),
								Timestamp:   now.Format(time.RFC3339),
							}
							s.ChannelMessageSendEmbed(logChan, embed)
						}
					} else {
						slog.Error("Failed to timeout user for spam", "guild", m.GuildID, "user", m.Author.ID, "error", err)
					}
				}
			} else {
				b.antiSpamTracking.Store(trackKey, validTimestamps)
			}
		}
	}

	// Auto-Threads System
	if b.DB != nil && m.GuildID != "" {
		config, err := b.DB.GetAutoThread(context.Background(), m.GuildID, m.ChannelID)
		if err == nil && config != nil {
			threadName := strings.ReplaceAll(config.ThreadNameTemplate, "{user}", m.Author.Username)
			if len(threadName) > 100 {
				threadName = threadName[:97] + "..."
			}
			_, err = s.MessageThreadStartComplex(m.ChannelID, m.ID, &discordgo.ThreadStart{
				Name:                threadName,
				AutoArchiveDuration: 60,
			})
			if err != nil {
				slog.Error("Failed to create auto-thread", "error", err, "guild_id", m.GuildID, "channel_id", m.ChannelID)
			}
		}
	}

	// Media-Only Channels System
	if b.DB != nil && m.GuildID != "" {
		isMedia, err := b.DB.IsMediaChannel(context.Background(), m.GuildID, m.ChannelID)
		if err == nil && isMedia {
			hasAttachment := len(m.Attachments) > 0
			hasURL := strings.Contains(m.Content, "http://") || strings.Contains(m.Content, "https://")
			if !hasAttachment && !hasURL {
				_ = s.ChannelMessageDelete(m.ChannelID, m.ID)
				return
			}
		}
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
		// Get custom prefix
		prefix := "!"
		if b.DB != nil && m.GuildID != "" {
			p, err := b.DB.GetGuildPrefix(context.Background(), m.GuildID)
			if err == nil && p != "" {
				prefix = p
			}
		}

		// Check if message starts with the prefix
		contentLowerStr := strings.ToLower(strings.TrimSpace(m.Content))
		prefixLower := strings.ToLower(prefix)
		if b.DB != nil && strings.HasPrefix(contentLowerStr, prefixLower) {
			parts := strings.Fields(contentLowerStr)
			if len(parts) > 0 {
				cmdName := strings.TrimPrefix(parts[0], prefixLower)
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
				matched := false
				if r.IsRegex && r.CompiledReg != nil {
					matched = r.CompiledReg.MatchString(m.Content)
				} else if !r.IsRegex {
					matched = strings.Contains(contentLower, r.TriggerWord)
				}

				if matched {
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
			// Check if channel is blacklisted
			isChannelBlacklisted := false
			blacklistedChannels, err := b.DB.GetLevelingChannelBlacklists(context.Background(), m.GuildID)
			if err == nil {
				for _, channelID := range blacklistedChannels {
					if m.ChannelID == channelID {
						isChannelBlacklisted = true
						break
					}
				}
			}

			// Check if user has a blacklisted role
			hasBlacklistedRole := false
			if m.Member != nil {
				blacklistedRoles, err := b.DB.GetLevelingBlacklists(context.Background(), m.GuildID)
				if err == nil && len(blacklistedRoles) > 0 {
					blacklistMap := make(map[string]bool)
					for _, br := range blacklistedRoles {
						blacklistMap[br] = true
					}
					for _, roleID := range m.Member.Roles {
						if blacklistMap[roleID] {
							hasBlacklistedRole = true
							break
						}
					}
				}
			}

			if !isChannelBlacklisted && !hasBlacklistedRole {
				// Award XP (e.g., random 15-25 XP)
				amount := rand.Intn(11) + 15
				multiplier := 1.0
				if m.Member != nil && len(m.Member.Roles) > 0 {
					multipliers, err := b.DB.GetLevelMultipliers(context.Background(), m.GuildID)
					if err == nil {
						for _, roleID := range m.Member.Roles {
							if val, ok := multipliers[roleID]; ok && val > multiplier {
								multiplier = val
							}
						}
					}
				}
				amount = int(math.Round(float64(amount) * multiplier))

				// Ensure user exists first
				err = b.DB.UpsertUser(context.Background(), m.Author.ID, m.Author.Username, m.Author.GlobalName, m.Author.AvatarURL(""))
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
								roleID, coinsReward, roleErr := b.DB.GetLevelRole(context.Background(), m.GuildID, newLevel)
								if roleErr != nil {
									slog.Error("Failed to check for level role", "error", roleErr, "guild_id", m.GuildID, "level", newLevel)
								} else {
									if roleID != nil && *roleID != "" {
										grantErr := s.GuildMemberRoleAdd(m.GuildID, m.Author.ID, *roleID)
										if grantErr != nil {
											slog.Error("Failed to grant level role", "error", grantErr, "guild_id", m.GuildID, "user_id", m.Author.ID, "role_id", *roleID)
										} else {
											slog.Info("Granted level role", "guild_id", m.GuildID, "user_id", m.Author.ID, "role_id", *roleID, "level", newLevel)
										}
									}
									if coinsReward > 0 {
										err := b.DB.AddCoins(context.Background(), m.GuildID, m.Author.ID, coinsReward)
										if err != nil {
											slog.Error("Failed to grant level role coins reward", "error", err, "guild_id", m.GuildID, "user_id", m.Author.ID, "coins", coinsReward)
										} else {
											slog.Info("Granted level role coins reward", "guild_id", m.GuildID, "user_id", m.Author.ID, "coins", coinsReward, "level", newLevel)
										}
									}
								}

								// Announce level up
								msg := fmt.Sprintf("🎉 Congratulations <@%s>, you just advanced to **Level %d**!", m.Author.ID, newLevel)
								if roleID != nil && *roleID != "" {
									msg += fmt.Sprintf(" You've been awarded the <@&%s> role!", *roleID)
								}
								if coinsReward > 0 {
									msg += fmt.Sprintf(" You've also been awarded **%d coins**!", coinsReward)
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
}

// checkExpiredMutes checks for expired mutes and removes them from the database.
func (b *Bot) checkExpiredMutes() {
	if b.DB == nil {
		return
	}

	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		mutes, err := b.DB.GetExpiredMutes(context.Background())
		if err != nil {
			slog.Error("Failed to fetch expired mutes", "error", err)
			time.Sleep(1 * time.Minute)
			continue
		}

		for _, m := range mutes {
			err = b.DB.RemoveMute(context.Background(), m.GuildID, m.UserID)
			if err != nil {
				slog.Error("Failed to remove expired mute from database", "error", err, "guild_id", m.GuildID, "user_id", m.UserID)
			}
		}

		<-ticker.C
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
	emojiID := r.Emoji.ID

	// Check for reaction menu roles
	items, err := b.DB.GetReactionMenuItems(context.Background(), r.MessageID)
	if err == nil && len(items) > 0 {
		for _, item := range items {
			if item.Emoji == emojiName {
				err = s.GuildMemberRoleAdd(r.GuildID, r.UserID, item.RoleID)
				if err != nil {
					slog.Error("Failed to add reaction menu role", "error", err, "user", r.UserID, "role", item.RoleID)
				}
				break
			}
		}
	}

	// Check for giveaway entry
	if emojiName == "🎉" && emojiID == "" {
		g, err := b.DB.GetGiveawayByMessage(context.Background(), r.MessageID)
		if err == nil && g != nil && !g.Ended {
			err = b.DB.AddGiveawayEntrant(context.Background(), g.ID, r.UserID)
			if err != nil {
				slog.Error("Failed to add giveaway entrant", "error", err)
			}
		}
	}

	rr, err := b.DB.GetReactionRole(context.Background(), r.MessageID, emojiName, emojiID)
	if err != nil {
		slog.Error("Failed to get reaction role config", "error", err)
		return
	}

	if rr != nil {

		err = s.GuildMemberRoleAdd(r.GuildID, r.UserID, rr.RoleID)
		if err != nil {
			slog.Error("Failed to add role %s to user %s via reaction", "arg1", rr.RoleID, "arg2", r.UserID, "error", err)
		} else {
			embed := &discordgo.MessageEmbed{
				Title:       "Role Assigned via Reaction",
				Description: fmt.Sprintf("User <@%s> was assigned the role <@&%s> via reaction.", r.UserID, rr.RoleID),
				Color:       0x2ECC71, // Green
			}
			b.handleAdvancedLog(s, r.GuildID, "role_update", embed)

			if rr.GroupID != nil {
				group, err := b.DB.GetReactionRoleGroup(context.Background(), *rr.GroupID)
				if err == nil && group != nil && group.IsExclusive {
					groupRoles, err := b.DB.GetGroupRoles(context.Background(), *rr.GroupID)
					if err == nil {
						member, err := s.GuildMember(r.GuildID, r.UserID)
						if err == nil {
							// Remove other roles in the group
							for _, roleID := range groupRoles {
								if roleID != rr.RoleID {
									for _, memberRole := range member.Roles {
										if roleID == memberRole {
											err = s.GuildMemberRoleRemove(r.GuildID, r.UserID, roleID)
											if err != nil {
												slog.Error("Failed to remove exclusive reaction role", "error", err)
											}
											break
										}
									}
								}
							}
						}
					}
				}
			}
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
	emojiID := r.Emoji.ID

	rr, err := b.DB.GetReactionRole(context.Background(), r.MessageID, emojiName, emojiID)
	if err != nil {
		slog.Error("Failed to get reaction role config", "error", err)
		return
	}

	if rr != nil {
		err = s.GuildMemberRoleRemove(r.GuildID, r.UserID, rr.RoleID)
		if err != nil {
			slog.Error("Failed to remove role %s from user %s via reaction", "arg1", rr.RoleID, "arg2", r.UserID, "error", err)
		} else {
			embed := &discordgo.MessageEmbed{
				Title:       "Role Removed via Reaction",
				Description: fmt.Sprintf("User <@%s> was removed from the role <@&%s> via reaction.", r.UserID, rr.RoleID),
				Color:       0xE74C3C, // Red
			}
			b.handleAdvancedLog(s, r.GuildID, "role_update", embed)
		}
	}

	// Check for reaction menu roles
	items, err := b.DB.GetReactionMenuItems(context.Background(), r.MessageID)
	if err == nil && len(items) > 0 {
		for _, item := range items {
			if item.Emoji == emojiName {
				err = s.GuildMemberRoleRemove(r.GuildID, r.UserID, item.RoleID)
				if err != nil {
					slog.Error("Failed to remove reaction menu role", "error", err, "user", r.UserID, "role", item.RoleID)
				}
				break
			}
		}
	}
}

// guildMemberRemoveHandler is called every time a member leaves a guild
func (b *Bot) memberCountLoop() {
	ticker := time.NewTicker(15 * time.Minute)
	defer ticker.Stop()

	for {
		<-ticker.C
		if b.DB == nil {
			continue
		}

		configs, err := b.DB.GetAllMemberCountChannels(context.Background())
		if err != nil {
			slog.Error("Failed to fetch member count configs", "error", err)
			continue
		}

		for _, cfg := range configs {
			guild, err := b.Session.State.Guild(cfg.GuildID)
			if err != nil || guild == nil {
				guild, err = b.Session.Guild(cfg.GuildID)
				if err != nil {
					slog.Error("Failed to fetch guild for member count loop", "guild_id", cfg.GuildID, "error", err)
					continue
				}
			}

			if guild != nil {
				memberCount := guild.MemberCount
				if memberCount == 0 {
					memberCount = len(guild.Members)
				}
				if memberCount > 0 {
					newName := fmt.Sprintf("Members: %d", memberCount)
					_, err = b.Session.ChannelEdit(cfg.ChannelID, &discordgo.ChannelEdit{Name: newName})
					if err != nil {
						slog.Error("Failed to update channel name in member count loop", "channel_id", cfg.ChannelID, "error", err)
					}
				}
			}
		}
	}
}

func (b *Bot) guildMemberRemoveHandler(s *discordgo.Session, m *discordgo.GuildMemberRemove) {
	if b.DB == nil || m.User == nil {
		return
	}

	// Send goodbye message if configured
	channelID, msg, err := b.DB.GetGoodbyeMessage(context.Background(), m.GuildID)
	if err != nil {
		slog.Error("Failed to get goodbye message", "error", err)
	} else if channelID != "" && msg != "" {
		msg = strings.ReplaceAll(msg, "{user}", m.User.Username)
		_, err = s.ChannelMessageSend(channelID, msg)
		if err != nil {
			slog.Error("Failed to send goodbye message", "channel_id", channelID, "error", err)
		}
	}

	// Attempt to get the member from the state cache before they are completely removed
	member, err := s.State.Member(m.GuildID, m.User.ID)
	if err != nil || member == nil {
		slog.Debug("Could not find member in state cache to save roles", "user_id", m.User.ID, "guild_id", m.GuildID)
		return
	}

	// Persist the user's roles when they leave so they can be restored if they rejoin
	if len(member.Roles) > 0 {
		err := b.DB.SaveUserRoles(context.Background(), m.User.ID, m.GuildID, member.Roles)
		if err != nil {
			slog.Error("Failed to save user roles on leave", "user_id", m.User.ID, "guild_id", m.GuildID, "error", err)
		} else {
			slog.Info("Saved user roles", "user_id", m.User.ID, "guild_id", m.GuildID, "role_count", len(member.Roles))
		}
	}
}

// guildMemberAddHandler is called every time a new member joins a guild
func (b *Bot) guildMemberAddHandler(s *discordgo.Session, m *discordgo.GuildMemberAdd) {
	if b.DB == nil {
		return
	}

	channelID, message, err := b.DB.GetWelcomeMessage(context.Background(), m.GuildID)
	if err != nil {
		slog.Error("Failed to get welcome message", "error", err)
	} else if channelID != "" && message != "" {
		_, err = s.ChannelMessageSend(channelID, message)
		if err != nil {
			slog.Error("Failed to send welcome message", "channel_id", channelID, "error", err)
		}
	}

	dmMessage, isEnabled, err := b.DB.GetWelcomeDM(context.Background(), m.GuildID)
	if err != nil {
		slog.Error("Failed to get welcome DM config", "error", err, "guild_id", m.GuildID)
	} else if isEnabled && dmMessage != "" {
		dmMessage = strings.ReplaceAll(dmMessage, "{user}", m.User.Mention())
		channel, err := s.UserChannelCreate(m.User.ID)
		if err == nil {
			_, err = s.ChannelMessageSend(channel.ID, dmMessage)
			if err != nil {
				slog.Error("Failed to send welcome DM", "user_id", m.User.ID, "error", err)
			}
		} else {
			slog.Error("Failed to create DM channel for welcome DM", "user_id", m.User.ID, "error", err)
		}
	}

	config, err := b.DB.GetGuildConfig(context.Background(), m.GuildID)
	if err != nil {
		slog.Error("Failed to get guild config for welcome message/auto-role", "error", err)
		return
	}

	stickyRoles, err := b.DB.GetStickyRoles(context.Background(), m.GuildID, m.User.ID)
	if err != nil {
		slog.Error("Failed to get sticky roles", "error", err)
	} else {
		for _, roleID := range stickyRoles {
			err = s.GuildMemberRoleAdd(m.GuildID, m.User.ID, roleID)
			if err != nil {
				slog.Error("Failed to restore sticky role", "role_id", roleID, "error", err)
			}
		}
	}

	if config.WelcomeChannelID != nil && *config.WelcomeChannelID != "" {
		welcomeMsg := fmt.Sprintf("Welcome to the server, <@%s>! We are glad to have you here.", m.User.ID)

		imageURL, imgErr := b.DB.GetWelcomeImage(context.Background(), m.GuildID)
		if imgErr != nil {
			slog.Error("Failed to get welcome image", "error", imgErr)
		}

		embed := &discordgo.MessageEmbed{
			Description: welcomeMsg,
			Color:       0x00FF00,
		}

		if imageURL != "" {
			embed.Image = &discordgo.MessageEmbedImage{
				URL: imageURL,
			}
		}

		_, err := s.ChannelMessageSendEmbed(*config.WelcomeChannelID, embed)
		if err != nil {
			slog.Error("Failed to send welcome message", "error", err)
		}
	}

	roleID, err := b.DB.GetAutoRole(context.Background(), m.GuildID)
	if err != nil {
		slog.Error("Failed to fetch auto-role for guild", "guild_id", m.GuildID, "error", err)
	} else if roleID != "" {
		err := s.GuildMemberRoleAdd(m.GuildID, m.User.ID, roleID)
		if err != nil {
			slog.Error("Failed to assign auto-role to user", "user_id", m.User.ID, "guild_id", m.GuildID, "error", err)
		}
	}

	// Restore previously saved user roles (if any)
	savedRoles, err := b.DB.GetUserRoles(context.Background(), m.User.ID, m.GuildID)
	if err != nil {
		slog.Error("Failed to fetch user roles", "user_id", m.User.ID, "guild_id", m.GuildID, "error", err)
	} else if len(savedRoles) > 0 {
		for _, savedRoleID := range savedRoles {
			// Skip adding if it's already the auto-role since it was just added
			if savedRoleID == roleID {
				continue
			}
			err := s.GuildMemberRoleAdd(m.GuildID, m.User.ID, savedRoleID)
			if err != nil {
				slog.Error("Failed to restore role to user", "role_id", savedRoleID, "user_id", m.User.ID, "guild_id", m.GuildID, "error", err)
			}
		}
		slog.Info("Restored user roles", "user_id", m.User.ID, "guild_id", m.GuildID, "role_count", len(savedRoles))
	}

	// Apply nickname template if configured
	nickTemplate, err := b.DB.GetNicknameTemplate(context.Background(), m.GuildID)
	if err != nil {
		slog.Error("Failed to fetch nickname template", "guild_id", m.GuildID, "error", err)
	} else if nickTemplate != nil && *nickTemplate != "" {
		newNickname := strings.ReplaceAll(*nickTemplate, "{user}", m.User.Username)
		if len(newNickname) > 32 {
			newNickname = newNickname[:32]
		}
		err := s.GuildMemberNickname(m.GuildID, m.User.ID, newNickname)
		if err != nil {
			slog.Error("Failed to apply nickname template", "guild_id", m.GuildID, "user_id", m.User.ID, "nickname", newNickname, "error", err)
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

				// Add a Snooze button
				components := []discordgo.MessageComponent{
					discordgo.ActionsRow{
						Components: []discordgo.MessageComponent{
							discordgo.Button{
								Label:    "Snooze 10m",
								Style:    discordgo.SecondaryButton,
								CustomID: fmt.Sprintf("snooze_%s_%d", r.UserID, r.ID),
								Emoji: &discordgo.ComponentEmoji{
									Name: "💤",
								},
							},
						},
					},
				}

				_, err := b.Session.ChannelMessageSendComplex(r.ChannelID, &discordgo.MessageSend{
					Content:    msg,
					Components: components,
				})

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

	// Voice Generator Logic
	if newChannelID != "" {
		generatorConfig, err := b.DB.GetVoiceGeneratorConfig(context.Background(), v.GuildID)
		if err == nil && generatorConfig != nil && generatorConfig.BaseChannelID == newChannelID {
			_, err := s.State.Guild(v.GuildID)
			baseChannel, err2 := s.State.Channel(generatorConfig.BaseChannelID)

			if err == nil && err2 == nil {
				currentGenerated := 0
				b.generatedVoiceChannels.Range(func(key, value interface{}) bool {
					channelID := key.(string)
					// Verify channel still exists and is in the correct category
					if channel, err := s.State.Channel(channelID); err == nil && channel.GuildID == v.GuildID && channel.ParentID == baseChannel.ParentID {
						currentGenerated++
					} else {
						b.generatedVoiceChannels.Delete(key)
					}
					return true
				})

				if currentGenerated < generatorConfig.MaxChannels {
					var name string
					if user != nil {
						name = user.Username + "'s Channel"
					} else {
						name = "Generated Channel"
					}

					channelData := discordgo.GuildChannelCreateData{
						Name:     name,
						Type:     discordgo.ChannelTypeGuildVoice,
						ParentID: baseChannel.ParentID,
					}

					createdChannel, err := s.GuildChannelCreateComplex(v.GuildID, channelData)
					if err != nil {
						slog.Error("Failed to create generated voice channel", "error", err)
					} else {
						// Track the generated channel
						b.generatedVoiceChannels.Store(createdChannel.ID, true)
						err = s.GuildMemberMove(v.GuildID, v.UserID, &createdChannel.ID)
						if err != nil {
							slog.Error("Failed to move user to generated voice channel", "error", err)
							_, _ = s.ChannelDelete(createdChannel.ID)
							b.generatedVoiceChannels.Delete(createdChannel.ID)
						}
					}
				}
			}
		}
	}

	// Dynamic Voice Channels Logic
	if newChannelID != "" {
		dynamicConfig, err := b.DB.GetDynamicVoiceConfig(context.Background(), v.GuildID)
		if err == nil && dynamicConfig != nil && dynamicConfig.TriggerChannelID == newChannelID {
			// User joined the trigger channel, create a new dynamic channel
			var name string
			if user != nil {
				name = user.Username + "'s Channel"
			} else {
				name = "Dynamic Channel"
			}

			channelData := discordgo.GuildChannelCreateData{
				Name:     name,
				Type:     discordgo.ChannelTypeGuildVoice,
				ParentID: dynamicConfig.CategoryID,
			}

			createdChannel, err := s.GuildChannelCreateComplex(v.GuildID, channelData)
			if err != nil {
				slog.Error("Failed to create dynamic voice channel", "error", err)
			} else {
				// Move the user to the new channel
				err = s.GuildMemberMove(v.GuildID, v.UserID, &createdChannel.ID)
				if err != nil {
					slog.Error("Failed to move user to dynamic voice channel", "error", err)
					// Clean up the channel if we can't move the user to it
					_, _ = s.ChannelDelete(createdChannel.ID)
				}
				// We don't save to DB for dynamic voice channels since we rely on ParentID
			}
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
		// Voice Generator Cleanup Logic
		if _, ok := b.generatedVoiceChannels.Load(oldChannelID); ok {
			generatorConfig, err := b.DB.GetVoiceGeneratorConfig(context.Background(), v.GuildID)
			if err == nil && generatorConfig != nil {
				channel, err := s.State.Channel(oldChannelID)
				baseChannel, err2 := s.State.Channel(generatorConfig.BaseChannelID)

				if err == nil && err2 == nil && channel.ParentID == baseChannel.ParentID && oldChannelID != generatorConfig.BaseChannelID {
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
							_, err := s.ChannelDelete(oldChannelID)
							if err != nil {
								slog.Error("Failed to delete empty generated voice channel", "error", err)
							}
							b.generatedVoiceChannels.Delete(oldChannelID)
						}
					}
				}
			}
		}

		// Dynamic Voice Cleanup Logic
		dynamicConfig, err := b.DB.GetDynamicVoiceConfig(context.Background(), v.GuildID)
		if err == nil && dynamicConfig != nil {
			// Check if the old channel was in the dynamic category and is not the trigger channel itself
			channel, err := s.State.Channel(oldChannelID)
			if err == nil && channel.ParentID == dynamicConfig.CategoryID && oldChannelID != dynamicConfig.TriggerChannelID {
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
						_, err := s.ChannelDelete(oldChannelID)
						if err != nil {
							slog.Error("Failed to delete empty dynamic voice channel", "error", err)
						}
					}
				}
			}
		}

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

		// Set voice join time for XP
		_ = b.DB.SetVoiceJoinTime(context.Background(), v.GuildID, v.UserID)
	} else if oldChannelID != "" && newChannelID == "" {
		// Left
		title = "🎙️ Voice Leave"
		description = fmt.Sprintf("<@%s> left voice channel <#%s>", v.UserID, oldChannelID)
		color = 0xFF0000 // Red

		// Calculate and award Voice XP
		joinTime, err := b.DB.GetVoiceJoinTime(context.Background(), v.GuildID, v.UserID)
		if err == nil && joinTime != nil {
			duration := time.Since(*joinTime)
			minutes := int(duration.Minutes())
			if minutes > 0 {
				// Award 1 XP and 1 Coin per minute
				_, _ = b.DB.AddXP(context.Background(), v.GuildID, v.UserID, minutes)
				_ = b.DB.AddCoins(context.Background(), v.GuildID, v.UserID, minutes)
			}
			_ = b.DB.RemoveVoiceJoinTime(context.Background(), v.GuildID, v.UserID)
		}
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

// applyInterestLoop runs periodically to apply daily interest to bank balances.
func (b *Bot) applyInterestLoop() {
	if b.DB == nil {
		return
	}
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for {
		ctx := context.Background()
		appliedCount, err := b.DB.ApplyInterest(ctx)
		if err != nil {
			slog.Error("Failed to apply bank interest", "error", err)
		} else if appliedCount > 0 {
			slog.Info("Applied bank interest", "accounts", appliedCount)
		}
		<-ticker.C
	}
}

// petStatsLoop periodically updates pet stats across all servers.
func (b *Bot) petStatsLoop() {
	if b.DB == nil {
		return
	}

	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		err := b.DB.UpdateAllPetStats(ctx)
		if err != nil {
			slog.Error("Failed to update pet stats in background loop", "error", err)
		}
		cancel()
	}
}

func (b *Bot) checkTempBans() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if b.DB == nil {
				continue
			}

			bans, err := b.DB.GetActiveTempBans()
			if err != nil {
				slog.Error("Error checking active temp bans", "error", err)
				continue
			}

			for _, ban := range bans {
				// Remove ban in discord
				err = b.Session.GuildBanDelete(ban.GuildID, ban.UserID)
				if err != nil {
					slog.Error("Failed to unban user after temp ban expired", "user", ban.UserID, "guild", ban.GuildID, "error", err)
					// If error is unknown ban, it might have been manually removed, so we still clear it from DB
					if !strings.Contains(err.Error(), "Unknown Ban") {
						continue
					}
				}

				// Remove from active bans
				err = b.DB.RemoveTempBan(ban.UserID, ban.GuildID)
				if err != nil {
					slog.Error("Failed to remove temp ban from db", "user", ban.UserID, "guild", ban.GuildID, "error", err)
				}

				// Mark Mod Action resolved
				// We don't have the exact action ID here easily, but we can resolve any active ban for this user
				// We can't mark by action ID easily without fetching it, so we added MarkAllUserModActionsResolved
				err = b.DB.MarkAllUserModActionsResolved(context.Background(), ban.GuildID, ban.UserID, "ban")
				if err != nil {
					slog.Error("Failed to mark temp ban mod action resolved", "user", ban.UserID, "guild", ban.GuildID, "error", err)
				}
			}
		}
	}
}

// handleAdvancedLog checks if the event should be logged and sends it to the configured channel
func (b *Bot) handleAdvancedLog(s *discordgo.Session, guildID string, eventType string, embed *discordgo.MessageEmbed) {
	if b.DB == nil {
		return
	}

	config, err := b.DB.GetAdvancedLogConfig(context.Background(), guildID)
	if err != nil || config == nil {
		return
	}

	events := strings.Split(config.Events, ",")
	shouldLog := false
	for _, e := range events {
		e = strings.TrimSpace(e)
		if e == "all" || e == eventType {
			shouldLog = true
			break
		}
	}

	if shouldLog {
		_, err := s.ChannelMessageSendEmbed(config.ChannelID, embed)
		if err != nil {
			slog.Error("Failed to send advanced log", "guild_id", guildID, "channel_id", config.ChannelID, "error", err)
		}
	}
}

// channelCreateHandler logs when a new channel is created
func (b *Bot) channelCreateHandler(s *discordgo.Session, c *discordgo.ChannelCreate) {
	embed := &discordgo.MessageEmbed{
		Title:       "Channel Created",
		Description: fmt.Sprintf("Channel **%s** (<#%s>) was created.", c.Name, c.ID),
		Color:       0x2ECC71, // Green
	}
	b.handleAdvancedLog(s, c.GuildID, "channel_create", embed)
}

// channelDeleteHandler logs when a channel is deleted
func (b *Bot) channelDeleteHandler(s *discordgo.Session, c *discordgo.ChannelDelete) {
	embed := &discordgo.MessageEmbed{
		Title:       "Channel Deleted",
		Description: fmt.Sprintf("Channel **%s** (%s) was deleted.", c.Name, c.ID),
		Color:       0xE74C3C, // Red
	}
	b.handleAdvancedLog(s, c.GuildID, "channel_delete", embed)
}

// guildRoleCreateHandler logs when a new role is created
func (b *Bot) guildRoleCreateHandler(s *discordgo.Session, r *discordgo.GuildRoleCreate) {
	embed := &discordgo.MessageEmbed{
		Title:       "Role Created",
		Description: fmt.Sprintf("Role **%s** (<@&%s>) was created.", r.Role.Name, r.Role.ID),
		Color:       0x2ECC71, // Green
	}
	b.handleAdvancedLog(s, r.GuildID, "role_create", embed)
}

// guildRoleDeleteHandler logs when a role is deleted
func (b *Bot) guildRoleDeleteHandler(s *discordgo.Session, r *discordgo.GuildRoleDelete) {
	embed := &discordgo.MessageEmbed{
		Title:       "Role Deleted",
		Description: fmt.Sprintf("Role with ID **%s** was deleted.", r.RoleID),
		Color:       0xE74C3C, // Red
	}
	b.handleAdvancedLog(s, r.GuildID, "role_delete", embed)
}

func (b *Bot) checkTempRoles() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-context.Background().Done():
			return
		case <-ticker.C:
			if b.DB == nil {
				continue
			}

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			roles, err := b.DB.GetExpiredTempRoles(ctx)
			if err != nil {
				slog.Error("Failed to fetch expired temp roles", "error", err)
				cancel()
				continue
			}

			for _, role := range roles {
				// Remove role via Discord API
				err := b.Session.GuildMemberRoleRemove(role.GuildID, role.UserID, role.RoleID)
				if err != nil {
					// Check if error is due to missing permissions (50013) or unknown role (10011) or member missing (10007).
					// Usually we should just proceed to clean up the DB regardless of discord error unless it's a temp API issue,
					// but removing from DB is safest to avoid endless looping on error.
					slog.Warn("Failed to remove expired temp role in Discord", "guild_id", role.GuildID, "user_id", role.UserID, "role_id", role.RoleID, "error", err)
				}

				// Clean up DB row
				err = b.DB.RemoveTempRole(ctx, role.ID)
				if err != nil {
					slog.Error("Failed to remove expired temp role from DB", "id", role.ID, "error", err)
				} else {
					slog.Info("Removed expired temp role", "guild_id", role.GuildID, "user_id", role.UserID, "role_id", role.RoleID)
				}
			}
			cancel()
		}
	}
}

func (b *Bot) checkTempNicknames() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-context.Background().Done():
			return
		case <-ticker.C:
			if b.DB == nil {
				continue
			}

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			nicks, err := b.DB.GetExpiredTempNicknames(ctx)
			if err != nil {
				slog.Error("Failed to fetch expired temp nicknames", "error", err)
				cancel()
				continue
			}

			for _, nick := range nicks {
				err := b.Session.GuildMemberNickname(nick.GuildID, nick.UserID, nick.OriginalNickname)
				if err != nil {
					slog.Warn("Failed to revert expired temp nickname in Discord", "guild_id", nick.GuildID, "user_id", nick.UserID, "error", err)
				}

				// Clean up DB row
				err = b.DB.RemoveTempNickname(ctx, nick.ID)
				if err != nil {
					slog.Error("Failed to remove expired temp nickname from DB", "id", nick.ID, "error", err)
				}
			}
			cancel()
		}
	}
}
