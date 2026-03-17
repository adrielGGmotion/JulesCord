package commands

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"julescord/internal/db"

	"github.com/bwmarrin/discordgo"
)

// Music returns the music slash command.
func Music(database *db.DB) *Command {
	return &Command{
		Definition: &discordgo.ApplicationCommand{
			Name:                     "music",
			Description:              "Configure the music system",
			DMPermission:             new(bool),
			DefaultMemberPermissions: func(p int64) *int64 { return &p }(discordgo.PermissionAdministrator),
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        "setup",
					Description: "Set the music channel for the server",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Name:        "channel",
							Description: "The text channel for music commands and player",
							Type:        discordgo.ApplicationCommandOptionChannel,
							Required:    true,
							ChannelTypes: []discordgo.ChannelType{
								discordgo.ChannelTypeGuildText,
							},
						},
					},
				},
			},
		},
		Handler: func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			options := i.ApplicationCommandData().Options
			if len(options) == 0 {
				return
			}
			subcommand := options[0].Name

			switch subcommand {
			case "setup":
				handleMusicSetup(s, i, database)
			}
		},
	}
}

func handleMusicSetup(s *discordgo.Session, i *discordgo.InteractionCreate, database *db.DB) {
	options := i.ApplicationCommandData().Options[0].Options
	var channelID string

	for _, opt := range options {
		if opt.Name == "channel" {
			channelID = opt.ChannelValue(s).ID
		}
	}

	guildID := i.GuildID

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := database.SetMusicChannel(ctx, guildID, channelID)
	if err != nil {
		slog.Error("Failed to set music channel", "error", err)
		SendError(s, i, "Failed to configure the music channel.")
		return
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("✅ Music channel successfully set to <#%s>.", channelID),
		},
	})
}
