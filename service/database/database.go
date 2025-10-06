package database

import (
	"database/sql"
	"errors"
)

// Dichiarazione degli errori custom (Devono essere definiti in un file del pacchetto database)
var AppErrorConversationNotFound = errors.New("conversazione non trovata")
var AppErrorUserNotMember = errors.New("l'utente non è membro della conversazione")
var AppErrorReplyToNotFound = errors.New("il messaggio di risposta (replyTo) non è stato trovato")
var AppErrorNomeGiaInUso = errors.New("nome utente già in uso") // 👈 AGGIUNTO: Usato in user.go

// --- 1. STRUTTURE DEI MODELLI (DTO) ---

type User struct {
	ID       uint64 `json:"id"`
	Name     string `json:"name"`
	PhotoURL string `json:"photo_url"`
}

type Conversation struct {
	ID              uint64 `json:"id"`
	Name            string `json:"name,omitempty"`
	IsGroup         bool `json:"is_group"`
	LastMessageID   uint64 `json:"last_message_id,omitempty"`
	PhotoURL        string `json:"photo_url,omitempty"`
	Members         []User `json:"members"`
}

// Message rappresenta un messaggio inviato in una conversazione.
type Message struct {
	ID             uint64 `json:"id"`
	ConversationID uint64 `json:"conversationId"`
	SenderID       uint64 `json:"senderId"`
	Content        string `json:"content"`
	Timestamp      string `json:"timestamp"`

	ReplyToID uint64 `json:"replyToId,omitempty"`

	IsPhoto bool `json:"isPhoto"`
}

// --- 2. STRUTTURA DI IMPLEMENTAZIONE ---

// appdbimpl è la struttura che implementa l'interfaccia AppDatabase.
type appdbimpl struct {
	c *sql.DB
}

// --- 3. INTERFACCIA APP DATABASE (IL TUO CONTRATTO) ---

type AppDatabase interface {
	// HEALTH CHECK
	Ping() error

	// 1. UTENTE
	CreateUser(name string) (uint64, error)
	GetUserByName(name string) (uint64, error)
	CheckUserExists(id uint64) error

	// 👈 CORREZIONE CRITICA: SetMyUserName era mancante
	SetMyUserName(id uint64, name string) error 

	SetUserPhotoURL(id uint64, url string) error
	SearchUsers(query string) ([]User, error)
	// SetUserName(id uint64, name string) error // Rimossa la firma duplicata/errata

	// 2. CONVERSAZIONI
	GetConversations(userID uint64) ([]Conversation, error)

	// 👈 AGGIUNTO: Per startNewConversation
	CreateOrGetPrivateConversation(user1ID, user2ID uint64) (uint64, error)
	// 👈 AGGIUNTO: Per getConversation
	GetConversationAndMessages(convID, userID uint64) (Conversation, []Message, error)

	// Metodi di gruppo
	CreateGroup(adminID uint64, name string, initialMembers []uint64) (uint64, error)
	SetConversationName(convID uint64, adminID uint64, newName string) error
	SetConversationPhotoURL(convID uint64, adminID uint64, url string) error
	AddMemberToConversation(convID uint64, adminID uint64, targetUserID uint64) error
	RemoveMemberFromConversation(convID uint64, removerID uint64, targetUserID uint64) error

	// 3. MESSAGGI
	// 👈 CORREZIONE FIRMA: Aggiunto replyToID e isForwarded, come usato in conversations.go
	CreateMessage(convID uint64, senderID uint64, content string, replyToID uint64, isForwarded bool) (uint64, error)
	// 👈 AGGIUNTO & CORRETTO FIRMA: Per invio messaggi con foto (con tutti gli argomenti)
	CreateMessageWithPhoto(convID uint64, senderID uint64, url string, replyToID uint64, isForwarded bool) (uint64, error)

	DeleteMessage(msgID uint64, userID uint64) error
	ForwardMessage(msgID uint64, senderID uint64, targetConvID uint64) (uint64, error)

	// 4. REAZIONI
	// 👈 AGGIUNTO: Per completare messages.go
	AddReaction(msgID uint64, userID uint64, reaction string) error
	RemoveReaction(msgID uint64, userID uint64) error
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