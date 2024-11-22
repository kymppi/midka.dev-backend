package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	lib "github.com/kymppi/midka.dev-backend/internal"
)

var (
	cachedTracks  lib.FriendlyRecentTracks
	cacheMutex    sync.RWMutex
	lastCacheTime time.Time
	cacheDuration = 5 * time.Second
)

type RequestLog struct {
	Route      string
	Method     string
	UserAgent  string
	Cached     bool
	Duration   time.Duration
	StatusCode int
}

func logRequest(r *http.Request, cached bool, duration time.Duration, statusCode int) {
	requestLog := RequestLog{
		Route:      r.URL.Path,
		Method:     r.Method,
		UserAgent:  r.UserAgent(),
		Cached:     cached,
		Duration:   duration,
		StatusCode: statusCode,
	}

	log.Printf(
		"Request - Route: %s, Method: %s, UserAgent: %s, Cached: %v, Duration: %v, Status: %d",
		requestLog.Route,
		requestLog.Method,
		requestLog.UserAgent,
		requestLog.Cached,
		requestLog.Duration,
		requestLog.StatusCode,
	)
}

func main() {
	log.SetOutput(os.Stdout)
	apiKey := os.Getenv("LASTFM_API_KEY")
	user := os.Getenv("LASTFM_USER")
	listen := os.Getenv("LISTEN")
	if listen == "" {
		listen = ":9123"
	}

	http.HandleFunc("/recent-tracks", func(w http.ResponseWriter, r *http.Request) {
		startTime := time.Now()
		statusCode := http.StatusOK

		limitParam := r.URL.Query().Get("limit")
		limit := 10
		if limitParam != "" {
			var err error
			limit, err = strconv.Atoi(limitParam)
			if err != nil {
				http.Error(w, "Invalid limit parameter", http.StatusBadRequest)
				logRequest(r, false, time.Since(startTime), http.StatusBadRequest)
				return
			}
		}
		if limit > 50 {
			http.Error(w, "Limit is too high (>50)", http.StatusBadRequest)
			logRequest(r, false, time.Since(startTime), http.StatusBadRequest)
			return
		}

		tracks, err := getCachedRecentTracks(apiKey, user, limit)
		if err != nil {
			statusCode = http.StatusInternalServerError
			json.NewEncoder(w).Encode([]lib.FriendlyTrack{})
			logRequest(r, false, time.Since(startTime), statusCode)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		json.NewEncoder(w).Encode(tracks)

		logRequest(r, tracks.Cached, time.Since(startTime), statusCode)
	})

	fmt.Printf("Server starting on %s\n", listen)
	http.ListenAndServe(listen, nil)
}

func getCachedRecentTracks(apiKey, user string, limit int) (lib.FriendlyRecentTracks, error) {
	cacheMutex.RLock()
	now := time.Now()
	if now.Sub(lastCacheTime) < cacheDuration {
		tracks := cachedTracks
		tracks.Cached = true
		cacheMutex.RUnlock()
		return tracks, nil
	}
	cacheMutex.RUnlock()

	tracks, err := lib.GetRecentTracksWithFriendlyFormat(apiKey, user, limit)
	if err != nil {
		return lib.FriendlyRecentTracks{}, err
	}

	cacheMutex.Lock()
	cachedTracks = tracks
	lastCacheTime = now
	cacheMutex.Unlock()

	return tracks, nil
}
