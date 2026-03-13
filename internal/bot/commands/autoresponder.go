package commands

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/bwmarrin/discordgo"
	"julescord/internal/db"
)

// AutoResponder creates the /autoresponder slash command
// botInstance interface{} is used to avoid circular dependency with internal/bot package
func AutoResponder(database *db.DB, botInstance interface{}) *Command {
	return &Command{
		Definition: &discordgo.ApplicationCommand{
			Name:        "autoresponder",
			Description: "Manage auto-responders for the server",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "add",
					Description: "Add a new auto-responder or update an existing one",
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionString,
							Name:        "trigger",
							Description: "The word or phrase to trigger the response",
							Required:    true,
						},
						{
							Type:        discordgo.ApplicationCommandOptionString,
							Name:        "response",
							Description: "The text to respond with",
							Required:    true,
						},
					},
				},
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "remove",
					Description: "Remove an auto-responder",
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionString,
							Name:        "trigger",
							Description: "The exact trigger word/phrase to remove",
							Required:    true,
						},
					},
				},
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "list",
					Description: "List all auto-responders in this server",
				},
			},
			DefaultMemberPermissions: func() *int64 {
				p := int64(discordgo.PermissionManageMessages)
				return &p
			}(),
		},
		Handler: func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			if database == nil {
				SendError(s, i, "Database connection is not available.")
				return
			}

			if i.GuildID == "" {
				SendError(s, i, "This command can only be used in a server.")
				return
			}

			options := i.ApplicationCommandData().Options
			if len(options) == 0 {
				SendError(s, i, "Please provide a subcommand.")
				return
			}

			subcommand := options[0]

			switch subcommand.Name {
			case "add":
				handleAddAutoResponder(s, i, database, subcommand.Options, botInstance)
			case "remove":
				handleRemoveAutoResponder(s, i, database, subcommand.Options, botInstance)
			case "list":
				handleListAutoResponders(s, i, database)
			default:
				SendError(s, i, "Unknown subcommand.")
			}
		},
	}
}

func handleAddAutoResponder(s *discordgo.Session, i *discordgo.InteractionCreate, database *db.DB, options []*discordgo.ApplicationCommandInteractionDataOption, botInstance interface{}) {
	var trigger, response string

	for _, opt := range options {
		if opt.Name == "trigger" {
			trigger = strings.ToLower(strings.TrimSpace(opt.StringValue()))
		} else if opt.Name == "response" {
			response = opt.StringValue()
		}
	}

	if trigger == "" || response == "" {
		SendError(s, i, "Both trigger and response are required.")
		return
	}

	err := database.AddAutoResponder(context.Background(), i.GuildID, trigger, response)
	if err != nil {
		slog.Error("Failed to add auto-responder", "error", err)
		SendError(s, i, "Failed to save the auto-responder.")
		return
	}

	// Update cache
	updateCache(botInstance, i.GuildID, database)

	SendEmbed(s, i, &discordgo.MessageEmbed{
		Title:       "Auto-Responder Added",
		Description: fmt.Sprintf("Successfully added auto-responder for **%s**", trigger),
		Color:       0x00FF00, // Green
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:  "Response",
				Value: response,
			},
		},
	})
}

func handleRemoveAutoResponder(s *discordgo.Session, i *discordgo.InteractionCreate, database *db.DB, options []*discordgo.ApplicationCommandInteractionDataOption, botInstance interface{}) {
	var trigger string

	for _, opt := range options {
		if opt.Name == "trigger" {
			trigger = strings.ToLower(strings.TrimSpace(opt.StringValue()))
		}
	}

	if trigger == "" {
		SendError(s, i, "Trigger is required.")
		return
	}

	err := database.RemoveAutoResponder(context.Background(), i.GuildID, trigger)
	if err != nil {
		slog.Error("Failed to remove auto-responder", "error", err)
		SendError(s, i, "Failed to remove the auto-responder.")
		return
	}

	// Update cache
	updateCache(botInstance, i.GuildID, database)

	SendEmbed(s, i, &discordgo.MessageEmbed{
		Title:       "Auto-Responder Removed",
		Description: fmt.Sprintf("Successfully removed auto-responder for **%s**", trigger),
		Color:       0xFF0000, // Red
	})
}

func handleListAutoResponders(s *discordgo.Session, i *discordgo.InteractionCreate, database *db.DB) {
	responders, err := database.ListAutoResponders(context.Background(), i.GuildID)
	if err != nil {
		slog.Error("Failed to list auto-responders", "error", err)
		SendError(s, i, "Failed to retrieve auto-responders.")
		return
	}

	if len(responders) == 0 {
		SendEmbed(s, i, &discordgo.MessageEmbed{
			Title:       "Auto-Responders",
			Description: "There are no auto-responders configured for this server.",
			Color:       0x3498DB, // Blue
		})
		return
	}

	var description strings.Builder
	for _, r := range responders {
		description.WriteString(fmt.Sprintf("**Trigger:** %s\n**Response:** %s\n\n", r.TriggerWord, r.Response))
	}

	// Discord embed description limit is 4096 characters, truncate if necessary
	descStr := description.String()
	if len(descStr) > 4096 {
		descStr = descStr[:4093] + "..."
	}

	SendEmbed(s, i, &discordgo.MessageEmbed{
		Title:       fmt.Sprintf("Auto-Responders (%d)", len(responders)),
		Description: descStr,
		Color:       0x3498DB, // Blue
	})
}
