package database

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
)

// GetConversations recupera tutte le conversazioni a cui partecipa un utente, inclusi i dati dei membri.
func (db *appdbimpl) GetConversations(userID uint64) ([]Conversation, error) {
	// Query per trovare tutti gli ID delle conversazioni a cui l'utente partecipa.
	rows, err := db.c.Query(`
        SELECT 
            c.id, c.name, c.is_group, c.last_message_id, c.photo_url
        FROM 
            conversations c
        JOIN 
            conversation_members cm ON c.id = cm.conversation_id
        WHERE 
            cm.user_id = ?
    `, userID)
	if err != nil {
		return nil, fmt.Errorf("errore nella query GetConversations: %w", err)
	}
	defer rows.Close()

	var conversationIDs []uint64
	conversationsMap := make(map[uint64]Conversation)

	for rows.Next() {
		var conv Conversation
		var isGroupInt int
		// Nota: È stata aggiunta c.photo_url (necessita di aggiornamento del modello Conversation in database.go)
		if err := rows.Scan(&conv.ID, &conv.Name, &isGroupInt, &conv.LastMessageID, &conv.PhotoURL); err != nil {
			return nil, fmt.Errorf("errore nella scansione della conversazione: %w", err)
		}

		conv.IsGroup = isGroupInt != 0
		conversationsMap[conv.ID] = conv
		conversationIDs = append(conversationIDs, conv.ID)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("errore dopo l'iterazione delle conversazioni: %w", err)
	}

	if len(conversationIDs) == 0 {
		return []Conversation{}, nil // Nessuna conversazione trovata
	}

	// Fase 2: Recupera i membri per ogni conversazione trovata (query più complessa)
	idStrings := make([]string, len(conversationIDs))
	for i, id := range conversationIDs {
		idStrings[i] = fmt.Sprint(id)
	}

	// Query per recuperare tutti i membri di tutte le conversazioni in una volta
	memberRows, err := db.c.Query(fmt.Sprintf(`
        SELECT 
            cm.conversation_id, u.id, u.name, u.photo_url
        FROM 
            conversation_members cm
        JOIN 
            users u ON cm.user_id = u.id
        WHERE 
            cm.conversation_id IN (%s)
    `, strings.Join(idStrings, ",")))
	if err != nil {
		return nil, fmt.Errorf("errore nella query dei membri della conversazione: %w", err)
	}
	defer memberRows.Close()

	for memberRows.Next() {
		var (
			convID uint64
			member User
		)
		if err := memberRows.Scan(&convID, &member.ID, &member.Name, &member.PhotoURL); err != nil {
			return nil, fmt.Errorf("errore nella scansione dei membri: %w", err)
		}

		// Aggiungi il membro alla conversazione corretta nella mappa
		conv := conversationsMap[convID]
		conv.Members = append(conv.Members, member)
		conversationsMap[convID] = conv
	}

	// Riconverti la mappa in slice (la lista finale da restituire)
	result := make([]Conversation, 0, len(conversationsMap))
	for _, conv := range conversationsMap {
		result = append(result, conv)
	}

	return result, nil
}

// CreateConversation crea una nuova conversazione 1-a-1 tra due utenti.
func (db *appdbimpl) CreateConversation(userID uint64, targetUserID uint64) (uint64, error) {
	tx, err := db.c.Begin()
	if err != nil {
		return 0, fmt.Errorf("impossibile iniziare la transazione: %w", err)
	}

	// 1. Inserisci la nuova conversazione (is_group=0 per 1-a-1)
	res, err := tx.Exec("INSERT INTO conversations (name, is_group, photo_url) VALUES (?, 0, ?)", "", "")
	if err != nil {
		_ = tx.Rollback()
		return 0, fmt.Errorf("errore nell'inserimento della conversazione: %w", err)
	}
	convID, err := res.LastInsertId()
	if err != nil {
		_ = tx.Rollback()
		return 0, fmt.Errorf("errore nel recupero del LastInsertId: %w", err)
	}

	// 2. Inserisci il primo membro
	_, err = tx.Exec("INSERT INTO conversation_members (conversation_id, user_id) VALUES (?, ?)", convID, userID)
	if err != nil {
		_ = tx.Rollback()
		return 0, fmt.Errorf("errore nell'inserimento del primo membro: %w", err)
	}

	// 3. Inserisci il secondo membro
	_, err = tx.Exec("INSERT INTO conversation_members (conversation_id, user_id) VALUES (?, ?)", convID, targetUserID)
	if err != nil {
		_ = tx.Rollback()
		return 0, fmt.Errorf("errore nell'inserimento del secondo membro: %w", err)
	}

	// 4. Commit della transazione
	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("errore nel commit della transazione: %w", err)
	}

	return uint64(convID), nil
}

// GetConversationByID recupera i dati di base di una singola conversazione (senza messaggi).
func (db *appdbimpl) GetConversationByID(convID uint64) (*Conversation, error) {
	var conv Conversation
	var isGroupInt int

	err := db.c.QueryRow("SELECT id, name, is_group, last_message_id, photo_url FROM conversations WHERE id = ?", convID).
		Scan(&conv.ID, &conv.Name, &isGroupInt, &conv.LastMessageID, &conv.PhotoURL)

	if err != nil {
		return nil, err
	}

	conv.IsGroup = isGroupInt != 0

	memberRows, err := db.c.Query(`
        SELECT u.id, u.name, u.photo_url FROM conversation_members cm
        JOIN users u ON cm.user_id = u.id
        WHERE cm.conversation_id = ?
    `, convID)
	if err != nil {
		return nil, fmt.Errorf("errore nella query dei membri per la conversazione singola: %w", err)
	}
	defer memberRows.Close()

	for memberRows.Next() {
		var member User
		if err := memberRows.Scan(&member.ID, &member.Name, &member.PhotoURL); err != nil {
			return nil, fmt.Errorf("errore nella scansione dei membri per la conversazione singola: %w", err)
		}
		conv.Members = append(conv.Members, member)
	}

	if err := memberRows.Err(); err != nil {
		return nil, fmt.Errorf("errore dopo l'iterazione dei membri: %w", err)
	}

	return &conv, nil
}

// CheckConversationMembership verifica se un utente appartiene a una conversazione.
func (db *appdbimpl) CheckConversationMembership(convID uint64, userID uint64) error {
	var count int
	err := db.c.QueryRow("SELECT COUNT(*) FROM conversation_members WHERE conversation_id = ? AND user_id = ?", convID, userID).
		Scan(&count)
	if err != nil {
		return err
	}
	if count == 0 {
		return sql.ErrNoRows // Usiamo ErrNoRows per indicare "non membro"
	}
	return nil
}

// GetConversationMessages recupera i messaggi di una conversazione, con paginazione.
func (db *appdbimpl) GetConversationMessages(convID uint64, limit int, offset int) ([]Message, error) {
	rows, err := db.c.Query(`
        SELECT id, conversation_id, sender_id, content, timestamp 
        FROM messages 
        WHERE conversation_id = ?
        ORDER BY timestamp DESC
        LIMIT ? OFFSET ?
    `, convID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("errore nella query dei messaggi: %w", err)
	}
	defer rows.Close()

	var messages []Message
	for rows.Next() {
		var msg Message
		if err := rows.Scan(&msg.ID, &msg.ConversationID, &msg.SenderID, &msg.Content, &msg.Timestamp); err != nil {
			return nil, fmt.Errorf("errore nella scansione del messaggio: %w", err)
		}
		messages = append(messages, msg)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("errore dopo l'iterazione dei messaggi: %w", err)
	}

	// Invertiamo per l'ordine cronologico corretto (dal più vecchio al più nuovo)
	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}

	return messages, nil
}

// CheckAdminStatus verifica se un utente è admin di una conversazione (utility interna).
func (db *appdbimpl) checkAdminStatus(convID uint64, userID uint64) error {
	var isAdmin int
	err := db.c.QueryRow("SELECT is_admin FROM conversation_members WHERE conversation_id = ? AND user_id = ?", convID, userID).
		Scan(&isAdmin)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return errors.New("l'utente non è membro della conversazione")
		}
		return err
	}
	if isAdmin == 0 {
		return errors.New("l'utente non è amministratore della conversazione")
	}
	return nil
}

// CreateGroup crea una nuova conversazione di gruppo.
func (db *appdbimpl) CreateGroup(adminID uint64, name string, initialMembers []uint64) (uint64, error) {
	tx, err := db.c.Begin()
	if err != nil {
		return 0, fmt.Errorf("impossibile iniziare la transazione per gruppo: %w", err)
	}

	// 1. Inserisci la nuova conversazione (is_group=1)
	res, err := tx.Exec("INSERT INTO conversations (name, is_group, photo_url) VALUES (?, 1, ?)", name, "")
	if err != nil {
		_ = tx.Rollback()
		return 0, fmt.Errorf("errore nell'inserimento della conversazione di gruppo: %w", err)
	}
	convID, err := res.LastInsertId()
	if err != nil {
		_ = tx.Rollback()
		return 0, fmt.Errorf("errore nel recupero del LastInsertId per gruppo: %w", err)
	}

	// 2. Inserisci i membri iniziali
	for _, memberID := range initialMembers {
		isAdmin := 0
		if memberID == adminID {
			isAdmin = 1 // L'utente che crea il gruppo è admin
		}

		_, err = tx.Exec("INSERT INTO conversation_members (conversation_id, user_id, is_admin) VALUES (?, ?, ?)", convID, memberID, isAdmin)
		if err != nil {
			_ = tx.Rollback()
			return 0, fmt.Errorf("errore nell'inserimento del membro %d nel gruppo: %w", memberID, err)
		}
	}

	// 3. Commit della transazione
	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("errore nel commit della transazione di gruppo: %w", err)
	}

	return uint64(convID), nil
}

// SetConversationName aggiorna il nome di una conversazione (solo se l'adminID è admin).
func (db *appdbimpl) SetConversationName(convID uint64, adminID uint64, newName string) error {
	// 1. Verifica i permessi di amministrazione
	if err := db.checkAdminStatus(convID, adminID); err != nil {
		return err
	}

	// 2. Aggiorna il nome
	_, err := db.c.Exec("UPDATE conversations SET name = ? WHERE id = ?", newName, convID)
	return err
}

// SetConversationPhotoURL aggiorna l'URL della foto di un gruppo.
func (db *appdbimpl) SetConversationPhotoURL(convID uint64, adminID uint64, url string) error {
	if err := db.checkAdminStatus(convID, adminID); err != nil {
		return err
	}

	// Esegue l'aggiornamento (si assume che la colonna photo_url esista in conversations)
	_, err := db.c.Exec("UPDATE conversations SET photo_url = ? WHERE id = ?", url, convID)
	return err
}

// AddMemberToConversation aggiunge un utente a un gruppo (solo se l'adminID è admin).
func (db *appdbimpl) AddMemberToConversation(convID uint64, adminID uint64, targetUserID uint64) error {
	if err := db.checkAdminStatus(convID, adminID); err != nil {
		return err
	}

	// Inserisce il nuovo membro. Se il membro è già presente, la PK (conversation_id, user_id) fallirà in silenzio.
	_, err := db.c.Exec("INSERT INTO conversation_members (conversation_id, user_id, is_admin) VALUES (?, ?, 0)", convID, targetUserID)
	return err
}

// RemoveMemberFromConversation rimuove un utente da un gruppo.
// L'utente può rimuovere sé stesso (lasciare) o un admin può rimuovere un targetUserID.
func (db *appdbimpl) RemoveMemberFromConversation(convID uint64, removerID uint64, targetUserID uint64) error {
	// Caso 1: L'utente lascia il gruppo da solo (removerID == targetUserID)
	if removerID == targetUserID {
		res, err := db.c.Exec("DELETE FROM conversation_members WHERE conversation_id = ? AND user_id = ?", convID, targetUserID)
		if err != nil {
			return err
		}
		rowsAffected, _ := res.RowsAffected()
		if rowsAffected == 0 {
			return sql.ErrNoRows // Non era membro
		}
		return nil
	}

	// Caso 2: Un admin rimuove un altro utente
	// 1. Verifica i permessi di amministrazione del remover
	if err := db.checkAdminStatus(convID, removerID); err != nil {
		return errors.New("solo gli amministratori possono rimuovere altri utenti")
	}

	// 2. Rimuovi il targetUserID
	res, err := db.c.Exec("DELETE FROM conversation_members WHERE conversation_id = ? AND user_id = ?", convID, targetUserID)
	if err != nil {
		return err
	}
	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return sql.ErrNoRows // Utente target non trovato
	}
	return nil
}



// CreateOrGetPrivateConversation crea una nuova conversazione privata tra due utenti se non esiste,
// altrimenti restituisce l'ID della conversazione esistente.
func (db *appdbimpl) CreateOrGetPrivateConversation(user1ID, user2ID uint64) (uint64, error) {
    // 1. Ordina gli ID per garantire l'unicità nella ricerca e nell'eventuale creazione.
    if user1ID > user2ID {
        user1ID, user2ID = user2ID, user1ID
    }

    var convID uint64
    // 2. Prova a trovare la conversazione esistente (conversazione non di gruppo con i due membri)
    err := db.c.QueryRow(`
        SELECT T1.conversation_id
        FROM conversation_members AS T1
        JOIN conversation_members AS T2 ON T1.conversation_id = T2.conversation_id
        JOIN conversations AS C ON T1.conversation_id = C.id
        WHERE T1.user_id = ? AND T2.user_id = ? AND C.is_group = 0
    `, user1ID, user2ID).Scan(&convID)

    if err == nil {
        return convID, nil // Conversazione trovata
    }
    
    if !errors.Is(err, sql.ErrNoRows) {
        return 0, fmt.Errorf("errore nel controllo conversazione esistente: %w", err)
    }

    // 3. Conversazione non trovata: creala.
    
    tx, err := db.c.Begin()
    if err != nil {
        return 0, fmt.Errorf("impossibile avviare la transazione: %w", err)
    }
    
    // Inserisci la nuova conversazione come non-gruppo (is_group = 0)
    res, err := tx.Exec("INSERT INTO conversations (is_group) VALUES (0)")
    if err != nil {
        _ = tx.Rollback()
        return 0, fmt.Errorf("impossibile creare la conversazione: %w", err)
    }
    
    lastInsertId, err := res.LastInsertId()
    if err != nil {
        _ = tx.Rollback()
        return 0, fmt.Errorf("impossibile ottenere l'ID della conversazione: %w", err)
    }
    convID = uint64(lastInsertId)

    // Aggiungi i due membri (is_admin = 0 nelle 1:1)
    _, err = tx.Exec("INSERT INTO conversation_members (conversation_id, user_id, is_admin) VALUES (?, ?, 0)", convID, user1ID)
    if err != nil {
        _ = tx.Rollback()
        return 0, fmt.Errorf("impossibile aggiungere il primo membro: %w", err)
    }
    
    _, err = tx.Exec("INSERT INTO conversation_members (conversation_id, user_id, is_admin) VALUES (?, ?, 0)", convID, user2ID)
    if err != nil {
        _ = tx.Rollback()
        return 0, fmt.Errorf("impossibile aggiungere il secondo membro: %w", err)
    }
    
    // Commit
    if err = tx.Commit(); err != nil {
        return 0, fmt.Errorf("impossibile fare il commit della transazione: %w", err)
    }

    return convID, nil
}