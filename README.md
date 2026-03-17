# CDC Microservice Migration: Monolith to Microservices 🚀

This repository demonstrates a real-world migration of a monolithic application to a microservices architecture using the **Strangler Fig Pattern** and **Change Data Capture (CDC)**.

## 🏗️ Architecture Overview

The project simulates a transformation from a single Go monolith with a shared database to independent microservices, each with its own isolated database, kept in sync via Kafka and Debezium.

- **Source**: Go Monolith (Users & Orders) + PostgreSQL
- **Streaming Backbone**: Kafka + ZooKeeper + Debezium
- **Target Microservice**: User Service (Go) + Isolated PostgreSQL

## 📂 Project Structure

- `/monolith`: The legacy application.
- `/user-service`: The new decomposed User microservice.
- `/traffic-generator`: Simulates real-world load on the monolith.
- `/stream-consumer`: A verification tool for Kafka events.
- `/docs`: Detailed guides for each migration phase.

## 🚀 Migration Phases

1. [**Phase 1: Foundation**](docs/phase-1.md) - Baseline monolith and infrastructure.
2. [**Phase 2: CDC Pipeline**](docs/phase-2.md) - Turning the DB into an event source.
3. [**Phase 3: Decomposition**](docs/phase-3.md) - Implementing the User service and data sync.

## 🛠️ Tech Stack

- **Language**: Go 1.21+
- **Databases**: PostgreSQL 15
- **Message Broker**: Kafka (Confluent Platform)
- **CDC Tool**: Debezium Connect
- **Deployment**: Docker Compose

## 🚦 Getting Started

1. Start the infrastructure:
   ```bash
   docker compose up -d
   ```

