package main

import (
	"encoding/json"
	"net/http"
	"slices"
	"time"

	"github.com/google/uuid"
	"github.com/tarikstupac/chirpy/internal/auth"
	"github.com/tarikstupac/chirpy/internal/database"
)

type Chirp struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Body      string    `json:"body"`
	UserID    uuid.UUID `json:"user_id"`
}

func (cfg *apiConfig) createChirpHandler(w http.ResponseWriter, req *http.Request) {
	type chirpCreateReq struct {
		Body string `json:"body"`
	}
	chirpCreate := chirpCreateReq{}
	decoder := json.NewDecoder(req.Body)
	err := decoder.Decode(&chirpCreate)

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Something went wrong", err)
		return
	}

	token, err := auth.GetBearerToken(req.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Invalid credentials", err)
		return
	}

	userID, err := auth.ValidateJWT(token, cfg.secretKey)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Invalid credentials", err)
		return
	}

	if len(chirpCreate.Body) > 140 {
		respondWithError(w, http.StatusBadRequest, "Chirp is too long", nil)
		return
	}

	cleanedMsg := removeProfanityFromMessage(chirpCreate.Body)
	chirpCreate.Body = cleanedMsg

	chirp, err := cfg.db.CreateChirp(req.Context(), database.CreateChirpParams{Body: chirpCreate.Body, UserID: userID})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error creating a chirp", err)
		return
	}
	newChirp := Chirp{ID: chirp.ID, CreatedAt: chirp.CreatedAt, UpdatedAt: chirp.UpdatedAt, Body: chirp.Body, UserID: chirp.UserID}
	respondWithJSON(w, http.StatusCreated, newChirp)
}

func (cfg *apiConfig) retrieveChirpsHandler(w http.ResponseWriter, req *http.Request) {
	authorId := req.URL.Query().Get("author_id")
	sortBy := req.URL.Query().Get("sort")
	var chirps []database.Chirp
	if authorId != "" {
		parsedId, err := uuid.Parse(authorId)
		if err != nil {
			respondWithError(w, http.StatusBadRequest, "Author ID is not a valid UUID", err)
			return
		}
		chirps, err = cfg.db.RetrieveChirpsByUserID(req.Context(), parsedId)
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "Error fetching chirps", err)
			return
		}
	} else {
		var err error
		chirps, err = cfg.db.RetrieveAllChirps(req.Context())
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "Error fetching chirps", err)
			return
		}
	}
	var jsonChirps []Chirp
	for _, c := range chirps {
		jsonChirps = append(jsonChirps, Chirp{ID: c.ID, CreatedAt: c.CreatedAt, UpdatedAt: c.UpdatedAt, Body: c.Body, UserID: c.UserID})
	}
	slices.SortFunc(jsonChirps, func(a, b Chirp) int {
		switch sortBy {
		case "asc", "":
			if a.CreatedAt.Before(b.CreatedAt) {
				return -1
			} else {
				return 1
			}
		case "desc":
			if a.CreatedAt.After(b.CreatedAt) {
				return -1
			} else {
				return 1
			}
		}
		return 0
	})
	respondWithJSON(w, http.StatusOK, jsonChirps)
}

func (cfg *apiConfig) retrieveChirpByIdHandler(w http.ResponseWriter, req *http.Request) {
	chirpID := req.PathValue("ID")
	if chirpID == "" {
		respondWithError(w, http.StatusBadRequest, "No request parameter supplied", nil)
		return
	}
	parsedId, err := uuid.Parse(chirpID)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "ID is not a valid UUID", err)
		return
	}
	dbChirp, err := cfg.db.RetrieveChirpByID(req.Context(), parsedId)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "Chirp not found!", err)
		return
	}
	chirp := Chirp{ID: dbChirp.ID, CreatedAt: dbChirp.CreatedAt, UpdatedAt: dbChirp.UpdatedAt, Body: dbChirp.Body, UserID: dbChirp.UserID}
	respondWithJSON(w, http.StatusOK, chirp)
}

func (cfg *apiConfig) deleteChirpByIdHandler(w http.ResponseWriter, req *http.Request) {
	token, err := auth.GetBearerToken(req.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Invalid credentials", err)
		return
	}

	userID, err := auth.ValidateJWT(token, cfg.secretKey)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Invalid credentials", err)
		return
	}

	chirpID := req.PathValue("ID")
	if chirpID == "" {
		respondWithError(w, http.StatusBadRequest, "No request parameter supplied", nil)
		return
	}
	parsedId, err := uuid.Parse(chirpID)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "ID is not a valid UUID", err)
		return
	}
	dbChirp, err := cfg.db.RetrieveChirpByID(req.Context(), parsedId)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "Chirp not found!", err)
		return
	}
	if dbChirp.UserID != userID {
		respondWithError(w, http.StatusForbidden, "You can only delete your own chirps", nil)
		return
	}
	err = cfg.db.DeleteChirpByID(req.Context(), dbChirp.ID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error deleting chirp", err)
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusNoContent)
}
