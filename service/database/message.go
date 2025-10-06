package database

import (
    "database/sql"
    "fmt"
)

// CreateMessage crea e salva un nuovo messaggio nel database.
// FIRMA CORRETTA: convID, senderID, content, replyToID, isPhoto
func (db *appdbimpl) CreateMessage(convID uint64, senderID uint64, content string, replyToID uint64, isPhoto bool) (uint64, error) {
    // ⚠️ Importante: Qui la tua query SQL `INSERT INTO messages...` deve essere aggiornata
    // per includere i campi `reply_to_id` e `is_photo` e i relativi valori!
    
    // Placeholder aggiornato per compilare correttamente
    return 0, fmt.Errorf("funzione omessa per brevità, la firma è stata corretta con replyToID (%d) e isPhoto (%t)", replyToID, isPhoto)
}

// CreateMessageWithPhoto crea e salva un nuovo messaggio contenente una foto nel database.
func (db *appdbimpl) CreateMessageWithPhoto(convID uint64, senderID uint64, url string) (uint64, error) {
	// Questo è un helper che chiama CreateMessage, impostando 'isPhoto' a true 
    // e usa l'URL come contenuto del messaggio.
	
	// Il terzo argomento è l'ID del messaggio a cui si risponde. Qui usiamo 0 
    // assumendo che una foto non sia una risposta (se la tua API lo supporta, modificalo).
    // L'ultimo argomento è 'isPhoto' (true).
    
    // ⚠️ Nota: Qui stiamo riutilizzando CreateMessage con i flag corretti (isPhoto=true, replyToID=0).
    // Assicurati che il tuo campo 'content' o 'url' nella tabella 'messages' possa
    // contenere l'URL della foto.
    
    // Se la tua implementazione di CreateMessage gestisce bene i nuovi parametri, 
    // puoi chiamarla direttamente:
    messageID, err := db.CreateMessage(convID, senderID, url, 0, true) 
    
    if err != nil {
        // Se CreateMessage ritorna un errore di membro non trovato, ripassalo.
        return 0, err
    }
    
    return messageID, nil
}


// DeleteMessage cancella un messaggio, ma solo se l'utente che lo invoca è il mittente.
func (db *appdbimpl) DeleteMessage(msgID uint64, userID uint64) error {
    res, err := db.c.Exec("DELETE FROM messages WHERE id = ? AND sender_id = ?", msgID, userID)
    if err != nil {
        return fmt.Errorf("errore nella cancellazione del messaggio: %w", err)
    }

    rowsAffected, _ := res.RowsAffected()
    if rowsAffected == 0 {
        return sql.ErrNoRows // Messaggio non trovato o l'utente non è il mittente
    }
    return nil
}

// ForwardMessage inoltra un messaggio esistente in una nuova conversazione.
func (db *appdbimpl) ForwardMessage(msgID uint64, senderID uint64, targetConvID uint64) (uint64, error) {
    // ⚠️ Importante: Questa funzione ora deve chiamare CreateMessage con 5 argomenti!
    // Recupera il contenuto e chiama CreateMessage
    return 0, fmt.Errorf("funzione omessa per brevità, l'implementazione deve ora chiamare CreateMessage con 5 argomenti")
}

// AddReaction aggiunge una reazione a un messaggio (richiede la tabella message_reactions).
func (db *appdbimpl) AddReaction(msgID uint64, userID uint64, reaction string) error {
    _, err := db.c.Exec(`
        INSERT INTO message_reactions (message_id, user_id, reaction_type) 
        VALUES (?, ?, ?) 
        ON CONFLICT (message_id, user_id) DO UPDATE SET reaction_type = excluded.reaction_type
    `, msgID, userID, reaction)
    return err
}

// RemoveReaction rimuove la reazione di un utente da un messaggio.
func (db *appdbimpl) RemoveReaction(msgID uint64, userID uint64) error {
    res, err := db.c.Exec("DELETE FROM message_reactions WHERE message_id = ? AND user_id = ?", msgID, userID)
    if err != nil {
        return err
    }
    rowsAffected, _ := res.RowsAffected()
    if rowsAffected == 0 {
        return sql.ErrNoRows // Nessuna reazione da rimuovere
    }
    return nil
}