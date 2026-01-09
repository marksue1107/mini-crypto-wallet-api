# 💸 mini-crypto-wallet-api

A production-ready cryptocurrency wallet backend API demonstrating enterprise-grade engineering practices for financial systems.

**Tech Stack**: Golang + Gin + GORM + PostgreSQL + Kafka + JWT

This project was built as a technical demo for crypto exchange interview purposes.
It simulates user creation, wallet management, fund transfer, and transaction history retrieval with robust concurrency safety and audit capabilities.

---

## ✨ Features

- RESTful API using Gin
- PostgreSQL (prod) / SQLite (dev) with GORM auto-migration
- Swagger (OpenAPI 3.0 docs)
- Concurrent-safe wallet transfers with row-level locking
- Multi-currency wallet support
- Transaction history with pagination
- JWT-based authentication and authorization
- Kafka `tx.created` event publishing for async processing
- Balance audit trail (BalanceHistory) for compliance
- Rate limiting and request tracing

---

## 🏗️ Engineering Highlights

### Concurrency Safety
- **Problem**: Race conditions in concurrent wallet transfers can cause balance inconsistencies
- **Solution**: PostgreSQL row-level pessimistic locking with `SELECT ... FOR UPDATE`
- **Implementation**: `GetWalletByUserIDWithTx` in `wallet_repository.go:38` uses GORM `clause.Locking`
- **Impact**: Zero race conditions under concurrent load, demonstrated in `concurrency_demo_test.go`

### Financial Precision
- **Problem**: Float arithmetic loses precision in financial calculations (e.g., 0.1 + 0.2 ≠ 0.3)
- **Solution**: `shopspring/decimal` library + DECIMAL(20,8) database type
- **Impact**: Accurate to 8 decimal places suitable for both crypto and fiat transactions

### Transaction Atomicity
- **Pattern**: Database transactions with explicit rollback handling
- **Guarantees**: Wallet updates + transaction records + audit logs succeed or fail together
- **Implementation**: Transfer executes 7 steps atomically:
  1. Lock source and destination wallets (`FOR UPDATE`)
  2. Validate balance sufficiency
  3. Update both wallet balances
  4. Create transaction record with hash/signature
  5. Record balance history (audit trail) for both users
  6. Commit transaction atomically
  7. Publish Kafka event (after commit, fire-and-forget)
- **Impact**: Data consistency guaranteed, no partial transfers possible

### Event-Driven Architecture
- **Integration**: Kafka producer publishes `tx.created` events after successful commit
- **Benefits**: Loose coupling enables async notifications, analytics, fraud detection, reporting
- **Resiliency**: Graceful degradation if Kafka unavailable (transactions still succeed)
- **Pattern**: Events published **after** DB commit to prevent inconsistencies

### Security Design

**Authentication**: JWT with HS256 signing
- 24-hour expiry with issued-at and not-before claims
- Custom claims include `user_id` and `username`
- Tokens passed via `Authorization: Bearer` header

**Password Security**: bcrypt hashing
- DefaultCost (10 rounds) for password hashing
- Timing-safe comparison with `bcrypt.CompareHashAndPassword`
- Passwords never stored in plaintext

**Authorization**: Resource-level access control
- Users can only access their own wallets and transactions
- `RequireUserID` middleware check in handlers prevents horizontal privilege escalation
- JWT claims validated on every protected endpoint

**Rate Limiting**: Token bucket algorithm
- Default: 60 requests/min per IP address
- `X-RateLimit-*` headers for client feedback
- Prevents API abuse and simple DDoS attempts

### Audit Trail & Compliance
- **Requirement**: Financial systems need tamper-proof transaction history
- **Solution**: `BalanceHistory` table records every balance change
- **Records Captured**:
  - Transaction ID linkage
  - Change type (credit/debit)
  - Balance before and after
  - Immutable timestamp
- **Impact**: Full auditability supports reconciliation, dispute resolution, and regulatory compliance

---

## 📐 Architecture

**Layered Architecture**:
```
Handlers (HTTP/REST)
    ↓
Services (Business Logic)
    ↓
Repositories (Data Access)
    ↓
Database (PostgreSQL/SQLite)
```

**Key Patterns**:
- **Repository Pattern**: Interfaces (`IWallet`, `ITransaction`, `IUser`) for testability and decoupling
- **Dependency Injection**: Repositories → Services → Handlers wired in `router/router.go`
- **Middleware Stack**: Trace ID, JWT Auth, Rate Limiting, Validation
- **Event Sourcing**: Kafka for async event publishing

**Database**: PostgreSQL (production), SQLite (development)
**Messaging**: Kafka for `tx.created` events
**Authentication**: JWT with Bearer tokens

---

## 🧪 Testing

Run all tests:
```bash
go test ./... -v
```

Run concurrency safety demonstration:
```bash
go test ./internal/test -v -run TestConcurrentTransfers
```

This test compares transfer behavior with and without database locking, proving the concurrency safety implementation.

---

## 📊 Design Trade-offs

### Why Pessimistic Locking?
- **Chosen**: Row-level locks with `SELECT ... FOR UPDATE`
- **Alternative**: Optimistic locking with version fields
- **Rationale**: Financial transactions require strong consistency guarantees over optimistic performance. Users expect immediate success/failure feedback rather than retry loops.

### Why Kafka After Commit?
- **Chosen**: Publish events only after successful DB commit
- **Alternative**: Two-phase commit or event sourcing as source of truth
- **Rationale**: Prioritizes consistency (DB as source of truth) over guaranteed event delivery. Acceptable trade-off: idempotency keys can address rare lost events in v2.

### Why Repository Pattern?
- **Chosen**: Interface-based repositories with dependency injection
- **Alternative**: Direct database access from services
- **Rationale**: Enables thorough unit testing with mocks, decouples business logic from persistence layer, follows SOLID principles (Dependency Inversion).

---

## 🔮 Future Enhancements

**High-Impact Features** that would extend this project:

- **Idempotency Keys**: Prevent duplicate charges on client retries (Stripe/PayPal pattern)
- **Distributed Tracing**: OpenTelemetry integration for microservices observability
- **Circuit Breaker**: Resilience pattern for Kafka failures (sony/gobreaker)
- **Read Replicas + CQRS**: Separate read/write databases for scalability
- **Webhook System**: Deliver transaction events to client URLs with retry logic
- **Saga Pattern**: Distributed transaction coordination for multi-service transfers

---

## 📦 API Endpoints

| Method | Path                         | Description                      | Auth Required |
|--------|------------------------------|----------------------------------|---------------|
| POST   | `/users`                     | Create a new user + wallet       | No            |
| POST   | `/auth/login`                | Authenticate and get JWT token   | No            |
| GET    | `/currencies`                | List all currencies              | No            |
| GET    | `/currencies/{id}`           | Get currency by ID               | No            |
| GET    | `/wallet/{user_id}`          | Get wallet balance               | Yes (JWT)     |
| POST   | `/wallet/transfer`           | Transfer funds between users     | Yes (JWT)     |
| GET    | `/transactions/{user_id}`    | Get transaction history (paginated) | Yes (JWT)  |
| GET    | `/tx/{hash}`                 | Query transaction by hash        | No            |
| GET    | `/health`                    | Health check                     | No            |
| GET    | `/ready`                     | Readiness check (DB connectivity) | No           |

**Documentation & Testing**:
- 📘 **Swagger UI**: [`/swagger/index.html`](http://localhost:8080/swagger/index.html)
- 📬 **Postman Collection**: Import `Mini-Crypto-Wallet-API.postman_collection.json` and `Mini-Crypto-Wallet.postman_environment.json`
  - Complete API flow with authentication
  - Automatic token management
  - Test scripts for validation
  - Negative test scenarios included

---

## 🛠️ How to Run

### 1. Build the Docker image

```bash
docker build -t mini-wallet-api .
```

### 2. Start Kafka and PostgreSQL

Use the provided `docker-compose.kafka.yml` file to launch the
supporting services:

```bash
docker compose -f docker-compose.kafka.yml up -d
```

### 3. Run the API container

```bash
docker run --rm -p 8080:8080 \
  -e APP_ENV=production \
  -e DB_DRIVER=postgres \
  -e POSTGRES_DSN="host=postgres user=postgres password=secret dbname=mini_wallet port=5432 sslmode=disable" \
  -e KAFKA_BROKER=kafka:9092 \
  mini-wallet-api
```

### Required environment variables

- `APP_ENV` – application environment
- `DB_DRIVER` – `postgres` or `sqlite`
- `POSTGRES_DSN` – PostgreSQL connection string
- `KAFKA_BROKER` – Kafka broker address

---

## 🧑‍💻 Author

Built by **Mark Syue** — for demo & interview purpose  
Feel free to connect or view my profile:


- 💼 [LinkedIn – Mark Syue](https://www.linkedin.com/in/syue-mark)
- 🎂 [CakeResume – Mark Syue](https://www.cake.me/s--i5n7w4G204d-tZ9T8Yv8ww--/mark-syue)
- 📧 Email: marksue1107@gmail.com