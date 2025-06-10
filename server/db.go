package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"time"

	cpb "github.com/BetterGR/courses-microservice/protos"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
	"k8s.io/klog/v2"
)

// CourseDBInterface defines the core CRUD operations for courses.
type CourseDBInterface interface {
	AddCourse(ctx context.Context, course *cpb.Course) (*Course, error)
	GetCourse(ctx context.Context, courseID string) (*Course, error)
	UpdateCourse(ctx context.Context, course *cpb.Course) (*Course, error)
	DeleteCourse(ctx context.Context, courseID string) error
	GetCoursesBySemester(ctx context.Context, semester string) ([]*Course, error)
}

// StudentDBInterface defines operations related to student enrollments.
type StudentDBInterface interface {
	AddStudentToCourse(ctx context.Context, courseID, studentID string) error
	RemoveStudentFromCourse(ctx context.Context, courseID, studentID string) error
	GetCourseStudents(ctx context.Context, courseID string) ([]string, error)
	GetStudentCourses(ctx context.Context, studentID string) ([]string, error)
}

// StaffDBInterface defines operations related to staff assignments.
type StaffDBInterface interface {
	AddStaffToCourse(ctx context.Context, courseID, staffID string) error
	RemoveStaffFromCourse(ctx context.Context, courseID, staffID string) error
	GetCourseStaff(ctx context.Context, courseID string) ([]string, error)
	GetStaffCourses(ctx context.Context, staffID string) ([]string, error)
}

// AnnouncementDBInterface defines operations related to course announcements.
type AnnouncementDBInterface interface {
	AddAnnouncement(ctx context.Context, req *cpb.AddAnnouncementRequest) error
	GetAnnouncements(ctx context.Context, courseID string) ([]Announcement, error)
	RemoveAnnouncement(ctx context.Context, courseID, announcementID string) error
}

// DBInterface combines all database operation interfaces.
type DBInterface interface {
	CourseDBInterface
	StudentDBInterface
	StaffDBInterface
	AnnouncementDBInterface
}

// Database encapsulates the PostgreSQL connection.
type Database struct {
	db *bun.DB
}

// Verify that Database implements DBInterface at compile time.
var _ DBInterface = (*Database)(nil)

var (
	ErrCourseNil         = errors.New("course is nil")
	ErrCourseIDEmpty     = errors.New("course ID is empty")
	ErrCourseNotFound    = errors.New("course not found")
	ErrStudentIDEmpty    = errors.New("student ID is empty")
	ErrStaffIDEmpty      = errors.New("staff ID is empty")
	ErrAnnouncementEmpty = errors.New("announcement is empty")
	ErrSemesterEmpty     = errors.New("semester is empty")
)

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

		klog.V(logLevelDebug).Infof("Database %s created successfully.", dbName)
	} else {
		klog.V(logLevelDebug).Infof("Database %s already exists.", dbName)
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

	klog.V(logLevelDebug).Info("Connected to PostgreSQL database.")

	return &Database{db: database}, nil
}

// createSchemaIfNotExists creates the database schema if it doesn't exist.
func (d *Database) createSchemaIfNotExists(ctx context.Context) error {
	models := []interface{}{
		(*Course)(nil),
		(*CourseStudent)(nil),
		(*CourseStaff)(nil),
		(*Announcement)(nil),
	}

	for _, model := range models {
		if _, err := d.db.NewCreateTable().IfNotExists().Model(model).Exec(ctx); err != nil {
			return fmt.Errorf("failed to create table: %w", err)
		}
	}

	klog.V(logLevelDebug).Info("Database schema initialized.")

	return nil
}

// Course represents the database schema for courses.
type Course struct {
	CourseID    string    `bun:"course_id,unique,pk,notnull"`
	CourseName  string    `bun:"course_name,notnull"`
	Semester    string    `bun:"semester,notnull"`
	Description string    `bun:"description"`
	CreatedAt   time.Time `bun:"created_at,default:current_timestamp"`
	UpdatedAt   time.Time `bun:"updated_at,default:current_timestamp"`
}

type Announcement struct {
	AnnouncementID string    `bun:"announcement_id,notnull"`
	CourseID       string    `bun:"course_id,notnull"`
	Title          string    `bun:"title,notnull"`
	Content        string    `bun:"content,notnull"`
	CreatedAt      time.Time `bun:"created_at,default:current_timestamp"`
	UpdatedAt      time.Time `bun:"updated_at,default:current_timestamp"`
}

type CourseStudent struct {
	CourseID  string `bun:"course_id,notnull"`
	StudentID string `bun:"student_id,notnull"`
}

type CourseStaff struct {
	CourseID string `bun:"course_id,notnull"`
	StaffID  string `bun:"staff_id,notnull"`
}

// AddCourse inserts a new course into the database using the proto message.
func (d *Database) AddCourse(ctx context.Context, course *cpb.Course) (*Course, error) {
	if course == nil {
		return nil, fmt.Errorf("%w", ErrCourseNil)
	}

	if course.GetCourseID() == "" {
		return nil, fmt.Errorf("%w", ErrCourseIDEmpty)
	}

	newCourse := &Course{
		CourseID:    course.GetCourseID(),
		CourseName:  course.GetCourseName(),
		Semester:    course.GetSemester(),
		Description: course.GetDescription(),
	}

	_, err := d.db.NewInsert().Model(newCourse).Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to add course: %w", err)
	}

	return newCourse, nil
}

// GetCourse retrieves a course by its course_id and returns the proto message.
func (d *Database) GetCourse(ctx context.Context, courseID string) (*Course, error) {
	if courseID == "" {
		return nil, fmt.Errorf("%w", ErrCourseIDEmpty)
	}

	course := new(Course)
	if err := d.db.NewSelect().Model(course).Where("course_id = ?", courseID).Scan(ctx); err != nil {
		return nil, fmt.Errorf("failed to get course: %w", err)
	}

	return course, nil
}

// UpdateCourse updates an existing course in the database using the proto message.
func (d *Database) UpdateCourse(ctx context.Context, course *cpb.Course) (*Course, error) {
	if course == nil {
		return nil, fmt.Errorf("%w", ErrCourseNil)
	}

	if course.GetCourseID() == "" {
		return nil, fmt.Errorf("%w", ErrCourseIDEmpty)
	}

	// get existing course.
	existingCourse, err := d.GetCourse(ctx, course.GetCourseID())
	if err != nil {
		return nil, fmt.Errorf("failed to get course: %w", err)
	}

	// Update the fields.
	updateField := func(field *string, newValue string) {
		if newValue != "" {
			*field = newValue
		}
	}

	updateField(&existingCourse.CourseName, course.GetCourseName())
	updateField(&existingCourse.Semester, course.GetSemester())
	updateField(&existingCourse.Description, course.GetDescription())

	_, err = d.db.NewUpdate().Model(existingCourse).WherePK().Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to update course: %w", err)
	}

	return existingCourse, nil
}

// DeleteCourse removes a course by course_id.
func (d *Database) DeleteCourse(ctx context.Context, courseID string) error {
	if courseID == "" {
		return fmt.Errorf("%w", ErrCourseIDEmpty)
	}

	res, err := d.db.NewDelete().Model((*Course)(nil)).Where("course_id = ?", courseID).Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to delete course: %w", err)
	}

	if num, _ := res.RowsAffected(); num == 0 {
		return fmt.Errorf("%w", ErrCourseNotFound)
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
	if courseID == "" {
		return fmt.Errorf("%w", ErrCourseIDEmpty)
	}

	if studentID == "" {
		return fmt.Errorf("%w", ErrStudentIDEmpty)
	}

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
	if courseID == "" {
		return fmt.Errorf("%w", ErrCourseIDEmpty)
	}

	if studentID == "" {
		return fmt.Errorf("%w", ErrStudentIDEmpty)
	}

	res, err := d.db.NewDelete().Model(
		(*CourseStudent)(nil)).Where("course_id = ? AND student_id = ?", courseID, studentID).Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to remove student from course: %w", err)
	}

	if num, _ := res.RowsAffected(); num == 0 {
		return fmt.Errorf("%w", ErrCourseNotFound)
	}

	return nil
}

// AddStaffToCourse adds a staff member to a course.
func (d *Database) AddStaffToCourse(ctx context.Context, courseID, staffID string) error {
	if courseID == "" {
		return fmt.Errorf("%w", ErrCourseIDEmpty)
	}

	if staffID == "" {
		return fmt.Errorf("%w", ErrStaffIDEmpty)
	}

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
	if courseID == "" {
		return fmt.Errorf("%w", ErrCourseIDEmpty)
	}

	if staffID == "" {
		return fmt.Errorf("%w", ErrStaffIDEmpty)
	}

	res, err := d.db.NewDelete().Model(
		(*CourseStaff)(nil)).Where("course_id = ? AND staff_id = ?", courseID, staffID).Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to remove staff from course: %w", err)
	}

	if num, _ := res.RowsAffected(); num == 0 {
		return fmt.Errorf("%w", ErrCourseNotFound)
	}

	return nil
}

// GetCourseStudents retrieves all students enrolled in a course.
func (d *Database) GetCourseStudents(ctx context.Context, courseID string) ([]string, error) {
	if courseID == "" {
		return nil, fmt.Errorf("%w", ErrCourseIDEmpty)
	}

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
	if courseID == "" {
		return nil, fmt.Errorf("%w", ErrCourseIDEmpty)
	}

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
	if studentID == "" {
		return nil, fmt.Errorf("%w", ErrStudentIDEmpty)
	}

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
	if staffID == "" {
		return nil, fmt.Errorf("%w", ErrStaffIDEmpty)
	}

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

// GetCoursesBySemester retrieves all courses for a specific semester.
func (d *Database) GetCoursesBySemester(ctx context.Context, semester string) ([]*Course, error) {
	if semester == "" {
		return nil, fmt.Errorf("%w", ErrSemesterEmpty)
	}

	var courses []*Course

	err := d.db.NewSelect().
		Model(&courses).
		Where("semester = ?", semester).
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get courses by semester: %w", err)
	}

	return courses, nil
}

// AddAnnouncement adds an announcement to a course.
func (d *Database) AddAnnouncement(ctx context.Context, req *cpb.AddAnnouncementRequest) error {
	if (req.GetCourseID() == "") || (req.GetAnnouncement().GetAnnouncementContent() == "") {
		return fmt.Errorf("%w", ErrCourseIDEmpty)
	}

	_, err := d.db.NewInsert().Model(&Announcement{
		CourseID:       req.GetCourseID(),
		AnnouncementID: req.GetAnnouncement().GetAnnouncementID(),
		Title:          req.GetAnnouncement().GetAnnouncementTitle(),
		Content:        req.GetAnnouncement().GetAnnouncementContent(),
	}).Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to add announcement: %w", err)
	}

	return nil
}

// GetAnnouncements retrieves all announcements for a course.
func (d *Database) GetAnnouncements(ctx context.Context, courseID string) ([]Announcement, error) {
	if courseID == "" {
		return nil, fmt.Errorf("%w", ErrCourseIDEmpty)
	}

	var announcements []Announcement

	err := d.db.NewSelect().
		Model((*Announcement)(nil)).
		Where("course_id = ?", courseID).
		Scan(ctx, &announcements)
	if err != nil {
		return nil, fmt.Errorf("failed to get announcements: %w", err)
	}

	return announcements, nil
}

// RemoveAnnouncement removes an announcement from a course.
func (d *Database) RemoveAnnouncement(ctx context.Context, courseID, announcementID string) error {
	if courseID == "" {
		return fmt.Errorf("%w", ErrCourseIDEmpty)
	}

	if announcementID == "" {
		return fmt.Errorf("%w", ErrAnnouncementEmpty)
	}

	res, err := d.db.NewDelete().
		Model((*Announcement)(nil)).
		Where("course_id = ? AND announcement_id = ?", courseID, announcementID).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to remove announcement: %w", err)
	}

	if num, _ := res.RowsAffected(); num == 0 {
		return fmt.Errorf("%w", ErrCourseNotFound)
	}

	return nil
}
