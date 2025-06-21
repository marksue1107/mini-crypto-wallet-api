# 💸 mini-crypto-wallet-api

A simple cryptocurrency wallet backend API built with **Golang** + **Gin** + **Kafka** + **GORM** + **SQLite**.

This project was built as a technical demo for crypto exchange interview purposes.  
It simulates user creation, wallet management, fund transfer, and transaction history retrieval.

---

## ✨ Features

- RESTful API using Gin
- SQLite + GORM (Auto migration)
- Swagger (OpenAPI 3.0 docs)
- Wallet transfer with balance update
- Transaction history
- ✔ Kafka `tx.created` event publishing (via `segmentio/kafka-go`)
- ⚡ Quick to run — lightweight setup with optional Kafka

---

## 📦 API Endpoints

| Method | Path                         | Description                      |
|--------|------------------------------|----------------------------------|
| POST   | `/users`                     | Create a new user + wallet       |
| GET    | `/wallet/{user_id}`          | Get wallet balance               |
| POST   | `/wallet/transfer`           | Transfer funds between users     |
| GET    | `/transactions/{user_id}`    | Get transaction history          |
| GET    | `/tx/{hash}`                 | Query transaction by hash        |

👉 API docs available at: [`/swagger/index.html`](http://localhost:8080/swagger/index.html)

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