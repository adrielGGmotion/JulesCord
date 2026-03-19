package commands

import (
	"context"
	"fmt"
	"strings"

	"julescord/internal/db"

	"github.com/bwmarrin/discordgo"
)

// Stock returns the /stock slash command
func Stock(database *db.DB) *Command {
	return &Command{
		Definition: &discordgo.ApplicationCommand{
			Name:        "stock",
			Description: "Economy stock market system",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        "market",
					Description: "View available stocks and their prices",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
				},
				{
					Name:        "portfolio",
					Description: "View your current stock holdings",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
				},
				{
					Name:        "buy",
					Description: "Buy shares of a stock",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionString,
							Name:        "symbol",
							Description: "The stock symbol",
							Required:    true,
						},
						{
							Type:        discordgo.ApplicationCommandOptionInteger,
							Name:        "quantity",
							Description: "Number of shares to buy",
							Required:    true,
							MinValue:    func() *float64 { v := 1.0; return &v }(),
						},
					},
				},
				{
					Name:        "sell",
					Description: "Sell shares of a stock",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionString,
							Name:        "symbol",
							Description: "The stock symbol",
							Required:    true,
						},
						{
							Type:        discordgo.ApplicationCommandOptionInteger,
							Name:        "quantity",
							Description: "Number of shares to sell",
							Required:    true,
							MinValue:    func() *float64 { v := 1.0; return &v }(),
						},
					},
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
			case "market":
				handleStockMarket(s, i, database)
			case "portfolio":
				handleStockPortfolio(s, i, database)
			case "buy":
				handleStockBuy(s, i, database, subcommand.Options)
			case "sell":
				handleStockSell(s, i, database, subcommand.Options)
			}
		},
	}
}

func handleStockMarket(s *discordgo.Session, i *discordgo.InteractionCreate, database *db.DB) {
	ctx := context.Background()
	stocks, err := database.GetStocks(ctx)
	if err != nil {
		SendError(s, i, "Failed to fetch stock market data.")
		return
	}

	if len(stocks) == 0 {
		SendError(s, i, "The stock market is currently empty.")
		return
	}

	description := ""
	for _, stock := range stocks {
		trend := "📈"
		if len(stock.History) >= 2 {
			if stock.CurrentPrice < stock.History[len(stock.History)-2] {
				trend = "📉"
			}
		}
		description += fmt.Sprintf("**%s** — %d coins %s\n", stock.Symbol, stock.CurrentPrice, trend)
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{
				{
					Title:       "📈 Stock Market",
					Description: description,
					Color:       0x00FF00,
				},
			},
		},
	})
}

func handleStockPortfolio(s *discordgo.Session, i *discordgo.InteractionCreate, database *db.DB) {
	ctx := context.Background()
	guildID := i.GuildID
	userID := i.Member.User.ID

	holdings, err := database.GetUserStocks(ctx, guildID, userID)
	if err != nil {
		SendError(s, i, "Failed to fetch your portfolio.")
		return
	}

	if len(holdings) == 0 {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Embeds: []*discordgo.MessageEmbed{
					{
						Title:       "📊 Your Portfolio",
						Description: "You do not own any stocks.",
						Color:       0x00FF00,
					},
				},
			},
		})
		return
	}

	description := ""
	totalValue := 0

	for _, holding := range holdings {
		stock, err := database.GetStock(ctx, holding.Symbol)
		if err != nil || stock == nil {
			continue
		}

		currentValue := stock.CurrentPrice * holding.Quantity
		totalValue += currentValue

		profit := currentValue - int(holding.AverageBuyPrice*float64(holding.Quantity))
		profitStr := fmt.Sprintf("+%d", profit)
		if profit < 0 {
			profitStr = fmt.Sprintf("%d", profit)
		}

		description += fmt.Sprintf("**%s**: %d shares\n", holding.Symbol, holding.Quantity)
		description += fmt.Sprintf("└ Value: %d coins (Avg Buy: %.2f) [%s]\n\n", currentValue, holding.AverageBuyPrice, profitStr)
	}

	description += fmt.Sprintf("**Total Portfolio Value:** %d coins", totalValue)

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{
				{
					Title:       "📊 Your Portfolio",
					Description: description,
					Color:       0x00FF00,
				},
			},
		},
	})
}

func handleStockBuy(s *discordgo.Session, i *discordgo.InteractionCreate, database *db.DB, options []*discordgo.ApplicationCommandInteractionDataOption) {
	ctx := context.Background()
	guildID := i.GuildID
	userID := i.Member.User.ID

	var symbol string
	var quantity int

	for _, opt := range options {
		switch opt.Name {
		case "symbol":
			symbol = opt.StringValue()
		case "quantity":
			quantity = int(opt.IntValue())
		}
	}

	err := database.BuyStock(ctx, guildID, userID, symbol, quantity)
	if err != nil {
		SendError(s, i, err.Error())
		return
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("✅ Successfully purchased %d shares of **%s**.", quantity, strings.ToUpper(symbol)),
		},
	})
}

func handleStockSell(s *discordgo.Session, i *discordgo.InteractionCreate, database *db.DB, options []*discordgo.ApplicationCommandInteractionDataOption) {
	ctx := context.Background()
	guildID := i.GuildID
	userID := i.Member.User.ID

	var symbol string
	var quantity int

	for _, opt := range options {
		switch opt.Name {
		case "symbol":
			symbol = opt.StringValue()
		case "quantity":
			quantity = int(opt.IntValue())
		}
	}

	err := database.SellStock(ctx, guildID, userID, symbol, quantity)
	if err != nil {
		SendError(s, i, err.Error())
		return
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("✅ Successfully sold %d shares of **%s**.", quantity, strings.ToUpper(symbol)),
		},
	})
}
