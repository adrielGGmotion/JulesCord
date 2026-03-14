package commands

import (
	"context"
	"fmt"
	"julescord/internal/db"

	"github.com/bwmarrin/discordgo"
)

// Thread returns the command definition for thread management.
func Thread(database *db.DB) *Command {
	return &Command{
		Definition: &discordgo.ApplicationCommand{
			Name:                     "thread",
			Description:              "Thread management commands",
			DefaultMemberPermissions: new(int64), // Required admin permissions
			DMPermission:             new(bool),  // Disable in DMs
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        "setup",
					Description: "Set the channel for automatic thread creation",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionChannel,
							Name:        "channel",
							Description: "The channel to create threads in",
							Required:    true,
							ChannelTypes: []discordgo.ChannelType{
								discordgo.ChannelTypeGuildText,
							},
						},
					},
				},
				{
					Name:        "lock",
					Description: "Lock the current thread",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
				},
				{
					Name:        "unlock",
					Description: "Unlock the current thread",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
				},
				{
					Name:        "archive",
					Description: "Archive the current thread",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
				},
			},
		},
		Handler: func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			if i.Member == nil {
				SendError(s, i, "This command can only be used in a server.")
				return
			}

			// Subcommand handling
			options := i.ApplicationCommandData().Options
			if len(options) == 0 {
				return
			}
			subcommand := options[0]

			switch subcommand.Name {
			case "setup":
				handleThreadSetup(s, i, database, subcommand.Options)
			case "lock":
				handleThreadLock(s, i)
			case "unlock":
				handleThreadUnlock(s, i)
			case "archive":
				handleThreadArchive(s, i)
			}
		},
	}
}

func handleThreadSetup(s *discordgo.Session, i *discordgo.InteractionCreate, database *db.DB, options []*discordgo.ApplicationCommandInteractionDataOption) {
	if database == nil {
		SendError(s, i, "Database connection not available.")
		return
	}

	var channelID string
	for _, opt := range options {
		if opt.Name == "channel" {
			channelID = opt.Value.(string)
		}
	}

	if channelID == "" {
		SendError(s, i, "Channel ID is required.")
		return
	}

	err := database.SetThreadConfig(context.Background(), i.GuildID, channelID)
	if err != nil {
		SendError(s, i, "Failed to set thread configuration.")
		return
	}

	embed := &discordgo.MessageEmbed{
		Title:       "✅ Thread Configuration Updated",
		Description: fmt.Sprintf("Automatic threads will now be created for messages in <#%s>.", channelID),
		Color:       0x00FF00,
	}
	SendEmbed(s, i, embed)
}

func handleThreadLock(s *discordgo.Session, i *discordgo.InteractionCreate) {
	channel, err := s.Channel(i.ChannelID)
	if err != nil {
		SendError(s, i, "Failed to retrieve channel information.")
		return
	}

	if channel.Type != discordgo.ChannelTypeGuildPublicThread && channel.Type != discordgo.ChannelTypeGuildPrivateThread && channel.Type != discordgo.ChannelTypeGuildNewsThread {
		SendError(s, i, "This command can only be used inside a thread.")
		return
	}

	_, err = s.ChannelEdit(i.ChannelID, &discordgo.ChannelEdit{
		Locked: newBool(true),
	})
	if err != nil {
		SendError(s, i, "Failed to lock the thread.")
		return
	}

	embed := &discordgo.MessageEmbed{
		Title:       "🔒 Thread Locked",
		Description: "This thread has been locked. Only moderators can send messages.",
		Color:       0xFFA500,
	}
	SendEmbed(s, i, embed)
}

func handleThreadUnlock(s *discordgo.Session, i *discordgo.InteractionCreate) {
	channel, err := s.Channel(i.ChannelID)
	if err != nil {
		SendError(s, i, "Failed to retrieve channel information.")
		return
	}

	if channel.Type != discordgo.ChannelTypeGuildPublicThread && channel.Type != discordgo.ChannelTypeGuildPrivateThread && channel.Type != discordgo.ChannelTypeGuildNewsThread {
		SendError(s, i, "This command can only be used inside a thread.")
		return
	}

	_, err = s.ChannelEdit(i.ChannelID, &discordgo.ChannelEdit{
		Locked: newBool(false),
	})
	if err != nil {
		SendError(s, i, "Failed to unlock the thread.")
		return
	}

	embed := &discordgo.MessageEmbed{
		Title:       "🔓 Thread Unlocked",
		Description: "This thread has been unlocked.",
		Color:       0x00FF00,
	}
	SendEmbed(s, i, embed)
}

func handleThreadArchive(s *discordgo.Session, i *discordgo.InteractionCreate) {
	channel, err := s.Channel(i.ChannelID)
	if err != nil {
		SendError(s, i, "Failed to retrieve channel information.")
		return
	}

	if channel.Type != discordgo.ChannelTypeGuildPublicThread && channel.Type != discordgo.ChannelTypeGuildPrivateThread && channel.Type != discordgo.ChannelTypeGuildNewsThread {
		SendError(s, i, "This command can only be used inside a thread.")
		return
	}

	// Send confirmation before archiving
	embed := &discordgo.MessageEmbed{
		Title:       "📦 Thread Archived",
		Description: "This thread has been archived.",
		Color:       0x808080,
	}
	SendEmbed(s, i, embed)

	_, err = s.ChannelEdit(i.ChannelID, &discordgo.ChannelEdit{
		Archived: newBool(true),
	})
	if err != nil {
		// Only log, because we already responded to the interaction
		fmt.Printf("Failed to archive thread %s: %v\n", i.ChannelID, err)
	}
}

func newBool(b bool) *bool {
	return &b
}
