package api

import (
	"database/sql"
	"errors"
	"net/http"
	"strconv"

	// Percorsi corretti
	"github.com/pioloLlanos/Wasa/service/api/reqcontext"
	"github.com/pioloLlanos/Wasa/service/database"

	"github.com/julienschmidt/httprouter"
)

// conversationDetails è la struct di risposta per GET /conversations/:conversationId
type conversationDetails struct {
	Conversation database.Conversation `json:"conversation"`
	Messages     []database.Message  `json:"messages"`
}

// Modelli per Request e Response
type createConversationRequest struct {
	TargetUserID uint64 `json:"target_user_id"`
}

type conversationIDResponse struct {
	ConversationID uint64 `json:"conversation_id"`
}

type sendMessageRequest struct {
	Content string `json:"content"`
}

type messageIDResponse struct {
	MessageID uint64 `json:"message_id"`
}

// getMyConversations (DA IMPLEMENTARE)
func (rt *_router) getMyConversations(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ctx reqcontext.RequestContext) {
	// ⚠️ Necessita dell'implementazione di GetUserConversations nel DB Layer
	rt.writeJSON(w, http.StatusNotImplemented, nil)
}

// startNewConversation implementa l'handler POST /conversations (Creazione Chat 1-a-1)
func (rt *_router) startNewConversation(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ctx reqcontext.RequestContext) {
	userID := ctx.UserID

	var req createConversationRequest
	// rt.decodeJSON ora accetta 'w'
	if err := rt.decodeJSON(w, r, &req); err != nil {
		rt.writeJSON(w, http.StatusBadRequest, nil)
		return
	}

	// 1. Validazione base
	if req.TargetUserID == 0 || userID == req.TargetUserID {
		rt.writeJSON(w, http.StatusBadRequest, nil) // Non puoi creare chat 1-a-1 con te stesso o con ID 0
		return
	}

	// 2. Controllo di esistenza dell'utente target
	if err := rt.db.CheckUserExists(req.TargetUserID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			rt.writeJSON(w, http.StatusNotFound, nil)
			return
		}
		ctx.Logger.WithError(err).Error("Database error during CheckUserExists for target user")
		rt.writeJSON(w, http.StatusInternalServerError, nil)
		return
	}

	// 3. Logica di business: Crea la conversazione (il DB si occuperà di creare o restituire l'esistente)
	convID, err := rt.db.CreateOrGetPrivateConversation(userID, req.TargetUserID)
	if err != nil {
		ctx.Logger.WithError(err).Error("Database error during CreateOrGetPrivateConversation")
		rt.writeJSON(w, http.StatusInternalServerError, nil)
		return
	}

	response := conversationIDResponse{ConversationID: convID}
	rt.writeJSON(w, http.StatusCreated, response) // 201 Created
}

// getConversation implementa l'handler GET /conversations/:conversationId (Dettagli Conversazione e Messaggi)
func (rt *_router) getConversation(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ctx reqcontext.RequestContext) {
	userID := ctx.UserID

	// 1. Ottieni ID Conversazione dal path
	convIDStr := ps.ByName("conversationId")
	convID, err := strconv.ParseUint(convIDStr, 10, 64)
	if err != nil {
		rt.writeJSON(w, http.StatusBadRequest, nil)
		return
	}

	// 2. Chiama la logica del DB (Assumiamo che il DB gestisca il controllo di membership)
	conversation, messages, err := rt.db.GetConversationAndMessages(convID, userID)

	if err != nil {
		if errors.Is(err, database.AppErrorConversationNotFound) {
			rt.writeJSON(w, http.StatusNotFound, nil) // 404
			return
		}
		// Se l'errore è dovuto al fatto che l'utente non è membro, restituisce 403 Forbidden
		if errors.Is(err, database.AppErrorUserNotMember) {
			rt.writeJSON(w, http.StatusForbidden, nil)
			return
		}
		ctx.Logger.WithError(err).Error("Database error during GetConversationAndMessages")
		rt.writeJSON(w, http.StatusInternalServerError, nil)
		return
	}

	response := conversationDetails{
		Conversation: conversation,
		Messages: messages,
	}

	// Variabili 'conversation' e 'messages' usate correttamente nella response
	rt.writeJSON(w, http.StatusOK, response)
}

// sendMessage implementa l'handler POST /conversations/:conversationId (Invio Messaggio)
func (rt *_router) sendMessage(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ctx reqcontext.RequestContext) {
	userID := ctx.UserID

	// 1. Ottieni ID Conversazione
	convIDStr := ps.ByName("conversationId")
	convID, err := strconv.ParseUint(convIDStr, 10, 64)
	if err != nil {
		rt.writeJSON(w, http.StatusBadRequest, nil)
		return
	}

	// 2. Parsa il Form Multipart (limite a 10MB per la foto)
	err = r.ParseMultipartForm(10 << 20) // 10MB
	if err != nil {
		ctx.Logger.WithError(err).Error("Error parsing multipart form")
		rt.writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Errore nel parsing del form. Max 10MB"})
		return
	}

	// 3. Estrai tutti i campi possibili
	content := r.FormValue("content")
	replyToStr := r.FormValue("replyTo")
	forwardedStr := r.FormValue("forwarded")
	file, fileHeader, fileErr := r.FormFile("image")

	hasContent := content != ""
	hasPhoto := fileErr == nil && fileHeader != nil

	// Validazione oneOf: Almeno uno tra testo e foto deve essere presente
	if !hasContent && !hasPhoto {
		rt.writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Il messaggio deve contenere testo ('content') o una foto ('image')"})
		return
	}

	var msgID uint64 // 👈 Variabile dichiarata una sola volta
	var replyToID uint64 = 0
	var isForwarded bool = false

	// Parsing replyTo
	if replyToStr != "" {
		replyToID, err = strconv.ParseUint(replyToStr, 10, 64)
		if err != nil {
			rt.writeJSON(w, http.StatusBadRequest, map[string]string{"error": "ID 'replyTo' non valido"})
			return
		}
	}

	// Parsing forwarded (booleano)
	if forwardedStr == "true" {
		isForwarded = true
	}

	if hasPhoto {
		defer file.Close()

		// 4. Logica di upload della foto
		// NOTA: rt.simulateFileUpload richiede 3 argomenti: (convID, userID, filename).
		photoURL, err := rt.simulateFileUpload(convID, userID, fileHeader.Filename)
		if err != nil {
			ctx.Logger.WithError(err).Error("Error saving file")
			rt.writeJSON(w, http.StatusInternalServerError, nil)
			return
		}

		// 5. Creazione del messaggio con foto. Usa l'assegnazione semplice `msgID, err =` (senza `var`)
		msgID, err = rt.db.CreateMessageWithPhoto(convID, userID, photoURL, replyToID, isForwarded) // 👈 Ho assunto la firma completa
	} else {
		// 5. Creazione del messaggio di solo testo
		// ASSUMI che tu abbia aggiornato CreateMessage con replyToID e isForwarded:
		msgID, err = rt.db.CreateMessage(convID, userID, content, replyToID, isForwarded)
	}

	// 6. Gestione degli errori del Database
	if err != nil {
		if errors.Is(err, database.AppErrorConversationNotFound) || errors.Is(err, database.AppErrorReplyToNotFound) {
			rt.writeJSON(w, http.StatusNotFound, nil)
			return
		}
		if errors.Is(err, database.AppErrorUserNotMember) {
			rt.writeJSON(w, http.StatusForbidden, nil) // 403 Forbidden
			return
		}
		ctx.Logger.WithError(err).Error("Database error during CreateMessage")
		rt.writeJSON(w, http.StatusInternalServerError, nil)
		return
	}

	// 7. Successo
	// Variabile 'msgID' usata correttamente nella response
	rt.writeJSON(w, http.StatusCreated, messageIDResponse{MessageID: msgID})
}