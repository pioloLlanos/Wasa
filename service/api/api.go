/*
Package api exposes the main API engine. All HTTP APIs are handled here - so-called "business logic" should be here, or
in a dedicated package (if that logic is complex enough).

To use this package, you should create a new instance with New() passing a valid Config. The resulting Router will have
the Router.Handler() function that returns a handler that can be used in a http.Server (or in other middlewares).

Example:

    // Create the API router
    apirouter, err := api.New(api.Config{
        Logger:   logger,
        Database: appdb,
    })
    if err != nil {
        logger.WithError(err).Error("error creating the API server instance")
        return fmt.Errorf("error creating the API server instance: %w", err)
    }
    // NOTA: La chiamata a apirouter.Handler() registra automaticamente tutte le rotte e applica il CORS.
    router := apirouter.Handler()

    // ... other stuff here, like middleware chaining, etc.

    // Create the API server
    apiserver := http.Server{
        Addr:              cfg.Web.APIHost,
        Handler:           router,
        ReadTimeout:       cfg.Web.ReadTimeout,
        ReadHeaderTimeout: cfg.Web.ReadTimeout,
        WriteTimeout:      cfg.Web.WriteTimeout,
    }

    // Start the service listening for requests in a separate goroutine
    apiserver.ListenAndServe()

See the `main.go` file inside the `cmd/webapi` for a full usage example.
*/
package api

import (
    "errors"
    "net/http"

    "github.com/julienschmidt/httprouter"
    "github.com/pioloLlanos/Wasa/service/database"
    "github.com/sirupsen/logrus"
)

// Config is used to provide dependencies and configuration to the New function.
type Config struct {
    // Logger where log entries are sent
    Logger logrus.FieldLogger

    // Database is the instance of database.AppDatabase where data are saved
    Database database.AppDatabase
}

// Router is the package API interface representing an API handler builder
type Router interface {
    // Handler returns an HTTP handler for APIs provided in this package
    Handler() http.Handler

    // Close terminates any resource used in the package
    Close() error
}

// New returns a new Router instance
func New(cfg Config) (Router, error) {
    // Check if the configuration is correct
    if cfg.Logger == nil {
        return nil, errors.New("logger is required")
    }
    if cfg.Database == nil {
        return nil, errors.New("database is required")
    }

    // Create a new router where we will register HTTP endpoints.
    router := httprouter.New()
    router.RedirectTrailingSlash = false
    router.RedirectFixedPath = false

    rt := &_router{
        router:     router,
        baseLogger: cfg.Logger,
        db:         cfg.Database,
    }

    return rt, nil
}

type _router struct {
    router *httprouter.Router

    // baseLogger is a logger for non-requests contexts.
    baseLogger logrus.FieldLogger

    db database.AppDatabase
}

// Handler registra le rotte chiamando il metodo routes() (definito in api-handler.go)
// e poi avvolge il router con il middleware CORS.
func (rt *_router) Handler() http.Handler {
    // 1. Registra tutte le rotte. Il metodo 'routes' è definito in api-handler.go.
    rt.routes() 

    // 2. Aggiungiamo l'handler CORS al di sopra del router completo.
    return rt.handleCORS(rt.router)
}

// handleCORS è il middleware che gestisce CORS per tutte le richieste.
func (rt *_router) handleCORS(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Permetti tutte le origini
        w.Header().Set("Access-Control-Allow-Origin", "*")
        
        // Definisci i metodi e gli header consentiti
        w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
        w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
        
        // Imposta Max-Age
        w.Header().Set("Access-Control-Max-Age", "1") 

        // Gestisce la richiesta pre-flight OPTIONS
        if r.Method == "OPTIONS" {
            w.WriteHeader(http.StatusOK)
            return
        }

        // Passa la richiesta all'handler successivo (il router)
        next.ServeHTTP(w, r)
    })
}

// Il metodo Close() è implementato in shutdown.go, non qui, per risolvere i conflitti.
// Il metodo routes() è implementato in api-handler.go, non qui, per risolvere i conflitti.
