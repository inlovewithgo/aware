package dashboard

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	"github.com/ravener/discord-oauth2"
	"golang.org/x/oauth2"
)

var (
	store        = sessions.NewCookieStore([]byte(os.Getenv("SESSION_SECRET")))
	oauthConfig  *oauth2.Config
)

type User struct {
	ID            string `json:"id"`
	Username      string `json:"username"`
	Discriminator string `json:"discriminator"`
	Avatar        string `json:"avatar"`
}

func Initialize(r *mux.Router) {
	oauthConfig = &oauth2.Config{
		RedirectURL:  os.Getenv("REDIRECT_URL"),
		ClientID:     os.Getenv("DISCORD_APP_ID"),
		ClientSecret: os.Getenv("DISCORD_SECRET"),
		Scopes:       []string{discord.ScopeIdentify},
		Endpoint:     discord.Endpoint,
	}

	r.HandleFunc("/login", handleLogin)
	r.HandleFunc("/callback", handleCallback)
	r.HandleFunc("/logout", handleLogout)
	r.HandleFunc("/dashboard", requireAuth(handleDashboard))
}

func handleLogin(w http.ResponseWriter, r *http.Request) {
	state := generateRandomState()
	
	session, _ := store.Get(r, "discord-auth")
	session.Values["state"] = state
	session.Save(r, w)
	
	url := oauthConfig.AuthCodeURL(state)
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

func handleCallback(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "discord-auth")
	state := session.Values["state"]
	
	if r.URL.Query().Get("state") != state {
		http.Error(w, "Invalid state parameter", http.StatusBadRequest)
		return
	}
	
	code := r.URL.Query().Get("code")
	token, err := oauthConfig.Exchange(r.Context(), code)
	if err != nil {
		http.Error(w, "Failed to exchange token: "+err.Error(), http.StatusInternalServerError)
		return
	}
	
	client := oauthConfig.Client(r.Context(), token)
	resp, err := client.Get("https://discord.com/api/users/@me")
	if err != nil {
		http.Error(w, "Failed to get user info: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()
	
	var user User
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		http.Error(w, "Failed to parse user info: "+err.Error(), http.StatusInternalServerError)
		return
	}
	
	session.Values["user"] = user
	session.Values["authenticated"] = true
	session.Save(r, w)
	
	http.Redirect(w, r, "/dashboard", http.StatusTemporaryRedirect)
}

func handleLogout(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "discord-auth")
	session.Values["authenticated"] = false
	session.Values["user"] = nil
	session.Save(r, w)
	
	http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
}

func handleDashboard(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "discord-auth")
	user := session.Values["user"].(User)
	
	fmt.Fprintf(w, "Welcome to your dashboard, %s!", user.Username)
}

func requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		session, _ := store.Get(r, "discord-auth")
		auth, ok := session.Values["authenticated"].(bool)
		
		if !ok || !auth {
			http.Redirect(w, r, "/login", http.StatusTemporaryRedirect)
			return
		}
		
		next(w, r)
	}
}

func generateRandomState() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}
