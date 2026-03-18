package commands

import (
	"context"
	"fmt"
	"strings"

	"julescord/internal/db"

	"github.com/bwmarrin/discordgo"
)

// ReactionGroup creates the /reactiongroup command.
func ReactionGroup(database *db.DB) *Command {
	return &Command{
		Definition: &discordgo.ApplicationCommand{
			Name:                     "reactiongroup",
			Description:              "Manage reaction role groups",
			DefaultMemberPermissions: func() *int64 { p := int64(discordgo.PermissionManageRoles); return &p }(),
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        "create",
					Description: "Create a new reaction role group",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Name:        "name",
							Description: "The name of the group",
							Type:        discordgo.ApplicationCommandOptionString,
							Required:    true,
						},
						{
							Name:        "exclusive",
							Description: "Whether to allow only one role from this group at a time",
							Type:        discordgo.ApplicationCommandOptionBoolean,
							Required:    true,
						},
						{
							Name:        "max_roles",
							Description: "Maximum roles a user can have from this group (0 for unlimited)",
							Type:        discordgo.ApplicationCommandOptionInteger,
							Required:    false,
						},
					},
				},
				{
					Name:        "list",
					Description: "List all reaction role groups",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
				},
				{
					Name:        "addrole",
					Description: "Add an existing reaction role to a group",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Name:        "group_name",
							Description: "The name of the group",
							Type:        discordgo.ApplicationCommandOptionString,
							Required:    true,
						},
						{
							Name:        "message_id",
							Description: "The message ID of the reaction role",
							Type:        discordgo.ApplicationCommandOptionString,
							Required:    true,
						},
						{
							Name:        "emoji",
							Description: "The emoji used for the reaction role (e.g. ⭐ or custom name/ID)",
							Type:        discordgo.ApplicationCommandOptionString,
							Required:    true,
						},
					},
				},
			},
		},
		Handler: func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			if database == nil {
				respondError(s, i, "Database is not configured.")
				return
			}

			options := i.ApplicationCommandData().Options
			if len(options) == 0 {
				return
			}
			subCommand := options[0].Name

			switch subCommand {
			case "create":
				handleReactionGroupCreate(s, i, database, options[0].Options)
			case "list":
				handleReactionGroupList(s, i, database)
			case "addrole":
				handleReactionGroupAddRole(s, i, database, options[0].Options)
			}
		},
	}
}

func handleReactionGroupCreate(s *discordgo.Session, i *discordgo.InteractionCreate, database *db.DB, options []*discordgo.ApplicationCommandInteractionDataOption) {
	name := ""
	exclusive := false
	maxRoles := 0

	for _, opt := range options {
		switch opt.Name {
		case "name":
			name = opt.Value.(string)
		case "exclusive":
			exclusive = opt.Value.(bool)
		case "max_roles":
			maxRoles = int(opt.Value.(float64))
		}
	}

	group := &db.ReactionRoleGroup{
		GuildID:     i.GuildID,
		Name:        name,
		IsExclusive: exclusive,
		MaxRoles:    maxRoles,
	}

	err := database.CreateReactionRoleGroup(context.Background(), group)
	if err != nil {
		respondError(s, i, "Failed to create reaction role group.")
		return
	}

	respondSuccess(s, i, fmt.Sprintf("Successfully created reaction role group `%s` (Exclusive: %t, Max Roles: %d)", name, exclusive, maxRoles))
}

func handleReactionGroupList(s *discordgo.Session, i *discordgo.InteractionCreate, database *db.DB) {
	groups, err := database.GetReactionRoleGroups(context.Background(), i.GuildID)
	if err != nil {
		respondError(s, i, "Failed to fetch reaction role groups.")
		return
	}

	if len(groups) == 0 {
		respondSuccess(s, i, "No reaction role groups found.")
		return
	}

	var sb strings.Builder
	for _, g := range groups {
		sb.WriteString(fmt.Sprintf("**%s** (ID: %d)\nExclusive: %t | Max Roles: %d\n\n", g.Name, g.ID, g.IsExclusive, g.MaxRoles))
	}

	embed := &discordgo.MessageEmbed{
		Title:       "Reaction Role Groups",
		Description: sb.String(),
		Color:       0x00FF00,
	}

	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
		},
	})
}

func handleReactionGroupAddRole(s *discordgo.Session, i *discordgo.InteractionCreate, database *db.DB, options []*discordgo.ApplicationCommandInteractionDataOption) {
	groupName := ""
	messageID := ""
	emojiInput := ""

	for _, opt := range options {
		switch opt.Name {
		case "group_name":
			groupName = opt.Value.(string)
		case "message_id":
			messageID = opt.Value.(string)
		case "emoji":
			emojiInput = opt.Value.(string)
		}
	}

	groups, err := database.GetReactionRoleGroups(context.Background(), i.GuildID)
	if err != nil {
		respondError(s, i, "Failed to fetch reaction role groups.")
		return
	}

	var groupID int
	found := false
	for _, g := range groups {
		if strings.EqualFold(g.Name, groupName) {
			groupID = g.ID
			found = true
			break
		}
	}

	if !found {
		respondError(s, i, fmt.Sprintf("Reaction role group `%s` not found.", groupName))
		return
	}

	// Parse emoji
	emojiName := emojiInput
	emojiID := ""
	if strings.HasPrefix(emojiInput, "<") && strings.HasSuffix(emojiInput, ">") {
		// Custom emoji format: <:name:id> or <a:name:id>
		parts := strings.Split(emojiInput, ":")
		if len(parts) == 3 {
			emojiName = parts[1]
			emojiID = strings.TrimSuffix(parts[2], ">")
		}
	}

	rr, err := database.GetReactionRole(context.Background(), messageID, emojiName, emojiID)
	if err != nil {
		respondError(s, i, "Failed to fetch reaction role.")
		return
	}
	if rr == nil {
		respondError(s, i, "Reaction role not found for the given message and emoji.")
		return
	}

	err = database.AssignRoleToGroup(context.Background(), messageID, emojiName, emojiID, groupID)
	if err != nil {
		respondError(s, i, "Failed to assign reaction role to group.")
		return
	}

	respondSuccess(s, i, fmt.Sprintf("Successfully assigned reaction role to group `%s`.", groupName))
}

func respondError(s *discordgo.Session, i *discordgo.InteractionCreate, message string) {
	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "❌ " + message,
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}

func respondSuccess(s *discordgo.Session, i *discordgo.InteractionCreate, message string) {
	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "✅ " + message,
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}
