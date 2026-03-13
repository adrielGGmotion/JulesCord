package commands

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"julescord/internal/db"

	"github.com/bwmarrin/discordgo"
)

// Ticket creates the ticket command to create and close tickets.
func Ticket(database *db.DB) *Command {
	return &Command{
		Definition: &discordgo.ApplicationCommand{
			Name:        "ticket",
			Description: "Ticket system commands",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        "create",
					Description: "Create a new support ticket",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionString,
							Name:        "reason",
							Description: "The reason for the ticket",
							Required:    true,
						},
					},
				},
				{
					Name:        "close",
					Description: "Close the current ticket",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
				},
			},
		},
		Handler: func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			if database == nil {
				SendError(s, i, "Database is not configured.")
				return
			}

			if i.GuildID == "" {
				SendError(s, i, "This command can only be used in a server.")
				return
			}

			subcommand := i.ApplicationCommandData().Options[0]
			switch subcommand.Name {
			case "create":
				handleCreateTicket(s, i, database, subcommand.Options)
			case "close":
				handleCloseTicket(s, i, database)
			}
		},
	}
}

func handleCreateTicket(s *discordgo.Session, i *discordgo.InteractionCreate, database *db.DB, options []*discordgo.ApplicationCommandInteractionDataOption) {
	reason := options[0].StringValue()

	var userID string
	var userName string
	if i.Member != nil {
		userID = i.Member.User.ID
		userName = i.Member.User.Username
	} else {
		userID = i.User.ID
		userName = i.User.Username
	}

	// Defer the response as channel creation might take a moment
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags: discordgo.MessageFlagsEphemeral,
		},
	})
	if err != nil {
		slog.Error("Failed to defer interaction response", "error", err)
		return
	}

	// Create a new text channel for the ticket
	channelName := fmt.Sprintf("ticket-%s", userName)

	// Determine category (optional, but good practice. For now, create at the root level or in the same category)
	var parentID string
	if i.ChannelID != "" {
		channel, err := s.Channel(i.ChannelID)
		if err == nil {
			parentID = channel.ParentID
		}
	}

	permissionOverwrites := []*discordgo.PermissionOverwrite{
		{
			ID:    i.GuildID, // @everyone role ID is the same as the guild ID
			Type:  discordgo.PermissionOverwriteTypeRole,
			Deny:  discordgo.PermissionViewChannel,
		},
		{
			ID:    s.State.User.ID, // Bot
			Type:  discordgo.PermissionOverwriteTypeMember,
			Allow: discordgo.PermissionViewChannel | discordgo.PermissionSendMessages,
		},
		{
			ID:    userID, // The user who created the ticket
			Type:  discordgo.PermissionOverwriteTypeMember,
			Allow: discordgo.PermissionViewChannel | discordgo.PermissionSendMessages | discordgo.PermissionReadMessageHistory,
		},
	}

	newChannel, err := s.GuildChannelCreateComplex(i.GuildID, discordgo.GuildChannelCreateData{
		Name:                 channelName,
		Type:                 discordgo.ChannelTypeGuildText,
		ParentID:             parentID,
		PermissionOverwrites: permissionOverwrites,
	})
	if err != nil {
		slog.Error("Failed to create ticket channel", "error", err)
		content := "Failed to create ticket channel. Make sure the bot has `Manage Channels` permissions."
		_, _ = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: &content,
		})
		return
	}

	// Save the ticket in the database
	err = database.CreateTicket(context.Background(), i.GuildID, userID, newChannel.ID, reason)
	if err != nil {
		slog.Error("Failed to create ticket record in DB", "error", err)
		// Clean up the channel if DB insertion fails
		_, _ = s.ChannelDelete(newChannel.ID)
		content := "Failed to save ticket data. Please try again."
		_, _ = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: &content,
		})
		return
	}

	// Send welcome message in the new ticket channel
	embed := &discordgo.MessageEmbed{
		Title:       "🎫 Support Ticket",
		Description: fmt.Sprintf("Welcome <@%s>!\n\n**Reason:** %s\n\nPlease describe your issue and a staff member will be with you shortly.\nTo close this ticket, use `/ticket close`.", userID, reason),
		Color:       0x3B82F6, // Blue
	}
	_, err = s.ChannelMessageSendEmbed(newChannel.ID, embed)
	if err != nil {
		slog.Error("Failed to send welcome embed in ticket channel", "error", err)
	}

	// Update the original response
	content := fmt.Sprintf("Ticket created successfully! Please check <#%s>.", newChannel.ID)
	_, _ = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Content: &content,
	})
}

func handleCloseTicket(s *discordgo.Session, i *discordgo.InteractionCreate, database *db.DB) {
	// Verify if the current channel is a ticket channel
	ticket, err := database.GetTicketByChannel(context.Background(), i.ChannelID)
	if err != nil {
		slog.Error("Failed to fetch ticket by channel", "error", err)
		SendError(s, i, "Failed to check ticket status.")
		return
	}

	if ticket == nil {
		SendError(s, i, "This channel is not a registered ticket.")
		return
	}

	if ticket.Status == "closed" {
		SendError(s, i, "This ticket is already closed.")
		return
	}

	// Mark ticket as closed in DB
	err = database.CloseTicket(context.Background(), i.ChannelID)
	if err != nil {
		slog.Error("Failed to close ticket in DB", "error", err)
		SendError(s, i, "Failed to update ticket status in database.")
		return
	}

	// Send closing message
	err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "Closing ticket in 5 seconds...",
		},
	})
	if err != nil {
		slog.Error("Failed to respond to close interaction", "error", err)
	}

	// Delete channel after delay in a goroutine
	go func() {
		time.Sleep(5 * time.Second)
		_, err := s.ChannelDelete(i.ChannelID)
		if err != nil {
			slog.Error("Failed to delete ticket channel", "error", err)
		}
	}()
}
