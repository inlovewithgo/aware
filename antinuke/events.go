package antinuke

import (
    "fmt"
    "database/sql"
    "time"
	"strings"
    "sync"
    "github.com/bwmarrin/discordgo"
)

var (
    actionCounts = make(map[string]*ActionCount)
    actionMutex  sync.RWMutex
)

type ActionCount struct {
    MinuteCount int
    HourCount   int
    LastMinute  time.Time
    LastHour    time.Time
}

func InitEvents(s *discordgo.Session) {
    // Set required intents first
    s.Identify.Intents = discordgo.IntentsGuildWebhooks |
        discordgo.IntentsGuildMembers |
        discordgo.IntentsGuildBans |
        discordgo.IntentsGuilds |
        discordgo.IntentsGuildMessages

    // Register handlers after setting intents
    s.AddHandler(handleChannelDelete)
    s.AddHandler(handleRoleDelete)
    s.AddHandler(handleBanAdd)
    s.AddHandler(handleMemberRemove)
    s.AddHandler(handleWebhookUpdate)
    s.AddHandler(handleGuildUpdate)
}

func isWhitelisted(guildID, userID string) bool {
    var count int
    err := db.QueryRow("SELECT COUNT(*) FROM antinuke_whitelist WHERE guild_id = ? AND user_id = ?", 
        guildID, userID).Scan(&count)
    
    if err != nil {
        fmt.Printf("Error checking whitelist: %v\n", err)
        return false
    }
    
    return count > 0
}


func getAuditLogUser(s *discordgo.Session, guildID string, actionType discordgo.AuditLogAction) (string, error) {
    auditLog, err := s.GuildAuditLog(guildID, "", "", int(actionType), 1)
    if err != nil {
        return "", err
    }
    
    if len(auditLog.AuditLogEntries) == 0 {
        return "", fmt.Errorf("no audit log entry found")
    }
    
    return auditLog.AuditLogEntries[0].UserID, nil
}

func verifyWebhookPermissions(s *discordgo.Session, channelID string) error {
    perms, err := s.State.UserChannelPermissions(s.State.User.ID, channelID)
    if err != nil {
        return err
    }
    
    if perms&discordgo.PermissionManageWebhooks == 0 {
        return fmt.Errorf("missing MANAGE_WEBHOOKS permission")
    }
    return nil
}


func sendWebhookEmbed(webhookURL string, embed *discordgo.MessageEmbed) error {
    if webhookURL == "" {
        return fmt.Errorf("webhook URL is empty")
    }

    webhook, err := discordgo.New("")
    if err != nil {
        return fmt.Errorf("failed to create webhook session: %v", err)
    }

    parts := strings.Split(webhookURL, "/")
    if len(parts) < 2 {
        return fmt.Errorf("invalid webhook URL format")
    }

    webhookID := parts[len(parts)-2]
    webhookToken := parts[len(parts)-1]

    _, err = webhook.WebhookExecute(webhookID, webhookToken, true, &discordgo.WebhookParams{
        Embeds: []*discordgo.MessageEmbed{embed},
        Username: "Mod Logs",
    })
    return err
}


func createLogEmbed(userID, action, reason string, color int) *discordgo.MessageEmbed {
    return &discordgo.MessageEmbed{
        Title: "Anti-Nuke Detection",
        Description: fmt.Sprintf("**User:** <@%s>\n**Action:** %s\n**Reason:** %s", 
            userID, action, reason),
        Color: color,
        Timestamp: time.Now().Format(time.RFC3339),
        Footer: &discordgo.MessageEmbedFooter{
            Text: "Server Secured by Aware",
        },
    }
}

func createModLogEmbed(userID, action, punishment string) *discordgo.MessageEmbed {
    return &discordgo.MessageEmbed{
        Title: "Punishment Applied",
        Description: fmt.Sprintf("**User:** <@%s>\n**Violation:** %s\n**Punishment:** %s",
            userID, action, punishment),
        Color: 0xff0000,
        Timestamp: time.Now().Format(time.RFC3339),
        Footer: &discordgo.MessageEmbedFooter{
            Text: "Aware Moderation",
        },
    }
}

func sendLogs(s *discordgo.Session, guildID, userID, action, reason string) {
    // Check if user is whitelisted - if so, add note to logs
    isUserWhitelisted := isWhitelisted(guildID, userID)
    
    if isUserWhitelisted {
        // If whitelisted, we just log the action but don't apply punishment
        reason = fmt.Sprintf("%s (Action allowed - User is whitelisted)", reason)
    }
    
    var webhookURL, modWebhookURL string
    err := db.QueryRow(`
        SELECT webhook_id, mod_webhook_id 
        FROM antinuke_config 
        WHERE guild_id = ?`, guildID).Scan(&webhookURL, &modWebhookURL)
    if err != nil {
        if err == sql.ErrNoRows {
            fmt.Printf("No webhook configuration found for guild %s\n", guildID)
        } else {
            fmt.Printf("Database error getting webhooks: %v\n", err)
        }
        return
    }

    if webhookURL == "" || modWebhookURL == "" {
        fmt.Printf("Missing webhook URLs for guild %s\n", guildID)
        return
    }

    // Send logs with retries
    for i := 0; i < 3; i++ {
        if err := sendWebhookEmbed(webhookURL, createLogEmbed(userID, action, reason, 0xff6b6b)); err != nil {
            if i == 2 {
                fmt.Printf("Final attempt to send antinuke log failed: %v\n", err)
            }
            time.Sleep(time.Second * 2)
            continue
        }
        break
    }

    // Only send mod logs if user is not whitelisted
    if !isUserWhitelisted {
        punishment := getPunishmentType(guildID)
        for i := 0; i < 3; i++ {
            if err := sendWebhookEmbed(modWebhookURL, createModLogEmbed(userID, action, punishment)); err != nil {
                if i == 2 {
                    fmt.Printf("Final attempt to send mod log failed: %v\n", err)
                }
                time.Sleep(time.Second * 2)
                continue
            }
            break
        }
    }
}

func checkLimits(guildID, userID string) bool {
    // Skip limit checks for whitelisted users
    if isWhitelisted(guildID, userID) {
        return true
    }
    
    actionMutex.Lock()
    defer actionMutex.Unlock()

    key := fmt.Sprintf("%s:%s", guildID, userID)
    count, exists := actionCounts[key]
    if !exists {
        count = &ActionCount{
            LastMinute: time.Now(),
            LastHour:   time.Now(),
        }
        actionCounts[key] = count
    }

    // Reset counters if needed
    now := time.Now()
    if now.Sub(count.LastMinute) >= time.Minute {
        count.MinuteCount = 0
        count.LastMinute = now
    }
    if now.Sub(count.LastHour) >= time.Hour {
        count.HourCount = 0
        count.LastHour = now
    }

    var apm, aph int
    err := db.QueryRow(`
        SELECT actions_per_minute, actions_per_hour 
        FROM antinuke_config 
        WHERE guild_id = ?`, guildID).Scan(&apm, &aph)
    
    if err != nil {
        apm = 2
        aph = 10
    }
    
    if count.MinuteCount >= apm || count.HourCount >= aph {
        return false
    }   

    count.MinuteCount++
    count.HourCount++

    return true
}


func getPunishmentType(guildID string) string {
    var punishType string
    err := db.QueryRow(`
        SELECT punishment_type 
        FROM antinuke_config 
        WHERE guild_id = ?`, guildID).Scan(&punishType)
    
    if err != nil {
        return "quarantine" // Default punishment
    }
    return punishType
}

func applyPunishment(s *discordgo.Session, guildID, userID, reason string) {
    punishType := getPunishmentType(guildID)
    
    // Add prefix to reason
    reason = fmt.Sprintf("Server Secured by Aware | %s", reason)
    
    switch punishType {
    case "ban":
        if err := s.GuildBanCreateWithReason(guildID, userID, reason, 0); err != nil {
            fmt.Printf("Failed to ban user %s: %v\n", userID, err)
            return
        }
    
    case "kick":
        if err := s.GuildMemberDeleteWithReason(guildID, userID, reason); err != nil {
            fmt.Printf("Failed to kick user %s: %v\n", userID, err)
            return
        }
    
    case "quarantine":
        // Get quarantine role ID
        var quarantineRoleID string
        err := db.QueryRow(`
            SELECT quarantine_role_id 
            FROM antinuke_config 
            WHERE guild_id = ?`, guildID).Scan(&quarantineRoleID)
        
        if err != nil {
            fmt.Printf("Failed to get quarantine role: %v\n", err)
            return
        }

        // Get member information
        member, err := s.GuildMember(guildID, userID)
        if err != nil {
            fmt.Printf("Failed to get member info: %v\n", err)
            return
        }

        // Store original roles before removing
        originalRoles := member.Roles

        // Remove all roles
        for _, roleID := range originalRoles {
            if err := s.GuildMemberRoleRemove(guildID, userID, roleID); err != nil {
                fmt.Printf("Failed to remove role %s: %v\n", roleID, err)
                continue
            }
        }

        if err := s.GuildMemberRoleAdd(guildID, userID, quarantineRoleID); err != nil {
            fmt.Printf("Failed to add quarantine role: %v\n", err)
            for _, roleID := range originalRoles {
                s.GuildMemberRoleAdd(guildID, userID, roleID)
            }
            return
        }
    }

    sendLogs(s, guildID, userID, "Punishment Applied", reason)
}

func handleRoleDelete(s *discordgo.Session, e *discordgo.GuildRoleDelete) {
    userID, err := getAuditLogUser(s, e.GuildID, discordgo.AuditLogActionRoleDelete)
    if err != nil {
        fmt.Printf("Error getting audit log user: %v\n", err)
        return
    }
    
    // Skip if user is whitelisted
    if isWhitelisted(e.GuildID, userID) {
        return
    }

    if !checkLimits(e.GuildID, userID) {
        reason := "Mass Role Deletion Detected"
        applyPunishment(s, e.GuildID, userID, reason)
        sendLogs(s, e.GuildID, userID, "Role Deletion", reason)
    }
}

func handleChannelDelete(s *discordgo.Session, e *discordgo.ChannelDelete) {
    userID, err := getAuditLogUser(s, e.GuildID, discordgo.AuditLogActionChannelDelete)
    if err != nil {
        return
    }
    
    // Skip if user is whitelisted
    if isWhitelisted(e.GuildID, userID) {
        return
    }

    if !checkLimits(e.GuildID, userID) {
        reason := "Mass Channel Deletion Detected"
        applyPunishment(s, e.GuildID, userID, reason)
        sendLogs(s, e.GuildID, userID, "Channel Deletion", reason)
    }
}

func handleWebhookUpdate(s *discordgo.Session, e *discordgo.WebhooksUpdate) {
    userID, err := getAuditLogUser(s, e.GuildID, discordgo.AuditLogActionWebhookCreate)
    if err != nil {
        return
    }
    
    // Skip if user is whitelisted
    if isWhitelisted(e.GuildID, userID) {
        return
    }

    if !checkLimits(e.GuildID, userID) {
        reason := "Mass Webhook Creation/Deletion Detected"
        applyPunishment(s, e.GuildID, userID, reason)
        sendLogs(s, e.GuildID, userID, "Webhook Update", reason)
    }
}

func handleGuildUpdate(s *discordgo.Session, e *discordgo.GuildUpdate) {
    userID, err := getAuditLogUser(s, e.Guild.ID, discordgo.AuditLogActionGuildUpdate)
    if err != nil {
        return
    }
    
    // Skip if user is whitelisted
    if isWhitelisted(e.Guild.ID, userID) {
        return
    }

    if !checkLimits(e.Guild.ID, userID) {
        reason := "Suspicious Guild Updates Detected"
        applyPunishment(s, e.Guild.ID, userID, reason)
        sendLogs(s, e.Guild.ID, userID, "Guild Update", reason)
    }
}

func handleBanAdd(s *discordgo.Session, e *discordgo.GuildBanAdd) {
    userID, err := getAuditLogUser(s, e.GuildID, discordgo.AuditLogActionMemberBanAdd)
    if err != nil {
        return
    }
    
    // Skip if user is whitelisted
    if isWhitelisted(e.GuildID, userID) {
        return
    }

    if !checkLimits(e.GuildID, userID) {
        reason := "Mass Ban Detected"
        applyPunishment(s, e.GuildID, userID, reason)
        sendLogs(s, e.GuildID, userID, "Member Ban", reason)
    }
}

func handleMemberRemove(s *discordgo.Session, e *discordgo.GuildMemberRemove) {
    userID, err := getAuditLogUser(s, e.GuildID, discordgo.AuditLogActionMemberKick)
    if err != nil {
        return
    }
    
    // Skip if user is whitelisted
    if isWhitelisted(e.GuildID, userID) {
        return
    }

    if !checkLimits(e.GuildID, userID) {
        reason := "Mass Kick Detected"
        applyPunishment(s, e.GuildID, userID, reason)
        sendLogs(s, e.GuildID, userID, "Member Kick", reason)
    }
}
