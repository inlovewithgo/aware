package main

import (
    "database/sql"
    _ "modernc.org/sqlite"
    "os"
    "log"
)

var db *sql.DB

func initDB() {
    var err error
    
    cwd, _ := os.Getwd()
    log.Printf("Current working directory: %s", cwd)
    
    dbPath := "./main.db"
    log.Printf("Opening database at: %s", dbPath)
    
    db, err = sql.Open("sqlite", dbPath)
    if err != nil {
        log.Fatal("Failed to open database:", err)
    }
    
    if err = db.Ping(); err != nil {
        log.Fatal("Database connection failed:", err)
    }
    
    log.Println("Database connection established successfully")

    _, err = db.Exec(`
    CREATE TABLE IF NOT EXISTS antinuke_whitelist (
        guild_id TEXT,
        user_id TEXT,
        added_by TEXT,
        added_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
        PRIMARY KEY (guild_id, user_id)
    )
`)
if err != nil {
    log.Fatal("Failed to create whitelist table:", err)
}
    
    _, err = db.Exec(`
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
        log.Fatal("Failed to create tables:", err)
    }
    
    log.Println("Database tables initialized")
}
