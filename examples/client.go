package main

import (
	"context"
	"os"
	"time"

	cpb "github.com/BetterGR/courses-microservice/protos"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"k8s.io/klog/v2"
)

func main() {
	// Ensure GRPC_PORT is set and well formed (e.g., "localhost:50051").
	addr := os.Getenv("GRPC_PORT")
	if addr == "" {
		klog.Fatalf("GRPC_PORT environment variable is empty; set it like 'localhost:50051'")
	}

	conn, err := grpc.Dial(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		klog.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()

	client := cpb.NewCoursesServiceClient(conn)

	// Test the courses server with all fields.
	courseID := createCourse(client) // includes Name, Description, Semester, token.
	getCourse(client, courseID)      // check all returned fields.
	updateCourse(client, courseID)   // update Name, Description, Semester.
	getCourse(client, courseID)      // verify update.
	addStudent(client, courseID)     // add a student.
	getStudents(client, courseID)    // check student list.
	addStaff(client, courseID)       // add a staff.
	getStaff(client, courseID)       // check staff list.
	removeStudent(client, courseID)  // remove the student.
	removeStaff(client, courseID)    // remove the staff.
	deleteCourse(client, courseID)   // finally delete the course.
}

// Test function to create a course.
func createCourse(client cpb.CoursesServiceClient) string {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	req := &cpb.CreateCourseRequest{
		Course: &cpb.Course{
			Id:          "course-123",
			Name:        "Algorithms",
			Description: "An introductory course on algorithms",
			Semester:    "Fall2023",
		},
		Token: "valid-token", // added token
	}
	resp, err := client.CreateCourse(ctx, req)
	if err != nil {
		klog.Fatalf("could not create course: %v", err)
	}
	klog.Infof("Course created: %v", resp.GetCourse())
	return resp.GetCourse().Id
}

// Test function to get a course.
func getCourse(client cpb.CoursesServiceClient, courseID string) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	req := &cpb.GetCourseRequest{
		Id:    courseID,
		Token: "valid-token", // added token
	}
	resp, err := client.GetCourse(ctx, req)
	if err != nil {
		klog.Fatalf("could not get course: %v", err)
	}
	klog.Infof("Course retrieved: %v", resp.GetCourse())
}

func updateCourse(client cpb.CoursesServiceClient, courseID string) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	req := &cpb.UpdateCourseRequest{
		Course: &cpb.Course{
			Id:          courseID,
			Name:        "Advanced Algorithms",
			Description: "A deeper dive into algorithms",
			Semester:    "Spring2024",
		},
		Token: "valid-token", // added token
	}
	resp, err := client.UpdateCourse(ctx, req)
	if err != nil {
		klog.Fatalf("could not update course: %v", err)
	}
	klog.Infof("Course updated: %v", resp.GetCourse())
}

func addStudent(client cpb.CoursesServiceClient, courseID string) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	req := &cpb.AddStudentRequest{
		CourseId:  courseID,
		StudentId: "student-123",
		Token:     "valid-token", // added token
	}
	_, err := client.AddStudentToCourse(ctx, req)
	if err != nil {
		klog.Fatalf("could not add student: %v", err)
	}
	klog.Info("Student added to course")
}

func getStudents(client cpb.CoursesServiceClient, courseID string) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	req := &cpb.GetStudentsRequest{
		Id:    courseID,
		Token: "valid-token", // added token
	}
	resp, err := client.GetStudents(ctx, req)
	if err != nil {
		klog.Fatalf("could not get students: %v", err)
	}
	klog.Infof("Students in course: %v", resp.GetStudentIds())
}

func addStaff(client cpb.CoursesServiceClient, courseID string) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	req := &cpb.AddStaffRequest{
		CourseId: courseID,
		StaffId:  "staff-456",
		Token:    "valid-token", // added token
	}
	_, err := client.AddStaffToCourse(ctx, req)
	if err != nil {
		klog.Fatalf("could not add staff: %v", err)
	}
	klog.Info("Staff added to course")
}

func getStaff(client cpb.CoursesServiceClient, courseID string) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	req := &cpb.GetStaffRequest{
		Id:    courseID,
		Token: "valid-token", // added token
	}
	resp, err := client.GetStaff(ctx, req)
	if err != nil {
		klog.Fatalf("could not get staff: %v", err)
	}
	klog.Infof("Staff in course: %v", resp.GetStaffIds())
}

func removeStudent(client cpb.CoursesServiceClient, courseID string) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	req := &cpb.RemoveStudentRequest{
		CourseId:  courseID,
		StudentId: "student-123",
		Token:     "valid-token", // added token
	}
	_, err := client.RemoveStudentFromCourse(ctx, req)
	if err != nil {
		klog.Fatalf("could not remove student: %v", err)
	}
	klog.Info("Student removed from course")
}

func removeStaff(client cpb.CoursesServiceClient, courseID string) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	req := &cpb.RemoveStaffRequest{
		CourseId: courseID,
		StaffId:  "staff-456",
		Token:    "valid-token", // added token
	}
	_, err := client.RemoveStaffFromCourse(ctx, req)
	if err != nil {
		klog.Fatalf("could not remove staff: %v", err)
	}
	klog.Info("Staff removed from course")
}

func deleteCourse(client cpb.CoursesServiceClient, courseID string) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	req := &cpb.DeleteCourseRequest{
		Id:    courseID,
		Token: "valid-token", // added token
	}
	_, err := client.DeleteCourse(ctx, req)
	if err != nil {
		klog.Fatalf("could not delete course: %v", err)
	}
	klog.Info("Course deleted")
}
