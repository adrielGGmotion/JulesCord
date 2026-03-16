package commands

import (
	"context"
	"fmt"
	"julescord/internal/db"
	"log/slog"
	"time"

	"github.com/bwmarrin/discordgo"
)

// BookmarkContext returns the command for bookmarking messages via right-click
func BookmarkContext(database *db.DB) *Command {
	name := "Bookmark"
	return &Command{
		Definition: &discordgo.ApplicationCommand{
			Name: name,
			Type: discordgo.MessageApplicationCommand,
		},
		Handler: func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			if i.Member == nil || i.Member.User == nil {
				SendError(s, i, "This command can only be used in a server.")
				return
			}

			// Defer response to avoid timeouts
			err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Flags: discordgo.MessageFlagsEphemeral,
				},
			})
			if err != nil {
				slog.Error("Failed to defer interaction", "err", err)
				return
			}

			data := i.ApplicationCommandData()
			messageMap := data.Resolved.Messages
			if len(messageMap) == 0 {
				_, _ = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
					Content: stringPtr("Failed to resolve message."),
				})
				return
			}

			var targetMessage *discordgo.Message
			for _, msg := range messageMap {
				targetMessage = msg
				break
			}

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			err = database.AddBookmark(ctx, i.Member.User.ID, targetMessage.ID, i.ChannelID, i.GuildID, nil)
			if err != nil {
				if err.Error() == "bookmark already exists" {
					_, _ = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
						Content: stringPtr("You have already bookmarked this message!"),
					})
					return
				}
				slog.Error("Failed to save bookmark", "err", err)
				_, _ = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
					Content: stringPtr("An error occurred while saving the bookmark."),
				})
				return
			}

			_, _ = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
				Content: stringPtr("Message bookmarked successfully! View it using `/bookmarks`."),
			})
		},
	}
}

// BookmarksSlash returns the slash command to view and manage bookmarks
func BookmarksSlash(database *db.DB) *Command {
	return &Command{
		Definition: &discordgo.ApplicationCommand{
			Name:        "bookmarks",
			Description: "View and manage your saved bookmarks",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        "list",
					Description: "List your bookmarked messages",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
				},
				{
					Name:        "remove",
					Description: "Remove a bookmarked message",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Name:        "id",
							Description: "The ID of the message you want to un-bookmark",
							Type:        discordgo.ApplicationCommandOptionString,
							Required:    true,
						},
					},
				},
			},
		},
		Handler: func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			if i.Member == nil || i.Member.User == nil {
				SendError(s, i, "This command can only be used in a server.")
				return
			}

			// Defer response to avoid timeouts
			err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Flags: discordgo.MessageFlagsEphemeral, // Keeps it private
				},
			})
			if err != nil {
				slog.Error("Failed to defer interaction", "err", err)
				return
			}

			data := i.ApplicationCommandData()
			subcommand := data.Options[0].Name

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			if subcommand == "list" {
				bookmarks, err := database.GetBookmarks(ctx, i.Member.User.ID)
				if err != nil {
					slog.Error("Failed to fetch bookmarks", "err", err)
					_, _ = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
						Content: stringPtr("An error occurred while fetching your bookmarks."),
					})
					return
				}

				if len(bookmarks) == 0 {
					_, _ = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
						Content: stringPtr("You have no bookmarked messages yet. Right click a message -> Apps -> Bookmark to save one!"),
					})
					return
				}

				var description string
				for _, b := range bookmarks {
					url := fmt.Sprintf("https://discord.com/channels/%s/%s/%s", b.GuildID, b.ChannelID, b.MessageID)
					description += fmt.Sprintf("[Message Link](%s) - Bookmarked %s\n", url, b.CreatedAt.Format("Jan 02, 2006"))
					description += fmt.Sprintf("`ID: %s`\n\n", b.MessageID)
				}

				if len(description) > 4096 {
					description = description[:4093] + "..."
				}

				embed := &discordgo.MessageEmbed{
					Title:       "Your Bookmarks",
					Description: description,
					Color:       0x00ff00,
				}

				_, _ = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
					Embeds: &[]*discordgo.MessageEmbed{embed},
				})
			} else if subcommand == "remove" {
				msgID := data.Options[0].Options[0].StringValue()

				err := database.RemoveBookmark(ctx, i.Member.User.ID, msgID)
				if err != nil {
					if err.Error() == "bookmark not found" {
						_, _ = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
							Content: stringPtr("Bookmark not found. Make sure you provided the correct message ID."),
						})
						return
					}
					slog.Error("Failed to remove bookmark", "err", err)
					_, _ = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
						Content: stringPtr("An error occurred while removing the bookmark."),
					})
					return
				}

				_, _ = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
					Content: stringPtr(fmt.Sprintf("Successfully removed bookmark for message ID `%s`.", msgID)),
				})
			}
		},
	}
}
