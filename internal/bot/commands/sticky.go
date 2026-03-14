package commands

import (
	"context"
	"log/slog"
	"time"

	"github.com/bwmarrin/discordgo"
	"julescord/internal/db"
)

// BotInterface defines the methods from the main Bot struct that commands need
type BotInterface interface {
	GetDB() *db.DB
}

// NewStickyCommand creates a new sticky command
func NewStickyCommand(botInstance interface{}) *Command {
	b, ok := botInstance.(BotInterface)
	if !ok {
		slog.Error("Failed to assert bot instance in NewStickyCommand")
		return nil
	}

	var memberPermissions int64 = discordgo.PermissionManageMessages

	return &Command{
		Definition: &discordgo.ApplicationCommand{
			Name:                     "sticky",
			Description:              "Manage sticky messages in the current channel",
			DefaultMemberPermissions: &memberPermissions,
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        "set",
					Description: "Set a sticky message for this channel",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionString,
							Name:        "message",
							Description: "The message text to stick to the bottom of the channel",
							Required:    true,
						},
					},
				},
				{
					Name:        "remove",
					Description: "Remove the sticky message from this channel",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
				},
			},
		},
		Handler: func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			database := b.GetDB()
			if database == nil {
				SendError(s, i, "Database connection is not available.")
				return
			}

			options := i.ApplicationCommandData().Options
			if len(options) == 0 {
				return
			}

			subcommand := options[0]
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			channelID := i.ChannelID
			guildID := i.GuildID

			switch subcommand.Name {
			case "set":
				messageText := subcommand.Options[0].StringValue()

				// Delete any existing sticky message in the channel first if there is one
				existingSticky, err := database.GetSticky(ctx, channelID)
				if err == nil && existingSticky != nil && existingSticky.LastMessageID != "" {
					_ = s.ChannelMessageDelete(channelID, existingSticky.LastMessageID)
				}

				// Set the sticky message in the database
				err = database.SetSticky(ctx, channelID, guildID, messageText)
				if err != nil {
					slog.Error("Failed to set sticky message", "error", err, "channel_id", channelID)
					SendError(s, i, "Failed to set sticky message in the database.")
					return
				}

				// Send the sticky message now so it's at the bottom
				msg, err := s.ChannelMessageSend(channelID, messageText)
				if err == nil {
					// Update the DB with the new message ID
					_ = database.UpdateStickyMessageID(context.Background(), channelID, msg.ID)
				} else {
					slog.Error("Failed to send initial sticky message", "error", err, "channel_id", channelID)
				}

				// Acknowledge the interaction ephemerally
				_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: "Sticky message set for this channel.",
						Flags:   discordgo.MessageFlagsEphemeral,
					},
				})

			case "remove":
				// Get the current sticky to delete the message if possible
				existingSticky, err := database.GetSticky(ctx, channelID)
				if err == nil && existingSticky != nil && existingSticky.LastMessageID != "" {
					_ = s.ChannelMessageDelete(channelID, existingSticky.LastMessageID)
				} else if err != nil {
					slog.Error("Failed to fetch existing sticky message during removal", "error", err)
				}

				err = database.RemoveSticky(ctx, channelID)
				if err != nil {
					slog.Error("Failed to remove sticky message", "error", err, "channel_id", channelID)
					SendError(s, i, "Failed to remove sticky message.")
					return
				}

				_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: "Sticky message removed from this channel.",
						Flags:   discordgo.MessageFlagsEphemeral,
					},
				})
			}
		},
	}
}
