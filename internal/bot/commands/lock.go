package commands

import (
	"fmt"
	"github.com/bwmarrin/discordgo"
	"julescord/internal/db"
)

func Lock(database *db.DB) *Command {
	return &Command{
		Definition: &discordgo.ApplicationCommand{
			Name:                     "lock",
			Description:              "Locks the current channel (denies @everyone send messages)",
			DefaultMemberPermissions: func() *int64 { p := int64(discordgo.PermissionManageChannels); return &p }(),
		},
		Handler: func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
			})
			if err != nil {
				return
			}

			channel, err := s.State.Channel(i.ChannelID)
			if err != nil {
				channel, err = s.Channel(i.ChannelID)
				if err != nil {
					SendError(s, i, "Failed to fetch channel.")
					return
				}
			}

			guildID := i.GuildID
			var allow int64 = 0
			var deny int64 = 0

			for _, ow := range channel.PermissionOverwrites {
				if ow.ID == guildID {
					allow = ow.Allow
					deny = ow.Deny
					break
				}
			}

			// Add SendMessages to deny
			deny |= discordgo.PermissionSendMessages

			err = s.ChannelPermissionSet(i.ChannelID, guildID, discordgo.PermissionOverwriteTypeRole, allow, deny)
			if err != nil {
				SendError(s, i, fmt.Sprintf("Failed to lock channel: %v", err))
				return
			}

			_, _ = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
				Content: stringPtr("🔒 This channel has been locked."),
			})
		},
	}
}

func Unlock(database *db.DB) *Command {
	return &Command{
		Definition: &discordgo.ApplicationCommand{
			Name:                     "unlock",
			Description:              "Unlocks the current channel (restores @everyone send messages)",
			DefaultMemberPermissions: func() *int64 { p := int64(discordgo.PermissionManageChannels); return &p }(),
		},
		Handler: func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
			})
			if err != nil {
				return
			}

			channel, err := s.State.Channel(i.ChannelID)
			if err != nil {
				channel, err = s.Channel(i.ChannelID)
				if err != nil {
					SendError(s, i, "Failed to fetch channel.")
					return
				}
			}

			guildID := i.GuildID
			var allow int64 = 0
			var deny int64 = 0

			for _, ow := range channel.PermissionOverwrites {
				if ow.ID == guildID {
					allow = ow.Allow
					deny = ow.Deny
					break
				}
			}

			// Clear SendMessages from deny
			deny &= ^discordgo.PermissionSendMessages

			err = s.ChannelPermissionSet(i.ChannelID, guildID, discordgo.PermissionOverwriteTypeRole, allow, deny)
			if err != nil {
				SendError(s, i, fmt.Sprintf("Failed to unlock channel: %v", err))
				return
			}

			_, _ = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
				Content: stringPtr("🔓 This channel has been unlocked."),
			})
		},
	}
}
