package commands

import (
	"context"
	"fmt"
	"julescord/internal/db"

	"github.com/bwmarrin/discordgo"
)

// GlobalEco returns the globaleco command
func GlobalEco(database *db.DB) *Command {
	return &Command{
		Definition: &discordgo.ApplicationCommand{
			Name:        "globaleco",
			Description: "Manage your cross-server global economy balance",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "balance",
					Description: "Check your global coin balance",
				},
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "transfer",
					Description: "Transfer global coins to another user",
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionUser,
							Name:        "user",
							Description: "The user to transfer global coins to",
							Required:    true,
						},
						{
							Type:        discordgo.ApplicationCommandOptionInteger,
							Name:        "amount",
							Description: "The amount of global coins to transfer",
							Required:    true,
							MinValue:    &[]float64{1}[0],
						},
					},
				},
			},
		},
		Handler: func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			options := i.ApplicationCommandData().Options
			subcommand := options[0].Name

			switch subcommand {
			case "balance":
				handleGlobalEcoBalance(s, i, database)
			case "transfer":
				handleGlobalEcoTransfer(s, i, database, options[0].Options)
			}
		},
	}
}

func handleGlobalEcoBalance(s *discordgo.Session, i *discordgo.InteractionCreate, database *db.DB) {
	ctx := context.Background()
	var userID string
	var user *discordgo.User

	if i.Member != nil {
		userID = i.Member.User.ID
		user = i.Member.User
	} else {
		userID = i.User.ID
		user = i.User
	}

	balance, err := database.GetGlobalCoins(ctx, userID)
	if err != nil {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "Error retrieving global balance.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	embed := &discordgo.MessageEmbed{
		Title: "🌍 Global Economy Balance",
		Description: fmt.Sprintf("**%s**, your cross-server global balance is **%d** coins.", user.Username, balance),
		Color: 0x00FF00, // Green
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
		},
	})
}

func handleGlobalEcoTransfer(s *discordgo.Session, i *discordgo.InteractionCreate, database *db.DB, options []*discordgo.ApplicationCommandInteractionDataOption) {
	ctx := context.Background()

	var senderID string
	if i.Member != nil {
		senderID = i.Member.User.ID
	} else {
		senderID = i.User.ID
	}

	targetUser := options[0].UserValue(s)
	amount := int64(options[1].IntValue())

	if targetUser.Bot {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "You cannot transfer global coins to a bot.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	if senderID == targetUser.ID {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "You cannot transfer global coins to yourself.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	err := database.TransferGlobalCoins(ctx, senderID, targetUser.ID, amount)
	if err != nil {
		msg := "An error occurred during the transfer."
		if err.Error() == "insufficient funds" {
			msg = "You do not have enough global coins for this transfer."
		}

		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: msg,
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	embed := &discordgo.MessageEmbed{
		Title: "🌍 Global Transfer Successful",
		Description: fmt.Sprintf("Successfully transferred **%d** global coins to **%s**.", amount, targetUser.Username),
		Color: 0x00FF00, // Green
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
		},
	})
}
