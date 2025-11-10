# PicsApp

A modern picture sharing application with Go backend and React frontend, featuring real-time updates via WebSocket.

## Features

- 📤 Upload pictures with drag & drop
- 🖼️ Display last 30 uploaded pictures in a 5x6 grid
- ❤️ Like pictures
- 📊 Presentation page showing pictures sorted by likes (descending)
- 🔄 Real-time updates via WebSocket
- 🌙 Modern dark theme with smooth animations

## Prerequisites

- Go 1.21 or later
- Node.js 16 or later
- npm or yarn

## Setup

### Backend (Go)

1. Install Go dependencies:
```bash
go mod download
```

2. Run the server:
```bash
go run main.go
```

The server will start on port 8080 (or the PORT environment variable if set).

### Frontend (React)

1. Install dependencies:
```bash
npm install
```

2. Start the development server:
```bash
npm start
```

3. Build for production:
```bash
npm run build
```

## Project Structure

```
picsapp/
├── main.go              # Go backend server
├── go.mod               # Go dependencies
├── package.json         # Node.js dependencies
├── uploads/            # Uploaded pictures (created automatically)
├── build/              # React build output (created after build)
└── src/
    ├── App.jsx         # Main React app with routing
    ├── components/     # React components
    │   ├── MainPage.jsx
    │   ├── Upload.jsx
    │   ├── PictureGrid.jsx
    │   ├── PictureCard.jsx
    │   └── Presentation.jsx
    └── index.jsx       # React entry point
```

## API Endpoints

- `POST /api/upload` - Upload a picture
- `GET /api/pictures` - Get last 30 pictures
- `POST /api/pictures/{id}/like` - Like a picture
- `GET /api/presentation` - Get all pictures sorted by likes
- `WS /ws` - WebSocket connection for real-time updates

## Development

The Go server serves the React build files in production. For development:

1. Run the Go server: `go run main.go`
2. Run React dev server: `npm start` (runs on port 3000)
3. Configure React to proxy API requests to `http://localhost:8080` (add to package.json if needed)

## Production Build

### Option 1: Using the build script (recommended)

```bash
./build.sh
```

This will:
1. Build the React frontend (`npm run build`)
2. Build the Go backend binary (`go build -o picsapp main.go`)

Then run the server:
```bash
./picsapp
```

### Option 2: Manual build

1. Build React app:
```bash
npm run build
```

2. Build Go server:
```bash
go build -o picsapp main.go
```

3. Run the server:
```bash
./picsapp
```

The server will:
- Serve the React app from the `build/` directory
- Handle API requests on `/api/*`
- Handle WebSocket connections on `/ws`
- Serve uploaded pictures from `/uploads/*`

### Environment Variables

- `PORT` - Server port (default: 8080)
  ```bash
  PORT=3000 ./picsapp
  ```

