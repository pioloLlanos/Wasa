package database

import (
	"database/sql"
	"errors" // Necessario per errors.New in New()
	// Non c'è bisogno di fmt o strings in questo file pulito, dato che i metodi che li usavano sono in conversation.go
)

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

type Message struct {
	ID             uint64 `json:"id"`
	SenderID       uint64 `json:"sender_id"`
	Content        string `json:"content"`
	Timestamp      string `json:"timestamp"`
	ConversationID uint64 `json:"conversation_id"`
}

// --- 2. STRUTTURA DI IMPLEMENTAZIONE ---

// appdbimpl è la struttura che implementa l'interfaccia AppDatabase.
type appdbimpl struct {
	c *sql.DB
}

// --- 3. INTERFACCIA PUBBLICA ---

// AppDatabase è l'interfaccia pubblica esposta dal livello database.
type AppDatabase interface {
	// Metodi Utente
	CreateUser(name string) (uint64, error)
	GetUserByName(name string) (uint64, error)
	SetUserName(id uint64, name string) error
	SetUserPhotoURL(id uint64, url string) error
	SearchUsers(query string) ([]User, error)
	CheckUserExists(id uint64) error // Assumendo che questo esista in user.go

	// Metodi di conversazione/gruppo
	GetConversations(userID uint64) ([]Conversation, error)
	CreateGroup(adminID uint64, name string, initialMembers []uint64) (uint64, error)
	SetConversationName(convID uint64, adminID uint64, newName string) error
	SetConversationPhotoURL(convID uint64, adminID uint64, url string) error
	AddMemberToConversation(convID uint64, adminID uint64, targetUserID uint64) error
	RemoveMemberFromConversation(convID uint64, removerID uint64, targetUserID uint64) error

	// Metodi per i messaggi
	CreateMessage(convID uint64, senderID uint64, content string) (uint64, error)
	DeleteMessage(msgID uint64, userID uint64) error
	ForwardMessage(msgID uint64, senderID uint64, targetConvID uint64) (uint64, error)

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