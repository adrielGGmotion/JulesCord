package commands

import (
	"context"
	"fmt"
	"log/slog"

	"julescord/internal/db"

	"github.com/bwmarrin/discordgo"
)

// Job returns the /job command definition and handler.
func Job(database *db.DB) *Command {
	return &Command{
		Definition: &discordgo.ApplicationCommand{
			Name:        "job",
			Description: "Manage your jobs and available jobs in the server.",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        "list",
					Description: "List all available jobs in the server.",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
				},
				{
					Name:        "apply",
					Description: "Apply for a job.",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionInteger,
							Name:        "job_id",
							Description: "The ID of the job you want to apply for.",
							Required:    true,
						},
					},
				},
				{
					Name:        "quit",
					Description: "Quit your current job.",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
				},
				{
					Name:        "info",
					Description: "View information about your current job.",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
				},
				{
					Name:        "create",
					Description: "Create a new job for the server (Admin only).",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionString,
							Name:        "name",
							Description: "The name of the job.",
							Required:    true,
						},
						{
							Type:        discordgo.ApplicationCommandOptionString,
							Name:        "description",
							Description: "A short description of the job.",
							Required:    true,
						},
						{
							Type:        discordgo.ApplicationCommandOptionInteger,
							Name:        "salary",
							Description: "The salary earned when working this job.",
							Required:    true,
						},
						{
							Type:        discordgo.ApplicationCommandOptionInteger,
							Name:        "required_level",
							Description: "The level required to apply for this job.",
							Required:    true,
						},
					},
				},
			},
		},
		Handler: func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			if i.GuildID == "" {
				SendError(s, i, "This command can only be used in a server.")
				return
			}

			if database == nil {
				SendError(s, i, "Database is not connected.")
				return
			}

			var user *discordgo.User
			if i.Member != nil {
				user = i.Member.User
			} else {
				user = i.User
			}

			if user == nil {
				return
			}

			options := i.ApplicationCommandData().Options
			if len(options) == 0 {
				return
			}

			subcommand := options[0].Name
			ctx := context.Background()

			switch subcommand {
			case "list":
				jobs, err := database.GetJobs(ctx, i.GuildID)
				if err != nil {
					slog.Error("Failed to fetch jobs", "error", err)
					SendError(s, i, "Failed to fetch jobs.")
					return
				}

				if len(jobs) == 0 {
					s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
						Type: discordgo.InteractionResponseChannelMessageWithSource,
						Data: &discordgo.InteractionResponseData{
							Embeds: []*discordgo.MessageEmbed{
								{
									Title:       "Available Jobs",
									Description: "There are no jobs available in this server.",
									Color:       0x3498db,
								},
							},
						},
					})
					return
				}

				var description string
				for _, j := range jobs {
					description += fmt.Sprintf("**ID: %d | %s**\n> %s\n> Salary: **%d coins** | Required Level: **%d**\n\n", j.ID, j.Name, j.Description, j.Salary, j.RequiredLevel)
				}

				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Embeds: []*discordgo.MessageEmbed{
							{
								Title:       "Available Jobs",
								Description: description,
								Color:       0x3498db,
							},
						},
					},
				})

			case "apply":
				subOptions := options[0].Options
				var jobID int
				for _, opt := range subOptions {
					if opt.Name == "job_id" {
						jobID = int(opt.IntValue())
					}
				}

				job, err := database.GetJob(ctx, jobID)
				if err != nil || job.GuildID != i.GuildID {
					SendError(s, i, "Invalid job ID.")
					return
				}

				eco, err := database.GetUserEconomy(ctx, i.GuildID, user.ID)
				if err != nil {
					SendError(s, i, "Failed to retrieve your economy data.")
					return
				}

				if eco.Level < job.RequiredLevel {
					SendError(s, i, fmt.Sprintf("You need to be level **%d** to apply for this job. You are currently level **%d**.", job.RequiredLevel, eco.Level))
					return
				}

				err = database.SetUserJob(ctx, i.GuildID, user.ID, jobID)
				if err != nil {
					slog.Error("Failed to apply for job", "error", err)
					SendError(s, i, "Failed to apply for the job.")
					return
				}

				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: fmt.Sprintf("✅ You have successfully applied for and started working as a **%s**!", job.Name),
					},
				})

			case "quit":
				err := database.RemoveUserJob(ctx, i.GuildID, user.ID)
				if err != nil {
					slog.Error("Failed to quit job", "error", err)
					SendError(s, i, "Failed to quit your job.")
					return
				}

				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: "✅ You have successfully quit your job.",
					},
				})

			case "info":
				eco, err := database.GetUserEconomy(ctx, i.GuildID, user.ID)
				if err != nil {
					SendError(s, i, "Failed to retrieve your economy data.")
					return
				}

				if eco.JobID == nil {
					s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
						Type: discordgo.InteractionResponseChannelMessageWithSource,
						Data: &discordgo.InteractionResponseData{
							Content: "You currently don't have a job. Use `/job list` to find one!",
						},
					})
					return
				}

				job, err := database.GetJob(ctx, *eco.JobID)
				if err != nil {
					slog.Error("Failed to retrieve user job", "error", err)
					SendError(s, i, "Failed to retrieve your job information.")
					return
				}

				embed := &discordgo.MessageEmbed{
					Title:       "Your Job Info",
					Description: fmt.Sprintf("You are currently working as a **%s**.", job.Name),
					Color:       0x2ecc71,
					Fields: []*discordgo.MessageEmbedField{
						{
							Name:   "Description",
							Value:  job.Description,
							Inline: false,
						},
						{
							Name:   "Salary",
							Value:  fmt.Sprintf("%d coins", job.Salary),
							Inline: true,
						},
						{
							Name:   "Required Level",
							Value:  fmt.Sprintf("%d", job.RequiredLevel),
							Inline: true,
						},
					},
				}

				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Embeds: []*discordgo.MessageEmbed{embed},
					},
				})

			case "create":
				// Check for administrator permissions
				if i.Member.Permissions&discordgo.PermissionAdministrator == 0 {
					SendError(s, i, "You must be an administrator to use this command.")
					return
				}

				subOptions := options[0].Options
				var name, description string
				var salary, requiredLevel int

				for _, opt := range subOptions {
					switch opt.Name {
					case "name":
						name = opt.StringValue()
					case "description":
						description = opt.StringValue()
					case "salary":
						salary = int(opt.IntValue())
					case "required_level":
						requiredLevel = int(opt.IntValue())
					}
				}

				err := database.CreateJob(ctx, i.GuildID, name, description, salary, requiredLevel)
				if err != nil {
					slog.Error("Failed to create job", "error", err)
					SendError(s, i, "Failed to create the job. Ensure a job with this name doesn't already exist.")
					return
				}

				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: fmt.Sprintf("✅ Job **%s** created successfully with salary **%d coins** and required level **%d**.", name, salary, requiredLevel),
					},
				})
			}
		},
	}
}
