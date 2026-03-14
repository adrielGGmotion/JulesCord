package commands

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/bwmarrin/discordgo"
	"julescord/internal/db"
)

func NewSuggestCommand(bot interface{}) *Command {
	var database *db.DB
	if b, ok := bot.(interface{ GetDB() *db.DB }); ok {
		database = b.GetDB()
	}

	return &Command{
		Definition: &discordgo.ApplicationCommand{
			Name:        "suggest",
			Description: "Submit or manage suggestions.",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "setup",
					Description: "Set the channel for suggestions (Admins only)",
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionChannel,
							Name:        "channel",
							Description: "The channel to send suggestions to",
							Required:    true,
							ChannelTypes: []discordgo.ChannelType{
								discordgo.ChannelTypeGuildText,
							},
						},
					},
				},
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "submit",
					Description: "Submit a new suggestion",
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionString,
							Name:        "suggestion",
							Description: "Your suggestion text",
							Required:    true,
						},
					},
				},
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "accept",
					Description: "Accept a suggestion (Mods only)",
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionInteger,
							Name:        "id",
							Description: "The ID of the suggestion to accept",
							Required:    true,
						},
						{
							Type:        discordgo.ApplicationCommandOptionString,
							Name:        "reason",
							Description: "Reason for accepting",
							Required:    false,
						},
					},
				},
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "reject",
					Description: "Reject a suggestion (Mods only)",
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionInteger,
							Name:        "id",
							Description: "The ID of the suggestion to reject",
							Required:    true,
						},
						{
							Type:        discordgo.ApplicationCommandOptionString,
							Name:        "reason",
							Description: "Reason for rejecting",
							Required:    false,
						},
					},
				},
			},
		},
		Handler: func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			if i.GuildID == "" {
				SendError(s, i, "This command can only be used in a server.")
				return
			}
			if database == nil {
				SendError(s, i, "Database is not configured.")
				return
			}

			options := i.ApplicationCommandData().Options
			if len(options) == 0 {
				return
			}
			subCommand := options[0]

			switch subCommand.Name {
			case "setup":
				handleSuggestSetup(s, i, subCommand.Options, database)
			case "submit":
				handleSuggestSubmit(s, i, subCommand.Options, database)
			case "accept":
				handleSuggestAccept(s, i, subCommand.Options, database)
			case "reject":
				handleSuggestReject(s, i, subCommand.Options, database)
			}
		},
	}
}

func handleSuggestSetup(s *discordgo.Session, i *discordgo.InteractionCreate, options []*discordgo.ApplicationCommandInteractionDataOption, database *db.DB) {
	// Require Administrator permission
	if i.Member.Permissions&discordgo.PermissionAdministrator == 0 {
		SendError(s, i, "You need Administrator permissions to setup the suggestion channel.")
		return
	}

	var targetChannel *discordgo.Channel
	for _, opt := range options {
		if opt.Name == "channel" {
			targetChannel = opt.ChannelValue(s)
		}
	}

	if targetChannel == nil {
		SendError(s, i, "Invalid channel provided.")
		return
	}

	err := database.SetSuggestionChannel(context.Background(), i.GuildID, targetChannel.ID)
	if err != nil {
		slog.Error("Failed to set suggestion channel", "guild_id", i.GuildID, "error", err)
		SendError(s, i, "Failed to save suggestion channel configuration.")
		return
	}

	embed := &discordgo.MessageEmbed{
		Title:       "Suggestions Setup Complete",
		Description: fmt.Sprintf("Suggestions will now be posted to <#%s>.", targetChannel.ID),
		Color:       0x00FF00, // Green
	}
	SendEmbed(s, i, embed)
}

func handleSuggestSubmit(s *discordgo.Session, i *discordgo.InteractionCreate, options []*discordgo.ApplicationCommandInteractionDataOption, database *db.DB) {
	channelID, err := database.GetSuggestionChannel(context.Background(), i.GuildID)
	if err != nil || channelID == "" {
		SendError(s, i, "Suggestions are not configured for this server. An admin must use `/suggest setup` first.")
		return
	}

	var content string
	for _, opt := range options {
		if opt.Name == "suggestion" {
			content = opt.StringValue()
		}
	}

	if content == "" {
		SendError(s, i, "Suggestion cannot be empty.")
		return
	}

	// Create embed for the suggestion channel
	suggestionEmbed := &discordgo.MessageEmbed{
		Title:       "New Suggestion",
		Description: content,
		Color:       0xF1C40F, // Yellow/Gold for pending
		Author: &discordgo.MessageEmbedAuthor{
			Name:    i.Member.User.Username,
			IconURL: i.Member.User.AvatarURL(""),
		},
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "Status",
				Value:  "🟡 Pending",
				Inline: true,
			},
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: fmt.Sprintf("User ID: %s", i.Member.User.ID),
		},
	}

	// Send to suggestion channel
	msg, err := s.ChannelMessageSendEmbed(channelID, suggestionEmbed)
	if err != nil {
		slog.Error("Failed to send suggestion to channel", "channel_id", channelID, "error", err)
		SendError(s, i, "Failed to post suggestion. The bot might lack permissions in the suggestion channel.")
		return
	}

	// Save to database
	suggestionID, err := database.CreateSuggestion(context.Background(), i.GuildID, i.Member.User.ID, msg.ID, content)
	if err != nil {
		slog.Error("Failed to save suggestion to DB", "guild_id", i.GuildID, "error", err)
		SendError(s, i, "Failed to save suggestion to database.")
		return
	}

	// Add suggestion ID to the embed footer now that we have it
	suggestionEmbed.Footer.Text = fmt.Sprintf("Suggestion ID: %d | User ID: %s", suggestionID, i.Member.User.ID)
	_, _ = s.ChannelMessageEditEmbed(channelID, msg.ID, suggestionEmbed)

	// Add voting reactions
	err = s.MessageReactionAdd(channelID, msg.ID, "👍")
	if err != nil {
		slog.Error("Failed to add thumbs up reaction", "error", err)
	}
	err = s.MessageReactionAdd(channelID, msg.ID, "👎")
	if err != nil {
		slog.Error("Failed to add thumbs down reaction", "error", err)
	}

	// Inform user of success
	SendEmbed(s, i, &discordgo.MessageEmbed{
		Description: fmt.Sprintf("Your suggestion has been submitted successfully to <#%s>. Suggestion ID: %d", channelID, suggestionID),
		Color:       0x00FF00,
	})
}

func handleSuggestAccept(s *discordgo.Session, i *discordgo.InteractionCreate, options []*discordgo.ApplicationCommandInteractionDataOption, database *db.DB) {
	updateSuggestion(s, i, options, database, "accepted", "🟢 Accepted", 0x2ECC71) // Green
}

func handleSuggestReject(s *discordgo.Session, i *discordgo.InteractionCreate, options []*discordgo.ApplicationCommandInteractionDataOption, database *db.DB) {
	updateSuggestion(s, i, options, database, "rejected", "🔴 Rejected", 0xE74C3C) // Red
}

func updateSuggestion(s *discordgo.Session, i *discordgo.InteractionCreate, options []*discordgo.ApplicationCommandInteractionDataOption, database *db.DB, status string, statusText string, color int) {
	// Check permissions
	if !hasModPermissions(s, i, database) {
		SendError(s, i, "You do not have permission to manage suggestions.")
		return
	}

	var suggestionID int64
	var reason string
	for _, opt := range options {
		if opt.Name == "id" {
			suggestionID = opt.IntValue()
		} else if opt.Name == "reason" {
			reason = opt.StringValue()
		}
	}

	suggestion, err := database.GetSuggestionByID(context.Background(), int(suggestionID))
	if err != nil {
		slog.Error("Failed to fetch suggestion", "id", suggestionID, "error", err)
		SendError(s, i, "An error occurred while fetching the suggestion.")
		return
	}
	if suggestion == nil || suggestion.GuildID != i.GuildID {
		SendError(s, i, "Suggestion not found.")
		return
	}

	if suggestion.Status != "pending" {
		SendError(s, i, fmt.Sprintf("Suggestion is already %s.", suggestion.Status))
		return
	}

	channelID, err := database.GetSuggestionChannel(context.Background(), i.GuildID)
	if err != nil || channelID == "" {
		SendError(s, i, "Suggestion channel configuration not found.")
		return
	}

	// Update DB
	err = database.UpdateSuggestionStatus(context.Background(), int(suggestionID), status)
	if err != nil {
		slog.Error("Failed to update suggestion status in DB", "id", suggestionID, "error", err)
		SendError(s, i, "Failed to update suggestion status in database.")
		return
	}

	// Update the message embed
	msg, err := s.ChannelMessage(channelID, suggestion.MessageID)
	if err != nil {
		slog.Error("Failed to fetch suggestion message", "channel_id", channelID, "message_id", suggestion.MessageID, "error", err)
		SendError(s, i, "Could not find the original suggestion message. DB updated successfully.")
		return
	}

	if len(msg.Embeds) > 0 {
		embed := msg.Embeds[0]
		embed.Color = color

		// Update Status field
		for _, field := range embed.Fields {
			if field.Name == "Status" {
				field.Value = statusText
				break
			}
		}

		// Add Reason field if provided
		if reason != "" {
			embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
				Name:   "Reason (" + i.Member.User.Username + ")",
				Value:  reason,
				Inline: false,
			})
		} else {
			embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
				Name:   "Moderator",
				Value:  i.Member.User.Username,
				Inline: false,
			})
		}

		_, err = s.ChannelMessageEditEmbed(channelID, suggestion.MessageID, embed)
		if err != nil {
			slog.Error("Failed to edit suggestion message", "error", err)
		}
	}

	SendEmbed(s, i, &discordgo.MessageEmbed{
		Description: fmt.Sprintf("Suggestion #%d has been **%s**.", suggestionID, status),
		Color:       color,
	})
}

// hasModPermissions checks if the user has Administrator or the configured Mod Role
func hasModPermissions(s *discordgo.Session, i *discordgo.InteractionCreate, database *db.DB) bool {
	if i.Member.Permissions&discordgo.PermissionAdministrator != 0 {
		return true
	}

	config, err := database.GetGuildConfig(context.Background(), i.GuildID)
	if err == nil && config != nil && config.ModRoleID != nil {
		for _, roleID := range i.Member.Roles {
			if roleID == *config.ModRoleID {
				return true
			}
		}
	}
	return false
}

// strPtr is a helper function that returns a pointer to a string.
