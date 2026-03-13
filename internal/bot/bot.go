package bot

import (
	"context"
	"fmt"
	"log"

	"github.com/bwmarrin/discordgo"
	"julescord/internal/bot/commands"
	"julescord/internal/config"
	"julescord/internal/db"
)

// Bot manages the Discord connection.
type Bot struct {
	Session  *discordgo.Session
	Config   *config.Config
	Registry *commands.Registry
	DB       *db.DB
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
