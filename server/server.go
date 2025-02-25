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
	logger.V(logLevelDebug).Info("Received GetCourse request", "courseId", req.GetId())

	course, err := s.db.GetCourse(ctx, req.GetId())
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "course not found: %v", err)
	}

	return &cpb.GetCourseResponse{Course: &cpb.Course{
		Id:          course.GetId(),
		Name:        course.GetName(),
		Description: course.GetDescription(),
		Semester:    course.GetSemester(),
		StaffIds:    course.GetStaffIds(),
		StudentsIds: course.GetStudentsIds(),
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
	logger.V(logLevelDebug).Info("Received CreateCourse request", "courseName", req.GetCourse().GetName())

	course := &cpb.Course{
		Id:             req.GetCourse().GetId(),
		Name:           req.GetCourse().GetName(),
		Semester:       req.GetCourse().GetSemester(),
		Description:    req.GetCourse().GetDescription(),
		StaffIds:       []string{},
		StudentsIds:    []string{},
		CourseMaterial: "",
	}
	if err := s.db.AddCourse(ctx, course); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to add course: %v", err)
	}

	return &cpb.CreateCourseResponse{Course: course}, nil
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
	logger.V(logLevelDebug).Info("Received UpdateCourse request", "courseId", req.GetCourse().GetId())

	course, err := s.db.GetCourse(ctx, req.GetCourse().GetId())
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "course not found: %v", err)
	}

	course.Name = req.GetCourse().GetName()
	course.Semester = req.GetCourse().GetSemester()
	course.Description = req.GetCourse().GetDescription()

	if err = s.db.UpdateCourse(ctx, course); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to update course: %v", err)
	}

	return &cpb.UpdateCourseResponse{Course: course}, nil
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
	logger.V(logLevelDebug).Info("Received DeleteCourse request", "courseId", req.GetId())

	if err := s.db.DeleteCourse(ctx, req.GetId()); err != nil {
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

// GetStudents retrieves the students enrolled in a course.
func (s *CoursesServer) GetStudents(
	ctx context.Context,
	req *cpb.GetStudentsRequest,
) (*cpb.GetStudentsResponse, error) {
	if _, err := s.VerifyToken(ctx, req.GetToken()); err != nil {
		return nil, fmt.Errorf("authentication failed: %w",
			status.Error(codes.Unauthenticated, err.Error()))
	}

	logger := klog.FromContext(ctx)
	logger.V(logLevelDebug).Info("Received GetStudents request", "courseId", req.GetId())

	course, err := s.db.GetCourse(ctx, req.GetId())
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "course not found: %v", err)
	}

	return &cpb.GetStudentsResponse{StudentIds: course.GetStudentsIds()}, nil
}

// GetStaff retrieves the staff members assigned to a course.
func (s *CoursesServer) GetStaff(ctx context.Context, req *cpb.GetStaffRequest) (*cpb.GetStaffResponse, error) {
	if _, err := s.VerifyToken(ctx, req.GetToken()); err != nil {
		return nil, fmt.Errorf("authentication failed: %w",
			status.Error(codes.Unauthenticated, err.Error()))
	}

	logger := klog.FromContext(ctx)
	logger.V(logLevelDebug).Info("Received GetStaff request", "courseId", req.GetId())

	course, err := s.db.GetCourse(ctx, req.GetId())
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "course not found: %v", err)
	}

	return &cpb.GetStaffResponse{StaffIds: course.GetStaffIds()}, nil
}

// UploadCourseMaterial handles updating the course material.
func (s *CoursesServer) UploadCourseMaterial(ctx context.Context,
	req *cpb.UploadCourseMaterialRequest,
) (*cpb.UploadCourseMaterialResponse, error) {
	if _, err := s.VerifyToken(ctx, req.GetToken()); err != nil {
		return nil, fmt.Errorf("authentication failed: %w",
			status.Error(codes.Unauthenticated, err.Error()))
	}

	logger := klog.FromContext(ctx)
	logger.V(logLevelDebug).Info("Received UploadCourseMaterial request", "courseId", req.GetCourseId())

	if err := s.db.UpdateCourseMaterial(ctx, req.GetCourseId(), req.GetCourseMaterial()); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to update course material: %v", err)
	}

	course, err := s.db.GetCourse(ctx, req.GetCourseId())
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "course not found: %v", err)
	}

	return &cpb.UploadCourseMaterialResponse{Course: &cpb.Course{
		Id:             course.GetId(),
		Name:           course.GetName(),
		Description:    course.GetDescription(),
		Semester:       course.GetSemester(),
		StaffIds:       course.GetStaffIds(),
		StudentsIds:    course.GetStudentsIds(),
		CourseMaterial: course.GetCourseMaterial(),
	}}, nil
}

// GetCourseMaterial handles retrieving the course material.
func (s *CoursesServer) GetCourseMaterial(ctx context.Context,
	req *cpb.GetCourseMaterialRequest,
) (*cpb.GetCourseMaterialResponse, error) {
	if _, err := s.VerifyToken(ctx, req.GetToken()); err != nil {
		return nil, fmt.Errorf("authentication failed: %w",
			status.Error(codes.Unauthenticated, err.Error()))
	}

	logger := klog.FromContext(ctx)
	logger.V(logLevelDebug).Info("Received GetCourseMaterial request", "courseId", req.GetCourseId())

	material, err := s.db.GetCourseMaterial(ctx, req.GetCourseId())
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "course not found: %v", err)
	}

	return &cpb.GetCourseMaterialResponse{CourseMaterial: material}, nil
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
