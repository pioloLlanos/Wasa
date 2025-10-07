package database

import (
	"database/sql"
	"errors"
	"fmt"
)

// CreateUser crea un nuovo utente nel database.
// Restituisce l'ID del nuovo utente o un errore se il nome è già in uso.
func (db *appdbimpl) CreateUser(name string) (uint64, error) {
	// Verifica prima se il nome è già in uso.
	// Se la query trova una riga, GetUserByName restituirà l'ID senza errori.
	// Se GetUserByName ha successo, significa che il nome è già preso.
	if _, err := db.GetUserByName(name); err == nil {
		return 0, AppErrorNomeGiaInUso
	} else if !errors.Is(err, sql.ErrNoRows) {
		// Se l'errore non è ErrNoRows (ovvero, un errore inaspettato del DB), lo restituiamo.
		return 0, fmt.Errorf("error checking existing user: %w", err)
	}

	// Inserisce il nuovo utente.
	res, err := db.c.Exec(`INSERT INTO users (Name) VALUES (?)`, name)
	if err != nil {
		// In teoria la verifica precedente copre questo caso, ma teniamo per sicurezza.
		if errors.Is(err, sql.ErrNoRows) {
			return 0, AppErrorNomeGiaInUso
		}
		return 0, fmt.Errorf("error creating user: %w", err)
	}

	// Ottiene l'ID dell'utente appena creato.
	lastInsertID, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("error getting last insert ID: %w", err)
	}

	return uint64(lastInsertID), nil
}

// GetUserByName recupera l'ID di un utente dato il suo nome.
// Restituisce sql.ErrNoRows se l'utente non è trovato.
func (db *appdbimpl) GetUserByName(name string) (uint64, error) {
	var userID uint64

	// Esegui la query per selezionare l'ID in base al nome.
	err := db.c.QueryRow(`SELECT ID FROM users WHERE Name = ?`, name).Scan(&userID)

	if errors.Is(err, sql.ErrNoRows) {
		// User not found (expected during initial registration attempt)
		return 0, sql.ErrNoRows
	}

	if err != nil {
		// Unexpected database error
		return 0, fmt.Errorf("error fetching user by name: %w", err)
	}

	return userID, nil
}

// CheckUserExists verifica se un utente con l'ID specificato esiste.
// Usato dal middleware di autenticazione.
func (db *appdbimpl) CheckUserExists(id uint64) error {
	var exists bool
	err := db.c.QueryRow(`SELECT EXISTS(SELECT 1 FROM users WHERE ID = ?)`, id).Scan(&exists)

	if err != nil {
		return fmt.Errorf("error checking user existence: %w", err)
	}

	if !exists {
		return sql.ErrNoRows // Usiamo sql.ErrNoRows come segnale di "non trovato"
	}

	return nil
}

// SetMyUserName implementa l'aggiornamento del nome utente.
func (db *appdbimpl) SetMyUserName(id uint64, name string) error {
	// 1. Verifica se il nuovo nome è già in uso da un altro utente
	existingID, err := db.GetUserByName(name)
	if err == nil && existingID != id {
		return AppErrorNomeGiaInUso
	} else if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("error checking name availability: %w", err)
	}

	// 2. Aggiorna il nome
	res, err := db.c.Exec(`UPDATE users SET Name = ? WHERE ID = ?`, name, id)
	if err != nil {
		return fmt.Errorf("error updating user name: %w", err)
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("error checking rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return sql.ErrNoRows // Nessun utente trovato con quell'ID
	}

	return nil
}

// SetUserPhotoURL implementa l'aggiornamento dell'URL della foto profilo.
func (db *appdbimpl) SetUserPhotoURL(id uint64, url string) error {
	res, err := db.c.Exec(`UPDATE users SET PhotoURL = ? WHERE ID = ?`, url, id)
	if err != nil {
		return fmt.Errorf("error updating user photo URL: %w", err)
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("error checking rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return sql.ErrNoRows // Nessun utente trovato con quell'ID
	}

	return nil
}

// SearchUsers implementa la ricerca di utenti in base al nome.
// Restituisce una lista di oggetti User.
func (db *appdbimpl) SearchUsers(query string) ([]User, error) {
	// Aggiunge i caratteri jolly SQL per la ricerca LIKE
	searchPattern := fmt.Sprintf("%%%s%%", query)

	rows, err := db.c.Query(
		`SELECT ID, Name, PhotoURL FROM users WHERE Name LIKE ? LIMIT 20`, // Limita i risultati per performance
		searchPattern,
	)
	if err != nil {
		return nil, fmt.Errorf("error executing user search query: %w", err)
	}
	// Garantisce che la risorsa sia rilasciata
	defer rows.Close()

	var users []User
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ID, &u.Name, &u.PhotoURL); err != nil {
			return nil, fmt.Errorf("error scanning user search result: %w", err)
		}
		users = append(users, u)
	}

	// Controlla gli errori che possono essersi verificati durante l'iterazione
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating over user search results: %w", err)
	}

	return users, nil
}
