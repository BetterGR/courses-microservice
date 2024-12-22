package main

import (
	"context"
	"errors"
	"flag"
	"time"

	pb "github.com/BetterGR/course-microservice/course_protobuf"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"k8s.io/klog/v2"
)

// Predefined static errors.
var (
	errCreateCourse = errors.New("failed to create course")
	errGetCourse    = errors.New("failed to get course")
)

const (
	// serverAddress defines the address of the gRPC server.
	serverAddress = "localhost:50052"
)

// main initializes the gRPC client and calls example RPC methods.
func main() {
	klog.InitFlags(nil) // Initialize klog.
	flag.Parse()
	defer klog.Flush()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	// Connect to gRPC server.
	//nolint:staticcheck // grpc.DialContext is supported in 1.x.
	conn, err := grpc.DialContext(ctx, serverAddress, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		klog.Fatalf("Failed to connect to server: %v", err)
	}
	defer conn.Close()

	client := pb.NewCourseServiceClient(conn)

	// Call example RPC methods.
	createCourse(ctx, client)
	getCourse(ctx, client)
}

// createCourse sends a request to create a new course.
func createCourse(ctx context.Context, client pb.CourseServiceClient) {
	logger := klog.FromContext(ctx)

	req := &pb.CreateCourseRequest{
		Name:        "Algorithms-1",
		Description: "Learn about algorithms",
		Semester:    "Spring-2025",
	}

	resp, err := client.CreateCourse(ctx, req)
	if err != nil {
		logger.Info("Error occurred while creating course", "error", errCreateCourse.Error(), "details", err.Error())
		return
	}

	// Log course creation.
	logger.Info("Course created successfully", "courseId", resp.GetCourseId())
}

// getCourse sends a request to retrieve a course by its ID.
func getCourse(ctx context.Context, client pb.CourseServiceClient) {
	logger := klog.FromContext(ctx)

	req := &pb.GetCourseRequest{CourseId: "C1"}

	resp, err := client.GetCourse(ctx, req)
	if err != nil {
		logger.Info("Error occurred while retrieving course", "error", errGetCourse.Error(), "details", err.Error())
		return
	}

	// Log course retrieval.
	logger.Info("Course retrieved successfully", "name", resp.GetName(), "semester", resp.GetSemester())
}
