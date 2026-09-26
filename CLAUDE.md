# CLAUDE.md

## 專案結構
- Go module 與所有後端原始碼都在 `api/`（`go.mod`、`main/`、`handlers/`、
  `services/`、`repositories/`、`models/`、`middleware/`、`internal/`、
  `db_conn/`、`docs/`）。根目錄只放文件、Docker Compose、Postman 收藏集。
- 分層固定為 `handlers → services → repositories → db_conn`。handler 不可
  直接碰資料庫；repository 是唯一允許出現 GORM 查詢語法的地方。
- `models/*.go`（GORM model）與 `models/*_dto.go`（HTTP request/response）
  嚴格分離：model 只放 `gorm` tag，DTO 只放 `json`/`binding` tag，不要混用。

## 金流鐵律
- 金額、餘額全程使用 `github.com/shopspring/decimal`。**任何情況下都不可以**
  在金流路徑（`Wallet.Balance`、`Transaction.Amount`、`BalanceHistory` 相關
  欄位）引入 `float32`/`float64`。
- 任何會同時鎖兩顆錢包的程式碼，鎖定順序永遠依 `user_id` 由小到大，不可依呼叫
  參數順序（否則兩個方向的併發轉帳會 deadlock，見 `services/transaction_service.go`
  的 `Transfer()` 實作與 `internal/test/deadlock_test.go` 的迴歸測試）。
- 查詢/鎖定錢包一律要同時帶 `user_id` 與 `currency_id`
  （`GetWalletByUserIDAndCurrencyWithTx`），不可只用 `user_id` 查詢後再事後比對
  幣別——一旦使用者能持有多顆不同幣別的錢包，「先撈任一筆再檢查」會撈到錯的錢包。
- 任何 `tx := db.Begin()` 之後，所有提前 `return` 的路徑都必須確保會
  `Rollback()` 或 `Commit()`，不能只在 `recover()` 裡處理 panic（一般的
  `return err` 不會觸發 `recover()`，會讓交易連線卡在 idle-in-transaction，
  查資金正確性相關程式碼時務必檢查這件事）。標準寫法是用一個 `committed bool`
  +單一 deferred 收尾函式，可參考 `services/transaction_service.go`。
- SQLite 模式（`db_driver: sqlite`／未設定）**不提供真正的併發安全**：
  GORM 的 sqlite driver 會把 `FOR UPDATE` 直接丟掉（SQLite 沒有 row-level
  lock）。只有 `db_driver: postgres` 才有真正的悲觀鎖保護。SQLite 僅供單機
  開發使用，不要在需要驗證併發正確性的情境下依賴它。

## API 錯誤格式
- 所有錯誤回應一律是 `models.ErrorResponse{error, code, message}`，透過
  `internal/errors.RespondError(c, status, code, err)` 送出，`code` 用
  `internal/errors/codes.go` 裡定義的常數。**不要**在 handler 或 middleware
  裡手刻 `gin.H{"error": ...}`。
- 需要幫一個新的業務錯誤配 code 時，在對應的 service 檔案裡定義套件層級的
  sentinel error（例如 `services.ErrInsufficientBalance`），handler 端用
  `errors.Is` 或查表對應到 HTTP 狀態碼與 `internal/errors` 的常數，不要靠比對
  錯誤訊息字串。
- go-playground/validator 產生的綁定錯誤，一律透過
  `middleware.HandleValidationError(c, err)` 處理（會在基本格式上多一個
  `details` 欄位列出逐欄位錯誤），不要自己 `c.JSON(400, ...)`。

## 前後端邊界
- 後端完全不含前端程式碼；前端若要放進這個 repo，放在根目錄新的資料夾
  （例如 `web/`），與 `api/` 平行，不要混進 `api/` 底下。
- CORS 預設關閉，前端網域要透過 `CORS_ALLOWED_ORIGINS` 環境變數明確加白名單
  才會生效（見 `router/router.go`）。
- Swagger/OpenAPI 文件是唯一的 API contract 來源。改動任何 handler 的
  `@Summary`/`@Param`/`@Success`/`@Failure` 註解後，一定要重新產生文件（見下方
  指令），不要讓文件與程式碼分岔。

## 設定與密鑰
- 沒有任何密鑰寫死在程式碼或 committed 的設定檔裡。`api/config.yaml` 已被
  gitignore，只有 `api/config.yaml.example`（純佔位值）會進版控。正式環境一律
  用環境變數：`JWT_SECRET`、`POSTGRES_DSN`、`DB_DRIVER`、`APP_ENV`、
  `KAFKA_BROKER`、`TRUSTED_PROXIES`、`CORS_ALLOWED_ORIGINS`、
  `MAX_TRANSFER_AMOUNT`。`JWT_SECRET` 沒設定或長度 < 32 時，服務會直接拒絕啟動
  （`internal/auth.ValidateSecretStrength`），不要加任何「找不到就用預設值」的
  fallback。
- `docker-compose.yml` 的 `api` service 從根目錄的 `.env` 讀取這些變數
  （`.env.example` 是範本，`.env` 已 gitignore）。

## 常用指令
全部在 `api/` 目錄下執行：
```bash
go build ./...
go vet ./...
go test ./... -race           # 部分測試會用 testcontainers 起真正的 Postgres，需要本機有 Docker 在跑；沒有 Docker 時會自動 skip
swag init -g main/main.go -o docs   # 改了 Swagger 註解後重新產生 swagger.json/.yaml
go run ./cmd/swagger2openapi        # 從 swagger.json 重新產生 OpenAPI 3.0 的 openapi.yaml
```
根目錄：
```bash
cp .env.example .env   # 設定 POSTGRES_PASSWORD、JWT_SECRET 後
docker compose up -d   # 一鍵起完整環境（postgres + kafka + zookeeper + api）
docker compose down    # 關閉（加 -v 會連 Postgres volume 一起清掉）
```

## 禁止事項
- 不要在非 `_test.go` 的檔案裡 `import "testing"`，也不要在正式程式碼裡放
  「刻意不安全」的示範/測試專用邏輯（這類東西屬於測試檔案）。
- 不要新增只寫一半、沒有實際呼叫端的中介層或工具函式；用不到就先不要提交。
- 不要把 `wallet-api`、`main/__debug_bin*`、`*.db`、`.env`、`api/config.yaml`
  加入版控——這些都已經在 `.gitignore` 裡，若編輯 `.gitignore` 請小心不要用
  過寬的 glob 誤傷需要進版控的檔案（例如 `.env.*` 這種規則要記得加
  `!.env.example` 例外；這個專案已經因為類似的誤植問題把 `README.md`／`docs/`
  誤放進 `.gitignore` 過一次）。
- 未經使用者確認，不要對 git 歷史做 `filter-repo`、`push --force` 之類的重寫
  操作，也不要自行安裝/新增使用者沒有明確授權的外部工具或套件（包含
  `npx`/`npm` 套件與 Go 依賴）——先说明要用什麼、為什麼，等確認再動手。
