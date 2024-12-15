package main

import (
	"context"
	"fmt"
	"log"
	"net"

	"k8s.io/klog/v2"
	"google.golang.org/grpc"
	pb "github.com/BetterGR/course-microservice/course_protobuf"
)

const (
	// address defines the server's listening address.
	address = "localhost:50052"
)

// courseServer implements the CourseService.
type courseServer struct {
	pb.UnimplementedCourseServiceServer
}

// courses is an in-memory data storage for demonstration.
var courses = map[string]*pb.GetCourseResponse{}

// GetCourse retrieves a course by its ID.
func (s *courseServer) GetCourse(ctx context.Context, req *pb.GetCourseRequest) (*pb.GetCourseResponse, error) {
	klog.Infof("Received GetCourse request: courseId=%s", req.CourseId)
	course, exists := courses[req.CourseId]
	if !exists {
		return nil, fmt.Errorf("course not found: %s", req.CourseId)
	}
	return course, nil
}

// CreateCourse creates a new course.
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
	klog.Infof("Created new course: courseId=%s, name=%s", courseID, req.Name)
	return &pb.CreateCourseResponse{CourseId: courseID}, nil
}

// main initializes and starts the CourseService server.
func main() {
	klog.InitFlags(nil)
	defer klog.Flush()

	listener, err := net.Listen("tcp", address)
	if err != nil {
		klog.Fatalf("Failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	pb.RegisterCourseServiceServer(grpcServer, &courseServer{})

	klog.Infof("CourseService is running on address: %s", address)
	if err := grpcServer.Serve(listener); err != nil {
		klog.Fatalf("Failed to serve: %v", err)
	}
}
