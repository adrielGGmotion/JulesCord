package commands

import (
	"context"
	"fmt"
	"log/slog"

	"julescord/internal/db"

	"github.com/bwmarrin/discordgo"
)

// NickTemplate returns the /nicktemplate command definition and handler.
func NickTemplate(database *db.DB) *Command {
	defaultMemberPermissions := int64(discordgo.PermissionManageNicknames)

	return &Command{
		Definition: &discordgo.ApplicationCommand{
			Name:                     "nicktemplate",
			Description:              "Manage the nickname template for new members.",
			DefaultMemberPermissions: &defaultMemberPermissions,
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        "set",
					Description: "Set the nickname template (e.g., '[Member] {user}').",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionString,
							Name:        "template",
							Description: "The nickname template. Use {user} for the user's name.",
							Required:    true,
						},
					},
				},
				{
					Name:        "view",
					Description: "View the current nickname template.",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
				},
			},
		},
		Handler: func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			if i.GuildID == "" {
				SendError(s, i, "This command can only be used in a server.")
				return
			}

			options := i.ApplicationCommandData().Options
			if len(options) == 0 {
				return
			}

			subCommand := options[0]

			switch subCommand.Name {
			case "set":
				handleNickTemplateSet(s, i, database, subCommand.Options)
			case "view":
				handleNickTemplateView(s, i, database)
			}
		},
	}
}

func handleNickTemplateSet(s *discordgo.Session, i *discordgo.InteractionCreate, database *db.DB, options []*discordgo.ApplicationCommandInteractionDataOption) {
	if database == nil {
		SendError(s, i, "Database connection not available.")
		return
	}

	var template string
	for _, option := range options {
		if option.Name == "template" {
			template = option.StringValue()
		}
	}

	if template == "" {
		SendError(s, i, "Invalid template provided.")
		return
	}

	err := database.SetNicknameTemplate(context.Background(), i.GuildID, template)
	if err != nil {
		slog.Error("Failed to set nickname template", "guild_id", i.GuildID, "error", err)
		SendError(s, i, "Failed to set nickname template. Please try again later.")
		return
	}

	embed := &discordgo.MessageEmbed{
		Title:       "Nickname Template Configured",
		Description: fmt.Sprintf("Successfully set the nickname template to:\n`%s`\nNew members will automatically receive this nickname upon joining.", template),
		Color:       0x00FF00, // Green
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
		},
	})
}

func handleNickTemplateView(s *discordgo.Session, i *discordgo.InteractionCreate, database *db.DB) {
	if database == nil {
		SendError(s, i, "Database connection not available.")
		return
	}

	template, err := database.GetNicknameTemplate(context.Background(), i.GuildID)
	if err != nil {
		slog.Error("Failed to fetch nickname template", "guild_id", i.GuildID, "error", err)
		SendError(s, i, "Failed to fetch nickname template. Please try again later.")
		return
	}

	if template == nil || *template == "" {
		embed := &discordgo.MessageEmbed{
			Title:       "Nickname Template",
			Description: "No nickname template is currently configured for this server.",
			Color:       0x3498db, // Blue
		}
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Embeds: []*discordgo.MessageEmbed{embed},
			},
		})
		return
	}

	embed := &discordgo.MessageEmbed{
		Title:       "Nickname Template",
		Description: fmt.Sprintf("The current nickname template is:\n`%s`", *template),
		Color:       0x3498db, // Blue
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
		},
	})
}
