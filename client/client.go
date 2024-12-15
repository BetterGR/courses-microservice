package main

import (
	"context"
	"log"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	pb "github.com/BetterGR/course-microservice/course_protobuf"
)

const (
	// serverAddress defines the address of the gRPC server
	serverAddress = "localhost:50052"
)

func main() {
	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	defer logger.Sync()

	conn, err := grpc.Dial(serverAddress, grpc.WithInsecure())
	if err != nil {
		logger.Fatal("Failed to connect to server", zap.Error(err))
	}
	defer conn.Close()

	client := pb.NewCourseServiceClient(conn)

	// Call example RPC methods
	createCourse(client, logger)
	getCourse(client, logger)
}

func createCourse(client pb.CourseServiceClient, logger *zap.Logger) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	req := &pb.CreateCourseRequest{
		Name:        "Algorithms 1",
		Description: "Learn about algorithms",
		Semester:    "Spring 2025",
	}

	resp, err := client.CreateCourse(ctx, req)
	if err != nil {
		logger.Error("Failed to create course", zap.Error(err))
		return
	}
	logger.Info("Created course", zap.String("courseId", resp.CourseId))
}

func getCourse(client pb.CourseServiceClient, logger *zap.Logger) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	req := &pb.GetCourseRequest{CourseId: "C1"}
	resp, err := client.GetCourse(ctx, req)
	if err != nil {
		logger.Error("Failed to get course", zap.Error(err))
		return
	}
	logger.Info("Retrieved course", zap.String("name", resp.Name), zap.String("semester", resp.Semester))
}
