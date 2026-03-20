# Phase 4: Order Service Decomposition

## Purpose
The goal of Phase 4 is to decompose the `orders` component from the monolith. This follows the same pattern as the User Service but introduces cross-service data constraints (Orders belonging to Users).

## 🧠 Concepts Involved
- **Loose Coupling**: Services are independent; they don't share a database.
- **Referential Integrity in Microservices**: How to handle IDs from other services without a physical FOREIGN KEY.
- **Data Locality**: Deciding which data belongs in which service.

## 🎓 What You Will Learn
1.  How to **scaffold a second microservice** in the same ecosystem.
2.  Why we **store `user_id` as a simple integer** instead of a constrained reference.
3.  The pattern of **Event-Driven Data Migration** applied to relational data.

## The Plan
1.  **Spin up `order-db`**: A separate database for Orders on port 5434.
2.  **Scaffold `order-service`**: A new Go project to manage order data.
3.  **Sync Data**: Implement a worker that consumes `monolith.public.orders` and populates the `order-db`.
4.  **Integrate**: Ensure the `order-service` can identify users by `user_id`.
5.  **Verify**: Access Order data via `http://localhost:8082/orders/{id}`.

## Technical Decisions
- **Port 8082**: The Order Service will listen on this port.
- **Port 5434**: The isolated Order DB will listen on this port.
- **Foreign Key Strategy**: In the microservice world, we store the `user_id` but do not enforce a database-level foreign key to the `user_db` (since they are separate). Consistency is handled at the application layer.

## Verification
- Monitor the `order-db` for synchronized records.
- Use `GET /orders/:id` on port 8082.
