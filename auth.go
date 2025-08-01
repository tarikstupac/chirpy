package main

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/tarikstupac/chirpy/internal/auth"
	"github.com/tarikstupac/chirpy/internal/database"
)

type LoginReq struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginRes struct {
	ID           uuid.UUID `json:"id"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	Email        string    `json:"email"`
	Token        string    `json:"token"`
	RefreshToken string    `json:"refresh_token"`
	IsChirpyRed  bool      `json:"is_chirpy_red"`
}

type RefreshRes struct {
	Token string `json:"token"`
}

func (cfg *apiConfig) loginHandler(w http.ResponseWriter, req *http.Request) {
	userLogin := LoginReq{}
	decoder := json.NewDecoder(req.Body)
	err := decoder.Decode(&userLogin)

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error decoding body", err)
		return
	}

	user, err := cfg.db.RetrieveUserByEmail(req.Context(), userLogin.Email)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "User not found", err)
		return
	}

	err = auth.CheckPasswordHash(user.HashedPassword, userLogin.Password)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Incorrect email or password", err)
		return
	}

	token, err := auth.MakeJWT(user.ID, cfg.secretKey, time.Hour)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Something went wrong", err)
		return
	}
	refreshToken, err := auth.MakeRefreshToken()
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Something went wrong", err)
		return
	}
	_, err = cfg.db.CreateRefreshToken(req.Context(), database.CreateRefreshTokenParams{
		Token:     refreshToken,
		UserID:    user.ID,
		ExpiresAt: time.Now().Add(60 * 24 * time.Hour),
	})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Something went wrong", err)
		return
	}
	userRes := LoginRes{ID: user.ID, CreatedAt: user.CreatedAt, UpdatedAt: user.UpdatedAt, Email: user.Email, Token: token, RefreshToken: refreshToken, IsChirpyRed: user.IsChirpyRed}
	respondWithJSON(w, http.StatusOK, userRes)
}

func (cfg *apiConfig) refreshHandler(w http.ResponseWriter, req *http.Request) {
	token, err := auth.GetBearerToken(req.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Invalid credentials", err)
		return
	}
	user, err := cfg.db.RetrieveUserByRefreshToken(req.Context(), token)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Invalid credentials", err)
		return
	}
	newToken, err := auth.MakeJWT(user.ID, cfg.secretKey, time.Hour)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Something went wrong", err)
		return
	}

	refreshRes := RefreshRes{Token: newToken}
	respondWithJSON(w, http.StatusOK, refreshRes)
}

func (cfg *apiConfig) revokeHandler(w http.ResponseWriter, req *http.Request) {
	token, err := auth.GetBearerToken(req.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Invalid credentials", err)
		return
	}
	err = cfg.db.RevokeRefreshToken(req.Context(), token)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Something went wrong", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
