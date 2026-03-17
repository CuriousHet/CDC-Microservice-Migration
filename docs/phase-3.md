# Phase 3: Decomposition (The User Service) 🏗️

## Objective
In this phase, we move from just "watching" data flow to actually **storing** it in a new, independent service. We are building the first "leaf" of our microservice tree: the **User Service**.

## Concepts
### 1. Database-per-Service
In a monolith, everything is in one big database. In microservices, each service **owns** its data.
- **Why?** So that a change in the `user-service` schema doesn't break the `order-service`.
- **The Rule:** No service should EVER touch another service's database directly.

### 2. The Migration Worker (The Bridge)
Since the Monolith is still the "Source of Truth" (where users sign up), we need a way to keep our new `User DB` in sync.
- The **Migration Worker** is a background process inside our new service.
- it listens to the Kafka topic we set up in Phase 2.
- When it see a `CREATE` event, it saves that user to its local `User DB`.

### 3. Idempotency (The "Double-Check" Rule)
In distributed systems, Kafka might send the same message twice (e.g., if there's a network glitch).
- **Idempotency** means that if we receive the "Create User 101" message 5 times, we only create the user **once**.
- **How?** We use the `id` from the Monolith as our primary key. Before inserting, we check if it already exists, or use an `ON CONFLICT DO NOTHING` SQL command.

## What You Should Learn
1.  How to set up a second, isolated database.
2.  How to write a Go service that "listens" to Kafka 24/7.
3.  How to transform "CDC events" into actual database rows.

---

## The Plan
1.  **Spin up `user-db`**: A fresh Postgres instance just for the User Service.
2.  **Scaffold `user-service`**: A new Go project.
3.  **Implement the Worker**: Code that heartbeats Kafka and syncs data.
4.  **Implement the API**: A simple `GET /users/:id` endpoint in the new service.

---

## 🛠️ Implementation Details & Challenges

### 1. Debezium Event Schema
Debezium wraps the data in a `payload` object. The Go worker must use a matching struct to avoid unmarshaling errors.
- **Key Fields:** `op` (Operation), `after` (New State), `before` (Old State).

### 2. Timestamp Handling (Crucial)
Debezium sends timestamps as **integers (microseconds)** from the epoch, while Go expects `time.Time` or seconds/nanoseconds.
- **The Fix:** `time.Unix(0, microsecondValue * 1000)` converts the value correctly to Go's time format.

### 3. Idempotent Upserts
To handle message retries, we use the `ON CONFLICT (id) DO UPDATE` pattern.
```sql
INSERT INTO users (id, email, name, created_at)
VALUES ($1, $2, $3, $4)
ON CONFLICT (id) DO UPDATE SET
    email = EXCLUDED.email,
    name = EXCLUDED.name;
```

## ✅ Verification
Use the following command to check if data is flowing:
```bash
docker exec -i user-db psql -U postgres -d user_service -c "SELECT count(*) FROM users;"
```
Check individual records:
```bash
curl http://localhost:8082/users/:id
```
