package main

import (
	"context"
	"database/sql"
	"log"
	"time"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
)

var db *bun.DB

// InitializeDatabase ensures that the database exists and initializes the schema.
func InitializeDatabase() {
	createDatabaseIfNotExists() // Create course_db if it does not exist.
	ConnectDB()                 // Connect to course_db.
}

// createDatabaseIfNotExists checks if the database exists and creates it if not.
func createDatabaseIfNotExists() {
	// Connect to the PostgreSQL server (not the course_db itself yet).
	dsn := "postgres://postgres:bettergr2425@localhost:5432/postgres?sslmode=disable"
	connector := pgdriver.NewConnector(pgdriver.WithDSN(dsn))
	sqldb := sql.OpenDB(connector)
	defer sqldb.Close()

	// Check if the database exists.
	ctx := context.Background()
	query := `
		SELECT 1 FROM pg_database WHERE datname = 'course_db';
	`
	var exists int
	err := sqldb.QueryRowContext(ctx, query).Scan(&exists)
	if err != nil && err != sql.ErrNoRows {
		log.Fatalf("Failed to check if database exists: %v", err)
	}

	// If the database does not exist, create it.
	if err == sql.ErrNoRows {
		_, err = sqldb.ExecContext(ctx, `CREATE DATABASE course_db;`)
		if err != nil {
			log.Fatalf("Failed to create database: %v", err)
		}
		log.Println("Database course_db created successfully.")
	} else {
		log.Println("Database course_db already exists.")
	}
}

// ConnectDB initializes the PostgreSQL database connection.
func ConnectDB() {
	dsn := "postgres://postgres:bettergr2425@localhost:5432/course_db?sslmode=disable"
	connector := pgdriver.NewConnector(pgdriver.WithDSN(dsn))
	sqldb := sql.OpenDB(connector)
	db = bun.NewDB(sqldb, pgdialect.New())

	// Test the connection.
	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to connect to the database: %v", err)
	}
	log.Println("Connected to PostgreSQL database.")
}

// CloseDB closes the PostgreSQL database connection.
func CloseDB() {
	if err := db.Close(); err != nil {
		log.Fatalf("Failed to close database: %v", err)
	}
}

// createSchemaIfNotExists creates the database schema if it doesn't exist.
func createSchemaIfNotExists(ctx context.Context) error {
	models := []interface{}{
		(*Course)(nil),
	}

	for _, model := range models {
		if _, err := db.NewCreateTable().IfNotExists().Model(model).Exec(ctx); err != nil {
			return err
		}
	}
	log.Println("Database schema initialized.")
	return nil
}

// Course represents the database schema for courses.
type Course struct {
	ID          int32     `bun:",pk,autoincrement"`
	Name        string    `bun:"name,notnull" validate:"required,min=1,max=100"`
	Description string    `bun:"description,notnull" validate:"required,min=1,max=500"`
	Semester    string    `bun:"semester,notnull" validate:"required,min=1,max=20"`
	StaffIDs    []string  `bun:",array"`
	StudentIDs  []string  `bun:",array"`
	CreatedAt   time.Time `bun:",nullzero,notnull,default:current_timestamp"`
	DeletedAt   time.Time `bun:",soft_delete,nullzero"`
}
