package main

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"strings"
	"testing"

	cpb "github.com/BetterGR/courses-microservice/protos"
	ms "github.com/TekClinic/MicroService-Lib"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"k8s.io/klog/v2"
)

// MockClaims overrides Claims behavior for testing.
type MockClaims struct {
	ms.Claims
}

// Always return true for HasRole.
func (m MockClaims) HasRole(_ string) bool {
	return true
}

// Always return "course" for GetRole.
func (m MockClaims) GetRole() string {
	return "test-role"
}

// TestCoursesServer wraps CoursesServer for testing.
type TestCoursesServer struct {
	*CoursesServer
}

func TestMain(m *testing.M) {
	// Load .env file.
	cmd := exec.Command("cat", "../.env")

	output, err := cmd.Output()
	if err != nil {
		panic("Error reading .env file: " + err.Error())
	}

	// Set environment variables.
	for _, line := range strings.Split(string(output), "\n") {
		if line = strings.TrimSpace(line); line != "" && !strings.HasPrefix(line, "#") {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				// Remove quotes from the value if they exist.
				value := strings.Trim(parts[1], `"'`)
				os.Setenv(parts[0], value)
			}
		}
	}

	// Run tests and capture the result.
	result := m.Run()

	if result == 0 {
		klog.Info("\n\n [Summary] All tests passed.")
	} else {
		klog.Errorf("\n\n [Summary] Some tests failed. number of tests that failed: %d", result)
	}

	// Exit with the test result code.
	os.Exit(result)
}

func createTestCourse() *cpb.Course {
	return &cpb.Course{
		CourseID:    uuid.New().String(),
		CourseName:  "Test Course",
		Semester:    "Fall 2023",
		Description: "This is a test course.",
	}
}

func startTestServer() (*grpc.Server, net.Listener, *TestCoursesServer, error) {
	server, err := initCoursesMicroserviceServer()
	if err != nil {
		return nil, nil, nil, err
	}

	server.Claims = MockClaims{}
	testServer := &TestCoursesServer{CoursesServer: server}
	grpcServer := grpc.NewServer()
	cpb.RegisterCoursesServiceServer(grpcServer, testServer)

	listener, err := net.Listen(connectionProtocol, os.Getenv("GRPC_PORT"))
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to listen on port %s: %w", os.Getenv("GRPC_PORT"), err)
	}

	go func() {
		if err := grpcServer.Serve(listener); err != nil {
			panic("Failed to serve: " + err.Error())
		}
	}()

	return grpcServer, listener, testServer, nil
}

func setupClient(t *testing.T) cpb.CoursesServiceClient {
	t.Helper()

	grpcServer, listener, _, err := startTestServer()
	require.NoError(t, err)
	t.Cleanup(func() {
		grpcServer.Stop()
	})

	conn, err := grpc.NewClient(listener.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	t.Cleanup(func() {
		conn.Close()
	})

	return cpb.NewCoursesServiceClient(conn)
}

func createAndCleanupCourse(t *testing.T, client cpb.CoursesServiceClient) *cpb.Course {
	t.Helper()

	course := createTestCourse()
	_, err := client.CreateCourse(t.Context(), &cpb.CreateCourseRequest{Course: course, Token: "test-token"})
	require.NoError(t, err)

	t.Cleanup(func() {
		_, _ = client.DeleteCourse(t.Context(), &cpb.DeleteCourseRequest{CourseID: course.GetCourseID(), Token: "test-token"})
	})

	return course
}

func TestGetCourseFound(t *testing.T) {
	client := setupClient(t)
	course := createAndCleanupCourse(t, client)

	req := &cpb.GetCourseRequest{CourseID: course.GetCourseID(), Token: "test-token"}
	resp, err := client.GetCourse(t.Context(), req)
	require.NoError(t, err)
	assert.Equal(t, course.GetCourseID(), resp.GetCourse().GetCourseID())
}

func TestGetCourseNotFound(t *testing.T) {
	client := setupClient(t)
	req := &cpb.GetCourseRequest{CourseID: "non-existent-id", Token: "test-token"}

	_, err := client.GetCourse(t.Context(), req)
	assert.Error(t, err)
}

func TestCreateCourseSuccessful(t *testing.T) {
	client := setupClient(t)
	course := createTestCourse()
	req := &cpb.CreateCourseRequest{Course: course, Token: "test-token"}

	resp, err := client.CreateCourse(t.Context(), req)
	require.NoError(t, err)
	assert.Equal(t, course.GetCourseID(), resp.GetCourse().GetCourseID())
}

func TestCreateCourseFailureOnDuplicate(t *testing.T) {
	client := setupClient(t)
	course := createAndCleanupCourse(t, client)

	req := &cpb.CreateCourseRequest{Course: course, Token: "test-token"}
	_, err := client.CreateCourse(t.Context(), req)
	require.Error(t, err)
}

func TestUpdateCourseSuccessful(t *testing.T) {
	client := setupClient(t)
	course := createAndCleanupCourse(t, client)

	course.CourseName = "Updated Course Name"
	req := &cpb.UpdateCourseRequest{Course: course, Token: "test-token"}

	resp, err := client.UpdateCourse(t.Context(), req)
	require.NoError(t, err)
	assert.Equal(t, course.GetCourseID(), resp.GetCourse().GetCourseID())
	assert.Equal(t, course.GetCourseName(), resp.GetCourse().GetCourseName())
}

func TestUpdateCourseFailureForNonExistentCourse(t *testing.T) {
	client := setupClient(t)
	course := createTestCourse()
	course.CourseID = "non-existent-id"
	req := &cpb.UpdateCourseRequest{Course: course, Token: "test-token"}

	_, err := client.UpdateCourse(t.Context(), req)
	assert.Error(t, err)
}

func TestDeleteCourseSuccessful(t *testing.T) {
	client := setupClient(t)
	course := createAndCleanupCourse(t, client)

	req := &cpb.DeleteCourseRequest{CourseID: course.GetCourseID(), Token: "test-token"}
	_, err := client.DeleteCourse(t.Context(), req)
	assert.NoError(t, err)
}

func TestDeleteCourseFailureForNonExistentCourse(t *testing.T) {
	client := setupClient(t)
	req := &cpb.DeleteCourseRequest{CourseID: "non-existent-id", Token: "test-token"}

	_, err := client.DeleteCourse(t.Context(), req)
	assert.Error(t, err)
}

func TestAddStudentToCourse(t *testing.T) {
	client := setupClient(t)
	course := createAndCleanupCourse(t, client)

	req := &cpb.AddStudentRequest{CourseID: course.GetCourseID(), StudentID: "student-1", Token: "test-token"}
	_, err := client.AddStudentToCourse(t.Context(), req)
	require.NoError(t, err)
}

func TestRemoveStudentFromCourse(t *testing.T) {
	client := setupClient(t)
	course := createAndCleanupCourse(t, client)

	_, err := client.AddStudentToCourse(t.Context(),
		&cpb.AddStudentRequest{CourseID: course.GetCourseID(), StudentID: "student-1", Token: "test-token"})
	require.NoError(t, err)

	req := &cpb.RemoveStudentRequest{CourseID: course.GetCourseID(), StudentID: "student-1", Token: "test-token"}
	_, err = client.RemoveStudentFromCourse(t.Context(), req)
	require.NoError(t, err)
}

func TestAddStaffToCourse(t *testing.T) {
	client := setupClient(t)
	course := createAndCleanupCourse(t, client)

	req := &cpb.AddStaffRequest{CourseID: course.GetCourseID(), StaffID: "staff-1", Token: "test-token"}
	_, err := client.AddStaffToCourse(t.Context(), req)
	require.NoError(t, err)
}

func TestRemoveStaffFromCourse(t *testing.T) {
	client := setupClient(t)
	course := createAndCleanupCourse(t, client)

	_, err := client.AddStaffToCourse(t.Context(),
		&cpb.AddStaffRequest{CourseID: course.GetCourseID(), StaffID: "staff-1", Token: "test-token"})
	require.NoError(t, err)

	req := &cpb.RemoveStaffRequest{CourseID: course.GetCourseID(), StaffID: "staff-1", Token: "test-token"}
	_, err = client.RemoveStaffFromCourse(t.Context(), req)
	require.NoError(t, err)
}

func TestGetCourseStudents(t *testing.T) {
	client := setupClient(t)
	course := createAndCleanupCourse(t, client)

	_, err := client.AddStudentToCourse(t.Context(),
		&cpb.AddStudentRequest{CourseID: course.GetCourseID(), StudentID: "student-1", Token: "test-token"})
	require.NoError(t, err)

	req := &cpb.GetCourseStudentsRequest{CourseID: course.GetCourseID(), Token: "test-token"}
	resp, err := client.GetCourseStudents(t.Context(), req)
	require.NoError(t, err)
	assert.Contains(t, resp.GetStudentsIDs(), "student-1")
}

func TestGetCourseStaff(t *testing.T) {
	client := setupClient(t)
	course := createAndCleanupCourse(t, client)

	_, err := client.AddStaffToCourse(t.Context(),
		&cpb.AddStaffRequest{CourseID: course.GetCourseID(), StaffID: "staff-1", Token: "test-token"})
	require.NoError(t, err)

	req := &cpb.GetCourseStaffRequest{CourseID: course.GetCourseID(), Token: "test-token"}
	resp, err := client.GetCourseStaff(t.Context(), req)
	require.NoError(t, err)
	assert.Contains(t, resp.GetStaffIDs(), "staff-1")
}

func TestGetStudentCourses(t *testing.T) {
	client := setupClient(t)
	course := createAndCleanupCourse(t, client)

	_, err := client.AddStudentToCourse(t.Context(),
		&cpb.AddStudentRequest{CourseID: course.GetCourseID(), StudentID: "student-1", Token: "test-token"})
	require.NoError(t, err)

	req := &cpb.GetStudentCoursesRequest{StudentID: "student-1", Token: "test-token"}
	resp, err := client.GetStudentCourses(t.Context(), req)
	require.NoError(t, err)
	assert.Contains(t, resp.GetCoursesIDs(), course.GetCourseID())
}

func TestGetStaffCourses(t *testing.T) {
	client := setupClient(t)
	course := createAndCleanupCourse(t, client)

	_, err := client.AddStaffToCourse(t.Context(),
		&cpb.AddStaffRequest{CourseID: course.GetCourseID(), StaffID: "staff-1", Token: "test-token"})
	require.NoError(t, err)

	req := &cpb.GetStaffCoursesRequest{StaffID: "staff-1", Token: "test-token"}
	resp, err := client.GetStaffCourses(t.Context(), req)
	require.NoError(t, err)
	assert.Contains(t, resp.GetCoursesIDs(), course.GetCourseID())
}

func TestAddAnnouncementToCourse(t *testing.T) {
	client := setupClient(t)
	course := createAndCleanupCourse(t, client)

	req := &cpb.AddAnnouncementRequest{
		CourseID:     course.GetCourseID(),
		Announcement: "New Announcement", Token: "test-token",
	}
	_, err := client.AddAnnouncementToCourse(t.Context(), req)
	require.NoError(t, err)
}

func TestGetCourseAnnouncements(t *testing.T) {
	client := setupClient(t)
	course := createAndCleanupCourse(t, client)

	_, err := client.AddAnnouncementToCourse(t.Context(),
		&cpb.AddAnnouncementRequest{CourseID: course.GetCourseID(), Announcement: "New Announcement", Token: "test-token"})
	require.NoError(t, err)

	req := &cpb.GetCourseAnnouncementsRequest{CourseID: course.GetCourseID(), Token: "test-token"}
	resp, err := client.GetCourseAnnouncements(t.Context(), req)
	require.NoError(t, err)
	assert.Contains(t, resp.GetAnnouncements(), "New Announcement")
}

func TestRemoveAnnouncementFromCourse(t *testing.T) {
	client := setupClient(t)
	course := createAndCleanupCourse(t, client)

	_, err := client.AddAnnouncementToCourse(t.Context(),
		&cpb.AddAnnouncementRequest{CourseID: course.GetCourseID(), Announcement: "New Announcement", Token: "test-token"})
	require.NoError(t, err)

	req := &cpb.RemoveAnnouncementRequest{
		CourseID:       course.GetCourseID(),
		AnnouncementID: "New Announcement", Token: "test-token",
	}
	_, err = client.RemoveAnnouncementFromCourse(t.Context(), req)
	require.NoError(t, err)
}
