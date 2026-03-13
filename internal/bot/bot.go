package bot

import (
	"fmt"
	"log"

	"github.com/bwmarrin/discordgo"
	"julescord/internal/config"
)

// Bot manages the Discord connection.
type Bot struct {
	Session *discordgo.Session
	Config  *config.Config
}

// New initializes a new bot instance.
func New(cfg *config.Config) (*Bot, error) {
	if cfg.DiscordToken == "" {
		return nil, fmt.Errorf("discord token is required")
	}

	session, err := discordgo.New("Bot " + cfg.DiscordToken)
	if err != nil {
		return nil, fmt.Errorf("error creating Discord session: %w", err)
	}

	bot := &Bot{
		Session: session,
		Config:  cfg,
	}

	// Register ready handler
	bot.Session.AddHandler(bot.readyHandler)

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
}
