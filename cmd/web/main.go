package main

import (
	"database/sql"
	"flag"
	"html/template"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/go-playground/form/v4" // New import
	_ "github.com/go-sql-driver/mysql" // New import

	"github.com/alexedwards/scs/mysqlstore" // New import
	"github.com/alexedwards/scs/v2"         // New import

	"snippetbox.steftech.com/internal/models"
)

type config struct {
	addr      string
	staticDir string
}

type application struct {
	logger         *slog.Logger
	snippets       *models.SnippetModel
	templateCache  map[string]*template.Template
	formDecoder    *form.Decoder
	sessionManager *scs.SessionManager
}

func main() {

	var cfg config

	flag.StringVar(&cfg.addr, "addr", ":4000", "HTTP network address")
	flag.StringVar(&cfg.staticDir, "static-dir", "./ui/static", "Path to static assets")

	flag.Parse()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	// Initialize a new template cache...
	templateCache, err := newTemplateCache()
	if err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}

	dsn := flag.String("dsn", "app_user:app_password@tcp(127.0.0.1:3306)/app_db?parseTime=true", "MySQL data source name")
	db, err := openDB(*dsn)
	if err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}
	defer db.Close()

	formDecoder := form.NewDecoder()

	sessionManager := scs.New()
	sessionManager.Store = mysqlstore.New(db)
	sessionManager.Lifetime = 12 * time.Hour

	app := &application{
		logger:         logger,
		snippets:       &models.SnippetModel{DB: db},
		templateCache:  templateCache,
		formDecoder:    formDecoder,
		sessionManager: sessionManager,
	}

	// mux := http.NewServeMux()

	// fileServer := http.FileServer(http.Dir("../../ui/static/"))
	// mux.Handle("GET /static/", http.StripPrefix("/static", fileServer))

	// mux.HandleFunc("GET /{$}", app.home)
	// mux.HandleFunc("GET /snippet/view/{id}/{$}", app.snippetView)
	// mux.HandleFunc("GET /snippet/create", app.snippetCreate)
	// mux.HandleFunc("POST /snippet/create", app.snippetCreatePost)

	logger.Info("starting server", "addr", cfg.addr)

	mainErr := http.ListenAndServe(cfg.addr, app.routes())
	logger.Error(mainErr.Error())
	os.Exit(1)
}

func openDB(dsn string) (*sql.DB, error) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}

	err = db.Ping()
	if err != nil {
		db.Close()
		return nil, err
	}

	return db, nil
}
