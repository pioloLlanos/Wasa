package api

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/julienschmidt/httprouter"

	"github.com/pioloLlanos/Wasa/service/api/reqcontext"
)

// Definizione delle strutture per i body delle richieste (Aggiornate secondo OpenAPI)

// createGroupRequest è il body atteso per POST /groups
type createGroupRequest struct {
	Name      string   `json:"name"`
	MemberIDs []uint64 `json:"memberIds"` // Modificato da 'Members' a 'MemberIDs'
}

// groupIDResponse è la risposta per POST /groups, contiene l'ID del nuovo gruppo
type groupIDResponse struct {
	GroupID uint64 `json:"groupId"` // Modificato da 'GroupID' a 'groupId'
}

// updateNameRequest è il body per PUT /groups/:groupId/name
type updateNameRequest struct {
	NewName string `json:"name"` // Modificato da 'NewName' a 'name'
}

// addMembersRequest è il body per POST /groups/:groupId/members
type addMembersRequest struct {
	UserIDs []uint64 `json:"userIds"` // Modificato per supportare array di ID
}

// simulateFileUpload simula la logica di salvataggio di un'immagine e restituisce un URL fittizio.
func (rt *_router) simulateFileUpload(convID uint64, userID uint64, fileHeader string) (string, error) {
	// Qui andrebbe la logica vera per salvare il file e ottenere un URL pubblico.
	// Per ora, restituiamo un URL segnaposto.
	// Esempio: "https://yourcdn.com/groups/123/photo.jpg"
	return fmt.Sprintf("/groups/photo/%d/%s", convID, strings.Split(fileHeader, ";")[0]), nil
}

// createGroup implementa l'handler POST /groups: crea una nuova conversazione di gruppo.
func (rt *_router) createGroup(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ctx reqcontext.RequestContext) {
	adminID := ctx.UserID // L'ID dell'utente autenticato è l'admin

	var req createGroupRequest
	if err := rt.decodeJSON(r, &req); err != nil {
		rt.writeJSON(w, http.StatusBadRequest, nil)
		return
	}

	if req.Name == "" || len(req.MemberIDs) == 0 {
		rt.writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Group name and memberIds cannot be empty"})
		return
	}

	// Assicura che l'admin sia nel gruppo e gestisce eventuali duplicati nella lista dei membri
	initialMembers := make(map[uint64]bool)
	initialMembers[adminID] = true // L'admin è sempre un membro

	// Aggiunge tutti gli altri membri alla mappa (evita duplicati)
	for _, memberID := range req.MemberIDs {
		initialMembers[memberID] = true
	}

	// Converte la mappa in slice per il DB
	memberList := make([]uint64, 0, len(initialMembers))
	for id := range initialMembers {
		memberList = append(memberList, id)
	}

	// 1. Chiama la logica del DB per creare il gruppo
	groupID, err := rt.db.CreateGroup(adminID, req.Name, memberList)
	if err != nil {
		ctx.Logger.WithError(err).Error("Database error during CreateGroup")
		// Possibile errore di chiave esterna se un membro non esiste
		rt.writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to create group or add members. Check if all member IDs exist."})
		return
	}

	// 2. Successo
	rt.writeJSON(w, http.StatusCreated, groupIDResponse{GroupID: groupID})
}

// setGroupName implementa l'handler PUT /groups/:groupId/name: modifica il nome del gruppo.
func (rt *_router) setGroupName(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ctx reqcontext.RequestContext) {
	adminID := ctx.UserID

	// Estrai l'ID della conversazione dal path
	convIDStr := ps.ByName("groupId")
	convID, err := strconv.ParseUint(convIDStr, 10, 64)
	if err != nil {
		rt.writeJSON(w, http.StatusBadRequest, nil)
		return
	}

	var req updateNameRequest
	if err := rt.decodeJSON(r, &req); err != nil {
		rt.writeJSON(w, http.StatusBadRequest, nil)
		return
	}

	if req.NewName == "" {
		rt.writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Name cannot be empty"})
		return
	}

	// Chiama la logica del DB (che verifica anche i permessi di amministrazione)
	err = rt.db.SetConversationName(convID, adminID, req.NewName)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) || strings.Contains(err.Error(), "l'utente non è membro") {
			rt.writeJSON(w, http.StatusNotFound, nil)
			return
		}
		if strings.Contains(err.Error(), "non è amministratore") {
			rt.writeJSON(w, http.StatusForbidden, nil)
			return
		}
		ctx.Logger.WithError(err).Error("Database error during SetConversationName")
		rt.writeJSON(w, http.StatusInternalServerError, nil)
		return
	}

	w.WriteHeader(http.StatusNoContent) // 204 No Content per successo senza body
}
// setGroupPhoto implementa l'handler PUT /groups/:groupId/photo
func (rt *_router) setGroupPhoto(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ctx reqcontext.RequestContext) {
	userID := ctx.UserID

	// 1. Ottieni ID Gruppo
	convIDStr := ps.ByName("groupId")
	convID, err := strconv.ParseUint(convIDStr, 10, 64)
	if err != nil {
		rt.writeJSON(w, http.StatusBadRequest, nil)
		return
	}
	
	// 2. Parsa il Form Multipart (limite a 5MB per la foto)
	err = r.ParseMultipartForm(5 << 20) // 5MB limit
	if err != nil {
		ctx.Logger.WithError(err).Error("Error parsing multipart form")
		rt.writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Errore nel parsing del form. Max 5MB"})
		return
	}

	// 3. Estrai il file "image"
	file, fileHeader, err := r.FormFile("image")
	if err != nil {
		if errors.Is(err, http.ErrMissingFile) {
			rt.writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Il campo 'image' è richiesto"})
			return
		}
		ctx.Logger.WithError(err).Error("Error reading file from multipart form")
		rt.writeJSON(w, http.StatusInternalServerError, nil)
		return
	}
	defer file.Close()

	// 4. Verifica che l'utente sia membro (o admin, se richiesto dal progetto)
	// Supponiamo che solo i membri possano modificare la foto, ma il DB deve impedire la modifica se non si è admin.
	// La funzione CheckUserMemberStatus deve essere implementata nel DB layer.
	// Per ora, assumiamo che SetGroupPhotoURL si occupi di verificare i permessi.

	// 5. Logica di upload
	photoURL, err := rt.simulateFileUpload(convID, fileHeader.Filename) 
	if err != nil {
		ctx.Logger.WithError(err).Error("Error saving file")
		rt.writeJSON(w, http.StatusInternalServerError, nil)
		return
	}

	// 6. Aggiorna l'URL della foto del gruppo nel database
	// SetGroupPhotoURL deve anche verificare se l'utente è un admin (o avere permessi sufficienti)
	if err := rt.db.SetGroupPhotoURL(convID, userID, photoURL); err != nil {
		if errors.Is(err, database.AppErrorUserNotMember) {
			rt.writeJSON(w, http.StatusForbidden, nil) // 403 Forbidden
			return
		}
		if errors.Is(err, database.AppErrorConversationNotFound) {
			rt.writeJSON(w, http.StatusNotFound, nil) // 404 Not Found
			return
		}
		ctx.Logger.WithError(err).Error("Database error during SetGroupPhotoURL")
		rt.writeJSON(w, http.StatusInternalServerError, nil)
		return
	}

	// 7. Successo: 200 OK
	w.WriteHeader(http.StatusOK)
}

// addToGroup implementa l'handler POST /groups/:groupId/members: aggiunge un elenco di utenti a un gruppo.
func (rt *_router) addToGroup(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ctx reqcontext.RequestContext) {
	adminID := ctx.UserID

	convIDStr := ps.ByName("groupId")
	convID, err := strconv.ParseUint(convIDStr, 10, 64)
	if err != nil {
		rt.writeJSON(w, http.StatusBadRequest, nil)
		return
	}

	var req addMembersRequest
	if err := rt.decodeJSON(r, &req); err != nil {
		rt.writeJSON(w, http.StatusBadRequest, nil)
		return
	}

	if len(req.UserIDs) == 0 {
		rt.writeJSON(w, http.StatusBadRequest, map[string]string{"error": "User IDs list cannot be empty"})
		return
	}

	// Aggiungi tutti i membri uno per uno (La logica del DB gestisce l'unicità e l'admin status)
	// Visto che lo schema OpenAPI parla di una lista, iteriamo e gestiamo gli errori.
	// Per semplicità, consideriamo un successo se almeno un utente viene aggiunto.
	
	// Nota: Un'implementazione più robusta userebbe una transazione per aggiungere tutti i membri o nessuno.
	// Qui usiamo l'approccio iterativo semplice.
	
	var successfulAdds int
	var firstError error

	for _, targetUserID := range req.UserIDs {
		if targetUserID == 0 {
			continue // Salta ID non validi
		}
		
		err = rt.db.AddMemberToConversation(convID, adminID, targetUserID)
		
		if err != nil {
			if strings.Contains(err.Error(), "non è amministratore") {
				// Se l'admin fallisce all'inizio, è un errore 403.
				rt.writeJSON(w, http.StatusForbidden, nil)
				return
			}
			
			// Se l'errore non è di admin, è legato al target user (es. non esiste, o è già membro)
			if firstError == nil {
				firstError = err // Salva il primo errore non di admin
			}
			continue
		}
		successfulAdds++
	}

	if successfulAdds == 0 {
		if firstError != nil {
			ctx.Logger.WithError(firstError).Error("Database error during AddMemberToConversation")
			rt.writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to add members. Check if users exist and you are an admin."})
			return
		}
		// Se successfulAdds è 0 ma non ci sono stati errori critici (es. tutti i membri erano già nel gruppo),
		// l'operazione è tecnicamente un successo (200 OK)
	}

	w.WriteHeader(http.StatusOK) // 200 OK
}

// leaveGroup implementa l'handler DELETE /groups/:groupId/members/{userId}: un membro lascia o viene rimosso dal gruppo.
func (rt *_router) leaveGroup(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ctx reqcontext.RequestContext) {
	removerID := ctx.UserID // L'utente che esegue l'azione (può essere l'utente stesso o un admin)

	convIDStr := ps.ByName("groupId")
	convID, err := strconv.ParseUint(convIDStr, 10, 64)
	if err != nil {
		rt.writeJSON(w, http.StatusBadRequest, nil)
		return
	}

	targetIDStr := ps.ByName("userId")
	targetUserID, err := strconv.ParseUint(targetIDStr, 10, 64)
	if err != nil {
		rt.writeJSON(w, http.StatusBadRequest, nil)
		return
	}

	// Chiama la logica del DB (gestisce sia la rimozione da admin che l'auto-rimozione)
	err = rt.db.RemoveMemberFromConversation(convID, removerID, targetUserID)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) || strings.Contains(err.Error(), "non è membro") {
			rt.writeJSON(w, http.StatusNotFound, nil) // Non era membro / Utente target non trovato
			return
		}
		if strings.Contains(err.Error(), "solo gli amministratori") {
			rt.writeJSON(w, http.StatusForbidden, nil) // Non admin (nel caso removerID != targetUserID)
			return
		}

		ctx.Logger.WithError(err).Error("Database error during RemoveMemberFromConversation")
		rt.writeJSON(w, http.StatusInternalServerError, nil)
		return
	}

	w.WriteHeader(http.StatusOK) // 200 OK (OpenAPI dice 200)
}
