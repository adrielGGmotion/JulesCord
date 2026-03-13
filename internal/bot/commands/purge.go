package commands

import (
	"context"
	"fmt"
	"log"

	"github.com/bwmarrin/discordgo"
	"julescord/internal/db"
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
				log.Printf("Failed to defer interaction for purge: %v", err)
				return
			}

			// Get the messages
			messages, err := s.ChannelMessages(i.ChannelID, count, "", "", "")
			if err != nil {
				log.Printf("Error fetching messages for purge in channel %s: %v", i.ChannelID, err)
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
					log.Printf("Error deleting single message in channel %s: %v", i.ChannelID, err)
					s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
						Content: "Failed to delete the message.",
					})
					return
				}
			} else {
				// Bulk delete
				err = s.ChannelMessagesBulkDelete(i.ChannelID, messageIDs)
				if err != nil {
					log.Printf("Error bulk deleting messages in channel %s: %v", i.ChannelID, err)
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
					log.Printf("Error logging mod action 'purge': %v", err)
				}

					LogModerationAction(s, database, i.GuildID, "Purge", nil, moderator, fmt.Sprintf("Purged %d messages in <#%s>", len(messageIDs), i.ChannelID))
			}

			// Respond with success
			s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
				Content: fmt.Sprintf("Successfully deleted %d messages.", len(messageIDs)),
			})
		},
	}
}
