package commands

import (
	"log/slog"

	"julescord/internal/db"

	"github.com/bwmarrin/discordgo"
)

// Lock returns the /lock command definition and handler.
func Lock(database *db.DB) *Command {
	defaultMemberPermissions := int64(discordgo.PermissionManageChannels)
	return &Command{
		Definition: &discordgo.ApplicationCommand{
			Name:                     "lock",
			Description:              "Deny SendMessages permission for @everyone in this channel.",
			DefaultMemberPermissions: &defaultMemberPermissions,
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

			channel, err := s.Channel(i.ChannelID)
			if err != nil {
				slog.Error("Error fetching channel", "channel_id", i.ChannelID, "error", err)
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: "An error occurred while fetching the channel.",
					},
				})
				return
			}

			var allow, deny int64
			// Find existing overwrite for @everyone
			for _, overwrite := range channel.PermissionOverwrites {
				if overwrite.ID == i.GuildID { // @everyone role ID is the guild ID
					allow = overwrite.Allow
					deny = overwrite.Deny
					break
				}
			}

			// Apply bitwise operations to deny SendMessages
			allow &= ^int64(discordgo.PermissionSendMessages)
			deny |= int64(discordgo.PermissionSendMessages)

			err = s.ChannelPermissionSet(i.ChannelID, i.GuildID, discordgo.PermissionOverwriteTypeRole, allow, deny)
			if err != nil {
				slog.Error("Error updating channel permissions", "channel_id", i.ChannelID, "error", err)
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: "An error occurred while locking the channel.",
					},
				})
				return
			}

			embed := &discordgo.MessageEmbed{
				Title:       "Channel Locked 🔒",
				Description: "The `@everyone` role can no longer send messages in this channel.",
				Color:       0xFF0000, // Red
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
