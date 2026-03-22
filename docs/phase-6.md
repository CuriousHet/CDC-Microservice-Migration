# Phase 5.5: Historical Data Backfill

## 🎯 Objective
To understand and implement a **Bulk Migration** strategy for syncing data that existed *before* the CDC pipeline was established.

## 🧠 Concepts to Learn

### 1. The Snapshot vs. The Stream
- **The Stream (CDC)**: Captures changes *happening now*.
- **The Snapshot (Initial Load)**: Captures the *current state* of the database.
Debezium usually does this automatically on startup, but in complex migrations, you often need a manual tool for:
- Re-syncing corrupted data.
- Migrating data to a different schema.
- Throttling the sync to avoid overloading the production database.

### 2. Idempotent Upserts
Our microservices use `INSERT ... ON CONFLICT (id) DO UPDATE`. This is the "Secret Sauce" of backfilling. It means we can run the backfill script 100 times, and it will never create duplicate data—it will only ensure the microservice is perfectly in sync with the monolith.

## 🛠️ What are we building?

We are building a **Backfill Utility** in `backfill/main.go`. This script will:
1.  Read all rows from the Monolith `users` table.
2.  Format them into the same JSON structure that Debezium produces.
3.  Push them directly to the `monolith.public.users` Kafka topic.

**Result**: The `user-service` will "think" a change just happened and sync the data automatically, even if that data was created 5 years ago.

