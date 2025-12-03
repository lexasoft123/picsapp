# Object Model

This document describes the data structures, models, and their relationships in PicsApp.

## Backend Models (Go)

### Picture

Represents a picture in the system.

**Location**: `main.go`

**Definition**:
```go
type Picture struct {
    ID         string    `json:"id"`
    Filename   string    `json:"filename"`
    URL        string    `json:"url"`
    Likes      int       `json:"likes"`
    UploadedAt time.Time `json:"uploadedAt"`
}
```

**Fields**:

| Field | Type | JSON Key | Description |
|-------|------|----------|-------------|
| `ID` | `string` | `id` | Unique identifier (e.g., `1762801393825964000.webp`) |
| `Filename` | `string` | `filename` | Original filename from upload |
| `URL` | `string` | `url` | URL path to serve image (e.g., `/uploads/1762801393825964000.webp`) |
| `Likes` | `int` | `likes` | Number of likes received |
| `UploadedAt` | `time.Time` | `uploadedAt` | Upload timestamp (RFC3339 format in JSON) |

**JSON Example**:
```json
{
  "id": "1762801393825964000.webp",
  "filename": "download.jpeg",
  "url": "/uploads/1762801393825964000.webp",
  "likes": 5,
  "uploadedAt": "2024-01-15T10:30:00Z"
}
```

**Usage**:
- Stored in SQLite `pictures` table
- Serialized to JSON for API responses
- Used in WebSocket broadcasts

---

### ConversionTask

Represents an image conversion task in the queue.

**Location**: `database.go`

**Definition**:
```go
type ConversionTask struct {
    ID           int64
    OriginalPath string
    OriginalName string
    PictureID    *string
    Status       string
    Error        *string
    CreatedAt    time.Time
    UpdatedAt    time.Time
}
```

**Fields**:

| Field | Type | Description |
|-------|------|-------------|
| `ID` | `int64` | Auto-incrementing task ID |
| `OriginalPath` | `string` | Full filesystem path to original image |
| `OriginalName` | `string` | Original filename (for display) |
| `PictureID` | `*string` | Existing picture ID (nil for new uploads) |
| `Status` | `string` | Task status: `pending`, `processing`, `completed`, `failed` |
| `Error` | `*string` | Error message if status is `failed` |
| `CreatedAt` | `time.Time` | Task creation timestamp |
| `UpdatedAt` | `time.Time` | Last update timestamp |

**Status Values**:
- `pending`: Queued, waiting for processing
- `processing`: Currently being converted
- `completed`: Successfully converted
- `failed`: Conversion failed

**Usage**:
- Stored in SQLite `conversion_tasks` table
- Managed by background worker
- Not exposed via API (internal use)

---

### Hub

Manages WebSocket connections for real-time updates.

**Location**: `main.go`

**Definition**:
```go
type Hub struct {
    clients    map[*websocket.Conn]bool
    broadcast  chan []byte
    register   chan *websocket.Conn
    unregister chan *websocket.Conn
}
```

**Fields**:

| Field | Type | Description |
|-------|------|-------------|
| `clients` | `map[*websocket.Conn]bool` | Active WebSocket connections |
| `broadcast` | `chan []byte` | Channel for broadcasting messages |
| `register` | `chan *websocket.Conn` | Channel for new connections |
| `unregister` | `chan *websocket.Conn` | Channel for disconnections |

**Methods**:
- `run()`: Main event loop for managing connections

**Usage**:
- Single global instance
- Handles all WebSocket connections
- Broadcasts picture updates to all clients

---

### Database

Database connection wrapper.

**Location**: `database.go`

**Definition**:
```go
type Database struct {
    db *sql.DB
}
```

**Fields**:

| Field | Type | Description |
|-------|------|-------------|
| `db` | `*sql.DB` | SQLite database connection |

**Methods**:
- `NewDatabase(dbPath string) (*Database, error)`: Initialize database
- `Close() error`: Close database connection
- `AddPicture(picture *Picture) error`: Insert picture
- `GetPicture(id string) (*Picture, error)`: Get picture by ID
- `GetLastPictures(n int) ([]*Picture, error)`: Get recent pictures
- `GetAllPicturesSortedByLikes() ([]*Picture, error)`: Get sorted pictures
- `IncrementLikes(id string) error`: Increment like count
- `UpdatePictureFile(oldID, newID, newURL string) error`: Update picture file
- `CreateConversionTask(path, name, pictureID string) error`: Create task
- `ClaimNextTask() (*ConversionTask, error)`: Claim next pending task
- `MarkTaskCompleted(id int64) error`: Mark task as completed
- `MarkTaskFailed(id int64, msg string) error`: Mark task as failed

---

## Frontend Models (JavaScript/React)

### Picture Object

Matches backend Picture structure.

**Type**: Plain JavaScript object

**Definition**:
```javascript
{
  id: string,           // e.g., "1762801393825964000.webp"
  filename: string,     // e.g., "download.jpeg"
  url: string,          // e.g., "/uploads/1762801393825964000.webp"
  likes: number,        // e.g., 5
  uploadedAt: string    // ISO 8601 timestamp, e.g., "2024-01-15T10:30:00Z"
}
```

**Usage**:
- Received from API endpoints
- Received via WebSocket
- Stored in React component state
- Rendered in UI components

**Example**:
```javascript
const picture = {
  id: "1762801393825964000.webp",
  filename: "download.jpeg",
  url: "/uploads/1762801393825964000.webp",
  likes: 5,
  uploadedAt: "2024-01-15T10:30:00Z"
};
```

---

## Component State Models

### MainPage State

**Location**: `src/components/MainPage.jsx`

**State Structure**:
```javascript
{
  pictures: Picture[],        // Array of picture objects
  loading: boolean,            // Loading state
  dragActive: boolean,        // Drag & drop active state
  uploading: boolean,         // Upload in progress
  uploadMessage: string       // Upload status message
}
```

**State Management**:
- `useState` hooks for local state
- WebSocket for real-time updates
- `fetch` for initial data load

---

### Presentation State

**Location**: `src/components/Presentation.jsx`

**State Structure**:
```javascript
{
  pictures: Picture[],         // Array of picture objects (sorted by likes)
  loading: boolean,           // Loading state
  layout: 'grid' | 'spiral',  // Layout mode
  swappingIds: Set<string>,   // IDs of pictures currently animating
  isInitialLoad: boolean,     // First load flag
  containerSize: {             // Container dimensions
    width: number,
    height: number
  },
  debugSpiral: boolean,       // Debug mode for spiral
  slowAnimation: boolean      // Slow animation mode
}
```

**Refs**:
- `wsRef`: WebSocket connection reference
- `prevPositionsRef`: Previous picture positions (Map)
- `positionsRef`: Current picture positions (object)
- `targetsRef`: Target positions for animation (object)
- `animRef`: Animation frame reference
- `containerRef`: Container DOM element reference

**Spiral Layout State**:
```javascript
{
  positions: {                 // Current positions
    [id]: { x: number, y: number, size: number }
  },
  targets: {                   // Target positions
    [id]: { x: number, y: number, size: number }
  },
  animState: {                  // Animation state
    active: boolean,
    progress: number,
    start: object,
    end: object,
    startOrder: string[],
    endOrder: string[],
    changedIds: string[]
  }
}
```

---

## Data Flow Models

### Upload Flow

```
User Upload
  ↓
FormData { picture: File }
  ↓
POST /api/upload
  ↓
Server: Save to uploads/original/
  ↓
Server: Create ConversionTask (status: pending)
  ↓
Background Worker: Claim task
  ↓
Worker: Convert to WebP
  ↓
Worker: Save to uploads/
  ↓
Worker: Create/Update Picture record
  ↓
Worker: Delete original file
  ↓
Worker: Mark task completed
  ↓
Worker: Broadcast via WebSocket
  ↓
Frontend: Receive update, refresh UI
```

### Like Flow

```
User Clicks Like
  ↓
POST /api/pictures/{id}/like
  ↓
Server: Increment likes in database
  ↓
Server: Get all pictures sorted by likes
  ↓
Server: Broadcast via WebSocket
  ↓
All Clients: Receive update, refresh UI
```

### WebSocket Message Flow

```
Client Connects
  ↓
Server: Send initial data (all pictures)
  ↓
Client: Render initial state
  ↓
[Event: Upload/Like]
  ↓
Server: Broadcast update
  ↓
All Clients: Receive update
  ↓
Clients: Update state, re-render
```

---

## Relationship Diagram

```
┌─────────────┐
│  Conversion │
│    Task     │──┐
└─────────────┘  │
                 │ (optional)
┌─────────────┐  │
│   Picture   │◄─┘
└─────────────┘
       │
       │ (many)
       │
┌──────▼──────┐
│   WebSocket │
│    Hub      │
└──────┬──────┘
       │
       │ (broadcasts)
       │
┌──────▼──────┐
│   Clients   │
│  (Browser)  │
└─────────────┘
```

**Relationships**:
- **ConversionTask → Picture**: Optional (nil for new uploads, set for re-conversions)
- **Picture → Hub**: Many-to-one (all pictures broadcast via single hub)
- **Hub → Clients**: One-to-many (hub manages multiple WebSocket connections)

---

## Type Conversions

### Backend to Frontend

**Time Format**:
- Backend: `time.Time` (Go)
- JSON: ISO 8601 string (e.g., `"2024-01-15T10:30:00Z"`)
- Frontend: JavaScript `Date` or string

**Example**:
```go
// Backend
picture.UploadedAt = time.Now()
// JSON: "2024-01-15T10:30:00Z"
// Frontend
const date = new Date(picture.uploadedAt);
```

**Null Handling**:
- Backend: `*string` (pointer, nil = NULL)
- JSON: `null` or string value
- Frontend: `null` or string

**Example**:
```go
// Backend
task.PictureID = nil  // or &stringValue
// JSON: null or "1762801393825964000.webp"
// Frontend
const pictureId = task.pictureId; // null or string
```

---

## Validation Rules

### Picture ID
- Format: `{timestamp}.webp`
- Timestamp: Nanoseconds since epoch
- Extension: Always `.webp`

### Filename
- Original filename from upload
- No validation (preserved as-is)

### URL
- Format: `/uploads/{id}`
- Must match picture ID

### Likes
- Minimum: 0
- Incremented atomically
- No maximum limit

### UploadedAt
- Format: RFC3339 / ISO 8601
- Timezone: UTC
- Set on upload

---

## Serialization

### JSON Serialization

**Backend (Go)**:
```go
json.Marshal(picture)  // Picture → JSON
json.Unmarshal(data, &picture)  // JSON → Picture
```

**Frontend (JavaScript)**:
```javascript
JSON.stringify(picture)  // Object → JSON
JSON.parse(json)  // JSON → Object
```

### WebSocket Messages

All WebSocket messages are JSON-encoded strings:
```javascript
// Server sends
const message = JSON.stringify(pictures);
conn.WriteMessage(websocket.TextMessage, []byte(message));

// Client receives
const pictures = JSON.parse(event.data);
```

---

## Error Models

### API Errors

**Format**: Plain text string

**Examples**:
- `"Picture not found"`
- `"Error parsing form"`
- `"Error fetching pictures"`

### Database Errors

**Handling**:
- Returned as Go `error` type
- Converted to HTTP status codes
- Logged server-side

### Conversion Errors

**Storage**: Stored in `ConversionTask.Error` field

**Format**: Error message string

**Example**: `"convert to webp: unsupported image format"`

