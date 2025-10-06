package reqcontext

import (
	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

// RequestContext is the context of the request, for request-dependent parameters
type RequestContext struct {
	// ReqUUID is the request unique ID
	ReqUUID uuid.UUID

	// Logger is a custom field logger for the request
	Logger logrus.FieldLogger

	// UserID è l'identificatore dell'utente autenticato (Aggiunto per l'autenticazione)
	UserID uint64 // 👈 QUESTO È IL CAMPO MANCANTE
}