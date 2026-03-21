# CDC Microservice Migration: Monolith to Microservices 🚀

This repository demonstrates a real-world migration of a monolithic application to a microservices architecture using the **Strangler Fig Pattern** and **Change Data Capture (CDC)**.

## 🏗️ Architecture Overview

The project simulates a transformation from a single Go monolith with a shared database to independent microservices, each with its own isolated database, kept in sync via Kafka and Debezium.

- **Source**: Go Monolith (Users & Orders) + PostgreSQL
- **Streaming Backbone**: Kafka + ZooKeeper + Debezium
- **Target Microservice 1**: User Service (Go) + Isolated PostgreSQL
- **Target Microservice 2**: Order Service (Go) + Isolated PostgreSQL
- **Entry Point**: API Gateway (Go) implementing **Strangler Fig Fallback**

## 📂 Project Structure

- `/monolith`: The legacy application (Source of Truth).
- `/api-gateway`: The "Strangler" entry point with transparent fallback logic.
- `/user-service`: The new decomposed User microservice.
- `/order-service`: The new decomposed Order microservice.
- `/traffic-generator`: Simulates real-world load on the monolith.
- `/stream-consumer`: A verification tool for Kafka events.
- `/docs`: Detailed guides for each migration phase.

## 🚀 Migration Phases

1. [**Phase 1: Foundation**](docs/phase-1.md) - Baseline monolith and infrastructure.
2. [**Phase 2: CDC Pipeline**](docs/phase-2.md) - Turning the DB into an event source.
3. [**Phase 3: Decomposition**](docs/phase-3.md) - Implementing the User service and data sync.
4. [**Phase 4: Scaling Out**](docs/phase-4.md) - Implementing the Order service.
5. [**Phase 5: Traffic Cutover**](docs/phase-5.md) - Implementing the API Gateway and Strangler Pattern.

## 🛠️ Tech Stack

- **Language**: Go 1.21+
- **Databases**: PostgreSQL 15, SQLite (optional for quick tests)
- **Message Broker**: Kafka (Confluent Platform)
- **CDC Tool**: Debezium Connect
- **Deployment**: Docker Compose

## 🚦 Getting Started

1. Start the infrastructure:
   ```bash
   docker compose up -d
   ```

