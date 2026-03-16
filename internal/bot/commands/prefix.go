package commands

import (
	"context"
	"fmt"
	"julescord/internal/db"

	"github.com/bwmarrin/discordgo"
)

// Prefix returns the /prefix command.
func Prefix(database *db.DB) *Command {
	return &Command{
		Definition: &discordgo.ApplicationCommand{
			Name:        "prefix",
			Description: "View or set the custom text prefix for this server.",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        "set",
					Description: "Set a new custom prefix.",
					Type:        discordgo.ApplicationCommandOptionString,
					Required:    false,
				},
			},
			DefaultMemberPermissions: func() *int64 { p := int64(discordgo.PermissionManageServer); return &p }(),
		},
		Handler: func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			if database == nil {
				SendError(s, i, "Database connection not available.")
				return
			}
			if i.GuildID == "" {
				SendError(s, i, "This command can only be used in a server.")
				return
			}

			// Defer response
			err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
			})
			if err != nil {
				return
			}

			// Get current prefix if no options are provided
			options := i.ApplicationCommandData().Options
			if len(options) == 0 {
				prefix, err := database.GetGuildPrefix(context.Background(), i.GuildID)
				if err != nil {
					_, _ = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
						Content: stringPtr("Failed to retrieve the current prefix."),
					})
					return
				}
				_, _ = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
					Content: stringPtr(fmt.Sprintf("The current custom prefix for this server is: `%s`", prefix)),
				})
				return
			}

			// Set new prefix
			newPrefix := options[0].StringValue()

			// Check prefix length limits
			if len(newPrefix) > 5 {
				_, _ = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
					Content: stringPtr("Prefix cannot be longer than 5 characters."),
				})
				return
			}

			err = database.SetGuildPrefix(context.Background(), i.GuildID, newPrefix)
			if err != nil {
				_, _ = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
					Content: stringPtr("Failed to update the prefix. Please try again later."),
				})
				return
			}

			_, _ = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
				Content: stringPtr(fmt.Sprintf("✅ The custom prefix has been successfully updated to: `%s`", newPrefix)),
			})
		},
	}
}
