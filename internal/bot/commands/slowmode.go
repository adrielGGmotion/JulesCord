package commands

import (
	"fmt"
	"github.com/bwmarrin/discordgo"
	"julescord/internal/db"
)

func Slowmode(database *db.DB) *Command {
	return &Command{
		Definition: &discordgo.ApplicationCommand{
			Name:                     "slowmode",
			Description:              "Sets the slowmode duration for the current channel",
			DefaultMemberPermissions: func() *int64 { p := int64(discordgo.PermissionManageChannels); return &p }(),
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionInteger,
					Name:        "seconds",
					Description: "Slowmode duration in seconds (0 to disable, max 21600)",
					Required:    true,
				},
			},
		},
		Handler: func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
			})
			if err != nil {
				return
			}

			seconds := int(i.ApplicationCommandData().Options[0].IntValue())
			if seconds < 0 || seconds > 21600 {
				SendError(s, i, "Slowmode duration must be between 0 and 21600 seconds (6 hours).")
				return
			}

			secPtr := &seconds
			_, err = s.ChannelEditComplex(i.ChannelID, &discordgo.ChannelEdit{
				RateLimitPerUser: secPtr,
			})
			if err != nil {
				SendError(s, i, fmt.Sprintf("Failed to set slowmode: %v", err))
				return
			}

			msg := fmt.Sprintf("⏱️ Channel slowmode set to %d seconds.", seconds)
			if seconds == 0 {
				msg = "⏱️ Channel slowmode disabled."
			}

			_, _ = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
				Content: stringPtr(msg),
			})
		},
	}
}
