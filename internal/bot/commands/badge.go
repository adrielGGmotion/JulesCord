package commands

import (
	"fmt"
	"julescord/internal/db"
	"strings"

	"github.com/bwmarrin/discordgo"
)

// Badge returns the badge command definition.
func Badge(database *db.DB) *Command {
	return &Command{
		Definition: &discordgo.ApplicationCommand{
			Name:        "badge",
			Description: "Manage and view user badges",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        "create",
					Description: "Create a new badge (Admin only)",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionString,
							Name:        "name",
							Description: "The name of the badge",
							Required:    true,
						},
						{
							Type:        discordgo.ApplicationCommandOptionString,
							Name:        "emoji",
							Description: "The emoji for the badge",
							Required:    true,
						},
						{
							Type:        discordgo.ApplicationCommandOptionString,
							Name:        "description",
							Description: "The description of the badge",
							Required:    true,
						},
					},
				},
				{
					Name:        "award",
					Description: "Award a badge to a user (Admin only)",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionUser,
							Name:        "user",
							Description: "The user to award the badge to",
							Required:    true,
						},
						{
							Type:        discordgo.ApplicationCommandOptionInteger,
							Name:        "badge_id",
							Description: "The ID of the badge to award",
							Required:    true,
						},
					},
				},
				{
					Name:        "remove",
					Description: "Remove a badge from a user (Admin only)",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionUser,
							Name:        "user",
							Description: "The user to remove the badge from",
							Required:    true,
						},
						{
							Type:        discordgo.ApplicationCommandOptionInteger,
							Name:        "badge_id",
							Description: "The ID of the badge to remove",
							Required:    true,
						},
					},
				},
				{
					Name:        "list",
					Description: "List all available badges",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
				},
			},
		},
		Handler: func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			options := i.ApplicationCommandData().Options
			if len(options) == 0 {
				return
			}

			subcommand := options[0].Name

			switch subcommand {
			case "create":
				handleCreateBadge(s, i, database)
			case "award":
				handleAwardBadge(s, i, database)
			case "remove":
				handleRemoveBadge(s, i, database)
			case "list":
				handleListBadges(s, i, database)
			}
		},
	}
}

func handleCreateBadge(s *discordgo.Session, i *discordgo.InteractionCreate, database *db.DB) {
	// Admin check
	if i.Member == nil || i.Member.Permissions&discordgo.PermissionAdministrator == 0 {
		SendError(s, i, "You need Administrator permissions to use this command.")
		return
	}

	options := i.ApplicationCommandData().Options[0].Options
	var name, emoji, description string
	for _, opt := range options {
		switch opt.Name {
		case "name":
			name = opt.StringValue()
		case "emoji":
			emoji = opt.StringValue()
		case "description":
			description = opt.StringValue()
		}
	}

	err := database.CreateBadge(name, emoji, description)
	if err != nil {
		SendError(s, i, "Failed to create badge: "+err.Error())
		return
	}

	embed := &discordgo.MessageEmbed{
		Title:       "Badge Created",
		Description: fmt.Sprintf("Successfully created badge **%s %s**\n*\"%s\"*", emoji, name, description),
		Color:       0x00FF00,
	}
	SendEmbed(s, i, embed)
}

func handleAwardBadge(s *discordgo.Session, i *discordgo.InteractionCreate, database *db.DB) {
	// Admin check
	if i.Member == nil || i.Member.Permissions&discordgo.PermissionAdministrator == 0 {
		SendError(s, i, "You need Administrator permissions to use this command.")
		return
	}

	options := i.ApplicationCommandData().Options[0].Options
	var targetUser *discordgo.User
	var badgeID int64
	for _, opt := range options {
		switch opt.Name {
		case "user":
			targetUser = opt.UserValue(s)
		case "badge_id":
			badgeID = opt.IntValue()
		}
	}

	err := database.AwardBadge(targetUser.ID, int(badgeID))
	if err != nil {
		SendError(s, i, "Failed to award badge: "+err.Error())
		return
	}

	embed := &discordgo.MessageEmbed{
		Title:       "Badge Awarded",
		Description: fmt.Sprintf("Successfully awarded badge (ID: %d) to <@%s>", badgeID, targetUser.ID),
		Color:       0x00FF00,
	}
	SendEmbed(s, i, embed)
}

func handleRemoveBadge(s *discordgo.Session, i *discordgo.InteractionCreate, database *db.DB) {
	// Admin check
	if i.Member == nil || i.Member.Permissions&discordgo.PermissionAdministrator == 0 {
		SendError(s, i, "You need Administrator permissions to use this command.")
		return
	}

	options := i.ApplicationCommandData().Options[0].Options
	var targetUser *discordgo.User
	var badgeID int64
	for _, opt := range options {
		switch opt.Name {
		case "user":
			targetUser = opt.UserValue(s)
		case "badge_id":
			badgeID = opt.IntValue()
		}
	}

	err := database.RemoveBadge(targetUser.ID, int(badgeID))
	if err != nil {
		SendError(s, i, "Failed to remove badge: "+err.Error())
		return
	}

	embed := &discordgo.MessageEmbed{
		Title:       "Badge Removed",
		Description: fmt.Sprintf("Successfully removed badge (ID: %d) from <@%s>", badgeID, targetUser.ID),
		Color:       0x00FF00,
	}
	SendEmbed(s, i, embed)
}

func handleListBadges(s *discordgo.Session, i *discordgo.InteractionCreate, database *db.DB) {
	badges, err := database.GetAllBadges()
	if err != nil {
		SendError(s, i, "Failed to fetch badges: "+err.Error())
		return
	}

	if len(badges) == 0 {
		embed := &discordgo.MessageEmbed{
			Title:       "Available Badges",
			Description: "There are no badges available yet.",
			Color:       0x3498DB,
		}
		SendEmbed(s, i, embed)
		return
	}

	var sb strings.Builder
	for _, b := range badges {
		sb.WriteString(fmt.Sprintf("**ID: %d** | %s **%s**\n* %s\n\n", b.ID, b.Emoji, b.Name, b.Description))
	}

	embed := &discordgo.MessageEmbed{
		Title:       "Available Badges",
		Description: sb.String(),
		Color:       0x3498DB,
	}
	SendEmbed(s, i, embed)
}
