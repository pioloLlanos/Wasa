package database

import (
	"database/sql"
	"errors"
)

// Dichiarazione degli errori custom (Dichiarati qui una sola volta nel package)
var AppErrorConversationNotFound = errors.New("conversazione non trovata")
var AppErrorUserNotMember = errors.New("l'utente non è membro della conversazione")
var AppErrorReplyToNotFound = errors.New("il messaggio di risposta (replyTo) non è stato trovato")
var AppErrorNomeGiaInUso = errors.New("nome utente già in uso")

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
	
	// 💡 AGGIUNTO: Metodo per creare lo schema del database all'avvio
	CreateSchema() error 

	// 1. UTENTE
	CreateUser(name string) (uint64, error)
	GetUserByName(name string) (uint64, error)
	CheckUserExists(id uint64) error

	// 👈 CORREZIONE CRITICA: SetMyUserName è ora presente nell'interfaccia
	SetMyUserName(id uint64, name string) error

	SetUserPhotoURL(id uint64, url string) error
	SearchUsers(query string) ([]User, error)

	// 2. CONVERSAZIONI
	GetConversations(userID uint64) ([]Conversation, error)
	CreateOrGetPrivateConversation(user1ID, user2ID uint64) (uint64, error)
	GetConversationAndMessages(convID, userID uint64) (Conversation, []Message, error)
	CreateGroup(adminID uint64, name string, initialMembers []uint64) (uint64, error)
	SetConversationName(convID uint64, adminID uint64, newName string) error
	SetConversationPhotoURL(convID uint64, adminID uint64, url string) error
	AddMemberToConversation(convID uint64, adminID uint64, targetUserID uint64) error
	RemoveMemberFromConversation(convID uint64, removerID uint64, targetUserID uint64) error

	// 3. MESSAGGI
	CreateMessage(convID uint64, senderID uint64, content string, replyToID uint64, isForwarded bool) (uint64, error)
	CreateMessageWithPhoto(convID uint64, senderID uint64, url string, replyToID uint64, isForwarded bool) (uint64, error)

	DeleteMessage(msgID uint64, userID uint64) error
	ForwardMessage(msgID uint64, senderID uint64, targetConvID uint64) (uint64, error)

	// 4. REAZIONI
	AddReaction(msgID uint64, userID uint64, reaction string) error
	RemoveReaction(msgID uint64, userID uint64) error
}

// --- 4. FUNZIONE COSTRUTTORE E METODI BASE ---

// New restituisce una nuova istanza di AppDatabase.
func New(db *sql.DB) (AppDatabase, error) {
	if db == nil {
		return nil, errors.New("database is required")
	}

	// In un un progetto reale, qui andrebbe la logica per migrare lo schema del DB

	return &appdbimpl{
		c: db,
	}, nil
}

// Ping implementa il controllo di salute del database.
func (db *appdbimpl) Ping() error {
	return db.c.Ping()
}

// CreateSchema implementa la logica per creare tutte le tabelle necessarie al primo avvio.
func (db *appdbimpl) CreateSchema() error {
	// Query SQL per creare la tabella 'users'
	// Nota: `ID INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT` gestisce l'assegnazione automatica di uint64
	usersQuery := `CREATE TABLE IF NOT EXISTS users (
		ID INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
		Name TEXT NOT NULL UNIQUE,
		PhotoURL TEXT DEFAULT ''
	);`

	// Query SQL per creare la tabella 'conversations'
	conversationsQuery := `CREATE TABLE IF NOT EXISTS conversations (
		ID INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
		Name TEXT,
		IsGroup BOOLEAN NOT NULL DEFAULT 0,
		LastMessageID INTEGER
	);`

	// Query SQL per creare la tabella 'members' (relazione molti a molti tra utenti e conversazioni)
	membersQuery := `CREATE TABLE IF NOT EXISTS members (
		ConversationID INTEGER NOT NULL,
		UserID INTEGER NOT NULL,
		IsAdmin BOOLEAN NOT NULL DEFAULT 0,
		FOREIGN KEY(ConversationID) REFERENCES conversations(ID) ON DELETE CASCADE,
		FOREIGN KEY(UserID) REFERENCES users(ID) ON DELETE CASCADE,
		PRIMARY KEY (ConversationID, UserID)
	);`

	// Query SQL per creare la tabella 'messages'
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

	// Query SQL per creare la tabella 'reactions'
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
	_, err := db.c.Exec(usersQuery)
	if err != nil {
		return err
	}
	_, err = db.c.Exec(conversationsQuery)
	if err != nil {
		return err
	}
	_, err = db.c.Exec(membersQuery)
	if err != nil {
		return err
	}
	_, err = db.c.Exec(messagesQuery)
	if err != nil {
		return err
	}
	_, err = db.c.Exec(reactionsQuery)
	if err != nil {
		return err
	}

	return nil
}
