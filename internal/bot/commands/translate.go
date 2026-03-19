package commands

import (
	"context"
	"fmt"
	"julescord/internal/db"

	"github.com/bwmarrin/discordgo"
)

// Translate returns the definition for the translate command.
func Translate(database *db.DB) *Command {
	return &Command{
		Definition: &discordgo.ApplicationCommand{
			Name:        "translate",
			Description: "Translate text or configure translation settings",
			DefaultMemberPermissions: func(p int64) *int64 { return &p }(discordgo.PermissionManageGuild),
			DMPermission: func(b bool) *bool { return &b }(false),
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        "text",
					Description: "Translate a piece of text",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionString,
							Name:        "text",
							Description: "The text to translate",
							Required:    true,
						},
						{
							Type:        discordgo.ApplicationCommandOptionString,
							Name:        "target",
							Description: "Target language (e.g. es, fr, en)",
							Required:    false,
						},
						{
							Type:        discordgo.ApplicationCommandOptionString,
							Name:        "source",
							Description: "Source language (optional)",
							Required:    false,
						},
					},
				},
				{
					Name:        "set-default",
					Description: "Set the default translation target language for this server",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionString,
							Name:        "language",
							Description: "The default language (e.g. en, es, fr)",
							Required:    true,
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
			case "set-default":
				handleSetDefaultLanguage(s, i, database, subcommand.Options)
			case "text":
				handleTranslateText(s, i, database, subcommand.Options)
			}
		},
	}
}

func handleSetDefaultLanguage(s *discordgo.Session, i *discordgo.InteractionCreate, database *db.DB, options []*discordgo.ApplicationCommandInteractionDataOption) {
	perm, err := s.UserChannelPermissions(i.Member.User.ID, i.ChannelID)
	if err != nil || perm&discordgo.PermissionManageGuild == 0 {
		SendError(s, i, "You need the Manage Server permission to use this command.")
		return
	}

	language := options[0].StringValue()

	err = database.SetTranslationConfig(context.Background(), i.GuildID, language)
	if err != nil {
		SendError(s, i, "Failed to save default translation language.")
		return
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("✅ Default translation language set to `%s`.", language),
		},
	})
}

func handleTranslateText(s *discordgo.Session, i *discordgo.InteractionCreate, database *db.DB, options []*discordgo.ApplicationCommandInteractionDataOption) {
	var text, target, source string

	for _, opt := range options {
		switch opt.Name {
		case "text":
			text = opt.StringValue()
		case "target":
			target = opt.StringValue()
		case "source":
			source = opt.StringValue()
		}
	}

	if target == "" {
		// Fetch default language from config
		defaultLang, err := database.GetTranslationConfig(context.Background(), i.GuildID)
		if err == nil && defaultLang != "" {
			target = defaultLang
		} else {
			target = "en"
		}
	}

	sourceStr := source
	if sourceStr == "" {
		sourceStr = "auto"
	}

	// Mock translation output
	translatedText := fmt.Sprintf("[Translated from %s to %s]: %s", sourceStr, target, text)

	embed := &discordgo.MessageEmbed{
		Title:       "Translation",
		Description: translatedText,
		Color:       0x1e90ff,
		Footer: &discordgo.MessageEmbedFooter{
			Text: fmt.Sprintf("Target Language: %s", target),
		},
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
		},
	})
}
