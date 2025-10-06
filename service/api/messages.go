package api

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	// Assumi percorsi corretti per il tuo progetto
	"github.com/pioloLlanos/Wasa/service/api/reqcontext"
	"github.com/pioloLlanos/Wasa/service/database"
	"github.com/julienschmidt/httprouter"
)

// forwardMessageRequest è la struttura per l'inoltro
type forwardMessageRequest struct {
	ConversationIDs []uint64 `json:"conversationIds"`
}

// reactionRequest è la struttura per le reazioni
type reactionRequest struct {
	Emoji string `json:"emoji"`
}

// deleteMessage implementa DELETE /messages/{messageId}
func (rt *_router) deleteMessage(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ctx reqcontext.RequestContext) {
	userID := ctx.UserID

	// 1. Ottieni l'ID del messaggio dal path
	messageID, err := strconv.ParseUint(ps.ByName("messageId"), 10, 64)
	if err != nil {
		rt.writeJSON(w, http.StatusBadRequest, nil)
		return
	}

	// 2. Chiama la logica del DB (verifica che userID sia il mittente)
	err = rt.db.DeleteMessage(messageID, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// Messaggio non trovato o l'utente non è il mittente (risposta 404)
			rt.writeJSON(w, http.StatusNotFound, nil) 
			return
		}
		ctx.Logger.WithError(err).Error("Database error during DeleteMessage")
		rt.writeJSON(w, http.StatusInternalServerError, nil)
		return
	}

	// 3. Successo
	w.WriteHeader(http.StatusNoContent) // 204 No Content è più appropriato per DELETE
}

// forwardMessage implementa POST /messages/{messageId}/forward
func (rt *_router) forwardMessage(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ctx reqcontext.RequestContext) {
	userID := ctx.UserID

	// 1. Ottieni l'ID del messaggio da inoltrare
	messageID, err := strconv.ParseUint(ps.ByName("messageId"), 10, 64)
	if err != nil {
		rt.writeJSON(w, http.StatusBadRequest, nil)
		return
	}

	// 2. Decodifica il body JSON con la lista di destinazioni
	var req forwardMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		rt.writeJSON(w, http.StatusBadRequest, nil)
		return
	}
	if len(req.ConversationIDs) == 0 {
		rt.writeJSON(w, http.StatusBadRequest, map[string]string{"error": "conversationIds non può essere vuoto"})
		return
	}

	// 3. Logica di business: Inoltra il messaggio a ciascuna conversazione
	var successfulForwards []uint64 // Per tracciare i nuovi ID dei messaggi
	var firstError error

	for _, targetConvID := range req.ConversationIDs {
		// La funzione del DB deve verificare la membership di userID in targetConvID
		newMsgID, err := rt.db.ForwardMessage(messageID, userID, targetConvID)
		
		if err != nil {
			if firstError == nil {
				firstError = err 
			}
			continue
		}
		successfulForwards = append(successfulForwards, newMsgID)
	}

	if len(successfulForwards) == 0 {
		if firstError != nil {
			// Se fallisce l'inoltro in tutte le chat
			if errors.Is(firstError, database.AppErrorUserNotMember) {
				rt.writeJSON(w, http.StatusForbidden, nil) // 403 Forbidden
				return
			}
			ctx.Logger.WithError(firstError).Error("Database error during ForwardMessage")
			rt.writeJSON(w, http.StatusInternalServerError, nil)
			return
		}
		// Caso in cui la lista era valida ma il messaggio originale non è stato trovato (o altra validazione interna)
		rt.writeJSON(w, http.StatusNotFound, nil) 
		return
	}

	// 4. Successo (restituisce la lista degli ID dei messaggi inoltrati con successo)
	rt.writeJSON(w, http.StatusOK, map[string][]uint64{"forwardedMessageIds": successfulForwards})
}

// commentMessage implementa POST /messages/{messageId}/reactions
func (rt *_router) commentMessage(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ctx reqcontext.RequestContext) {
	userID := ctx.UserID

	// 1. Ottieni l'ID del messaggio
	messageID, err := strconv.ParseUint(ps.ByName("messageId"), 10, 64)
	if err != nil {
		rt.writeJSON(w, http.StatusBadRequest, nil)
		return
	}

	// 2. Decodifica il body JSON con l'emoji
	var req reactionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		rt.writeJSON(w, http.StatusBadRequest, nil)
		return
	}
	if req.Emoji == "" {
		rt.writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Il campo 'emoji' è richiesto"})
		return
	}

	// 3. Chiama la logica del DB (aggiunge o aggiorna la reazione)
	err = rt.db.AddReaction(messageID, userID, req.Emoji)
	if err != nil {
		if errors.Is(err, database.AppErrorConversationNotFound) { // Se il messaggio non esiste/non è accessibile
			rt.writeJSON(w, http.StatusNotFound, nil) 
			return
		}
		ctx.Logger.WithError(err).Error("Database error during AddReaction")
		rt.writeJSON(w, http.StatusInternalServerError, nil)
		return
	}

	// 4. Successo
	w.WriteHeader(http.StatusOK) 
}

// uncommentMessage implementa DELETE /messages/{messageId}/reactions
func (rt *_router) uncommentMessage(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ctx reqcontext.RequestContext) {
	userID := ctx.UserID
	
	// 1. Ottieni l'ID del messaggio
	messageID, err := strconv.ParseUint(ps.ByName("messageId"), 10, 64)
	if err != nil {
		rt.writeJSON(w, http.StatusBadRequest, nil)
		return
	}

	// 2. Chiama la logica del DB (rimuove la reazione)
	err = rt.db.RemoveReaction(messageID, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// Reazione non trovata (o messaggio non trovato, ma non è un errore per l'utente)
			rt.writeJSON(w, http.StatusNotFound, nil) 
			return
		}
		ctx.Logger.WithError(err).Error("Database error during RemoveReaction")
		rt.writeJSON(w, http.StatusInternalServerError, nil)
		return
	}

	// 3. Successo
	w.WriteHeader(http.StatusOK) 
}