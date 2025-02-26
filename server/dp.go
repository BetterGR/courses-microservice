package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"time"

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
		(*CourseStudent)(nil),
		(*CourseStaff)(nil),
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
	CourseID      string    `bun:"course_id,unique,notnull"`
	CourseName    string    `bun:"course_name,notnull"`
	Semester      string    `bun:"semester,notnull"`
	Description   string    `bun:"description"`
	Announcements []string  `bun:"announcements,type:jsonb"`
	CreatedAt     time.Time `bun:"created_at,default:current_timestamp"`
	UpdatedAt     time.Time `bun:"updated_at,default:current_timestamp"`
}

type CourseStudent struct {
	CourseID  string `bun:"course_id"`
	StudentID string `bun:"student_id"`
}

type CourseStaff struct {
	CourseID string `bun:"course_id"`
	StaffID  string `bun:"staff_id"`
}

// AddCourse inserts a new course into the database using the proto message.
func (d *Database) AddCourse(ctx context.Context, course *spb.Course) error {
	_, err := d.db.NewInsert().Model(&Course{
		CourseID:   course.GetCourseId(),
		CourseName: course.GetCourseName(),
		Semester:   course.GetSemester(),
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
		CourseId:      course.CourseID,
		CourseName:    course.CourseName,
		Semester:      course.Semester,
		Description:   course.Description,
		Announcements: course.Announcements,
	}, nil
}

// UpdateCourse updates an existing course in the database using the proto message.
func (d *Database) UpdateCourse(ctx context.Context, course *spb.Course) error {
	_, err := d.db.NewUpdate().Model(&Course{
		CourseID:    course.GetCourseId(),
		CourseName:  course.GetCourseName(),
		Semester:    course.GetSemester(),
		Description: course.GetDescription(),
	}).Where("course_id = ?", course.GetCourseId()).Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to update course: %w", err)
	}

	return nil
}

// DeleteCourse removes a course by course_id.
func (d *Database) DeleteCourse(ctx context.Context, courseID string) error {
	_, err := d.db.NewDelete().Model((*Course)(nil)).Where("course_id = ?", courseID).Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to delete course: %w", err)
	}

	// Delete all students and staff associated with the course.
	_, err = d.db.NewDelete().Model((*CourseStudent)(nil)).Where("course_id = ?", courseID).Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to delete course students: %w", err)
	}

	_, err = d.db.NewDelete().Model((*CourseStaff)(nil)).Where("course_id = ?", courseID).Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to delete course staff: %w", err)
	}

	return nil
}

// AddStudentToCourse adds a student to a course.
func (d *Database) AddStudentToCourse(ctx context.Context, courseID, studentID string) error {
	_, err := d.db.NewInsert().Model(&CourseStudent{
		CourseID:  courseID,
		StudentID: studentID,
	}).Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to add student to course: %w", err)
	}

	return nil
}

// RemoveStudentFromCourse removes a student from a course.
func (d *Database) RemoveStudentFromCourse(ctx context.Context, courseID, studentID string) error {
	_, err := d.db.NewDelete().Model(
		(*CourseStudent)(nil)).Where("course_id = ? AND student_id = ?", courseID, studentID).Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to remove student from course: %w", err)
	}

	return nil
}

// AddStaffToCourse adds a staff member to a course.
func (d *Database) AddStaffToCourse(ctx context.Context, courseID, staffID string) error {
	_, err := d.db.NewInsert().Model(&CourseStaff{
		CourseID: courseID,
		StaffID:  staffID,
	}).Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to add staff to course: %w", err)
	}

	return nil
}

// RemoveStaffFromCourse removes a staff member from a course.
func (d *Database) RemoveStaffFromCourse(ctx context.Context, courseID, staffID string) error {
	_, err := d.db.NewDelete().Model(
		(*CourseStaff)(nil)).Where("course_id = ? AND staff_id = ?", courseID, staffID).Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to remove staff from course: %w", err)
	}

	return nil
}

// GetCourseStudents retrieves all students enrolled in a course.
func (d *Database) GetCourseStudents(ctx context.Context, courseID string) ([]string, error) {
	var studentIDs []string

	// Query the database for student IDs enrolled in the course
	err := d.db.NewSelect().
		Model((*CourseStudent)(nil)). // Use a pointer to the model type
		Column("student_id").
		Where("course_id = ?", courseID).
		Scan(ctx, &studentIDs) // Scan directly into the slice of strings
	if err != nil {
		return nil, fmt.Errorf("failed to get course students: %w", err)
	}

	return studentIDs, nil
}

// GetCourseStaff retrieves all staff members associated with a course.
func (d *Database) GetCourseStaff(ctx context.Context, courseID string) ([]string, error) {
	var staffIDs []string

	err := d.db.NewSelect().
		Model((*CourseStaff)(nil)).
		Column("staff_id").
		Where("course_id = ?", courseID).
		Scan(ctx, &staffIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to get course staff: %w", err)
	}

	return staffIDs, nil
}

// GetStudentCourses retrieves all courses a student is enrolled in.
func (d *Database) GetStudentCourses(ctx context.Context, studentID string) ([]string, error) {
	var courseIDs []string

	err := d.db.NewSelect().
		Model((*CourseStudent)(nil)).
		Column("course_id").
		Where("student_id = ?", studentID).
		Scan(ctx, &courseIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to get student courses: %w", err)
	}

	return courseIDs, nil
}

// GetStaffCourses retrieves all courses a staff member is associated with.
func (d *Database) GetStaffCourses(ctx context.Context, staffID string) ([]string, error) {
	var courseIDs []string

	err := d.db.NewSelect().
		Model((*CourseStaff)(nil)).
		Column("course_id").
		Where("staff_id = ?", staffID).
		Scan(ctx, &courseIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to get staff courses: %w", err)
	}

	return courseIDs, nil
}

// AddAnnouncement adds an announcement to a course.
func (d *Database) AddAnnouncement(ctx context.Context, courseID, announcement string) error {
	course := new(Course)
	if err := d.db.NewSelect().Model(course).Where("course_id = ?", courseID).Scan(ctx); err != nil {
		return fmt.Errorf("failed to get course: %w", err)
	}

	course.Announcements = append(course.Announcements, announcement)

	_, err := d.db.NewUpdate().Model(course).Column("announcements").Where("course_id = ?", courseID).Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to add announcement: %w", err)
	}

	return nil
}

// GetAnnouncements retrieves all announcements for a course.
func (d *Database) GetAnnouncements(ctx context.Context, courseID string) ([]string, error) {
	course := new(Course)
	if err := d.db.NewSelect().Model(course).Where("course_id = ?", courseID).Scan(ctx); err != nil {
		return nil, fmt.Errorf("failed to get course: %w", err)
	}

	return course.Announcements, nil
}

// RemoveAnnouncement removes an announcement from a course.
func (d *Database) RemoveAnnouncement(ctx context.Context, courseID, announcement string) error {
	course := new(Course)
	if err := d.db.NewSelect().Model(course).Where("course_id = ?", courseID).Scan(ctx); err != nil {
		return fmt.Errorf("failed to get course: %w", err)
	}

	// Find the index of the announcement to remove.
	index := -1

	for i, a := range course.Announcements {
		if a == announcement {
			index = i
			break
		}
	}

	// Remove the announcement if found.
	if index != -1 {
		course.Announcements = append(course.Announcements[:index], course.Announcements[index+1:]...)
	}

	_, err := d.db.NewUpdate().Model(course).Column("announcements").Where("course_id = ?", courseID).Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to remove announcement: %w", err)
	}

	return nil
}

// UpdateAnnouncement updates an announcement in a course.
func (d *Database) UpdateAnnouncement(ctx context.Context, courseID, oldAnnouncement, newAnnouncement string) error {
	course := new(Course)
	if err := d.db.NewSelect().Model(course).Where("course_id = ?", courseID).Scan(ctx); err != nil {
		return fmt.Errorf("failed to get course: %w", err)
	}

	// Find the index of the announcement to update.
	index := -1

	for i, a := range course.Announcements {
		if a == oldAnnouncement {
			index = i
			break
		}
	}

	// Update the announcement if found.
	if index != -1 {
		course.Announcements[index] = newAnnouncement
	}

	_, err := d.db.NewUpdate().Model(course).Column("announcements").Where("course_id = ?", courseID).Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to update announcement: %w", err)
	}

	return nil
}
