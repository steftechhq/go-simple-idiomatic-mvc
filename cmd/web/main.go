package main

import (
	"crypto/tls"
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
	debug    bool // Add a new debug field.
	logger   *slog.Logger
	snippets models.SnippetModelInterface
	users    models.UserModelInterface

	templateCache  map[string]*template.Template
	formDecoder    *form.Decoder
	sessionManager *scs.SessionManager
}

func main() {

	var cfg config

	flag.StringVar(&cfg.addr, "addr", ":4000", "HTTP network address")
	flag.StringVar(&cfg.staticDir, "static-dir", "./ui/static", "Path to static assets")
	debug := flag.Bool("debug", false, "Enable debug mode")
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
	sessionManager.Cookie.Secure = true

	app := &application{
		debug:    *debug,
		logger:   logger,
		snippets: &models.SnippetModel{DB: db},
		users:    &models.UserModel{DB: db},

		templateCache:  templateCache,
		formDecoder:    formDecoder,
		sessionManager: sessionManager,
	}

	tlsConfig := &tls.Config{
		CurvePreferences: []tls.CurveID{tls.X25519, tls.CurveP256},
		MinVersion:       tls.VersionTLS10,
		MaxVersion:       tls.VersionTLS12,
	}

	srv := &http.Server{
		Addr:      cfg.addr,
		Handler:   app.routes(),
		ErrorLog:  slog.NewLogLogger(logger.Handler(), slog.LevelError),
		TLSConfig: tlsConfig,

		IdleTimeout:    time.Minute,
		ReadTimeout:    5 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 524288,
	}

	logger.Info("starting server", "addr", cfg.addr)

	// mainErr := srv.ListenAndServe()
	mainErr := srv.ListenAndServeTLS("../../tls/cert.pem", "../../tls/key.pem")
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
