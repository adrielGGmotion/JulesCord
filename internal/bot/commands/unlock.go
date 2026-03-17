package commands

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
)

// Unlock creates the /unlock command.
func Unlock() *Command {
	return &Command{
		Definition: &discordgo.ApplicationCommand{
			Name:                     "unlock",
			Description:              "Unlock the current channel, allowing @everyone to send messages",
			DMPermission:             func() *bool { b := false; return &b }(),
			DefaultMemberPermissions: func() *int64 { p := int64(discordgo.PermissionManageChannels); return &p }(),
		},
		Handler: func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
			})
			if err != nil {
				return
			}

			// Fetch the channel to get existing overwrites
			channel, err := s.Channel(i.ChannelID)
			if err != nil {
				_, _ = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
					Content: func() *string { m := fmt.Sprintf("❌ Failed to fetch channel: %v", err); return &m }(),
				})
				return
			}

			// In Discord, the @everyone role ID is the same as the Guild ID
			everyoneRoleID := i.GuildID

			var allow, deny int64
			for _, overwrite := range channel.PermissionOverwrites {
				if overwrite.ID == everyoneRoleID {
					allow = overwrite.Allow
					deny = overwrite.Deny
					break
				}
			}

			// Clear SendMessages from deny, so it resets to the default (or allow)
			deny &= ^discordgo.PermissionSendMessages

			err = s.ChannelPermissionSet(
				i.ChannelID,
				everyoneRoleID,
				discordgo.PermissionOverwriteTypeRole,
				allow,
				deny,
			)
			if err != nil {
				_, _ = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
					Content: func() *string { m := fmt.Sprintf("❌ Failed to unlock the channel: %v", err); return &m }(),
				})
				return
			}

			msg := "🔓 This channel has been **unlocked**."
			_, _ = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
				Content: &msg,
			})
		},
	}
}
