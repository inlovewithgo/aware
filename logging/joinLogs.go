package logging

import (
    "fmt"
    "time"
    "github.com/bwmarrin/discordgo"
)

const LogChannelID = "1341080255435247637"

func LogGuildJoin(s *discordgo.Session, guild *discordgo.Guild) {
	if !IsEnabled() {
        return
    }
    embed := &discordgo.MessageEmbed{
        Title: "Bot Joined New Guild",
        Color: 0x00ff00,
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
        },
        Timestamp: time.Now().Format(time.RFC3339),
        Footer: &discordgo.MessageEmbedFooter{
            Text: "Bot Guild Join Log",
        },
    }

    if guild.Icon != "" {
        embed.Thumbnail = &discordgo.MessageEmbedThumbnail{
            URL: fmt.Sprintf("https://cdn.discordapp.com/icons/%s/%s.png", guild.ID, guild.Icon),
        }
    }

    _, err := s.ChannelMessageSendEmbed(LogChannelID, embed)
    if err != nil {
        fmt.Printf("Error sending guild join log: %v\n", err)
    }
}
