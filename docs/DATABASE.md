# Database Structure

This document describes the SQLite database schema, tables, indexes, and data relationships in PicsApp.

## Database File

- **Location**: `picsapp.db` (configurable via `DATABASE_PATH` environment variable)
- **Type**: SQLite 3
- **Driver**: `github.com/mattn/go-sqlite3`

## Schema Overview

The database consists of two main tables:
1. **pictures** - Stores picture metadata
2. **conversion_tasks** - Manages image conversion queue

## Tables

### `pictures` Table

Stores metadata for all uploaded pictures.

#### Schema

```sql
CREATE TABLE pictures (
    id TEXT PRIMARY KEY,
    filename TEXT NOT NULL,
    url TEXT NOT NULL,
    likes INTEGER DEFAULT 0,
    uploaded_at DATETIME NOT NULL
);
```

#### Columns

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| `id` | TEXT | PRIMARY KEY | Unique identifier (filename with .webp extension) |
| `filename` | TEXT | NOT NULL | Original filename from upload |
| `url` | TEXT | NOT NULL | URL path to serve the image (e.g., `/uploads/123.webp`) |
| `likes` | INTEGER | DEFAULT 0 | Number of likes received |
| `uploaded_at` | DATETIME | NOT NULL | ISO 8601 timestamp of upload |

#### Indexes

```sql
CREATE INDEX idx_uploaded_at ON pictures(uploaded_at);
CREATE INDEX idx_likes ON pictures(likes);
```

- **idx_uploaded_at**: Optimizes queries for recent pictures
- **idx_likes**: Optimizes queries sorted by likes

#### Example Data

```json
{
  "id": "1762801393825964000.webp",
  "filename": "download.jpeg",
  "url": "/uploads/1762801393825964000.webp",
  "likes": 5,
  "uploaded_at": "2024-01-15T10:30:00Z"
}
```

### `conversion_tasks` Table

Manages the queue of images waiting to be converted to WebP format.

#### Schema

```sql
CREATE TABLE conversion_tasks (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    original_path TEXT NOT NULL UNIQUE,
    original_name TEXT,
    picture_id TEXT,
    status TEXT NOT NULL DEFAULT 'pending',
    error TEXT,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
```

#### Columns

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| `id` | INTEGER | PRIMARY KEY AUTOINCREMENT | Auto-incrementing task ID |
| `original_path` | TEXT | NOT NULL UNIQUE | Full filesystem path to original image |
| `original_name` | TEXT | NULL | Original filename (for display) |
| `picture_id` | TEXT | NULL | Existing picture ID (for re-conversion) |
| `status` | TEXT | NOT NULL DEFAULT 'pending' | Task status: `pending`, `processing`, `completed`, `failed` |
| `error` | TEXT | NULL | Error message if status is `failed` |
| `created_at` | DATETIME | NOT NULL DEFAULT CURRENT_TIMESTAMP | Task creation timestamp |
| `updated_at` | DATETIME | NOT NULL DEFAULT CURRENT_TIMESTAMP | Last update timestamp |

#### Indexes

```sql
CREATE INDEX idx_conversion_status ON conversion_tasks(status);
```

- **idx_conversion_status**: Optimizes queries for pending tasks

#### Status Values

- **pending**: Task is queued, waiting to be processed
- **processing**: Task is currently being processed by worker
- **completed**: Task completed successfully
- **failed**: Task failed with an error (error message stored in `error` column)

#### Example Data

```json
{
  "id": 1,
  "original_path": "uploads/original/1762801393825964000.jpeg",
  "original_name": "download.jpeg",
  "picture_id": null,
  "status": "completed",
  "error": null,
  "created_at": "2024-01-15T10:30:00Z",
  "updated_at": "2024-01-15T10:30:05Z"
}
```

## Data Relationships

### Picture Lifecycle

1. **Upload**: File saved to `uploads/original/`, task created in `conversion_tasks`
2. **Conversion**: Worker processes task, converts to WebP
3. **Storage**: Converted file saved to `uploads/`, record created in `pictures`
4. **Cleanup**: Original file deleted, task marked as `completed`

### Re-conversion Flow

When a picture needs to be re-converted (e.g., legacy non-WebP files):
1. Task created with existing `picture_id`
2. Worker processes task
3. Picture record updated with new file ID and URL
4. Old file deleted

## Database Operations

### Picture Operations

#### Add Picture
```go
db.AddPicture(picture *Picture) error
```
- Inserts new picture record
- Uses RFC3339 timestamp format

#### Get Picture
```go
db.GetPicture(id string) (*Picture, error)
```
- Retrieves single picture by ID
- Returns error if not found

#### Get Last Pictures
```go
db.GetLastPictures(n int) ([]*Picture, error)
```
- Returns last N pictures ordered by `uploaded_at DESC`
- Used for home page grid (typically 30 pictures)

#### Get All Pictures Sorted by Likes
```go
db.GetAllPicturesSortedByLikes() ([]*Picture, error)
```
- Returns all pictures ordered by `likes DESC, uploaded_at DESC`
- Used for presentation page

#### Increment Likes
```go
db.IncrementLikes(id string) error
```
- Atomically increments like count
- Returns error if picture not found

#### Update Picture File
```go
db.UpdatePictureFile(oldID, newID, newURL string) error
```
- Updates picture ID and URL (for re-conversion)
- Used when converting existing pictures

### Conversion Task Operations

#### Create Conversion Task
```go
db.CreateConversionTask(path, name, pictureID string) error
```
- Creates new task with status `pending`
- Uses `INSERT OR IGNORE` to prevent duplicates
- `pictureID` can be empty string (converted to NULL)

#### Claim Next Task
```go
db.ClaimNextTask() (*ConversionTask, error)
```
- **Atomic operation** using transaction
- Selects oldest `pending` task
- Updates status to `processing` in same transaction
- Returns `nil, nil` if no tasks available
- Prevents race conditions with multiple workers

#### Mark Task Completed
```go
db.MarkTaskCompleted(id int64) error
```
- Updates status to `completed`
- Clears error message
- Updates `updated_at` timestamp

#### Mark Task Failed
```go
db.MarkTaskFailed(id int64, msg string) error
```
- Updates status to `failed`
- Stores error message
- Updates `updated_at` timestamp

## Migration and Schema Evolution

The database uses a simple migration approach:

1. **Initial Schema**: Tables created with `CREATE TABLE IF NOT EXISTS`
2. **Column Addition**: `ALTER TABLE` with error handling for existing columns
3. **Index Creation**: `CREATE INDEX IF NOT EXISTS`

Example from `database.go`:
```go
// Ensure picture_id column exists for legacy DBs
if _, err := d.db.Exec(`ALTER TABLE conversion_tasks ADD COLUMN picture_id TEXT`); err != nil {
    if !strings.Contains(err.Error(), "duplicate column name") {
        log.Printf("warning: unable to add picture_id column: %v", err)
    }
}
```

## Data Types and Formats

### Timestamps
- **Storage Format**: ISO 8601 / RFC3339 (e.g., `2024-01-15T10:30:00Z`)
- **Go Parsing**: `time.Parse(time.RFC3339, uploadedAtStr)`
- **Go Storage**: `time.Now().Format(time.RFC3339)`

### Picture IDs
- **Format**: `{timestamp_nanoseconds}.webp`
- **Example**: `1762801393825964000.webp`
- **Uniqueness**: Guaranteed by nanosecond timestamp

### URLs
- **Format**: `/uploads/{id}`
- **Example**: `/uploads/1762801393825964000.webp`
- **Serving**: Handled by Go file server

## Query Patterns

### Recent Pictures (Home Page)
```sql
SELECT id, filename, url, likes, uploaded_at 
FROM pictures 
ORDER BY uploaded_at DESC 
LIMIT 30;
```

### Top Pictures (Presentation)
```sql
SELECT id, filename, url, likes, uploaded_at 
FROM pictures 
ORDER BY likes DESC, uploaded_at DESC;
```

### Pending Conversion Tasks
```sql
SELECT id, original_path, original_name, picture_id, status, error, created_at, updated_at 
FROM conversion_tasks 
WHERE status = 'pending' 
ORDER BY created_at 
LIMIT 1;
```

### Task Status Update (Atomic)
```sql
BEGIN TRANSACTION;
SELECT ... FROM conversion_tasks WHERE status = 'pending' ...;
UPDATE conversion_tasks SET status = 'processing', updated_at = CURRENT_TIMESTAMP WHERE id = ? AND status = 'pending';
COMMIT;
```

## Performance Considerations

1. **Indexes**: Both tables have indexes on frequently queried columns
2. **Atomic Operations**: Task claiming uses transactions to prevent race conditions
3. **Connection Pooling**: SQLite handles connections efficiently for single-server use
4. **Query Optimization**: LIMIT clauses prevent loading all records
5. **Batch Operations**: WebSocket broadcasts use single query for all pictures

## Backup and Maintenance

- **Backup**: Copy `picsapp.db` file (SQLite is a single file)
- **Vacuum**: Run `VACUUM` to reclaim space after deletions
- **Integrity Check**: Run `PRAGMA integrity_check;` to verify database
- **Statistics**: Run `ANALYZE;` to update query optimizer statistics

