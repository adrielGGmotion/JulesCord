package commands

import (
	"fmt"
	"log/slog"

	"julescord/internal/db"

	"github.com/bwmarrin/discordgo"
)

// Slowmode returns the /slowmode command definition and handler.
func Slowmode(database *db.DB) *Command {
	defaultMemberPermissions := int64(discordgo.PermissionManageChannels)
	return &Command{
		Definition: &discordgo.ApplicationCommand{
			Name:                     "slowmode",
			Description:              "Set the slowmode duration for this channel.",
			DefaultMemberPermissions: &defaultMemberPermissions,
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionInteger,
					Name:        "duration",
					Description: "The slowmode duration in seconds (0 to disable).",
					Required:    true,
				},
			},
		},
		Handler: func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			if i.GuildID == "" {
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: "This command can only be used in a server.",
					},
				})
				return
			}

			var duration int
			for _, option := range i.ApplicationCommandData().Options {
				if option.Name == "duration" {
					duration = int(option.IntValue())
				}
			}

			if duration < 0 || duration > 21600 {
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: "Invalid duration. Slowmode must be between 0 and 21600 seconds (6 hours).",
					},
				})
				return
			}

			_, err := s.ChannelEditComplex(i.ChannelID, &discordgo.ChannelEdit{
				RateLimitPerUser: &duration,
			})
			if err != nil {
				slog.Error("Error setting slowmode", "channel_id", i.ChannelID, "error", err)
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: "An error occurred while setting slowmode.",
					},
				})
				return
			}

			var description string
			if duration == 0 {
				description = "Slowmode has been disabled in this channel."
			} else {
				description = fmt.Sprintf("Slowmode has been set to **%d seconds** in this channel.", duration)
			}

			embed := &discordgo.MessageEmbed{
				Title:       "Slowmode Updated ⏱️",
				Description: description,
				Color:       0x00A0FF, // Blue
			}

			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Embeds: []*discordgo.MessageEmbed{embed},
				},
			})
		},
	}
}
