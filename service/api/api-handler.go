package api

import (
	"net/http"
)

// Handler returns an instance of httprouter.Router that handle APIs registered here
func (rt *_router) Handler() http.Handler {

	// 1. LOGIN (NON WRAPPED) - Gestisce la creazione di sessione/login
	rt.router.POST("/session", rt.doLogin)

	// 2. UTENTE E RICERCA (WRAPPED) - Gestisce le operazioni sul profilo e la ricerca utenti
	rt.router.PUT("/me/name", rt.wrap(rt.setMyUserName))   // Aggiorna il nome utente
	rt.router.PUT("/me/photo", rt.wrap(rt.setMyPhoto))   // Aggiorna la foto profilo
	rt.router.GET("/users/search", rt.wrap(rt.searchUsers)) // Ricerca utenti

	// 3. CONVERSAZIONI (WRAPPED) - Gestisce conversazioni e messaggi diretti
	rt.router.GET("/conversations", rt.wrap(rt.getMyConversations))           // Ottiene la lista delle conversazioni dell'utente
	rt.router.POST("/conversations", rt.wrap(rt.startNewConversation))       // Avvia una nuova conversazione 1:1
	rt.router.GET("/conversations/:conversationId", rt.wrap(rt.getConversation)) // Ottiene i dettagli di una conversazione (inclusi i messaggi)
	rt.router.POST("/conversations/:conversationId", rt.wrap(rt.sendMessage))    // Invia un nuovo messaggio in una conversazione

	// 4. MESSAGGI (WRAPPED) - Gestisce le operazioni specifiche sui messaggi
	rt.router.DELETE("/messages/:messageId", rt.wrap(rt.deleteMessage))           // Elimina un messaggio
	rt.router.POST("/messages/:messageId/forward", rt.wrap(rt.forwardMessage))     // Inoltra un messaggio
	rt.router.POST("/messages/:messageId/reactions", rt.wrap(rt.commentMessage))   // Aggiunge una reazione a un messaggio
	rt.router.DELETE("/messages/:messageId/reactions", rt.wrap(rt.uncommentMessage)) // Rimuove una reazione da un messaggio

	// 5. GRUPPI (WRAPPED) - Gestisce la creazione e la modifica dei gruppi
	rt.router.POST("/groups", rt.wrap(rt.createGroup))                         // Crea un nuovo gruppo
	rt.router.GET("/groups/:groupId", rt.wrap(rt.getGroupDetails))             // 👈 AGGIUNTO: Ottiene i dettagli del gruppo
	rt.router.PUT("/groups/:groupId/name", rt.wrap(rt.setGroupName))           // Modifica il nome del gruppo
	rt.router.PUT("/groups/:groupId/photo", rt.wrap(rt.setGroupPhoto))         // Modifica la foto del gruppo
	rt.router.POST("/groups/:groupId/members", rt.wrap(rt.addToGroup))         // Aggiunge un membro al gruppo
	rt.router.DELETE("/groups/:groupId/members/:userId", rt.wrap(rt.leaveGroup)) // Rimuove/Abbandona il gruppo

	// Special routes (LIVENESS) - Rotta per il controllo di stato dell'applicazione
	rt.router.GET("/liveness", rt.liveness)

	return rt.handleCORS(rt.router)
}
