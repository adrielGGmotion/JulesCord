package commands

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
)

// Slowmode creates the /slowmode command.
func Slowmode() *Command {
	return &Command{
		Definition: &discordgo.ApplicationCommand{
			Name:                     "slowmode",
			Description:              "Set the slowmode delay for the current channel",
			DMPermission:             func() *bool { b := false; return &b }(),
			DefaultMemberPermissions: func() *int64 { p := int64(discordgo.PermissionManageChannels); return &p }(),
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionInteger,
					Name:        "seconds",
					Description: "Delay in seconds (0 to disable)",
					Required:    true,
					MinValue:    func() *float64 { v := float64(0); return &v }(),
					MaxValue:    21600, // max is 6 hours
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

			edit := &discordgo.ChannelEdit{
				RateLimitPerUser: &seconds,
			}

			_, err = s.ChannelEdit(i.ChannelID, edit)
			if err != nil {
				_, _ = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
					Content: func() *string { m := fmt.Sprintf("❌ Failed to set slowmode: %v", err); return &m }(),
				})
				return
			}

			var msg string
			if seconds == 0 {
				msg = "✅ Slowmode has been disabled for this channel."
			} else {
				msg = fmt.Sprintf("✅ Slowmode has been set to **%d seconds**.", seconds)
			}

			_, _ = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
				Content: &msg,
			})
		},
	}
}
