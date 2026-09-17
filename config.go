package main

import (
	"sync/atomic"

	"github.com/AaryanCode69/chirpy/internal/database"
)

// apiConfig holds everything the handlers share:
// the visit counter, the database, and which environment we run in.
type apiConfig struct {
	fileserverHits atomic.Int32
	db             *database.Queries
	platform       string
}
