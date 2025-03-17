package logging

import (
    "fmt"
    "time"
    "github.com/bwmarrin/discordgo"
)

func LogGuildLeave(s *discordgo.Session, guild *discordgo.Guild) {
    if !IsEnabled() {
        return
    }

    embed := &discordgo.MessageEmbed{
        Title: "Bot Left Guild",
        Color: 0xff0000,
        Fields: []*discordgo.MessageEmbedField{
            {
                Name:   "Guild Name",
                Value:  guild.Name,
                Inline: true,
            },
            {
                Name:   "Guild ID",
                Value:  guild.ID,
                Inline: true,
            },
            {
                Name:   "Member Count",
                Value:  fmt.Sprintf("%d members", guild.MemberCount),
                Inline: true,
            },
            {
                Name:   "Owner ID",
                Value:  guild.OwnerID,
                Inline: true,
            },
            {
                Name:   "Left At",
                Value:  time.Now().Format("2006-01-02 15:04:05 MST"),
                Inline: false,
            },
        },
        Timestamp: time.Now().Format(time.RFC3339),
        Footer: &discordgo.MessageEmbedFooter{
            Text: "Bot Guild Leave Log",
        },
    }

    if guild.Icon != "" {
        embed.Thumbnail = &discordgo.MessageEmbedThumbnail{
            URL: fmt.Sprintf("https://cdn.discordapp.com/icons/%s/%s.png", guild.ID, guild.Icon),
        }
    }

    _, err := s.ChannelMessageSendEmbed(LogChannelID, embed)
    if err != nil {
        fmt.Printf("Error sending guild leave log: %v\n", err)
    }
}
