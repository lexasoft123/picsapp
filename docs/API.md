# API Reference

This document describes all API endpoints, WebSocket protocol, request/response formats, and error handling.

## Base URL

- **Development**: `http://localhost:8080`
- **Production**: Configured via `PORT` environment variable (default: 8080)

## REST API Endpoints

### Upload Picture

Upload a new picture file for processing.

**Endpoint**: `POST /api/upload`

**Content-Type**: `multipart/form-data`

**Request Body**:
- `picture` (file): Image file (JPEG, PNG, GIF, WebP)
- Max size: 10 MB

**Response** (200 OK):
```json
{
  "status": "queued"
}
```

**Response** (400 Bad Request):
- `"Error parsing form"` - Invalid multipart form
- `"Error retrieving file"` - File field missing or invalid

**Response** (405 Method Not Allowed):
- `"Method not allowed"` - Wrong HTTP method

**Response** (500 Internal Server Error):
- `"Error creating upload directory"` - Filesystem error
- `"Error saving file"` - File write error
- `"Error queueing image conversion"` - Database error

**Example**:
```bash
curl -X POST http://localhost:8080/api/upload \
  -F "picture=@image.jpg"
```

**Processing Flow**:
1. File saved to `uploads/original/` with timestamp-based name
2. Conversion task created in database
3. Background worker processes conversion
4. WebSocket broadcast sent when complete

---

### Get Pictures List

Get the last 30 uploaded pictures, sorted by upload date (newest first).

**Endpoint**: `GET /api/pictures`

**Response** (200 OK):
```json
[
  {
    "id": "1762801393825964000.webp",
    "filename": "download.jpeg",
    "url": "/uploads/1762801393825964000.webp",
    "likes": 5,
    "uploadedAt": "2024-01-15T10:30:00Z"
  },
  ...
]
```

**Response** (500 Internal Server Error):
- `"Error fetching pictures"` - Database error

**Example**:
```bash
curl http://localhost:8080/api/pictures
```

**Notes**:
- Returns maximum 30 pictures
- Ordered by `uploaded_at DESC`
- Used by home page grid

---

### Like a Picture

Increment the like count for a picture.

**Endpoint**: `POST /api/pictures/{id}/like`

**Path Parameters**:
- `id` (string): Picture ID (e.g., `1762801393825964000.webp`)

**Response** (200 OK):
```json
{
  "id": "1762801393825964000.webp",
  "filename": "download.jpeg",
  "url": "/uploads/1762801393825964000.webp",
  "likes": 6,
  "uploadedAt": "2024-01-15T10:30:00Z"
}
```

**Response** (404 Not Found):
- `"Picture not found"` - Invalid picture ID

**Response** (405 Method Not Allowed):
- `"Method not allowed"` - Wrong HTTP method

**Example**:
```bash
curl -X POST http://localhost:8080/api/pictures/1762801393825964000.webp/like
```

**Side Effects**:
- Like count incremented in database
- WebSocket broadcast sent to all connected clients
- Broadcast contains all pictures sorted by likes

---

### Get Presentation Data

Get all pictures sorted by likes (descending), then by upload date (descending).

**Endpoint**: `GET /api/presentation`

**Response** (200 OK):
```json
[
  {
    "id": "1762801393825964000.webp",
    "filename": "download.jpeg",
    "url": "/uploads/1762801393825964000.webp",
    "likes": 10,
    "uploadedAt": "2024-01-15T10:30:00Z"
  },
  {
    "id": "1762801393825964001.webp",
    "filename": "image.png",
    "url": "/uploads/1762801393825964001.webp",
    "likes": 8,
    "uploadedAt": "2024-01-15T11:00:00Z"
  },
  ...
]
```

**Response** (500 Internal Server Error):
- `"Error fetching pictures"` - Database error

**Example**:
```bash
curl http://localhost:8080/api/presentation
```

**Notes**:
- Returns all pictures (no limit)
- Ordered by `likes DESC, uploaded_at DESC`
- Used by presentation page

---

## WebSocket API

### Connection

**Endpoint**: `WS /ws` or `WSS /ws`

**Protocol**: WebSocket (RFC 6455)

**Connection URL**:
- Development: `ws://localhost:8080/ws`
- Production: `wss://your-domain.com/ws` (if HTTPS)

**Upgrade Headers**: Automatically handled by browser WebSocket API

**Connection Flow**:
1. Client connects to `/ws`
2. Server upgrades HTTP connection to WebSocket
3. Server sends initial data (all pictures sorted by likes)
4. Server broadcasts updates on picture changes

### Message Format

All messages are JSON-encoded strings.

#### Initial Message (Server → Client)

Sent immediately after connection:

```json
[
  {
    "id": "1762801393825964000.webp",
    "filename": "download.jpeg",
    "url": "/uploads/1762801393825964000.webp",
    "likes": 10,
    "uploadedAt": "2024-01-15T10:30:00Z"
  },
  ...
]
```

#### Update Message (Server → Client)

Sent when:
- New picture is uploaded and converted
- Picture is liked
- Picture is re-converted

Format: Same as initial message (array of all pictures sorted by likes)

#### Client Messages (Client → Server)

Currently, clients don't send messages. The connection is kept alive by reading any incoming messages (ping/pong handled by WebSocket protocol).

### Broadcast Events

The server broadcasts updates in these scenarios:

1. **New Picture Uploaded**:
   - After conversion task completes
   - New picture added to database
   - All clients receive updated sorted list

2. **Picture Liked**:
   - After like count incremented
   - All clients receive updated sorted list

3. **Picture Re-converted**:
   - After legacy picture conversion
   - Picture record updated
   - All clients receive updated list

### Connection Management

**Reconnection**: Clients should implement automatic reconnection with exponential backoff.

**Example Client Code**:
```javascript
const ws = new WebSocket('ws://localhost:8080/ws');

ws.onopen = () => {
  console.log('WebSocket connected');
};

ws.onmessage = (event) => {
  const pictures = JSON.parse(event.data);
  // Update UI with pictures
};

ws.onclose = () => {
  // Reconnect after delay
  setTimeout(() => connectWebSocket(), 3000);
};
```

---

## Static File Serving

### Uploaded Images

**Endpoint**: `GET /uploads/{filename}`

**Example**: `GET /uploads/1762801393825964000.webp`

**Response**: Image file (WebP format)

**Content-Type**: Determined by file extension

**Notes**:
- Files are served directly from `uploads/` directory
- All images are converted to WebP format
- Original files are deleted after conversion

### React Build Files

**Endpoint**: `GET /static/{path}`

**Example**: `GET /static/css/main.6d0ffb3f.css`

**Response**: Static asset file

**Notes**:
- Served from `build/static/` directory
- Handled by React build process

### SPA Routing Fallback

**Endpoint**: `GET /*` (any path not matching API or static files)

**Response**: `build/index.html`

**Notes**:
- Enables React Router client-side routing
- All non-API routes serve index.html
- React Router handles routing on client side

---

## Error Responses

All error responses follow this format:

**HTTP Status Codes**:
- `200` - Success
- `400` - Bad Request (invalid input)
- `404` - Not Found (resource doesn't exist)
- `405` - Method Not Allowed (wrong HTTP method)
- `500` - Internal Server Error (server error)

**Error Response Format**:
```
Error message text
```

**Example**:
```
Picture not found
```

---

## Rate Limiting

Currently, there is no rate limiting implemented. Consider adding:
- Upload rate limiting (e.g., 10 uploads per minute)
- Like rate limiting (e.g., 1 like per second per IP)
- WebSocket connection limits

---

## CORS

CORS is not explicitly configured. The server accepts requests from:
- Same origin (production)
- Development proxy (React dev server on port 3000)

For cross-origin access, configure CORS headers in the Go server.

---

## Authentication

Currently, there is no authentication. All endpoints are publicly accessible.

Consider adding:
- User authentication
- Upload permissions
- Like tracking per user
- Admin endpoints

---

## Request/Response Examples

### Complete Upload Flow

```bash
# 1. Upload picture
curl -X POST http://localhost:8080/api/upload \
  -F "picture=@image.jpg"

# Response: {"status":"queued"}

# 2. Wait for conversion (poll or use WebSocket)

# 3. Get pictures list
curl http://localhost:8080/api/pictures

# 4. Like a picture
curl -X POST http://localhost:8080/api/pictures/1762801393825964000.webp/like

# 5. Get presentation data
curl http://localhost:8080/api/presentation
```

### WebSocket Example

```javascript
const ws = new WebSocket('ws://localhost:8080/ws');

ws.onopen = () => {
  console.log('Connected');
};

ws.onmessage = (event) => {
  const pictures = JSON.parse(event.data);
  console.log('Received', pictures.length, 'pictures');
  // Update UI
};

ws.onerror = (error) => {
  console.error('WebSocket error:', error);
};

ws.onclose = () => {
  console.log('Disconnected');
};
```

---

## API Versioning

Currently, there is no API versioning. All endpoints are under `/api/`.

For future versions, consider:
- `/api/v1/upload`
- `/api/v2/upload`
- Version negotiation via headers

