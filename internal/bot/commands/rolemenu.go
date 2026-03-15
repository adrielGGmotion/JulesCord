package commands

import (
	"context"
	"fmt"
	"julescord/internal/db"
	"log/slog"

	"github.com/bwmarrin/discordgo"
)

// RoleMenu returns the /rolemenu command definition
func RoleMenu(database *db.DB) *Command {
	return &Command{
		Definition: &discordgo.ApplicationCommand{
			Name:        "rolemenu",
			Description: "Manage role menus",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        "setup",
					Description: "Setup a new role menu message in this channel",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Name:        "title",
							Description: "Title of the role menu embed",
							Type:        discordgo.ApplicationCommandOptionString,
							Required:    true,
						},
						{
							Name:        "description",
							Description: "Description of the role menu embed",
							Type:        discordgo.ApplicationCommandOptionString,
							Required:    false,
						},
					},
				},
				{
					Name:        "add_role",
					Description: "Add a role to an existing role menu",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Name:        "message_id",
							Description: "The ID of the role menu message",
							Type:        discordgo.ApplicationCommandOptionString,
							Required:    true,
						},
						{
							Name:        "role",
							Description: "The role to add",
							Type:        discordgo.ApplicationCommandOptionRole,
							Required:    true,
						},
						{
							Name:        "emoji",
							Description: "The emoji for this role",
							Type:        discordgo.ApplicationCommandOptionString,
							Required:    true,
						},
						{
							Name:        "label",
							Description: "The text label for this role",
							Type:        discordgo.ApplicationCommandOptionString,
							Required:    true,
						},
						{
							Name:        "desc",
							Description: "The description for this role option",
							Type:        discordgo.ApplicationCommandOptionString,
							Required:    false,
						},
					},
				},
			},
			DefaultMemberPermissions: new(int64),
		},
		Handler: func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			if database == nil {
				SendError(s, i, "Database connection not available")
				return
			}

			options := i.ApplicationCommandData().Options
			if len(options) == 0 {
				return
			}
			subcommand := options[0]

			switch subcommand.Name {
			case "setup":
				handleRoleMenuSetup(s, i, database, subcommand.Options)
			case "add_role":
				handleRoleMenuAddRole(s, i, database, subcommand.Options)
			}
		},
	}
}

func handleRoleMenuSetup(s *discordgo.Session, i *discordgo.InteractionCreate, database *db.DB, options []*discordgo.ApplicationCommandInteractionDataOption) {
	title := options[0].StringValue()
	desc := "Please select your roles from the dropdown below."
	if len(options) > 1 {
		desc = options[1].StringValue()
	}

	embed := &discordgo.MessageEmbed{
		Title:       title,
		Description: desc,
		Color:       0x3498db, // Blue
	}

	// Create a placeholder disabled select menu
	components := []discordgo.MessageComponent{
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.SelectMenu{
					CustomID:    "role_menu_select",
					Placeholder: "No roles available yet",
					Disabled:    true,
					Options: []discordgo.SelectMenuOption{
						{
							Label: "Placeholder",
							Value: "placeholder",
						},
					},
				},
			},
		},
	}

	msg, err := s.ChannelMessageSendComplex(i.ChannelID, &discordgo.MessageSend{
		Embeds:     []*discordgo.MessageEmbed{embed},
		Components: components,
	})

	if err != nil {
		slog.Error("Failed to send role menu message", "error", err)
		SendError(s, i, "Failed to create role menu message.")
		return
	}

	// Save to database
	err = database.CreateRoleMenu(context.Background(), msg.ID, i.GuildID, i.ChannelID)
	if err != nil {
		slog.Error("Failed to save role menu to database", "error", err)
		SendError(s, i, "Created message but failed to save to database.")
		return
	}

	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("Role menu created! Use `/rolemenu add_role message_id:%s ...` to add roles to it.", msg.ID),
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}

func handleRoleMenuAddRole(s *discordgo.Session, i *discordgo.InteractionCreate, database *db.DB, options []*discordgo.ApplicationCommandInteractionDataOption) {
	messageID := options[0].StringValue()
	roleID := options[1].RoleValue(s, i.GuildID).ID
	emojiStr := options[2].StringValue()
	label := options[3].StringValue()
	var desc string
	if len(options) > 4 {
		desc = options[4].StringValue()
	}

	// Add to database
	err := database.AddRoleMenuOption(context.Background(), messageID, roleID, emojiStr, label, desc)
	if err != nil {
		slog.Error("Failed to add role menu option to db", "error", err)
		SendError(s, i, "Failed to add role menu option to the database.")
		return
	}

	// Fetch all options for this message to rebuild the select menu
	menuOptions, err := database.GetRoleMenu(context.Background(), messageID)
	if err != nil {
		slog.Error("Failed to fetch role menu options", "error", err)
		SendError(s, i, "Failed to fetch role menu options to update the message.")
		return
	}

	// Rebuild the select menu options
	var selectOptions []discordgo.SelectMenuOption
	for _, opt := range menuOptions {

		// Parse emoji if it's a custom emoji (e.g., <:name:id>)
		var emoji discordgo.ComponentEmoji
		// Simple approach: try to use it directly, if it's unicode it works, if custom we would need parsing.
		// For simplicity, we just set the Name. A full robust bot would parse `<:name:id>`
		emoji.Name = opt.Emoji

		selectOptions = append(selectOptions, discordgo.SelectMenuOption{
			Label:       opt.Label,
			Value:       opt.RoleID,
			Description: opt.Description,
			Emoji:       &emoji,
		})
	}

	// Fetch the original message
	msg, err := s.ChannelMessage(i.ChannelID, messageID)
	if err != nil {
		slog.Error("Failed to fetch original role menu message", "error", err)
		SendError(s, i, "Failed to fetch the role menu message. Make sure the ID is correct and it's in this channel.")
		return
	}

	// Rebuild the components with the new options
	components := []discordgo.MessageComponent{
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.SelectMenu{
					CustomID:    "role_menu_select",
					Placeholder: "Select your roles",
					Options:     selectOptions,
					MinValues:   intPtr(0),
					MaxValues:   len(selectOptions),
				},
			},
		},
	}

	// Edit the message
	_, err = s.ChannelMessageEditComplex(&discordgo.MessageEdit{
		ID:         msg.ID,
		Channel:    msg.ChannelID,
		Embeds:     &msg.Embeds,
		Components: &components,
	})

	if err != nil {
		slog.Error("Failed to edit role menu message", "error", err)
		SendError(s, i, "Failed to update the role menu message with the new option.")
		return
	}

	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "Role added to the menu successfully!",
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}

func intPtr(i int) *int {
	return &i
}
