# Phase 1: Foundation & Legacy Baseline

## Purpose
The primary goal of Phase 1 is to establish a working monolithic system that experiences "production-like" traffic. This serves as our starting point (the system to be migrated). 

By completing this phase, we ensure:
1. We have a target to "strangle" in later phases.
2. We have a shared data source (Postgres) supporting the monolith.
3. we have the infrastructure (Kafka/Debezium) ready for Phase 2.

## Components
1. **Infrastructure (Docker):** A shared environment containing our databases and streaming backbone.
2. **The Monolith (Go):** A simple service managing `Users` and `Orders` in a single database.
3. **Traffic Generator (Go):** A tool that constantly hammers the monolith with requests to simulate real-world usage.

## Technical Decisions
- **Go:** Chosen for its concurrency support and standard usage in high-traffic backend systems.
- **PostgreSQL:** The source of truth for our monolithic system.
- **Docker Compose:** To ensure the environment is reproducible and isolated.
