package commands

import (
	"context"
	"fmt"
	"julescord/internal/db"
	"strconv"
	"strings"

	"github.com/bwmarrin/discordgo"
)

// Embed returns the /embed command
func Embed(database *db.DB) *Command {
	return &Command{
		Definition: &discordgo.ApplicationCommand{
			Name:        "embed",
			Description: "Manage and send custom embeds",
			DMPermission: func(b bool) *bool { return &b }(false),
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        "create",
					Description: "Create a new custom embed",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Name:        "name",
							Description: "The internal name for this embed",
							Type:        discordgo.ApplicationCommandOptionString,
							Required:    true,
						},
						{
							Name:        "title",
							Description: "The title of the embed",
							Type:        discordgo.ApplicationCommandOptionString,
							Required:    true,
						},
						{
							Name:        "description",
							Description: "The description of the embed",
							Type:        discordgo.ApplicationCommandOptionString,
							Required:    true,
						},
						{
							Name:        "color",
							Description: "The hex color code (e.g. #FF0000)",
							Type:        discordgo.ApplicationCommandOptionString,
							Required:    true,
						},
					},
				},
				{
					Name:        "list",
					Description: "List all custom embeds for this server",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
				},
				{
					Name:        "send",
					Description: "Send a custom embed to the current channel",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Name:        "name",
							Description: "The internal name of the embed to send",
							Type:        discordgo.ApplicationCommandOptionString,
							Required:    true,
						},
					},
				},
				{
					Name:        "delete",
					Description: "Delete a custom embed",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Name:        "name",
							Description: "The internal name of the embed to delete",
							Type:        discordgo.ApplicationCommandOptionString,
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
			case "create":
				handleEmbedCreate(s, i, database)
			case "list":
				handleEmbedList(s, i, database)
			case "send":
				handleEmbedSend(s, i, database)
			case "delete":
				handleEmbedDelete(s, i, database)
			}
		},
	}
}

func handleEmbedCreate(s *discordgo.Session, i *discordgo.InteractionCreate, database *db.DB) {
	perm, err := s.UserChannelPermissions(i.Member.User.ID, i.ChannelID)
	if err != nil || perm&discordgo.PermissionManageMessages == 0 {
		SendError(s, i, "You must have the Manage Messages permission to create embeds.")
		return
	}

	options := i.ApplicationCommandData().Options[0].Options
	var name, title, description, color string

	for _, opt := range options {
		switch opt.Name {
		case "name":
			name = opt.StringValue()
		case "title":
			title = opt.StringValue()
		case "description":
			description = opt.StringValue()
		case "color":
			color = opt.StringValue()
		}
	}

	if !strings.HasPrefix(color, "#") {
		color = "#" + color
	}

	_, err = strconv.ParseInt(strings.TrimPrefix(color, "#"), 16, 64)
	if err != nil {
		SendError(s, i, "Invalid color code. Please use a valid hex color code (e.g. #FF0000).")
		return
	}

	err = database.AddCustomEmbed(context.Background(), i.GuildID, name, title, description, color)
	if err != nil {
		SendError(s, i, "Failed to create custom embed.")
		return
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("Successfully created custom embed **%s**.", name),
		},
	})
}

func handleEmbedList(s *discordgo.Session, i *discordgo.InteractionCreate, database *db.DB) {
	embeds, err := database.ListCustomEmbeds(context.Background(), i.GuildID)
	if err != nil {
		SendError(s, i, "Failed to fetch custom embeds.")
		return
	}

	if len(embeds) == 0 {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "No custom embeds found for this server.",
			},
		})
		return
	}

	var list strings.Builder
	for _, e := range embeds {
		list.WriteString(fmt.Sprintf("**%s**\nTitle: %s\nColor: %s\n\n", e.Name, e.Title, e.Color))
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{
				{
					Title:       "Custom Embeds",
					Description: list.String(),
					Color:       0x00FF00,
				},
			},
		},
	})
}

func handleEmbedSend(s *discordgo.Session, i *discordgo.InteractionCreate, database *db.DB) {
	perm, err := s.UserChannelPermissions(i.Member.User.ID, i.ChannelID)
	if err != nil || perm&discordgo.PermissionManageMessages == 0 {
		SendError(s, i, "You must have the Manage Messages permission to send embeds.")
		return
	}

	options := i.ApplicationCommandData().Options[0].Options
	name := options[0].StringValue()

	e, err := database.GetCustomEmbed(context.Background(), i.GuildID, name)
	if err != nil {
		SendError(s, i, "Failed to fetch custom embed.")
		return
	}

	if e == nil {
		SendError(s, i, fmt.Sprintf("Custom embed **%s** not found.", name))
		return
	}

	colorInt, _ := strconv.ParseInt(strings.TrimPrefix(e.Color, "#"), 16, 64)

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{
				{
					Title:       e.Title,
					Description: e.Description,
					Color:       int(colorInt),
				},
			},
		},
	})
}

func handleEmbedDelete(s *discordgo.Session, i *discordgo.InteractionCreate, database *db.DB) {
	perm, err := s.UserChannelPermissions(i.Member.User.ID, i.ChannelID)
	if err != nil || perm&discordgo.PermissionManageMessages == 0 {
		SendError(s, i, "You must have the Manage Messages permission to delete embeds.")
		return
	}

	options := i.ApplicationCommandData().Options[0].Options
	name := options[0].StringValue()

	err = database.DeleteCustomEmbed(context.Background(), i.GuildID, name)
	if err != nil {
		SendError(s, i, "Failed to delete custom embed.")
		return
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("Successfully deleted custom embed **%s**.", name),
		},
	})
}
