package commands

import (
	"context"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"julescord/internal/db"
)

// Heist creates the /heist command.
func Heist(database *db.DB) *Command {
	if database == nil {
		return nil
	}

	return &Command{
		Definition: &discordgo.ApplicationCommand{
			Name:        "heist",
			Description: "Start or join a heist on another user's bank",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        "start",
					Description: "Start a new heist",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionUser,
							Name:        "target",
							Description: "The user you want to heist",
							Required:    true,
						},
					},
				},
				{
					Name:        "join",
					Description: "Join an active heist",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionUser,
							Name:        "target",
							Description: "The user whose heist you want to join",
							Required:    true,
						},
					},
				},
			},
		},
		Handler: func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			if i.Type != discordgo.InteractionApplicationCommand {
				return
			}

			options := i.ApplicationCommandData().Options
			if len(options) == 0 {
				return
			}

			subcommand := options[0].Name
			targetUser := options[0].Options[0].UserValue(s)

			if targetUser.ID == i.Member.User.ID {
				SendError(s, i, "You cannot heist yourself!")
				return
			}
			if targetUser.Bot {
				SendError(s, i, "You cannot heist a bot!")
				return
			}

			ctx := context.Background()

			// Check if target has economy record
			targetEcon, err := database.GetUserEconomy(ctx, i.GuildID, targetUser.ID)
			if err != nil || targetEcon == nil || targetEcon.Bank <= 0 {
				SendError(s, i, "This user has no coins in their bank to heist!")
				return
			}

			// Check if initiator has minimum balance (e.g. 1000 coins to start/join)
			initiatorEcon, err := database.GetUserEconomy(ctx, i.GuildID, i.Member.User.ID)
			if err != nil || initiatorEcon == nil || initiatorEcon.Coins < 1000 {
				SendError(s, i, "You need at least 1,000 coins in your wallet to participate in a heist!")
				return
			}

			if subcommand == "start" {
				// Create the heist
				err = database.CreateHeist(ctx, i.GuildID, targetUser.ID, i.Member.User.ID)
				if err != nil {
					SendError(s, i, "A heist is already active for this target, or an error occurred.")
					return
				}

				embed := &discordgo.MessageEmbed{
					Title:       "🚨 Heist Started! 🚨",
					Description: fmt.Sprintf("**%s** is organizing a heist on **%s**'s bank!\n\nUse `/heist join target:@%s` to join the crew.\nThe heist will execute in 1 minute!", i.Member.User.Username, targetUser.Username, targetUser.Username),
					Color:       0xff0000,
				}
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Embeds: []*discordgo.MessageEmbed{embed},
					},
				})

			} else if subcommand == "join" {
				// Join the heist
				err = database.JoinHeist(ctx, i.GuildID, targetUser.ID, i.Member.User.ID)
				if err != nil {
					SendError(s, i, "Could not join the heist. It might not exist, already started, or you are already in the crew.")
					return
				}

				embed := &discordgo.MessageEmbed{
					Title:       "🔫 Crew Member Joined!",
					Description: fmt.Sprintf("**%s** has joined the heist on **%s**!", i.Member.User.Username, targetUser.Username),
					Color:       0xffa500,
				}
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Embeds: []*discordgo.MessageEmbed{embed},
					},
				})
			}
		},
	}
}
