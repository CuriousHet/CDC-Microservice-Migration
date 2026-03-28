# Phase 9: Debugging & Verification Guide

Phase 9 focused on creating the "Human Interface" for the migration—a set of tools and guides to help developers troubleshoot the complex CDC pipeline.

## Objective
Empower the team to identify exactly where a record is failing in the transition from Monolith to Microservice.

## Concepts Involved
- **End-to-End Tracing**: Following a record from WAL -> Kafka -> Worker -> DB.
- **CLI Tooling**: Using `docker exec` and `curl` for rapid verification.
- **Parity Checking**: Explicitly comparing "Expected" (Monolith) vs "Actual" (Microservice) data.

## Key Artifacts

### 1. The Debug Guide (`DEBUG.md`)
- A single-source-of-truth command reference found in the project root.
- Includes commands for checking row counts, inspecting Kafka topics, and viewing service logs.

### 2. The Verification Workflow (`docs/debugging-steps.md`)
- A conceptual guide explaining the 5 stages of the data journey:
  1. The Source (Monolith DB)
  2. The Capture (Debezium)
  3. The Transport (Kafka)
  4. The Sync (Migration Workers)
  5. The Verify (API Gateway / Parity Checker)

## What You Should Learn
- How to debug "invisible" data flows like CDC.
- Using Kafka CLI tools to inspect streaming events.
- Creating idempotent verification logic (`POST /verify/:id`).

## Verification Commands
- Check User Service Parity: `curl -X POST http://localhost:8081/verify/1`
- Check Order Service Parity: `curl -X POST http://localhost:8082/verify/1`
