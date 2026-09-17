package main

import (
	"encoding/json"
	"net/http"
	"strings"
)

const maxChirpLength = 140

// badWords is a set: we only care whether a word is in it.
var badWords = map[string]struct{}{
	"kerfuffle": {},
	"sharbert":  {},
	"fornax":    {},
}

// handlerValidateChirp checks a chirp's length and hides bad words.
func handlerValidateChirp(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Body string `json:"body"`
	}
	type returnVals struct {
		CleanedBody string `json:"cleaned_body"`
	}

	params := parameters{}
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		respondWithError(w, http.StatusBadRequest, "Couldn't decode parameters", err)
		return
	}

	// Note: len() counts bytes, not characters. Emoji count as more than 1.
	if len(params.Body) > maxChirpLength {
		respondWithError(w, http.StatusBadRequest, "Chirp is too long", nil)
		return
	}

	respondWithJSON(w, http.StatusOK, returnVals{
		CleanedBody: cleanBody(params.Body),
	})
}

// cleanBody replaces each bad word with "****".
// It has no HTTP code in it, so it is easy to unit test.
func cleanBody(body string) string {
	words := strings.Split(body, " ")
	for i, word := range words {
		if _, isBad := badWords[strings.ToLower(word)]; isBad {
			words[i] = "****"
		}
	}
	return strings.Join(words, " ")
}
