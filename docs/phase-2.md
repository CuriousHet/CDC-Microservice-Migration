# Phase 2: CDC Pipeline & Data Streaming

## Purpose
The goal of this phase is to turn our database into an event source. Instead of just storing data, every change (INSERT, UPDATE, DELETE) in the monolith database will now be published as an event to Kafka.

This is the "Heart" of the Strangler Pattern—it allows us to keep the new microservices in sync with the legacy system without modifying the legacy code.

## Components
1. **Debezium Postgres Connector:** A service that watches the Postgres Write-Ahead Log (WAL) and produces events to Kafka.
2. **Kafka Topics:** Automatically created topics that hold the change events (e.g., `monolith.public.users`).
3. **Verification Consumer:** A simple script/tool to prove that events are flowing through the pipeline.

## Technical Decisions
- **Logical Decoding:** We use Postgres' `pgoutput` plugin for logical replication.
- **JSON Serialization:** Events are serialized as JSON for simplicity and readability, though Avro/Protobuf are often preferred in high-scale production.
- **Snapshot Mode:** Debezium will perform an initial snapshot of existing data before starting to stream real-time changes.
