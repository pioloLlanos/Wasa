package database

import (
    "database/sql"
    // Altri import come errors, fmt, strings, ecc.
)

// 1. Strutture dei Modelli (D.T.O.) - Richieste da conversation.go
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
    ID              uint64    `json:"id"`
    SenderID        uint64    `json:"sender_id"`
    Content         string    `json:"content"`
    Timestamp       string    `json:"timestamp"` 
    ConversationID  uint64    `json:"conversation_id"`
}


// 2. Struttura di Implementazione del Database - Richiesta da tutti i metodi
type appdbimpl struct {
    c *sql.DB 
}


// 3. Funzione Costruttore (New) - Richiesta da main.go
func New(db *sql.DB) (AppDatabase, error) {
    if db == nil {
        return nil, errors.New("database is required")
    }
    
    // In un progetto reale, qui andrebbe la logica per migrare lo schema del DB
    
    return &appdbimpl{
        c: db,
    }, nil
}

// 4. Interfaccia Pubblica (AppDatabase) - Richiesta da api.go e main.go
// Devi includere qui TUTTE le firme dei metodi implementati in database.go, conversation.go, message.go, user.go, ecc.
type AppDatabase interface {
    CreateUser(name string) (uint64, error)
    GetUserByName(name string) (uint64, error)
    SetUserName(id uint64, name string) error
    SetUserPhotoURL(id uint64, url string) error
    SearchUsers(query string) ([]User, error)
    CheckUserExists(id uint64) error // Richiesto in api-context-wrapper.go
    
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
    Ping() error // Richiesto in liveness.go
}

// Implementazione di Ping() - Richiesto in liveness.go
func (db *appdbimpl) Ping() error {
    return db.c.Ping()
}

// checkAdminStatus è una funzione di utilità per verificare se un utente è admin di una conversazione.
func (db *appdbimpl) checkAdminStatus(convID uint64, userID uint64) error {
	var isAdmin int
	err := db.c.QueryRow("SELECT is_admin FROM conversation_members WHERE conversation_id = ? AND user_id = ?",
		convID, userID).Scan(&isAdmin)

	if errors.Is(err, sql.ErrNoRows) {
		return errors.New("l'utente non è membro della conversazione")
	} else if err != nil {
		return fmt.Errorf("errore nel controllo dell'amministratore: %w", err)
	}

	if isAdmin == 0 {
		return errors.New("l'utente non è amministratore della conversazione")
	}
	return nil
}

// CreateGroup crea una nuova conversazione di gruppo, imposta l'admin e aggiunge i membri iniziali.
func (db *appdbimpl) CreateGroup(adminID uint64, name string, initialMembers []uint64) (uint64, error) {
	// Avvia una transazione per assicurare l'atomicità
	tx, err := db.c.Begin()
	if err != nil {
		return 0, fmt.Errorf("impossibile avviare la transazione per CreateGroup: %w", err)
	}

	// 1. Inserisci la nuova conversazione come Gruppo
	res, err := tx.Exec("INSERT INTO conversations (name, is_group) VALUES (?, 1)", name)
	if err != nil {
		_ = tx.Rollback()
		return 0, fmt.Errorf("impossibile creare la conversazione: %w", err)
	}

	convID, err := res.LastInsertId()
	if err != nil {
		_ = tx.Rollback()
		return 0, fmt.Errorf("impossibile recuperare l'ID della conversazione: %w", err)
	}
	convIDUint := uint64(convID)

	// 2. Prepara l'inserimento dei membri
	var valueStrings []string
	var valueArgs []interface{}

	for _, memberID := range initialMembers {
		// Verifica se il membro è l'admin per impostare is_admin = 1
		isAdmin := 0
		if memberID == adminID {
			isAdmin = 1
		}

		// Aggiunge l'utente, l'ID della conversazione e lo stato di admin ai parametri
		valueStrings = append(valueStrings, "(?, ?, ?)")
		valueArgs = append(valueArgs, convIDUint, memberID, isAdmin)
	}

	// 3. Esegui l'inserimento di massa dei membri
	if len(valueStrings) > 0 {
		stmt := fmt.Sprintf("INSERT INTO conversation_members (conversation_id, user_id, is_admin) VALUES %s",
			strings.Join(valueStrings, ","))

		_, err = tx.Exec(stmt, valueArgs...)
		if err != nil {
			_ = tx.Rollback()
			// Qui si potrebbe verificare un errore FK se un memberID non esiste in `users`
			return 0, fmt.Errorf("impossibile aggiungere i membri iniziali: %w", err)
		}
	}

	// 4. Commit della transazione
	err = tx.Commit()
	if err != nil {
		return 0, fmt.Errorf("impossibile eseguire il commit della transazione: %w", err)
	}

	return convIDUint, nil
}

// SetConversationName aggiorna il nome di un gruppo (solo se l'utente è un admin).
func (db *appdbimpl) SetConversationName(convID uint64, adminID uint64, newName string) error {
	// 1. Verifica i permessi di amministrazione
	if err := db.checkAdminStatus(convID, adminID); err != nil {
		return err
	}

	// 2. Aggiorna il nome della conversazione
	// Assicura che venga aggiornata solo una conversazione di gruppo (is_group = 1)
	_, err := db.c.Exec("UPDATE conversations SET name = ? WHERE id = ? AND is_group = 1", newName, convID)
	if err != nil {
		return fmt.Errorf("impossibile aggiornare il nome del gruppo: %w", err)
	}

	return nil
}

// SetConversationPhotoURL aggiorna l'URL della foto di un gruppo (solo se l'utente è un admin).
func (db *appdbimpl) SetConversationPhotoURL(convID uint64, adminID uint64, url string) error {
	// 1. Verifica i permessi di amministrazione
	if err := db.checkAdminStatus(convID, adminID); err != nil {
		return err
	}

	// 2. Aggiorna l'URL della foto della conversazione
	_, err := db.c.Exec("UPDATE conversations SET photo_url = ? WHERE id = ? AND is_group = 1", url, convID)
	if err != nil {
		return fmt.Errorf("impossibile aggiornare la foto del gruppo: %w", err)
	}

	return nil
}

// AddMemberToConversation aggiunge un utente a un gruppo (solo se l'utente chiamante è un admin).
func (db *appdbimpl) AddMemberToConversation(convID uint64, adminID uint64, targetUserID uint64) error {
	// 1. Verifica i permessi di amministrazione
	if err := db.checkAdminStatus(convID, adminID); err != nil {
		return err
	}

	// 2. Controlla se l'utente target esiste (opzionale, ma consigliato per errori puliti)
	var exists bool
	err := db.c.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE id = ?)", targetUserID).Scan(&exists)
	if err != nil || !exists {
		return errors.New("l'utente target non esiste")
	}

	// 3. Aggiunge il membro (is_admin = 0 per default)
	_, err = db.c.Exec("INSERT INTO conversation_members (conversation_id, user_id, is_admin) VALUES (?, ?, 0)", convID, targetUserID)
	if err != nil {
		// Se l'utente è già membro, l'INSERT fallirà per violazione di chiave primaria (che ignoriamo)
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return nil // Già membro, consideriamo l'operazione un successo
		}
		return fmt.Errorf("impossibile aggiungere il membro: %w", err)
	}

	return nil
}

// RemoveMemberFromConversation rimuove un utente da un gruppo.
// L'azione è consentita se:
// a) removerID == targetUserID (l'utente sta uscendo dal gruppo).
// b) removerID è un admin (l'admin sta rimuovendo un altro utente).
func (db *appdbimpl) RemoveMemberFromConversation(convID uint64, removerID uint64, targetUserID uint64) error {
	// L'utente si sta rimuovendo da solo
	if removerID != targetUserID {
		// Se l'utente non è l'utente target, deve essere un admin
		if err := db.checkAdminStatus(convID, removerID); err != nil {
			return errors.New("solo gli amministratori possono rimuovere altri utenti")
		}
	}
	// Se l'utente si sta rimuovendo da solo, non c'è bisogno di verificare se è admin (può uscire chiunque)

	// Esegui la rimozione
	result, err := db.c.Exec("DELETE FROM conversation_members WHERE conversation_id = ? AND user_id = ?", convID, targetUserID)
	if err != nil {
		return fmt.Errorf("impossibile rimuovere il membro: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("errore RowsAffected: %w", err)
	}
	if rowsAffected == 0 {
		return sql.ErrNoRows // Nessun utente rimosso (forse non era membro)
	}

	return nil
}
