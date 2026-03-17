package commands

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
)

// Lock creates the /lock command.
func Lock() *Command {
	return &Command{
		Definition: &discordgo.ApplicationCommand{
			Name:                     "lock",
			Description:              "Lock the current channel, preventing @everyone from sending messages",
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

			// Clear SendMessages from allow, and add it to deny
			allow &= ^discordgo.PermissionSendMessages
			deny |= discordgo.PermissionSendMessages

			err = s.ChannelPermissionSet(
				i.ChannelID,
				everyoneRoleID,
				discordgo.PermissionOverwriteTypeRole,
				allow,
				deny,
			)
			if err != nil {
				_, _ = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
					Content: func() *string { m := fmt.Sprintf("❌ Failed to lock the channel: %v", err); return &m }(),
				})
				return
			}

			msg := "🔒 This channel has been **locked**."
			_, _ = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
				Content: &msg,
			})
		},
	}
}
