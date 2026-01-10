# AGENTS.md

## Project Overview

This project is the back-end service for a WeChat mini-program called "GoJxust", designed to provide campus services for Jiangxi University of Science and Technology (JXUST). The application is written in Go and utilizes the Gin web framework. It offers features such-as course selection assistance, course schedules, viewing exam failure rates, and more. The project follows a standard layered architecture, separating concerns into handlers, services, and models.

**Key Technologies:**

*   **Language:** Go
*   **Web Framework:** Gin
*   **Database:** MySQL
*   **ORM:** GORM
*   **Cache:** Redis
*   **Authentication:** JWT
*   **Configuration:** Environment variables
*   **Object Storage:** MinIO

## Building and Running

The project uses a `Makefile` to streamline common development tasks.

**Build the application:**

```shell
make build-apiserver
```

This command builds a Linux binary and places it in the `./bin` directory.

**Run the application:**

The `README.md` suggests running the application directly using `go run`:

```shell
go run cmd/apiserver/main.go
```

Before running, you need to set up the configuration in a `.env` file. You can copy the example file:

```shell
cp .env.example .env
```

And then edit the `.env` file with your database, JWT, and other settings.

**Run unit tests:**

```shell
make test
```

**Run E2E tests:**

E2E tests are written in Python using httpx. Before running, ensure:
1. The API server is running in non-release mode (to enable mock login endpoint)
2. RBAC roles and permissions are initialized in the database

```shell
# Initialize RBAC data (first time only)
mysql -u your_username -p your_database < scripts/init_rbac.sql

# Install dependencies
pip install httpx

# Run E2E tests (server must be running on localhost:8080)
python scripts/e2e_test.py

# Or specify a custom base URL
python scripts/e2e_test.py --base-url http://localhost:8085
```

The E2E test script (`scripts/e2e_test.py`) covers:
- Public endpoints: health check, reviews, config, heroes, notifications, categories
- Authenticated endpoints: user profile, reviews CRUD, course table, fail rate, points, contributions, countdowns, study tasks
- Admin endpoints: reviews management, notifications management, heroes management, config management

**Supported test user types for mock login:**
- `basic`: Basic user with standard permissions
- `active`: Active user with additional permissions
- `verified`: Verified user
- `operator`: Operator with content management permissions
- `admin`: Administrator with full permissions

**Build a Docker image:**

```shell
make docker-build
```

## Development Conventions

*   **Architecture:** The project follows a layered architecture:
    *   `internal/handlers`: Contains the HTTP handlers that receive requests and send responses.
    *   `internal/services`: Contains the business logic.
    *   `internal/models`: Defines the data structures and interacts with the database.
    *   `internal/router`: Defines the API routes and wires up the handlers.
    *   `internal/dto`: Defines data transfer objects for requests and responses.
*   **API Versioning:** The API is versioned under the `/api/v0` path.
*   **Authentication:** JWT-based authentication is used for protected routes. The authentication middleware is in `internal/middleware/middleware.go`.
*   **Database Migrations:** The application uses GORM's auto-migration feature to keep the database schema up-to-date.
*   **Dependency Management:** Go modules are used for dependency management. Dependencies are listed in the `go.mod` file.
*   **Configuration:** Configuration is managed through environment variables, loaded from a `.env` file.
*   **Code Style:** The project follows standard Go formatting. Use `gofmt` to format your code.
