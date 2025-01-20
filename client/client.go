package main

import (
	"context"
	"log"
	"time"

	pb "github.com/BetterGR/course-microservice/protos"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	serverAddress = "localhost:50052"
	authToken     = "test-token" // Replace with actual token if required
)

func main() {
	// Connect to the gRPC server
	conn, err := grpc.NewClient(serverAddress, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	client := pb.NewCourseServiceClient(conn)

	// Test RPCs one by one
	////testCreateCourse(client)
	////testGetCourse(client)
	//testUpdateCourse(client)
	////testAddStudentToCourse(client)
	//testRemoveStudentFromCourse(client)
	//testAddAnnouncement(client)
	//testListAnnouncements(client)
	// testRemoveAnnouncement(client)
	// testDeleteCourse(client)
}

func testCreateCourse(client pb.CourseServiceClient) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	log.Println("Testing CreateCourse...")
	resp, err := client.CreateCourse(ctx, &pb.CreateCourseRequest{
		CourseId:    "236343",
		Name:        "Theory of Computation",
		Description: "Advanced course on computation theory",
		Semester:    "Spring 2025",
		Token:       authToken,
	})
	if err != nil {
		log.Fatalf("CreateCourse failed: %v", err)
	}
	log.Printf("CreateCourse success: Course ID = %s", resp.GetCourseId())
}

func testGetCourse(client pb.CourseServiceClient) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	log.Println("Testing GetCourse...")
	resp, err := client.GetCourse(ctx, &pb.GetCourseRequest{
		CourseId: "236343",
		Token:    authToken,
	})
	if err != nil {
		log.Fatalf("GetCourse failed: %v", err)
	}
	log.Printf("GetCourse success: %+v", resp)
}

func testUpdateCourse(client pb.CourseServiceClient) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	log.Println("Testing UpdateCourse...")
	resp, err := client.UpdateCourse(ctx, &pb.UpdateCourseRequest{
		CourseId:    "236343",
		Name:        "Updated Theory of Computation",
		Description: "Updated course on computation theory",
		Semester:    "Fall 2025",
		Token:       authToken,
	})
	if err != nil {
		log.Fatalf("UpdateCourse failed: %v", err)
	}
	log.Printf("UpdateCourse success: %v", resp.GetSuccess())
}

func testAddStudentToCourse(client pb.CourseServiceClient) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	log.Println("Testing AddStudentToCourse...")
	resp, err := client.AddStudentToCourse(ctx, &pb.AddStudentRequest{
		CourseId:  "236343",
		StudentId: "323910828",
		Token:     authToken,
	})
	if err != nil {
		log.Fatalf("AddStudentToCourse failed: %v", err)
	}
	log.Printf("AddStudentToCourse success: %v", resp.GetSuccess())
}

func testRemoveStudentFromCourse(client pb.CourseServiceClient) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	log.Println("Testing RemoveStudentFromCourse...")
	resp, err := client.RemoveStudentFromCourse(ctx, &pb.RemoveStudentRequest{
		CourseId:  "236343",
		StudentId: "323910828",
		Token:     authToken,
	})
	if err != nil {
		log.Fatalf("RemoveStudentFromCourse failed: %v", err)
	}
	log.Printf("RemoveStudentFromCourse success: %v", resp.GetSuccess())
}

func testAddAnnouncement(client pb.CourseServiceClient) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	log.Println("Testing AddAnnouncement...")
	resp, err := client.AddAnnouncement(ctx, &pb.AddAnnouncementRequest{
		CourseId: "236343",
		Title:    "Midterm Exam Announcement",
		Content:  "The midterm exam will take place on May 10th, 2025.",
	})
	if err != nil {
		log.Fatalf("AddAnnouncement failed: %v", err)
	}
	log.Printf("AddAnnouncement success: %v", resp.GetSuccess())
}

func testListAnnouncements(client pb.CourseServiceClient) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	log.Println("Testing ListAnnouncements...")
	resp, err := client.ListAnnouncements(ctx, &pb.ListAnnouncementsRequest{
		CourseId: "236343",
	})
	if err != nil {
		log.Fatalf("ListAnnouncements failed: %v", err)
	}
	log.Printf("ListAnnouncements success: %v", resp.GetAnnouncements())
}

func testRemoveAnnouncement(client pb.CourseServiceClient) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	log.Println("Testing RemoveAnnouncement...")
	resp, err := client.RemoveAnnouncement(ctx, &pb.RemoveAnnouncementRequest{
		CourseId:     "236343",
		Announcement: "Midterm Exam Announcement",
	})
	if err != nil {
		log.Fatalf("RemoveAnnouncement failed: %v", err)
	}
	log.Printf("RemoveAnnouncement success: %v", resp.GetSuccess())
}

func testDeleteCourse(client pb.CourseServiceClient) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	log.Println("Testing DeleteCourse...")
	resp, err := client.DeleteCourse(ctx, &pb.DeleteCourseRequest{
		CourseId: "236343",
		Token:    authToken,
	})
	if err != nil {
		log.Fatalf("DeleteCourse failed: %v", err)
	}
	log.Printf("DeleteCourse success: %v", resp.GetSuccess())
}
