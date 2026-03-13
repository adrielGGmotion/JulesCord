package bot

import (
	"context"
	"fmt"
	"log"
	"math"
	"math/rand"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
	"julescord/internal/bot/commands"
	"julescord/internal/config"
	"julescord/internal/db"
)

// Bot manages the Discord connection.
type Bot struct {
	Session    *discordgo.Session
	Config     *config.Config
	Registry   *commands.Registry
	DB         *db.DB
	xpCooldown sync.Map // map[string]time.Time (key: guildID_channelID_userID)
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

	bot := &Bot{
		Session:  session,
		Config:   cfg,
		Registry: registry,
		DB:       database,
	}

	// Register ready handler
	bot.Session.AddHandler(bot.readyHandler)

	// Register guild create handler
	bot.Session.AddHandler(bot.guildCreateHandler)

	// Register interaction handler
	bot.Session.AddHandler(bot.interactionCreateHandler)

	// Register message create handler
	bot.Session.AddHandler(bot.messageCreateHandler)

	// Set intentions
	bot.Session.Identify.Intents = discordgo.IntentsGuilds | discordgo.IntentsGuildMessages

	return bot, nil
}

// Start opens the connection to Discord.
func (b *Bot) Start() error {
	log.Println("Starting Discord bot...")
	err := b.Session.Open()
	if err != nil {
		return fmt.Errorf("error opening Discord connection: %w", err)
	}

	log.Println("Discord bot started successfully.")
	return nil
}

// Stop closes the connection to Discord gracefully.
func (b *Bot) Stop() error {
	log.Println("Stopping Discord bot...")
	err := b.Session.Close()
	if err != nil {
		return fmt.Errorf("error closing Discord connection: %w", err)
	}

	log.Println("Discord bot stopped gracefully.")
	return nil
}

// readyHandler triggers when the bot connects to Discord.
func (b *Bot) readyHandler(s *discordgo.Session, event *discordgo.Ready) {
	log.Printf("Bot is ready! Logged in as %s#%s", event.User.Username, event.User.Discriminator)

	// Register commands with Discord when ready
	err := b.Registry.RegisterWithDiscord(s, b.Config.DiscordClientID, "")
	if err != nil {
		log.Printf("Error registering commands: %v", err)
	}
}

// guildCreateHandler is called when the bot joins a new guild or a guild becomes available.
func (b *Bot) guildCreateHandler(s *discordgo.Session, event *discordgo.GuildCreate) {
	if b.DB == nil {
		return
	}

	err := b.DB.UpsertGuild(context.Background(), event.Guild.ID)
	if err != nil {
		log.Printf("Failed to upsert guild %s: %v", event.Guild.ID, err)
	} else {
		log.Printf("Guild registered/upserted: %s", event.Guild.ID)
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
				log.Printf("Failed to upsert user %s: %v", user.ID, err)
			}

			// Log command execution
			commandName := i.ApplicationCommandData().Name
			err = b.DB.LogCommand(context.Background(), commandName, user.ID, i.GuildID)
			if err != nil {
				log.Printf("Failed to log command execution for %s: %v", commandName, err)
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

		if !onCooldown {
			// Award XP (e.g., random 15-25 XP)
			amount := rand.Intn(11) + 15

			// Ensure user exists first
			err := b.DB.UpsertUser(context.Background(), m.Author.ID, m.Author.Username, m.Author.GlobalName, m.Author.AvatarURL(""))
			if err != nil {
				log.Printf("Failed to upsert user %s for XP: %v", m.Author.ID, err)
			} else {
				// Add XP
				newXP, err := b.DB.AddXP(context.Background(), m.GuildID, m.Author.ID, amount)
				if err != nil {
					log.Printf("Failed to add XP to user %s: %v", m.Author.ID, err)
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
							log.Printf("Failed to update level for user %s: %v", m.Author.ID, err)
						} else {
							// Announce level up
							msg := fmt.Sprintf("🎉 Congratulations <@%s>, you just advanced to **Level %d**!", m.Author.ID, newLevel)
							_, err = s.ChannelMessageSend(m.ChannelID, msg)
							if err != nil {
								log.Printf("Failed to send level up message: %v", err)
							}
						}
					}
				}
			}
		}
	}
}
