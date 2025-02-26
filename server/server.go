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
	db *Database
	// throws unimplemented error
	cpb.UnimplementedCoursesServiceServer
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
	if _, err := s.VerifyToken(ctx, req.GetToken()); err != nil {
		return nil, fmt.Errorf("authentication failed: %w",
			status.Error(codes.Unauthenticated, err.Error()))
	}

	logger := klog.FromContext(ctx)
	logger.V(logLevelDebug).Info("Received GetCourse request", "courseId", req.GetCourseId())

	course, err := s.db.GetCourse(ctx, req.GetCourseId())
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "course not found: %v", err)
	}

	return &cpb.GetCourseResponse{Course: &cpb.Course{
		CourseId:       course.GetCourseId(),
		CourseName:     course.GetCourseName(),
		Description:    course.GetDescription(),
		Semester:       course.GetSemester(),
		CourseMaterial: course.GetCourseMaterial(),
		Announcements:  course.GetAnnouncements(),
	}}, nil
}

// CreateCourse creates a new course.
func (s *CoursesServer) CreateCourse(
	ctx context.Context,
	req *cpb.CreateCourseRequest,
) (*cpb.CreateCourseResponse, error) {
	if _, err := s.VerifyToken(ctx, req.GetToken()); err != nil {
		return nil, fmt.Errorf("authentication failed: %w",
			status.Error(codes.Unauthenticated, err.Error()))
	}

	logger := klog.FromContext(ctx)
	logger.V(logLevelDebug).Info("Received CreateCourse request", "courseName", req.GetCourse().GetCourseName())

	if err := s.db.AddCourse(ctx, &cpb.Course{
		CourseId:       req.GetCourse().GetCourseId(),
		CourseName:     req.GetCourse().GetCourseName(),
		Semester:       req.GetCourse().GetSemester(),
		Description:    req.GetCourse().GetDescription(),
		CourseMaterial: "",
		Announcements:  []string{},
	}); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to add course: %v", err)
	}

	return &cpb.CreateCourseResponse{}, nil
}

// UpdateCourse updates an existing course.
func (s *CoursesServer) UpdateCourse(
	ctx context.Context,
	req *cpb.UpdateCourseRequest,
) (*cpb.UpdateCourseResponse, error) {
	if _, err := s.VerifyToken(ctx, req.GetToken()); err != nil {
		return nil, fmt.Errorf("authentication failed: %w",
			status.Error(codes.Unauthenticated, err.Error()))
	}

	logger := klog.FromContext(ctx)
	logger.V(logLevelDebug).Info("Received UpdateCourse request", "courseId", req.GetCourse().GetCourseId())

	course, err := s.db.GetCourse(ctx, req.GetCourse().GetCourseId())
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "course not found: %v", err)
	}

	course.CourseName = req.GetCourse().GetCourseName()
	course.Semester = req.GetCourse().GetSemester()
	course.Description = req.GetCourse().GetDescription()
	course.CourseMaterial = req.GetCourse().GetCourseMaterial()

	if err = s.db.UpdateCourse(ctx, course); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to update course: %v", err)
	}

	return &cpb.UpdateCourseResponse{}, nil
}

// DeleteCourse deletes a course by its ID.
func (s *CoursesServer) DeleteCourse(
	ctx context.Context,
	req *cpb.DeleteCourseRequest,
) (*cpb.DeleteCourseResponse, error) {
	if _, err := s.VerifyToken(ctx, req.GetToken()); err != nil {
		return nil, fmt.Errorf("authentication failed: %w",
			status.Error(codes.Unauthenticated, err.Error()))
	}

	logger := klog.FromContext(ctx)
	logger.V(logLevelDebug).Info("Received DeleteCourse request", "courseId", req.GetCourseId())

	if err := s.db.DeleteCourse(ctx, req.GetCourseId()); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to delete course: %v", err)
	}

	return &cpb.DeleteCourseResponse{}, nil
}

// AddStudentToCourse adds a student to a course.
func (s *CoursesServer) AddStudentToCourse(
	ctx context.Context,
	req *cpb.AddStudentRequest,
) (*cpb.AddStudentResponse, error) {
	if _, err := s.VerifyToken(ctx, req.GetToken()); err != nil {
		return nil, fmt.Errorf("authentication failed: %w",
			status.Error(codes.Unauthenticated, err.Error()))
	}

	logger := klog.FromContext(ctx)
	logger.V(logLevelDebug).Info("Received AddStudentToCourse request",
		"courseId", req.GetCourseId(), "studentId", req.GetStudentId())

	if err := s.db.AddStudentToCourse(ctx, req.GetCourseId(), req.GetStudentId()); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to add student to course: %v", err)
	}

	return &cpb.AddStudentResponse{}, nil
}

// RemoveStudentFromCourse removes a student from a course.
func (s *CoursesServer) RemoveStudentFromCourse(
	ctx context.Context,
	req *cpb.RemoveStudentRequest,
) (*cpb.RemoveStudentResponse, error) {
	if _, err := s.VerifyToken(ctx, req.GetToken()); err != nil {
		return nil, fmt.Errorf("authentication failed: %w",
			status.Error(codes.Unauthenticated, err.Error()))
	}

	logger := klog.FromContext(ctx)
	logger.V(logLevelDebug).Info("Received RemoveStudentFromCourse request",
		"courseId", req.GetCourseId(), "studentId", req.GetStudentId())

	if err := s.db.RemoveStudentFromCourse(ctx, req.GetCourseId(), req.GetStudentId()); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to remove student from course: %v", err)
	}

	return &cpb.RemoveStudentResponse{}, nil
}

// AddStaffToCourse adds a staff member to a course.
func (s *CoursesServer) AddStaffToCourse(ctx context.Context, req *cpb.AddStaffRequest) (*cpb.AddStaffResponse, error) {
	if _, err := s.VerifyToken(ctx, req.GetToken()); err != nil {
		return nil, fmt.Errorf("authentication failed: %w",
			status.Error(codes.Unauthenticated, err.Error()))
	}

	logger := klog.FromContext(ctx)
	logger.V(logLevelDebug).Info("Received AddStaffToCourse request",
		"courseId", req.GetCourseId(), "staffId", req.GetStaffId())

	if err := s.db.AddStaffToCourse(ctx, req.GetCourseId(), req.GetStaffId()); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to add staff to course: %v", err)
	}

	return &cpb.AddStaffResponse{}, nil
}

// RemoveStaffFromCourse removes a staff member from a course.
func (s *CoursesServer) RemoveStaffFromCourse(
	ctx context.Context,
	req *cpb.RemoveStaffRequest,
) (*cpb.RemoveStaffResponse, error) {
	if _, err := s.VerifyToken(ctx, req.GetToken()); err != nil {
		return nil, fmt.Errorf("authentication failed: %w",
			status.Error(codes.Unauthenticated, err.Error()))
	}

	logger := klog.FromContext(ctx)
	logger.V(logLevelDebug).Info("Received RemoveStaffFromCourse request",
		"courseId", req.GetCourseId(), "staffId", req.GetStaffId())

	if err := s.db.RemoveStaffFromCourse(ctx, req.GetCourseId(), req.GetStaffId()); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to remove staff from course: %v", err)
	}

	return &cpb.RemoveStaffResponse{}, nil
}

// GetCourseStudents retrieves the students enrolled in a course.
func (s *CoursesServer) GetCourseStudents(
	ctx context.Context,
	req *cpb.GetCourseStudentsRequest,
) (*cpb.GetCourseStudentsResponse, error) {
	if _, err := s.VerifyToken(ctx, req.GetToken()); err != nil {
		return nil, fmt.Errorf("authentication failed: %w",
			status.Error(codes.Unauthenticated, err.Error()))
	}

	logger := klog.FromContext(ctx)
	logger.V(logLevelDebug).Info("Received GetCourseStudents request", "courseId", req.GetCourseId())

	studentIDs, err := s.db.GetCourseStudents(ctx, req.GetCourseId())
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "course not found: %v", err)
	}

	return &cpb.GetCourseStudentsResponse{StudentIds: studentIDs}, nil
}

// GetCourseStaff retrieves the staff members assigned to a course.
func (s *CoursesServer) GetCourseStaff(ctx context.Context,
	req *cpb.GetCourseStaffRequest,
) (*cpb.GetCourseStaffResponse, error) {
	if _, err := s.VerifyToken(ctx, req.GetToken()); err != nil {
		return nil, fmt.Errorf("authentication failed: %w",
			status.Error(codes.Unauthenticated, err.Error()))
	}

	logger := klog.FromContext(ctx)
	logger.V(logLevelDebug).Info("Received GetCourseStaff request", "courseId", req.GetCourseId())

	staffIDs, err := s.db.GetCourseStaff(ctx, req.GetCourseId())
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "course not found: %v", err)
	}

	return &cpb.GetCourseStaffResponse{StaffIds: staffIDs}, nil
}

// GetStudentCourses retrieves the courses a student is enrolled in.
func (s *CoursesServer) GetStudentCourses(ctx context.Context,
	req *cpb.GetStudentCoursesRequest,
) (*cpb.GetStudentCoursesResponse, error) {
	if _, err := s.VerifyToken(ctx, req.GetToken()); err != nil {
		return nil, fmt.Errorf("authentication failed: %w",
			status.Error(codes.Unauthenticated, err.Error()))
	}

	logger := klog.FromContext(ctx)
	logger.V(logLevelDebug).Info("Received GetStudentCourses request", "studentId", req.GetStudentId())

	courseIDs, err := s.db.GetStudentCourses(ctx, req.GetStudentId())
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "student not found: %v", err)
	}

	return &cpb.GetStudentCoursesResponse{CoursesIds: courseIDs}, nil
}

// GetStaffCourses retrieves the courses a staff member is associated with.
func (s *CoursesServer) GetStaffCourses(ctx context.Context,
	req *cpb.GetStaffCoursesRequest,
) (*cpb.GetStaffCoursesResponse, error) {
	if _, err := s.VerifyToken(ctx, req.GetToken()); err != nil {
		return nil, fmt.Errorf("authentication failed: %w",
			status.Error(codes.Unauthenticated, err.Error()))
	}

	logger := klog.FromContext(ctx)
	logger.V(logLevelDebug).Info("Received GetStaffCourses request", "staffId", req.GetStaffId())

	courseIDs, err := s.db.GetStaffCourses(ctx, req.GetStaffId())
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "staff not found: %v", err)
	}

	return &cpb.GetStaffCoursesResponse{CoursesIds: courseIDs}, nil
}

// AddAnnouncementToCourse adds an announcement to a course.
func (s *CoursesServer) AddAnnouncementToCourse(ctx context.Context,
	req *cpb.AddAnnouncementRequest,
) (*cpb.AddAnnouncementResponse, error) {
	if _, err := s.VerifyToken(ctx, req.GetToken()); err != nil {
		return nil, fmt.Errorf("authentication failed: %w",
			status.Error(codes.Unauthenticated, err.Error()))
	}

	logger := klog.FromContext(ctx)
	logger.V(logLevelDebug).Info("Received AddAnnouncementToCourse request", "courseId", req.GetCourseId())

	if err := s.db.AddAnnouncement(ctx, req.GetCourseId(), req.GetAnnouncement()); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to add announcement to course: %v", err)
	}

	return &cpb.AddAnnouncementResponse{}, nil
}

// RemoveAnnouncementFromCourse removes an announcement from a course.
func (s *CoursesServer) RemoveAnnouncementFromCourse(ctx context.Context,
	req *cpb.RemoveAnnouncementRequest,
) (*cpb.RemoveAnnouncementResponse, error) {
	if _, err := s.VerifyToken(ctx, req.GetToken()); err != nil {
		return nil, fmt.Errorf("authentication failed: %w",
			status.Error(codes.Unauthenticated, err.Error()))
	}

	logger := klog.FromContext(ctx)
	logger.V(logLevelDebug).Info("Received RemoveAnnouncementFromCourse request",
		"courseId", req.GetCourseId(), "announcementId", req.GetAnnouncementId())

	if err := s.db.RemoveAnnouncement(ctx, req.GetCourseId(), req.GetAnnouncementId()); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to remove announcement from course: %v", err)
	}

	return &cpb.RemoveAnnouncementResponse{}, nil
}

// UpdateAnnouncementInCourse updates an announcement in a course.
func (s *CoursesServer) UpdateAnnouncementInCourse(ctx context.Context,
	req *cpb.UpdateAnnouncementRequest,
) (*cpb.UpdateAnnouncementResponse, error) {
	if _, err := s.VerifyToken(ctx, req.GetToken()); err != nil {
		return nil, fmt.Errorf("authentication failed: %w",
			status.Error(codes.Unauthenticated, err.Error()))
	}

	logger := klog.FromContext(ctx)
	logger.V(logLevelDebug).Info("Received UpdateAnnouncementInCourse request",
		"courseId", req.GetCourseId(), "announcementId", req.GetAnnouncementId())

	if err := s.db.UpdateAnnouncement(ctx, req.GetCourseId(),
		req.GetAnnouncementId(), req.GetUpdatedAnnouncement()); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to update announcement in course: %v", err)
	}

	return &cpb.UpdateAnnouncementResponse{}, nil
}

func main() {
	// init klog
	klog.InitFlags(nil)
	flag.Parse()

	err := godotenv.Load()
	if err != nil {
		klog.Fatalf("Error loading .env file")
	}

	// init the CoursesServer
	server, err := initCoursesMicroserviceServer()
	if err != nil {
		klog.Fatalf("Failed to init CoursesServer: %v", err)
	}

	// create a listener on port 'address'
	address := os.Getenv("GRPC_PORT")

	lis, err := net.Listen(connectionProtocol, address)
	if err != nil {
		klog.Fatalf("Failed to listen: %v", err)
	}

	klog.Info("Starting CoursesServer on port: ", address)
	// create a grpc CoursesServer
	grpcServer := grpc.NewServer()
	cpb.RegisterCoursesServiceServer(grpcServer, server)

	// serve the grpc CoursesServer
	if err := grpcServer.Serve(lis); err != nil {
		klog.Fatalf("Failed to serve: %v", err)
	}
}
