# Courses Microservice
This repository provides a gRPC service for managing courses records. The service is implemented in Go and uses Protobuf to define message types and service methods.

## **Setup**

**1. Setup Prerequisites**

Make sure you have the following installed:

- Go (latest stable version recommended)
- Protobuf Compiler (protoc)
- Docker (if using Docker deployment)
- PostgreSQL database (ensure PostgreSQL is installed and running.)

**2. Clone the Repository**

    git clone https://github.com/BetterGR/courses-microservice.git

**3. Set Up Environment Variables**

Create a .env file in the root directory with the following environment variables:

    GRPC_PORT=localhost:50054  
    AUTH_ISSUER=http://auth.BetterGR.org  
    DSN=postgres://postgres:bettergr2425@localhost:5432/bettergr?sslmode=disable  
    DB_NAME=bettergr  

**4. Configure MicroService Library**

This repository depends on the TekClinic/MicroService-Lib library for authentication and environment variable management. Proper configuration of the required environment variables from TekClinic/MicroService-Lib is essential. Refer to its documentation for proper setup.

**4. Start the gRPC Server**

To start the server, open the terminal in the courses-microservice directory and run the following:

    go mod init github.com/BetterGR/courses-microservice
    make run

**5. Testing**

To run unit tests:

    make tests

**6. Makefile Help**

For more available commands and their descriptions, run: `make help`.

## **License**

This project is licensed under the Apache 2.0 License. See the LICENSE file for more details.