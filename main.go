package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/chai2010/webp"
	"github.com/disintegration/imaging"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	_ "golang.org/x/image/webp"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
)

type Picture struct {
	ID         string    `json:"id"`
	Filename   string    `json:"filename"`
	URL        string    `json:"url"`
	Likes      int       `json:"likes"`
	UploadedAt time.Time `json:"uploadedAt"`
}

type Hub struct {
	clients    map[*websocket.Conn]bool
	broadcast  chan []byte
	register   chan *websocket.Conn
	unregister chan *websocket.Conn
}

var (
	db  *Database
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
	uploadDir   = "uploads"
	originalDir = "uploads/original"
	dbPath      = "picsapp.db"
	logger      = log.New(os.Stdout, "", log.LstdFlags|log.Lmicroseconds)
)

func logInfo(format string, args ...interface{}) {
	logger.Printf("[INFO] "+format, args...)
}

func logWarn(format string, args ...interface{}) {
	logger.Printf("[WARN] "+format, args...)
}

func logError(format string, args ...interface{}) {
	logger.Printf("[ERROR] "+format, args...)
}

func (h *Hub) run() {
	for {
		select {
		case conn := <-h.register:
			h.clients[conn] = true
			logInfo("websocket client connected (clients=%d)", len(h.clients))
		case conn := <-h.unregister:
			if _, ok := h.clients[conn]; ok {
				delete(h.clients, conn)
				conn.Close()
				logInfo("websocket client disconnected (clients=%d)", len(h.clients))
			}
		case message := <-h.broadcast:
			for conn := range h.clients {
				err := conn.WriteMessage(websocket.TextMessage, message)
				if err != nil {
					delete(h.clients, conn)
					conn.Close()
					logWarn("broadcast failed to client: %v", err)
				}
			}
		}
	}
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rw := &responseWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rw, r)
		duration := time.Since(start)
		logInfo("%s %s -> %d (%s)", r.Method, r.URL.Path, rw.status, duration)
	})
}

type responseWriter struct {
	http.ResponseWriter
	status int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Flush() {
	if flusher, ok := rw.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

func (rw *responseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if h, ok := rw.ResponseWriter.(http.Hijacker); ok {
		return h.Hijack()
	}
	return nil, nil, errors.New("hijacker not supported")
}

func (rw *responseWriter) Push(target string, opts *http.PushOptions) error {
	if pusher, ok := rw.ResponseWriter.(http.Pusher); ok {
		return pusher.Push(target, opts)
	}
	return http.ErrNotSupported
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

	if err := os.MkdirAll(originalDir, 0755); err != nil {
		http.Error(w, "Error creating upload directory", http.StatusInternalServerError)
		return
	}

	idBase := strconv.FormatInt(time.Now().UnixNano(), 10)
	ext := strings.ToLower(filepath.Ext(handler.Filename))
	if ext == "" {
		ext = ".img"
	}
	originalName := fmt.Sprintf("%s%s", idBase, ext)
	originalPath := filepath.Join(originalDir, originalName)

	dst, err := os.Create(originalPath)
	if err != nil {
		logError("create original file failed: %v", err)
		http.Error(w, "Error saving file", http.StatusInternalServerError)
		return
	}
	if _, err := io.Copy(dst, file); err != nil {
		dst.Close()
		logError("write original file failed: %v", err)
		http.Error(w, "Error saving file", http.StatusInternalServerError)
		return
	}
	dst.Close()

	if err := db.CreateConversionTask(originalPath, handler.Filename, ""); err != nil {
		logError("create conversion task failed: %v", err)
		http.Error(w, "Error queueing image conversion", http.StatusInternalServerError)
		return
	}

	logInfo("queued image for conversion: %s", handler.Filename)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "queued"})
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
		logError("get pictures for broadcast failed: %v", err)
	} else {
		update, _ := json.Marshal(pictures)
		hub.broadcast <- update
		logInfo("broadcast likes update (picture=%s)", id)
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
		logError("websocket upgrade failed: %v", err)
		return
	}

	hub.register <- conn

	// Send initial data
	pictures, err := db.GetAllPicturesSortedByLikes()
	if err != nil {
		logError("get pictures for websocket failed: %v", err)
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
				logWarn("websocket read error: %v", err)
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

	logInfo("database initialized: %s", dbPath)

	// Ensure uploads directory exists
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		log.Fatalf("Failed to create uploads directory: %v", err)
	}
	if err := os.MkdirAll(originalDir, 0755); err != nil {
		log.Fatalf("Failed to create original uploads directory: %v", err)
	}
	logInfo("uploads directory: %s", uploadDir)

	if err := enqueueLegacyConversionTasks(); err != nil {
		logWarn("enqueue legacy conversions: %v", err)
	}

	go startConversionWorker()

	// Start hub
	go hub.run()

	r := mux.NewRouter()

	r.Use(loggingMiddleware)

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

	logInfo("server starting on port %s", port)
	logInfo("database: %s", dbPath)
	logInfo("uploads: %s", uploadDir)
	log.Fatal(http.ListenAndServe(":"+port, r))
}

const maxImageDimension = 1600

func convertToWebP(data []byte) ([]byte, error) {
	img, err := imaging.Decode(bytes.NewReader(data), imaging.AutoOrientation(true))
	if err != nil {
		return nil, err
	}

	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()
	if width > maxImageDimension || height > maxImageDimension {
		img = imaging.Fit(img, maxImageDimension, maxImageDimension, imaging.Lanczos)
	}

	buf := &bytes.Buffer{}
	if err := webp.Encode(buf, img, &webp.Options{Quality: 82}); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func startConversionWorker() {
	for {
		task, err := db.ClaimNextTask()
		if err != nil {
			logError("claim conversion task: %v", err)
			time.Sleep(time.Second)
			continue
		}
		if task == nil {
			time.Sleep(400 * time.Millisecond)
			continue
		}
		logInfo("processing conversion task id=%d file=%s", task.ID, task.OriginalName)
		if err := processConversionTask(task); err != nil {
			logError("conversion task %d failed: %v", task.ID, err)
			db.MarkTaskFailed(task.ID, err.Error())
		} else {
			db.MarkTaskCompleted(task.ID)
			logInfo("conversion task %d completed", task.ID)
		}
	}
}

func processConversionTask(task *ConversionTask) error {
	data, err := os.ReadFile(task.OriginalPath)
	if err != nil {
		return fmt.Errorf("read original: %w", err)
	}

	processed, err := convertToWebP(data)
	if err != nil {
		return fmt.Errorf("convert to webp: %w", err)
	}

	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		return fmt.Errorf("ensure upload dir: %w", err)
	}

	base := strconv.FormatInt(time.Now().UnixNano(), 10)
	if task.PictureID != nil && *task.PictureID != "" {
		trim := strings.TrimSuffix(*task.PictureID, filepath.Ext(*task.PictureID))
		if trim != "" {
			base = trim
		}
	}

	newID := base + ".webp"
	newPath := filepath.Join(uploadDir, newID)
	if _, err := os.Stat(newPath); err == nil {
		base = fmt.Sprintf("%s_%d", base, time.Now().UnixNano())
		newID = base + ".webp"
		newPath = filepath.Join(uploadDir, newID)
	}

	if err := os.WriteFile(newPath, processed, 0644); err != nil {
		return fmt.Errorf("write converted file: %w", err)
	}

	if task.PictureID != nil && *task.PictureID != "" {
		oldID := *task.PictureID
		if err := db.UpdatePictureFile(oldID, newID, fmt.Sprintf("/uploads/%s", newID)); err != nil {
			return fmt.Errorf("update picture record: %w", err)
		}
		oldPath := filepath.Join(uploadDir, oldID)
		if oldPath != newPath {
			if err := os.Remove(oldPath); err != nil && !os.IsNotExist(err) {
				logWarn("warning: remove old file %s: %v", oldPath, err)
			}
		}
	} else {
		picture := &Picture{
			ID:         newID,
			Filename:   task.OriginalName,
			URL:        fmt.Sprintf("/uploads/%s", newID),
			Likes:      0,
			UploadedAt: time.Now(),
		}
		if err := db.AddPicture(picture); err != nil {
			return fmt.Errorf("insert picture: %w", err)
		}
	}

	if err := os.Remove(task.OriginalPath); err != nil && !os.IsNotExist(err) {
		logWarn("remove original file %s: %v", task.OriginalPath, err)
	}

	if pictures, err := db.GetAllPicturesSortedByLikes(); err == nil {
		update, _ := json.Marshal(pictures)
		hub.broadcast <- update
	}
	return nil
}

func enqueueLegacyConversionTasks() error {
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		return err
	}

	// Existing picture records with non-webp ids
	pics, err := db.GetAllPicturesSortedByLikes()
	if err != nil {
		return err
	}
	for _, pic := range pics {
		if !strings.HasSuffix(strings.ToLower(pic.ID), ".webp") {
			path := filepath.Join(uploadDir, pic.ID)
			if _, err := os.Stat(path); err == nil {
				if err := db.CreateConversionTask(path, pic.Filename, pic.ID); err != nil {
					logWarn("queue legacy picture %s: %v", pic.ID, err)
				}
			}
		}
	}

	// Any original files waiting without tasks
	entries, err := os.ReadDir(originalDir)
	if err == nil {
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			path := filepath.Join(originalDir, entry.Name())
			if err := db.CreateConversionTask(path, entry.Name(), ""); err != nil {
				logWarn("queue legacy original %s: %v", entry.Name(), err)
			}
		}
	}

	return nil
}
