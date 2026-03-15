package commands

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"julescord/internal/db"

	"github.com/bwmarrin/discordgo"
)

// Quote returns the quote slash command.
func Quote(database *db.DB) *Command {
	return &Command{
		Definition: &discordgo.ApplicationCommand{
			Name:        "quote",
			Description: "Save and retrieve quotes",
			DMPermission: new(bool),
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        "add",
					Description: "Save a new quote",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Name:        "user",
							Description: "The user being quoted",
							Type:        discordgo.ApplicationCommandOptionUser,
							Required:    true,
						},
						{
							Name:        "content",
							Description: "The quote content",
							Type:        discordgo.ApplicationCommandOptionString,
							Required:    true,
						},
					},
				},
				{
					Name:        "get",
					Description: "Get a specific quote by ID",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Name:        "id",
							Description: "The quote ID",
							Type:        discordgo.ApplicationCommandOptionInteger,
							Required:    true,
						},
					},
				},
				{
					Name:        "random",
					Description: "Get a random quote",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
				},
				{
					Name:        "delete",
					Description: "Delete a quote by ID",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Name:        "id",
							Description: "The quote ID",
							Type:        discordgo.ApplicationCommandOptionInteger,
							Required:    true,
						},
					},
				},
			},
		},
		Handler: func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			options := i.ApplicationCommandData().Options
			subcommand := options[0].Name

			switch subcommand {
			case "add":
				handleAddQuote(s, i, database)
			case "get":
				handleGetQuote(s, i, database)
			case "random":
				handleRandomQuote(s, i, database)
			case "delete":
				handleDeleteQuote(s, i, database)
			}
		},
	}
}

func handleAddQuote(s *discordgo.Session, i *discordgo.InteractionCreate, database *db.DB) {
	options := i.ApplicationCommandData().Options[0].Options
	var targetUser *discordgo.User
	var content string

	for _, opt := range options {
		switch opt.Name {
		case "user":
			targetUser = opt.UserValue(s)
		case "content":
			content = opt.StringValue()
		}
	}

	guildID := i.GuildID
	userID := i.Member.User.ID
	authorID := targetUser.ID

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	id, err := database.AddQuote(ctx, guildID, userID, authorID, content)
	if err != nil {
		slog.Error("Failed to add quote", "error", err)
		SendError(s, i, "Failed to save the quote.")
		return
	}

	embed := &discordgo.MessageEmbed{
		Title:       fmt.Sprintf("Quote #%d Saved!", id),
		Description: fmt.Sprintf("\"%s\"\n\n— <@%s>", content, authorID),
		Color:       0x00FF00,
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
		},
	})
}

func handleGetQuote(s *discordgo.Session, i *discordgo.InteractionCreate, database *db.DB) {
	options := i.ApplicationCommandData().Options[0].Options
	id := int(options[0].IntValue())
	guildID := i.GuildID

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	quote, err := database.GetQuote(ctx, id, guildID)
	if err != nil {
		slog.Error("Failed to get quote", "error", err)
		SendError(s, i, "Failed to retrieve the quote.")
		return
	}

	if quote == nil {
		SendError(s, i, "Quote not found.")
		return
	}

	sendQuoteEmbed(s, i, quote)
}

func handleRandomQuote(s *discordgo.Session, i *discordgo.InteractionCreate, database *db.DB) {
	guildID := i.GuildID

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	quote, err := database.GetRandomQuote(ctx, guildID)
	if err != nil {
		slog.Error("Failed to get random quote", "error", err)
		SendError(s, i, "Failed to retrieve a random quote.")
		return
	}

	if quote == nil {
		SendError(s, i, "No quotes found for this server.")
		return
	}

	sendQuoteEmbed(s, i, quote)
}

func handleDeleteQuote(s *discordgo.Session, i *discordgo.InteractionCreate, database *db.DB) {
	options := i.ApplicationCommandData().Options[0].Options
	id := int(options[0].IntValue())
	guildID := i.GuildID

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Check if the user is an admin or the creator of the quote
	quote, err := database.GetQuote(ctx, id, guildID)
	if err != nil {
		slog.Error("Failed to get quote for deletion", "error", err)
		SendError(s, i, "Failed to check the quote.")
		return
	}
	if quote == nil {
		SendError(s, i, "Quote not found.")
		return
	}

	hasPermission := i.Member.Permissions&discordgo.PermissionAdministrator != 0 || quote.UserID == i.Member.User.ID
	if !hasPermission {
		SendError(s, i, "You do not have permission to delete this quote. You must be an administrator or the creator of the quote.")
		return
	}

	err = database.DeleteQuote(ctx, id, guildID)
	if err != nil {
		slog.Error("Failed to delete quote", "error", err)
		SendError(s, i, "Failed to delete the quote.")
		return
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("✅ Quote #%d deleted.", id),
		},
	})
}

func sendQuoteEmbed(s *discordgo.Session, i *discordgo.InteractionCreate, quote *db.Quote) {
	embed := &discordgo.MessageEmbed{
		Description: fmt.Sprintf("\"%s\"\n\n— <@%s>", quote.Content, quote.AuthorID),
		Color:       0x3498DB,
		Footer: &discordgo.MessageEmbedFooter{
			Text: fmt.Sprintf("Quote #%d | Saved by %s", quote.ID, quote.UserID),
		},
		Timestamp: quote.CreatedAt.Format(time.RFC3339),
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
		},
	})
}
