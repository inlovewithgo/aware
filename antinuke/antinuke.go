package antinuke

import (
    "github.com/bwmarrin/discordgo"
    "fmt"
    "database/sql"
    "strconv"
)

var (
    db *sql.DB
    setupButton = "setup_antinuke"
    deleteButton = "delete_antinuke"
    
    configButton = "config_antinuke"
    limitsButton = "limits_antinuke"
    punishButton = "punish_antinuke"
    quarantineRole = "Quarantined"
)

var (
    kickButton = "kick_antinuke"
    banButton = "ban_antinuke"
    quarantineButton = "quarantine_antinuke" 
)

type Config struct {
    ActionsPerMinute int
    ActionsPerHour   int
    PunishmentType   string
}

func stringPtr(s string) *string {
    return &s
}


func InitAntinuke(database *sql.DB) {
    db = database
    createTables()
}

func insertInitialConfig(guildID string) error {
    _, err := db.Exec(`
        INSERT OR IGNORE INTO antinuke_config (
            guild_id, 
            actions_per_minute,
            actions_per_hour,
            punishment_type,
            enabled
        ) VALUES (?, 5, 20, 'quarantine', true)`,
        guildID,
    )
    return err
}

func ensureGuildConfig(guildID string) error {
    // Check if config exists
    var count int
    err := db.QueryRow("SELECT COUNT(*) FROM antinuke_config WHERE guild_id = ?", guildID).Scan(&count)
    if err != nil {
        return err
    }

    // Insert initial config if none exists
    if count == 0 {
        _, err = db.Exec(`
            INSERT INTO antinuke_config (
                guild_id,
                actions_per_minute,
                actions_per_hour,
                punishment_type,
                enabled
            ) VALUES (?, 5, 20, 'quarantine', true)`,
            guildID,
        )
    }
    return err
}


func createTables() {
    // Create table with all required columns (only if it doesn't exist)
    _, err := db.Exec(`
        CREATE TABLE IF NOT EXISTS antinuke_config (
            guild_id TEXT PRIMARY KEY,
            logs_channel_id TEXT,
            mod_logs_channel_id TEXT,
            actions_per_minute INTEGER DEFAULT 5,
            actions_per_hour INTEGER DEFAULT 20,
            punishment_type TEXT DEFAULT 'quarantine',
            quarantine_role_id TEXT,
            webhook_id TEXT,
            mod_webhook_id TEXT,
            enabled BOOLEAN DEFAULT true
        )
    `)
    if err != nil {
        fmt.Printf("Error creating tables: %v\n", err)
    }
}


func Int64Ptr(i int64) *int64 {
    return &i
}

func SetupCommand(s *discordgo.Session, m *discordgo.MessageCreate) {

    
    if m.Content != ",antinuke setup" {
        return
    }

    guild, err := s.Guild(m.GuildID)
    if err != nil {
        return
    }

    if m.Author.ID != guild.OwnerID {
        s.ChannelMessageSend(m.ChannelID, "Only the server owner can use this command!")
        return
    }

    embed := &discordgo.MessageEmbed{
        Title:       "Anti-Nuke Setup",
        Description: "Configure the Anti-Nuke protection system for your server",
        Color:       0x3498db,
        Fields: []*discordgo.MessageEmbedField{
            {
                Name:  "Setup",
                Value: "Creates required channels, webhooks, and roles",
            },
            {
                Name:  "Delete",
                Value: "Removes all Anti-Nuke configurations",
            },
            {
                Name:  "Config",
                Value: "Displays current Anti-Nuke settings",
            },
        },
    }

    

    components := []discordgo.MessageComponent{
        discordgo.ActionsRow{
            Components: []discordgo.MessageComponent{
                discordgo.Button{
                    Label:    "Start Setup",
                    Style:    discordgo.SuccessButton,
                    CustomID: setupButton,
                },
                discordgo.Button{
                    Label:    "Delete Setup",
                    Style:    discordgo.DangerButton,
                    CustomID: deleteButton,
                },
                discordgo.Button{
                    Label:    "View Config",
                    Style:    discordgo.PrimaryButton,
                    CustomID: configButton,
                },
                discordgo.Button{
                    Label:    "Set Limits",
                    Style:    discordgo.SecondaryButton,
                    CustomID: limitsButton,
                },
                discordgo.Button{
                    Label:    "Punishment",
                    Style:    discordgo.SecondaryButton,
                    CustomID: punishButton,
                },
            },
        },
    }    

    _, err = s.ChannelMessageSendComplex(m.ChannelID, &discordgo.MessageSend{
        Embeds:     []*discordgo.MessageEmbed{embed},
        Components: components,
    })

    if err != nil {
        fmt.Printf("Error sending setup message: %v\n", err)
    }
}

func handlePunishmentOptions(s *discordgo.Session, i *discordgo.InteractionCreate) {
    // Send an ephemeral message with punishment options
    punishEmbed := &discordgo.MessageEmbed{
        Title:       "Punishment Options",
        Description: "Select a punishment type for users who trigger anti-nuke protection",
        Color:       0xff0000,
    }

    punishmentComponents := []discordgo.MessageComponent{
        discordgo.ActionsRow{
            Components: []discordgo.MessageComponent{
                discordgo.Button{
                    Label:    "Kick",
                    Style:    discordgo.DangerButton,
                    CustomID: kickButton,
                },
                discordgo.Button{
                    Label:    "Ban",
                    Style:    discordgo.DangerButton, 
                    CustomID: banButton,
                },
                discordgo.Button{
                    Label:    "Quarantine",
                    Style:    discordgo.PrimaryButton,
                    CustomID: quarantineButton,
                },
            },
        },
    }

    s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
        Type: discordgo.InteractionResponseChannelMessageWithSource,
        Data: &discordgo.InteractionResponseData{
            Embeds:     []*discordgo.MessageEmbed{punishEmbed},
            Components: punishmentComponents,
            Flags:      discordgo.MessageFlagsEphemeral,
        },
    })
}


func handleStartSetup(s *discordgo.Session, i *discordgo.InteractionCreate) {
    s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
        Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
    })

    guildID := i.GuildID

    

    // Create Anti-Nuke Logs channel
    antiNukeLogs, err := s.GuildChannelCreate(guildID, "antinuke-logs", discordgo.ChannelTypeGuildText)
    if err != nil {
        respondWithError(s, i, "Failed to create antinuke-logs channel")
        return
    }

    // Hide channel from @everyone
    err = s.ChannelPermissionSet(
        antiNukeLogs.ID, 
        guildID, // Changed from i.Guild.ID to guildID
        discordgo.PermissionOverwriteTypeRole,
        0, 
        discordgo.PermissionViewChannel,
    )
    if err != nil {
        respondWithError(s, i, "Failed to set channel permissions")
        return
    }


    // Create Mod Logs channel with hidden permissions
    modLogs, err := s.GuildChannelCreate(guildID, "mod-logs", discordgo.ChannelTypeGuildText)
    if err != nil {
        respondWithError(s, i, "Failed to create mod-logs channel")
        return
    }

    // Hide channel from @everyone
    err = s.ChannelPermissionSet(
        modLogs.ID, 
        guildID, // Changed from i.Guild.ID to guildID
        discordgo.PermissionOverwriteTypeRole,
        0, 
        discordgo.PermissionViewChannel,
    )
        if err != nil {
        respondWithError(s, i, "Failed to set channel permissions")
        return
    }

    

    // Create webhook for Anti-Nuke Logs
    antiNukeWebhook, err := s.WebhookCreate(antiNukeLogs.ID, "Anti-Nuke Logs", "")
    if err != nil {
        respondWithError(s, i, "Failed to create antinuke webhook")
        return
    }

    // Create webhook for Mod Logs
    modWebhook, err := s.WebhookCreate(modLogs.ID, "Mod Logs", "")
    if err != nil {
        respondWithError(s, i, "Failed to create mod webhook")
        return
    }

// Create Quarantined role
    var permissions int64 = 0
    color := 0x36393f
    roleParams := &discordgo.RoleParams{
        Name:        quarantineRole,
        Permissions: &permissions,
        Color:       &color,
    }

    
    quarantinedRole, err := s.GuildRoleCreate(guildID, roleParams)
    if err != nil {
        respondWithError(s, i, "Failed to create quarantined role")
        return
    }

    // Get all existing channels
    channels, err := s.GuildChannels(guildID)
    if err != nil {
        respondWithError(s, i, "Failed to get guild channels")
        return
    }

    // Disable view permissions for quarantine role in all existing channels
    for _, channel := range channels {
        err = s.ChannelPermissionSet(
            channel.ID,
            quarantinedRole.ID,
            discordgo.PermissionOverwriteTypeRole,
            0,
            discordgo.PermissionViewChannel,
        )
        if err != nil {
            fmt.Printf("Error setting permissions for channel %s: %v\n", channel.ID, err)
            continue
        }
    }

    // Create event handler for new channels
    s.AddHandler(func(s *discordgo.Session, c *discordgo.ChannelCreate) {
        if c.GuildID == guildID {
            err := s.ChannelPermissionSet(
                c.ID,
                quarantinedRole.ID,
                discordgo.PermissionOverwriteTypeRole,
                0,
                discordgo.PermissionViewChannel,
            )
            if err != nil {
                fmt.Printf("Error setting permissions for new channel %s: %v\n", c.ID, err)
            }
        }
    })

    // Store role ID in database for future reference
    _, err = db.Exec(`
        UPDATE antinuke_config 
        SET quarantine_role_id = ?
        WHERE guild_id = ?`,
        quarantinedRole.ID, guildID,
    )
    if err != nil {
        fmt.Printf("Error storing quarantine role in database: %v\n", err)
    }

    // Store configuration in database
    _, err = db.Exec(`
        INSERT OR REPLACE INTO antinuke_config 
        (guild_id, logs_channel_id, mod_logs_channel_id, quarantine_role_id, webhook_id, mod_webhook_id, enabled) 
        VALUES (?, ?, ?, ?, ?, ?, true)`,
        guildID, antiNukeLogs.ID, modLogs.ID, quarantinedRole.ID, antiNukeWebhook.ID, modWebhook.ID,
    )
    if err != nil {
        fmt.Printf("Error storing config in database: %v\n", err)
    }

    successEmbed := &discordgo.MessageEmbed{
        Title:       "Setup Complete",
        Description: "Anti-Nuke system has been successfully configured!",
        Color:       0x00ff00,
        Fields: []*discordgo.MessageEmbedField{
            {
                Name:  "Channels Created",
                Value: fmt.Sprintf("• %s\n• %s", antiNukeLogs.Mention(), modLogs.Mention()),
            },
            {
                Name:  "Role Created",
                Value: fmt.Sprintf("• %s", quarantinedRole.Name),
            },
        },
    }

    punishmentComponents := []discordgo.MessageComponent{
        discordgo.ActionsRow{
            Components: []discordgo.MessageComponent{
                discordgo.Button{
                    Label:    "Kick",
                    Style:    discordgo.DangerButton,
                    CustomID: kickButton,
                },
                discordgo.Button{
                    Label:    "Ban",
                    Style:    discordgo.DangerButton, 
                    CustomID: banButton,
                },
                discordgo.Button{
                    Label:    "Quarantine",
                    Style:    discordgo.PrimaryButton,
                    CustomID: quarantineButton,
                },
            },
        },
    }
    

    // Send punishment panel
    punishEmbed := &discordgo.MessageEmbed{
        Title:       "Punishment Panel",
        Description: "Select an action to punish users",
        Color:       0xff0000,
    }

    s.ChannelMessageSendComplex(i.ChannelID, &discordgo.MessageSend{
        Embeds:     []*discordgo.MessageEmbed{punishEmbed},
        Components: punishmentComponents,
    })    

    s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
        Embeds: &[]*discordgo.MessageEmbed{successEmbed},
    })

    // Store complete webhook URLs
    webhookURL := fmt.Sprintf("https://discord.com/api/webhooks/%s/%s", antiNukeWebhook.ID, antiNukeWebhook.Token)
    modWebhookURL := fmt.Sprintf("https://discord.com/api/webhooks/%s/%s", modWebhook.ID, modWebhook.Token)
    
    // Store complete webhook URLs
    result, err := db.Exec(`
        INSERT OR REPLACE INTO antinuke_config 
        (guild_id, logs_channel_id, mod_logs_channel_id, quarantine_role_id, webhook_id, mod_webhook_id, enabled) 
        VALUES (?, ?, ?, ?, ?, ?, true)`,
        guildID, antiNukeLogs.ID, modLogs.ID, quarantinedRole.ID, webhookURL, modWebhookURL,
        
    )

    _ = result
    
    if err != nil {
        fmt.Printf("Error storing webhook URLs in database: %v\n", err)
        return
    }
    
}

func handleDeleteSetup(s *discordgo.Session, i *discordgo.InteractionCreate) {
    s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
        Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
    })

    guildID := i.GuildID

    // Get configuration from database
    var logsChannelID, modLogsChannelID, quarantineRoleID string
    err := db.QueryRow(`
        SELECT logs_channel_id, mod_logs_channel_id, quarantine_role_id 
        FROM antinuke_config 
        WHERE guild_id = ?`, guildID).Scan(&logsChannelID, &modLogsChannelID, &quarantineRoleID)
    
    if err == nil {
        // Delete channels
        if logsChannelID != "" {
            s.ChannelDelete(logsChannelID)
        }
        if modLogsChannelID != "" {
            s.ChannelDelete(modLogsChannelID)
        }
        if quarantineRoleID != "" {
            s.GuildRoleDelete(guildID, quarantineRoleID)
        }
    }

    // Remove from database
    _, err = db.Exec(`DELETE FROM antinuke_config WHERE guild_id = ?`, guildID)
    if err != nil {
        fmt.Printf("Error removing config from database: %v\n", err)
    }

    deleteEmbed := &discordgo.MessageEmbed{
        Title:       "Setup Deleted",
        Description: "All Anti-Nuke configurations have been removed",
        Color:       0xff0000,
    }

    s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
        Embeds: &[]*discordgo.MessageEmbed{deleteEmbed},
    })
}



func disableButton(s *discordgo.Session, i *discordgo.InteractionCreate, buttonID string) {
    for _, row := range i.Message.Components {
        if actionRow, ok := row.(discordgo.ActionsRow); ok {
            for _, comp := range actionRow.Components {
                if button, ok := comp.(discordgo.Button); ok {
                    if button.CustomID == buttonID {
                        button.Disabled = true
                        components := i.Message.Components
                        s.ChannelMessageEditComplex(&discordgo.MessageEdit{
                            Channel:    i.ChannelID,
                            ID:        i.Message.ID,
                            Components: &components,
                        })
                        return
                    }
                }
            }
        }
    }
}


func handleViewConfig(s *discordgo.Session, i *discordgo.InteractionCreate) {
    s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
        Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
    })

    guildID := i.GuildID

    var logsChannelID, modLogsChannelID, quarantineRoleID string
    var enabled bool

    err := db.QueryRow(`
        SELECT logs_channel_id, mod_logs_channel_id, quarantine_role_id, enabled 
        FROM antinuke_config 
        WHERE guild_id = ?`, guildID).Scan(&logsChannelID, &modLogsChannelID, &quarantineRoleID, &enabled)

    status := "Not Configured"
    if err == nil && enabled {
        status = "Active"
    }

    configEmbed := &discordgo.MessageEmbed{
        Title: "Anti-Nuke Configuration",
        Color: 0x3498db,
        Fields: []*discordgo.MessageEmbedField{
            {
                Name:  "Status",
                Value: status,
            },
            {
                Name:  "Channels",
                Value: fmt.Sprintf("Logs Channel: <#%s>\nMod Logs: <#%s>", logsChannelID, modLogsChannelID),
            },
        },
    }

    s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
        Embeds: &[]*discordgo.MessageEmbed{configEmbed},
    })
}


func HandleSetupButton(s *discordgo.Session, i *discordgo.InteractionCreate) {
    if i.Type != discordgo.InteractionMessageComponent {
        return
    }

    switch i.MessageComponentData().CustomID {
    case setupButton:
        handleStartSetup(s, i)
        disableButton(s, i, setupButton)
    case deleteButton:
        handleDeleteSetup(s, i)
    case configButton:
        handleViewConfig(s, i)
    case punishButton:
        handlePunishmentOptions(s, i)
    case kickButton, banButton, quarantineButton:
        handlePunishmentSetup(s, i)
    case limitsButton:
        handleLimitsSetup(s, i)
    }
}


func respondWithError(s *discordgo.Session, i *discordgo.InteractionCreate, message string) {
    errorEmbed := &discordgo.MessageEmbed{
        Title:       "Setup Error",
        Description: message,
        Color:       0xff0000,
    }

    s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
        Embeds: &[]*discordgo.MessageEmbed{errorEmbed},
    })
}

func handlePunishmentSetup(s *discordgo.Session, i *discordgo.InteractionCreate) {
    switch i.MessageComponentData().CustomID {
    case kickButton:
        setPunishment(s, i, "kick")
    case banButton:
        setPunishment(s, i, "ban")
    case quarantineButton:
        setPunishment(s, i, "quarantine")
    }
}

func setPunishment(s *discordgo.Session, i *discordgo.InteractionCreate, punishType string) {
    // First respond to interaction to prevent timeout
    s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
        Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
    })

    // Ensure config exists
    if err := ensureGuildConfig(i.GuildID); err != nil {
        s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
            Content: stringPtr("Failed to initialize config: " + err.Error()),
        })
        return
    }

    _, err := db.Exec(`
        UPDATE antinuke_config 
        SET punishment_type = ?
        WHERE guild_id = ?`,
        punishType, i.GuildID,
    )
    
    if err != nil {
        s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
            Content: stringPtr("Failed to set punishment type: " + err.Error()),
        })
        return
    }

    successEmbed := &discordgo.MessageEmbed{
        Title:       "Punishment Updated",
        Description: fmt.Sprintf("Punishment type set to: %s", punishType),
        Color:       0x00ff00,
    }

    s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
        Embeds: &[]*discordgo.MessageEmbed{successEmbed},
    })
}

func handleLimitsSetup(s *discordgo.Session, i *discordgo.InteractionCreate) {
    modal := &discordgo.InteractionResponse{
        Type: discordgo.InteractionResponseModal,
        Data: &discordgo.InteractionResponseData{
            CustomID: "limits_modal",
            Title:    "Set Action Limits",
            Components: []discordgo.MessageComponent{
                discordgo.ActionsRow{
                    Components: []discordgo.MessageComponent{
                        discordgo.TextInput{
                            CustomID:    "actions_per_minute",
                            Label:       "Actions Per Minute",
                            Style:       discordgo.TextInputShort,
                            Placeholder: "Default: 5",
                            Required:    true,
                            MinLength:   1,
                            MaxLength:   2,
                        },
                    },
                },
                discordgo.ActionsRow{
                    Components: []discordgo.MessageComponent{
                        discordgo.TextInput{
                            CustomID:    "actions_per_hour",
                            Label:       "Actions Per Hour",
                            Style:       discordgo.TextInputShort,
                            Placeholder: "Default: 20",
                            Required:    true,
                            MinLength:   1,
                            MaxLength:   3,
                        },
                    },
                },
            },
        },
    }

    s.InteractionRespond(i.Interaction, modal)
}

func HandleLimitsModal(s *discordgo.Session, i *discordgo.InteractionCreate) {
    data := i.ModalSubmitData()
    
    // Parse actions per minute
    apm, err := strconv.Atoi(data.Components[0].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value)

    if err := ensureGuildConfig(i.GuildID); err != nil {
        s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
            Type: discordgo.InteractionResponseChannelMessageWithSource,
            Data: &discordgo.InteractionResponseData{
                Content: "Failed to initialize config: " + err.Error(),
                Flags:   discordgo.MessageFlagsEphemeral,
            },
        })
        return
    }

    if err != nil {
        s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
            Type: discordgo.InteractionResponseChannelMessageWithSource,
            Data: &discordgo.InteractionResponseData{
                Content: "Invalid number for actions per minute",
                Flags:   discordgo.MessageFlagsEphemeral,
            },
        })
        return
    }

    // Parse actions per hour
    aph, err := strconv.Atoi(data.Components[1].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value)
    if err != nil {
        s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
            Type: discordgo.InteractionResponseChannelMessageWithSource,
            Data: &discordgo.InteractionResponseData{
                Content: "Invalid number for actions per hour",
                Flags:   discordgo.MessageFlagsEphemeral,
            },
        })
        return
    }

    // Validate the numbers
    if apm < 1 || aph < 1 {
        s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
            Type: discordgo.InteractionResponseChannelMessageWithSource,
            Data: &discordgo.InteractionResponseData{
                Content: "Values must be greater than 0",
                Flags:   discordgo.MessageFlagsEphemeral,
            },
        })
        return
    }

    _, err = db.Exec(`
        UPDATE antinuke_config 
        SET actions_per_minute = ?, actions_per_hour = ?
        WHERE guild_id = ?`,
        apm, aph, i.GuildID,
    )

    if err != nil {
        s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
            Type: discordgo.InteractionResponseChannelMessageWithSource,
            Data: &discordgo.InteractionResponseData{
                Content: "Failed to update limits in database",
                Flags:   discordgo.MessageFlagsEphemeral,
            },
        })
        return
    }

    successEmbed := &discordgo.MessageEmbed{
        Title: "Limits Updated",
        Description: fmt.Sprintf("Actions per minute: %d\nActions per hour: %d", apm, aph),
        Color: 0x00ff00,
    }

    s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
        Type: discordgo.InteractionResponseChannelMessageWithSource,
        Data: &discordgo.InteractionResponseData{
            Embeds: []*discordgo.MessageEmbed{successEmbed},
            Flags:  discordgo.MessageFlagsEphemeral,
        },
    })
}

