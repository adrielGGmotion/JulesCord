package commands

import (
	"context"
	"fmt"
	"strings"

	"julescord/internal/db"

	"github.com/bwmarrin/discordgo"
)

func ReactionTrigger(database *db.DB) *Command {
	return &Command{
		Definition: &discordgo.ApplicationCommand{
			Name:                     "reactiontrigger",
			Description:              "Manage reaction triggers",
			DMPermission:             func(b bool) *bool { return &b }(false),
			DefaultMemberPermissions: func(p int64) *int64 { return &p }(discordgo.PermissionManageGuild),
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        "add",
					Description: "Add a new reaction trigger",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Name:        "keyword",
							Description: "The keyword to trigger on",
							Type:        discordgo.ApplicationCommandOptionString,
							Required:    true,
						},
						{
							Name:        "emoji",
							Description: "The emoji to react with",
							Type:        discordgo.ApplicationCommandOptionString,
							Required:    true,
						},
					},
				},
				{
					Name:        "remove",
					Description: "Remove a reaction trigger",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Name:        "keyword",
							Description: "The keyword of the trigger to remove",
							Type:        discordgo.ApplicationCommandOptionString,
							Required:    true,
						},
					},
				},
				{
					Name:        "list",
					Description: "List all reaction triggers",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
				},
			},
		},
		Handler: func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			options := i.ApplicationCommandData().Options
			if len(options) == 0 {
				return
			}

			subcommand := options[0]

			switch subcommand.Name {
			case "add":
				handleAddReactionTrigger(s, i, database, subcommand.Options)
			case "remove":
				handleRemoveReactionTrigger(s, i, database, subcommand.Options)
			case "list":
				handleListReactionTriggers(s, i, database)
			}
		},
	}
}

func handleAddReactionTrigger(s *discordgo.Session, i *discordgo.InteractionCreate, database *db.DB, options []*discordgo.ApplicationCommandInteractionDataOption) {
	var keyword, emoji string

	for _, opt := range options {
		switch opt.Name {
		case "keyword":
			keyword = strings.ToLower(opt.Value.(string))
		case "emoji":
			emoji = opt.Value.(string)
		}
	}

	err := database.AddReactionTrigger(context.Background(), i.GuildID, keyword, emoji)
	if err != nil {
		SendError(s, i, "Failed to add reaction trigger.")
		return
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("Added reaction trigger for keyword `%s` with emoji %s.", keyword, emoji),
		},
	})
}

func handleRemoveReactionTrigger(s *discordgo.Session, i *discordgo.InteractionCreate, database *db.DB, options []*discordgo.ApplicationCommandInteractionDataOption) {
	var keyword string

	for _, opt := range options {
		if opt.Name == "keyword" {
			keyword = strings.ToLower(opt.Value.(string))
		}
	}

	err := database.RemoveReactionTrigger(context.Background(), i.GuildID, keyword)
	if err != nil {
		SendError(s, i, "Failed to remove reaction trigger.")
		return
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("Removed reaction trigger for keyword `%s`.", keyword),
		},
	})
}

func handleListReactionTriggers(s *discordgo.Session, i *discordgo.InteractionCreate, database *db.DB) {
	triggers, err := database.GetReactionTriggers(context.Background(), i.GuildID)
	if err != nil {
		SendError(s, i, "Failed to fetch reaction triggers.")
		return
	}

	if len(triggers) == 0 {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "There are no reaction triggers configured for this server.",
			},
		})
		return
	}

	var description strings.Builder
	for _, t := range triggers {
		description.WriteString(fmt.Sprintf("**Keyword:** `%s` -> %s\n", t.Keyword, t.Emoji))
	}

	embed := &discordgo.MessageEmbed{
		Title:       "Reaction Triggers",
		Description: description.String(),
		Color:       0x00FF00,
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
		},
	})
}
