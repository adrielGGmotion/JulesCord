package commands

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"julescord/internal/db"

	"github.com/bwmarrin/discordgo"
)

// Remind creates the remind command to add, list, and delete reminders.
func Remind(database *db.DB) *Command {
	return &Command{
		Definition: &discordgo.ApplicationCommand{
			Name:        "remind",
			Description: "Set, list, or delete reminders",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        "add",
					Description: "Add a new reminder",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionString,
							Name:        "message",
							Description: "What you want to be reminded about",
							Required:    true,
						},
						{
							Type:        discordgo.ApplicationCommandOptionInteger,
							Name:        "minutes",
							Description: "In how many minutes",
							Required:    true,
						},
					},
				},
				{
					Name:        "list",
					Description: "List your pending reminders",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
				},
				{
					Name:        "delete",
					Description: "Delete a reminder",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionInteger,
							Name:        "id",
							Description: "The ID of the reminder to delete",
							Required:    true,
						},
					},
				},
			},
		},
		Handler: func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			if database == nil {
				SendError(s, i, "Database is not configured.")
				return
			}

			subcommand := i.ApplicationCommandData().Options[0]
			switch subcommand.Name {
			case "add":
				handleAddReminder(s, i, database, subcommand.Options)
			case "list":
				handleListReminders(s, i, database)
			case "delete":
				handleDeleteReminder(s, i, database, subcommand.Options)
			}
		},
	}
}

func handleAddReminder(s *discordgo.Session, i *discordgo.InteractionCreate, database *db.DB, options []*discordgo.ApplicationCommandInteractionDataOption) {
	message := options[0].StringValue()
	minutes := options[1].IntValue()

	if minutes <= 0 {
		SendError(s, i, "Minutes must be greater than 0.")
		return
	}

	var userID string
	if i.Member != nil {
		userID = i.Member.User.ID
	} else {
		userID = i.User.ID
	}

	var guildID *string
	if i.GuildID != "" {
		guildID = &i.GuildID
	}

	dueAt := time.Now().Add(time.Duration(minutes) * time.Minute)

	err := database.AddReminder(context.Background(), userID, i.ChannelID, guildID, message, dueAt)
	if err != nil {
		slog.Error("Failed to add reminder", "error", err)
		SendError(s, i, "Failed to add reminder. Please try again later.")
		return
	}

	timeFormat := fmt.Sprintf("<t:%d:R>", dueAt.Unix())
	embed := &discordgo.MessageEmbed{
		Title:       "⏰ Reminder Set",
		Description: fmt.Sprintf("I will remind you about **%s** %s.", message, timeFormat),
		Color:       0x10B981, // Green
	}
	SendEmbed(s, i, embed)
}

func handleListReminders(s *discordgo.Session, i *discordgo.InteractionCreate, database *db.DB) {
	var userID string
	if i.Member != nil {
		userID = i.Member.User.ID
	} else {
		userID = i.User.ID
	}

	reminders, err := database.GetPendingReminders(context.Background(), userID)
	if err != nil {
		slog.Error("Failed to list reminders", "error", err)
		SendError(s, i, "Failed to fetch reminders.")
		return
	}

	if len(reminders) == 0 {
		embed := &discordgo.MessageEmbed{
			Title:       "⏰ Your Reminders",
			Description: "You don't have any pending reminders.",
			Color:       0x3B82F6, // Blue
		}
		SendEmbed(s, i, embed)
		return
	}

	description := ""
	for _, r := range reminders {
		description += fmt.Sprintf("**ID: %d** — %s (<t:%d:R>)\n", r.ID, r.Message, r.DueAt.Unix())
	}

	embed := &discordgo.MessageEmbed{
		Title:       "⏰ Your Reminders",
		Description: description,
		Color:       0x3B82F6, // Blue
		Footer: &discordgo.MessageEmbedFooter{
			Text: "Use /remind delete <id> to remove a reminder.",
		},
	}
	SendEmbed(s, i, embed)
}

func handleDeleteReminder(s *discordgo.Session, i *discordgo.InteractionCreate, database *db.DB, options []*discordgo.ApplicationCommandInteractionDataOption) {
	id := options[0].IntValue()

	var userID string
	if i.Member != nil {
		userID = i.Member.User.ID
	} else {
		userID = i.User.ID
	}

	deleted, err := database.DeleteReminder(context.Background(), int(id), userID)
	if err != nil {
		slog.Error("Failed to delete reminder", "error", err)
		SendError(s, i, "Failed to delete reminder.")
		return
	}

	if !deleted {
		SendError(s, i, fmt.Sprintf("Reminder with ID %d not found or you don't own it.", id))
		return
	}

	embed := &discordgo.MessageEmbed{
		Title:       "🗑️ Reminder Deleted",
		Description: fmt.Sprintf("Successfully deleted reminder ID %d.", id),
		Color:       0x10B981, // Green
	}
	SendEmbed(s, i, embed)
}
