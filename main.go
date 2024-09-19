package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sync"
	"time"
)

type GithubErrorResponse struct {
	Message          string `json:"message"`
	DocumentationURL string `json:"documentation_url"`
	Status           string `json:"status"`
}

type GithubEvent struct {
	Type  string `json:"type"`
	Actor struct {
		Login string `json:"login"`
	} `json:"actor"`
	Repo struct {
		Name string `json:"name"`
		URL  string `json:"url"`
	} `json:"repo"`
	CreatedAt time.Time `json:"created_at"`
}

type CacheItem struct {
	Data      []GithubEvent
	ExpiresAt time.Time
}

var cache = make(map[string]CacheItem)
var cacheMutex sync.Mutex
var cacheFile = "cache.json"

// Load cache from file
func loadCache() {
	cacheMutex.Lock()
	defer cacheMutex.Unlock()

	file, err := os.ReadFile(cacheFile)
	if err != nil {
		if os.IsNotExist(err) {
			// If file doesn't exist, skip loading
			fmt.Println("Cache file not found, starting fresh")
			return
		}

		log.Fatalf("Error reading cache file: %v", err)
	}

	err = json.Unmarshal(file, &cache)
	if err != nil {
		log.Fatalf("Error parsing cache file: %v", err)
	}

	fmt.Println("Cache loaded successfully")
}

// Save cache to file
func saveCache() {
	cacheMutex.Lock()
	defer cacheMutex.Unlock()

	file, err := json.MarshalIndent(cache, "", " ")
	if err != nil {
		log.Fatalf("Error serializing cache file: %v", err)
	}

	err = os.WriteFile(cacheFile, file, 0644)
	if err != nil {
		log.Fatalf("Error saving cache file: %v", err)
	}

	fmt.Println("Cache saved successfully")
}

func getGithubEvents(username string) ([]GithubEvent, error) {
	cacheKey := fmt.Sprintf("github-events-%s", username)

	// Check existing cache
	cacheMutex.Lock()
	item, found := cache[cacheKey]
	fmt.Printf("Cache found: %v, ExpiresAt: %v\n", found, item.ExpiresAt) // Debugging log
	cacheMutex.Unlock()

	// Check if we have a valid cache hit
	if found {
		fmt.Println("Cache hit, checking expiration...")
		if time.Now().Before(item.ExpiresAt) {
			fmt.Println("Returning cached data")
			return item.Data, nil
		}
		fmt.Println("Cache expired, fetching fresh data")
	} else {
		fmt.Println("Cache miss, fetching fresh data")
	}

	// If not in cache or cache expired, make a request
	githubUrl := fmt.Sprintf("https://api.github.com/users/%s/events", username)
	resp, err := http.Get(githubUrl)
	if err != nil {
		return nil, err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			return
		}
	}(resp.Body)

	// Handling if the username is not found or error occurred
	if resp.StatusCode != http.StatusOK {
		var githubErrorResponse GithubErrorResponse
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}

		err = json.Unmarshal(body, &githubErrorResponse)
		if err != nil {
			return nil, err
		}

		return nil, errors.New(githubErrorResponse.Message)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var events []GithubEvent
	err = json.Unmarshal(body, &events)
	if err != nil {
		return nil, err
	}

	// Store the response in cache with a 10-minute expiration
	cacheMutex.Lock()
	cache[cacheKey] = CacheItem{
		Data:      events,
		ExpiresAt: time.Now().Add(10 * time.Minute),
	}
	fmt.Printf("Cache updated for key: %s, ExpiresAt: %v\n", cacheKey, cache[cacheKey].ExpiresAt) // Debugging log
	cacheMutex.Unlock()

	// Save the cache to a file
	saveCache()

	fmt.Println("Returning fresh data")
	return events, nil
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run main.go [command: github username]")
		return
	}

	loadCache()

	githubUsername := os.Args[1]
	events, err := getGithubEvents(githubUsername)
	if err != nil {
		log.Fatalf("Error fetching events: %v", err)
		return
	}

	for _, event := range events {
		fmt.Printf("Type: %s\n", event.Type)
		fmt.Printf("Actor Login: %s\n", event.Actor.Login)
		fmt.Printf("Repo Name: %s\n", event.Repo.Name)
		fmt.Printf("Repo URL: %s\n", event.Repo.URL)
		fmt.Printf("Created At: %s\n", event.CreatedAt.Format("2006-01-02 15:04:05"))
		fmt.Println("----------------------")
	}

	// Save the cache before exiting
	saveCache()
}
