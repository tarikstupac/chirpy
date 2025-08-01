package main

import (
	"database/sql"
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
	"github.com/tarikstupac/chirpy/internal/auth"
	"github.com/tarikstupac/chirpy/internal/database"
)

func (cfg *apiConfig) polkaWebhookHandler(w http.ResponseWriter, req *http.Request) {
	key, err := auth.GetAPIKey(req.Header)
	if err != nil || key != cfg.polkaKey {
		respondWithError(w, http.StatusUnauthorized, "Invalid API key", err)
		return
	}
	type polkaPayload struct {
		Event string `json:"event"`
		Data  struct {
			UserID string `json:"user_id"`
		} `json:"data"`
	}
	payload := polkaPayload{}
	decoder := json.NewDecoder(req.Body)
	err = decoder.Decode(&payload)

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Something went wrong", err)
		return
	}
	if payload.Event != "user.upgraded" {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	parsedUserID, err := uuid.Parse(payload.Data.UserID)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid user ID", err)
		return
	}
	_, err = cfg.db.RetrieveUserById(req.Context(), parsedUserID)
	if err != nil {
		if err == sql.ErrNoRows {
			respondWithError(w, http.StatusNotFound, "User not found", err)
			return
		}
		respondWithError(w, http.StatusInternalServerError, "Error retrieving user", err)
		return
	}
	err = cfg.db.UpdateUserChirpyRedStatus(req.Context(), database.UpdateUserChirpyRedStatusParams{
		ID:          parsedUserID,
		IsChirpyRed: true,
	})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error updating user status", err)
		return
	}
	respondWithJSON(w, http.StatusNoContent, nil)
}
