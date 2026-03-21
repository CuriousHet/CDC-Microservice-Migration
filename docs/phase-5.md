# Phase 5: Strangler Pattern Rollout

## Purpose
The goal of Phase 5 is to implement the **Strangler Fig Pattern**. This pattern allows us to progressively migrate traffic from the Legacy Monolith to our new Microservices without any downtime or "big bang" cutover.

## 🧠 Concepts Involved
-   **API Gateway**: A single entry point for all clients. It handles request routing, composition, and protocol translation.
-   **Reverse Proxy**: The gateway acts as a proxy, forwarding requests to the appropriate backend service.
-   **Canary Release / Traffic Shifting**: Moving a small percentage of traffic to a new system to verify its behavior in production.
-   **Strangler Fig Pattern**: Incrementally replacing system functionality by "strangling" the old system with new services.

## 🎓 What You Will Learn
1.  How a **Reverse Proxy** works in a microservices architecture.
2.  The logic of **Path-based Routing** (deciding where to send a request based on its URL).
3.  How to implement a **Graceful Fallback** (if the microservice fails or has no data, ask the monolith).

## The Plan
1.  **Build the `api-gateway`**: A lightweight Go service that listens on port `8000`.
2.  **Define Routes**:
    -   `GET /users/:id` -> Try `user-service`. If 404/Error, fallback to `monolith`.
    -   `GET /orders/:id` -> Try `order-service`. If 404/Error, fallback to `monolith`.
    -   **Everything else** -> Forward directly to 8080 (`monolith`).
3.  **Update infrastructure**: Add the gateway to `docker-compose.yml`.
4.  **Verify**: Access the entire system via one single port (`8000`).

## Technical Decisions
-   **Go + Gin**: Leverages the same stack as our other services for consistency.
-   **Explicit Proxying**: We won't use a heavy tool like Nginx or Kong yet; building it in Go helps understand the underlying logic.
-   **No "Writes" to Microservices**: To keep the Monolith as the "Source of Truth" during the transition, all `POST`, `PUT`, and `DELETE` requests stay on the Monolith.

## Verification
-   Run `curl http://localhost:8000/users/1`.
-   Check gateway logs to see it routing to `user-service`.
-   Verify that even if `user-service` is down, the gateway can fallback to the `monolith`.
