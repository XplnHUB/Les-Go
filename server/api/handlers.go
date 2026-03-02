package api

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/XplnHUB/Les-Go/server/db"
	"github.com/XplnHUB/Les-Go/server/ws"
	"github.com/golang-jwt/jwt/v5"
)

var jwtSecret = []byte("super-secret-lesgo-key-for-dev")

/////////////////////////////////////////////////////////////////////////////
// Models
/////////////////////////////////////////////////////////////////////////////

type RegisterReq struct {
	Username  string `json:"username"`
	Password  string `json:"password"`
	PublicKey string `json:"public_key"`
}

type LoginReq struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type SendMsgReq struct {
	To        string `json:"to"`
	Encrypted string `json:"encrypted_data"`
}

type AckMsgReq struct {
	MessageID string `json:"message_id"`
}

type MsgResponse struct {
	ID        string `json:"id"`
	From      string `json:"from"`
	Encrypted string `json:"encrypted_data"`
	Timestamp string `json:"timestamp"`
}

/////////////////////////////////////////////////////////////////////////////
// Handlers
/////////////////////////////////////////////////////////////////////////////

type Server struct {
	DB  *db.InMemoryDB
	Hub *ws.Hub
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

func (s *Server) HandleRegister() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req RegisterReq
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "Invalid request body")
			return
		}

		if err := s.DB.CreateUser(req.Username, req.Password, req.PublicKey); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}

		writeJSON(w, http.StatusCreated, map[string]string{"message": "User registered"})
	}
}

func (s *Server) HandleLogin() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req LoginReq
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "Invalid request body")
			return
		}

		user, err := s.DB.Authenticate(req.Username, req.Password)
		if err != nil {
			writeError(w, http.StatusUnauthorized, "Invalid credentials")
			return
		}

		// Generate JWT
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"username": user.Username,
			"exp":      time.Now().Add(time.Hour * 72).Unix(),
		})

		tokenString, err := token.SignedString(jwtSecret)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "Could not generate token")
			return
		}

		writeJSON(w, http.StatusOK, map[string]string{"token": tokenString})
	}
}

func (s *Server) HandleGetKey() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// e.g. /api/users/username/key
		parts := strings.Split(r.URL.Path, "/")
		if len(parts) < 4 {
			writeError(w, http.StatusBadRequest, "Invalid URL")
			return
		}
		username := parts[3]

		key, err := s.DB.GetPublicKey(username)
		if err != nil {
			writeError(w, http.StatusNotFound, "User not found")
			return
		}

		writeJSON(w, http.StatusOK, map[string]string{"public_key": key})
	}
}

// Middleware
func authMiddleware(next func(w http.ResponseWriter, r *http.Request, username string)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			// Query param fallback for WS route
			tokenStr := r.URL.Query().Get("token")
			if tokenStr == "" {
				writeError(w, http.StatusUnauthorized, "Missing Authorization Header")
				return
			}
			authHeader = "Bearer " + tokenStr
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			return jwtSecret, nil
		})

		if err != nil || !token.Valid {
			writeError(w, http.StatusUnauthorized, "Invalid Token")
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			writeError(w, http.StatusUnauthorized, "Invalid Claims")
			return
		}

		username := claims["username"].(string)
		next(w, r, username)
	}
}

func (s *Server) HandleSendMessage() http.HandlerFunc {
	return authMiddleware(func(w http.ResponseWriter, r *http.Request, sender string) {
		var req SendMsgReq
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "Invalid JSON")
			return
		}

		_, err := s.DB.StoreMessage(sender, req.To, req.Encrypted)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "Could not store message")
			return
		}

		// Push to WS Hub
		event := ws.Event{
			Type:    "CHAT_MESSAGE",
			Payload: req.Encrypted,
			From:    sender,
		}

		// Try forwarding
		s.Hub.Forward(req.To, event)

		writeJSON(w, http.StatusOK, map[string]string{"status": "sent"})
	})
}

func (s *Server) HandleGetUnread() http.HandlerFunc {
	return authMiddleware(func(w http.ResponseWriter, r *http.Request, username string) {
		msgs := s.DB.GetUnreadMessages(username)
		var res []MsgResponse

		for _, msg := range msgs {
			res = append(res, MsgResponse{
				ID:        msg.ID,
				From:      msg.Sender,
				Encrypted: msg.Encrypted,
				Timestamp: msg.CreatedAt.Format(time.RFC3339),
			})
		}

		writeJSON(w, http.StatusOK, res)
	})
}

func (s *Server) HandleAckMessage() http.HandlerFunc {
	return authMiddleware(func(w http.ResponseWriter, r *http.Request, username string) {
		var req AckMsgReq
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "Invalid JSON")
			return
		}

		if err := s.DB.MarkMessageAsRead(username, req.MessageID); err != nil {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}

		writeJSON(w, http.StatusOK, map[string]string{"status": "acknowledged"})
	})
}

func (s *Server) HandleWS() http.HandlerFunc {
	return authMiddleware(func(w http.ResponseWriter, r *http.Request, username string) {
		s.Hub.ServeWS(w, r, username)
	})
}
