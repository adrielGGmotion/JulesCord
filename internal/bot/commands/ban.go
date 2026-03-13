package commands

import (
	"context"
	"fmt"
	"log"

	"github.com/bwmarrin/discordgo"
	"julescord/internal/db"
)

// Ban returns the /ban command definition and handler.
func Ban(database *db.DB) *Command {
	defaultMemberPermissions := int64(discordgo.PermissionBanMembers)
	return &Command{
		Definition: &discordgo.ApplicationCommand{
			Name:                     "ban",
			Description:              "Bans a user.",
			DefaultMemberPermissions: &defaultMemberPermissions,
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionUser,
					Name:        "user",
					Description: "The user to ban",
					Required:    true,
				},
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "reason",
					Description: "The reason for the ban",
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

			// Execute the ban
			err := s.GuildBanCreateWithReason(i.GuildID, targetUser.ID, fmt.Sprintf("Banned by %s: %s", moderator.Username, reason), 0)
			if err != nil {
				log.Printf("Error banning user %s from guild %s: %v", targetUser.ID, i.GuildID, err)
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: "Failed to ban the user. Ensure the bot has higher permissions than the target user.",
					},
				})
				return
			}

			// Upsert user if they don't exist
			if database != nil {
				err = database.UpsertUser(context.Background(), targetUser.ID, targetUser.Username, targetUser.GlobalName, targetUser.AvatarURL(""))
				if err != nil {
					log.Printf("Failed to upsert user %s for ban: %v", targetUser.ID, err)
				}

				// Log Moderation Action
				err = database.LogModAction(context.Background(), i.GuildID, targetUser.ID, moderator.ID, "ban", reason)
				if err != nil {
					log.Printf("Error logging mod action 'ban' for user %s: %v", targetUser.ID, err)
				}
			}

			// Respond with Embed
			embed := &discordgo.MessageEmbed{
				Title:       "User Banned",
				Description: fmt.Sprintf("<@%s> has been banned.", targetUser.ID),
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

			SendModLog(s, database, i.GuildID, embed)
		},
	}
}
