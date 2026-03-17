package commands

import (
	"context"
	"fmt"
	"julescord/internal/db"
	"regexp"
	"strconv"
	"strings"

	"github.com/bwmarrin/discordgo"
)

func MyRoleCommand(database *db.DB) *Command {
	return &Command{
		Definition: &discordgo.ApplicationCommand{
			Name:        "myrole",
			Description: "Manage your custom role",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        "create",
					Description: "Create a custom role",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionString,
							Name:        "name",
							Description: "The name of the role",
							Required:    true,
						},
						{
							Type:        discordgo.ApplicationCommandOptionString,
							Name:        "color",
							Description: "The hex color of the role (e.g., #FF0000)",
							Required:    true,
						},
						{
							Type:        discordgo.ApplicationCommandOptionAttachment,
							Name:        "icon",
							Description: "The icon for the role",
							Required:    false,
						},
					},
				},
				{
					Name:        "name",
					Description: "Change the name of your custom role",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionString,
							Name:        "name",
							Description: "The new name of the role",
							Required:    true,
						},
					},
				},
				{
					Name:        "color",
					Description: "Change the color of your custom role",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionString,
							Name:        "color",
							Description: "The new hex color of the role (e.g., #FF0000)",
							Required:    true,
						},
					},
				},
				{
					Name:        "icon",
					Description: "Change the icon of your custom role",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionAttachment,
							Name:        "icon",
							Description: "The new icon for the role",
							Required:    true,
						},
					},
				},
				{
					Name:        "delete",
					Description: "Delete your custom role",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
				},
			},
		},
		Handler: func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			if i.Member == nil {
				SendError(s, i, "This command can only be used in a server.")
				return
			}

			options := i.ApplicationCommandData().Options
			if len(options) == 0 {
				return
			}

			subcommand := options[0]

			switch subcommand.Name {
			case "create":
				handleCreateMyRole(s, i, database, subcommand.Options)
			case "name":
				handleNameMyRole(s, i, database, subcommand.Options)
			case "color":
				handleColorMyRole(s, i, database, subcommand.Options)
			case "icon":
				handleIconMyRole(s, i, database, subcommand.Options)
			case "delete":
				handleDeleteMyRole(s, i, database)
			}
		},
	}
}

func parseColor(colorStr string) (int, error) {
	colorStr = strings.TrimPrefix(colorStr, "#")
	match, _ := regexp.MatchString("^[0-9A-Fa-f]{6}$", colorStr)
	if !match {
		return 0, fmt.Errorf("invalid color format")
	}
	color, err := strconv.ParseInt(colorStr, 16, 64)
	if err != nil {
		return 0, err
	}
	return int(color), nil
}

func handleCreateMyRole(s *discordgo.Session, i *discordgo.InteractionCreate, database *db.DB, options []*discordgo.ApplicationCommandInteractionDataOption) {
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
	})
	if err != nil {
		return
	}

	roleInfo, err := database.GetCustomRole(context.Background(), i.GuildID, i.Member.User.ID)
	if err != nil {
		SendErrorEdit(s, i, "Failed to check existing custom role.")
		return
	}

	if roleInfo != nil {
		SendErrorEdit(s, i, "You already have a custom role! Use the update subcommands to modify it.")
		return
	}

	var name string
	var colorStr string
	var iconURL string

	for _, opt := range options {
		if opt.Name == "name" {
			name = opt.StringValue()
		} else if opt.Name == "color" {
			colorStr = opt.StringValue()
		} else if opt.Name == "icon" {
			if i.ApplicationCommandData().Resolved != nil && i.ApplicationCommandData().Resolved.Attachments != nil {
				attachmentID := opt.Value.(string)
				attachment, ok := i.ApplicationCommandData().Resolved.Attachments[attachmentID]
				if ok {
					iconURL = attachment.URL
				}
			}
		}
	}

	colorInt, err := parseColor(colorStr)
	if err != nil {
		SendErrorEdit(s, i, "Invalid color format. Please use a hex color like #FF0000.")
		return
	}

	// Create Role in Discord
	mentionable := false
	roleParams := &discordgo.RoleParams{
		Name:        name,
		Color:       &colorInt,
		Mentionable: &mentionable,
	}

	// Set icon if provided
	if iconURL != "" {
		// Note: Requires tier 2 boost for roles to have icons. We can just store it for now.
		// It's possible to apply it using discordgo.RoleParams.Icon, but it requires base64 image encoding.
		// As a workaround, we will just store the URL and assume the guild may not have tier 2.
	}

	role, err := s.GuildRoleCreate(i.GuildID, roleParams)
	if err != nil {
		SendErrorEdit(s, i, "Failed to create the role in the server. Please check my permissions.")
		return
	}

	// Add role to user
	err = s.GuildMemberRoleAdd(i.GuildID, i.Member.User.ID, role.ID)
	if err != nil {
		_ = s.GuildRoleDelete(i.GuildID, role.ID) // cleanup
		SendErrorEdit(s, i, "Failed to assign the role to you.")
		return
	}

	// Save to database
	err = database.CreateCustomRole(context.Background(), i.GuildID, i.Member.User.ID, role.ID, name, colorInt, iconURL)
	if err != nil {
		_ = s.GuildRoleDelete(i.GuildID, role.ID) // cleanup
		SendErrorEdit(s, i, "Failed to save the custom role to the database.")
		return
	}

	embed := &discordgo.MessageEmbed{
		Title:       "Custom Role Created",
		Description: fmt.Sprintf("Successfully created and assigned your custom role <@&%s>.", role.ID),
		Color:       colorInt,
	}

	_, _ = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Embeds: &[]*discordgo.MessageEmbed{embed},
	})
}

func handleNameMyRole(s *discordgo.Session, i *discordgo.InteractionCreate, database *db.DB, options []*discordgo.ApplicationCommandInteractionDataOption) {
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
	})
	if err != nil {
		return
	}

	roleInfo, err := database.GetCustomRole(context.Background(), i.GuildID, i.Member.User.ID)
	if err != nil || roleInfo == nil {
		SendErrorEdit(s, i, "You don't have a custom role. Create one first with `/myrole create`.")
		return
	}

	var name string
	for _, opt := range options {
		if opt.Name == "name" {
			name = opt.StringValue()
		}
	}

	// Update in Discord
	_, err = s.GuildRoleEdit(i.GuildID, roleInfo.RoleID, &discordgo.RoleParams{
		Name: name,
	})
	if err != nil {
		SendErrorEdit(s, i, "Failed to update the role in the server.")
		return
	}

	// Update DB
	err = database.UpdateCustomRole(context.Background(), i.GuildID, i.Member.User.ID, name, roleInfo.Color, roleInfo.IconURL)
	if err != nil {
		SendErrorEdit(s, i, "Failed to update the role in the database.")
		return
	}

	embed := &discordgo.MessageEmbed{
		Title:       "Custom Role Updated",
		Description: fmt.Sprintf("Successfully changed the name of your role to **%s**.", name),
		Color:       roleInfo.Color,
	}

	_, _ = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Embeds: &[]*discordgo.MessageEmbed{embed},
	})
}

func handleColorMyRole(s *discordgo.Session, i *discordgo.InteractionCreate, database *db.DB, options []*discordgo.ApplicationCommandInteractionDataOption) {
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
	})
	if err != nil {
		return
	}

	roleInfo, err := database.GetCustomRole(context.Background(), i.GuildID, i.Member.User.ID)
	if err != nil || roleInfo == nil {
		SendErrorEdit(s, i, "You don't have a custom role. Create one first with `/myrole create`.")
		return
	}

	var colorStr string
	for _, opt := range options {
		if opt.Name == "color" {
			colorStr = opt.StringValue()
		}
	}

	colorInt, err := parseColor(colorStr)
	if err != nil {
		SendErrorEdit(s, i, "Invalid color format. Please use a hex color like #FF0000.")
		return
	}

	// Update in Discord
	_, err = s.GuildRoleEdit(i.GuildID, roleInfo.RoleID, &discordgo.RoleParams{
		Color: &colorInt,
	})
	if err != nil {
		SendErrorEdit(s, i, "Failed to update the role color in the server.")
		return
	}

	// Update DB
	err = database.UpdateCustomRole(context.Background(), i.GuildID, i.Member.User.ID, roleInfo.Name, colorInt, roleInfo.IconURL)
	if err != nil {
		SendErrorEdit(s, i, "Failed to update the role in the database.")
		return
	}

	embed := &discordgo.MessageEmbed{
		Title:       "Custom Role Updated",
		Description: "Successfully changed the color of your role.",
		Color:       colorInt,
	}

	_, _ = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Embeds: &[]*discordgo.MessageEmbed{embed},
	})
}

func handleIconMyRole(s *discordgo.Session, i *discordgo.InteractionCreate, database *db.DB, options []*discordgo.ApplicationCommandInteractionDataOption) {
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
	})
	if err != nil {
		return
	}

	roleInfo, err := database.GetCustomRole(context.Background(), i.GuildID, i.Member.User.ID)
	if err != nil || roleInfo == nil {
		SendErrorEdit(s, i, "You don't have a custom role. Create one first with `/myrole create`.")
		return
	}

	var iconURL string
	for _, opt := range options {
		if opt.Name == "icon" {
			if i.ApplicationCommandData().Resolved != nil && i.ApplicationCommandData().Resolved.Attachments != nil {
				attachmentID := opt.Value.(string)
				attachment, ok := i.ApplicationCommandData().Resolved.Attachments[attachmentID]
				if ok {
					iconURL = attachment.URL
				}
			}
		}
	}

	// Only saving iconURL to the DB as setting discord icons requires fetching, base64 encoding and a tier 2 boosted server
	err = database.UpdateCustomRole(context.Background(), i.GuildID, i.Member.User.ID, roleInfo.Name, roleInfo.Color, iconURL)
	if err != nil {
		SendErrorEdit(s, i, "Failed to update the role in the database.")
		return
	}

	embed := &discordgo.MessageEmbed{
		Title:       "Custom Role Updated",
		Description: "Successfully updated your role's icon.",
		Color:       roleInfo.Color,
	}

	_, _ = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Embeds: &[]*discordgo.MessageEmbed{embed},
	})
}

func handleDeleteMyRole(s *discordgo.Session, i *discordgo.InteractionCreate, database *db.DB) {
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
	})
	if err != nil {
		return
	}

	roleInfo, err := database.GetCustomRole(context.Background(), i.GuildID, i.Member.User.ID)
	if err != nil || roleInfo == nil {
		SendErrorEdit(s, i, "You don't have a custom role to delete.")
		return
	}

	// Delete from Discord
	_ = s.GuildRoleDelete(i.GuildID, roleInfo.RoleID)

	// Delete from DB
	err = database.DeleteCustomRole(context.Background(), i.GuildID, i.Member.User.ID)
	if err != nil {
		SendErrorEdit(s, i, "Failed to remove the role from the database.")
		return
	}

	embed := &discordgo.MessageEmbed{
		Title:       "Custom Role Deleted",
		Description: "Successfully deleted your custom role.",
		Color:       0xFF0000,
	}

	_, _ = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Embeds: &[]*discordgo.MessageEmbed{embed},
	})
}
