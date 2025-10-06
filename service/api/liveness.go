package api

import (
	"github.com/julienschmidt/httprouter"
	"net/http"
)

// liveness è un gestore HTTP che verifica lo stato del server API.
func (rt *_router) liveness(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	// Esegui il check sul database
	if err := rt.db.Ping(); err != nil {
		w.WriteHeader(http.StatusInternalServerError) // 500 se il DB non risponde
		return
	}
    // OK
    w.WriteHeader(http.StatusOK)
}