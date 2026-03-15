package commands

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"julescord/internal/db"
)

// Todo creates the /todo command and its subcommands.
func Todo(database *db.DB) *Command {
	return &Command{
		Definition: &discordgo.ApplicationCommand{
			Name:        "todo",
			Description: "Manage your personal to-do list",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        "add",
					Description: "Add a new task to your to-do list",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Name:        "task",
							Description: "The task you want to add",
							Type:        discordgo.ApplicationCommandOptionString,
							Required:    true,
						},
					},
				},
				{
					Name:        "list",
					Description: "View your to-do list",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
				},
				{
					Name:        "complete",
					Description: "Mark a task as completed",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Name:        "id",
							Description: "The ID of the task to complete",
							Type:        discordgo.ApplicationCommandOptionInteger,
							Required:    true,
						},
					},
				},
				{
					Name:        "remove",
					Description: "Remove a task from your to-do list",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Name:        "id",
							Description: "The ID of the task to remove",
							Type:        discordgo.ApplicationCommandOptionInteger,
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
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			var userID string
			if i.Member != nil {
				userID = i.Member.User.ID
			} else {
				userID = i.User.ID
			}

			switch subcommand.Name {
			case "add":
				task := subcommand.Options[0].StringValue()
				err := database.AddTodo(ctx, userID, task)
				if err != nil {
					SendError(s, i, "Failed to add task to your to-do list.")
					return
				}

				_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: "✅ Task added to your to-do list!",
						Flags:   discordgo.MessageFlagsEphemeral,
					},
				})

			case "list":
				todos, err := database.GetTodos(ctx, userID)
				if err != nil {
					SendError(s, i, "Failed to fetch your to-do list.")
					return
				}

				if len(todos) == 0 {
					_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
						Type: discordgo.InteractionResponseChannelMessageWithSource,
						Data: &discordgo.InteractionResponseData{
							Content: "📭 Your to-do list is empty! Add a task with `/todo add`.",
							Flags:   discordgo.MessageFlagsEphemeral,
						},
					})
					return
				}

				var content strings.Builder
				content.WriteString("📋 **Your To-Do List**\n\n")

				for _, todo := range todos {
					status := "⏳"
					if todo.Completed {
						status = "✅"
					}
					content.WriteString(fmt.Sprintf("`%d.` %s %s\n", todo.ID, status, todo.Content))
				}

				embed := &discordgo.MessageEmbed{
					Title:       "To-Do List",
					Description: content.String(),
					Color:       0x00B0F4, // Blue
				}

				_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Embeds: []*discordgo.MessageEmbed{embed},
						Flags:  discordgo.MessageFlagsEphemeral,
					},
				})

			case "complete":
				taskID := int(subcommand.Options[0].IntValue())
				err := database.CompleteTodo(ctx, userID, taskID)
				if err != nil {
					SendError(s, i, "Failed to mark task as completed.")
					return
				}

				_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: fmt.Sprintf("✅ Task `%d` marked as completed!", taskID),
						Flags:   discordgo.MessageFlagsEphemeral,
					},
				})

			case "remove":
				taskID := int(subcommand.Options[0].IntValue())
				err := database.RemoveTodo(ctx, userID, taskID)
				if err != nil {
					SendError(s, i, "Failed to remove task.")
					return
				}

				_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: fmt.Sprintf("🗑️ Task `%d` removed from your to-do list.", taskID),
						Flags:   discordgo.MessageFlagsEphemeral,
					},
				})
			}
		},
	}
}
