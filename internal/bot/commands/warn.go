package commands

import (
	"context"
	"fmt"
	"log"

	"github.com/bwmarrin/discordgo"
	"julescord/internal/db"
)

// Warn returns the /warn command definition and handler.
func Warn(database *db.DB) *Command {
	defaultMemberPermissions := int64(discordgo.PermissionKickMembers | discordgo.PermissionManageMessages)
	return &Command{
		Definition: &discordgo.ApplicationCommand{
			Name:                     "warn",
			Description:              "Warns a user.",
			DefaultMemberPermissions: &defaultMemberPermissions,
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionUser,
					Name:        "user",
					Description: "The user to warn",
					Required:    true,
				},
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "reason",
					Description: "The reason for the warning",
					Required:    true,
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
						Content: "Database is not connected. Cannot issue warnings.",
					},
				})
				return
			}

			// Get options
			var targetUser *discordgo.User
			var reason string
			for _, option := range i.ApplicationCommandData().Options {
				switch option.Name {
				case "user":
					targetUser = option.UserValue(s)
				case "reason":
					reason = option.StringValue()
				}
			}

			if targetUser == nil {
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: "Could not find the specified user.",
					},
				})
				return
			}

			// Upsert user if they don't exist yet (in case they haven't sent a message)
			err := database.UpsertUser(context.Background(), targetUser.ID, targetUser.Username, targetUser.GlobalName, targetUser.AvatarURL(""))
			if err != nil {
				log.Printf("Failed to upsert user %s for warning: %v", targetUser.ID, err)
			}

			moderator := i.Member.User

			// Add Warning
			err = database.AddWarning(context.Background(), i.GuildID, targetUser.ID, moderator.ID, reason)
			if err != nil {
				log.Printf("Error adding warning for user %s: %v", targetUser.ID, err)
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: "An error occurred while trying to warn the user.",
					},
				})
				return
			}

			// Log Moderation Action
			err = database.LogModAction(context.Background(), i.GuildID, targetUser.ID, moderator.ID, "warn", reason)
			if err != nil {
				log.Printf("Error logging mod action 'warn' for user %s: %v", targetUser.ID, err)
			}

			// Respond with Embed
			embed := &discordgo.MessageEmbed{
				Title:       "User Warned",
				Description: fmt.Sprintf("<@%s> has been warned.", targetUser.ID),
				Color:       0xFFA500, // Orange
				Fields: []*discordgo.MessageEmbedField{
					{
						Name:   "Reason",
						Value:  reason,
						Inline: false,
					},
					{
						Name:   "Moderator",
						Value:  fmt.Sprintf("<@%s>", moderator.ID),
						Inline: true,
					},
				},
			}

			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Embeds: []*discordgo.MessageEmbed{embed},
				},
			})
		},
	}
}
