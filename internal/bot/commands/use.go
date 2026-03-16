package commands

import (
	"context"
	"fmt"
	"log/slog"

	"julescord/internal/db"

	"github.com/bwmarrin/discordgo"
)

// Use creates the /use command to consume items from the inventory.
func Use(database *db.DB) *Command {
	return &Command{
		Definition: &discordgo.ApplicationCommand{
			Name:        "use",
			Description: "Use an item from your inventory",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "item",
					Description: "The exact name of the item you want to use",
					Required:    true,
				},
			},
		},
		Handler: func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			if database == nil {
				SendError(s, i, "Database connection not available")
				return
			}

			var itemName string
			options := i.ApplicationCommandData().Options
			for _, opt := range options {
				if opt.Name == "item" {
					itemName = opt.StringValue()
				}
			}

			var userID string
			if i.Member != nil && i.Member.User != nil {
				userID = i.Member.User.ID
			} else if i.User != nil {
				userID = i.User.ID
			}

			ctx := context.Background()

			// Look up item by name to get its ID
			shopItem, err := database.GetShopItem(ctx, i.GuildID, itemName)
			if err != nil {
				slog.Error("Failed to fetch shop item for /use", "error", err)
				SendError(s, i, "Failed to verify item existence.")
				return
			}
			if shopItem == nil {
				SendError(s, i, "Item not found in the server shop.")
				return
			}

			// Try to remove 1 quantity of the item from the user's inventory
			err = database.RemoveUserItem(ctx, i.GuildID, userID, shopItem.ID, 1)
			if err != nil {
				slog.Error("Failed to consume user item", "error", err)
				SendError(s, i, "You do not own this item or do not have enough quantity.")
				return
			}

			// Predefined effects logic could go here based on `shopItem.Name`.
			// For this phase, we just send a generic success message to confirm consumption.
			effectMessage := fmt.Sprintf("You used **%s**! It had a mysterious effect...", shopItem.Name)

			SendEmbed(s, i, &discordgo.MessageEmbed{
				Title:       "Item Used",
				Description: effectMessage,
				Color:       0x00FF00, // Green
			})
		},
	}
}
