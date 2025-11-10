package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
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

type Hub struct {
	clients    map[*websocket.Conn]bool
	broadcast  chan []byte
	register   chan *websocket.Conn
	unregister chan *websocket.Conn
}

var (
	db *Database
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
	uploadDir = "uploads"
	dbPath    = "picsapp.db"
)


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
		ID:         id,
		Filename:   handler.Filename,
		URL:        fmt.Sprintf("/uploads/%s", id),
		Likes:      0,
		UploadedAt: time.Now(),
	}

	// Save to database
	if err := db.AddPicture(picture); err != nil {
		log.Printf("Error saving picture to database: %v", err)
		http.Error(w, "Error saving picture", http.StatusInternalServerError)
		return
	}

	// Broadcast update
	pictures, err := db.GetAllPicturesSortedByLikes()
	if err != nil {
		log.Printf("Error getting pictures for broadcast: %v", err)
	} else {
		update, _ := json.Marshal(pictures)
		hub.broadcast <- update
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(picture)
}

func handleList(w http.ResponseWriter, r *http.Request) {
	pictures, err := db.GetLastPictures(30) // 5x6 = 30
	if err != nil {
		log.Printf("Error getting pictures: %v", err)
		http.Error(w, "Error fetching pictures", http.StatusInternalServerError)
		return
	}
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

	if err := db.IncrementLikes(id); err != nil {
		http.Error(w, "Picture not found", http.StatusNotFound)
		return
	}

	// Broadcast update
	pictures, err := db.GetAllPicturesSortedByLikes()
	if err != nil {
		log.Printf("Error getting pictures for broadcast: %v", err)
	} else {
		update, _ := json.Marshal(pictures)
		hub.broadcast <- update
	}

	pic, err := db.GetPicture(id)
	if err != nil {
		http.Error(w, "Picture not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(pic)
}

func handlePresentation(w http.ResponseWriter, r *http.Request) {
	pictures, err := db.GetAllPicturesSortedByLikes()
	if err != nil {
		log.Printf("Error getting pictures: %v", err)
		http.Error(w, "Error fetching pictures", http.StatusInternalServerError)
		return
	}
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
	pictures, err := db.GetAllPicturesSortedByLikes()
	if err != nil {
		log.Printf("Error getting pictures for WebSocket: %v", err)
		pictures = []*Picture{}
	}
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
	// Initialize database
	var err error
	db, err = NewDatabase(dbPath)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	log.Printf("Database initialized: %s", dbPath)

	// Ensure uploads directory exists
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		log.Fatalf("Failed to create uploads directory: %v", err)
	}
	log.Printf("Uploads directory: %s", uploadDir)

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
	r.PathPrefix("/uploads/").Handler(http.StripPrefix("/uploads/", http.FileServer(http.Dir(uploadDir))))
	r.PathPrefix("/").Handler(http.FileServer(http.Dir("build/")))

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	fmt.Printf("Server starting on port %s\n", port)
	fmt.Printf("Database: %s\n", dbPath)
	fmt.Printf("Uploads: %s\n", uploadDir)
	log.Fatal(http.ListenAndServe(":"+port, r))
}

