package main

import (
    "fmt"
    "log"
    "time"
    "os"
    "os/signal"


    "crypto/rand"
    "encoding/base64"

    "syscall"

    "aware/logging"
    "aware/antinuke"

    "github.com/bwmarrin/discordgo"
)

func guildJoinHandler(s *discordgo.Session, g *discordgo.GuildCreate) {
    logging.LogGuildJoin(s, g.Guild)
}

func guildLeaveHandler(s *discordgo.Session, g *discordgo.GuildCreate) {
    logging.LogGuildLeave(s, g.Guild)
}

func generateSessionSecret() (string, error) {
    b := make([]byte, 32)
    _, err := rand.Read(b)
    if err != nil {
        return "", err
    }
    return base64.StdEncoding.EncodeToString(b), nil
}


func main() {

    initDB()
    defer db.Close()

    antinuke.InitAntinuke(db)

    dg, err := discordgo.New("Bot " + "")
    if err != nil {
        log.Fatal("Error creating Discord session: ", err)
    }

    dg.AddHandler(antinuke.SetupCommand)
    dg.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
        antinuke.HandleSetupButton(s, i)
    })

    dg.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
        if i.Type == discordgo.InteractionModalSubmit && i.ModalSubmitData().CustomID == "limits_modal" {
            antinuke.HandleLimitsModal(s, i)
        }
    })

    dg.Identify.Intents = discordgo.IntentsAll

    antinuke.InitEvents(dg)
    dg.AddHandler(messageCreate)
    dg.AddHandler(guildJoinHandler)

    dg.AddHandler(antinuke.WhitelistCommand)
    dg.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
        antinuke.HandleWhitelistButton(s, i)
    })
    dg.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
        antinuke.HandleWhitelistSelect(s, i)
    })


    dg.AddHandler(guildLeaveHandler)

    err = dg.Open()
    if err != nil {
        log.Fatal("Error opening connection: ", err)
    }

    err = dg.UpdateGameStatus(0, "aware.wtf/discord")
    if err != nil {
        log.Println("Error setting status: ", err)
    }

    fmt.Println("Bot is running. Press CTRL-C to exit.")
    sc := make(chan os.Signal, 1)
    signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM)
    <-sc

    generateSessionSecret()

    dg.Close()
}

func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
    if m.Author.ID == s.State.User.ID {
        return
    }

    _, err := db.Exec(`
        INSERT INTO messages (id, author_id, content) 
        VALUES (?, ?, ?)
    `, m.ID, m.Author.ID, m.Content)
    
    if err != nil {
        log.Printf("Error storing message: %v", err)
        return
    }

    if m.Content == "/protected" {
        s.ChannelMessageSend(m.ChannelID, "Protection mode is active! 🛡️")
    }

    if m.Content == ",help" {
        helpMessage := fmt.Sprintf("<@%s> https://aware.wtf/help", m.Author.ID)
        s.ChannelMessageSend(m.ChannelID, helpMessage)
    }

    if m.Content == ",ping" {
        msgTime := m.Timestamp
        
        latency := time.Since(msgTime).Milliseconds()
        wsLatency := s.HeartbeatLatency().Milliseconds()
        
        pingMessage := fmt.Sprintf("🏓 Pong!\n⏱️ API Latency: %dms\n📡 WebSocket: %dms", latency, wsLatency)
        s.ChannelMessageSend(m.ChannelID, pingMessage)
    }
    
}