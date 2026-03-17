package commands

import (
	"context"
	"fmt"
	"julescord/internal/db"

	"github.com/bwmarrin/discordgo"
)

func FactCommand(database *db.DB) *Command {
	return &Command{
		Definition: &discordgo.ApplicationCommand{
			Name:        "fact",
			Description: "Fact system commands",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        "add",
					Description: "Add a new fact to the server",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionString,
							Name:        "text",
							Description: "The fact to add",
							Required:    true,
						},
					},
				},
				{
					Name:        "random",
					Description: "Get a random fact",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
				},
				{
					Name:        "delete",
					Description: "Delete a fact",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionInteger,
							Name:        "id",
							Description: "The ID of the fact to delete",
							Required:    true,
						},
					},
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
			case "add":
				handleAddFact(s, i, database, subcommand.Options)
			case "random":
				handleRandomFact(s, i, database)
			case "delete":
				handleDeleteFact(s, i, database, subcommand.Options)
			}
		},
	}
}

func handleAddFact(s *discordgo.Session, i *discordgo.InteractionCreate, database *db.DB, options []*discordgo.ApplicationCommandInteractionDataOption) {
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
	})
	if err != nil {
		return
	}

	var text string
	for _, opt := range options {
		if opt.Name == "text" {
			text = opt.StringValue()
		}
	}

	id, err := database.AddFact(context.Background(), i.GuildID, i.Member.User.ID, text)
	if err != nil {
		SendErrorEdit(s, i, "Failed to add fact.")
		return
	}

	embed := &discordgo.MessageEmbed{
		Title:       "Fact Added",
		Description: fmt.Sprintf("Successfully added fact **#%d**", id),
		Color:       0x00FF00,
	}

	_, _ = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Embeds: &[]*discordgo.MessageEmbed{embed},
	})
}

func handleRandomFact(s *discordgo.Session, i *discordgo.InteractionCreate, database *db.DB) {
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
	})
	if err != nil {
		return
	}

	id, text, authorID, err := database.GetRandomFact(context.Background(), i.GuildID)
	if err != nil {
		SendErrorEdit(s, i, "Failed to fetch a random fact.")
		return
	}

	if id == nil {
		SendErrorEdit(s, i, "There are no facts in this server. Add some with `/fact add`.")
		return
	}

	authorName := *authorID
	authorUser, err := s.User(*authorID)
	if err == nil {
		authorName = authorUser.Username
	}

	embed := &discordgo.MessageEmbed{
		Title:       fmt.Sprintf("Fact #%d", *id),
		Description: *text,
		Color:       0x3498db,
		Footer: &discordgo.MessageEmbedFooter{
			Text: fmt.Sprintf("Submitted by %s", authorName),
		},
	}

	_, _ = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Embeds: &[]*discordgo.MessageEmbed{embed},
	})
}

func handleDeleteFact(s *discordgo.Session, i *discordgo.InteractionCreate, database *db.DB, options []*discordgo.ApplicationCommandInteractionDataOption) {
	// Require Administrator or Manage Messages
	perms := i.Member.Permissions
	if perms&discordgo.PermissionAdministrator == 0 && perms&discordgo.PermissionManageMessages == 0 {
		SendError(s, i, "You do not have permission to delete facts.")
		return
	}

	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
	})
	if err != nil {
		return
	}

	var factID int
	for _, opt := range options {
		if opt.Name == "id" {
			factID = int(opt.IntValue())
		}
	}

	err = database.DeleteFact(context.Background(), i.GuildID, factID)
	if err != nil {
		SendErrorEdit(s, i, "Failed to delete fact. Make sure the ID is correct.")
		return
	}

	embed := &discordgo.MessageEmbed{
		Title:       "Fact Deleted",
		Description: fmt.Sprintf("Successfully deleted fact **#%d**", factID),
		Color:       0xFF0000,
	}

	_, _ = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Embeds: &[]*discordgo.MessageEmbed{embed},
	})
}
