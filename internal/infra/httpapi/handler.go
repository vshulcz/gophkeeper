package httpapi

import (
	"encoding/json"
	"errors"
	"gophkeeper/internal/app/auth"
	"gophkeeper/internal/app/vault"
	"gophkeeper/internal/domain"
	"gophkeeper/internal/domain/secret"
	"gophkeeper/internal/infra/httpapi/dto"
	"gophkeeper/internal/ports"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Server handles HTTP requests.
type Server struct {
	auth   *auth.Service
	vault  *vault.Service
	signer ports.TokenSigner
	ttl    time.Duration
}

// NewServer creates a Server.
func NewServer(auth *auth.Service, vault *vault.Service, signer ports.TokenSigner, tokenTTL time.Duration) *Server {
	return &Server{auth: auth, vault: vault, signer: signer, ttl: tokenTTL}
}

// Routes returns the http.Handler with all routes configured.
func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", s.handleHealth)
	mux.HandleFunc("/register", s.handleRegister)
	mux.HandleFunc("/login", s.handleLogin)
	mux.Handle("/items", s.withAuth(http.HandlerFunc(s.handleItems)))
	mux.Handle("/items/", s.withAuth(http.HandlerFunc(s.handleItemByID)))
	return mux
}

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

func (s *Server) handleRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req dto.CredentialsRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if req.Username == "" || req.Password == "" {
		writeError(w, http.StatusBadRequest, "username and password required")
		return
	}
	token, err := s.auth.Register(r.Context(), req.Username, req.Password)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, dto.TokenResponse{Token: token, ExpiresInSeconds: int64(s.ttl.Seconds())})
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req dto.CredentialsRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if req.Username == "" || req.Password == "" {
		writeError(w, http.StatusBadRequest, "username and password required")
		return
	}
	token, err := s.auth.Login(r.Context(), req.Username, req.Password)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, dto.TokenResponse{Token: token, ExpiresInSeconds: int64(s.ttl.Seconds())})
}

func (s *Server) handleItems(w http.ResponseWriter, r *http.Request) {
	userID := userIDFromContext(r)
	if userID == uuid.Nil {
		writeError(w, http.StatusUnauthorized, "missing user")
		return
	}

	switch r.Method {
	case http.MethodGet:
		since := int64(0)
		if v := r.URL.Query().Get("since"); v != "" {
			parsed, err := strconv.ParseInt(v, 10, 64)
			if err != nil {
				writeError(w, http.StatusBadRequest, "invalid since")
				return
			}
			since = parsed
		}
		items, err := s.vault.SyncSince(r.Context(), userID, since)
		if err != nil {
			writeDomainError(w, err)
			return
		}
		resp := make([]dto.ItemResponse, 0, len(items))
		for _, item := range items {
			resp = append(resp, itemResponseFromDomain(item))
		}
		log.Printf("sync items user=%s since=%d count=%d", userID, since, len(items))
		writeJSON(w, http.StatusOK, resp)
	case http.MethodPost:
		var req dto.ItemUpsertRequest
		if err := decodeJSON(r, &req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid json")
			return
		}
		itemType, err := secret.ParseType(req.Type)
		if err != nil {
			writeDomainError(w, err)
			return
		}
		itemID := uuid.New()
		if req.ID != "" {
			parsed, err := uuid.Parse(req.ID)
			if err != nil {
				writeError(w, http.StatusBadRequest, "invalid id")
				return
			}
			itemID = parsed
		}
		item, err := s.vault.Upsert(r.Context(), userID, itemID, itemType, req.Payload, req.Version)
		if err != nil {
			writeDomainError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, itemResponseFromDomain(item))
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (s *Server) handleItemByID(w http.ResponseWriter, r *http.Request) {
	userID := userIDFromContext(r)
	if userID == uuid.Nil {
		writeError(w, http.StatusUnauthorized, "missing user")
		return
	}
	idStr := strings.TrimPrefix(r.URL.Path, "/items/")
	itemID, err := uuid.Parse(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	switch r.Method {
	case http.MethodGet:
		item, err := s.vault.Get(r.Context(), userID, itemID)
		if err != nil {
			writeDomainError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, itemResponseFromDomain(item))
	case http.MethodDelete:
		version := int64(0)
		if v := r.URL.Query().Get("version"); v != "" {
			parsed, err := strconv.ParseInt(v, 10, 64)
			if err != nil {
				writeError(w, http.StatusBadRequest, "invalid version")
				return
			}
			version = parsed
		}
		item, err := s.vault.Delete(r.Context(), userID, itemID, version)
		if err != nil {
			writeDomainError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, itemResponseFromDomain(item))
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (s *Server) withAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		parts := strings.SplitN(auth, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			writeError(w, http.StatusUnauthorized, "missing token")
			return
		}
		userID, _, err := s.signer.Verify(parts[1])
		if err != nil {
			writeError(w, http.StatusUnauthorized, "invalid token")
			return
		}
		uid, err := uuid.Parse(userID)
		if err != nil {
			writeError(w, http.StatusUnauthorized, "invalid token")
			return
		}
		ctx := withUserID(r.Context(), uid)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func writeDomainError(w http.ResponseWriter, err error) {
	if errors.Is(err, domain.ErrUnauthorized) {
		writeError(w, http.StatusUnauthorized, err.Error())
		return
	}
	if errors.Is(err, domain.ErrConflict) {
		log.Printf("conflict: %v", err)
		writeError(w, http.StatusConflict, err.Error())
		return
	}
	if errors.Is(err, domain.ErrNotFound) {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	if errors.Is(err, domain.ErrInvalidItemType) {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeError(w, http.StatusInternalServerError, "internal error")
}

func itemResponseFromDomain(item secret.Item) dto.ItemResponse {
	return dto.ItemResponse{
		ID:        item.ID.String(),
		Type:      string(item.Type),
		Payload:   item.Payload,
		Deleted:   item.Deleted,
		Version:   item.Version,
		UpdatedAt: item.UpdatedAt,
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

func decodeJSON(r *http.Request, v any) error {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		return err
	}
	if err := dec.Decode(&struct{}{}); err != io.EOF {
		return errors.New("invalid json")
	}
	return nil
}
