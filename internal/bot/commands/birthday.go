package commands

import (
	"context"
	"fmt"
	"julescord/internal/db"

	"github.com/bwmarrin/discordgo"
)

// Birthday returns the definition for the /birthday command.
func Birthday(database *db.DB) *Command {
	return &Command{
		Definition: &discordgo.ApplicationCommand{
			Name:        "birthday",
			DMPermission: new(bool),

			Description: "Manage server birthdays",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        "setup",
					Description: "Set the channel for birthday announcements (Admin only)",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Name:        "channel",
							Description: "The channel to send birthday announcements",
							Type:        discordgo.ApplicationCommandOptionChannel,
							Required:    true,
						},
					},
				},
				{
					Name:        "set",
					Description: "Set your birthday",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Name:        "month",
							Description: "Month of birth (1-12)",
							Type:        discordgo.ApplicationCommandOptionInteger,
							Required:    true,
							MinValue:    &[]float64{1}[0],
							MaxValue:    12,
						},
						{
							Name:        "day",
							Description: "Day of birth (1-31)",
							Type:        discordgo.ApplicationCommandOptionInteger,
							Required:    true,
							MinValue:    &[]float64{1}[0],
							MaxValue:    31,
						},
					},
				},
				{
					Name:        "remove",
					Description: "Remove your birthday",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
				},
				{
					Name:        "list",
					Description: "List all birthdays in the server",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
				},
			},
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
				handleBirthdaySetup(s, i, database, subcommand.Options)
			case "set":
				handleBirthdaySet(s, i, database, subcommand.Options)
			case "remove":
				handleBirthdayRemove(s, i, database)
			case "list":
				handleBirthdayList(s, i, database)
			}
		},
	}
}

func handleBirthdaySetup(s *discordgo.Session, i *discordgo.InteractionCreate, database *db.DB, options []*discordgo.ApplicationCommandInteractionDataOption) {
	// Check permissions
	if i.Member.Permissions&discordgo.PermissionAdministrator == 0 {
		SendError(s, i, "You need Administrator permissions to use this command.")
		return
	}

	var channelID string
	for _, opt := range options {
		if opt.Name == "channel" {
			channelID = opt.ChannelValue(s).ID
		}
	}

	err := database.SetBirthdayChannel(context.Background(), i.GuildID, channelID)
	if err != nil {
		SendError(s, i, "Failed to configure birthday channel: "+err.Error())
		return
	}

	embed := &discordgo.MessageEmbed{
		Title:       "🎂 Birthday Channel Configured",
		Description: fmt.Sprintf("Birthday announcements will now be sent to <#%s>.", channelID),
		Color:       0xFFA500, // Orange
	}
	SendEmbed(s, i, embed)
}

func handleBirthdaySet(s *discordgo.Session, i *discordgo.InteractionCreate, database *db.DB, options []*discordgo.ApplicationCommandInteractionDataOption) {
	var month, day int
	for _, opt := range options {
		switch opt.Name {
		case "month":
			month = int(opt.IntValue())
		case "day":
			day = int(opt.IntValue())
		}
	}

	// Basic validation for days in month
	if day > 31 || (month == 2 && day > 29) || ((month == 4 || month == 6 || month == 9 || month == 11) && day > 30) {
		SendError(s, i, "Invalid date.")
		return
	}

	var userID string
	if i.Member != nil && i.Member.User != nil {
		userID = i.Member.User.ID
	} else if i.User != nil {
		userID = i.User.ID
	} else {
		SendError(s, i, "Could not determine user ID.")
		return
	}

	err := database.SetBirthday(context.Background(), i.GuildID, userID, month, day)
	if err != nil {
		SendError(s, i, "Failed to set birthday: "+err.Error())
		return
	}

	embed := &discordgo.MessageEmbed{
		Title:       "🎂 Birthday Set",
		Description: fmt.Sprintf("Your birthday has been set to %02d/%02d.", month, day),
		Color:       0x00FF00,
	}
	SendEmbed(s, i, embed)
}

func handleBirthdayRemove(s *discordgo.Session, i *discordgo.InteractionCreate, database *db.DB) {
	var userID string
	if i.Member != nil && i.Member.User != nil {
		userID = i.Member.User.ID
	} else if i.User != nil {
		userID = i.User.ID
	} else {
		SendError(s, i, "Could not determine user ID.")
		return
	}

	err := database.RemoveBirthday(context.Background(), i.GuildID, userID)
	if err != nil {
		SendError(s, i, "Failed to remove birthday: "+err.Error())
		return
	}

	embed := &discordgo.MessageEmbed{
		Title:       "🗑️ Birthday Removed",
		Description: "Your birthday has been removed.",
		Color:       0x00FF00,
	}
	SendEmbed(s, i, embed)
}

func handleBirthdayList(s *discordgo.Session, i *discordgo.InteractionCreate, database *db.DB) {
	birthdays, err := database.GetGuildBirthdays(context.Background(), i.GuildID)
	if err != nil {
		SendError(s, i, "Failed to fetch birthdays: "+err.Error())
		return
	}

	if len(birthdays) == 0 {
		SendEmbed(s, i, &discordgo.MessageEmbed{
			Title:       "🎂 Server Birthdays",
			Description: "No birthdays have been set in this server yet.",
			Color:       0xFFA500,
		})
		return
	}

	description := ""
	for _, bday := range birthdays {
		description += fmt.Sprintf("<@%s>: %02d/%02d\n", bday.UserID, bday.BirthMonth, bday.BirthDay)
	}

	SendEmbed(s, i, &discordgo.MessageEmbed{
		Title:       "🎂 Server Birthdays",
		Description: description,
		Color:       0x00FF00,
	})
}
