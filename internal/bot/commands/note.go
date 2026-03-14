package commands

import (
	"context"
	"fmt"
	"julescord/internal/db"
	"log/slog"

	"github.com/bwmarrin/discordgo"
)

// NoteCommand represents the /note command for adding, listing, and removing user notes.
func NoteCommand(database *db.DB) *Command {
	return &Command{
		Definition: &discordgo.ApplicationCommand{
			Name:        "note",
			Description: "Manage notes for users",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "add",
					Description: "Add a note to a user",
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionUser,
							Name:        "user",
							Description: "The user to add a note to",
							Required:    true,
						},
						{
							Type:        discordgo.ApplicationCommandOptionString,
							Name:        "note",
							Description: "The note content",
							Required:    true,
						},
					},
				},
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "list",
					Description: "List notes for a user",
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionUser,
							Name:        "user",
							Description: "The user to list notes for",
							Required:    true,
						},
					},
				},
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "remove",
					Description: "Remove a specific note by ID",
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionInteger,
							Name:        "id",
							Description: "The ID of the note to remove",
							Required:    true,
						},
					},
				},
			},
			DefaultMemberPermissions: func(p int64) *int64 { return &p }(discordgo.PermissionManageMessages),
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
				var user *discordgo.User
				var note string
				for _, opt := range subcommand.Options {
					switch opt.Name {
					case "user":
						user = opt.UserValue(s)
					case "note":
						note = opt.StringValue()
					}
				}

				if user == nil {
					SendError(s, i, "Invalid user.")
					return
				}

				err := database.AddNote(context.Background(), i.GuildID, user.ID, i.Member.User.ID, note)
				if err != nil {
					slog.Error("Failed to add note", "error", err)
					SendError(s, i, "Failed to add note.")
					return
				}

				SendEmbed(s, i, &discordgo.MessageEmbed{
					Title:       "Note Added",
					Description: fmt.Sprintf("Added a note to <@%s>", user.ID),
					Color:       0x00FF00,
					Fields: []*discordgo.MessageEmbedField{
						{
							Name:  "Note",
							Value: note,
						},
					},
				})

			case "list":
				var user *discordgo.User
				for _, opt := range subcommand.Options {
					if opt.Name == "user" {
						user = opt.UserValue(s)
					}
				}

				if user == nil {
					SendError(s, i, "Invalid user.")
					return
				}

				notes, err := database.GetNotes(context.Background(), i.GuildID, user.ID)
				if err != nil {
					slog.Error("Failed to get notes", "error", err)
					SendError(s, i, "Failed to retrieve notes.")
					return
				}

				if len(notes) == 0 {
					SendEmbed(s, i, &discordgo.MessageEmbed{
						Title:       "Notes for " + user.Username,
						Description: "This user has no notes.",
						Color:       0xAAAAAA,
					})
					return
				}

				embed := &discordgo.MessageEmbed{
					Title: "Notes for " + user.Username,
					Color: 0x00AFFF,
				}

				for _, n := range notes {
					embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
						Name:  fmt.Sprintf("ID: %d | By: <@%s> | Date: <t:%d:R>", n.ID, n.ModeratorID, n.CreatedAt.Unix()),
						Value: n.Note,
					})
				}

				SendEmbed(s, i, embed)

			case "remove":
				var id int
				for _, opt := range subcommand.Options {
					if opt.Name == "id" {
						id = int(opt.IntValue())
					}
				}

				err := database.RemoveNote(context.Background(), i.GuildID, id)
				if err != nil {
					slog.Error("Failed to remove note", "error", err)
					SendError(s, i, "Failed to remove note.")
					return
				}

				SendEmbed(s, i, &discordgo.MessageEmbed{
					Title:       "Note Removed",
					Description: fmt.Sprintf("Removed note with ID %d.", id),
					Color:       0xFF0000,
				})
			}
		},
	}
}
