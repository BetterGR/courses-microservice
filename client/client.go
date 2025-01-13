package main

import (
	"context"
	"flag"
	"log"
	"time"

	pb "github.com/BetterGR/course-microservice/protos"
	pb "github.com/BetterGR/course-microservice/protos"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const serverAddress = "localhost:50052"

func main() {
	flag.Parse()

	// Connect to the gRPC server
	conn, err := grpc.Dial(serverAddress, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect to the server: %v", err)
	}
	defer conn.Close()

	client := pb.NewCourseServiceClient(conn)

	// Add the "Theory of Computation" course
	addCourse(client, "Theory of Computation", "Advanced course covering automata, Turing machines, and computational theory", "Spring 2025")
}

func addCourse(client pb.CourseServiceClient, name, description, semester string) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	req := &pb.CreateCourseRequest{
		Name:        name,
		Description: description,
		Semester:    semester,
	}

	res, err := client.CreateCourse(ctx, req)
	if err != nil {
		log.Fatalf("Failed to create course: %v", err)
	}

	log.Printf("Course created successfully with ID: %s", res.CourseId)
}
