package commands

import (
	"context"
	"fmt"
	"julescord/internal/db"
	"strconv"
	"time"

	"github.com/bwmarrin/discordgo"
)

// Multiplier creates the multiplier slash command to set global multipliers for a guild.
func Multiplier(database *db.DB) *Command {
	return &Command{
		Definition: &discordgo.ApplicationCommand{
			Name:                     "multiplier",
			Description:              "Manage the global economy multiplier for the server (Admin only)",
			DefaultMemberPermissions: func(p int64) *int64 { return &p }(discordgo.PermissionManageGuild),
			DMPermission:             func(b bool) *bool { return &b }(false),
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        "set",
					Description: "Set a global economy multiplier",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionNumber,
							Name:        "factor",
							Description: "The multiplier factor (e.g., 1.5, 2.0)",
							Required:    true,
							MinValue:    func(f float64) *float64 { return &f }(0.1),
							MaxValue:    100.0,
						},
						{
							Type:        discordgo.ApplicationCommandOptionString,
							Name:        "duration",
							Description: "Duration of the multiplier (e.g., '24h', '2d')",
							Required:    true,
						},
					},
				},
				{
					Name:        "view",
					Description: "View the current active multiplier",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
				},
			},
		},
		Handler: func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			options := i.ApplicationCommandData().Options
			subcommand := options[0]

			switch subcommand.Name {
			case "set":
				handleMultiplierSet(s, i, database, subcommand.Options)
			case "view":
				handleMultiplierView(s, i, database)
			}
		},
	}
}

func handleMultiplierSet(s *discordgo.Session, i *discordgo.InteractionCreate, database *db.DB, options []*discordgo.ApplicationCommandInteractionDataOption) {
	var factor float64
	var durationStr string

	for _, opt := range options {
		switch opt.Name {
		case "factor":
			factor = opt.FloatValue()
		case "duration":
			durationStr = opt.StringValue()
		}
	}

	// Basic duration parsing (handle "d" manually, let time.ParseDuration handle the rest)
	var parsedDuration time.Duration
	var err error

	if len(durationStr) > 1 && durationStr[len(durationStr)-1] == 'd' {
		daysStr := durationStr[:len(durationStr)-1]
		days, pErr := strconv.Atoi(daysStr)
		if pErr != nil {
			SendError(s, i, "Invalid duration format. Use e.g., '2d' or '24h'.")
			return
		}
		parsedDuration = time.Duration(days) * 24 * time.Hour
	} else {
		parsedDuration, err = time.ParseDuration(durationStr)
		if err != nil {
			SendError(s, i, "Invalid duration format. Use e.g., '24h' or '30m'.")
			return
		}
	}

	if parsedDuration <= 0 {
		SendError(s, i, "Duration must be positive.")
		return
	}

	err = database.SetGlobalMultiplier(context.Background(), i.GuildID, factor, parsedDuration)
	if err != nil {
		SendError(s, i, "Failed to set global multiplier.")
		return
	}

	expiresAt := time.Now().Add(parsedDuration).Unix()

	embed := &discordgo.MessageEmbed{
		Title:       "🎉 Global Multiplier Activated!",
		Description: fmt.Sprintf("A **%.2fx** economy multiplier is now active!\n\nExpires: <t:%d:R>", factor, expiresAt),
		Color:       0x00FF00,
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
		},
	})
}

func handleMultiplierView(s *discordgo.Session, i *discordgo.InteractionCreate, database *db.DB) {
	factor, err := database.GetActiveMultiplier(context.Background(), i.GuildID)
	if err != nil {
		SendError(s, i, "Failed to retrieve the active multiplier.")
		return
	}

	if factor <= 1.0 {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "No global multiplier is currently active.",
			},
		})
		return
	}

	embed := &discordgo.MessageEmbed{
		Title:       "Global Multiplier",
		Description: fmt.Sprintf("A **%.2fx** economy multiplier is currently active!", factor),
		Color:       0x00FF00,
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
		},
	})
}
