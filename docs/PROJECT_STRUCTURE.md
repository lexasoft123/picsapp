# Project Structure

This document describes the organization and structure of the PicsApp codebase.

## Directory Structure

```
picsapp/
├── docs/                    # Documentation (this directory)
│   ├── README.md
│   ├── PROJECT_STRUCTURE.md
│   ├── DATABASE.md
│   ├── API.md
│   └── OBJECT_MODEL.md
│
├── src/                     # React frontend source
│   ├── App.jsx              # Main app component with routing
│   ├── App.css              # App-level styles
│   ├── index.jsx            # React entry point
│   ├── index.css            # Global styles
│   └── components/          # React components
│       ├── MainPage.jsx     # Home page with upload & grid
│       ├── MainPage.css
│       ├── Upload.jsx       # Upload component
│       ├── Upload.css
│       ├── PictureGrid.jsx  # Grid layout component
│       ├── PictureGrid.css
│       ├── PictureCard.jsx  # Individual picture card
│       ├── PictureCard.css
│       ├── Presentation.jsx # Presentation page (sorted by likes)
│       └── Presentation.css
│
├── public/                  # Static public files
│   └── index.html           # HTML template
│
├── build/                   # React production build (generated)
│   ├── index.html
│   ├── static/
│   │   ├── css/
│   │   └── js/
│   └── asset-manifest.json
│
├── uploads/                 # Uploaded images (generated)
│   ├── original/            # Original files before conversion
│   └── *.webp               # Converted WebP files
│
├── main.go                  # Go backend server (main entry point)
├── database.go              # Database operations and schema
├── go.mod                   # Go module dependencies
├── go.sum                   # Go dependency checksums
├── package.json             # Node.js dependencies and scripts
├── package-lock.json        # NPM lock file
├── build.sh                 # Build script for production
├── Dockerfile               # Docker container definition
├── docker-compose.yml       # Docker Compose configuration
├── .dockerignore            # Docker ignore patterns
├── picsapp.db               # SQLite database (generated)
├── picsapp                  # Compiled Go binary (generated)
└── README.md                # Project README
```

## Backend Structure (Go)

### `main.go`
Main server file containing:
- **HTTP Server Setup**: Gorilla Mux router configuration
- **WebSocket Hub**: Real-time communication hub
- **API Handlers**: REST endpoint handlers
- **Image Processing**: WebP conversion worker
- **Middleware**: Request logging
- **Static File Serving**: React build and uploads

**Key Components:**
- `Picture` struct - Picture data model
- `Hub` struct - WebSocket connection manager
- `handleUpload()` - File upload handler
- `handleList()` - Get pictures list
- `handleLike()` - Like a picture
- `handlePresentation()` - Get sorted pictures
- `handleWebSocket()` - WebSocket connection handler
- `startConversionWorker()` - Background image processor
- `processConversionTask()` - Convert image to WebP

### `database.go`
Database layer containing:
- **Database Struct**: SQLite connection wrapper
- **Schema Initialization**: Table creation and migrations
- **CRUD Operations**: Picture and task management
- **Transaction Handling**: Task claiming with locks

**Key Functions:**
- `NewDatabase()` - Initialize database connection
- `initSchema()` - Create tables and indexes
- `AddPicture()` - Insert new picture
- `GetPicture()` - Retrieve single picture
- `GetLastPictures()` - Get recent pictures
- `GetAllPicturesSortedByLikes()` - Get sorted list
- `IncrementLikes()` - Update like count
- `CreateConversionTask()` - Queue conversion
- `ClaimNextTask()` - Atomic task claiming
- `MarkTaskCompleted()` / `MarkTaskFailed()` - Update task status

## Frontend Structure (React)

### `src/App.jsx`
Main application component:
- **Routing**: React Router setup
- **Navigation**: NavLinks component
- **Routes**: `/` (MainPage) and `/presentation` (Presentation)

### `src/components/MainPage.jsx`
Home page component:
- **State Management**: Pictures list, loading, upload status
- **WebSocket Connection**: Real-time updates
- **File Upload**: Drag & drop and file input
- **Picture Display**: Grid of last 30 pictures
- **Like Functionality**: Like button handler

**Key Features:**
- Fetches last 30 pictures sorted by upload date
- WebSocket for real-time picture updates
- Drag & drop upload support
- Upload status messages

### `src/components/Presentation.jsx`
Presentation page component:
- **Dual Layout**: Grid and Spiral views
- **Sorting**: Pictures sorted by likes (descending)
- **Real-time Updates**: WebSocket for live like updates
- **Animation**: Smooth transitions when likes change
- **Spiral Layout**: Archimedean spiral positioning

**Key Features:**
- Grid layout with rank numbers
- Spiral layout with physics-based animation
- Debug mode for spiral visualization
- Slow animation mode for testing
- URL hash-based layout switching (`#spiral`)

### `src/components/PictureGrid.jsx`
Grid layout component:
- Displays pictures in responsive grid
- Handles like button clicks
- Shows loading states

### `src/components/PictureCard.jsx`
Individual picture card:
- Displays image thumbnail
- Like button and count
- Hover effects

### `src/components/Upload.jsx`
Upload component:
- File input button
- Drag & drop zone
- Upload progress indicator

## Build System

### `build.sh`
Production build script:
1. Builds React frontend (`npm run build`)
2. Compiles Go backend (`go build -o picsapp`)

### `package.json`
NPM configuration:
- **Scripts**: `start`, `build`, `test`
- **Dependencies**: React, React Router
- **Proxy**: Development proxy to Go server

### `go.mod`
Go module configuration:
- Module name: `picsapp`
- Go version: 1.21
- Dependencies: Gorilla packages, SQLite, imaging libraries

## Docker Configuration

### `Dockerfile`
Multi-stage build:
1. Build React frontend
2. Build Go backend
3. Create minimal runtime image

### `docker-compose.yml`
Docker Compose configuration:
- Service definition
- Volume mounts
- Port mapping
- Environment variables

## Data Flow

### Upload Flow
1. User uploads file → `MainPage.jsx` → `POST /api/upload`
2. Server saves to `uploads/original/`
3. Server creates conversion task in database
4. Background worker processes task
5. Worker converts to WebP, saves to `uploads/`
6. Worker creates/updates picture record
7. Worker broadcasts update via WebSocket
8. Frontend receives update and refreshes

### Like Flow
1. User clicks like → `MainPage.jsx` → `POST /api/pictures/{id}/like`
2. Server increments likes in database
3. Server fetches all pictures sorted by likes
4. Server broadcasts update via WebSocket
5. All connected clients receive update
6. Frontend updates UI with new order

### Presentation Flow
1. User navigates to `/presentation`
2. Component fetches `GET /api/presentation`
3. Component connects to WebSocket
4. Component receives initial data via WebSocket
5. Component displays in grid or spiral layout
6. Real-time updates via WebSocket maintain sort order

## Key Design Decisions

1. **SQLite Database**: Embedded database for simplicity, no external dependencies
2. **WebP Conversion**: Automatic conversion for better performance and storage
3. **Background Processing**: Async conversion to avoid blocking uploads
4. **WebSocket Hub**: Centralized real-time updates for all clients
5. **SPA Routing**: React Router with server-side fallback to index.html
6. **Dual Storage**: Original files temporarily stored, then converted and original deleted
7. **Atomic Task Claiming**: Database-level locking prevents duplicate processing

