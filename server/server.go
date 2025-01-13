package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"

	pb "github.com/BetterGR/course-microservice/protos"
	// ms "github.com/TekClinic/MicroService-Lib"
	"google.golang.org/grpc"
	"k8s.io/klog/v2"
)

const (
	// address defines the server's listening address.
	address  = "localhost:50052"
	protocol = "tcp"
)

// courseServer implements the CourseService with mslib integration.
type courseServer struct {
	// ms.BaseServiceServer
	pb.UnimplementedCourseServiceServer
}

// GetCourse retrieves a course by its ID.
func (s *courseServer) GetCourse(ctx context.Context, req *pb.GetCourseRequest) (*pb.GetCourseResponse, error) {
	logger := klog.FromContext(ctx)
	logger.V(5).Info("Received GetCourse request", "courseId", req.CourseId)

	course := new(Course)
	err := db.NewSelect().Model(course).Where("id = ?", req.CourseId).Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("course not found: %w", err)
	}

	return &pb.GetCourseResponse{
		CourseId:    fmt.Sprintf("%d", course.ID),
		Name:        course.Name,
		Description: course.Description,
		Semester:    course.Semester,
		StaffIds:    course.StaffIDs,
		StudentIds:  course.StudentIDs,
	}, nil
}

// CreateCourse creates a new course.
func (s *courseServer) CreateCourse(ctx context.Context, req *pb.CreateCourseRequest) (*pb.CreateCourseResponse, error) {
	logger := klog.FromContext(ctx)

	course := &Course{
		Name:        req.GetName(),
		Description: req.GetDescription(),
		Semester:    req.GetSemester(),
	}

	_, err := db.NewInsert().Model(course).Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create course: %w", err)
	}

	logger.V(5).Info("Created new course", "courseId", course.ID, "name", req.Name)
	return &pb.CreateCourseResponse{CourseId: fmt.Sprintf("%d", course.ID)}, nil
}

func main() {
	// Initialize klog for logging.
	klog.InitFlags(nil)
	flag.Parse()
	defer klog.Flush()

	// Connect to the PostgreSQL database.
	createDatabaseIfNotExists()
	ConnectDB()
	defer CloseDB()

	// Create the database schema if it doesn't exist.
	ctx := context.Background()
	if err := createSchemaIfNotExists(ctx); err != nil {
		log.Fatalf("Failed to initialize database schema: %v", err)
	}

	// Start the gRPC server.
	listener, err := net.Listen("tcp", address)
	if err != nil {
		klog.ErrorS(err, "Failed to listen")
		return
	}

	grpcServer := grpc.NewServer()
	pb.RegisterCourseServiceServer(grpcServer, &courseServer{})

	klog.InfoS("CourseService is running", "address", address)
	if err := grpcServer.Serve(listener); err != nil {
		klog.ErrorS(err, "Failed to serve")
	}
}
