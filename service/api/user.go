package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	// Percorsi corretti
	"github.com/pioloLlanos/Wasa/service/api/reqcontext"
	"github.com/pioloLlanos/Wasa/service/database"
	
	"github.com/julienschmidt/httprouter"
)

// setUserNameRequest è la struttura per deserializzare il body della richiesta PUT /me/name
type setUserNameRequest struct {
	NewName string `json:"name"`
}

// setMyUserName implementa l'handler PUT /me/name per aggiornare il nome utente
func (rt *_router) setMyUserName(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ctx reqcontext.RequestContext) {
	userID := ctx.UserID

	// 1. Leggi il body della richiesta
	var req setUserNameRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		ctx.Logger.WithError(err).Error("Error decoding request body")
		rt.writeJSON(w, http.StatusBadRequest, nil)
		return
	}

	// 2. Validazione: il nome non deve essere vuoto
	if req.NewName == "" {
		ctx.Logger.Error("New name is required")
		rt.writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Il nome non può essere vuoto"})
		return
	}

	// 3. Logica di business: Aggiorna il nome nel database
	// Presupponendo che rt.db.SetUserName sia definito nell'interfaccia AppDatabase
	if err := rt.db.SetUserName(userID, req.NewName); err != nil {
		if errors.Is(err, database.ErrUserNotFound) {
			rt.writeJSON(w, http.StatusNotFound, nil) // Non dovrebbe accadere se l'autenticazione funziona
		} else if errors.Is(err, database.ErrNameAlreadyTaken) {
			rt.writeJSON(w, http.StatusConflict, map[string]string{"error": "Nome utente già in uso"})
		} else {
			ctx.Logger.WithError(err).Error("Database error during SetUserName")
			rt.writeJSON(w, http.StatusInternalServerError, nil)
		}
		return
	}

	// 4. Successo: 204 No Content
	w.WriteHeader(http.StatusNoContent)
}
// setMyPhoto implementa l'handler PUT /me/photo per aggiornare la foto profilo
func (rt *_router) setMyPhoto(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ctx reqcontext.RequestContext) {
	userID := ctx.UserID

	// 1. Parsa il Form Multipart (limite a 5MB per la foto)
	err := r.ParseMultipartForm(5 << 20) // 5MB limit
	if err != nil {
		ctx.Logger.WithError(err).Error("Error parsing multipart form")
		rt.writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Errore nel parsing del form. Max 5MB"})
		return
	}

	// 2. Estrai il file "image"
	file, fileHeader, err := r.FormFile("image")
	if err != nil {
		// Se l'errore è dovuto a 'missing file', restituisci 400 Bad Request
		if errors.Is(err, http.ErrMissingFile) {
			rt.writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Il campo 'image' è richiesto"})
			return
		}
		// Altrimenti, è un errore generico del server
		ctx.Logger.WithError(err).Error("Error reading file from multipart form")
		rt.writeJSON(w, http.StatusInternalServerError, nil)
		return
	}
	defer file.Close()

	// 3. Logica di upload e aggiornamento URL
	// NOTA: user.go ha già una funzione rt.simulateFileUpload definita in base agli snippet forniti.
	photoURL, err := rt.simulateFileUpload(userID, fileHeader.Filename) 
	if err != nil {
		ctx.Logger.WithError(err).Error("Error saving file")
		rt.writeJSON(w, http.StatusInternalServerError, nil)
		return
	}

	// 4. Aggiorna l'URL della foto nel database
	if err := rt.db.SetUserPhotoURL(userID, photoURL); err != nil {
		// Gestione dell'errore (es. se l'utente non esiste, anche se improbabile qui)
		if errors.Is(err, sql.ErrNoRows) {
			rt.writeJSON(w, http.StatusNotFound, nil) 
			return
		}
		ctx.Logger.WithError(err).Error("Database error during SetUserPhotoURL")
		rt.writeJSON(w, http.StatusInternalServerError, nil)
		return
	}

	// 5. Successo: 200 OK (OpenAPI specifica 200 OK senza body)
	w.WriteHeader(http.StatusOK)
}




// searchUsers implementa l'handler GET /users/search
func (rt *_router) searchUsers(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ctx reqcontext.RequestContext) {
	// 1. Ottieni il parametro di query "q" (query string)
	query := r.URL.Query().Get("name")

	if query == "" {
		// Se la query è vuota, restituisce una lista vuota o un errore 400
		rt.writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Il parametro di ricerca 'name' è richiesto."})
		return
	}

	// 2. Logica di business: Cerca gli utenti nel database
	// Assumiamo che SearchUsers restituisca database.User
	users, err := rt.db.SearchUsers(query) 

	if err != nil {
		// Non è un errore critico se non trova utenti (restituirà una lista vuota)
		ctx.Logger.WithError(err).Error("Database error during SearchUsers")
		rt.writeJSON(w, http.StatusInternalServerError, nil)
		return
	}

	// 3. Successo: 200 OK con la lista di utenti
	rt.writeJSON(w, http.StatusOK, users)
}
