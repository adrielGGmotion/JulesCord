package commands

import (
	"context"
	"fmt"
	"julescord/internal/db"
	"log/slog"
	"strconv"

	"github.com/bwmarrin/discordgo"
)

// NewMarryCommand creates the /marry command
func NewMarryCommand(database *db.DB) *Command {
	return &Command{
		Definition: &discordgo.ApplicationCommand{
			Name:        "marry",
			Description: "Marriage system commands",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        "propose",
					Description: "Propose marriage to someone",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionUser,
							Name:        "user",
							Description: "The user to propose to",
							Required:    true,
						},
					},
				},
				{
					Name:        "accept",
					Description: "Accept a marriage proposal",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionUser,
							Name:        "user",
							Description: "The user who proposed to you",
							Required:    true,
						},
					},
				},
				{
					Name:        "divorce",
					Description: "Divorce your current partner or cancel a pending proposal",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
				},
				{
					Name:        "joint-bank",
					Description: "Enable or disable your joint bank account",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionBoolean,
							Name:        "enable",
							Description: "True to enable, False to disable",
							Required:    true,
						},
					},
				},
				{
					Name:        "deposit",
					Description: "Deposit coins into your joint bank",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionString,
							Name:        "amount",
							Description: "Amount to deposit (or 'all')",
							Required:    true,
						},
					},
				},
				{
					Name:        "withdraw",
					Description: "Withdraw coins from your joint bank",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionString,
							Name:        "amount",
							Description: "Amount to withdraw (or 'all')",
							Required:    true,
						},
					},
				},
			},
		},
		Handler: func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			if database == nil {
				SendError(s, i, "Database connection not available")
				return
			}

			if i.Member == nil || i.Member.User == nil {
				SendError(s, i, "This command can only be used in a server.")
				return
			}

			options := i.ApplicationCommandData().Options
			if len(options) == 0 {
				return
			}

			subcommand := options[0]

			switch subcommand.Name {
			case "propose":
				handleMarryPropose(s, i, database, subcommand.Options)
			case "accept":
				handleMarryAccept(s, i, database, subcommand.Options)
			case "divorce":
				handleMarryDivorce(s, i, database)
			case "joint-bank":
				handleMarryJointBank(s, i, database, subcommand.Options)
			case "deposit":
				handleMarryDeposit(s, i, database, subcommand.Options)
			case "withdraw":
				handleMarryWithdraw(s, i, database, subcommand.Options)
			}
		},
	}
}

func handleMarryPropose(s *discordgo.Session, i *discordgo.InteractionCreate, database *db.DB, options []*discordgo.ApplicationCommandInteractionDataOption) {
	proposerID := i.Member.User.ID
	var proposee *discordgo.User

	for _, opt := range options {
		if opt.Name == "user" {
			proposee = opt.UserValue(s)
		}
	}

	if proposee == nil {
		SendError(s, i, "User not found.")
		return
	}

	if proposee.ID == proposerID {
		SendError(s, i, "You cannot marry yourself.")
		return
	}

	if proposee.Bot {
		SendError(s, i, "You cannot marry a bot.")
		return
	}

	err := database.ProposeMarriage(context.Background(), i.GuildID, proposerID, proposee.ID)
	if err != nil {
		slog.Error("Failed to propose marriage", "err", err)
		SendError(s, i, err.Error())
		return
	}

	SendEmbed(s, i, &discordgo.MessageEmbed{
		Title:       "Marriage Proposal",
		Description: fmt.Sprintf("<@%s>, you have received a marriage proposal from <@%s>! Use `/marry accept user:@%s` to accept.", proposee.ID, proposerID, i.Member.User.Username),
		Color:       0xFF69B4, // Hot Pink
	})
}

func handleMarryAccept(s *discordgo.Session, i *discordgo.InteractionCreate, database *db.DB, options []*discordgo.ApplicationCommandInteractionDataOption) {
	proposeeID := i.Member.User.ID
	var proposer *discordgo.User

	for _, opt := range options {
		if opt.Name == "user" {
			proposer = opt.UserValue(s)
		}
	}

	if proposer == nil {
		SendError(s, i, "User not found.")
		return
	}

	err := database.AcceptMarriage(context.Background(), i.GuildID, proposeeID, proposer.ID)
	if err != nil {
		slog.Error("Failed to accept marriage", "err", err)
		SendError(s, i, err.Error())
		return
	}

	SendEmbed(s, i, &discordgo.MessageEmbed{
		Title:       "Married!",
		Description: fmt.Sprintf("Congratulations! <@%s> and <@%s> are now married! 🎉💖", proposer.ID, proposeeID),
		Color:       0xFF69B4, // Hot Pink
	})
}

func handleMarryDivorce(s *discordgo.Session, i *discordgo.InteractionCreate, database *db.DB) {
	userID := i.Member.User.ID

	err := database.Divorce(context.Background(), i.GuildID, userID)
	if err != nil {
		slog.Error("Failed to divorce", "err", err)
		SendError(s, i, err.Error())
		return
	}

	SendEmbed(s, i, &discordgo.MessageEmbed{
		Title:       "Divorced",
		Description: fmt.Sprintf("<@%s> is no longer married and has cancelled any pending proposals. 💔", userID),
		Color:       0x000000, // Black
	})
}

func handleMarryJointBank(s *discordgo.Session, i *discordgo.InteractionCreate, database *db.DB, options []*discordgo.ApplicationCommandInteractionDataOption) {
	userID := i.Member.User.ID
	enable := options[0].BoolValue()

	err := database.SetJointBank(context.Background(), i.GuildID, userID, enable)
	if err != nil {
		slog.Error("Failed to set joint bank", "err", err)
		SendError(s, i, err.Error())
		return
	}

	status := "disabled"
	if enable {
		status = "enabled"
	}

	SendEmbed(s, i, &discordgo.MessageEmbed{
		Title:       "Joint Bank Updated",
		Description: fmt.Sprintf("Your joint bank account has been **%s**.", status),
		Color:       0x2ecc71, // Green
	})
}

func handleMarryDeposit(s *discordgo.Session, i *discordgo.InteractionCreate, database *db.DB, options []*discordgo.ApplicationCommandInteractionDataOption) {
	userID := i.Member.User.ID
	amountStr := options[0].StringValue()

	econ, err := database.GetUserEconomy(context.Background(), i.GuildID, userID)
	if err != nil || econ == nil {
		SendError(s, i, "Failed to get your economy data.")
		return
	}

	var depositAmount int64
	if amountStr == "all" {
		depositAmount = econ.Coins
	} else {
		parsed, err := strconv.ParseInt(amountStr, 10, 64)
		if err != nil || parsed <= 0 {
			SendError(s, i, "Please provide a valid positive number or 'all'.")
			return
		}
		depositAmount = parsed
	}

	if depositAmount <= 0 {
		SendError(s, i, "You don't have any coins to deposit!")
		return
	}

	err = database.DepositJoint(context.Background(), i.GuildID, userID, depositAmount)
	if err != nil {
		slog.Error("Failed to deposit to joint bank", "err", err)
		SendError(s, i, err.Error())
		return
	}

	SendEmbed(s, i, &discordgo.MessageEmbed{
		Title:       "Joint Bank Deposit",
		Description: fmt.Sprintf("Successfully deposited **%d** coins into your joint bank.", depositAmount),
		Color:       0x2ecc71,
	})
}

func handleMarryWithdraw(s *discordgo.Session, i *discordgo.InteractionCreate, database *db.DB, options []*discordgo.ApplicationCommandInteractionDataOption) {
	userID := i.Member.User.ID
	amountStr := options[0].StringValue()

	jointBalance, err := database.GetJointBalance(context.Background(), i.GuildID, userID)
	if err != nil {
		SendError(s, i, "Failed to get joint balance. Are you married with a joint bank enabled?")
		return
	}

	var withdrawAmount int64
	if amountStr == "all" {
		withdrawAmount = jointBalance
	} else {
		parsed, err := strconv.ParseInt(amountStr, 10, 64)
		if err != nil || parsed <= 0 {
			SendError(s, i, "Please provide a valid positive number or 'all'.")
			return
		}
		withdrawAmount = parsed
	}

	if withdrawAmount <= 0 {
		SendError(s, i, "There are no coins to withdraw!")
		return
	}

	err = database.WithdrawJoint(context.Background(), i.GuildID, userID, withdrawAmount)
	if err != nil {
		slog.Error("Failed to withdraw from joint bank", "err", err)
		SendError(s, i, err.Error())
		return
	}

	SendEmbed(s, i, &discordgo.MessageEmbed{
		Title:       "Joint Bank Withdrawal",
		Description: fmt.Sprintf("Successfully withdrew **%d** coins from your joint bank.", withdrawAmount),
		Color:       0x2ecc71,
	})
}
