package commands

import (
	"context"
	"fmt"
	"log/slog"
	"regexp"
	"strings"

	"julescord/internal/db"

	"github.com/bwmarrin/discordgo"
)

var messageLinkRegex = regexp.MustCompile(`https?://(?:ptb\.|canary\.)?discord\.com/channels/(\d+)/(\d+)/(\d+)`)

func HighlightCommand(database *db.DB) *Command {
	return &Command{
		Definition: &discordgo.ApplicationCommand{
			Name:        "highlight",
			Description: "Manage server highlights",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        "add",
					Description: "Add a message to the server highlights",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Name:        "message_link",
							Description: "The link to the Discord message",
							Type:        discordgo.ApplicationCommandOptionString,
							Required:    true,
						},
					},
				},
				{
					Name:        "list",
					Description: "List the server highlights",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
				},
				{
					Name:        "remove",
					Description: "Remove a message from the server highlights",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Name:        "id",
							Description: "The ID of the highlight to remove",
							Type:        discordgo.ApplicationCommandOptionInteger,
							Required:    true,
						},
					},
				},
			},
		},
		Handler: func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			options := i.ApplicationCommandData().Options
			if len(options) == 0 {
				return
			}

			subcommand := options[0].Name
			if subcommand == "add" {
				handleAddHighlight(s, i, database, options[0].Options)
			} else if subcommand == "list" {
				handleListHighlights(s, i, database)
			} else if subcommand == "remove" {
				handleRemoveHighlight(s, i, database, options[0].Options)
			}
		},
	}
}

func handleAddHighlight(s *discordgo.Session, i *discordgo.InteractionCreate, database *db.DB, options []*discordgo.ApplicationCommandInteractionDataOption) {
	if len(options) == 0 {
		return
	}
	messageLink := options[0].StringValue()

	matches := messageLinkRegex.FindStringSubmatch(messageLink)
	if len(matches) != 4 {
		SendError(s, i, "Invalid message link. Please provide a valid Discord message link.")
		return
	}

	linkGuildID := matches[1]
	linkChannelID := matches[2]
	linkMessageID := matches[3]

	if linkGuildID != i.GuildID {
		SendError(s, i, "You can only highlight messages from this server.")
		return
	}

	msg, err := s.ChannelMessage(linkChannelID, linkMessageID)
	if err != nil {
		slog.Error("Failed to fetch message for highlight", "error", err)
		SendError(s, i, "Failed to fetch the message. Make sure the link is correct and the bot has access to that channel.")
		return
	}

	userID := ""
	if i.Member != nil {
		userID = i.Member.User.ID
	} else {
		userID = i.User.ID
	}

	err = database.AddHighlight(context.Background(), i.GuildID, linkMessageID, linkChannelID, msg.Author.ID, userID)
	if err != nil {
		slog.Error("Failed to add highlight to database", "error", err)
		SendError(s, i, "Failed to save the highlight. It might already be highlighted.")
		return
	}

	embed := &discordgo.MessageEmbed{
		Title:       "Message Highlighted",
		Description: fmt.Sprintf("[Jump to Message](%s)", messageLink),
		Color:       0x00FF00, // Green
	}
	err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
		},
	})
	if err != nil {
		slog.Error("Failed to send add highlight response", "error", err)
	}
}

func handleListHighlights(s *discordgo.Session, i *discordgo.InteractionCreate, database *db.DB) {
	highlights, err := database.GetHighlights(context.Background(), i.GuildID)
	if err != nil {
		slog.Error("Failed to get highlights from database", "error", err)
		SendError(s, i, "Failed to retrieve highlights.")
		return
	}

	if len(highlights) == 0 {
		SendError(s, i, "There are no highlighted messages in this server.")
		return
	}

	var description strings.Builder
	count := 0
	for _, h := range highlights {
		link := fmt.Sprintf("https://discord.com/channels/%s/%s/%s", h.GuildID, h.ChannelID, h.MessageID)
		line := fmt.Sprintf("**ID:** %d | [Jump to Message](%s) | Added by <@%s>\n", h.ID, link, h.AddedBy)
		if description.Len()+len(line) > 4000 {
			description.WriteString("\n*...and more.*")
			break
		}
		description.WriteString(line)
		count++
	}

	embed := &discordgo.MessageEmbed{
		Title:       "Server Highlights",
		Description: description.String(),
		Color:       0xFFD700, // Gold
	}
	err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
		},
	})
	if err != nil {
		slog.Error("Failed to send list highlights response", "error", err)
	}
}

func handleRemoveHighlight(s *discordgo.Session, i *discordgo.InteractionCreate, database *db.DB, options []*discordgo.ApplicationCommandInteractionDataOption) {
	if len(options) == 0 {
		return
	}
	id := int(options[0].IntValue())

	// Only allow server administrators to remove highlights
	hasPerm := false
	if i.Member != nil && (i.Member.Permissions&discordgo.PermissionAdministrator == discordgo.PermissionAdministrator) {
		hasPerm = true
	}
	if !hasPerm {
		SendError(s, i, "You must be a server administrator to remove highlights.")
		return
	}

	err := database.RemoveHighlight(context.Background(), id, i.GuildID)
	if err != nil {
		slog.Error("Failed to remove highlight from database", "error", err)
		if strings.Contains(err.Error(), "highlight not found") {
			SendError(s, i, "Highlight not found or you don't have permission to remove it in this server.")
		} else {
			SendError(s, i, "An error occurred while removing the highlight.")
		}
		return
	}

	embed := &discordgo.MessageEmbed{
		Title:       "Highlight Removed",
		Description: fmt.Sprintf("Successfully removed highlight with ID **%d**.", id),
		Color:       0x00FF00, // Green
	}
	err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
		},
	})
	if err != nil {
		slog.Error("Failed to send remove highlight response", "error", err)
	}
}