package commands

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"julescord/internal/db"

	"github.com/bwmarrin/discordgo"
)

// ReactionMenu returns the /reactionmenu command definition and handler.
func ReactionMenu(database *db.DB) *Command {
	defaultMemberPermissions := int64(discordgo.PermissionManageRoles)
	return &Command{
		Definition: &discordgo.ApplicationCommand{
			Name:                     "reactionmenu",
			Description:              "Manage reaction role menus.",
			DefaultMemberPermissions: &defaultMemberPermissions,
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "create",
					Description: "Create a new reaction role menu message",
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionString,
							Name:        "title",
							Description: "Title of the embed",
							Required:    true,
						},
						{
							Type:        discordgo.ApplicationCommandOptionString,
							Name:        "description",
							Description: "Description of the embed",
							Required:    true,
						},
					},
				},
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "add-role",
					Description: "Add a role to an existing reaction menu",
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionString,
							Name:        "message_id",
							Description: "The ID of the menu message",
							Required:    true,
						},
						{
							Type:        discordgo.ApplicationCommandOptionString,
							Name:        "emoji",
							Description: "The emoji to react with",
							Required:    true,
						},
						{
							Type:        discordgo.ApplicationCommandOptionRole,
							Name:        "role",
							Description: "The role to give when reacting",
							Required:    true,
						},
					},
				},
			},
		},
		Handler: func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			if database == nil {
				SendError(s, i, "Database not available")
				return
			}

			options := i.ApplicationCommandData().Options
			subCommand := options[0].Name

			switch subCommand {
			case "create":
				handleReactionMenuCreate(s, i, database, options[0].Options)
			case "add-role":
				handleReactionMenuAddRole(s, i, database, options[0].Options)
			}
		},
	}
}

func handleReactionMenuCreate(s *discordgo.Session, i *discordgo.InteractionCreate, database *db.DB, options []*discordgo.ApplicationCommandInteractionDataOption) {
	title := ""
	description := ""

	for _, opt := range options {
		switch opt.Name {
		case "title":
			title = opt.StringValue()
		case "description":
			description = strings.ReplaceAll(opt.StringValue(), "\\n", "\n")
		}
	}

	embed := &discordgo.MessageEmbed{
		Title:       title,
		Description: description,
		Color:       0x3498db,
	}

	msg, err := s.ChannelMessageSendEmbed(i.ChannelID, embed)
	if err != nil {
		slog.Error("Failed to send reaction menu message", "error", err)
		SendError(s, i, "Failed to create the menu message.")
		return
	}

	err = database.CreateReactionMenu(context.Background(), msg.ID, i.GuildID, i.ChannelID)
	if err != nil {
		SendError(s, i, "Failed to save the menu to the database.")
		return
	}

	err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("Reaction menu created successfully!\nMessage ID: `%s`\nUse `/reactionmenu add-role` to add roles to this menu.", msg.ID),
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
	if err != nil {
		slog.Error("Failed to respond to reactionmenu create", "error", err)
	}
}

func handleReactionMenuAddRole(s *discordgo.Session, i *discordgo.InteractionCreate, database *db.DB, options []*discordgo.ApplicationCommandInteractionDataOption) {
	messageID := ""
	emoji := ""
	roleID := ""

	for _, opt := range options {
		switch opt.Name {
		case "message_id":
			messageID = opt.StringValue()
		case "emoji":
			emoji = opt.StringValue()
		case "role":
			roleID = opt.RoleValue(s, i.GuildID).ID
		}
	}

	if strings.HasPrefix(emoji, "<:") && strings.HasSuffix(emoji, ">") {
		emoji = emoji[2 : len(emoji)-1]
	} else if strings.HasPrefix(emoji, "<a:") && strings.HasSuffix(emoji, ">") {
		emoji = emoji[3 : len(emoji)-1]
	}

	dbEmojiName := emoji
	parts := strings.Split(emoji, ":")
	if len(parts) > 0 {
		dbEmojiName = parts[0]
	}

	menu, err := database.GetReactionMenu(context.Background(), messageID)
	if err != nil || menu == nil {
		SendError(s, i, "Failed to find the reaction menu in the database. Ensure the message ID is correct.")
		return
	}

	err = database.AddReactionMenuItem(context.Background(), messageID, dbEmojiName, roleID)
	if err != nil {
		SendError(s, i, "Failed to save the reaction menu item to the database.")
		return
	}

	err = s.MessageReactionAdd(menu.ChannelID, messageID, emoji)
	if err != nil {
		slog.Error("Failed to add reaction to menu message", "error", err)
		SendError(s, i, "Failed to add the reaction to the message. Make sure the message ID is correct and the bot has permissions.")
		return
	}

	err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("Successfully added role <@&%s> for emoji %s on message `%s`.", roleID, emoji, messageID),
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
	if err != nil {
		slog.Error("Failed to respond to reactionmenu add-role", "error", err)
	}
}
