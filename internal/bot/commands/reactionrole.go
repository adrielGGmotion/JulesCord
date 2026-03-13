package commands

import (
	"context"
	"fmt"
	"log/slog"

	"julescord/internal/db"

	"github.com/bwmarrin/discordgo"
)

// ReactionRole returns the /reactionrole command definition and handler.
func ReactionRole(database *db.DB) *Command {
	defaultMemberPermissions := int64(discordgo.PermissionAdministrator)
	return &Command{
		Definition: &discordgo.ApplicationCommand{
			Name:                     "reactionrole",
			Description:              "Manage reaction roles.",
			DefaultMemberPermissions: &defaultMemberPermissions,
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "add",
					Description: "Adds a reaction role to a message.",
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionString,
							Name:        "message_id",
							Description: "The ID of the message to add the reaction to.",
							Required:    true,
						},
						{
							Type:        discordgo.ApplicationCommandOptionString,
							Name:        "emoji",
							Description: "The emoji to react with.",
							Required:    true,
						},
						{
							Type:        discordgo.ApplicationCommandOptionRole,
							Name:        "role",
							Description: "The role to assign.",
							Required:    true,
						},
					},
				},
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "remove",
					Description: "Removes a reaction role from a message.",
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionString,
							Name:        "message_id",
							Description: "The ID of the message.",
							Required:    true,
						},
						{
							Type:        discordgo.ApplicationCommandOptionString,
							Name:        "emoji",
							Description: "The emoji.",
							Required:    true,
						},
					},
				},
			},
		},
		Handler: func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			if i.GuildID == "" {
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: "This command can only be used in a server.",
					},
				})
				return
			}

			if database == nil {
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: "Database is not connected.",
					},
				})
				return
			}

			options := i.ApplicationCommandData().Options
			if len(options) == 0 {
				return
			}

			subCommand := options[0]

			switch subCommand.Name {
			case "add":
				var messageID, emoji string
				var role *discordgo.Role

				for _, option := range subCommand.Options {
					switch option.Name {
					case "message_id":
						messageID = option.StringValue()
					case "emoji":
						emoji = option.StringValue()
					case "role":
						role = option.RoleValue(s, i.GuildID)
					}
				}

				if role == nil {
					s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
						Type: discordgo.InteractionResponseChannelMessageWithSource,
						Data: &discordgo.InteractionResponseData{
							Content: "Role not found.",
						},
					})
					return
				}

				err := database.AddReactionRole(context.Background(), messageID, emoji, role.ID)
				if err != nil {
					slog.Error("Failed to add reaction role", "error", err)
					s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
						Type: discordgo.InteractionResponseChannelMessageWithSource,
						Data: &discordgo.InteractionResponseData{
							Content: "An error occurred while saving the reaction role.",
						},
					})
					return
				}

				// Add the reaction to the message so users can easily click it
				err = s.MessageReactionAdd(i.ChannelID, messageID, emoji)
				if err != nil {
					slog.Error("Failed to add initial reaction %s to message %s", "arg1", emoji, "arg2", messageID, "error", err)
					// We continue even if we fail to add the reaction, as it might be an external emoji the bot can't use
				}

				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: fmt.Sprintf("Reaction role added! Users reacting with %s to message %s will receive <@&%s>.", emoji, messageID, role.ID),
					},
				})

			case "remove":
				var messageID, emoji string

				for _, option := range subCommand.Options {
					switch option.Name {
					case "message_id":
						messageID = option.StringValue()
					case "emoji":
						emoji = option.StringValue()
					}
				}

				err := database.RemoveReactionRole(context.Background(), messageID, emoji)
				if err != nil {
					slog.Error("Failed to remove reaction role", "error", err)
					s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
						Type: discordgo.InteractionResponseChannelMessageWithSource,
						Data: &discordgo.InteractionResponseData{
							Content: "An error occurred while removing the reaction role.",
						},
					})
					return
				}

				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: fmt.Sprintf("Reaction role removed for emoji %s on message %s.", emoji, messageID),
					},
				})
			}
		},
	}
}
