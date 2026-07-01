FreshTrack – Warehouse Receiving & Inventory Management System
Project Overview

FreshTrack is a production-oriented warehouse receiving and inventory management system built using Go, PostgreSQL, Redis, JWT Authentication, Docker, and a frontend developed with HTML, CSS, and Vanilla JavaScript.

The application is designed for organizations that receive inventory across multiple warehouses. It enables warehouse operators to scan incoming products, reconcile received quantities against invoices, and allows administrators to manage warehouses, users, reports, and audit logs from a centralized dashboard.

The project follows a clean backend architecture using Go while keeping the frontend lightweight without any JavaScript frameworks.

Technology Stack
Backend
Go
PostgreSQL
Redis
JWT Authentication
Docker & Docker Compose
Frontend
HTML
CSS
Vanilla JavaScript
User Roles
1. Central Admin

The administrator has complete control over the system and can:

Login and Logout
View dashboard analytics
Create, edit and disable users
Create and manage warehouses
Assign one or multiple warehouses to hub users
Upload invoices using CSV or Excel (.xlsx)
View uploaded invoices
Generate reconciliation reports
Download reports in CSV and Excel formats
View audit logs
2. Hub User

Warehouse users can:

Login and Logout
Access only assigned warehouses
Select an assigned warehouse
View warehouse invoices
Scan products using barcode scanners
Perform manual quantity adjustments
Finish receiving
View receiving history

Warehouse isolation is enforced on the server, ensuring users cannot access data from warehouses that are not assigned to them.

Major Features
Authentication
JWT-based authentication
Role-based authorization
Session expiration handling
Secure password hashing using bcrypt
Warehouse Management
Warehouse CRUD
Multiple warehouse assignment per user
Warehouse isolation for hub users
Invoice Management

Supports both:

CSV Upload
Excel (.xlsx) Upload

Upload validation includes:

Warehouse existence
Unique invoice IDs
Positive quantities
Non-empty SKU validation
Preventing invoices from spanning multiple warehouses
Receiving Workflow

Warehouse operators receive inventory using barcode scanners.

Features include:

Barcode scanning
Manual increment/decrement
Receiving completion
Automatic invoice status updates

Invoice lifecycle:

Pending

↓

Receiving

↓

Completed

Redis Integration

Redis is used for two important purposes:

Duplicate Scan Protection

Accidental duplicate barcode scans occurring within milliseconds are prevented using Redis SETNX with expiration.

Scan Queue Processing

Barcode scan events are pushed into Redis Streams and processed asynchronously by a background worker before updating PostgreSQL and writing audit logs.

Real-Time Updates

The application uses Server-Sent Events (SSE) to provide live receiving progress without requiring page refreshes.

Reporting

Administrators can generate reconciliation reports filtered by:

Date
Warehouse
Vendor
Status

Reports can be exported as:

CSV
Excel
Audit Logging

Every inventory operation is logged, including:

Timestamp
User
Warehouse
Invoice
SKU
Previous quantity
Updated quantity
Action performed
Reason (when applicable)
Frontend

The UI is built entirely using:

HTML
CSS
Vanilla JavaScript

Features include:

Responsive design
Dark sidebar navigation
Search
Pagination
Sorting
Client-side validation
Loading indicators
Toast notifications
Warehouse-optimized barcode scanning interface
Project Structure
cmd/server              Application entry point

internal/auth           Authentication & JWT

internal/config         Configuration

internal/db             PostgreSQL connection & migrations

internal/handlers       HTTP handlers

internal/middleware     JWT & role middleware

internal/models         Shared models

internal/queue          Redis stream worker

internal/redisclient    Redis configuration

internal/router         Route registration

web                     HTML/CSS/JavaScript frontend
Running the Project
Requirements
Go 1.22+
Docker
Docker Compose
PostgreSQL 16
Redis 7
Setup
cp .env.example .env
go mod tidy
Start the application
docker compose up --build

The application will be available at:

http://localhost:8080

If running the Go application locally, start only PostgreSQL and Redis:

docker compose up postgres redis

go run ./cmd/server
Testing

Run:

go test ./...
go build ./cmd/server
docker compose config
Demonstration

After starting the application, the main pages are:

Admin

/admin/dashboard.html
/admin/users.html
/admin/warehouses.html
/admin/upload.html
/admin/reports.html
/admin/audit.html

Hub

/hub/dashboard.html
/hub/invoices.html
/hub/scan.html
/hub/history.html
Summary

FreshTrack demonstrates the implementation of a complete warehouse receiving workflow using Go and modern backend practices. The project incorporates secure authentication, warehouse-based authorization, invoice management, barcode-driven receiving, asynchronous processing with Redis Streams, duplicate scan protection, audit logging, reporting, and a responsive frontend built entirely with HTML, CSS, and Vanilla JavaScript.
