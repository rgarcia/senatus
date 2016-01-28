package main

import (
	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	env "github.com/segmentio/go-env"
	"github.com/templaedhel/senatus/db"
	"github.com/templaedhel/senatus/handler"
	"net/http"
	"os"
	"runtime"
)

func sessionMiddleware(h *handler.Handler, store sessions.Store, routeHandler func(http.ResponseWriter, *http.Request, *db.User)) func(http.ResponseWriter, *http.Request) {
	return func(res http.ResponseWriter, req *http.Request) {
		session, err := store.Get(req, "session")
		if err != nil {
			http.SetCookie(res, &http.Cookie{Name: "session", MaxAge: -1, Path: "/"})
			routeHandler(res, req, nil)
			return
		}
		id, ok := session.Values["id"].(string)
		name, ok := session.Values["name"].(string)
		if id != "" && name != "" && ok {
			user := db.User{id, name}
			routeHandler(res, req, &user)
		} else {
			routeHandler(res, req, nil)
		}
	}
}

func main() {

	requiredEnv := []string{"PORT", "MONGO_URL", "CLIENT_ID", "CLIENT_SECRET", "REDIRECT_URI", "SESSION_SECRET"}
	for _, envVar := range requiredEnv {
		env.MustGet(envVar)
	}

	runtime.GOMAXPROCS(runtime.NumCPU())

	database, err := db.NewMongoDB(os.Getenv("MONGO_URL"))
	if err != nil {
		panic("Error connecting to mongo")
	}

	sessionStore := sessions.NewCookieStore([]byte(os.Getenv("SESSION_SECRET")))

	h := handler.NewHandler(database, sessionStore)

	r := mux.NewRouter()
	r.HandleFunc("/", sessionMiddleware(h, sessionStore, h.IndexHandler)).Methods("GET")
	r.HandleFunc("/topics", sessionMiddleware(h, sessionStore, h.ViewTopicsHandler)).Methods("GET")
	r.HandleFunc("/topics/new", sessionMiddleware(h, sessionStore, h.NewTopicGetHandler)).Methods("GET")
	r.HandleFunc("/topics/new", sessionMiddleware(h, sessionStore, h.NewTopicPostHandler)).Methods("POST")
	r.HandleFunc("/topics/{id}", sessionMiddleware(h, sessionStore, h.ViewTopicHandler)).Methods("GET")
	r.HandleFunc("/topics/{id}/question/{question_id}/vote", sessionMiddleware(h, sessionStore, h.VoteForQuestionHandler)).Methods("POST")
	r.HandleFunc("/topics/{id}/question/{question_id}/unvote", sessionMiddleware(h, sessionStore, h.UnvoteForQuestionHandler)).Methods("POST")
	r.HandleFunc("/login/{id}", h.LoginHandler).Methods("GET")
	r.HandleFunc("/login", h.LoginHandler).Methods("GET")
	r.HandleFunc("/topics/{id}/question", sessionMiddleware(h, sessionStore, h.NewQuestionHandler)).Methods("POST")
	r.HandleFunc("/oauth", h.OAuthCallback).Methods("GET")
	r.HandleFunc("/logout", h.LogoutHandler).Methods("GET")
	r.HandleFunc("/favicon.ico", func(res http.ResponseWriter, req *http.Request) {
		res.Header().Set("Content-Type", "image/x-icon")
		http.ServeFile(res, req, "./public/favicon.ico")
	}).Methods("GET")
	r.PathPrefix("/public/").Handler(http.StripPrefix("/public/", http.FileServer(http.Dir("./public"))))
	r.HandleFunc("/{url:.*}", h.NotFoundHandler)

	http.Handle("/", r)
	http.ListenAndServe(":"+os.Getenv("PORT"), nil)
}
