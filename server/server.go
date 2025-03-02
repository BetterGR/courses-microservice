package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	"os"

	cpb "github.com/BetterGR/courses-microservice/protos"
	ms "github.com/TekClinic/MicroService-Lib"
	"github.com/joho/godotenv"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"k8s.io/klog/v2"
)

const (
	// define address.
	connectionProtocol = "tcp"
	// Debugging logs.
	logLevelDebug = 5
)

// CoursesServer is an implementation of GRPC Courses microservice.
type CoursesServer struct {
	ms.BaseServiceServer
	db DBInterface
	cpb.UnimplementedCoursesServiceServer
	Claims ms.Claims
}

// VerifyToken returns the injected Claims instead of the default.
func (s *CoursesServer) VerifyToken(ctx context.Context, token string) error {
	if s.Claims != nil {
		return nil
	}

	// Default behavior.
	if _, err := s.BaseServiceServer.VerifyToken(ctx, token); err != nil {
		return fmt.Errorf("failed to verify token: %w", err)
	}

	return nil
}

// initCoursesMicroserviceServer initializes the CoursesServer.
func initCoursesMicroserviceServer() (*CoursesServer, error) {
	base, err := ms.CreateBaseServiceServer()
	if err != nil {
		return nil, fmt.Errorf("failed to create base service: %w", err)
	}

	database, err := InitializeDatabase()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	return &CoursesServer{
		BaseServiceServer:                 base,
		db:                                database,
		UnimplementedCoursesServiceServer: cpb.UnimplementedCoursesServiceServer{},
	}, nil
}

// GetCourse retrieves a course by its ID.
func (s *CoursesServer) GetCourse(ctx context.Context, req *cpb.GetCourseRequest) (*cpb.GetCourseResponse, error) {
	if err := s.VerifyToken(ctx, req.GetToken()); err != nil {
		return nil, fmt.Errorf("authentication failed: %w",
			status.Error(codes.Unauthenticated, err.Error()))
	}

	logger := klog.FromContext(ctx)
	logger.V(logLevelDebug).Info("Received GetCourse request", "courseId", req.GetCourseID())

	course, err := s.db.GetCourse(ctx, req.GetCourseID())
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "course not found: %v", err)
	}

	newCourse := &cpb.Course{
		CourseID:    course.CourseID,
		CourseName:  course.CourseName,
		Semester:    course.Semester,
		Description: course.Description,
	}

	return &cpb.GetCourseResponse{Course: newCourse}, nil
}

// CreateCourse creates a new course.
func (s *CoursesServer) CreateCourse(
	ctx context.Context,
	req *cpb.CreateCourseRequest,
) (*cpb.CreateCourseResponse, error) {
	if err := s.VerifyToken(ctx, req.GetToken()); err != nil {
		return nil, fmt.Errorf("authentication failed: %w",
			status.Error(codes.Unauthenticated, err.Error()))
	}

	logger := klog.FromContext(ctx)
	logger.V(logLevelDebug).Info("Received CreateCourse request", "courseName", req.GetCourse().GetCourseName())

	if _, err := s.db.AddCourse(ctx, &cpb.Course{
		CourseID:    req.GetCourse().GetCourseID(),
		CourseName:  req.GetCourse().GetCourseName(),
		Semester:    req.GetCourse().GetSemester(),
		Description: req.GetCourse().GetDescription(),
	}); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to add course: %v", err)
	}

	return &cpb.CreateCourseResponse{Course: req.GetCourse()}, nil
}

// UpdateCourse updates an existing course.
func (s *CoursesServer) UpdateCourse(
	ctx context.Context,
	req *cpb.UpdateCourseRequest,
) (*cpb.UpdateCourseResponse, error) {
	if err := s.VerifyToken(ctx, req.GetToken()); err != nil {
		return nil, fmt.Errorf("authentication failed: %w",
			status.Error(codes.Unauthenticated, err.Error()))
	}

	logger := klog.FromContext(ctx)
	logger.V(logLevelDebug).Info("Received UpdateCourse request", "courseId", req.GetCourse().GetCourseID())

	updatedCourse, err := s.db.UpdateCourse(ctx, req.GetCourse())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to update course: %v", err)
	}

	course := &cpb.Course{
		CourseID:    updatedCourse.CourseID,
		CourseName:  updatedCourse.CourseName,
		Semester:    updatedCourse.Semester,
		Description: updatedCourse.Description,
	}

	return &cpb.UpdateCourseResponse{Course: course}, nil
}

// DeleteCourse deletes a course by its ID.
func (s *CoursesServer) DeleteCourse(
	ctx context.Context,
	req *cpb.DeleteCourseRequest,
) (*cpb.DeleteCourseResponse, error) {
	if err := s.VerifyToken(ctx, req.GetToken()); err != nil {
		return nil, fmt.Errorf("authentication failed: %w",
			status.Error(codes.Unauthenticated, err.Error()))
	}

	logger := klog.FromContext(ctx)
	logger.V(logLevelDebug).Info("Received DeleteCourse request", "courseId", req.GetCourseID())

	if err := s.db.DeleteCourse(ctx, req.GetCourseID()); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to delete course: %v", err)
	}

	return &cpb.DeleteCourseResponse{}, nil
}

// AddStudentToCourse adds a student to a course.
func (s *CoursesServer) AddStudentToCourse(
	ctx context.Context,
	req *cpb.AddStudentRequest,
) (*cpb.AddStudentResponse, error) {
	if err := s.VerifyToken(ctx, req.GetToken()); err != nil {
		return nil, fmt.Errorf("authentication failed: %w",
			status.Error(codes.Unauthenticated, err.Error()))
	}

	logger := klog.FromContext(ctx)
	logger.V(logLevelDebug).Info("Received AddStudentToCourse request",
		"courseId", req.GetCourseID(), "studentId", req.GetStudentID())

	if err := s.db.AddStudentToCourse(ctx, req.GetCourseID(), req.GetStudentID()); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to add student to course: %v", err)
	}

	return &cpb.AddStudentResponse{}, nil
}

// RemoveStudentFromCourse removes a student from a course.
func (s *CoursesServer) RemoveStudentFromCourse(
	ctx context.Context,
	req *cpb.RemoveStudentRequest,
) (*cpb.RemoveStudentResponse, error) {
	if err := s.VerifyToken(ctx, req.GetToken()); err != nil {
		return nil, fmt.Errorf("authentication failed: %w",
			status.Error(codes.Unauthenticated, err.Error()))
	}

	logger := klog.FromContext(ctx)
	logger.V(logLevelDebug).Info("Received RemoveStudentFromCourse request",
		"courseId", req.GetCourseID(), "studentId", req.GetStudentID())

	if err := s.db.RemoveStudentFromCourse(ctx, req.GetCourseID(), req.GetStudentID()); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to remove student from course: %v", err)
	}

	return &cpb.RemoveStudentResponse{}, nil
}

// AddStaffToCourse adds a staff member to a course.
func (s *CoursesServer) AddStaffToCourse(ctx context.Context, req *cpb.AddStaffRequest) (*cpb.AddStaffResponse, error) {
	if err := s.VerifyToken(ctx, req.GetToken()); err != nil {
		return nil, fmt.Errorf("authentication failed: %w",
			status.Error(codes.Unauthenticated, err.Error()))
	}

	logger := klog.FromContext(ctx)
	logger.V(logLevelDebug).Info("Received AddStaffToCourse request",
		"courseId", req.GetCourseID(), "staffId", req.GetStaffID())

	if err := s.db.AddStaffToCourse(ctx, req.GetCourseID(), req.GetStaffID()); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to add staff to course: %v", err)
	}

	return &cpb.AddStaffResponse{}, nil
}

// RemoveStaffFromCourse removes a staff member from a course.
func (s *CoursesServer) RemoveStaffFromCourse(
	ctx context.Context,
	req *cpb.RemoveStaffRequest,
) (*cpb.RemoveStaffResponse, error) {
	if err := s.VerifyToken(ctx, req.GetToken()); err != nil {
		return nil, fmt.Errorf("authentication failed: %w",
			status.Error(codes.Unauthenticated, err.Error()))
	}

	logger := klog.FromContext(ctx)
	logger.V(logLevelDebug).Info("Received RemoveStaffFromCourse request",
		"courseId", req.GetCourseID(), "staffId", req.GetStaffID())

	if err := s.db.RemoveStaffFromCourse(ctx, req.GetCourseID(), req.GetStaffID()); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to remove staff from course: %v", err)
	}

	return &cpb.RemoveStaffResponse{}, nil
}

// GetCourseStudents retrieves the students enrolled in a course.
func (s *CoursesServer) GetCourseStudents(
	ctx context.Context,
	req *cpb.GetCourseStudentsRequest,
) (*cpb.GetCourseStudentsResponse, error) {
	if err := s.VerifyToken(ctx, req.GetToken()); err != nil {
		return nil, fmt.Errorf("authentication failed: %w",
			status.Error(codes.Unauthenticated, err.Error()))
	}

	logger := klog.FromContext(ctx)
	logger.V(logLevelDebug).Info("Received GetCourseStudents request", "courseId", req.GetCourseID())

	studentIDs, err := s.db.GetCourseStudents(ctx, req.GetCourseID())
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "course not found: %v", err)
	}

	return &cpb.GetCourseStudentsResponse{StudentsIDs: studentIDs}, nil
}

// GetCourseStaff retrieves the staff members assigned to a course.
func (s *CoursesServer) GetCourseStaff(ctx context.Context,
	req *cpb.GetCourseStaffRequest,
) (*cpb.GetCourseStaffResponse, error) {
	if err := s.VerifyToken(ctx, req.GetToken()); err != nil {
		return nil, fmt.Errorf("authentication failed: %w",
			status.Error(codes.Unauthenticated, err.Error()))
	}

	logger := klog.FromContext(ctx)
	logger.V(logLevelDebug).Info("Received GetCourseStaff request", "courseId", req.GetCourseID())

	staffIDs, err := s.db.GetCourseStaff(ctx, req.GetCourseID())
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "course not found: %v", err)
	}

	return &cpb.GetCourseStaffResponse{StaffIDs: staffIDs}, nil
}

// GetStudentCourses retrieves the courses a student is enrolled in.
func (s *CoursesServer) GetStudentCourses(ctx context.Context,
	req *cpb.GetStudentCoursesRequest,
) (*cpb.GetStudentCoursesResponse, error) {
	if err := s.VerifyToken(ctx, req.GetToken()); err != nil {
		return nil, fmt.Errorf("authentication failed: %w",
			status.Error(codes.Unauthenticated, err.Error()))
	}

	logger := klog.FromContext(ctx)
	logger.V(logLevelDebug).Info("Received GetStudentCourses request", "studentId", req.GetStudentID())

	courseIDs, err := s.db.GetStudentCourses(ctx, req.GetStudentID())
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "student not found: %v", err)
	}

	return &cpb.GetStudentCoursesResponse{CoursesIDs: courseIDs}, nil
}

// GetStaffCourses retrieves the courses a staff member is associated with.
func (s *CoursesServer) GetStaffCourses(ctx context.Context,
	req *cpb.GetStaffCoursesRequest,
) (*cpb.GetStaffCoursesResponse, error) {
	if err := s.VerifyToken(ctx, req.GetToken()); err != nil {
		return nil, fmt.Errorf("authentication failed: %w",
			status.Error(codes.Unauthenticated, err.Error()))
	}

	logger := klog.FromContext(ctx)
	logger.V(logLevelDebug).Info("Received GetStaffCourses request", "staffId", req.GetStaffID())

	courseIDs, err := s.db.GetStaffCourses(ctx, req.GetStaffID())
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "staff not found: %v", err)
	}

	return &cpb.GetStaffCoursesResponse{CoursesIDs: courseIDs}, nil
}

// AddAnnouncementToCourse adds an announcement to a course.
func (s *CoursesServer) AddAnnouncementToCourse(ctx context.Context,
	req *cpb.AddAnnouncementRequest,
) (*cpb.AddAnnouncementResponse, error) {
	if err := s.VerifyToken(ctx, req.GetToken()); err != nil {
		return nil, fmt.Errorf("authentication failed: %w",
			status.Error(codes.Unauthenticated, err.Error()))
	}

	logger := klog.FromContext(ctx)
	logger.V(logLevelDebug).Info("Received AddAnnouncementToCourse request",
		"courseId", req.GetCourseID())

	if err := s.db.AddAnnouncement(ctx, req); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to add announcement to course: %v", err)
	}

	return &cpb.AddAnnouncementResponse{}, nil
}

// GetCourseAnnouncements retrieves the announcements associated with a course.
func (s *CoursesServer) GetCourseAnnouncements(ctx context.Context,
	req *cpb.GetCourseAnnouncementsRequest,
) (*cpb.GetCourseAnnouncementsResponse, error) {
	if err := s.VerifyToken(ctx, req.GetToken()); err != nil {
		return nil, fmt.Errorf("authentication failed: %w",
			status.Error(codes.Unauthenticated, err.Error()))
	}

	logger := klog.FromContext(ctx)
	logger.V(logLevelDebug).Info("Received GetCourseAnnouncements request", "courseId", req.GetCourseID())

	resp, err := s.db.GetAnnouncements(ctx, req.GetCourseID())
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "course not found: %v", err)
	}

	announcements := make([]*cpb.Announcement, 0)
	for _, a := range resp {
		announcements = append(announcements, &cpb.Announcement{
			AnnouncementID:      a.AnnouncementID,
			AnnouncementTitle:   a.Title,
			AnnouncementContent: a.Content,
		})
	}

	return &cpb.GetCourseAnnouncementsResponse{Announcements: announcements}, nil
}

// RemoveAnnouncementFromCourse removes an announcement from a course.
func (s *CoursesServer) RemoveAnnouncementFromCourse(ctx context.Context,
	req *cpb.RemoveAnnouncementRequest,
) (*cpb.RemoveAnnouncementResponse, error) {
	if err := s.VerifyToken(ctx, req.GetToken()); err != nil {
		return nil, fmt.Errorf("authentication failed: %w",
			status.Error(codes.Unauthenticated, err.Error()))
	}

	logger := klog.FromContext(ctx)
	logger.V(logLevelDebug).Info("Received RemoveAnnouncementFromCourse request",
		"courseId", req.GetCourseID(), "announcementId", req.GetAnnouncementID())

	if err := s.db.RemoveAnnouncement(ctx, req.GetCourseID(), req.GetAnnouncementID()); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to remove announcement from course: %v", err)
	}

	return &cpb.RemoveAnnouncementResponse{}, nil
}

func main() {
	// init klog.
	klog.InitFlags(nil)
	flag.Parse()

	err := godotenv.Load()
	if err != nil {
		klog.Fatalf("Error loading .env file")
	}

	// init the CoursesServer.
	server, err := initCoursesMicroserviceServer()
	if err != nil {
		klog.Fatalf("Failed to init CoursesServer: %v", err)
	}

	// create a listener on port 'address'.
	address := "localhost:" + os.Getenv("GRPC_PORT")

	lis, err := net.Listen(connectionProtocol, address)
	if err != nil {
		klog.Fatalf("Failed to listen: %v", err)
	}

	klog.V(logLevelDebug).Info("Starting CoursesServer on port: ", address)
	// create a grpc CoursesServer.
	grpcServer := grpc.NewServer()
	cpb.RegisterCoursesServiceServer(grpcServer, server)

	// serve the grpc CoursesServer.
	if err := grpcServer.Serve(lis); err != nil {
		klog.Fatalf("Failed to serve: %v", err)
	}
}
