package main

import (
	"context"
	"errors"
	"fmt"
	"sync"

	cpb "github.com/BetterGR/courses-microservice/protos"
)

// ErrCourseAlreadyExists is returned when trying to add a course that already exists.
var ErrCourseAlreadyExists = errors.New("course already exists")

// MockDatabase is a in-memory implementation of DBInterface for testing.
type MockDatabase struct {
	courses        map[string]*Course
	courseStudents map[string][]string
	courseStaff    map[string][]string
	studentCourses map[string][]string
	staffCourses   map[string][]string
	announcements  map[string][]Announcement
	mutex          sync.RWMutex
}

// Verify that MockDatabase implements DBInterface at compile time.
var _ DBInterface = (*MockDatabase)(nil)

// NewMockDatabase creates a new MockDatabase instance.
func NewMockDatabase() *MockDatabase {
	return &MockDatabase{
		courses:        make(map[string]*Course),
		courseStudents: make(map[string][]string),
		courseStaff:    make(map[string][]string),
		studentCourses: make(map[string][]string),
		staffCourses:   make(map[string][]string),
		announcements:  make(map[string][]Announcement),
	}
}

// AddCourse adds a course to the mock database.
func (m *MockDatabase) AddCourse(_ context.Context, course *cpb.Course) (*Course, error) {
	if course == nil {
		return nil, fmt.Errorf("%w", ErrCourseNil)
	}

	if course.GetCourseID() == "" {
		return nil, fmt.Errorf("%w", ErrCourseIDEmpty)
	}

	m.mutex.Lock()
	defer m.mutex.Unlock()

	// Check if course already exists.
	if _, exists := m.courses[course.GetCourseID()]; exists {
		return nil, ErrCourseAlreadyExists
	}

	newCourse := &Course{
		CourseID:    course.GetCourseID(),
		CourseName:  course.GetCourseName(),
		Semester:    course.GetSemester(),
		Description: course.GetDescription(),
	}

	m.courses[course.GetCourseID()] = newCourse

	return newCourse, nil
}

// GetCourse retrieves a course from the mock database.
func (m *MockDatabase) GetCourse(_ context.Context, courseID string) (*Course, error) {
	if courseID == "" {
		return nil, fmt.Errorf("%w", ErrCourseIDEmpty)
	}

	m.mutex.RLock()
	defer m.mutex.RUnlock()

	course, exists := m.courses[courseID]
	if !exists {
		return nil, fmt.Errorf("%w", ErrCourseNotFound)
	}

	return course, nil
}

// UpdateCourse updates a course in the mock database.
func (m *MockDatabase) UpdateCourse(_ context.Context, course *cpb.Course) (*Course, error) {
	if course == nil {
		return nil, fmt.Errorf("%w", ErrCourseNil)
	}

	if course.GetCourseID() == "" {
		return nil, fmt.Errorf("%w", ErrCourseIDEmpty)
	}

	m.mutex.Lock()
	defer m.mutex.Unlock()

	existingCourse, exists := m.courses[course.GetCourseID()]
	if !exists {
		return nil, fmt.Errorf("%w", ErrCourseNotFound)
	}

	// Update the fields.
	if course.GetCourseName() != "" {
		existingCourse.CourseName = course.GetCourseName()
	}

	if course.GetSemester() != "" {
		existingCourse.Semester = course.GetSemester()
	}

	if course.GetDescription() != "" {
		existingCourse.Description = course.GetDescription()
	}

	m.courses[course.GetCourseID()] = existingCourse

	return existingCourse, nil
}

// DeleteCourse removes a course from the mock database.
func (m *MockDatabase) DeleteCourse(_ context.Context, courseID string) error {
	if courseID == "" {
		return fmt.Errorf("%w", ErrCourseIDEmpty)
	}

	m.mutex.Lock()
	defer m.mutex.Unlock()

	if _, exists := m.courses[courseID]; !exists {
		return fmt.Errorf("%w", ErrCourseNotFound)
	}

	delete(m.courses, courseID)
	delete(m.courseStudents, courseID)
	delete(m.courseStaff, courseID)
	delete(m.announcements, courseID)

	// Clean up student-course associations.
	for studentID, courses := range m.studentCourses {
		updatedCourses := make([]string, 0)

		for _, cID := range courses {
			if cID != courseID {
				updatedCourses = append(updatedCourses, cID)
			}
		}

		m.studentCourses[studentID] = updatedCourses
	}

	// Clean up staff-course associations.
	for staffID, courses := range m.staffCourses {
		updatedCourses := make([]string, 0)

		for _, cID := range courses {
			if cID != courseID {
				updatedCourses = append(updatedCourses, cID)
			}
		}

		m.staffCourses[staffID] = updatedCourses
	}

	return nil
}

// addEntityToCourse is a helper method for adding a student or staff to a course.
func (m *MockDatabase) addEntityToCourse(courseID, entityID string,
	entityMap map[string][]string, courseMap map[string][]string, emptyErr error,
) error {
	if courseID == "" {
		return fmt.Errorf("%w", ErrCourseIDEmpty)
	}

	if entityID == "" {
		return fmt.Errorf("%w", emptyErr)
	}

	// Check if course exists.
	if _, exists := m.courses[courseID]; !exists {
		return fmt.Errorf("%w", ErrCourseNotFound)
	}

	// Add entity to course.
	if _, exists := entityMap[courseID]; !exists {
		entityMap[courseID] = make([]string, 0)
	}

	for _, eID := range entityMap[courseID] {
		if eID == entityID {
			return nil
		}
	}

	entityMap[courseID] = append(entityMap[courseID], entityID)

	// Add course to entity.
	if _, exists := courseMap[entityID]; !exists {
		courseMap[entityID] = make([]string, 0)
	}

	for _, cID := range courseMap[entityID] {
		if cID == courseID {
			return nil
		}
	}

	courseMap[entityID] = append(courseMap[entityID], courseID)

	return nil
}

// validateRemoveEntityParams validates parameters for entity removal operations.
func (m *MockDatabase) validateRemoveEntityParams(courseID, entityID string, emptyErr error) error {
	if courseID == "" {
		return fmt.Errorf("%w", ErrCourseIDEmpty)
	}

	if entityID == "" {
		return fmt.Errorf("%w", emptyErr)
	}

	// Check if course exists
	if _, exists := m.courses[courseID]; !exists {
		return fmt.Errorf("%w", ErrCourseNotFound)
	}

	return nil
}

// removeEntityFromMap removes an entity from a course's entity list and returns true if found.
func (m *MockDatabase) removeEntityFromMap(courseID, entityID string, entityMap map[string][]string) bool {
	found := false

	if entities, exists := entityMap[courseID]; exists {
		updatedEntities := make([]string, 0)

		for _, eID := range entities {
			if eID != entityID {
				updatedEntities = append(updatedEntities, eID)
			} else {
				found = true
			}
		}

		entityMap[courseID] = updatedEntities
	}

	return found
}

// removeCourseFromEntityMap removes a course from an entity's course list.
func (m *MockDatabase) removeCourseFromEntityMap(courseID, entityID string, courseMap map[string][]string) {
	if courses, exists := courseMap[entityID]; exists {
		updatedCourses := make([]string, 0)

		for _, cID := range courses {
			if cID != courseID {
				updatedCourses = append(updatedCourses, cID)
			}
		}

		courseMap[entityID] = updatedCourses
	}
}

// removeEntityFromCourse is a helper method for removing a student or staff from a course.
func (m *MockDatabase) removeEntityFromCourse(courseID, entityID string,
	entityMap map[string][]string, courseMap map[string][]string, emptyErr error,
) error {
	// Validate inputs.
	if err := m.validateRemoveEntityParams(courseID, entityID, emptyErr); err != nil {
		return err
	}

	// Process entity removal.
	if !m.removeEntityFromMap(courseID, entityID, entityMap) {
		return fmt.Errorf("%w", ErrCourseNotFound)
	}

	// Process course removal from entity's list.
	m.removeCourseFromEntityMap(courseID, entityID, courseMap)

	return nil
}

// AddStudentToCourse adds a student to a course in the mock database.
func (m *MockDatabase) AddStudentToCourse(_ context.Context, courseID, studentID string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	return m.addEntityToCourse(courseID, studentID, m.courseStudents, m.studentCourses, ErrStudentIDEmpty)
}

// RemoveStudentFromCourse removes a student from a course in the mock database.
func (m *MockDatabase) RemoveStudentFromCourse(_ context.Context, courseID, studentID string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	return m.removeEntityFromCourse(courseID, studentID, m.courseStudents, m.studentCourses, ErrStudentIDEmpty)
}

// AddStaffToCourse adds a staff member to a course in the mock database.
func (m *MockDatabase) AddStaffToCourse(_ context.Context, courseID, staffID string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	return m.addEntityToCourse(courseID, staffID, m.courseStaff, m.staffCourses, ErrStaffIDEmpty)
}

// RemoveStaffFromCourse removes a staff member from a course in the mock database.
func (m *MockDatabase) RemoveStaffFromCourse(_ context.Context, courseID, staffID string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	return m.removeEntityFromCourse(courseID, staffID, m.courseStaff, m.staffCourses, ErrStaffIDEmpty)
}

// GetCourseStudents retrieves all students enrolled in a course from the mock database.
func (m *MockDatabase) GetCourseStudents(_ context.Context, courseID string) ([]string, error) {
	if courseID == "" {
		return nil, fmt.Errorf("%w", ErrCourseIDEmpty)
	}

	m.mutex.RLock()
	defer m.mutex.RUnlock()

	// Check if course exists.
	if _, exists := m.courses[courseID]; !exists {
		return nil, fmt.Errorf("%w", ErrCourseNotFound)
	}

	students, exists := m.courseStudents[courseID]
	if !exists {
		return []string{}, nil
	}

	// Return a copy to prevent modification of the original slice.
	result := make([]string, len(students))
	copy(result, students)

	return result, nil
}

// GetCourseStaff retrieves all staff members assigned to a course from the mock database.
func (m *MockDatabase) GetCourseStaff(_ context.Context, courseID string) ([]string, error) {
	if courseID == "" {
		return nil, fmt.Errorf("%w", ErrCourseIDEmpty)
	}

	m.mutex.RLock()
	defer m.mutex.RUnlock()

	// Check if course exists.
	if _, exists := m.courses[courseID]; !exists {
		return nil, fmt.Errorf("%w", ErrCourseNotFound)
	}

	staff, exists := m.courseStaff[courseID]
	if !exists {
		return []string{}, nil
	}

	// Return a copy to prevent modification of the original slice.
	result := make([]string, len(staff))
	copy(result, staff)

	return result, nil
}

// GetStudentCourses retrieves all courses a student is enrolled in from the mock database.
func (m *MockDatabase) GetStudentCourses(_ context.Context, studentID string) ([]string, error) {
	if studentID == "" {
		return nil, fmt.Errorf("%w", ErrStudentIDEmpty)
	}

	m.mutex.RLock()
	defer m.mutex.RUnlock()

	courses, exists := m.studentCourses[studentID]
	if !exists {
		return []string{}, nil
	}

	// Return a copy to prevent modification of the original slice.
	result := make([]string, len(courses))
	copy(result, courses)

	return result, nil
}

// GetStaffCourses retrieves all courses a staff member is assigned to from the mock database.
func (m *MockDatabase) GetStaffCourses(_ context.Context, staffID string) ([]string, error) {
	if staffID == "" {
		return nil, fmt.Errorf("%w", ErrStaffIDEmpty)
	}

	m.mutex.RLock()
	defer m.mutex.RUnlock()

	courses, exists := m.staffCourses[staffID]
	if !exists {
		return []string{}, nil
	}

	// Return a copy to prevent modification of the original slice
	result := make([]string, len(courses))
	copy(result, courses)

	return result, nil
}

// AddAnnouncement adds an announcement to a course in the mock database.
func (m *MockDatabase) AddAnnouncement(_ context.Context, req *cpb.AddAnnouncementRequest) error {
	if req.GetCourseID() == "" || req.GetAnnouncement().GetAnnouncementContent() == "" {
		return fmt.Errorf("%w", ErrCourseIDEmpty)
	}

	m.mutex.Lock()
	defer m.mutex.Unlock()

	// Check if course exists.
	if _, exists := m.courses[req.GetCourseID()]; !exists {
		return fmt.Errorf("%w", ErrCourseNotFound)
	}

	announcement := Announcement{
		CourseID:       req.GetCourseID(),
		AnnouncementID: req.GetAnnouncement().GetAnnouncementID(),
		Title:          req.GetAnnouncement().GetAnnouncementTitle(),
		Content:        req.GetAnnouncement().GetAnnouncementContent(),
	}

	if _, exists := m.announcements[req.GetCourseID()]; !exists {
		m.announcements[req.GetCourseID()] = make([]Announcement, 0)
	}

	m.announcements[req.GetCourseID()] = append(m.announcements[req.GetCourseID()], announcement)

	return nil
}

// GetAnnouncements retrieves all announcements for a course from the mock database.
func (m *MockDatabase) GetAnnouncements(_ context.Context, courseID string) ([]Announcement, error) {
	if courseID == "" {
		return nil, fmt.Errorf("%w", ErrCourseIDEmpty)
	}

	m.mutex.RLock()
	defer m.mutex.RUnlock()

	// Check if course exists.
	if _, exists := m.courses[courseID]; !exists {
		return nil, fmt.Errorf("%w", ErrCourseNotFound)
	}

	announcements, exists := m.announcements[courseID]
	if !exists {
		return []Announcement{}, nil
	}

	// Return a copy to prevent modification of the original slice.
	result := make([]Announcement, len(announcements))
	copy(result, announcements)

	return result, nil
}

// RemoveAnnouncement removes an announcement from a course in the mock database.
func (m *MockDatabase) RemoveAnnouncement(_ context.Context, courseID, announcementID string) error {
	if courseID == "" {
		return fmt.Errorf("%w", ErrCourseIDEmpty)
	}

	if announcementID == "" {
		return fmt.Errorf("%w", ErrAnnouncementEmpty)
	}

	m.mutex.Lock()
	defer m.mutex.Unlock()

	// Check if course exists.
	if _, exists := m.courses[courseID]; !exists {
		return fmt.Errorf("%w", ErrCourseNotFound)
	}

	// Remove announcement from course.
	found := false

	if announcements, exists := m.announcements[courseID]; exists {
		updatedAnnouncements := make([]Announcement, 0)

		for _, a := range announcements {
			if a.AnnouncementID != announcementID {
				updatedAnnouncements = append(updatedAnnouncements, a)
			} else {
				found = true
			}
		}

		m.announcements[courseID] = updatedAnnouncements
	}

	if !found {
		return fmt.Errorf("%w", ErrCourseNotFound)
	}

	return nil
}
