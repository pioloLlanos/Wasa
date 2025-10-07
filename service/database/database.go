package database

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/sirupsen/logrus"
)

// Dichiarazione degli errori custom (Dichiarati qui una sola volta nel package)
var AppErrorConversationNotFound = errors.New("conversazione non trovata")
var AppErrorUserNotMember = errors.New("l'utente non è membro della conversazione")
var AppErrorReplyToNotFound = errors.New("il messaggio di risposta (replyTo) non è stato trovato")
var AppErrorNomeGiaInUso = errors.New("nome utente già in uso")
var AppErrorUserNotFound = errors.New("utente non trovato")

// --- 1. STRUTTURE DEI MODELLI (DTO) ---

type User struct {
	ID       uint64 `json:"id"`
	Name     string `json:"name"`
	PhotoURL string `json:"photo_url"`
}

type Conversation struct {
	ID              uint64 `json:"id"`
	Name            string `json:"name,omitempty"`
	IsGroup         bool   `json:"is_group"`
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

// --- 2. INTERFACCIA E IMPLEMENTAZIONE DEL DATABASE ---

// AppDatabase definisce l'interfaccia per tutte le operazioni sul database
type AppDatabase interface {
	// Lifecycle
	Close() error
	Ping() error

	// User Operations (in db_user_ops.go)
	CreateUser(name string) (uint64, error)
	GetUserByName(name string) (uint64, error)
	CheckUserExists(id uint64) error
	SetUserName(id uint64, newName string) error
	SetUserPhotoURL(id uint64, url string) error
	SearchUsers(query string) ([]User, error)
	GetUserByID(id uint64) (User, error)

	// Conversation Operations (in conversation.go)
	GetConversations(userID uint64) ([]Conversation, error)
	StartNewConversation(userID1 uint64, userID2 uint64) (uint64, error)
	GetConversationAndMessages(convID uint64, userID uint64) (Conversation, []Message, error)
	
	// Group Operations (in conversation.go)
	CreateGroup(name string, adminID uint64, memberIDs []uint64) (uint64, error)
	SetGroupName(convID uint64, adminID uint64, newName string) error
	SetGroupPhotoURL(convID uint64, adminID uint64, url string) error
	AddMembersToConversation(convID uint64, adminID uint64, newMemberIDs []uint64) error
	RemoveMemberFromConversation(convID uint64, adminID uint64, targetUserID uint64) error

	// Message Operations (in message.go)
	CreateMessage(convID uint64, senderID uint64, content string, replyToID uint64, isForwarded bool) (uint64, error)
	CreateMessageWithPhoto(convID uint64, senderID uint64, url string, replyToID uint64, isForwarded bool) (uint64, error)
	DeleteMessage(msgID uint64, senderID uint64) error
	ForwardMessage(msgID uint64, senderID uint64, targetConvID uint64) (uint64, error)
	AddReaction(msgID uint64, userID uint64, reaction string) error
	RemoveReaction(msgID uint64, userID uint64) error
}

type appdbimpl struct {
	c      *sql.DB
	logger *logrus.Logger
}

// New è la funzione che main.go chiama per creare l'istanza del DB.
func New(dbPath string, logger *logrus.Logger) (AppDatabase, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("errore nell'apertura del database: %w", err)
	}

	appDB := &appdbimpl{
		c:      db,
		logger: logger,
	}

	// 💡 CORREZIONE: Inizializza le tabelle all'avvio. Questo risolve "no such table".
	if err := appDB.Init(); err != nil {
		return nil, fmt.Errorf("errore nell'inizializzazione delle tabelle: %w", err)
	}

	return appDB, nil
}

// Init esegue tutte le query CREATE TABLE IF NOT EXISTS.
func (db *appdbimpl) Init() error {
	usersQuery := `CREATE TABLE IF NOT EXISTS users (
        ID INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
        Name TEXT NOT NULL UNIQUE,
        PhotoURL TEXT NOT NULL DEFAULT ''
    );`

	conversationsQuery := `CREATE TABLE IF NOT EXISTS conversations (
        ID INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
        Name TEXT,
        IsGroup BOOLEAN NOT NULL DEFAULT 0,
        LastMessageID INTEGER,
        PhotoURL TEXT,
        FOREIGN KEY(LastMessageID) REFERENCES messages(ID) ON DELETE SET NULL
    );`

	membersQuery := `CREATE TABLE IF NOT EXISTS conversation_members (
        ConversationID INTEGER NOT NULL,
        UserID INTEGER NOT NULL,
        IsAdmin BOOLEAN NOT NULL DEFAULT 0,
        FOREIGN KEY(ConversationID) REFERENCES conversations(ID) ON DELETE CASCADE,
        FOREIGN KEY(UserID) REFERENCES users(ID) ON DELETE CASCADE,
        PRIMARY KEY (ConversationID, UserID)
    );`

	messagesQuery := `CREATE TABLE IF NOT EXISTS messages (
        ID INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
        ConversationID INTEGER NOT NULL,
        SenderID INTEGER NOT NULL,
        Content TEXT NOT NULL,
        Timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
        ReplyToID INTEGER,
        IsPhoto BOOLEAN NOT NULL DEFAULT 0,
        IsForwarded BOOLEAN NOT NULL DEFAULT 0,
        FOREIGN KEY(ConversationID) REFERENCES conversations(ID) ON DELETE CASCADE,
        FOREIGN KEY(SenderID) REFERENCES users(ID) ON DELETE CASCADE,
        FOREIGN KEY(ReplyToID) REFERENCES messages(ID) ON DELETE SET NULL
    );`

	reactionsQuery := `CREATE TABLE IF NOT EXISTS reactions (
        MessageID INTEGER NOT NULL,
        UserID INTEGER NOT NULL,
        Reaction TEXT NOT NULL,
        Timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
        FOREIGN KEY(MessageID) REFERENCES messages(ID) ON DELETE CASCADE,
        FOREIGN KEY(UserID) REFERENCES users(ID) ON DELETE CASCADE,
        PRIMARY KEY (MessageID, UserID)
    );`

	// Esegui tutte le query in sequenza
	queries := []string{usersQuery, conversationsQuery, membersQuery, messagesQuery, reactionsQuery}
	for _, query := range queries {
		if _, err := db.c.Exec(query); err != nil {
			return fmt.Errorf("fallimento nella creazione della tabella: %w", err)
		}
	}

	return nil
}

// Close chiude la connessione al database
func (db *appdbimpl) Close() error {
	return db.c.Close()
}

// Ping verifica che la connessione al database sia attiva
func (db *appdbimpl) Ping() error {
	return db.c.Ping()
}

// NOTA: Le altre funzioni (GetUserByName, CreateUser, ecc.) dovrebbero trovarsi
// nei rispettivi file (db_user_ops.go, conversation.go, message.go)