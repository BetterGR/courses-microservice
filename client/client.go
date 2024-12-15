package main

import (
	"context"
	"flag"
	"time"

	"k8s.io/klog/v2"
	"google.golang.org/grpc"
	pb "github.com/BetterGR/course-microservice/course_protobuf"
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

	conn, err := grpc.Dial(serverAddress, grpc.WithInsecure())
	if err != nil {
		klog.Fatalf("Failed to connect to server: %v", err)
	}
	defer conn.Close()

	client := pb.NewCourseServiceClient(conn)

	createCourse(client) // Call createCourse to create a course.
	getCourse(client)    // Call getCourse to retrieve a course.
}

// createCourse sends a request to create a new course.
func createCourse(client pb.CourseServiceClient) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	logger := klog.FromContext(ctx)
	req := &pb.CreateCourseRequest{
		Name:        "Algorithms-1",
		Description: "Learn about algorithms",
		Semester:    "Spring-2025",
	}

	resp, err := client.CreateCourse(ctx, req)
	if err != nil {
		logger.ErrorS(err, "Failed to create course")
		return
	}
	logger.InfoS("Created course", "courseId", resp.CourseId)
}

// getCourse sends a request to retrieve a course by its ID.
func getCourse(client pb.CourseServiceClient) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	logger := klog.FromContext(ctx)
	req := &pb.GetCourseRequest{CourseId: "C1"}
	resp, err := client.GetCourse(ctx, req)
	if err != nil {
		logger.ErrorS(err, "Failed to get course")
		return
	}
	logger.InfoS("Retrieved course", "name", resp.Name, "semester", resp.Semester)
}
