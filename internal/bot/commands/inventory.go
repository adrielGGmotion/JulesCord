package commands

import (
	"context"
	"fmt"
	"log/slog"

	"julescord/internal/db"

	"github.com/bwmarrin/discordgo"
)

// Inventory creates the /inventory command to view purchased items.
func Inventory(database *db.DB) *Command {
	return &Command{
		Definition: &discordgo.ApplicationCommand{
			Name:        "inventory",
			Description: "View the items you have purchased from the shop",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionUser,
					Name:        "user",
					Description: "The user whose inventory you want to view (defaults to yourself)",
					Required:    false,
				},
			},
		},
		Handler: func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			if database == nil {
				SendError(s, i, "Database connection not available")
				return
			}

			var targetUser *discordgo.User
			if i.Member != nil && i.Member.User != nil {
				targetUser = i.Member.User
			} else if i.User != nil {
				targetUser = i.User
			}

			options := i.ApplicationCommandData().Options
			for _, opt := range options {
				if opt.Name == "user" {
					targetUser = opt.UserValue(s)
				}
			}

			if targetUser == nil {
				SendError(s, i, "Could not determine user.")
				return
			}

			ctx := context.Background()
			items, err := database.GetUserInventory(ctx, i.GuildID, targetUser.ID)
			if err != nil {
				slog.Error("Failed to fetch user inventory", "error", err)
				SendError(s, i, "Failed to fetch inventory.")
				return
			}

			if len(items) == 0 {
				SendEmbed(s, i, &discordgo.MessageEmbed{
					Title:       fmt.Sprintf("%s's Inventory", targetUser.Username),
					Description: "This inventory is empty.",
					Color:       0x5865F2,
					Thumbnail: &discordgo.MessageEmbedThumbnail{
						URL: targetUser.AvatarURL(""),
					},
				})
				return
			}

			embed := &discordgo.MessageEmbed{
				Title:       fmt.Sprintf("%s's Inventory", targetUser.Username),
				Description: fmt.Sprintf("Items owned: **%d**", len(items)),
				Color:       0x5865F2,
				Thumbnail: &discordgo.MessageEmbedThumbnail{
					URL: targetUser.AvatarURL(""),
				},
			}

			// Format list of items
			var inventoryList string
			for idx, item := range items {
				inventoryList += fmt.Sprintf("• **%s** (Acquired: <t:%d:D>)\n", item.ItemName, item.AcquiredAt.Unix())
				if idx >= 20 {
					inventoryList += "*...and more*\n"
					break
				}
			}

			embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
				Name:  "Items",
				Value: inventoryList,
			})

			SendEmbed(s, i, embed)
		},
	}
}
