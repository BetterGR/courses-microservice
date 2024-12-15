package main

import (
	"context"
	"fmt"
	"log"
	"net"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	pb "github.com/BetterGR/course-microservice/course_protobuf"
)

const (
	// address defines the server's listening address
	address = "localhost:50052"
)

// courseServer implements the CourseService
type courseServer struct {
	pb.UnimplementedCourseServiceServer
	logger *zap.Logger // Structured logger for better debugging
}

// NewCourseServer creates a new courseServer with logging
func NewCourseServer(logger *zap.Logger) *courseServer {
	return &courseServer{logger: logger}
}

// In-memory data storage for demonstration
var courses = map[string]*pb.GetCourseResponse{}

// GetCourse retrieves a course by its ID
func (s *courseServer) GetCourse(ctx context.Context, req *pb.GetCourseRequest) (*pb.GetCourseResponse, error) {
	s.logger.Info("Received GetCourse request", zap.String("courseId", req.CourseId))
	course, exists := courses[req.CourseId]
	if !exists {
		return nil, fmt.Errorf("course not found: %s", req.CourseId)
	}
	return course, nil
}

// CreateCourse creates a new course
func (s *courseServer) CreateCourse(ctx context.Context, req *pb.CreateCourseRequest) (*pb.CreateCourseResponse, error) {
	courseID := fmt.Sprintf("C%d", len(courses)+1)
	courses[courseID] = &pb.GetCourseResponse{
		CourseId:    courseID,
		Name:        req.Name,
		Description: req.Description,
		Semester:    req.Semester,
		StaffIds:    []string{},
		StudentIds:  []string{},
	}
	s.logger.Info("Created new course", zap.String("courseId", courseID), zap.String("name", req.Name))
	return &pb.CreateCourseResponse{CourseId: courseID}, nil
}

// Other RPC implementations go here...
// AddStudentToCourse, RemoveStudentFromCourse, ListStudents, etc.

func main() {
	// Initialize logger
	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	defer logger.Sync()

	// Start server
	listener, err := net.Listen("tcp", address)
	if err != nil {
		logger.Fatal("Failed to listen", zap.Error(err))
	}

	grpcServer := grpc.NewServer()
	pb.RegisterCourseServiceServer(grpcServer, NewCourseServer(logger))

	logger.Info("CourseService is running", zap.String("address", address))
	if err := grpcServer.Serve(listener); err != nil {
		logger.Fatal("Failed to serve", zap.Error(err))
	}
}
