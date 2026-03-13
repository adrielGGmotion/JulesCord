package commands

import (
	"context"
	"fmt"
	"log/slog"

	"julescord/internal/db"

	"github.com/bwmarrin/discordgo"
)

// Purge returns the /purge command definition and handler.
func Purge(database *db.DB) *Command {
	defaultMemberPermissions := int64(discordgo.PermissionManageMessages)
	minValue := float64(1)
	return &Command{
		Definition: &discordgo.ApplicationCommand{
			Name:                     "purge",
			Description:              "Bulk deletes up to 100 messages in the current channel.",
			DefaultMemberPermissions: &defaultMemberPermissions,
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionInteger,
					Name:        "count",
					Description: "Number of messages to delete (1-100)",
					Required:    true,
					MinValue:    &minValue,
					MaxValue:    100.0,
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

			var count int
			for _, option := range i.ApplicationCommandData().Options {
				if option.Name == "count" {
					count = int(option.IntValue())
				}
			}

			if count < 1 || count > 100 {
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: "Please provide a count between 1 and 100.",
						Flags:   discordgo.MessageFlagsEphemeral,
					},
				})
				return
			}

			// Acknowledge the interaction first as fetching and deleting messages might take a while
			err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Flags: discordgo.MessageFlagsEphemeral,
				},
			})
			if err != nil {
				slog.Error("Failed to defer interaction for purge", "error", err)
				return
			}

			// Get the messages
			messages, err := s.ChannelMessages(i.ChannelID, count, "", "", "")
			if err != nil {
				slog.Error("Error fetching messages for purge in channel %s", "arg1", i.ChannelID, "error", err)
				s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
					Content: "Failed to fetch messages to delete.",
				})
				return
			}

			if len(messages) == 0 {
				s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
					Content: "No messages found to delete.",
				})
				return
			}

			var messageIDs []string
			for _, m := range messages {
				messageIDs = append(messageIDs, m.ID)
			}

			if len(messageIDs) == 1 {
				// Single delete
				err = s.ChannelMessageDelete(i.ChannelID, messageIDs[0])
				if err != nil {
					slog.Error("Error deleting single message in channel %s", "arg1", i.ChannelID, "error", err)
					s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
						Content: "Failed to delete the message.",
					})
					return
				}
			} else {
				// Bulk delete
				err = s.ChannelMessagesBulkDelete(i.ChannelID, messageIDs)
				if err != nil {
					slog.Error("Error bulk deleting messages in channel %s", "arg1", i.ChannelID, "error", err)
					s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
						Content: "Failed to delete messages. Note: Discord does not allow bulk deleting messages older than 14 days.",
					})
					return
				}
			}

			moderator := i.Member.User

			// Log moderation action if DB is available
			if database != nil {
				err = database.LogModAction(context.Background(), i.GuildID, moderator.ID, moderator.ID, "purge", fmt.Sprintf("Purged %d messages", len(messageIDs)))
				if err != nil {
					slog.Error("Error logging mod action 'purge'", "error", err)
				}
			}

			// Respond with success
			s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
				Content: fmt.Sprintf("Successfully deleted %d messages.", len(messageIDs)),
			})

			embed := &discordgo.MessageEmbed{
				Title:       "Messages Purged",
				Description: fmt.Sprintf("%d messages were deleted in <#%s>.", len(messageIDs), i.ChannelID),
				Color:       0x808080, // Gray
				Fields: []*discordgo.MessageEmbedField{
					{
						Name:   "Moderator",
						Value:  fmt.Sprintf("<@%s>", moderator.ID),
						Inline: true,
					},
				},
			}

			SendModLog(s, database, i.GuildID, embed)
		},
	}
}
