package main

import (
	"context"
	"time"

	"k8s.io/klog/v2"
	"google.golang.org/grpc"
	pb "github.com/BetterGR/course-microservice/course_protobuf"
)

const (
	// serverAddress defines the address of the gRPC server
	serverAddress = "localhost:50052"
)

func main() {
	// Initialize klog
	klog.InitFlags(nil)
	defer klog.Flush()

	conn, err := grpc.Dial(serverAddress, grpc.WithInsecure())
	if err != nil {
		klog.Fatalf("Failed to connect to server: %v", err)
	}
	defer conn.Close()

	client := pb.NewCourseServiceClient(conn)

	// Call example RPC methods
	createCourse(client)
	getCourse(client)
}

func createCourse(client pb.CourseServiceClient) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	req := &pb.CreateCourseRequest{
		Name:        "Algorithms 1",
		Description: "Learn about algorithms",
		Semester:    "Spring 2025",
	}

	resp, err := client.CreateCourse(ctx, req)
	if err != nil {
		klog.Errorf("Failed to create course: %v", err)
		return
	}
	klog.Infof("Created course: courseId=%s", resp.CourseId)
}

func getCourse(client pb.CourseServiceClient) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	req := &pb.GetCourseRequest{CourseId: "C1"}
	resp, err := client.GetCourse(ctx, req)
	if err != nil {
		klog.Errorf("Failed to get course: %v", err)
		return
	}
	klog.Infof("Retrieved course: name=%s, semester=%s", resp.Name, resp.Semester)
}
