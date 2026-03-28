# Phase 8: Data Synchronization Visibility

The goal of Phase 8 was to provide real-time insights into the migration progress and ensure data parity between the Monolith and the new Microservices.

## Objective
Implement a monitoring layer that allows engineers to track the "Sync Progress" and verify data integrity without manually querying individual databases.

## Concepts Involved
- **Real-time Metrics**: Pulling counts from multiple isolated databases.
- **Visual Dashboard**: Using HTML templates served by the API Gateway to visualize system health.
- **Sync Progress Calculation**: Measuring the ratio of records in the destination vs the source.

## Implementation Details

### 1. Dashboard UI
- Created `dashboard.html` in the `api-gateway` folder.
- Uses a modern, dark-themed UI with status bars for User and Order synchronization.
- **URL**: `http://localhost:8000/dashboard`

### 2. Metrics Backend
- Updated `api-gateway/main.go` with a background database poller (`startDBPoller`).
- Periodically queries:
  - `monolith-db.users` vs `user-db.users`
  - `monolith-db.orders` vs `order-db.orders`
- Exposes data via `GET /metrics`.

## What You Should Learn
- How to coordinate observability across a distributed system.
- The importance of "Confidence Metrics" during a live migration.
- How to build a simple but effective monitoring tool using Go and HTML templates.
