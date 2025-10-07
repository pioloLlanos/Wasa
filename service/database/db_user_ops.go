package database

import (
    "database/sql"
    "fmt"
    "strings" 
)


// CreateUser crea un nuovo utente.
func (db *appdbimpl) CreateUser(name string) (uint64, error) {
    // ... (Implementazione omessa per brevità, è corretta) ...
    res, err := db.c.Exec("INSERT INTO users (name, photo_url) VALUES (?, '')", name)
    if err != nil {
        if strings.Contains(err.Error(), "UNIQUE constraint failed") {
            return 0, AppErrorNomeGiaInUso
        }
        return 0, fmt.Errorf("errore nell'inserimento dell'utente: %w", err)
    }

    lastInsertId, err := res.LastInsertId()
    if err != nil {
        return 0, fmt.Errorf("errore nel recupero del LastInsertId: %w", err)
    }
    return uint64(lastInsertId), nil
}

// GetUserByName recupera l'ID di un utente dato il suo nome.
func (db *appdbimpl) GetUserByName(name string) (uint64, error) {
    // ... (Implementazione omessa per brevità, è corretta) ...
    var userID uint64
    err := db.c.QueryRow("SELECT id FROM users WHERE name = ?", name).Scan(&userID)
    if err != nil {
        return 0, err
    }
    return userID, nil
}

// CheckUserExists verifica l'esistenza di un utente tramite ID.
func (db *appdbimpl) CheckUserExists(id uint64) error {
    // ... (Implementazione omessa per brevità, è corretta) ...
    var count int
    err := db.c.QueryRow("SELECT COUNT(id) FROM users WHERE id = ?", id).Scan(&count)
    if err != nil {
        return err
    }
    if count == 0 {
        return sql.ErrNoRows
    }
    return nil
}

// SetMyUserName aggiorna il nome di un utente esistente.
func (db *appdbimpl) SetMyUserName(id uint64, newName string) error {
    // ... (Implementazione omessa per brevità, è corretta) ...
    res, err := db.c.Exec("UPDATE users SET name = ? WHERE id = ?", newName, id)
    if err != nil {
        if strings.Contains(err.Error(), "UNIQUE constraint failed") {
            return AppErrorNomeGiaInUso
        }
        return fmt.Errorf("errore nell'aggiornamento del nome utente: %w", err)
    }
    rowsAffected, _ := res.RowsAffected()
    if rowsAffected == 0 {
        return sql.ErrNoRows 
    }
    return nil
}

// SetUserPhotoURL aggiorna l'URL della foto profilo di un utente esistente.
func (db *appdbimpl) SetUserPhotoURL(id uint64, url string) error {
	res, err := db.c.Exec("UPDATE users SET photo_url = ? WHERE id = ?", url, id)
	if err != nil {
		return fmt.Errorf("errore nell'aggiornamento della foto utente: %w", err)
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return sql.ErrNoRows 
	}
	return nil
}

// SearchUsers cerca gli utenti il cui nome corrisponde parzialmente alla query.
func (db *appdbimpl) SearchUsers(query string) ([]User, error) {
	searchPattern := "%" + query + "%"
	
	rows, err := db.c.Query("SELECT id, name, photo_url FROM users WHERE name LIKE ? LIMIT 10", searchPattern)
	if err != nil {
		return nil, fmt.Errorf("errore nella query di ricerca utenti: %w", err)
	}
	defer rows.Close()

	var users []User
	
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ID, &u.Name, &u.PhotoURL); err != nil {
			return nil, fmt.Errorf("errore nella scansione della riga utente: %w", err)
		}
		users = append(users, u)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("errore dopo l'iterazione della query: %w", err)
	}
	
	return users, nil
}