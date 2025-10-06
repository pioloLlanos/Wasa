package api

import (
	"database/sql"
	"errors"
	"net/http"
	
	"github.com/pioloLlanos/Wasa/service/database" 
	"github.com/julienschmidt/httprouter"
)

// Definisce la struttura attesa nel corpo della richiesta POST /session
type loginRequest struct {
	Name string `json:"name"`
}

// Definisce la struttura della risposta JSON (ID utente)
type loginResponse struct {
	Identifier uint64 `json:"identifier"`
}

// doLogin implementa l'handler POST /session (Login/Registrazione)
// Questo handler NON usa rt.wrap (quindi non necessita di reqcontext.RequestContext).
func (rt *_router) doLogin(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	var req loginRequest
	
	// 1. Decodifica la richiesta JSON
	if err := rt.decodeJSON(r, &req); err != nil {
		rt.writeJSON(w, http.StatusBadRequest, nil)
		return
	}
	
	// 2. Tenta di trovare l'utente per nome
	id, err := rt.db.GetUserByName(req.Name)
	
	if err == nil {
		// CASO 1: Utente Trovato (Login)
		rt.baseLogger.WithField("name", req.Name).Info("User logged in successfully")
		response := loginResponse{Identifier: id}
		rt.writeJSON(w, http.StatusCreated, response)
		return
	}
	
	if !errors.Is(err, sql.ErrNoRows) {
		// CASO 2: Errore del Database non dovuto a "Non Trovato"
		rt.baseLogger.WithError(err).Error("Database error during GetUserByName")
		rt.writeJSON(w, http.StatusInternalServerError, nil)
		return
	}

	// CASO 3: Utente Non Trovato (Registrazione)
	
	// Tenta di creare il nuovo utente
	newID, err := rt.db.CreateUser(req.Name)
	if err != nil {
		if errors.Is(err, database.AppErrorNomeGiaInUso) {
			rt.writeJSON(w, http.StatusConflict, nil) // 409 Conflict
			return
		}
		rt.baseLogger.WithError(err).Error("Database error during CreateUser")
		rt.writeJSON(w, http.StatusInternalServerError, nil)
		return
	}
	
	// Successo nella registrazione
	rt.baseLogger.WithField("name", req.Name).WithField("id", newID).Info("New user registered successfully")
	response := loginResponse{Identifier: newID}
	rt.writeJSON(w, http.StatusCreated, response)
}
