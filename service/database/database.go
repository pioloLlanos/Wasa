package database

import (
    "database/sql"
    "errors"
)
// Dichiarazione degli errori custom (Devono essere definiti in un file del pacchetto database)
var AppErrorConversationNotFound = errors.New("conversazione non trovata")
var AppErrorUserNotMember = errors.New("l'utente non è membro della conversazione")
var AppErrorReplyToNotFound = errors.New("il messaggio di risposta (replyTo) non è stato trovato")
// --- 1. STRUTTURE DEI MODELLI (DTO) ---

type User struct {
	ID       uint64 `json:"id"`
	Name     string `json:"name"`
	PhotoURL string `json:"photo_url"`
}

type Conversation struct {
	ID              uint64   `json:"id"`
	Name            string   `json:"name,omitempty"`
	IsGroup         bool     `json:"is_group"`
	LastMessageID   uint64   `json:"last_message_id,omitempty"`
	PhotoURL        string   `json:"photo_url,omitempty"`
	Members         []User   `json:"members"`
}

// Message rappresenta un messaggio inviato in una conversazione.
type Message struct {
    ID             uint64    `json:"id"`
    ConversationID uint64    `json:"conversationId"`
    SenderID       uint64    `json:"senderId"`
    Content        string    `json:"content"`
    Timestamp      string    `json:"timestamp"`
    
    ReplyToID      uint64    `json:"replyToId,omitempty"` 
    
    IsPhoto        bool      `json:"isPhoto"`           
}


// --- 2. STRUTTURA DI IMPLEMENTAZIONE ---

// appdbimpl è la struttura che implementa l'interfaccia AppDatabase.
type appdbimpl struct {
	c *sql.DB
}


type AppDatabase interface {
    // ... Metodi Utente e Gruppo (Lascia i tuoi esistenti)
    CreateUser(name string) (uint64, error)
    GetUserByName(name string) (uint64, error)
    SetUserName(id uint64, name string) error
    SetUserPhotoURL(id uint64, url string) error
    SearchUsers(query string) ([]User, error)
    CheckUserExists(id uint64) error

    // Metodi di conversazione/gruppo
    GetConversations(userID uint64) ([]Conversation, error)
    
    // 👈 MANCANTE 1: Per startNewConversation
    CreateOrGetPrivateConversation(user1ID, user2ID uint64) (uint64, error)
    // 👈 MANCANTE 2: Per getConversation
    GetConversationAndMessages(convID, userID uint64) (Conversation, []Message, error)

    CreateGroup(adminID uint64, name string, initialMembers []uint64) (uint64, error)
    SetConversationName(convID uint64, adminID uint64, newName string) error
    SetConversationPhotoURL(convID uint64, adminID uint64, url string) error
    AddMemberToConversation(convID uint64, adminID uint64, targetUserID uint64) error
    RemoveMemberFromConversation(convID uint64, removerID uint64, targetUserID uint64) error

    // Metodi per i messaggi
    // 👈 AGGIORNATO: Correzione del numero di argomenti (5 invece di 3)
    CreateMessage(convID uint64, senderID uint64, content string, replyToID uint64, isPhoto bool) (uint64, error)
    // 👈 MANCANTE 3: Per invio messaggi con foto
    CreateMessageWithPhoto(convID uint64, senderID uint64, url string) (uint64, error)
    
    DeleteMessage(msgID uint64, userID uint64) error
    ForwardMessage(msgID uint64, senderID uint64, targetConvID uint64) (uint64, error)
    // ... altri metodi (AddReaction, RemoveReaction, ecc.)
    
    // Health Check
    Ping() error
}
// --- 4. FUNZIONE COSTRUTTORE E METODI BASE ---

// New restituisce una nuova istanza di AppDatabase.
func New(db *sql.DB) (AppDatabase, error) {
	if db == nil {
		return nil, errors.New("database is required")
	}

	// In un progetto reale, qui andrebbe la logica per migrare lo schema del DB

	return &appdbimpl{
		c: db,
	}, nil
}

// Ping implementa il controllo di salute del database.
func (db *appdbimpl) Ping() error {
	return db.c.Ping()
}