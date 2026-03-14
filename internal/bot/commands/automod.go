package commands

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"julescord/internal/db"

	"github.com/bwmarrin/discordgo"
)

// Automod creates the /automod command.
func Automod(database *db.DB) *Command {
	if database == nil {
		return nil
	}

	return &Command{
		Definition: &discordgo.ApplicationCommand{
			Name:                     "automod",
			Description:              "Configure the auto-moderation system",
			DefaultMemberPermissions: func() *int64 { p := int64(discordgo.PermissionManageGuild); return &p }(),
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        "setup",
					Description: "Set up auto-moderation rules",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionChannel,
							Name:        "log_channel",
							Description: "Channel where automod actions will be logged",
							Required:    true,
							ChannelTypes: []discordgo.ChannelType{
								discordgo.ChannelTypeGuildText,
							},
						},
						{
							Type:        discordgo.ApplicationCommandOptionBoolean,
							Name:        "filter_links",
							Description: "Delete messages containing HTTP links",
							Required:    true,
						},
						{
							Type:        discordgo.ApplicationCommandOptionBoolean,
							Name:        "filter_invites",
							Description: "Delete messages containing Discord invites",
							Required:    true,
						},
					},
				},
				{
					Name:        "word",
					Description: "Manage the bad word filter list",
					Type:        discordgo.ApplicationCommandOptionSubCommandGroup,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Name:        "add",
							Description: "Add a word to the bad word list",
							Type:        discordgo.ApplicationCommandOptionSubCommand,
							Options: []*discordgo.ApplicationCommandOption{
								{
									Type:        discordgo.ApplicationCommandOptionString,
									Name:        "word",
									Description: "The word to block",
									Required:    true,
								},
							},
						},
						{
							Name:        "remove",
							Description: "Remove a word from the bad word list",
							Type:        discordgo.ApplicationCommandOptionSubCommand,
							Options: []*discordgo.ApplicationCommandOption{
								{
									Type:        discordgo.ApplicationCommandOptionString,
									Name:        "word",
									Description: "The word to allow",
									Required:    true,
								},
							},
						},
						{
							Name:        "list",
							Description: "List all blocked words",
							Type:        discordgo.ApplicationCommandOptionSubCommand,
						},
					},
				},
			},
		},
		Handler: func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			if i.Type != discordgo.InteractionApplicationCommand {
				return
			}

			guildID := i.GuildID
			if guildID == "" {
				SendError(s, i, "This command can only be used in a server.")
				return
			}

			options := i.ApplicationCommandData().Options
			if len(options) == 0 {
				return
			}

			subcommand := options[0]

			switch subcommand.Name {
			case "setup":
				handleAutomodSetup(s, i, database, subcommand.Options)
			case "word":
				if len(subcommand.Options) == 0 {
					return
				}
				action := subcommand.Options[0]
				switch action.Name {
				case "add":
					handleAutomodWordAdd(s, i, database, action.Options)
				case "remove":
					handleAutomodWordRemove(s, i, database, action.Options)
				case "list":
					handleAutomodWordList(s, i, database)
				}
			}
		},
	}
}

func handleAutomodSetup(s *discordgo.Session, i *discordgo.InteractionCreate, database *db.DB, options []*discordgo.ApplicationCommandInteractionDataOption) {
	var logChannelID string
	var filterLinks, filterInvites bool

	for _, opt := range options {
		switch opt.Name {
		case "log_channel":
			logChannelID = opt.ChannelValue(nil).ID
		case "filter_links":
			filterLinks = opt.BoolValue()
		case "filter_invites":
			filterInvites = opt.BoolValue()
		}
	}

	err := database.SetAutomodConfig(context.Background(), i.GuildID, logChannelID, filterLinks, filterInvites)
	if err != nil {
		slog.Error("Failed to set automod config", "guild_id", i.GuildID, "error", err)
		SendError(s, i, "An error occurred while setting up the auto-moderation system.")
		return
	}

	embed := &discordgo.MessageEmbed{
		Title:       "Auto-Moderation Setup Complete",
		Description: "The auto-moderation system has been configured.",
		Color:       0x00FF00, // Green
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "Log Channel",
				Value:  fmt.Sprintf("<#%s>", logChannelID),
				Inline: true,
			},
			{
				Name:   "Filter Links",
				Value:  fmt.Sprintf("%t", filterLinks),
				Inline: true,
			},
			{
				Name:   "Filter Invites",
				Value:  fmt.Sprintf("%t", filterInvites),
				Inline: true,
			},
		},
		Timestamp: time.Now().Format(time.RFC3339),
	}

	SendEmbed(s, i, embed)
}

func handleAutomodWordAdd(s *discordgo.Session, i *discordgo.InteractionCreate, database *db.DB, options []*discordgo.ApplicationCommandInteractionDataOption) {
	if len(options) == 0 {
		return
	}
	word := strings.ToLower(options[0].StringValue())

	err := database.AddAutomodWord(context.Background(), i.GuildID, word)
	if err != nil {
		slog.Error("Failed to add automod word", "guild_id", i.GuildID, "word", word, "error", err)
		SendError(s, i, "An error occurred while adding the word to the filter list.")
		return
	}

	SendEmbed(s, i, &discordgo.MessageEmbed{
		Title:       "Word Added to Filter",
		Description: fmt.Sprintf("The word `%s` has been added to the bad word filter.", word),
		Color:       0x00FF00,
	})
}

func handleAutomodWordRemove(s *discordgo.Session, i *discordgo.InteractionCreate, database *db.DB, options []*discordgo.ApplicationCommandInteractionDataOption) {
	if len(options) == 0 {
		return
	}
	word := strings.ToLower(options[0].StringValue())

	err := database.RemoveAutomodWord(context.Background(), i.GuildID, word)
	if err != nil {
		slog.Error("Failed to remove automod word", "guild_id", i.GuildID, "word", word, "error", err)
		SendError(s, i, "An error occurred while removing the word from the filter list.")
		return
	}

	SendEmbed(s, i, &discordgo.MessageEmbed{
		Title:       "Word Removed from Filter",
		Description: fmt.Sprintf("The word `%s` has been removed from the bad word filter.", word),
		Color:       0x00FF00,
	})
}

func handleAutomodWordList(s *discordgo.Session, i *discordgo.InteractionCreate, database *db.DB) {
	words, err := database.GetAutomodWords(context.Background(), i.GuildID)
	if err != nil {
		slog.Error("Failed to get automod words", "guild_id", i.GuildID, "error", err)
		SendError(s, i, "An error occurred while fetching the bad word filter list.")
		return
	}

	if len(words) == 0 {
		SendEmbed(s, i, &discordgo.MessageEmbed{
			Title:       "Filtered Words",
			Description: "The bad word filter list is currently empty.",
			Color:       0x00FF00,
		})
		return
	}

	wordList := strings.Join(words, ", ")
	if len(wordList) > 2000 {
		wordList = wordList[:1997] + "..."
	}

	SendEmbed(s, i, &discordgo.MessageEmbed{
		Title:       "Filtered Words",
		Description: fmt.Sprintf("The following words are currently filtered:\n\n%s", wordList),
		Color:       0x00FF00,
	})
}
