package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

type Picture struct {
	ID        string    `json:"id"`
	Filename  string    `json:"filename"`
	URL       string    `json:"url"`
	Likes     int       `json:"likes"`
	UploadedAt time.Time `json:"uploadedAt"`
}

type PictureStore struct {
	mu       sync.RWMutex
	pictures map[string]*Picture
	order    []string // Maintain upload order
}

type Hub struct {
	clients    map[*websocket.Conn]bool
	broadcast  chan []byte
	register   chan *websocket.Conn
	unregister chan *websocket.Conn
}

var (
	store = &PictureStore{
		pictures: make(map[string]*Picture),
		order:    make([]string, 0),
	}
	hub = &Hub{
		clients:    make(map[*websocket.Conn]bool),
		broadcast:  make(chan []byte),
		register:   make(chan *websocket.Conn),
		unregister: make(chan *websocket.Conn),
	}
	upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
)

func (s *PictureStore) Add(picture *Picture) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pictures[picture.ID] = picture
	s.order = append(s.order, picture.ID)
}

func (s *PictureStore) Get(id string) (*Picture, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	pic, ok := s.pictures[id]
	return pic, ok
}

func (s *PictureStore) GetLast(n int) []*Picture {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	var result []*Picture
	start := len(s.order) - n
	if start < 0 {
		start = 0
	}
	
	for i := len(s.order) - 1; i >= start; i-- {
		if pic, ok := s.pictures[s.order[i]]; ok {
			result = append(result, pic)
		}
	}
	
	return result
}

func (s *PictureStore) GetAllSortedByLikes() []*Picture {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	var result []*Picture
	for _, pic := range s.pictures {
		result = append(result, pic)
	}
	
	sort.Slice(result, func(i, j int) bool {
		if result[i].Likes == result[j].Likes {
			return result[i].UploadedAt.After(result[j].UploadedAt)
		}
		return result[i].Likes > result[j].Likes
	})
	
	return result
}

func (s *PictureStore) IncrementLikes(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if pic, ok := s.pictures[id]; ok {
		pic.Likes++
		return true
	}
	return false
}

func (h *Hub) run() {
	for {
		select {
		case conn := <-h.register:
			h.clients[conn] = true
		case conn := <-h.unregister:
			if _, ok := h.clients[conn]; ok {
				delete(h.clients, conn)
				conn.Close()
			}
		case message := <-h.broadcast:
			for conn := range h.clients {
				err := conn.WriteMessage(websocket.TextMessage, message)
				if err != nil {
					delete(h.clients, conn)
					conn.Close()
				}
			}
		}
	}
}

func handleUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	err := r.ParseMultipartForm(10 << 20) // 10 MB max
	if err != nil {
		http.Error(w, "Error parsing form", http.StatusBadRequest)
		return
	}

	file, handler, err := r.FormFile("picture")
	if err != nil {
		http.Error(w, "Error retrieving file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Create uploads directory if it doesn't exist
	uploadDir := "uploads"
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		http.Error(w, "Error creating upload directory", http.StatusInternalServerError)
		return
	}

	// Generate unique filename
	id := fmt.Sprintf("%d_%s", time.Now().UnixNano(), handler.Filename)
	filename := filepath.Join(uploadDir, id)

	dst, err := os.Create(filename)
	if err != nil {
		http.Error(w, "Error saving file", http.StatusInternalServerError)
		return
	}
	defer dst.Close()

	if _, err := io.Copy(dst, file); err != nil {
		http.Error(w, "Error saving file", http.StatusInternalServerError)
		return
	}

	// Create picture record
	picture := &Picture{
		ID:        id,
		Filename:  handler.Filename,
		URL:       fmt.Sprintf("/uploads/%s", id),
		Likes:     0,
		UploadedAt: time.Now(),
	}

	store.Add(picture)

	// Broadcast update
	pictures := store.GetAllSortedByLikes()
	update, _ := json.Marshal(pictures)
	hub.broadcast <- update

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(picture)
}

func handleList(w http.ResponseWriter, r *http.Request) {
	pictures := store.GetLast(30) // 5x6 = 30
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(pictures)
}

func handleLike(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	vars := mux.Vars(r)
	id := vars["id"]

	if store.IncrementLikes(id) {
		// Broadcast update
		pictures := store.GetAllSortedByLikes()
		update, _ := json.Marshal(pictures)
		hub.broadcast <- update

		if pic, ok := store.Get(id); ok {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(pic)
		} else {
			http.Error(w, "Picture not found", http.StatusNotFound)
		}
	} else {
		http.Error(w, "Picture not found", http.StatusNotFound)
	}
}

func handlePresentation(w http.ResponseWriter, r *http.Request) {
	pictures := store.GetAllSortedByLikes()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(pictures)
}

func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}

	hub.register <- conn

	// Send initial data
	pictures := store.GetAllSortedByLikes()
	initial, _ := json.Marshal(pictures)
	conn.WriteMessage(websocket.TextMessage, initial)

	// Keep connection alive
	go func() {
		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				hub.unregister <- conn
				break
			}
		}
	}()
}

func main() {
	// Start hub
	go hub.run()

	r := mux.NewRouter()

	// API routes
	r.HandleFunc("/api/upload", handleUpload).Methods("POST")
	r.HandleFunc("/api/pictures", handleList).Methods("GET")
	r.HandleFunc("/api/pictures/{id}/like", handleLike).Methods("POST")
	r.HandleFunc("/api/presentation", handlePresentation).Methods("GET")
	r.HandleFunc("/ws", handleWebSocket)

	// Serve static files
	r.PathPrefix("/uploads/").Handler(http.StripPrefix("/uploads/", http.FileServer(http.Dir("uploads/"))))
	r.PathPrefix("/").Handler(http.FileServer(http.Dir("build/")))

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	fmt.Printf("Server starting on port %s\n", port)
	log.Fatal(http.ListenAndServe(":"+port, r))
}

