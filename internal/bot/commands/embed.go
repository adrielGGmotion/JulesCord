package commands

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"julescord/internal/db"

	"github.com/bwmarrin/discordgo"
)

// Embed creates the /embed command.
func Embed(database *db.DB) *Command {
	return &Command{
		Definition: &discordgo.ApplicationCommand{
			Name:        "embed",
			Description: "Create, view, delete, or list custom embeds.",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        "create",
					Description: "Create a new custom embed",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionString,
							Name:        "name",
							Description: "The name to identify this embed",
							Required:    true,
						},
						{
							Type:        discordgo.ApplicationCommandOptionString,
							Name:        "title",
							Description: "The title of the embed",
							Required:    false,
						},
						{
							Type:        discordgo.ApplicationCommandOptionString,
							Name:        "description",
							Description: "The description of the embed",
							Required:    false,
						},
						{
							Type:        discordgo.ApplicationCommandOptionString,
							Name:        "color",
							Description: "The color hex code (e.g. #FF0000)",
							Required:    false,
						},
					},
				},
				{
					Name:        "view",
					Description: "View a custom embed",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionString,
							Name:        "name",
							Description: "The name of the embed to view",
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
							Type:        discordgo.ApplicationCommandOptionString,
							Name:        "name",
							Description: "The name of the embed to delete",
							Required:    true,
						},
					},
				},
				{
					Name:        "list",
					Description: "List all custom embeds in this server",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
				},
			},
		},
		Handler: func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			if i.Member == nil {
				SendError(s, i, "This command can only be used in a server.")
				return
			}

			// Required permission: Manage Channels (or similar admin perm)
			hasPerm := false
			if i.Member.Permissions&discordgo.PermissionManageChannels != 0 {
				hasPerm = true
			}
			if !hasPerm {
				SendError(s, i, "You need the Manage Channels permission to use this command.")
				return
			}

			options := i.ApplicationCommandData().Options
			if len(options) == 0 {
				return
			}

			subcommand := options[0]
			switch subcommand.Name {
			case "create":
				handleEmbedCreate(s, i, database, subcommand.Options)
			case "view":
				handleEmbedView(s, i, database, subcommand.Options)
			case "delete":
				handleEmbedDelete(s, i, database, subcommand.Options)
			case "list":
				handleEmbedList(s, i, database)
			}
		},
	}
}

func handleEmbedCreate(s *discordgo.Session, i *discordgo.InteractionCreate, database *db.DB, options []*discordgo.ApplicationCommandInteractionDataOption) {
	var name string
	var title *string
	var description *string
	var color *int

	for _, opt := range options {
		switch opt.Name {
		case "name":
			name = opt.StringValue()
		case "title":
			t := opt.StringValue()
			title = &t
		case "description":
			d := opt.StringValue()
			description = &d
		case "color":
			cStr := strings.TrimPrefix(opt.StringValue(), "#")
			cInt, err := strconv.ParseInt(cStr, 16, 64)
			if err == nil {
				c := int(cInt)
				color = &c
			} else {
				SendError(s, i, "Invalid color format. Please use a valid hex color code (e.g. #FF0000).")
				return
			}
		}
	}

	if title == nil && description == nil {
		SendError(s, i, "You must provide either a title or a description for the embed.")
		return
	}

	err := database.SaveCustomEmbed(context.Background(), i.GuildID, name, title, description, color)
	if err != nil {
		SendError(s, i, "Failed to save the custom embed.")
		return
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("Successfully created/updated the custom embed `%s`.", name),
		},
	})
}

func handleEmbedView(s *discordgo.Session, i *discordgo.InteractionCreate, database *db.DB, options []*discordgo.ApplicationCommandInteractionDataOption) {
	name := options[0].StringValue()

	embed, err := database.GetCustomEmbed(context.Background(), i.GuildID, name)
	if err != nil {
		SendError(s, i, "An error occurred while fetching the embed.")
		return
	}
	if embed == nil {
			SendError(s, i, fmt.Sprintf("No custom embed found with the name `%s`.", name))
		return
	}

	discordEmbed := &discordgo.MessageEmbed{}
	if embed.Title != nil {
		discordEmbed.Title = *embed.Title
	}
	if embed.Description != nil {
		discordEmbed.Description = *embed.Description
	}
	if embed.Color != nil {
		discordEmbed.Color = *embed.Color
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{discordEmbed},
		},
	})
}

func handleEmbedDelete(s *discordgo.Session, i *discordgo.InteractionCreate, database *db.DB, options []*discordgo.ApplicationCommandInteractionDataOption) {
	name := options[0].StringValue()

	err := database.DeleteCustomEmbed(context.Background(), i.GuildID, name)
	if err != nil {
		SendError(s, i, "Failed to delete the custom embed.")
		return
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("Successfully deleted the custom embed `%s`.", name),
		},
	})
}

func handleEmbedList(s *discordgo.Session, i *discordgo.InteractionCreate, database *db.DB) {
	embeds, err := database.ListCustomEmbeds(context.Background(), i.GuildID)
	if err != nil {
		SendError(s, i, "An error occurred while fetching the list of custom embeds.")
		return
	}

	if len(embeds) == 0 {
		SendError(s, i, "There are no custom embeds saved in this server.")
		return
	}

	var description strings.Builder
	for _, embed := range embeds {
		title := "None"
		if embed.Title != nil {
			title = *embed.Title
		}
		description.WriteString(fmt.Sprintf("**%s** (Title: %s)\n", embed.Name, title))
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: description.String(),
		},
	})
}
