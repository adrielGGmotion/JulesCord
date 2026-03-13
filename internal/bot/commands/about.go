package commands

import (
	"github.com/bwmarrin/discordgo"
)

// About creates the /about command.
func About() *Command {
	return &Command{
		Definition: &discordgo.ApplicationCommand{
			Name:        "about",
			Description: "Learn about JulesCord and its autonomous build loop",
		},
		Handler: func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			embed := &discordgo.MessageEmbed{
				Title:       "About JulesCord 🤖",
				Description: "JulesCord is a production-grade, complex Discord bot written in Go.\n\n" +
					"**I am built entirely by [Jules](https://jules.google.com) — Google's autonomous coding agent.**\n" +
					"Zero human code contributions.\n\n" +
					"Every 15 minutes, Jules reads my state file, picks the next task, implements it, and opens a PR. " +
					"The PR gets auto-merged. I am improving forever in a continuous loop.",
				Color: 0x0099ff,
				Fields: []*discordgo.MessageEmbedField{
					{
						Name:  "Tech Stack",
						Value: "Go, DiscordGo, Gin, PostgreSQL (pgx/v5), React + Tailwind Web Dashboard",
						Inline: false,
					},
					{
						Name:  "Status",
						Value: "Phase 2 — Database & Core Features",
						Inline: false,
					},
				},
				Footer: &discordgo.MessageEmbedFooter{
					Text: "Built by Jules • jules.google.com",
				},
			}

			_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Embeds: []*discordgo.MessageEmbed{embed},
				},
			})
		},
	}
}
