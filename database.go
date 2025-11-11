package main

import (
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type Database struct {
	db *sql.DB
}

func NewDatabase(dbPath string) (*Database, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	database := &Database{db: db}

	if err := database.initSchema(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	return database, nil
}

func (d *Database) initSchema() error {
	query := `
	CREATE TABLE IF NOT EXISTS pictures (
		id TEXT PRIMARY KEY,
		filename TEXT NOT NULL,
		url TEXT NOT NULL,
		likes INTEGER DEFAULT 0,
		uploaded_at DATETIME NOT NULL
	);
	
	CREATE INDEX IF NOT EXISTS idx_uploaded_at ON pictures(uploaded_at);
	CREATE INDEX IF NOT EXISTS idx_likes ON pictures(likes);

	CREATE TABLE IF NOT EXISTS conversion_tasks (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		original_path TEXT NOT NULL UNIQUE,
		original_name TEXT,
		picture_id TEXT,
		status TEXT NOT NULL DEFAULT 'pending',
		error TEXT,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_conversion_status ON conversion_tasks(status);
	`

	if _, err := d.db.Exec(query); err != nil {
		return err
	}

	// Ensure picture_id column exists for legacy DBs
	if _, err := d.db.Exec(`ALTER TABLE conversion_tasks ADD COLUMN picture_id TEXT`); err != nil {
		if !strings.Contains(err.Error(), "duplicate column name") {
			log.Printf("warning: unable to add picture_id column: %v", err)
		}
	}

	return nil
}

func (d *Database) Close() error {
	return d.db.Close()
}

func (d *Database) AddPicture(picture *Picture) error {
	query := `INSERT INTO pictures (id, filename, url, likes, uploaded_at) VALUES (?, ?, ?, ?, ?)`
	_, err := d.db.Exec(query, picture.ID, picture.Filename, picture.URL, picture.Likes, picture.UploadedAt.Format(time.RFC3339))
	return err
}

func (d *Database) GetPicture(id string) (*Picture, error) {
	query := `SELECT id, filename, url, likes, uploaded_at FROM pictures WHERE id = ?`
	row := d.db.QueryRow(query, id)

	var picture Picture
	var uploadedAtStr string
	err := row.Scan(&picture.ID, &picture.Filename, &picture.URL, &picture.Likes, &uploadedAtStr)
	if err != nil {
		return nil, err
	}

	picture.UploadedAt, err = time.Parse(time.RFC3339, uploadedAtStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse time: %w", err)
	}

	return &picture, nil
}

func (d *Database) GetLastPictures(n int) ([]*Picture, error) {
	query := `SELECT id, filename, url, likes, uploaded_at FROM pictures ORDER BY uploaded_at DESC LIMIT ?`
	rows, err := d.db.Query(query, n)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var pictures []*Picture
	for rows.Next() {
		var picture Picture
		var uploadedAtStr string
		if err := rows.Scan(&picture.ID, &picture.Filename, &picture.URL, &picture.Likes, &uploadedAtStr); err != nil {
			return nil, err
		}

		picture.UploadedAt, err = time.Parse(time.RFC3339, uploadedAtStr)
		if err != nil {
			log.Printf("Warning: failed to parse time for picture %s: %v", picture.ID, err)
			continue
		}

		pictures = append(pictures, &picture)
	}

	return pictures, rows.Err()
}

func (d *Database) GetAllPicturesSortedByLikes() ([]*Picture, error) {
	query := `SELECT id, filename, url, likes, uploaded_at FROM pictures ORDER BY likes DESC, uploaded_at DESC`
	rows, err := d.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var pictures []*Picture
	for rows.Next() {
		var picture Picture
		var uploadedAtStr string
		if err := rows.Scan(&picture.ID, &picture.Filename, &picture.URL, &picture.Likes, &uploadedAtStr); err != nil {
			return nil, err
		}

		picture.UploadedAt, err = time.Parse(time.RFC3339, uploadedAtStr)
		if err != nil {
			log.Printf("Warning: failed to parse time for picture %s: %v", picture.ID, err)
			continue
		}

		pictures = append(pictures, &picture)
	}

	return pictures, rows.Err()
}

func (d *Database) IncrementLikes(id string) error {
	query := `UPDATE pictures SET likes = likes + 1 WHERE id = ?`
	result, err := d.db.Exec(query, id)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return fmt.Errorf("picture not found")
	}

	return nil
}

func (d *Database) LoadAllPictures() ([]*Picture, error) {
	return d.GetAllPicturesSortedByLikes()
}

func (d *Database) UpdatePictureFile(oldID, newID, newURL string) error {
	query := `UPDATE pictures SET id = ?, url = ? WHERE id = ?`
	_, err := d.db.Exec(query, newID, newURL, oldID)
	return err
}

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

func (d *Database) CreateConversionTask(path, name, pictureID string) error {
	query := `INSERT OR IGNORE INTO conversion_tasks (original_path, original_name, picture_id) VALUES (?, ?, NULLIF(?, ''))`
	_, err := d.db.Exec(query, path, name, pictureID)
	return err
}

func (d *Database) ClaimNextTask() (*ConversionTask, error) {
	tx, err := d.db.Begin()
	if err != nil {
		return nil, err
	}

	row := tx.QueryRow(`SELECT id, original_path, original_name, picture_id, status, error, created_at, updated_at FROM conversion_tasks WHERE status = 'pending' ORDER BY created_at LIMIT 1`)
	var task ConversionTask
	var errStr sql.NullString
	var pictureID sql.NullString
	if err := row.Scan(&task.ID, &task.OriginalPath, &task.OriginalName, &pictureID, &task.Status, &errStr, &task.CreatedAt, &task.UpdatedAt); err != nil {
		if err == sql.ErrNoRows {
			tx.Rollback()
			return nil, nil
		}
		tx.Rollback()
		return nil, err
	}
	if pictureID.Valid {
		task.PictureID = &pictureID.String
	}
	if errStr.Valid {
		task.Error = &errStr.String
	}

	res, err := tx.Exec(`UPDATE conversion_tasks SET status = 'processing', updated_at = CURRENT_TIMESTAMP WHERE id = ? AND status = 'pending'`, task.ID)
	if err != nil {
		tx.Rollback()
		return nil, err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		tx.Rollback()
		return nil, err
	}
	if rows == 0 {
		tx.Rollback()
		return nil, nil
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return &task, nil
}

func (d *Database) MarkTaskCompleted(id int64) error {
	_, err := d.db.Exec(`UPDATE conversion_tasks SET status = 'completed', error = NULL, updated_at = CURRENT_TIMESTAMP WHERE id = ?`, id)
	return err
}

func (d *Database) MarkTaskFailed(id int64, msg string) error {
	_, err := d.db.Exec(`UPDATE conversion_tasks SET status = 'failed', error = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`, msg, id)
	return err
}
