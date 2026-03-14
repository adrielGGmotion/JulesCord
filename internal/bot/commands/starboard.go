package commands

import (
	"context"
	"fmt"

	"github.com/bwmarrin/discordgo"
	"julescord/internal/db"
)

// Starboard creates the /starboard command.
func Starboard(database *db.DB) *Command {
	if database == nil {
		return nil
	}

	return &Command{
		Definition: &discordgo.ApplicationCommand{
			Name:                     "starboard",
			Description:              "Configure the starboard system",
			DefaultMemberPermissions: func() *int64 { p := int64(discordgo.PermissionAdministrator); return &p }(),
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        "setup",
					Description: "Set the starboard channel and threshold",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionChannel,
							Name:        "channel",
							Description: "The channel to send starboard messages to",
							Required:    true,
							ChannelTypes: []discordgo.ChannelType{
								discordgo.ChannelTypeGuildText,
							},
						},
						{
							Type:        discordgo.ApplicationCommandOptionInteger,
							Name:        "min_stars",
							Description: "The minimum number of stars required to post on the starboard (default 3)",
							Required:    false,
							MinValue:    func() *float64 { v := float64(1); return &v }(),
						},
					},
				},
			},
		},
		Handler: func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			if i.Type != discordgo.InteractionApplicationCommand {
				return
			}

			options := i.ApplicationCommandData().Options
			if len(options) == 0 {
				return
			}

			subcommand := options[0].Name
			if subcommand == "setup" {
				handleStarboardSetup(s, i, database, options[0].Options)
			}
		},
	}
}

func handleStarboardSetup(s *discordgo.Session, i *discordgo.InteractionCreate, database *db.DB, options []*discordgo.ApplicationCommandInteractionDataOption) {
	var channelID string
	minStars := 3 // Default

	for _, opt := range options {
		if opt.Name == "channel" {
			channelID = opt.ChannelValue(nil).ID
		} else if opt.Name == "min_stars" {
			minStars = int(opt.IntValue())
		}
	}

	err := database.SetStarboardConfig(context.Background(), i.GuildID, channelID, minStars)
	if err != nil {
		SendError(s, i, "Failed to save starboard configuration.")
		return
	}

	msg := fmt.Sprintf("✅ Starboard configured! Messages with **%d** or more ⭐ reactions will be posted in <#%s>.", minStars, channelID)
	err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: msg,
		},
	})
	if err != nil {
		// Log but do nothing
	}
}
