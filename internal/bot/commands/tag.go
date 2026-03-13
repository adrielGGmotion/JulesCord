package commands

import (
	"context"
	"fmt"
	"strings"

	"julescord/internal/db"

	"github.com/bwmarrin/discordgo"
)

// Tag returns the application command for the tag system.
func Tag(database *db.DB) *Command {
	return &Command{
		Definition: &discordgo.ApplicationCommand{
			Name:        "tag",
			Description: "Manage and use custom text tags",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        "create",
					Description: "Create a new tag",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Name:        "name",
							Description: "The name of the tag (no spaces)",
							Type:        discordgo.ApplicationCommandOptionString,
							Required:    true,
						},
						{
							Name:        "content",
							Description: "The content of the tag",
							Type:        discordgo.ApplicationCommandOptionString,
							Required:    true,
						},
					},
				},
				{
					Name:        "list",
					Description: "List all tags in the server",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
				},
				{
					Name:        "delete",
					Description: "Delete a tag you created",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Name:        "name",
							Description: "The name of the tag to delete",
							Type:        discordgo.ApplicationCommandOptionString,
							Required:    true,
						},
					},
				},
				{
					Name:        "view",
					Description: "View a tag's content",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Name:        "name",
							Description: "The name of the tag to view",
							Type:        discordgo.ApplicationCommandOptionString,
							Required:    true,
						},
					},
				},
			},
		},
		Handler: func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			if i.Type != discordgo.InteractionApplicationCommand {
				return
			}

			guildID := i.GuildID
			if guildID == "" {
				SendError(s, i, "This command can only be used in a server.")
				return
			}

			options := i.ApplicationCommandData().Options
			if len(options) == 0 {
				SendError(s, i, "Please specify a subcommand.")
				return
			}

			subcommand := options[0]

			switch subcommand.Name {
			case "create":
				handleTagCreate(s, i, database, guildID, subcommand.Options)
			case "list":
				handleTagList(s, i, database, guildID)
			case "delete":
				handleTagDelete(s, i, database, guildID, subcommand.Options)
			case "view":
				handleTagView(s, i, database, guildID, subcommand.Options)
			default:
				SendError(s, i, "Unknown subcommand.")
			}
		},
	}
}

func handleTagCreate(s *discordgo.Session, i *discordgo.InteractionCreate, database *db.DB, guildID string, options []*discordgo.ApplicationCommandInteractionDataOption) {
	var name, content string
	for _, opt := range options {
		if opt.Name == "name" {
			name = strings.ToLower(opt.StringValue())
		} else if opt.Name == "content" {
			content = opt.StringValue()
		}
	}

	if strings.Contains(name, " ") {
		SendError(s, i, "Tag names cannot contain spaces.")
		return
	}

	var authorID string
	if i.Member != nil {
		authorID = i.Member.User.ID
	} else if i.User != nil {
		authorID = i.User.ID
	}

	err := database.CreateTag(context.Background(), guildID, name, content, authorID)
	if err != nil {
		if strings.Contains(err.Error(), "unique constraint") || strings.Contains(err.Error(), "duplicate key") {
			SendError(s, i, fmt.Sprintf("A tag named `%s` already exists in this server.", name))
		} else {
			SendError(s, i, "Failed to create tag.")
		}
		return
	}

	SendEmbed(s, i, &discordgo.MessageEmbed{
		Title:       "Tag Created",
		Description: fmt.Sprintf("Successfully created tag `%s`.", name),
		Color:       0x00FF00, // Green
	})
}

func handleTagDelete(s *discordgo.Session, i *discordgo.InteractionCreate, database *db.DB, guildID string, options []*discordgo.ApplicationCommandInteractionDataOption) {
	name := strings.ToLower(options[0].StringValue())

	var userID string
	if i.Member != nil {
		userID = i.Member.User.ID
	} else if i.User != nil {
		userID = i.User.ID
	}

	// Fetch tag to verify authorship or permissions
	tag, err := database.GetTag(context.Background(), guildID, name)
	if err != nil {
		SendError(s, i, "Failed to retrieve tag.")
		return
	}
	if tag == nil {
		SendError(s, i, fmt.Sprintf("No tag named `%s` found.", name))
		return
	}

	// Check permissions (author or admin)
	hasPermission := false
	if tag.AuthorID == userID {
		hasPermission = true
	} else if i.Member != nil && (i.Member.Permissions&discordgo.PermissionAdministrator) != 0 {
		hasPermission = true
	}

	if !hasPermission {
		SendError(s, i, "You do not have permission to delete this tag. You must be the author or an administrator.")
		return
	}

	err = database.DeleteTag(context.Background(), guildID, name)
	if err != nil {
		SendError(s, i, "Failed to delete tag.")
		return
	}

	SendEmbed(s, i, &discordgo.MessageEmbed{
		Title:       "Tag Deleted",
		Description: fmt.Sprintf("Successfully deleted tag `%s`.", name),
		Color:       0x00FF00, // Green
	})
}

func handleTagList(s *discordgo.Session, i *discordgo.InteractionCreate, database *db.DB, guildID string) {
	tags, err := database.ListTags(context.Background(), guildID)
	if err != nil {
		SendError(s, i, "Failed to retrieve tags.")
		return
	}

	if len(tags) == 0 {
		SendEmbed(s, i, &discordgo.MessageEmbed{
			Title:       "Server Tags",
			Description: "There are no tags created in this server yet. Use `/tag create` to add one.",
			Color:       0x0099FF,
		})
		return
	}

	var descriptionBuilder strings.Builder
	for _, tag := range tags {
		descriptionBuilder.WriteString(fmt.Sprintf("- `%s` (by <@%s>)\n", tag.Name, tag.AuthorID))
	}

	SendEmbed(s, i, &discordgo.MessageEmbed{
		Title:       "Server Tags",
		Description: descriptionBuilder.String(),
		Color:       0x0099FF,
	})
}

func handleTagView(s *discordgo.Session, i *discordgo.InteractionCreate, database *db.DB, guildID string, options []*discordgo.ApplicationCommandInteractionDataOption) {
	name := strings.ToLower(options[0].StringValue())

	tag, err := database.GetTag(context.Background(), guildID, name)
	if err != nil {
		SendError(s, i, "Failed to retrieve tag.")
		return
	}

	if tag == nil {
		SendError(s, i, fmt.Sprintf("Tag `%s` not found.", name))
		return
	}

	// For tags, it's common to just send the raw text content so people can use it
	// rather than wrapping it in an embed, but we can do a simple message.
	err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: tag.Content,
			AllowedMentions: &discordgo.MessageAllowedMentions{
				Parse: []discordgo.AllowedMentionType{}, // No mentions parsed
			},
		},
	})
	if err != nil {
		SendError(s, i, "Failed to send tag content.")
	}
}
