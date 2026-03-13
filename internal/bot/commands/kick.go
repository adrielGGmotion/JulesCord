package commands

import (
	"context"
	"fmt"
	"log"

	"github.com/bwmarrin/discordgo"
	"julescord/internal/db"
)

// Kick returns the /kick command definition and handler.
func Kick(database *db.DB) *Command {
	defaultMemberPermissions := int64(discordgo.PermissionKickMembers)
	return &Command{
		Definition: &discordgo.ApplicationCommand{
			Name:                     "kick",
			Description:              "Kicks a user.",
			DefaultMemberPermissions: &defaultMemberPermissions,
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionUser,
					Name:        "user",
					Description: "The user to kick",
					Required:    true,
				},
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "reason",
					Description: "The reason for the kick",
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

			moderator := i.Member.User

			// Execute the kick
			err := s.GuildMemberDeleteWithReason(i.GuildID, targetUser.ID, fmt.Sprintf("Kicked by %s: %s", moderator.Username, reason))
			if err != nil {
				log.Printf("Error kicking user %s from guild %s: %v", targetUser.ID, i.GuildID, err)
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: "Failed to kick the user. Ensure the bot has higher permissions than the target user.",
					},
				})
				return
			}

			// Upsert user if they don't exist
			if database != nil {
				err = database.UpsertUser(context.Background(), targetUser.ID, targetUser.Username, targetUser.GlobalName, targetUser.AvatarURL(""))
				if err != nil {
					log.Printf("Failed to upsert user %s for kick: %v", targetUser.ID, err)
				}

				// Log Moderation Action
				err = database.LogModAction(context.Background(), i.GuildID, targetUser.ID, moderator.ID, "kick", reason)
				if err != nil {
					log.Printf("Error logging mod action 'kick' for user %s: %v", targetUser.ID, err)
				}
			}

			// Respond with Embed
			embed := &discordgo.MessageEmbed{
				Title:       "User Kicked",
				Description: fmt.Sprintf("<@%s> has been kicked.", targetUser.ID),
				Color:       0xFF0000, // Red
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
