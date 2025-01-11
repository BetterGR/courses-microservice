package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"time"

	pb "github.com/BetterGR/course-microservice/protos"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	serverAddress = "localhost:50052"
)

func main() {
	flag.Parse()

	// Connect to the gRPC server
	//nolint:staticcheck
	conn, err := grpc.Dial(serverAddress, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect to server: %v", err)
	}
	defer conn.Close()

	client := pb.NewCourseServiceClient(conn)

	// Create a course
	createCourse(client)

	// Get the created course
	getCourse(client, "C1")
}

func createCourse(client pb.CourseServiceClient) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	req := &pb.CreateCourseRequest{
		Name:        "Algorithms-1",
		Description: "An introduction to algorithms",
		Semester:    "Spring-2025",
	}

	resp, err := client.CreateCourse(ctx, req)
	if err != nil {
		log.Fatalf("Failed to create course: %v", err)
	}

	fmt.Printf("Created course with ID: %s\n", resp.GetCourseId())
}

func getCourse(client pb.CourseServiceClient, courseId string) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	req := &pb.GetCourseRequest{
		CourseId: courseId,
	}

	resp, err := client.GetCourse(ctx, req)
	if err != nil {
		log.Fatalf("Failed to get course: %v", err)
	}

	fmt.Printf("Retrieved course:\n")
	fmt.Printf("ID: %s\n", resp.GetCourseId())
	fmt.Printf("Name: %s\n", resp.GetName())
	fmt.Printf("Description: %s\n", resp.GetDescription())
	fmt.Printf("Semester: %s\n", resp.GetSemester())
}
