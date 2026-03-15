package commands

import (
	"context"
	"fmt"
	"log/slog"

	"julescord/internal/db"

	"github.com/bwmarrin/discordgo"
)

// Modmail returns the modmail command and its subcommands.
func Modmail(database *db.DB) *Command {
	return &Command{
		Definition: &discordgo.ApplicationCommand{
			Name:        "modmail",
			Description: "Manage the modmail system",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        "setup",
					Description: "Set the category where new modmail threads will be created",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionChannel,
							Name:        "category",
							Description: "The category to create threads in",
							Required:    true,
							ChannelTypes: []discordgo.ChannelType{
								discordgo.ChannelTypeGuildCategory,
							},
						},
					},
				},
				{
					Name:        "reply",
					Description: "Reply to a modmail thread (must be used in the thread channel)",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionString,
							Name:        "message",
							Description: "The message to send to the user",
							Required:    true,
						},
					},
				},
				{
					Name:        "close",
					Description: "Close the current modmail thread",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
				},
			},
		},
		Handler: func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			if database == nil {
				SendError(s, i, "Database connection not available")
				return
			}

			options := i.ApplicationCommandData().Options
			if len(options) == 0 {
				SendError(s, i, "Invalid command usage")
				return
			}

			subcommand := options[0].Name

			switch subcommand {
			case "setup":
				handleModmailSetup(s, i, database, options[0].Options)
			case "reply":
				handleModmailReply(s, i, database, options[0].Options)
			case "close":
				handleModmailClose(s, i, database)
			}
		},
	}
}

func handleModmailSetup(s *discordgo.Session, i *discordgo.InteractionCreate, database *db.DB, options []*discordgo.ApplicationCommandInteractionDataOption) {
	// Check permissions
	member := i.Member
	if member == nil || (member.Permissions&discordgo.PermissionAdministrator) == 0 {
		SendError(s, i, "You need Administrator permissions to use this command.")
		return
	}

	categoryID := options[0].ChannelValue(s).ID

	err := database.SetModmailConfig(context.Background(), i.GuildID, categoryID)
	if err != nil {
		slog.Error("Failed to set modmail config", "error", err)
		SendError(s, i, "Failed to set modmail category.")
		return
	}

	SendEmbed(s, i, &discordgo.MessageEmbed{
		Title:       "Modmail Configured",
		Description: fmt.Sprintf("Modmail threads will now be created in the <#%s> category.", categoryID),
		Color:       0x00FF00,
	})
}

func handleModmailReply(s *discordgo.Session, i *discordgo.InteractionCreate, database *db.DB, options []*discordgo.ApplicationCommandInteractionDataOption) {
	// Check if this channel is a modmail thread
	thread, err := database.GetModmailThreadByChannel(context.Background(), i.ChannelID)
	if err != nil {
		slog.Error("Failed to fetch modmail thread", "error", err)
		SendError(s, i, "Failed to check thread status.")
		return
	}

	if thread == nil {
		SendError(s, i, "This command can only be used in an active modmail thread channel.")
		return
	}

	messageContent := options[0].StringValue()

	// Defer response since sending DM might take time
	err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
	})
	if err != nil {
		slog.Error("Failed to defer interaction response", "error", err)
		return
	}

	// Fetch the user to get their DM channel
	user, err := s.User(thread.UserID)
	if err != nil {
		slog.Error("Failed to fetch user for modmail reply", "error", err)
		_, _ = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: stringPtr("❌ Failed to find the user to send the reply."),
		})
		return
	}

	dmChannel, err := s.UserChannelCreate(user.ID)
	if err != nil {
		slog.Error("Failed to create DM channel for modmail reply", "error", err)
		_, _ = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: stringPtr("❌ Failed to open a DM channel with the user. They might have DMs disabled."),
		})
		return
	}

	// Send message to the user
	embed := &discordgo.MessageEmbed{
		Title:       "Support Response",
		Description: messageContent,
		Color:       0x5865F2, // Discord Blurple
		Footer: &discordgo.MessageEmbedFooter{
			Text: fmt.Sprintf("From %s", i.Member.User.String()),
			IconURL: i.Member.User.AvatarURL(""),
		},
	}

	_, err = s.ChannelMessageSendEmbed(dmChannel.ID, embed)
	if err != nil {
		slog.Error("Failed to send modmail reply via DM", "error", err)
		_, _ = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: stringPtr("❌ Failed to send the message. The user might have DMs disabled."),
		})
		return
	}

	// Confirm to the moderator
	_, _ = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Content: stringPtr(fmt.Sprintf("✅ **Message sent to <@%s>**:\n%s", user.ID, messageContent)),
	})
}

func handleModmailClose(s *discordgo.Session, i *discordgo.InteractionCreate, database *db.DB) {
	thread, err := database.GetModmailThreadByChannel(context.Background(), i.ChannelID)
	if err != nil {
		slog.Error("Failed to fetch modmail thread for closing", "error", err)
		SendError(s, i, "Failed to check thread status.")
		return
	}

	if thread == nil {
		SendError(s, i, "This command can only be used in an active modmail thread channel.")
		return
	}

	// Mark as closed in DB
	err = database.CloseModmailThread(context.Background(), i.ChannelID)
	if err != nil {
		slog.Error("Failed to close modmail thread in DB", "error", err)
		SendError(s, i, "Failed to close the thread in the database.")
		return
	}

	// Try to notify the user
	user, err := s.User(thread.UserID)
	if err == nil {
		dmChannel, err := s.UserChannelCreate(user.ID)
		if err == nil {
			embed := &discordgo.MessageEmbed{
				Title:       "Thread Closed",
				Description: "Your support thread has been closed. If you need further assistance, you can send another message to open a new thread.",
				Color:       0xED4245, // Red
			}
			_, _ = s.ChannelMessageSendEmbed(dmChannel.ID, embed)
		}
	}

	SendEmbed(s, i, &discordgo.MessageEmbed{
		Description: "✅ Modmail thread closed. Deleting channel in 5 seconds...",
		Color:       0xFF0000,
	})

	// Delete channel
	_, err = s.ChannelDelete(i.ChannelID)
	if err != nil {
		slog.Error("Failed to delete modmail channel", "error", err)
	}
}
