# Postman Collection Guide

## Quick Start

### 1. Import Files into Postman

1. Open Postman
2. Click **Import** button (top left)
3. Import both files:
   - `Mini-Crypto-Wallet-API.postman_collection.json`
   - `Mini-Crypto-Wallet.postman_environment.json`
4. Select the **"Mini Crypto Wallet - Local"** environment (top right dropdown)

### 2. Start the API Server

```bash
# Ensure PostgreSQL and Kafka are running (or use SQLite for dev)
go run main/main.go
```

Server should start on `http://localhost:8080`

### 3. Run the Complete Flow

Execute requests in order:

**Step 1: Authentication Flow** (Folder 1)
- Run "Register Alice" → Creates user, saves `alice_id` to environment
- Run "Register Bob" → Creates user, saves `bob_id` to environment
- Run "Login Alice" → Gets JWT token, saves `alice_token` to environment
- Run "Login Bob" → Gets JWT token, saves `bob_token` to environment

**Step 2: Check Currencies** (Folder 2)
- Run "Get All Currencies" → Saves first `currency_id` to environment

**Step 3: Check Wallets** (Folder 3)
- Run "Get Alice's Wallet" → See initial balance (1000 USDT)
- Run "Get Bob's Wallet" → See initial balance (1000 USDT)

**Step 4: Perform Transfer** (Folder 4)
- Run "Transfer: Alice → Bob (100 USDT)" → Transfer succeeds
- Check balances again to verify update

**Step 5: View History** (Folder 5)
- Run "Get Alice's Transactions" → See transfer record
- Run "Query Transaction by Hash" → Look up specific transaction

---

## Collection Structure

### Folder 1: Authentication Flow
**Purpose**: Register users and obtain JWT tokens

- **Register Alice** - Creates user account with initial 1000 USDT wallet
- **Register Bob** - Creates second user for transfer testing
- **Login Alice** - Get JWT token (auto-saved to `alice_token`)
- **Login Bob** - Get JWT token (auto-saved to `bob_token`)

### Folder 2: Currencies
**Purpose**: Query available currencies

- **Get All Currencies** - List all supported currencies (USDT, BTC, ETH, etc.)
- **Get Currency by ID** - Get details of specific currency

### Folder 3: Wallet Operations
**Purpose**: Check wallet balances with authorization

- **Get Alice's Wallet** - Check balance (requires Alice's token)
- **Get Bob's Wallet** - Check balance (requires Bob's token)
- **[Negative] Get Other User's Wallet (403)** - Demonstrates authorization failure

### Folder 4: Transfer Scenarios
**Purpose**: Money transfer with validation and error handling

**Successful Transfer:**
- **Transfer: Alice → Bob (100 USDT)** - Complete transfer demonstrating:
  - Row-level locking (concurrency safety)
  - Atomic balance updates
  - Transaction record creation
  - Balance history audit trail
  - Kafka event publishing

**Error Scenarios:**
- **[Negative] Transfer to Same Account (400)** - Validation error
- **[Negative] Insufficient Balance (400)** - Balance check
- **[Negative] Unauthorized Transfer (403)** - Authorization check (Alice can't transfer from Bob's account)
- **[Negative] Missing Auth Token (401)** - Authentication required

### Folder 5: Transaction History
**Purpose**: Query transaction records

- **Get Alice's Transactions** - Paginated history (auto-saves last transaction hash)
- **Get Bob's Transactions** - Paginated history
- **Query Transaction by Hash** - Look up specific transaction

### Folder 6: Health & Monitoring
**Purpose**: Service health checks

- **Health Check** - Basic liveness check
- **Readiness Check** - Database connectivity verification

---

## Environment Variables

All variables are automatically populated by test scripts:

| Variable | Purpose | Auto-Populated By |
|----------|---------|-------------------|
| `base_url` | API server URL | Pre-configured (http://localhost:8080) |
| `alice_token` | Alice's JWT token | Login Alice request |
| `bob_token` | Bob's JWT token | Login Bob request |
| `alice_id` | Alice's user ID | Register Alice / Login Alice |
| `bob_id` | Bob's user ID | Register Bob / Login Bob |
| `currency_id` | Default currency ID | Get All Currencies |
| `last_tx_hash` | Last transaction hash | Get Alice's Transactions |

**Note**: You don't need to manually set these - the collection's test scripts handle it automatically!

---

## Test Scripts Included

Each request includes automatic validation:

**Authentication Requests:**
```javascript
// Auto-extract token from login response
if (pm.response.code === 200) {
    const response = pm.response.json();
    pm.environment.set("alice_token", response.token);
    pm.environment.set("alice_id", response.user_id);
}
```

**Transfer Validation:**
```javascript
pm.test("Transfer successful", () => {
    pm.response.to.have.status(200);
    const response = pm.response.json();
    pm.expect(response.message).to.eql('transfer successful');
});
```

**Negative Test Validation:**
```javascript
pm.test("Should return 403 Forbidden", () => {
    pm.response.to.have.status(403);
});
```

---

## Running All Tests

### Option 1: Collection Runner
1. Click on "Mini Crypto Wallet API" collection
2. Click "Run" button
3. Select environment: "Mini Crypto Wallet - Local"
4. Click "Run Mini Crypto Wallet API"
5. Watch all tests execute in sequence

### Option 2: Manual Execution
Execute folders in order:
1. Folder 1: Authentication Flow (4 requests)
2. Folder 2: Currencies (2 requests)
3. Folder 3: Wallet Operations (3 requests)
4. Folder 4: Transfer Scenarios (5 requests)
5. Folder 5: Transaction History (3 requests)
6. Folder 6: Health & Monitoring (2 requests)

**Total**: 19 requests demonstrating complete API functionality

---

## Troubleshooting

### "Request failed: connection refused"
- Ensure API server is running: `go run main/main.go`
- Check server is on port 8080: `lsof -i :8080`

### "401 Unauthorized" on protected endpoints
- Run "Login Alice" or "Login Bob" first to get fresh token
- JWT tokens expire after 24 hours

### "404 Not Found" on transaction hash query
- Ensure you've run a transfer first
- Check `last_tx_hash` environment variable is set

### "403 Forbidden" on wallet/transfer
- Ensure you're using the correct user's token
- Alice can only transfer from Alice's account (and vice versa)

### Database errors
- Check PostgreSQL is running: `docker ps`
- Or use SQLite for development (configured in `config.yaml`)

---

## What This Demonstrates

### Backend Engineering Skills

**1. Authentication & Authorization**
- JWT-based authentication with Bearer tokens
- Resource-level authorization (users can only access own data)
- Horizontal privilege escalation prevention

**2. Data Validation**
- Amount validation (positive, non-zero)
- Same account check
- Balance sufficiency check
- Currency matching

**3. Concurrency Safety**
- Row-level locking prevents race conditions
- Atomic transactions (all-or-nothing)
- Demonstrated in transfer operations

**4. API Design Best Practices**
- RESTful endpoints
- Proper HTTP status codes (200, 400, 401, 403, 404)
- Pagination support (transactions endpoint)
- Public vs protected routes

**5. Observability**
- Health and readiness checks
- Transaction hashing for tracking
- Audit trail (BalanceHistory)

**6. Error Handling**
- Clear error messages
- Appropriate status codes
- No partial state on failures

---

## Next Steps

After exploring the API with Postman, consider:

1. **Load Testing**: Use Postman Collection Runner to test concurrent requests
2. **Custom Scenarios**: Create your own requests exploring edge cases
3. **CI/CD Integration**: Export collection and run with Newman CLI
4. **Explore Code**: Review implementation in `handlers/`, `services/`, `repositories/`

---

## Collection Metadata

- **Total Requests**: 19
- **Folders**: 6
- **Negative Tests**: 4
- **Auth Required**: 7 endpoints
- **Test Scripts**: All requests include validation
- **Environment Variables**: 7 (auto-populated)

**Last Updated**: 2026-01-09
