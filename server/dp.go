package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"

	spb "github.com/BetterGR/courses-microservice/protos"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
	"k8s.io/klog/v2"
)

// Database encapsulates the PostgreSQL connection.
type Database struct {
	db *bun.DB
}

// InitializeDatabase ensures that the database exists and initializes the schema.
func InitializeDatabase() (*Database, error) {
	createDatabaseIfNotExists()

	database, err := ConnectDB()
	if err != nil {
		return nil, err
	}

	if err := database.createSchemaIfNotExists(context.Background()); err != nil {
		klog.Fatalf("Failed to create schema: %v", err)
	}

	return database, nil
}

// createDatabaseIfNotExists ensures the database exists.
func createDatabaseIfNotExists() {
	dsn := os.Getenv("DSN")
	connector := pgdriver.NewConnector(pgdriver.WithDSN(dsn))

	sqldb := sql.OpenDB(connector)
	defer sqldb.Close()

	ctx := context.Background()
	dbName := os.Getenv("DP_NAME")
	query := "SELECT 1 FROM pg_database WHERE datname = $1;"

	var exists int

	err := sqldb.QueryRowContext(ctx, query, dbName).Scan(&exists)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		klog.Fatalf("Failed to check db existence: %v", err)
	}

	if errors.Is(err, sql.ErrNoRows) {
		if _, err = sqldb.ExecContext(ctx, fmt.Sprintf("CREATE DATABASE %s;", dbName)); err != nil {
			klog.Fatalf("Failed to create database: %v", err)
		}

		klog.Infof("Database %s created successfully.", dbName)
	} else {
		klog.Infof("Database %s already exists.", dbName)
	}
}

// ConnectDB connects to the database.
func ConnectDB() (*Database, error) {
	dsn := os.Getenv("DSN")
	connector := pgdriver.NewConnector(pgdriver.WithDSN(dsn))
	sqldb := sql.OpenDB(connector)
	database := bun.NewDB(sqldb, pgdialect.New())

	// Test the connection.
	if err := database.Ping(); err != nil {
		return nil, fmt.Errorf("failed to connect to the database: %w", err)
	}

	klog.Info("Connected to PostgreSQL database.")

	return &Database{db: database}, nil
}

// createSchemaIfNotExists creates the database schema if it doesn't exist.
func (d *Database) createSchemaIfNotExists(ctx context.Context) error {
	models := []interface{}{
		(*Course)(nil),
	}

	for _, model := range models {
		if _, err := d.db.NewCreateTable().IfNotExists().Model(model).Exec(ctx); err != nil {
			return fmt.Errorf("failed to create table: %w", err)
		}
	}

	klog.Info("Database schema initialized.")

	return nil
}

// Course represents the database schema for courses.
type Course struct {
	UniqueID       string   `bun:",pk,default:gen_random_uuid()"`
	CourseID       string   `bun:"course_id,unique,notnull"`
	CourseName     string   `bun:"course_name,notnull"`
	Semester       string   `bun:"semester,notnull"`
	CourseMaterial string   `bun:"course_material,notnull"` // JSON string storing name and semester.
	Description    string   `bun:"description"`
	StaffIDs       []string `bun:"staff_ids,notnull"`
	StudentsIDs    []string `bun:"students_ids,notnull"`
}

// AddCourse inserts a new course into the database using the proto message.
func (d *Database) AddCourse(ctx context.Context, course *spb.Course) error {
	_, err := d.db.NewInsert().Model(&Course{
		CourseID:       course.GetId(),
		CourseName:     course.GetName(),
		Semester:       course.GetSemester(),
		CourseMaterial: course.GetCourseMaterial(),
		Description:    course.GetDescription(),
		StaffIDs:       course.GetStaffIds(),
		StudentsIDs:    course.GetStudentsIds(),
	}).Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to add course: %w", err)
	}

	return nil
}

// GetCourse retrieves a course by its course_id and returns the proto message.
func (d *Database) GetCourse(ctx context.Context, id string) (*spb.Course, error) {
	course := new(Course)
	if err := d.db.NewSelect().Model(course).Where("course_id = ?", id).Scan(ctx); err != nil {
		return nil, fmt.Errorf("failed to get course: %w", err)
	}

	return &spb.Course{
		Id:             course.CourseID,
		Name:           course.CourseName,
		Semester:       course.Semester,
		CourseMaterial: course.CourseMaterial,
		Description:    course.Description,
		StaffIds:       course.StaffIDs,
		StudentsIds:    course.StudentsIDs,
	}, nil
}

// UpdateCourse updates an existing course in the database using the proto message.
func (d *Database) UpdateCourse(ctx context.Context, course *spb.Course) error {
	_, err := d.db.NewUpdate().Model(&Course{
		CourseID:       course.GetId(),
		CourseName:     course.GetName(),
		Semester:       course.GetSemester(),
		CourseMaterial: course.GetCourseMaterial(),
		Description:    course.GetDescription(),
		StaffIDs:       course.GetStaffIds(),
		StudentsIDs:    course.GetStudentsIds(),
	}).Where("course_id = ?", course.GetId()).Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to update course: %w", err)
	}

	return nil
}

// DeleteCourse removes a course by course_id.
func (d *Database) DeleteCourse(ctx context.Context, id string) error {
	_, err := d.db.NewDelete().Model((*Course)(nil)).Where("course_id = ?", id).Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to delete course: %w", err)
	}

	return nil
}

// AddStudentToCourse adds a student to a course.
func (d *Database) AddStudentToCourse(ctx context.Context, courseID, studentID string) error {
	course, err := d.GetCourse(ctx, courseID)
	if err != nil {
		return err
	}
	// Check if studentID already exists.
	for _, v := range course.GetStudentsIds() {
		if v == studentID {
			return nil
		}
	}

	course.StudentsIds = append(course.StudentsIds, studentID)

	return d.UpdateCourse(ctx, course)
}

// RemoveStudentFromCourse removes a student from a course.
func (d *Database) RemoveStudentFromCourse(ctx context.Context, courseID, studentID string) error {
	course, err := d.GetCourse(ctx, courseID)
	if err != nil {
		return err
	}

	newArr := []string{}

	for _, v := range course.GetStudentsIds() {
		if v != studentID {
			newArr = append(newArr, v)
		}
	}

	course.StudentsIds = newArr

	return d.UpdateCourse(ctx, course)
}

// AddStaffToCourse adds a staff member to a course.
func (d *Database) AddStaffToCourse(ctx context.Context, courseID, staffID string) error {
	course, err := d.GetCourse(ctx, courseID)
	if err != nil {
		return err
	}

	for _, v := range course.GetStaffIds() {
		if v == staffID {
			return nil
		}
	}

	course.StaffIds = append(course.StaffIds, staffID)

	return d.UpdateCourse(ctx, course)
}

// RemoveStaffFromCourse removes a staff member from a course.
func (d *Database) RemoveStaffFromCourse(ctx context.Context, courseID, staffID string) error {
	course, err := d.GetCourse(ctx, courseID)
	if err != nil {
		return err
	}

	newArr := []string{}

	for _, v := range course.GetStaffIds() {
		if v != staffID {
			newArr = append(newArr, v)
		}
	}

	course.StaffIds = newArr

	return d.UpdateCourse(ctx, course)
}

// UpdateCourseMaterial updates the course material for a given course.
func (d *Database) UpdateCourseMaterial(ctx context.Context, courseID, material string) error {
	course, err := d.GetCourse(ctx, courseID)
	if err != nil {
		return err
	}

	course.CourseMaterial = material

	return d.UpdateCourse(ctx, course)
}

// GetCourseMaterial retrieves the course material for a given course.
func (d *Database) GetCourseMaterial(ctx context.Context, courseID string) (string, error) {
	course, err := d.GetCourse(ctx, courseID)
	if err != nil {
		return "", err
	}

	return course.GetCourseMaterial(), nil
}
