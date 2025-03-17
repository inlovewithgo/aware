package antinuke

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

const (
	whitelistAddButton    = "whitelist_add"
	whitelistRemoveButton = "whitelist_remove"
	whitelistListButton   = "whitelist_list"
	whitelistUserSelect   = "whitelist_user_select"
	whitelistUserRemove   = "whitelist_user_remove"
	whitelistConfirmAdd   = "whitelist_confirm_add"
	whitelistConfirmRemove = "whitelist_confirm_remove"
)

func WhitelistCommand(s *discordgo.Session, m *discordgo.MessageCreate) {
	if !strings.HasPrefix(m.Content, ",whitelist") {
		return
	}

	// Check if user is server owner
	guild, err := s.Guild(m.GuildID)
	if err != nil {
		s.ChannelMessageSend(m.ChannelID, "Error fetching guild information.")
		return
	}

	if m.Author.ID != guild.OwnerID {
		s.ChannelMessageSend(m.ChannelID, "Only the server owner can use this command!")
		return
	}

	args := strings.Fields(m.Content)
	
	// Handle subcommands
	if len(args) > 1 {
		switch args[1] {
		case "add":
			if len(args) > 2 {
				// Direct add via mention or ID
				userID := strings.Trim(args[2], "<@!>")
				addUserToWhitelist(s, m.GuildID, userID, m.Author.ID)
				s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("User <@%s> has been added to the whitelist.", userID))
			} else {
				s.ChannelMessageSend(m.ChannelID, "Please specify a user to add: `,whitelist add @user`")
			}
			return
		case "remove":
			if len(args) > 2 {
				// Direct remove via mention or ID
				userID := strings.Trim(args[2], "<@!>")
				removeUserFromWhitelist(s, m.GuildID, userID)
				s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("User <@%s> has been removed from the whitelist.", userID))
			} else {
				s.ChannelMessageSend(m.ChannelID, "Please specify a user to remove: `,whitelist remove @user`")
			}
			return
		case "list":
			// List all whitelisted users
			listWhitelistedUsers(s, m.ChannelID, m.GuildID)
			return
		}
	}

	// Main whitelist command - show buttons
	showWhitelistMenu(s, m.ChannelID, m.GuildID)
}

func showWhitelistMenu(s *discordgo.Session, channelID, guildID string) {
	embed := &discordgo.MessageEmbed{
		Title:       "Anti-Nuke Whitelist Management",
		Description: "Manage users who are exempt from anti-nuke protection\nWhitelist Manager Commands\n- `,whitelist add`\n- `,whitelist remove`\n- `,whitelist list`",
		Color:       0x3498db,
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:  "Add User",
				Value: "Add a user to the whitelist",
			},
			{
				Name:  "Remove User",
				Value: "Remove a user from the whitelist",
			},
			{
				Name:  "List Users",
				Value: "View all whitelisted users",
			},
		},
	}

	components := []discordgo.MessageComponent{
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.Button{
					Label:    "Add User",
					Style:    discordgo.SuccessButton,
					CustomID: whitelistAddButton,
				},
				discordgo.Button{
					Label:    "Remove User",
					Style:    discordgo.DangerButton,
					CustomID: whitelistRemoveButton,
				},
				discordgo.Button{
					Label:    "List Users",
					Style:    discordgo.PrimaryButton,
					CustomID: whitelistListButton,
				},
			},
		},
	}

	_, err := s.ChannelMessageSendComplex(channelID, &discordgo.MessageSend{
		Embeds:     []*discordgo.MessageEmbed{embed},
		Components: components,
	})

	if err != nil {
		log.Printf("Error sending whitelist menu: %v\n", err)
	}
}

func HandleWhitelistButton(s *discordgo.Session, i *discordgo.InteractionCreate) {
    if i.Type != discordgo.InteractionMessageComponent {
        return
    }

    customID := i.MessageComponentData().CustomID
    
    // Handle the case where customID starts with a prefix
    if strings.HasPrefix(customID, whitelistConfirmAdd) || 
       strings.HasPrefix(customID, whitelistConfirmRemove) {
        // These are handled by their specific functions
        if strings.HasPrefix(customID, whitelistConfirmAdd) {
            handleConfirmAdd(s, i)
        } else if strings.HasPrefix(customID, whitelistConfirmRemove) {
            handleConfirmRemove(s, i)
        }
        return
    }

    // Handle exact matches
    switch customID {
    case whitelistAddButton:
        handleWhitelistAdd(s, i)
    case whitelistRemoveButton:
        handleWhitelistRemove(s, i)
    case whitelistListButton:
        handleWhitelistList(s, i)
    case "cancel_whitelist":
        handleCancelWhitelist(s, i)
    }
}

func handleCancelWhitelist(s *discordgo.Session, i *discordgo.InteractionCreate) {
    // First, acknowledge the interaction immediately
    err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
        Type: discordgo.InteractionResponseDeferredMessageUpdate,
    })
    
    if err != nil {
        log.Printf("Error responding to interaction: %v", err)
        return
    }
    
    // Now edit the message to show operation was cancelled
    _, err = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
        Content: stringPtr("Operation cancelled."),
        Components: &[]discordgo.MessageComponent{},
    })
    
    if err != nil {
        log.Printf("Error editing interaction response: %v", err)
    }
}

func HandleWhitelistSelect(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.Type != discordgo.InteractionMessageComponent {
		return
	}

	data := i.MessageComponentData()
	
	if data.CustomID == whitelistUserSelect {
		// User selected from dropdown for adding
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseUpdateMessage,
			Data: &discordgo.InteractionResponseData{
				Content: fmt.Sprintf("Add <@%s> to whitelist?", data.Values[0]),
				Components: []discordgo.MessageComponent{
					discordgo.ActionsRow{
						Components: []discordgo.MessageComponent{
							discordgo.Button{
								Label:    "Confirm",
								Style:    discordgo.SuccessButton,
								CustomID: whitelistConfirmAdd + ":" + data.Values[0],
							},
							discordgo.Button{
								Label:    "Cancel",
								Style:    discordgo.DangerButton,
								CustomID: "cancel_whitelist",
							},
						},
					},
				},
				Flags: discordgo.MessageFlagsEphemeral,
			},
		})
	} else if data.CustomID == whitelistUserRemove {
		// User selected from dropdown for removing
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseUpdateMessage,
			Data: &discordgo.InteractionResponseData{
				Content: fmt.Sprintf("Remove <@%s> from whitelist?", data.Values[0]),
				Components: []discordgo.MessageComponent{
					discordgo.ActionsRow{
						Components: []discordgo.MessageComponent{
							discordgo.Button{
								Label:    "Confirm",
								Style:    discordgo.DangerButton,
								CustomID: whitelistConfirmRemove + ":" + data.Values[0],
							},
							discordgo.Button{
								Label:    "Cancel",
								Style:    discordgo.SecondaryButton,
								CustomID: "cancel_whitelist",
							},
						},
					},
				},
				Flags: discordgo.MessageFlagsEphemeral,
			},
		})
	}
}

func handleWhitelistAdd(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// Check if user is server owner
	guild, err := s.Guild(i.GuildID)
	if err != nil || i.Member.User.ID != guild.OwnerID {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "Only the server owner can manage the whitelist.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	// Get guild members for dropdown
	members, err := s.GuildMembers(i.GuildID, "", 100)
	if err != nil {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "Error fetching guild members.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	// Create options for user select menu
	options := []discordgo.SelectMenuOption{}
	for _, member := range members {
		// Skip bots
		if member.User.Bot {
			continue
		}
		
		var count int
err := db.QueryRow("SELECT COUNT(*) FROM antinuke_whitelist WHERE guild_id = ? AND user_id = ?", 
    i.GuildID, member.User.ID).Scan(&count)

if err != nil || count > 0 {
    continue
}

		
		options = append(options, discordgo.SelectMenuOption{
			Label:       member.User.Username,
			Value:       member.User.ID,
			Description: "ID: " + member.User.ID,
		})
		
		// Limit to 25 options (Discord max)
		if len(options) >= 25 {
			break
		}
	}

	if len(options) == 0 {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "No eligible users found to add to whitelist.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "Select a user to add to the whitelist:",
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.SelectMenu{
							CustomID:    whitelistUserSelect,
							Placeholder: "Select a user",
							Options:     options,
						},
					},
				},
			},
			Flags: discordgo.MessageFlagsEphemeral,
		},
	})
}

func handleWhitelistRemove(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// Check if user is server owner
	guild, err := s.Guild(i.GuildID)
	if err != nil || i.Member.User.ID != guild.OwnerID {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "Only the server owner can manage the whitelist.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	// Get whitelisted users
	rows, err := db.Query("SELECT user_id FROM antinuke_whitelist WHERE guild_id = ?", i.GuildID)
	if err != nil {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "Error fetching whitelisted users.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}
	defer rows.Close()

	options := []discordgo.SelectMenuOption{}
	for rows.Next() {
		var userID string
		if err := rows.Scan(&userID); err != nil {
			continue
		}
		
		// Try to get user info
		user, err := s.User(userID)
		var label string
		if err != nil {
			label = "Unknown User"
		} else {
			label = user.Username
		}
		
		options = append(options, discordgo.SelectMenuOption{
			Label:       label,
			Value:       userID,
			Description: "ID: " + userID,
		})
	}

	if len(options) == 0 {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "No users in whitelist.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "Select a user to remove from the whitelist:",
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.SelectMenu{
							CustomID:    whitelistUserRemove,
							Placeholder: "Select a user",
							Options:     options,
						},
					},
				},
			},
			Flags: discordgo.MessageFlagsEphemeral,
		},
	})
}

func handleWhitelistList(s *discordgo.Session, i *discordgo.InteractionCreate) {
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags: discordgo.MessageFlagsEphemeral,
		},
	})
	
	listWhitelistedUsers(s, "", i.GuildID)
	
	// Create a formatted list of whitelisted users
	rows, err := db.Query("SELECT user_id, added_by, added_at FROM antinuke_whitelist WHERE guild_id = ?", i.GuildID)
	if err != nil {
		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: stringPtr("Error fetching whitelisted users."),
		})
		return
	}
	defer rows.Close()

	var users strings.Builder
	count := 0
	
	for rows.Next() {
		var userID, addedBy, addedAt string
		if err := rows.Scan(&userID, &addedBy, &addedAt); err != nil {
			continue
		}
		
		count++
		users.WriteString(fmt.Sprintf("%d. <@%s> (Added by <@%s>)\n", count, userID, addedBy))
	}

	if count == 0 {
		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: stringPtr("No users in whitelist."),
		})
		return
	}

	embed := &discordgo.MessageEmbed{
		Title:       "Whitelisted Users",
		Description: users.String(),
		Color:       0x3498db,
		Footer: &discordgo.MessageEmbedFooter{
			Text: fmt.Sprintf("Total: %d users", count),
		},
	}

	s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Embeds: &[]*discordgo.MessageEmbed{embed},
	})
}

func extractUserIDFromCustomID(customID string) string {
    parts := strings.Split(customID, ":")
    if len(parts) != 2 {
        return ""
    }
    return parts[1]
}

func handleConfirmAdd(s *discordgo.Session, i *discordgo.InteractionCreate) {
    // First, acknowledge the interaction immediately
    err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
        Type: discordgo.InteractionResponseDeferredMessageUpdate,
    })
    
    if err != nil {
        log.Printf("Error responding to interaction: %v", err)
        return
    }
    
    // Extract user ID from custom ID
    parts := strings.Split(i.MessageComponentData().CustomID, ":")
    if len(parts) != 2 {
        log.Printf("Invalid custom ID format: %s", i.MessageComponentData().CustomID)
        s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
            Content: stringPtr("❌ Error: Invalid button data."),
        })
        return
    }
    userID := parts[1]
    
    // Add user to whitelist with proper error handling
    err = addUserToWhitelist(s, i.GuildID, userID, i.Member.User.ID)
    if err != nil {
        log.Printf("Error adding user to whitelist: %v", err)
        s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
            Content: stringPtr("❌ Error adding user to whitelist."),
        })
        return
    }
    
    // Now edit the message after the database operation
    _, err = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
        Content: stringPtr(fmt.Sprintf("✅ User <@%s> has been added to the whitelist.", userID)),
        Components: &[]discordgo.MessageComponent{},
    })
    
    if err != nil {
        log.Printf("Error editing interaction response: %v", err)
    }
}

func handleConfirmRemove(s *discordgo.Session, i *discordgo.InteractionCreate) {
    // First, acknowledge the interaction immediately
    err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
        Type: discordgo.InteractionResponseDeferredMessageUpdate,
    })
    
    if err != nil {
        log.Printf("Error responding to interaction: %v", err)
        return
    }
    
    // Extract user ID from custom ID
    parts := strings.Split(i.MessageComponentData().CustomID, ":")
    if len(parts) != 2 {
        log.Printf("Invalid custom ID format: %s", i.MessageComponentData().CustomID)
        s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
            Content: stringPtr("❌ Error: Invalid button data."),
        })
        return
    }
    userID := parts[1]
    
    // Remove user from whitelist with proper error handling
    err = removeUserFromWhitelist(s, i.GuildID, userID)
    if err != nil {
        log.Printf("Error removing user from whitelist: %v", err)
        s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
            Content: stringPtr("❌ Error removing user from whitelist."),
        })
        return
    }
    
    // Now edit the message after the database operation
    _, err = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
        Content: stringPtr(fmt.Sprintf("✅ User <@%s> has been removed from the whitelist.", userID)),
        Components: &[]discordgo.MessageComponent{},
    })
    
    if err != nil {
        log.Printf("Error editing interaction response: %v", err)
    }
}


func addUserToWhitelist(s *discordgo.Session, guildID, userID, addedByID string) error {
	_, err := db.Exec(`
		INSERT OR REPLACE INTO antinuke_whitelist 
		(guild_id, user_id, added_by, added_at) 
		VALUES (?, ?, ?, ?)`,
		guildID, userID, addedByID, time.Now().Format(time.RFC3339))
	
	return err
}

func removeUserFromWhitelist(s *discordgo.Session, guildID, userID string) error {
    _, err := db.Exec(`
        DELETE FROM antinuke_whitelist 
        WHERE guild_id = ? AND user_id = ?`,
        guildID, userID)
    
    return err
}

func listWhitelistedUsers(s *discordgo.Session, channelID, guildID string) error {
    rows, err := db.Query("SELECT user_id FROM antinuke_whitelist WHERE guild_id = ?", guildID)
    if err != nil {
        return err
    }
    defer rows.Close()
    
    var users strings.Builder
    count := 0
    
    for rows.Next() {
        var userID string
        if err := rows.Scan(&userID); err != nil {
            continue
        }
        
        count++
        users.WriteString(fmt.Sprintf("%d. <@%s>\n", count, userID))
    }
    
    if channelID != "" {
        if count == 0 {
            s.ChannelMessageSend(channelID, "No users in whitelist.")
            return nil
        }
        
        embed := &discordgo.MessageEmbed{
            Title:       "Whitelisted Users",
            Description: users.String(),
            Color:       0x3498db,
            Footer: &discordgo.MessageEmbedFooter{
                Text: fmt.Sprintf("Total: %d users", count),
            },
        }
        
        _, err = s.ChannelMessageSendEmbed(channelID, embed)
    }
    
    return err
}