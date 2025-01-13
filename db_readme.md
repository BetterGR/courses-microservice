# README: Integrating a PostgreSQL Database into a Microservice

This documentation explains the step-by-step process to integrate a PostgreSQL database into a microservice, including changes made to the `Makefile`, setting up PostgreSQL, and creating the necessary files.

---

## Prerequisites
1. **PostgreSQL Installed**: Ensure PostgreSQL is installed on your system.
2. **Go Installed**: Verify that Go is installed.
3. **Protobuf Compiler**: Install the Protocol Buffers compiler (`protoc`).
4. **gRPC and Bun Libraries**: Install Go dependencies.

---

## Step 1: Download and Set Up PostgreSQL

1. **Download PostgreSQL**:
   - Visit the [PostgreSQL official website](https://www.postgresql.org/download/) and download the installer for your OS.

2. **Install PostgreSQL**:
   - Follow the installation steps.
   - During setup, note the username, password (example `bettergr2425`), and port (default is `5432`).

3. **Start PostgreSQL Service**:
   - Open the `pgAdmin` tool or use the command-line service manager.
   - Ensure the PostgreSQL service is running.


---

## Step 2: Project Structure

Organize your project directory as follows:
```
course-microservice/
├── client/
├── protos/
├── server/
│   ├── db.go
│   ├── server.go
├── Dockerfile
├── go.mod
├── go.sum
├── LICENSE
├── Makefile
└── README.md
```

### File Details:
1. **`db.go`**:
   Contains all the database-related logic, such as connecting to PostgreSQL, defining schemas, and creating the database if it doesn't exist.
   

2. **`server.go`**:
   Implements the microservice functionality, integrating with the database through `db.go`.

---

## Step 3: Changes to `db.go`

```go
package main

import (
	"context"
	"database/sql"
	"log"
	"time"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
)

var db *bun.DB

// InitializeDatabase ensures that the database exists and initializes the schema.
func InitializeDatabase() {
	createDatabaseIfNotExists() // Create course_db if it does not exist.
	ConnectDB()                 // Connect to course_db.
}

// createDatabaseIfNotExists checks if the database exists and creates it if not.
func createDatabaseIfNotExists() {
	// Connect to the PostgreSQL server (not the course_db itself yet).
	dsn := "postgres://postgres:bettergr2425@localhost:5432/postgres?sslmode=disable"
	connector := pgdriver.NewConnector(pgdriver.WithDSN(dsn))
	sqldb := sql.OpenDB(connector)
	defer sqldb.Close()

	// Check if the database exists.
	ctx := context.Background()
	query := `
		SELECT 1 FROM pg_database WHERE datname = 'course_db';
	`
	var exists int
	err := sqldb.QueryRowContext(ctx, query).Scan(&exists)
	if err != nil && err != sql.ErrNoRows {
		log.Fatalf("Failed to check if database exists: %v", err)
	}

	// If the database does not exist, create it.
	if err == sql.ErrNoRows {
		_, err = sqldb.ExecContext(ctx, `CREATE DATABASE course_db;`)
		if err != nil {
			log.Fatalf("Failed to create database: %v", err)
		}
		log.Println("Database course_db created successfully.")
	} else {
		log.Println("Database course_db already exists.")
	}
}

// ConnectDB initializes the PostgreSQL database connection.
func ConnectDB() {
	dsn := "postgres://postgres:bettergr2425@localhost:5432/course_db?sslmode=disable"
	connector := pgdriver.NewConnector(pgdriver.WithDSN(dsn))
	sqldb := sql.OpenDB(connector)
	db = bun.NewDB(sqldb, pgdialect.New())

	// Test the connection.
	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to connect to the database: %v", err)
	}
	log.Println("Connected to PostgreSQL database.")
}

// CloseDB closes the PostgreSQL database connection.
func CloseDB() {
	if err := db.Close(); err != nil {
		log.Fatalf("Failed to close database: %v", err)
	}
}

// createSchemaIfNotExists creates the database schema if it doesn't exist.
func createSchemaIfNotExists(ctx context.Context) error {
	models := []interface{}{
		(*Course)(nil),
	}

	for _, model := range models {
		if _, err := db.NewCreateTable().IfNotExists().Model(model).Exec(ctx); err != nil {
			return err
		}
	}
	log.Println("Database schema initialized.")
	return nil
}

// Course represents the database schema for courses.
type Course struct {
	ID          int32     `bun:",pk,autoincrement"`
	Name        string    `bun:"name,notnull" validate:"required,min=1,max=100"`
	Description string    `bun:"description,notnull" validate:"required,min=1,max=500"`
	Semester    string    `bun:"semester,notnull" validate:"required,min=1,max=20"`
	StaffIDs    []string  `bun:",array"`
	StudentIDs  []string  `bun:",array"`
	CreatedAt   time.Time `bun:",nullzero,notnull,default:current_timestamp"`
	DeletedAt   time.Time `bun:",soft_delete,nullzero"`
}

```

---

## Step 4: Changes to `server.go`

```go
package main

import (
	"context"
	"flag"
	"log"
	"net"

	pb "github.com/BetterGR/course-microservice/protos"
	"google.golang.org/grpc"
	"k8s.io/klog/v2"
)

const (
	address = "localhost:50052"
)

// courseServer implements the CourseService with database integration.
type courseServer struct {
	pb.UnimplementedCourseServiceServer
}

func main() {
	klog.InitFlags(nil)
	flag.Parse()
	defer klog.Flush()

	// Connect to the database
	ConnectDB()
	defer CloseDB()

	// Initialize the database schema
	ctx := context.Background()
	if err := createSchemaIfNotExists(ctx); err != nil {
		log.Fatalf("Failed to initialize database schema: %v", err)
	}

	listener, err := net.Listen("tcp", address)
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	pb.RegisterCourseServiceServer(grpcServer, &courseServer{})

	log.Printf("Server is running on %s", address)
	if err := grpcServer.Serve(listener); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
```

---

## Step 5: Changes to `Makefile`

Update the `Makefile` to include the `db.go` and `server.go` files during build and run:

```makefile
# Build server
build: proto fmt vet lint
	@echo [BUILD] Building server binary...
	@go build -o server/server ./server/*.go

# Run the server
run: proto fmt vet lint
	@echo [RUN] Starting server...
	@go run ./server/*.go
```

---

## Step 6: Running the Microservice

1. **Run `make run`**:
   ```bash
   make run
   ```

2. **Server Logs**:
   - Check logs to ensure the database is connected and the schema is initialized.

---

## Step 7: Testing the Microservice

1. Use a client or gRPC testing tool to test the `CreateCourse` and `GetCourse` methods.
2. Check the `course_db` database using `pgAdmin` or a SQL client to ensure data is inserted and retrieved correctly.

---

This concludes the integration of PostgreSQL with your microservice. If you have further questions or issues, feel free to reach out!

