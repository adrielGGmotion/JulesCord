package commands

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/bwmarrin/discordgo"
	"julescord/internal/db"
)

func NewReactionTriggerCommand(db *db.DB) *Command {
	return &Command{
		Definition: &discordgo.ApplicationCommand{
			Name:                     "reactiontrigger",
			Description:              "Manage reaction triggers for this server",
			DefaultMemberPermissions: func() *int64 { p := int64(discordgo.PermissionManageServer); return &p }(),
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        "add",
					Description: "Add a new reaction trigger",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionString,
							Name:        "trigger",
							Description: "The word or phrase to trigger on",
							Required:    true,
						},
						{
							Type:        discordgo.ApplicationCommandOptionString,
							Name:        "emoji",
							Description: "The emoji to react with",
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
							Type:        discordgo.ApplicationCommandOptionInteger,
							Name:        "id",
							Description: "The ID of the trigger to remove (use /reactiontrigger list to find IDs)",
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

			subCmd := options[0]

			switch subCmd.Name {
			case "add":
				handleAddReactionTrigger(s, i, db, subCmd.Options)
			case "remove":
				handleRemoveReactionTrigger(s, i, db, subCmd.Options)
			case "list":
				handleListReactionTriggers(s, i, db)
			}
		},
	}
}

func handleAddReactionTrigger(s *discordgo.Session, i *discordgo.InteractionCreate, db *db.DB, options []*discordgo.ApplicationCommandInteractionDataOption) {
	trigger := ""
	emoji := ""

	for _, opt := range options {
		switch opt.Name {
		case "trigger":
			trigger = strings.ToLower(opt.StringValue())
		case "emoji":

			emoji = opt.StringValue()
			// Handle custom emojis (<:name:id> or <a:name:id>)
			if strings.HasPrefix(emoji, "<") && strings.HasSuffix(emoji, ">") {
				parts := strings.Split(emoji, ":")
				if len(parts) >= 3 {
					// <a:name:id> -> a:name:id
					// <:name:id> -> name:id
					name := parts[1]
					id := strings.TrimSuffix(parts[2], ">")
					emoji = name + ":" + id
				}
			}
		}
	}

	err := db.AddReactionTrigger(context.Background(), i.GuildID, trigger, emoji)
	if err != nil {
		slog.Error("Failed to add reaction trigger", "guild_id", i.GuildID, "error", err)
		SendError(s, i, "Failed to add reaction trigger. Please try again later.")
		return
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("✅ Added reaction trigger for `%s` with %s", trigger, emoji),
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}

func handleRemoveReactionTrigger(s *discordgo.Session, i *discordgo.InteractionCreate, db *db.DB, options []*discordgo.ApplicationCommandInteractionDataOption) {
	id := 0

	for _, opt := range options {
		if opt.Name == "id" {
			id = int(opt.IntValue())
		}
	}

	err := db.RemoveReactionTrigger(context.Background(), i.GuildID, id)
	if err != nil {
		if err.Error() == "reaction trigger not found" {
			SendError(s, i, "Reaction trigger not found. Please check the ID and try again.")
			return
		}
		slog.Error("Failed to remove reaction trigger", "guild_id", i.GuildID, "id", id, "error", err)
		SendError(s, i, "Failed to remove reaction trigger. Please try again later.")
		return
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("✅ Removed reaction trigger with ID `%d`", id),
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}

func handleListReactionTriggers(s *discordgo.Session, i *discordgo.InteractionCreate, db *db.DB) {
	triggers, err := db.GetReactionTriggers(context.Background(), i.GuildID)
	if err != nil {
		slog.Error("Failed to list reaction triggers", "guild_id", i.GuildID, "error", err)
		SendError(s, i, "Failed to list reaction triggers. Please try again later.")
		return
	}

	if len(triggers) == 0 {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "There are no reaction triggers configured for this server.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	var sb strings.Builder
	for _, t := range triggers {
		sb.WriteString(fmt.Sprintf("**ID:** `%d` | **Trigger:** `%s` | **Emoji:** %s\n", t.ID, t.TriggerWord, t.Emoji))
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{
				{
					Title:       "Reaction Triggers",
					Description: sb.String(),
					Color:       0x00FF00,
				},
			},
		},
	})
}
