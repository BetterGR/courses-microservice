package main

import (
	"os"
	"testing"

	cpb "github.com/BetterGR/courses-microservice/protos"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupTestDatabase creates a database connection for testing.
func setupTestDatabase(t *testing.T) *Database {
	t.Helper()

	// Use environment variables or set default test values.
	if os.Getenv("DSN") == "" {
		t.Setenv("DSN", "postgres://postgres:postgres@localhost:5432/test_courses?sslmode=disable")
	}

	if os.Getenv("DP_NAME") == "" {
		t.Setenv("DP_NAME", "test_courses")
	}

	// Initialize the database.
	database, err := InitializeDatabase()
	require.NoError(t, err, "Failed to initialize database")
	require.NotNil(t, database, "Database should not be nil")

	return database
}

// cleanupTestDatabase cleans up test data.
func cleanupTestDatabase(t *testing.T, database *Database) {
	t.Helper()

	// Use a fixed courseID since all tests use the same one
	courseID := "TEST101"
	err := database.DeleteCourse(t.Context(), courseID)

	if err != nil && err.Error() != ErrCourseNotFound.Error() {
		t.Logf("Error cleaning up course %s: %v", courseID, err)
	}
}

// buildTestCourse creates a test course.
func buildTestCourse() *cpb.Course {
	return &cpb.Course{
		CourseID:    "TEST101",
		CourseName:  "Test Course",
		Semester:    "Test Semester",
		Description: "Test Description",
	}
}

// checkSkipTest checks if test should be skipped.
func checkSkipTest(t *testing.T) {
	t.Helper()

	// Skip if not in test environment to prevent accidental data modification.
	if os.Getenv("TEST_ENV") != "true" && os.Getenv("CI") != "true" {
		t.Skip("Skipping database test in non-test environment. Set TEST_ENV=true to run.")
	}
}

// TestDatabaseOperations tests the basic database operations.
func TestDatabaseOperations(t *testing.T) {
	checkSkipTest(t)

	t.Run("TestCourseOperations", testCourseOperations)
	t.Run("TestStudentEnrollment", testStudentEnrollment)
	t.Run("TestStaffAssignments", testStaffAssignments)
	t.Run("TestAnnouncements", testAnnouncements)
}

// testCourseOperations tests basic CRUD operations for courses.
func testCourseOperations(t *testing.T) {
	database := setupTestDatabase(t)
	defer cleanupTestDatabase(t, database)

	testCourse := buildTestCourse()

	// Test AddCourse.
	t.Run("AddCourse", func(t *testing.T) {
		course, err := database.AddCourse(t.Context(), testCourse)
		require.NoError(t, err, "Should add course without error")
		assert.Equal(t, testCourse.GetCourseID(), course.CourseID, "Course ID should match")
		assert.Equal(t, testCourse.GetCourseName(), course.CourseName, "Course name should match")
	})

	// Test GetCourse.
	t.Run("GetCourse", func(t *testing.T) {
		course, err := database.GetCourse(t.Context(), testCourse.GetCourseID())
		require.NoError(t, err, "Should get course without error")
		assert.Equal(t, testCourse.GetCourseID(), course.CourseID, "Course ID should match")
		assert.Equal(t, testCourse.GetCourseName(), course.CourseName, "Course name should match")
	})

	// Test UpdateCourse.
	t.Run("UpdateCourse", func(t *testing.T) {
		updatedCourse := &cpb.Course{
			CourseID:    testCourse.GetCourseID(),
			CourseName:  "Updated Test Course",
			Semester:    testCourse.GetSemester(),
			Description: "Updated Test Description",
		}

		course, err := database.UpdateCourse(t.Context(), updatedCourse)
		require.NoError(t, err, "Should update course without error")
		assert.Equal(t, updatedCourse.GetCourseName(), course.CourseName, "Course name should be updated")
		assert.Equal(t, updatedCourse.GetDescription(), course.Description, "Course description should be updated")
	})

	// Test DeleteCourse.
	t.Run("DeleteCourse", func(t *testing.T) {
		err := database.DeleteCourse(t.Context(), testCourse.GetCourseID())
		require.NoError(t, err, "Should delete course without error")

		// Verify course was deleted.
		_, err = database.GetCourse(t.Context(), testCourse.GetCourseID())
		assert.Error(t, err, "Should return error when getting deleted course")
	})
}

// testEntityOperations is a helper function to test both student and staff operations
// since they follow the same pattern.
func testEntityOperations(t *testing.T, isStudent bool) {
	t.Helper()

	database := setupTestDatabase(t)
	defer cleanupTestDatabase(t, database)

	// Add a course for testing
	testCourse := buildTestCourse()
	_, err := database.AddCourse(t.Context(), testCourse)

	require.NoError(t, err, "Should add course without error")

	var entityID, entityType string
	if isStudent {
		entityID = "TESTSTUDENT1"
		entityType = "student"
	} else {
		entityID = "TESTSTAFF1"
		entityType = "staff"
	}

	// Test adding, getting, and removing entities
	testAddEntity(t, database, testCourse.GetCourseID(), entityID, entityType, isStudent)
	testGetEntityRelationships(t, database, testCourse.GetCourseID(), entityID, entityType, isStudent)
	testRemoveEntity(t, database, testCourse.GetCourseID(), entityID, entityType, isStudent)
}

// testAddEntity tests adding an entity (student or staff) to a course.
func testAddEntity(t *testing.T, database *Database, courseID, entityID, entityType string, isStudent bool) {
	t.Helper()

	var err error
	// Add entity to course
	if isStudent {
		err = database.AddStudentToCourse(t.Context(), courseID, entityID)
	} else {
		err = database.AddStaffToCourse(t.Context(), courseID, entityID)
	}

	require.NoError(t, err, "Should add %s to course without error", entityType)
}

// testGetEntityRelationships tests retrieving entity-course relationships.
func testGetEntityRelationships(t *testing.T, database *Database, courseID,
	entityID, entityType string, isStudent bool,
) {
	t.Helper()

	var err error
	// Get course entities
	var entities []string
	if isStudent {
		entities, err = database.GetCourseStudents(t.Context(), courseID)
	} else {
		entities, err = database.GetCourseStaff(t.Context(), courseID)
	}

	require.NoError(t, err, "Should get course %ss without error", entityType)
	assert.Contains(t, entities, entityID, "%ss list should contain the added %s ID", entityType, entityType)

	// Get entity courses
	var courses []string
	if isStudent {
		courses, err = database.GetStudentCourses(t.Context(), entityID)
	} else {
		courses, err = database.GetStaffCourses(t.Context(), entityID)
	}

	require.NoError(t, err, "Should get %s courses without error", entityType)
	assert.Contains(t, courses, courseID, "Courses list should contain the course ID")
}

// testRemoveEntity tests removing an entity (student or staff) from a course.
func testRemoveEntity(t *testing.T, database *Database, courseID, entityID, entityType string, isStudent bool) {
	t.Helper()

	var err error
	// Remove entity from course
	if isStudent {
		err = database.RemoveStudentFromCourse(t.Context(), courseID, entityID)
	} else {
		err = database.RemoveStaffFromCourse(t.Context(), courseID, entityID)
	}

	require.NoError(t, err, "Should remove %s from course without error", entityType)

	// Verify entity was removed
	var entities []string
	if isStudent {
		entities, err = database.GetCourseStudents(t.Context(), courseID)
	} else {
		entities, err = database.GetCourseStaff(t.Context(), courseID)
	}

	require.NoError(t, err, "Should get course %ss without error", entityType)
	assert.NotContains(t, entities, entityID, "%ss list should not contain the removed %s ID", entityType, entityType)
}

// testStudentEnrollment tests student enrollment functionality.
func testStudentEnrollment(t *testing.T) {
	testEntityOperations(t, true)
}

// testStaffAssignments tests staff assignment functionality.
func testStaffAssignments(t *testing.T) {
	testEntityOperations(t, false)
}

// testAnnouncements tests announcement functionality.
func testAnnouncements(t *testing.T) {
	database := setupTestDatabase(t)
	defer cleanupTestDatabase(t, database)

	// Add a course for testing
	testCourse := buildTestCourse()
	_, err := database.AddCourse(t.Context(), testCourse)
	require.NoError(t, err, "Should add course without error")

	announcementID := "TESTANNOUNCEMENT1"
	announcement := &cpb.AddAnnouncementRequest{
		CourseID: testCourse.GetCourseID(),
		Announcement: &cpb.Announcement{
			AnnouncementID:      announcementID,
			AnnouncementTitle:   "Test Announcement",
			AnnouncementContent: "This is a test announcement.",
		},
	}

	// Add announcement.
	err = database.AddAnnouncement(t.Context(), announcement)
	require.NoError(t, err, "Should add announcement without error")

	// Get announcements.
	announcements, err := database.GetAnnouncements(t.Context(), testCourse.GetCourseID())
	require.NoError(t, err, "Should get announcements without error")
	assert.NotEmpty(t, announcements, "Announcements list should not be empty")

	// Remove announcement.
	err = database.RemoveAnnouncement(t.Context(), testCourse.GetCourseID(), announcementID)
	require.NoError(t, err, "Should remove announcement without error")
}
