package api

import (
	"encoding/json"
	"net/http"
)

// decodeJSON è un helper per decodificare il corpo JSON della richiesta.
// Include anche la limitazione della dimensione del corpo della richiesta e la gestione dei campi sconosciuti.
// Accetta 'w' per permettere a http.MaxBytesReader di scrivere un errore in caso di overflow del corpo.
func (rt *_router) decodeJSON(w http.ResponseWriter, r *http.Request, v interface{}) error {
	// Limita la dimensione del corpo della richiesta a 1MB (1024*1024 byte) per sicurezza.
	// Se la dimensione viene superata, http.MaxBytesReader scrive un errore 413 a 'w'.
	r.Body = http.MaxBytesReader(w, r.Body, 1024*1024)

	// Crea un decoder
	dec := json.NewDecoder(r.Body)

	// Assicurati che vengano decodificati solo i campi noti per prevenire errori
	dec.DisallowUnknownFields()

	// Decodifica il JSON nel valore fornito
	err := dec.Decode(v)

	if err != nil {
		// Logga l'errore se necessario
		rt.baseLogger.WithError(err).Error("Error decoding JSON body")
		return err
	}

	return nil
}

// writeJSON è un helper per scrivere una risposta JSON
func (rt *_router) writeJSON(w http.ResponseWriter, status int, data interface{}) {
	// Imposta l'header Content-Type
	w.Header().Set("Content-Type", "application/json")

	// Imposta lo status code
	w.WriteHeader(status)

	// Se data è nil, non scrivere un corpo
	if data != nil {
		// Codifica il valore in JSON e scrivilo nel writer
		if err := json.NewEncoder(w).Encode(data); err != nil {
			// Se c'è un errore qui, è troppo tardi per inviare un codice di errore,
			// ma lo loggiamo per la diagnostica.
			rt.baseLogger.WithError(err).Error("Error encoding JSON response")
		}
	}
}
