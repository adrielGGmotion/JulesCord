package commands

import (
	"context"
	"fmt"
	"julescord/internal/db"
	"time"

	"github.com/bwmarrin/discordgo"
)

func Timezone(database *db.DB) *Command {
	return &Command{
		Definition: &discordgo.ApplicationCommand{
			Name:        "timezone",
			Description: "Set and view user timezones",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        "set",
					Description: "Set your timezone (e.g., America/New_York or Europe/London)",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionString,
							Name:        "timezone",
							Description: "The timezone string (e.g., America/New_York)",
							Required:    true,
						},
					},
				},
				{
					Name:        "get",
					Description: "View the local time of a user",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionUser,
							Name:        "user",
							Description: "The user to check",
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

			subcommand := options[0].Name
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			switch subcommand {
			case "set":
				tzString := options[0].Options[0].StringValue()

				// Validate timezone string
				_, err := time.LoadLocation(tzString)
				if err != nil {
					_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
						Type: discordgo.InteractionResponseChannelMessageWithSource,
						Data: &discordgo.InteractionResponseData{
							Content: fmt.Sprintf("Invalid timezone: `%s`. Please use IANA format (e.g. `America/New_York`, `Europe/London`).\nYou can find a list here: https://en.wikipedia.org/wiki/List_of_tz_database_time_zones", tzString),
							Flags:   discordgo.MessageFlagsEphemeral,
						},
					})
					return
				}

				err = database.SetUserTimezone(ctx, i.Member.User.ID, tzString)
				if err != nil {
					SendError(s, i, "Failed to save timezone to database.")
					return
				}

				_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: fmt.Sprintf("Your timezone has been successfully set to `%s`.", tzString),
						Flags:   discordgo.MessageFlagsEphemeral,
					},
				})

			case "get":
				targetUser := options[0].Options[0].UserValue(s)

				tzString, err := database.GetUserTimezone(ctx, targetUser.ID)
				if err != nil {
					SendError(s, i, "Failed to fetch user's timezone from database.")
					return
				}

				if tzString == "" {
					_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
						Type: discordgo.InteractionResponseChannelMessageWithSource,
						Data: &discordgo.InteractionResponseData{
							Content: fmt.Sprintf("<@%s> has not set their timezone yet.", targetUser.ID),
						},
					})
					return
				}

				loc, err := time.LoadLocation(tzString)
				if err != nil {
					SendError(s, i, "User has an invalid timezone saved.")
					return
				}

				localTime := time.Now().In(loc)
				timeFormat := localTime.Format("3:04 PM (15:04) on Monday, January 2, 2006")

				_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: fmt.Sprintf("It is currently **%s** for <@%s> (`%s`).", timeFormat, targetUser.ID, tzString),
						AllowedMentions: &discordgo.MessageAllowedMentions{
							Users: []string{}, // Do not ping the user
						},
					},
				})
			}
		},
	}
}
