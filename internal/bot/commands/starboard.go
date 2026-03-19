package commands

import (
	"context"
	"fmt"

	"github.com/bwmarrin/discordgo"
	"julescord/internal/db"
)

// Starboard creates the /starboard command.
func Starboard(database *db.DB) *Command {
	if database == nil {
		return nil
	}

	return &Command{
		Definition: &discordgo.ApplicationCommand{
			Name:                     "starboard",
			Description:              "Configure the starboard system",
			DefaultMemberPermissions: func() *int64 { p := int64(discordgo.PermissionAdministrator); return &p }(),
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        "setup",
					Description: "Set the starboard channel and threshold",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionChannel,
							Name:        "channel",
							Description: "The channel to send starboard messages to",
							Required:    true,
							ChannelTypes: []discordgo.ChannelType{
								discordgo.ChannelTypeGuildText,
							},
						},
						{
							Type:        discordgo.ApplicationCommandOptionInteger,
							Name:        "min_stars",
							Description: "The minimum number of stars required to post on the starboard (default 3)",
							Required:    false,
							MinValue:    func() *float64 { v := float64(1); return &v }(),
						},
					},
				},
				{
					Name:        "multi",
					Description: "Configure multi-channel starboards (custom emoji based)",
					Type:        discordgo.ApplicationCommandOptionSubCommandGroup,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Name:        "add",
							Description: "Add a multi-channel starboard",
							Type:        discordgo.ApplicationCommandOptionSubCommand,
							Options: []*discordgo.ApplicationCommandOption{
								{
									Type:        discordgo.ApplicationCommandOptionChannel,
									Name:        "channel",
									Description: "The channel to post to",
									Required:    true,
									ChannelTypes: []discordgo.ChannelType{
										discordgo.ChannelTypeGuildText,
										discordgo.ChannelTypeGuildNews,
									},
								},
								{
									Type:        discordgo.ApplicationCommandOptionString,
									Name:        "emoji",
									Description: "The custom emoji or unicode to trigger this starboard",
									Required:    true,
								},
								{
									Type:        discordgo.ApplicationCommandOptionInteger,
									Name:        "min_stars",
									Description: "Minimum required (default 3)",
									Required:    false,
									MinValue:    func() *float64 { v := float64(1); return &v }(),
								},
							},
						},
						{
							Name:        "remove",
							Description: "Remove a multi-channel starboard",
							Type:        discordgo.ApplicationCommandOptionSubCommand,
							Options: []*discordgo.ApplicationCommandOption{
								{
									Type:        discordgo.ApplicationCommandOptionChannel,
									Name:        "channel",
									Description: "The channel mapped to this starboard",
									Required:    true,
								},
								{
									Type:        discordgo.ApplicationCommandOptionString,
									Name:        "emoji",
									Description: "The emoji triggering this starboard",
									Required:    true,
								},
							},
						},
						{
							Name:        "list",
							Description: "List multi-channel starboards",
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

			options := i.ApplicationCommandData().Options
			if len(options) == 0 {
				return
			}

			subcommandGroup := options[0].Name
			if subcommandGroup == "setup" {
				handleStarboardSetup(s, i, database, options[0].Options)
			} else if subcommandGroup == "multi" {
				subcommand := options[0].Options[0].Name
				opts := options[0].Options[0].Options

				if subcommand == "add" {
					handleMultiAdd(s, i, database, opts)
				} else if subcommand == "remove" {
					handleMultiRemove(s, i, database, opts)
				} else if subcommand == "list" {
					handleMultiList(s, i, database)
				}
			}
		},
	}
}

func handleMultiAdd(s *discordgo.Session, i *discordgo.InteractionCreate, database *db.DB, options []*discordgo.ApplicationCommandInteractionDataOption) {
	channelID := ""
	emoji := ""
	minStars := 3

	for _, opt := range options {
		if opt.Name == "channel" {
			channelID = opt.Value.(string)
		} else if opt.Name == "emoji" {
			emoji = opt.Value.(string)
		} else if opt.Name == "min_stars" {
			minStars = int(opt.Value.(float64))
		}
	}

	err := database.AddMultiStarboard(context.Background(), i.GuildID, channelID, emoji, minStars)
	if err != nil {
		SendError(s, i, "Failed to add multi-starboard rule: "+err.Error())
		return
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("✅ Multi-starboard added! Post to <#%s> with `%s` and a threshold of %d.", channelID, emoji, minStars),
		},
	})
}

func handleMultiRemove(s *discordgo.Session, i *discordgo.InteractionCreate, database *db.DB, options []*discordgo.ApplicationCommandInteractionDataOption) {
	channelID := ""
	emoji := ""

	for _, opt := range options {
		if opt.Name == "channel" {
			channelID = opt.Value.(string)
		} else if opt.Name == "emoji" {
			emoji = opt.Value.(string)
		}
	}

	err := database.RemoveMultiStarboard(context.Background(), i.GuildID, channelID, emoji)
	if err != nil {
		SendError(s, i, "Failed to remove multi-starboard rule: "+err.Error())
		return
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("✅ Multi-starboard removed for <#%s> with emoji `%s`.", channelID, emoji),
		},
	})
}

func handleMultiList(s *discordgo.Session, i *discordgo.InteractionCreate, database *db.DB) {
	list, err := database.GetMultiStarboards(context.Background(), i.GuildID)
	if err != nil || len(list) == 0 {
		SendError(s, i, "No multi-starboards configured.")
		return
	}

	desc := ""
	for _, l := range list {
		desc += fmt.Sprintf("• <#%s> - `%s` (Threshold: %d)\n", l.ChannelID, l.Emoji, l.Threshold)
	}

	embed := &discordgo.MessageEmbed{
		Title:       "Multi-Starboards",
		Description: desc,
		Color:       0xffff00,
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
		},
	})
}

func handleStarboardSetup(s *discordgo.Session, i *discordgo.InteractionCreate, database *db.DB, options []*discordgo.ApplicationCommandInteractionDataOption) {
	var channelID string
	minStars := 3 // Default

	for _, opt := range options {
		if opt.Name == "channel" {
			channelID = opt.ChannelValue(nil).ID
		} else if opt.Name == "min_stars" {
			minStars = int(opt.IntValue())
		}
	}

	err := database.SetStarboardConfig(context.Background(), i.GuildID, channelID, minStars)
	if err != nil {
		SendError(s, i, "Failed to save starboard configuration.")
		return
	}

	msg := fmt.Sprintf("✅ Starboard configured! Messages with **%d** or more ⭐ reactions will be posted in <#%s>.", minStars, channelID)
	err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: msg,
		},
	})
	if err != nil {
		// Log but do nothing
	}
}
