# Aware Discord Bot

**Aware** is a Discord security bot written in **Go** that provides anti-nuke protection, logging features, a session-based dashboard, and more.

---

## 🛡 Features

- ⚔️ Anti-nuke (ban/kick prevention, whitelist support)
- 📋 Logging (join, leave, and config logs)
- 🧠 Simple session-based dashboard
- 🔐 Secure token loading via `.env`
- ⚡ Fast and minimal — pure Go with no bloated dependencies

---

## 📁 Project Structure

```bash
├── antinuke/
│ ├── antinuke.go
│ ├── events.go
│ └── whitelist.go
├── dashboard/
│ ├── dashboard.go
│ └── session-gen.go
├── logging/
│ ├── config.go
│ ├── joinLogs.go
│ └── leaveLogs.go
├── main.go
├── database.go
├── main.db
├── sql.DB
├── go.mod
├── go.sum
├── .env
└── .gitignore
```
## 🚀 Getting Started

### 1. Clone the repo
```bash
git clone https://github.com/inlovewithgo/aware.git
cd aware
```

### 2. Set up the environment
Create a `.env` file:
```env
DISCORD_APP_ID=
DISCORD_SECRET=
REDIRECT_URL=http://localhost:8080/callback
SESSION_SECRET=
PORT=8080
```

### 3. Install dependencies
```bash
go mod tidy
```

### 4. Run the bot
```bash
go run main.go
```

## 📦 Dependencies
- `github.com/bwmarrin/discordgo` – Discord API library for Go
- `github.com/joho/godotenv` – Loads .env into os.Getenv
- `Standard Go packages` (net/http, fmt, os, etc.)

## 🧠 Notes
- Make sure `.env` is in your .gitignore (it is by default)
- SQLite is used for local storage via main.db
- This project is modular and clean for easy contribution or scaling
