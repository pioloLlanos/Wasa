package api

import (
	"github.com/pioloLlanos/Wasa/service/database"
	"github.com/gofrs/uuid"
	"github.com/julienschmidt/httprouter"
	"github.com/sirupsen/logrus"
	"net/http"
	"strconv" // 👈 AGGIUNTO
	"strings" // 👈 AGGIUNTO
	"github.com/pioloLlanos/Wasa/service/api/reqcontext" 
)

// httpRouterHandler e _router rimangono invariati...

// httpRouterHandler è la firma per le funzioni che accettano un reqcontext.RequestContext in aggiunta a quelli
// richiesti dal pacchetto httprouter.
type httpRouterHandler func(http.ResponseWriter, *http.Request, httprouter.Params, reqcontext.RequestContext)

// wrap implementa il middleware di logging e AUTENTICAZIONE.
func (rt *_router) wrap(fn httpRouterHandler) func(http.ResponseWriter, *http.Request, httprouter.Params) {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		reqUUID, err := uuid.NewV4()
		if err != nil {
			rt.baseLogger.WithError(err).Error("can't generate a request UUID")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		// Assumiamo che reqcontext.RequestContext contenga un campo UserID uint64
		var ctx = reqcontext.RequestContext{
			ReqUUID: reqUUID,
		}

		// Create a request-specific logger
		ctx.Logger = rt.baseLogger.WithFields(logrus.Fields{
			"reqid":     ctx.ReqUUID.String(),
			"remote-ip": r.RemoteAddr,
		})

		// ----------------------------------------------------
		// LOGICA DI AUTENTICAZIONE
		// ----------------------------------------------------
		
		authHeader := r.Header.Get("Authorization")
		
		// 1. Check header
		if !strings.HasPrefix(authHeader, "Bearer ") {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		// 2. Extract token
		tokenString := authHeader[len("Bearer "):]
		
		// 3. Convert ID
		userID, err := strconv.ParseUint(tokenString, 10, 64)
		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		// 4. Check existence in DB
		if err := rt.db.CheckUserExists(userID); err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		
		// 5. Inject ID into context
		ctx.UserID = userID // 👈 Necessita del campo UserID in reqcontext.RequestContext
		
		// ----------------------------------------------------

		// Call the next handler in chain
		fn(w, r, ps, ctx)
	}
}