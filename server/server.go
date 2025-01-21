package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"

	pb "github.com/BetterGR/course-microservice/protos"
	"google.golang.org/grpc"
	"k8s.io/klog/v2"
)

const (
	// address defines the server's listening address.
	address  = "localhost:50054"
	protocol = "tcp"
)

// courseServer implements the CourseService with mslib integration.
type courseServer struct {
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
		CourseId:      course.ID,
		Name:          course.Name,
		Description:   course.Description,
		Semester:      course.Semester,
		StaffIds:      course.StaffIDs,
		StudentIds:    course.StudentIDs,
		Announcements: course.Announcements, // Include announcements in the response
	}, nil
}

// CreateCourse creates a new course.
func (s *courseServer) CreateCourse(ctx context.Context, req *pb.CreateCourseRequest) (*pb.CreateCourseResponse, error) {
	logger := klog.FromContext(ctx)

	course := &Course{
		ID:          req.GetCourseId(),
		Name:        req.GetName(),
		Description: req.GetDescription(),
		Semester:    req.GetSemester(),
	}

	_, err := db.NewInsert().Model(course).Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create course: %w", err)
	}

	logger.V(5).Info("Created new course", "courseId", course.ID, "name", req.Name)
	return &pb.CreateCourseResponse{CourseId: course.ID}, nil
}

// UpdateCourse updates the details of an existing course.
func (s *courseServer) UpdateCourse(ctx context.Context, req *pb.UpdateCourseRequest) (*pb.UpdateCourseResponse, error) {
	logger := klog.FromContext(ctx)
	logger.V(5).Info("Received UpdateCourse request", "courseId", req.CourseId)

	course := new(Course)
	err := db.NewSelect().Model(course).Where("id = ?", req.CourseId).Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("course not found: %w", err)
	}

	course.Name = req.GetName()
	course.Description = req.GetDescription()
	course.Semester = req.GetSemester()

	_, err = db.NewUpdate().Model(course).WherePK().Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to update course: %w", err)
	}

	logger.V(5).Info("Updated course", "courseId", course.ID)
	return &pb.UpdateCourseResponse{Success: true}, nil
}

// AddStudentToCourse adds a student to a course.
func (s *courseServer) AddStudentToCourse(ctx context.Context, req *pb.AddStudentRequest) (*pb.AddStudentResponse, error) {
	logger := klog.FromContext(ctx)
	logger.V(5).Info("Received AddStudentToCourse request", "courseId", req.CourseId, "studentId", req.StudentId)

	// Fetch the course
	course := new(Course)
	err := db.NewSelect().Model(course).Where("id = ?", req.CourseId).Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("course not found: %w", err)
	}

	// Use raw SQL to append the student ID to the PostgreSQL array
	_, err = db.NewRaw(`
		UPDATE courses
		SET student_i_ds = array_append(student_i_ds, ?)
		WHERE id = ?`, req.StudentId, req.CourseId).Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to update student list: %w", err)
	}

	logger.V(5).Info("Successfully added student to course", "courseId", req.CourseId, "studentId", req.StudentId)
	return &pb.AddStudentResponse{Success: true}, nil
}

// RemoveStudentFromCourse removes a student from a course.
func (s *courseServer) RemoveStudentFromCourse(ctx context.Context, req *pb.RemoveStudentRequest) (*pb.RemoveStudentResponse, error) {
	logger := klog.FromContext(ctx)
	logger.V(5).Info("Received RemoveStudentFromCourse request", "courseId", req.CourseId, "studentId", req.StudentId)

	// Check if the course exists
	course := new(Course)
	err := db.NewSelect().Model(course).Where("id = ?", req.CourseId).Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("course not found: %w", err)
	}

	// Use raw SQL to remove the student ID from the PostgreSQL array
	_, err = db.NewRaw(`
		UPDATE courses
		SET student_i_ds = array_remove(student_i_ds, ?)
		WHERE id = ?`, req.StudentId, req.CourseId).Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to remove student from course: %w", err)
	}

	logger.V(5).Info("Successfully removed student from course", "courseId", req.CourseId, "studentId", req.StudentId)
	return &pb.RemoveStudentResponse{Success: true}, nil
}

// AddAnnouncement adds a new announcement to a course.
func (s *courseServer) AddAnnouncement(ctx context.Context, req *pb.AddAnnouncementRequest) (*pb.AddAnnouncementResponse, error) {
	logger := klog.FromContext(ctx)
	logger.V(5).Info("Received AddAnnouncement request", "courseId", req.CourseId, "title", req.Title)

	// Check if the course exists
	course := new(Course)
	err := db.NewSelect().Model(course).Where("id = ?", req.CourseId).Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("course not found: %w", err)
	}

	// Use raw SQL to append the announcement to PostgreSQL array
	_, err = db.NewRaw(`
		UPDATE courses
		SET announcements = array_append(announcements, ?)
		WHERE id = ?`, fmt.Sprintf("%s: %s", req.Title, req.Content), req.CourseId).Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to add announcement: %w", err)
	}

	logger.V(5).Info("Successfully added announcement", "courseId", req.CourseId, "title", req.Title)
	return &pb.AddAnnouncementResponse{Success: true}, nil
}

// ListAnnouncements lists all announcements for a course.
func (s *courseServer) ListAnnouncements(ctx context.Context, req *pb.ListAnnouncementsRequest) (*pb.ListAnnouncementsResponse, error) {
	logger := klog.FromContext(ctx)
	logger.V(5).Info("Received ListAnnouncements request", "courseId", req.CourseId)

	course := new(Course)
	err := db.NewSelect().Model(course).Where("id = ?", req.CourseId).Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("course not found: %w", err)
	}

	return &pb.ListAnnouncementsResponse{
		Announcements: course.Announcements,
	}, nil
}

// RemoveAnnouncement removes an announcement from a course.
func (s *courseServer) RemoveAnnouncement(ctx context.Context, req *pb.RemoveAnnouncementRequest) (*pb.RemoveAnnouncementResponse, error) {
	logger := klog.FromContext(ctx)
	logger.V(5).Info("Received RemoveAnnouncement request", "courseId", req.CourseId, "announcement", req.Announcement)

	// Check if the course exists
	course := new(Course)
	err := db.NewSelect().Model(course).Where("id = ?", req.CourseId).Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("course not found: %w", err)
	}

	// Use raw SQL to remove the announcement from PostgreSQL array
	_, err = db.NewRaw(`
		UPDATE courses
		SET announcements = array_remove(announcements, ?)
		WHERE id = ?`, req.Announcement, req.CourseId).Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to remove announcement: %w", err)
	}

	logger.V(5).Info("Successfully removed announcement", "courseId", req.CourseId, "announcement", req.Announcement)
	return &pb.RemoveAnnouncementResponse{Success: true}, nil
}

// GetAnnouncement provides a hardcoded announcement for a course according to id.
func (s *courseServer) GetAnnouncement(ctx context.Context, req *pb.GetAnnouncementRequest) (*pb.GetAnnouncementResponse, error) {
	logger := klog.FromContext(ctx)
	logger.V(5).Info("Received GetAnnouncement request", "courseId", req.CourseId)

	// Hardcoded announcements for specific courses
	hardcodedAnnouncements := map[string]string{
		"236781": "Midterm exam on April 10th. Don't forget to review chapters 1-5.",
		"234311": "Final project deadline is June 1st. Submit via the course portal.",
	}

	announcement, exists := hardcodedAnnouncements[req.CourseId]
	if !exists {
		return nil, fmt.Errorf("no announcement found for course ID: %s", req.CourseId)
	}

	return &pb.GetAnnouncementResponse{
		CourseId:     req.CourseId,
		Announcement: announcement,
	}, nil
}

func (s *courseServer) GetHomework(ctx context.Context, req *pb.GetHomeworkRequest) (*pb.GetHomeworkResponse, error) {
	logger := klog.FromContext(ctx)
	logger.V(5).Info("Received GetHomework request", "courseId", req.CourseId, "homeworkId", req.HomeworkId)

	// Hardcoded homework data based on course_id and homework_id
	hardcodedHomeworks := map[string]map[string]*pb.GetHomeworkResponse{
		"236781": {
			"1": {
				CourseId:    "236781",
				HomeworkId:  "1",
				Title:       "Assignment 1",
				Description: "Complete the first assignment covering chapters 1-3",
				DueDate:     "2025-03-15",
			},
			"2": {
				CourseId:    "236781",
				HomeworkId:  "2",
				Title:       "Assignment 2",
				Description: "Write a report on theoretical aspects",
				DueDate:     "2025-04-10",
			},
		},
		"234311": {
			"1": {
				CourseId:    "234311",
				HomeworkId:  "1",
				Title:       "Final Project",
				Description: "Develop a final project using course concepts",
				DueDate:     "2025-06-01",
			},
		},
	}

	// Check if the course and homework exist
	if courseHomeworks, ok := hardcodedHomeworks[req.CourseId]; ok {
		if homework, ok := courseHomeworks[req.HomeworkId]; ok {
			logger.V(5).Info("Homework found", "courseId", req.CourseId, "homeworkId", req.HomeworkId)
			return homework, nil
		}
	}

	logger.V(5).Info("Homework not found", "courseId", req.CourseId, "homeworkId", req.HomeworkId)
	return nil, fmt.Errorf("homework with id %s not found in course %s", req.HomeworkId, req.CourseId)
}

// DeleteCourse deletes a course by its ID.
func (s *courseServer) DeleteCourse(ctx context.Context, req *pb.DeleteCourseRequest) (*pb.DeleteCourseResponse, error) {
	logger := klog.FromContext(ctx)
	logger.V(5).Info("Received DeleteCourse request", "courseId", req.CourseId)

	// Create a new instance of Course and specify it in the deletion query
	course := &Course{}
	res, err := db.NewDelete().Model(course).Where("id = ?", req.CourseId).Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to delete course: %w", err)
	}

	// Check the number of rows affected
	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return nil, fmt.Errorf("course not found or already deleted")
	}

	logger.V(5).Info("Deleted course", "courseId", req.CourseId)
	return &pb.DeleteCourseResponse{Success: true}, nil
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
