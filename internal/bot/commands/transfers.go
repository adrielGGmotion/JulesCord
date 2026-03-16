package commands

import (
	"context"
	"fmt"
	"julescord/internal/db"
	"log/slog"

	"github.com/bwmarrin/discordgo"
)

// Transfers returns the /transfers command definition and handler.
func Transfers(database *db.DB) *Command {
	return &Command{
		Definition: &discordgo.ApplicationCommand{
			Name:        "transfers",
			Description: "View your recent coin transfer history.",
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
				SendError(s, i, "Database is not connected.")
				return
			}

			var user *discordgo.User
			if i.Member != nil {
				user = i.Member.User
			} else {
				user = i.User
			}

			// Defer response since DB operation might take a moment
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
			})

			ctx := context.Background()

			transfers, err := database.GetTransfers(ctx, i.GuildID, user.ID, 10)
			if err != nil {
				slog.Error("Error fetching transfer history", "error", err)
				s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
					Content: stringPtr("Failed to retrieve transfer history."),
				})
				return
			}

			if len(transfers) == 0 {
				s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
					Content: stringPtr("You have no transfer history in this server."),
				})
				return
			}

			description := ""
			for _, t := range transfers {
				if t.SenderID == user.ID {
					// Sent coins
					description += fmt.Sprintf("🔴 Sent **%d** coins to <@%s> (<t:%d:R>)\n", t.Amount, t.ReceiverID, t.CreatedAt.Unix())
				} else {
					// Received coins
					description += fmt.Sprintf("🟢 Received **%d** coins from <@%s> (<t:%d:R>)\n", t.Amount, t.SenderID, t.CreatedAt.Unix())
				}
			}

			embed := &discordgo.MessageEmbed{
				Title:       "Recent Transfer History",
				Description: description,
				Color:       0x3498db, // Blue
				Author: &discordgo.MessageEmbedAuthor{
					Name:    user.Username,
					IconURL: user.AvatarURL(""),
				},
			}

			s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
				Embeds: &[]*discordgo.MessageEmbed{embed},
			})
		},
	}
}
