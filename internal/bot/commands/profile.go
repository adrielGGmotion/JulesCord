package commands

import (
	"context"
	"fmt"
	"julescord/internal/db"
	"log/slog"
	"regexp"
	"strconv"
	"strings"

	"github.com/bwmarrin/discordgo"
)

// NewProfileCommand creates the /profile command
func NewProfileCommand(database *db.DB) *Command {
	return &Command{
		Definition: &discordgo.ApplicationCommand{
			Name:        "profile",
			Description: "Manage and view user profiles",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        "view",
					Description: "View a user's profile",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionUser,
							Name:        "user",
							Description: "The user to view (defaults to yourself)",
							Required:    false,
						},
					},
				},
				{
					Name:        "set-bio",
					Description: "Set your profile bio",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionString,
							Name:        "text",
							Description: "Your new bio (max 300 characters)",
							Required:    true,
						},
					},
				},
				{
					Name:        "set-color",
					Description: "Set your profile embed color",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionString,
							Name:        "hex",
							Description: "Hex color code (e.g., #FF5733 or FF5733)",
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

			options := i.ApplicationCommandData().Options
			if len(options) == 0 {
				return
			}

			subcommand := options[0]

			switch subcommand.Name {
			case "view":
				handleProfileView(s, i, database, subcommand.Options)
			case "set-bio":
				handleProfileSetBio(s, i, database, subcommand.Options)
			case "set-color":
				handleProfileSetColor(s, i, database, subcommand.Options)
			}
		},
	}
}

func handleProfileView(s *discordgo.Session, i *discordgo.InteractionCreate, database *db.DB, options []*discordgo.ApplicationCommandInteractionDataOption) {
	var targetUser *discordgo.User
	if i.Member != nil && i.Member.User != nil {
		targetUser = i.Member.User
	} else if i.User != nil {
		targetUser = i.User
	}
	for _, opt := range options {
		if opt.Name == "user" {
			targetUser = opt.UserValue(s)
		}
	}

	ctx := context.Background()

	// 1. Fetch Profile (bio & color)
	profile, err := database.GetProfile(ctx, i.GuildID, targetUser.ID)
	if err != nil {
		slog.Error("Failed to fetch profile", "guild", i.GuildID, "user", targetUser.ID, "err", err)
		SendError(s, i, "Failed to fetch profile.")
		return
	}

	// 2. Fetch Economy (Level, XP, Coins)
	economy, err := database.GetUserEconomy(ctx, i.GuildID, targetUser.ID)
	if err != nil {
		slog.Error("Failed to fetch economy", "guild", i.GuildID, "user", targetUser.ID, "err", err)
		// We don't fail, we just show 0s
	}

	// 3. Fetch Reputation
	rep, err := database.GetReputation(ctx, i.GuildID, targetUser.ID)
	if err != nil {
		slog.Error("Failed to fetch reputation", "guild", i.GuildID, "user", targetUser.ID, "err", err)
		// We don't fail, we just show 0
	}

	// 4. Fetch Marriage Status
	marriage, err := database.GetMarriage(ctx, i.GuildID, targetUser.ID)
	if err != nil && err.Error() != "no rows in result set" {
		slog.Error("Failed to fetch marriage status", "guild", i.GuildID, "user", targetUser.ID, "err", err)
	}

	// 5. Fetch Badges
	badges, err := database.GetUserBadges(targetUser.ID)
	if err != nil {
		slog.Error("Failed to fetch badges", "user", targetUser.ID, "err", err)
	}

	// Format embed color
	embedColor := 0x5865F2 // Default Blurple
	if profile != nil && profile.Color != nil {
		hexStr := strings.TrimPrefix(*profile.Color, "#")
		if parsedColor, err := strconv.ParseInt(hexStr, 16, 64); err == nil {
			embedColor = int(parsedColor)
		}
	}

	// Format Bio
	bio := "No bio set. Use `/profile set-bio` to set one!"
	if profile != nil && profile.Bio != nil {
		bio = *profile.Bio
	}

	// Prepare economy defaults
	var level int
	var xp, coins int64
	if economy != nil {
		level = economy.Level
		xp = economy.XP
		coins = economy.Coins
	}

	// Format Marriage
	marriageStatus := "Single"
	if marriage != nil {
		partnerID := marriage.User2ID
		if targetUser.ID == marriage.User2ID {
			partnerID = marriage.User1ID
		}
		if marriage.Status == "accepted" {
			marriageStatus = fmt.Sprintf("Married to <@%s> 💖", partnerID)
		} else {
			marriageStatus = fmt.Sprintf("Pending proposal with <@%s> 💍", partnerID)
		}
	}

	// Format Badges
	badgeStr := "No badges yet."
	if len(badges) > 0 {
		var sb strings.Builder
		for _, b := range badges {
			sb.WriteString(fmt.Sprintf("%s ", b.Emoji))
		}
		badgeStr = sb.String()
	}

	// Build Embed fields
	fields := []*discordgo.MessageEmbedField{
		{
			Name:   "Economy",
			Value:  fmt.Sprintf("Level: **%d**\nXP: **%d**\nCoins: **%d**", level, xp, coins),
			Inline: true,
		},
		{
			Name:   "Reputation",
			Value:  fmt.Sprintf("Rep Points: **%d**", rep),
			Inline: true,
		},
		{
			Name:   "Marriage",
			Value:  marriageStatus,
			Inline: true,
		},
		{
			Name:   "Badges",
			Value:  badgeStr,
			Inline: false,
		},
	}

	embed := &discordgo.MessageEmbed{
		Title:       fmt.Sprintf("%s's Profile", targetUser.Username),
		Description: bio,
		Color:       embedColor,
		Thumbnail: &discordgo.MessageEmbedThumbnail{
			URL: targetUser.AvatarURL(""),
		},
		Fields: fields,
	}

	err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
		},
	})
	if err != nil {
		slog.Error("Failed to respond to profile view", "error", err)
	}
}

func handleProfileSetBio(s *discordgo.Session, i *discordgo.InteractionCreate, database *db.DB, options []*discordgo.ApplicationCommandInteractionDataOption) {
	var userID string
	if i.Member != nil && i.Member.User != nil {
		userID = i.Member.User.ID
	} else if i.User != nil {
		userID = i.User.ID
	}
	var bio string
	for _, opt := range options {
		if opt.Name == "text" {
			bio = opt.StringValue()
		}
	}

	if len(bio) > 300 {
		SendError(s, i, "Bio must be 300 characters or less.")
		return
	}

	err := database.SetProfileBio(context.Background(), i.GuildID, userID, bio)
	if err != nil {
		slog.Error("Failed to set bio", "err", err)
		SendError(s, i, "An error occurred while setting your bio.")
		return
	}

	SendEmbed(s, i, &discordgo.MessageEmbed{
		Title:       "Bio Updated",
		Description: "Your profile bio has been updated successfully.",
		Color:       0x00FF00,
	})
}

func handleProfileSetColor(s *discordgo.Session, i *discordgo.InteractionCreate, database *db.DB, options []*discordgo.ApplicationCommandInteractionDataOption) {
	var userID string
	if i.Member != nil && i.Member.User != nil {
		userID = i.Member.User.ID
	} else if i.User != nil {
		userID = i.User.ID
	}
	var hexColor string
	for _, opt := range options {
		if opt.Name == "hex" {
			hexColor = opt.StringValue()
		}
	}

	// Validate hex color
	hexColor = strings.TrimPrefix(hexColor, "#")
	match, _ := regexp.MatchString(`^[0-9A-Fa-f]{6}$`, hexColor)
	if !match {
		SendError(s, i, "Invalid hex color format. Please use a format like `#FF5733` or `FF5733`.")
		return
	}

	// Store with # prefix for consistency
	fullHexColor := "#" + strings.ToUpper(hexColor)

	err := database.SetProfileColor(context.Background(), i.GuildID, userID, fullHexColor)
	if err != nil {
		slog.Error("Failed to set profile color", "err", err)
		SendError(s, i, "An error occurred while setting your profile color.")
		return
	}

	embedColor, _ := strconv.ParseInt(hexColor, 16, 64)

	SendEmbed(s, i, &discordgo.MessageEmbed{
		Title:       "Color Updated",
		Description: fmt.Sprintf("Your profile embed color has been updated to **%s**.", fullHexColor),
		Color:       int(embedColor),
	})
}
