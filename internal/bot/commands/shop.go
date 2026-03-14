package commands

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"

	"julescord/internal/db"

	"github.com/bwmarrin/discordgo"
)

// Shop creates the /shop command.
func Shop(database *db.DB) *Command {
	return &Command{
		Definition: &discordgo.ApplicationCommand{
			Name:        "shop",
			Description: "Manage and interact with the server shop",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        "add",
					Description: "Add a new item to the shop (Admins only)",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionString,
							Name:        "name",
							Description: "The name of the item",
							Required:    true,
						},
						{
							Type:        discordgo.ApplicationCommandOptionInteger,
							Name:        "price",
							Description: "The price in coins",
							Required:    true,
						},
						{
							Type:        discordgo.ApplicationCommandOptionString,
							Name:        "description",
							Description: "A short description of the item",
							Required:    false,
						},
						{
							Type:        discordgo.ApplicationCommandOptionRole,
							Name:        "role",
							Description: "A role to grant upon purchase",
							Required:    false,
						},
					},
				},
				{
					Name:        "remove",
					Description: "Remove an item from the shop (Admins only)",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionString,
							Name:        "name",
							Description: "The exact name of the item to remove",
							Required:    true,
						},
					},
				},
				{
					Name:        "list",
					Description: "List all items available in the shop",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
				},
				{
					Name:        "buy",
					Description: "Buy an item from the shop",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionString,
							Name:        "name",
							Description: "The exact name of the item to buy",
							Required:    true,
						},
					},
				},
			},
		},
		Handler: func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			if database == nil {
				SendError(s, i, "Database is required for this command.")
				return
			}

			options := i.ApplicationCommandData().Options
			if len(options) == 0 {
				return
			}

			subcommand := options[0]

			switch subcommand.Name {
			case "add":
				// Admin check
				if i.Member == nil || (i.Member.Permissions&discordgo.PermissionAdministrator) == 0 {
					SendError(s, i, "You must be an administrator to use this command.")
					return
				}
				handleShopAdd(s, i, database, subcommand.Options)
			case "remove":
				// Admin check
				if i.Member == nil || (i.Member.Permissions&discordgo.PermissionAdministrator) == 0 {
					SendError(s, i, "You must be an administrator to use this command.")
					return
				}
				handleShopRemove(s, i, database, subcommand.Options)
			case "list":
				handleShopList(s, i, database)
			case "buy":
				handleShopBuy(s, i, database, subcommand.Options)
			}
		},
	}
}

func handleShopAdd(s *discordgo.Session, i *discordgo.InteractionCreate, database *db.DB, options []*discordgo.ApplicationCommandInteractionDataOption) {
	var name string
	var price int64
	var description *string
	var roleID *string

	for _, opt := range options {
		switch opt.Name {
		case "name":
			name = opt.StringValue()
		case "price":
			price = opt.IntValue()
		case "description":
			val := opt.StringValue()
			description = &val
		case "role":
			role := opt.RoleValue(s, "")
			if role != nil {
				roleID = &role.ID
			}
		}
	}

	if price <= 0 {
		SendError(s, i, "Price must be greater than 0.")
		return
	}

	err := database.AddShopItem(context.Background(), i.GuildID, name, description, price, roleID)
	if err != nil {
		slog.Error("Failed to add shop item", "error", err)
		SendError(s, i, "Failed to add shop item. Ensure the name is unique.")
		return
	}

	descText := "None"
	if description != nil {
		descText = *description
	}
	roleText := "None"
	if roleID != nil {
		roleText = fmt.Sprintf("<@&%s>", *roleID)
	}

	SendEmbed(s, i, &discordgo.MessageEmbed{
		Title:       "Shop Item Added",
		Description: fmt.Sprintf("Successfully added **%s** to the shop.", name),
		Color:       0x00FF00,
		Fields: []*discordgo.MessageEmbedField{
			{Name: "Price", Value: strconv.FormatInt(price, 10) + " coins", Inline: true},
			{Name: "Role", Value: roleText, Inline: true},
			{Name: "Description", Value: descText, Inline: false},
		},
	})
}

func handleShopRemove(s *discordgo.Session, i *discordgo.InteractionCreate, database *db.DB, options []*discordgo.ApplicationCommandInteractionDataOption) {
	var name string
	for _, opt := range options {
		if opt.Name == "name" {
			name = opt.StringValue()
		}
	}

	err := database.RemoveShopItem(context.Background(), i.GuildID, name)
	if err != nil {
		slog.Error("Failed to remove shop item", "error", err)
		SendError(s, i, "Failed to remove shop item.")
		return
	}

	SendEmbed(s, i, &discordgo.MessageEmbed{
		Title:       "Shop Item Removed",
		Description: fmt.Sprintf("Successfully removed **%s** from the shop.", name),
		Color:       0x00FF00,
	})
}

func handleShopList(s *discordgo.Session, i *discordgo.InteractionCreate, database *db.DB) {
	items, err := database.GetShopItems(context.Background(), i.GuildID)
	if err != nil {
		slog.Error("Failed to get shop items", "error", err)
		SendError(s, i, "Failed to retrieve shop items.")
		return
	}

	if len(items) == 0 {
		SendEmbed(s, i, &discordgo.MessageEmbed{
			Title:       "Server Shop",
			Description: "The shop is currently empty.",
			Color:       0x5865F2,
		})
		return
	}

	embed := &discordgo.MessageEmbed{
		Title: "Server Shop",
		Color: 0x5865F2,
	}

	for _, item := range items {
		val := fmt.Sprintf("Price: **%d coins**\n", item.Price)
		if item.Description != nil {
			val += fmt.Sprintf("Description: *%s*\n", *item.Description)
		}
		if item.RoleID != nil {
			val += fmt.Sprintf("Grants Role: <@&%s>\n", *item.RoleID)
		}
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:  item.Name,
			Value: val,
		})
	}

	SendEmbed(s, i, embed)
}

func handleShopBuy(s *discordgo.Session, i *discordgo.InteractionCreate, database *db.DB, options []*discordgo.ApplicationCommandInteractionDataOption) {
	var name string
	for _, opt := range options {
		if opt.Name == "name" {
			name = opt.StringValue()
		}
	}

	ctx := context.Background()
	item, err := database.GetShopItem(ctx, i.GuildID, name)
	if err != nil {
		slog.Error("Failed to fetch item for purchase", "error", err)
		SendError(s, i, "Failed to find that item in the shop.")
		return
	}

	if item == nil {
		SendError(s, i, "Item not found in the shop.")
		return
	}

	var userID string
	if i.Member != nil && i.Member.User != nil {
		userID = i.Member.User.ID
	} else if i.User != nil {
		userID = i.User.ID
	}

	err = database.BuyItem(ctx, i.GuildID, userID, item.ID, item.Price)
	if err != nil {
		slog.Error("Failed to buy item", "error", err)
		SendError(s, i, "Purchase failed. Do you have enough coins?")
		return
	}

	// Grant role if applicable
	if item.RoleID != nil {
		err = s.GuildMemberRoleAdd(i.GuildID, userID, *item.RoleID)
		if err != nil {
			slog.Error("Failed to grant role after purchase", "error", err)
			SendEmbed(s, i, &discordgo.MessageEmbed{
				Title:       "Purchase Successful (Role Failed)",
				Description: fmt.Sprintf("You bought **%s** for %d coins, but I failed to grant the role. Please contact an admin.", name, item.Price),
				Color:       0xFFA500, // Orange
			})
			return
		}
	}

	SendEmbed(s, i, &discordgo.MessageEmbed{
		Title:       "Purchase Successful!",
		Description: fmt.Sprintf("You have successfully purchased **%s** for %d coins.", name, item.Price),
		Color:       0x00FF00,
	})
}
