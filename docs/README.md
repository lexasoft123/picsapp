# PicsApp Documentation

This directory contains comprehensive documentation for the PicsApp project. Use these documents to understand the project structure, database schema, APIs, and object models.

## Documentation Index

1. **[Project Structure](./PROJECT_STRUCTURE.md)** - Overview of the codebase organization, directory structure, and file purposes
2. **[Database Structure](./DATABASE.md)** - Complete database schema, tables, relationships, and data flow
3. **[API Reference](./API.md)** - REST API endpoints, WebSocket protocol, request/response formats
4. **[OpenAPI Specification](./openapi.yaml)** - Machine-readable API specification in OpenAPI 3.0 format
5. **[OpenAPI Guide](./OPENAPI_GUIDE.md)** - How to use, view, and generate code from the OpenAPI spec
6. **[Object Model](./OBJECT_MODEL.md)** - Data structures, models, and their relationships

## Quick Start

- **Backend**: Go 1.21+ with SQLite database
- **Frontend**: React 18 with React Router
- **Real-time**: WebSocket for live updates
- **Image Processing**: Automatic WebP conversion with resizing

## Key Features

- Picture upload with drag & drop
- Real-time like updates via WebSocket
- Two view modes: Grid (home) and Presentation (sorted by likes)
- Automatic image conversion to WebP format
- Background task processing for image conversion

## Architecture Overview

```
┌─────────────┐
│   React     │  Frontend (SPA)
│   Frontend  │  - MainPage (upload & grid)
└──────┬──────┘  - Presentation (sorted view)
       │
       │ HTTP/WebSocket
       │
┌──────▼──────┐
│   Go Server │  Backend
│   (Gorilla) │  - REST API
└──────┬──────┘  - WebSocket Hub
       │
       │ SQL
       │
┌──────▼──────┐
│   SQLite    │  Database
│   Database  │  - Pictures table
└─────────────┘  - Conversion tasks table
```

## Technology Stack

### Backend
- **Go 1.21** - Main server language
- **Gorilla Mux** - HTTP router
- **Gorilla WebSocket** - WebSocket support
- **SQLite** - Embedded database
- **disintegration/imaging** - Image processing
- **chai2010/webp** - WebP encoding

### Frontend
- **React 18** - UI framework
- **React Router 6** - Client-side routing
- **WebSocket API** - Real-time communication

## Environment Variables

- `PORT` - Server port (default: 8080)
- `DATABASE_PATH` - SQLite database file path (default: picsapp.db)

## Development Workflow

1. **Backend**: `go run main.go` (runs on port 8080)
2. **Frontend Dev**: `npm start` (runs on port 3000, proxies to 8080)
3. **Production Build**: `./build.sh` or `npm run build && go build`

## File Locations

- **Database**: `picsapp.db` (SQLite file)
- **Uploads**: `uploads/` directory (converted WebP files)
- **Originals**: `uploads/original/` directory (temporary storage before conversion)
- **Build Output**: `build/` directory (React production build)

## Documentation Maintenance

This project uses Cursor Rules (`.cursor/rules/`) to automatically keep documentation in sync with code changes. When you modify:

- **API endpoints** → Update `API.md` and `openapi.yaml`
- **Database schema** → Update `DATABASE.md`
- **Data models** → Update `OBJECT_MODEL.md`
- **Project structure** → Update `PROJECT_STRUCTURE.md`

See `.cursor/rules/` directory for detailed documentation update guidelines. The AI assistant will automatically update relevant documentation files when code changes are made.

### Rule Files

- `documentation-maintenance.mdc` - General documentation update rules
- `api-endpoint-updates.mdc` - API endpoint update procedures
- `database-updates.mdc` - Database schema update procedures
- `data-model-updates.mdc` - Data model update procedures
- `component-updates.mdc` - Component update procedures
- `openapi-standards.mdc` - OpenAPI specification standards
- `documentation-checklist.mdc` - Documentation update checklist
- `documentation-priority.mdc` - Update priority guidelines

