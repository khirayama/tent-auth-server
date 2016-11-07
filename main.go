package main

import (
	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
	"github.com/markbates/goth"
	"github.com/markbates/goth/gothic"
	"github.com/markbates/goth/providers/twitter"
	"log"
	"net/http"
	"os"
)

var store = sessions.NewCookieStore([]byte(os.Getenv("SESSION_SECRET")))
var db, _ = gorm.Open("sqlite3", "development.db")

const SessionName = "_tent_session"

func init() {
	goth.UseProviders(
		twitter.New(os.Getenv("TWITTER_KEY"),
			os.Getenv("TWITTER_SECRET"),
			"http://localhost:8080/auth/callback?provider=twitter"),
	)
}

func main() {
	db.AutoMigrate(&User{})

	r := mux.NewRouter()

	r.HandleFunc("/auth", gothic.BeginAuthHandler)
	r.HandleFunc("/auth/callback", sessionCreateHandler)
	r.HandleFunc("/logout", logoutHandler)

	r.HandleFunc("/sessions/{id}", authenticationHandler)

	http.Handle("/", r)
	http.ListenAndServe(":8080", nil)
}

// handlers
func sessionCreateHandler(w http.ResponseWriter, r *http.Request) {
	user, err := gothic.CompleteUserAuth(w, r)
	if err != nil {
		panic(err)
	}

	// call API server to find user
	// Move to API server - start
	currentUser := &User{
		Provider: user.Provider,
		Uid:      user.UserID,
		Nickname: user.NickName,
		ImageUrl: user.AvatarURL,
	}
	db.Where("provider = ? AND uid = ?", user.Provider, user.UserID).Find(&currentUser)
	if db.NewRecord(currentUser) {
		db.Create(currentUser)
	}
	// Move to API server - end

	// create session
	session_ := &Session{
		UserID: currentUser.ID,
	}
	db.Create(session)
	session, err := store.Get(r, SessionName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	session.Values["session_id"] = session_.ID
	session.Save(r, w)

	http.Redirect(w, r, "/", http.StatusFound)
}

func logoutHandler(w http.ResponseWriter, r *http.Request) {
	session, err := store.Get(r, SessionName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	session.Values["user_id"] = nil
	session.Save(r, w)

	http.Redirect(w, r, "/", http.StatusFound)
}

func authenticationHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	sessionId := vars["id"]

	var session string
	db.Where("id = ?", sessionId).Find(&session)

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(session)
}

func authenticate(w http.ResponseWriter, r *http.Request) {
	session, err := store.Get(r, SessionName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	userId := session.Values["user_id"]
	if userId == nil {
		log.Print(userId)
		http.Redirect(w, r, "/", http.StatusFound)
	}
}

// models
type User struct {
	gorm.Model
	Provider string
	Uid      string
	Nickname string
	ImageUrl string
}

type Session struct {
	gorm.Model
	UserID string
	// Expires
}
