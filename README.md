# ConvertPDFGo API Documentation

## Introduction
This is the API documentation for **ConvertPDFGo**, a service designed for handling PDF manipulation such as merging, splitting, compressing, converting, and more. This API provides a variety of endpoints to perform actions on PDF files and manage user authentication and authorization.

## Authentication API
- **Signup** (`POST /auth/signup`): Register a new user.
- **Login** (`POST /auth/login`): User login with email and password.
- **Logout** (`POST /auth/logout`): User logout, requiring authentication.
- **Change Password** (`POST /auth/change-password`): Change user password, requiring authentication.
- **Google OAuth** (`POST /auth/google`): Google OAuth login or registration.
- **GitHub OAuth** (`POST /auth/github`): GitHub OAuth login or registration.
- **Facebook OAuth** (`POST /auth/facebook`): Facebook OAuth login or registration.
- **Password Reset Request** (`POST /auth/request-password-reset`): Request a password reset link via email.
- **Reset Password** (`POST /auth/reset-password`): Reset the password using the reset token.

## User Profile
- **Get My Profile** (`GET /me`): Fetch user profile details, requiring authentication.
- **Get User Preferences** (`GET /me/preferences`): Get user preferences, requiring authentication.
- **Update User Preferences** (`PATCH /me/preferences`): Update user preferences, requiring authentication.

## File Management API
- **File Upload** (`POST /file`): Upload a file to the server (Guest access allowed).
- **List User Files** (`GET /file`): List all files associated with a user, requiring authentication.
- **Get File** (`GET /file/:id`): Retrieve a specific file by ID, requiring authentication.
- **Delete File** (`DELETE /file/:id`): Delete a file by ID, requiring authentication.

## PDF Services API
- **Merge PDFs** (`POST /pdf/merge`): Create a new merge job.
- **Get Merge Job** (`GET /pdf/merge/:id`): Retrieve the status of a merge job.
- **Split PDFs** (`POST /pdf/split`): Create a new split job.
- **Get Split Job** (`GET /pdf/split/:id`): Retrieve the status of a split job.
- **Compress PDFs** (`POST /pdf/compress`): Create a new compress job.
- **Get Compress Job** (`GET /pdf/compress/:id`): Retrieve the status of a compress job.
- **Convert PDFs to JPG** (`POST /pdf/pdf-to-jpg`): Create a new PDF to JPG conversion job.
- **Get PDF to JPG Job** (`GET /pdf/pdf-to-jpg/:id`): Retrieve the status of a PDF to JPG conversion job.
- **Convert PDFs to Word** (`POST /pdf/pdf-to-word`): Create a new PDF to Word conversion job.
- **Get PDF to Word Job** (`GET /pdf/pdf-to-word/:id`): Retrieve the status of a PDF to Word conversion job.
- **Convert Word to PDF** (`POST /pdf/word-to-pdf`): Create a new Word to PDF conversion job.
- **Get Word to PDF Job** (`GET /pdf/word-to-pdf/:id`): Retrieve the status of a Word to PDF conversion job.
- **Convert Excel to PDF** (`POST /pdf/excel-to-pdf`): Create a new Excel to PDF conversion job.
- **Get Excel to PDF Job** (`GET /pdf/excel-to-pdf/:id`): Retrieve the status of an Excel to PDF conversion job.
- **Convert PowerPoint to PDF** (`POST /pdf/ppt-to-pdf`): Create a new PowerPoint to PDF conversion job.
- **Get PowerPoint to PDF Job** (`GET /pdf/ppt-to-pdf/:id`): Retrieve the status of a PowerPoint to PDF conversion job.

## Admin API
- **Promote User to Admin** (`POST /admin/users/:id/promote`): Promote a user to admin role, requires authentication and admin role.
- **Demote User** (`POST /admin/users/:id/demote`): Demote a user to a regular user role, requires authentication and admin role.
- **Set User Role** (`POST /admin/users/:id/role`): Set a user's role (admin or user), requires authentication and admin role.
- **Get Logs by Job ID** (`GET /admin/logs/:id`): Retrieve logs for a specific job by ID, requires authentication and admin role.
- **Clean Up Files** (`POST /admin/files/cleanup`): Clean up old or unnecessary files, requires authentication and admin role.

## Miscellaneous API
- **Contact Us** (`POST /contact`): Submit a contact form.
- **Download Job Files** (`GET /jobs/:type/:id/download`): Download files related to a specific job, requires authentication.

---

## Installation and Setup

### Prerequisites
- Go (Golang) version 1.18 or higher
- Docker (for local development with services like PostgreSQL, Redis, etc.)

### Clone the Repository
```bash
git clone https://github.com/your-repository-url.git
cd your-project-directory

Install Dependencies

go mod tidy
Running the Application Locally
Set up your environment variables in the .env file.

Run the application:


go run main.go
Docker Setup
You can also use Docker for containerized deployment. Make sure Docker is installed and then run:


docker-compose up --build
This will bring up the necessary services like PostgreSQL, Redis, etc., in containers.

API Usage
Once the application is running, you can interact with the API through the provided endpoints. You may use tools like Postman or Swagger to test the endpoints.

Swagger Documentation: http://localhost:8080/swagger

Authentication: All endpoints starting with /me and /file require user authentication (JWT token).

Contributing
Feel free to fork the repository, create issues, and submit pull requests for new features or bug fixes.

License
This project is licensed under the MIT License - see the LICENSE file for details.



This `README.md` contains a clear overview of your project, the API endpoints available, how to set up and run the 
