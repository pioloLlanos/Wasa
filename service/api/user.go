package api

import (
	"database/sql" // Necessario per sql.ErrNoRows in setMyPhoto
	"errors"
	"net/http"

	// Percorsi corretti
	"github.com/julienschmidt/httprouter"
	"github.com/pioloLlanos/Wasa/service/api/reqcontext"
	"github.com/pioloLlanos/Wasa/service/database"
)

// setUserNameRequest è la struttura per deserializzare il body della richiesta PUT /me/name
type setUserNameRequest struct {
	NewName string `json:"name"` // Il campo corretto da OpenAPI (assumo)
}

// setMyUserName implementa l'handler PUT /me/name per aggiornare il nome utente
func (rt *_router) setMyUserName(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ctx reqcontext.RequestContext) {
	userID := ctx.UserID

	// 1. Leggi il body della richiesta
	var req setUserNameRequest
	// Usa rt.decodeJSON con 'w' come primo argomento
	if err := rt.decodeJSON(w, r, &req); err != nil {
		rt.writeJSON(w, http.StatusBadRequest, nil)
		return
	}

	// 2. Validazione: il nome non deve essere vuoto
	if req.NewName == "" {
		rt.writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Il nome non può essere vuoto"})
		return
	}

	// 3. Logica di business: Aggiorna il nome nel database
	if err := rt.db.SetMyUserName(userID, req.NewName); err != nil {
		if errors.Is(err, database.AppErrorNomeGiaInUso) {
			rt.writeJSON(w, http.StatusConflict, map[string]string{"error": "Nome utente già in uso"})
			return
		}
		if errors.Is(err, sql.ErrNoRows) {
			rt.writeJSON(w, http.StatusNotFound, nil)
			return
		}
		ctx.Logger.WithError(err).Error("Database error during SetMyUserName")
		rt.writeJSON(w, http.StatusInternalServerError, nil)
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
		rt.writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Errore nel parsing del form. Max 5MB"})
		return
	}

	// 2. Estrai il file "image"
	file, fileHeader, err := r.FormFile("image")
	if err != nil {
		if errors.Is(err, http.ErrMissingFile) {
			rt.writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Il campo 'image' è richiesto"})
			return
		}
		rt.writeJSON(w, http.StatusInternalServerError, nil)
		return
	}
	defer file.Close()

	// 3. Logica di upload e aggiornamento URL
	// CORREZIONE: rt.simulateFileUpload richiede 3 argomenti: (convID, userID, filename).
	photoURL, err := rt.simulateFileUpload(0, userID, fileHeader.Filename)
	if err != nil {
		ctx.Logger.WithError(err).Error("Error saving file")
		rt.writeJSON(w, http.StatusInternalServerError, nil)
		return
	}

	// 4. Aggiorna l'URL della foto nel database
	if err := rt.db.SetUserPhotoURL(userID, photoURL); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			rt.writeJSON(w, http.StatusNotFound, nil)
			return
		}
		ctx.Logger.WithError(err).Error("Database error during SetUserPhotoURL")
		rt.writeJSON(w, http.StatusInternalServerError, nil)
		return
	}

	// 5. Successo: 200 OK
	w.WriteHeader(http.StatusOK)
}

// searchUsers implementa l'handler GET /users/search
func (rt *_router) searchUsers(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ctx reqcontext.RequestContext) {
	// 1. Ottieni il parametro di query "name" (query string)
	query := r.URL.Query().Get("name")

	if query == "" {
		// Se la query è vuota, restituisce 400 Bad Request
		rt.writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Il parametro di ricerca 'name' è richiesto."})
		return
	}

	// 2. Logica di business: Cerca gli utenti nel database
	users, err := rt.db.SearchUsers(query)

	if err != nil {
		ctx.Logger.WithError(err).Error("Database error during SearchUsers")
		rt.writeJSON(w, http.StatusInternalServerError, nil)
		return
	}

	// 3. Successo: 200 OK con la lista di utenti
	rt.writeJSON(w, http.StatusOK, users)
}