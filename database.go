package main

import (
	"database/sql"
	"fmt"
	"log"
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
	`

	_, err := d.db.Exec(query)
	return err
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

