# mini-crypto-wallet-api 稽核報告

日期：2026-09-26
方法：全原始碼閱讀（handlers/services/repositories/middleware/models/config/docs）、`go build ./...`、`go vet ./...`、`go test ./... -race`（可重現）、`git ls-files` 盤點已提交檔案、逐一比對 README / IMPROVEMENTS_SUMMARY / REFACTORING_SUMMARY 與實際程式碼、逐一比對 swagger.json 與 router.go。本輪未修改任何程式碼。

---

## 1. 系統現況總結

這是一個履歷/面試導向的示範專案，分層架構（handler → service → repository）大致乾淨、DTO 與 GORM model 確實分離、金額全程使用 `shopspring/decimal`（未發現 float64 混用在金流路徑上）、水平越權檢查（`RequireUserID`）在 wallet/transfer/transactions 端點都有正確套用、密碼用 bcrypt、登入不洩漏帳號是否存在。但代碼品質與文件的「宣稱」和「實際行為」有明顯落差：README 宣稱已具備的 Rate Limiting 其實從未掛載到路由上；Swagger 宣稱是 OpenAPI 3.0 實際輸出是 2.0；JWT 密鑰與 Postgres 密碼以明文提交在已追蹤的 `config.yaml`；兩顆編譯後的執行檔（合計約 109MB）被提交進 git；`docker-compose.yml` 沒有 API 服務、README 引用的 `docker-compose.kafka.yml` 根本不存在。最嚴重的是：**目前 main 分支上 `go test ./... -race` 會直接 panic**（`services` 套件的核心轉帳測試），而且轉帳鎖定順序未固定、SQLite 模式下的列鎖是被 GORM 靜默丟棄的空操作 —— 這兩點直接影響「金額正確性」，在接上 AI 前端、開放更多流量之前必須優先處理。整體判斷：核心轉帳邏輯的「單元測試層面」設計思路正確（decimal、事務、審計軌跡都在），但「測試本身跑不動」+「併發鎖在非 Postgres 環境下失效」+「文件與現實脫節」是這一輪最需要處理的三個主軸。

---

## 2. 發現清單

嚴重度：Critical（可直接造成資金損失/資安破口/服務完全跑不動）／High（高機率被觸發或影響面大）／Medium（有條件觸發或影響前端整合）／Low（衛生/品質問題）。

| # | 嚴重度 | 類別 | 位置 | 問題 | 建議修法 |
|---|--------|------|------|------|----------|
| M1 | **Critical** | 金流-併發 | `services/transaction_service.go:57-64` | `Transfer(fromID, toID, ...)` 一律先鎖 `fromID` 再鎖 `toID`，鎖定順序取決於呼叫參數順序而非固定規則。A→B 與 B→A 兩筆轉帳同時發生時（在 Postgres 真的有 row lock 的情況下），會形成經典鎖等待環，造成 deadlock。 | 鎖定前先依 `min(fromID, toID)`／`max(fromID, toID)` 排序，永遠先鎖 ID 較小的錢包，兩個方向都會用相同鎖定順序。 |
| M2 | **Critical** | 金流-併發 | `repositories/wallet_repository.go:30-43`；驗證於 `gorm.io/driver/sqlite@v1.6.0/sqlite.go:124-129` | GORM 官方 sqlite driver 原始碼明確寫著 `// SQLite3 does not support row-level locking. return` —— 也就是說 `clause.Locking{Strength:"UPDATE"}` 在 SQLite 模式下會被**靜默丟棄**，`GetWalletByUserIDWithTx` 完全沒有真正鎖到任何東西。而 SQLite 是本專案在 `config.yaml` 讀取失敗、或 `db_driver` 未設為 `postgres` 時的**預設 fallback**（`db_conn/conn.go:18-24`，本次稽核在乾淨環境下執行 `go test` 時就親身重現：讀不到 config → 自動落到 SQLite）。在 deferred transaction 下，兩筆並發轉帳都能各自讀到轉帳前餘額，其中一筆提交後、另一筆才寫入，會造成 lost update（餘額被覆蓋，總量不守恆）。 | 若要支援 SQLite 做為「可信賴」的部署模式：(a) 對寫入交易使用 `BEGIN IMMEDIATE`（立刻取得寫鎖）取代預設 deferred transaction，或 (b) 用應用層 mutex/樂觀鎖（version 欄位 + CAS）取代 `FOR UPDATE`，或 (c) 明確在文件與啟動時聲明「SQLite 模式僅供單機開發用，不保證併發正確性，正式環境一律要求 Postgres」，並在 `db_conn.InitDatabase()` 對 SQLite + 非 dev 環境加上顯式警告或拒絕啟動。 |
| M3 | **Critical** | 測試/金流 | `services/transaction_service_test.go:32-48`；`internal/test/test_helpers.go:16` | `go test ./... -race` 在 `services` 套件直接 **panic**（nil pointer dereference）。根因鏈：①`test.SetupTestDB()` 用 `sqlite.Open(":memory:")`，未設定 `SetMaxOpenConns(1)` 或 `cache=shared`，`tx := db_conn...MasterDB.Begin()`（transaction_service.go:43）向連線池借用連線時可能借到「另一個空的 in-memory DB」，導致 `GetWalletByUserIDWithTx` 回報 `from_user wallet not found`；②`TestTransfer_Success_ValidTransfer` 用 `assert.NoError(t, err)` 而非 `require.NoError`，錯誤發生後測試繼續往下執行，對一個是 `nil` 的 `*Wallet` 呼叫 `.Balance.String()` 造成整個測試 binary panic 中止（單獨執行 `-run TestTransfer_Success_ValidTransfer` 一樣會 panic，非測試順序問題，已驗證）。`IMPROVEMENTS_SUMMARY.md` 裡本來就寫著這批測試「needs DB initialization fix」，8 個多月來沒人修。 | 1) 測試 DB 改用檔案型 SQLite（temp file）或對 `:memory:` 設定 `db.DB().SetMaxOpenConns(1)`；2) 所有金流斷言一律用 `require`，一旦前置條件失敗立刻停止該測試而非讓後續程式碼對 nil 解參考；3) 把這個修復當作本輪最優先的 PR，之後才有意義新增測試。 |
| M4 | High | 金流-正確性（潛在） | `repositories/wallet_repository.go:30-43` vs `wallet_repository.go:45-51` | `GetWalletByUserIDWithTx(userID, tx)` 只用 `user_id` 查詢（未帶 `currency_id`），`Transfer()` 再事後檢查 `fromWalletLocked.CurrencyID != currencyID`。目前因為 `UserService.CreateUser` 只會建立「一顆」預設幣別錢包（`services/user_service.go:64-68`），所以每個 user 永遠只有一筆 wallet，這個缺陷被掩蓋。一旦之後新增多幣別錢包（README 已宣稱「Multi-currency wallet support」，schema 也已支援），`.First()` 撈到的可能不是你要轉帳的那顆幣別錢包，會出現「明明有 USDT 錢包卻回報 wallet not found」的假錯誤。 | 改成 `GetWalletByUserIDAndCurrencyWithTx(userID, currencyID, tx)`，查詢時直接帶 `currency_id` 條件並加鎖，而不是鎖了任意一筆再事後比對。 |
| M5 | Medium | 金流-完整性 | `models/transaction.go:39-49`；呼叫點 `services/transaction_service.go:94-95` | `GenerateSignature()` 在 `CreateTransaction`（GORM insert）之前呼叫，此時 `t.CreatedAt` 還是零值，所以同一個 `(FromUserID, Amount)` 組合每次產生的「簽名」其實完全相同、且不是任何形式的密碼學簽章（無私鑰、無 HMAC），純粹是展示用字串。目前註解已誠實寫「simulated signature for demonstration」，風險可控，但若未來把 Signature 當作完整性依據就會出問題。 | 若只是 demo 用途：文件明確標註「非真實簽章」；若要延伸成真正的交易簽章功能，改為對 hash 做 HMAC-SHA256（使用伺服器密鑰）或走非對稱簽章，且必須在 hash/ID 產生**之後**才計算。 |
| M6 | High | 金流-輸入驗證 | `models/transfer_request.go`；`utils/decimal.go:18-25` | `Amount` 只驗證 `binding:"required"` + `IsPositive()`，沒有限制小數位數（幣別的 `Decimals` 欄位存在但轉帳時完全沒用來檢查）、沒有上限金額檢查。可以轉帳 `0.123456789`（9 位小數，超過 `decimal(20,8)` 精度，會被資料庫靜默四捨五入/截斷，需確認 GORM+driver 的實際行為）或天文數字金額。 | Service 層在扣款前驗證 `amount.Exponent() >= -currency.Decimals`（或用 `amount.Round(currency.Decimals).Equal(amount)` 判斷），並加上業務可接受的最大單筆金額檢查。 |
| S1 | **Critical** | 安全-密鑰外洩 | `config.yaml:6,12`（已 git 追蹤） | JWT 簽章密鑰 `your-secret-key-change-in-production-min-32-chars` 與 Postgres 密碼 `secret` 以明文提交在版控中的 `config.yaml`。任何拿到這個 repo（包含即將加入的前端協作者/AI agent）的人都能離線偽造任意 `user_id` 的合法 JWT。 | 立刻視同已外洩並輪替：改由環境變數/密鑰管理服務注入，`config.yaml` 移出版控只留 `config.yaml.example`（值用佔位符），並在 `.gitignore` 補上規則；若此 repo 曾經公開過，兩組密鑰都要視為已洩漏並更換。 |
| S2 | High | 安全-JWT | `router/router.go:23-27`；`internal/config/loader.go:18-20` | 當 `config.Config.JWTSecret` 為空字串時，`router.go` 會 fallback 成寫死在原始碼裡的 `default-secret-key-change-in-production-min-32-chars`；而 config 讀取失敗只會 `log.Printf` 警告、不會 fatal（`loader.go:19`），代表「忘記設定環境變數」時服務不會啟動失敗，而是靜默用一組任何人 clone 原始碼都看得到的密鑰簽發 JWT。 | 沒有設定 `JWTSecret`（或長度 < 32）時應該直接 `log.Fatal`，不要有預設密鑰這種 fallback。 |
| S3 | High | 安全-文件與現實不符 | `middleware/ratelimit.go` 全檔；`router/router.go`（全檔搜尋不到任何 `RateLimit` 字樣） | Rate limiting 中介層寫好了但**從未在 `router.go` 掛載**，全 repo 搜尋 `RateLimitMiddleware`/`DefaultRateLimit` 只有定義沒有呼叫。README 卻明確宣稱「Rate Limiting: Token bucket algorithm ... Prevents API abuse and simple DDoS attempts」為已完成功能。目前 `/auth/login`、`/users`、`/tx/:hash` 全部沒有任何速率限制。 | 在 `router.go` 對至少 `/auth/login`、`/users`、`/wallet/transfer` 掛上 `middleware.DefaultRateLimit()`（或依端點給不同的限流策略），並同步更新 README 只在真的掛上之後才宣稱完成。 |
| S4 | Medium | 安全-限流可被繞過 | `middleware/ratelimit.go:49`；全 repo 搜尋不到 `SetTrustedProxies` | 就算 S3 修好，`RateLimitMiddleware` 是用 `c.ClientIP()` 當 key；Gin 從未呼叫 `SetTrustedProxies`，預設會信任所有代理層解析 `X-Forwarded-For`/`X-Real-IP`。若服務直接暴露在網路上（前面沒有會覆寫該表頭的可信反向代理），攻擊者可自帶 `X-Forwarded-For` 標頭無限繞過 IP 限流。 | 明確呼叫 `r.SetTrustedProxies(nil)`（不信任任何代理，直接用 socket peer IP）或設定成實際反向代理的真實 IP／CIDR。 |
| S5 | High | 安全-CORS 缺失 | `router/router.go` 全檔（無任何 CORS 設定，`go.mod` 未引入 `gin-contrib/cors`） | 完全沒有 CORS 中介層。之後接上瀏覽器端前端時，所有跨來源請求會被瀏覽器擋下，前端串接會直接失敗。 | 導入 `gin-contrib/cors`，白名單列出前端實際 origin（不要對帶憑證的請求用 `*`），並讓允許的 headers 涵蓋 `Authorization`。 |
| S6 | Medium（需確認） | 安全-越權/資料揭露 | `handlers/transaction_handler.go:112-133`；最新一次 commit `80e6220` 剛把 `/tx/:hash` 從受保護路由移出來變成公開端點 | `/tx/:hash` 不需登入即可查詢，回傳內容包含 `from_user_id`、`to_user_id`、`amount`（`models/transaction_dto.go:14-23`）。Hash 是 `SHA256(fromID|toID|amount|UnixNano())`，實務上無法暴力枚舉，風險可控；但這是「剛剛」才從受保護改成公開，且沒有任何限流（見 S3），也沒有文件說明這是刻意設計（類似區塊鏈瀏覽器的「拿到 hash 才能查」模式）還是疏漏。 | 需向產品/需求方確認：這是否為刻意的「類公開帳本」設計？若是，補上限流與文件說明；若非刻意，改回要求登入並驗證呼叫者是交易雙方之一。 |
| S7 | Medium | 安全-錯誤語意 | `handlers/user_handler.go:46-50` | 建立使用者失敗（含最常見的「使用者名稱/Email 重複」）一律回傳 `500 Internal Server Error` + 固定字串 `"failed to create user"`。正常的業務衝突（重複帳號）跟真的伺服器錯誤混在一起，前端無法分辨，監控也會被大量「假 500」淹沒。 | Service 層偵測 unique constraint 違反（或先查一次是否存在），回傳明確的 `409 Conflict` + `USER_ALREADY_EXISTS` code。 |
| S8 | High | API 契約-錯誤格式不一致 | `models/error_response.go`、`internal/errors/codes.go`（定義了但全 repo 搜尋不到任何呼叫端） vs. 各 handler 手刻的 `gin.H{"error": err.Error()}` vs. `middleware/validator.go:67-71`（多了 `code`+`details`）vs. `middleware/ratelimit.go:56-59`（多了 `code`，格式又不同） | 專案裡其實已經設計了統一錯誤格式（`ErrorResponse` + `ErrCode*` 常數），但**完全沒被使用**。實際每個端點回傳的錯誤 JSON 形狀都不一樣：有的只有 `error`，有的有 `error`+`code`，有的還有 `details` 陣列。前端無法寫一支通用的錯誤處理器。 | 統一改成全部 handler 都回傳 `models.ErrorResponse{Error, Code, Message}`，把 `internal/errors` 的常數實際接上每個回傳點；這是**接前端前必須先做**的事（見第 3 節）。 |
| S9 | Low | 安全-錯誤洩漏 | `services/transaction_service.go:128-130` | `tx.Commit()` 失敗時，原始 GORM/DB 錯誤直接透過 `err.Error()` 一路回傳到 HTTP 層，可能洩漏資料表/欄位/約束名稱等內部資訊。 | Service 層對 commit 失敗回傳固定的「transaction failed, please retry」，把原始錯誤只寫進 log（帶 trace id）。 |
| S10 | Low（需確認） | 安全-輸入未過濾 | `middleware/trace.go:14-17` | 客戶端可自帶 `X-Trace-ID` 標頭，內容原封不動被塞進 log 與回應標頭，沒有格式/長度驗證。目前只是回顯，風險低，但如果之後把 trace id 寫進會被下游系統解析的 log pipeline，就有 log injection 疑慮。 | 驗證格式（如 UUID）或長度上限，不符合格式就自行產生新的。 |
| S11 | Medium | 安全-暴力破解 | `handlers/user_handler.go:68-95`；密碼規則 `models/user_dto.go:11`（`min=6`，無複雜度要求） | `/auth/login` 沒有失敗次數鎖定，疊加 S3（限流未掛載），弱密碼（僅要求 6 碼）容易被暴力破解。 | 掛上限流（S3）＋針對登入端點做更嚴格的每帳號/每 IP 節流；評估提高密碼最小長度或要求複雜度。 |
| S12 | Medium（需確認） | 安全-Token 生命週期 | `internal/auth/jwt.go` 全檔；找不到任何 logout / token 黑名單邏輯；`redis_addr` 設定存在但整個 repo 沒有任何 Redis client 程式碼 | JWT 固定 24 小時效期、沒有 refresh token、沒有登出端點、沒有撤銷機制。`config.yaml` 註解寫著 Redis「用於分散式鎖和速率限制」，但程式碼裡完全沒有用到 Redis —— 純屬尚未實作的規劃欄位。 | 若要做 token 撤銷，用 Redis 存黑名單（順便解決 A4 提到的跨副本限流問題）；否則至少在文件中說明目前無登出/撤銷能力是已知限制。 |
| A1 | Medium | 架構-文件不符 | `README.md:76-79` vs. S3 | README 宣稱 Rate Limiting 已實作並生效，實際未掛載。 | 見 S3；文件與程式碼要一起改。 |
| A2 | Medium | 架構-文件不符 | `README.md:16`（「Swagger (OpenAPI 3.0 docs)」）vs. `docs/swagger.json:2`（`"swagger": "2.0"`，程式化驗證過） | 實際產出的是 **Swagger 2.0**，不是 OpenAPI 3.0。這會影響前端要選哪套型別產生工具（例如 `openapi-typescript` 需要 3.0，2.0 要用 `swagger-typescript-api` 或先用 `swagger2openapi` 轉檔）。 | 更新 README 用詞為「Swagger 2.0 (OpenAPI v2)」；若前端工具鏈需要 3.0，評估把 `swaggo` 換成支援 OpenAPI 3 的產生器，或加一道 `swagger2openapi` 轉換步驟。 |
| A3 | Low | 架構-文件不符 | `README.md:199-206`（引用 `docker-compose.kafka.yml`） | 該檔案不存在於 repo（`git ls-files` 確認只有 `docker-compose.yml`）。照 README 步驟操作會直接失敗。 | 要嘛把 `docker-compose.yml` 改名/補一份 `docker-compose.kafka.yml`，要嘛修正 README 指令。 |
| A4 | Low | 架構-死設定 | `internal/config/config.go:9`（`RedisAddr`）；全 repo 搜尋不到任何 redis client 引用 | `redis_addr` 有欄位、有註解，但沒有任何程式碼真的連 Redis。多副本部署時，`middleware/ratelimit.go:14-19` 的限流狀態是存在單一 process 記憶體裡的 map，多個 API 副本之間完全不會同步。 | 若要橫向擴展，導入 Redis 做分散式限流/鎖（如 `redis_rate` 或自製 token bucket in Redis）；否則從設定與文件移除，避免誤導。 |
| A5 | Low | 架構-死代碼 | `middleware/validator.go`（`ValidationMiddleware`/`HandleValidationError`/`FormatValidationError`）、`utils/decimal.go`（`DecimalFromFloat`/`DecimalFromString`）— 皆搜尋不到任何呼叫端 | 多處基礎設施寫好但沒接上：驗證中介層是個空的 passthrough 且沒被 `Use()`；`FormatValidationError`/`HandleValidationError` 沒有任何 handler 呼叫；`utils` 的兩個 decimal 轉換函式零呼叫。 | 要嘛實際接上（例如把驗證錯誤處理統一走 `HandleValidationError`，呼應 S8），要嘛刪除，避免新加入的人（含 AI agent）誤以為這些機制生效中。 |
| A6 | High | 架構-層次破壞 | `services/transaction_service.go:11`（`import "testing"`）、`166-190`（`TransferWithLockOption`） | 正式（非 `_test.go`）原始碼直接 `import "testing"`，並公開一個帶 `*testing.T` 參數、內含「刻意不安全」轉帳邏輯的 exported method，只被 `internal/test/concurrency_demo_test.go` 呼叫。這把測試專用工具混進編譯後的正式產出物，也破壞了 service 層該有的單一職責。 | 把 `TransferWithLockOption` 連同其未鎖定示範邏輯搬到 `services/transaction_service_demo_test.go`（測試檔），正式 `TransactionService` 只保留 `Transfer`。 |
| A7 | Medium | 架構-不一致 | `db_conn/sqlite.go:7-18`（`modernc.org/sqlite`，純 Go）vs. `internal/test/test_helpers.go:9,16`（`gorm.io/driver/sqlite` 預設走 `mattn/go-sqlite3`，需要 cgo） | 正式環境與測試環境用的是兩個不同的 SQLite driver 實作，測試沒有真正驗證正式環境會用到的 driver 行為（例如 M2 提到的鎖定行為兩者可能不同，需確認）。 | 統一使用同一個 driver（建議測試也改用 `modernc.org/sqlite` 對應的 dialector），確保測試環境貼近正式環境。 |
| Q1 | **Critical** | 工程品質 | 見 M3 | `go test ./... -race` 目前會 panic，等同沒有可信賴的 CI gate。 | 同 M3；建議加一支 GitHub Actions workflow 在每個 PR 跑 `go build ./... && go vet ./... && go test ./... -race`，本次稽核未在 repo 中找到任何 `.github/workflows`（需確認是否用其他 CI 系統）。 |
| Q2 | High | 工程品質-版控衛生 | `git ls-files` 確認：`main/__debug_bin2424212485`（54MB，今天日期，明顯是 IDE debug session 產物）、`wallet-api`（55MB） | 兩個編譯後執行檔被提交進版控，合計約 109MB。`.gitignore` 只擋了 `*.exe/*.dll/*.so/*.dylib/*.test/*.out`，沒有規則擋「無副檔名的 Go binary」，這正是它們漏網的原因。 | `git rm --cached main/__debug_bin2424212485 wallet-api`，並在 `.gitignore` 補上 `/wallet-api`、`**/__debug_bin*`。若要徹底瘦身歷史記錄，評估 `git filter-repo`（需與使用者確認是否要重寫歷史）。 |
| Q3 | Medium | 工程品質-版控衛生 | `.idea/.gitignore`、`.idea/mini-crypto-wallet-api.iml`、`.idea/modules.xml`、`.idea/vcs.xml` 皆被 `git ls-files` 列為已追蹤，但 `.gitignore:33` 已寫 `.idea/` | IDE 設定檔在加入 `.gitignore` 規則前就已經被提交，規則對既有已追蹤檔案沒有追溯效果。 | `git rm --cached -r .idea .vscode`（若 `.vscode` 也有殘留）。 |
| Q4 | Low | 工程品質-設定矛盾 | `.gitignore` 同時列了 `README.md` 與 `docs/`，但兩者都是目前主動維護、已追蹤的核心檔案（README、Swagger 文件） | 目前無實害（已追蹤檔案不受 `.gitignore` 影響），但規則本身矛盾、容易誤導未來貢獻者，且會讓「`docs/` 下未來新增的檔案」被意外忽略。 | 從 `.gitignore` 移除這兩條規則。 |
| Q5 | High | 工程品質-容器安全 | `Dockerfile` 全檔 | 單一階段建置，最終映像檔基於 `golang:1.24-bullseye`（含完整 Go 工具鏈，體積大），沒有 `USER` 指令，容器內以 root 執行 `./wallet-api`。 | 改成 multi-stage：builder stage 用 `golang:1.24` 編譯，最終 stage 用 `gcr.io/distroless/base-debian12` 或 `alpine`，新增非 root user（`USER 65532:65532` 或自建 user）後再 `COPY --from=builder`。 |
| Q6 | High | 工程品質-環境可重現性 | `docker-compose.yml` 全檔（只有 `zookeeper`/`kafka`/`postgres`，沒有 `api` 服務） | 無法「一鍵」啟動完整環境；必須額外手動 `docker build` + `docker run` API 並自行組所有環境變數（README 有寫但屬手動步驟）。 | 在 `docker-compose.yml` 新增 `api` service（`build: .`，`depends_on` 含健康檢查），讓 `docker compose up` 真的能起完整可用環境。 |
| Q7 | Medium | 工程品質-部署 | `main/main.go:43`（`r.Run("localhost:8080")`） | 綁定在 `localhost`，在容器內只接受 loopback 連線；一旦照 Q6 建議把 API 放進 compose 並做 port mapping，外部連不進去。 | 改成 `r.Run("0.0.0.0:8080")` 或 `r.Run(":8080")`。 |
| Q8 | Medium | 工程品質-穩定性 | `main/main.go` 全檔 | 沒有訊號處理（SIGTERM/SIGINT）、沒有 graceful shutdown、沒有 `defer producer.Close()`；部署/重啟時可能中斷處理中的請求或遺失尚未寫出的 Kafka 訊息。 | 改用 `http.Server` + `srv.Shutdown(ctx)`，收到中止訊號時先停止接新連線、等現有請求完成、再關閉 Kafka producer。 |
| Q9 | Medium | 測試涵蓋率 | `services/*_test.go`、`internal/test/*.go` | 目前只有 `services/` 有測試（12 條，且核心那條會 panic），`handlers/`、`middleware/`、`repositories/`、`router/`、DTO 轉換函式完全零測試。詳細建議清單見下方。 | 見第 4 節「建議新增測試」。 |
| Q10 | Low | 測試穩定性 | `internal/test/concurrency_demo_test.go:20-23` | `TestConcurrentTransfers` 會呼叫 `config.LoadConfig()` + `db_conn.InitDatabase()`，因為 `config.yaml` 設定 `db_driver: postgres`，在沒有本機 Postgres 的環境（例如乾淨的 CI runner 或新同事的筆電）會直接 `log.Fatal` 整個測試 process 退出。 | 讓這支測試改用 SQLite in-memory 或 testcontainers-go 啟一個一次性 Postgres，不要依賴開發者本機剛好有服務在跑。 |

---

## 3. 接前端之前「必須先修」的項目

依「不修就會讓前端串接卡住或產生資安事故」排序：

1. **S1／S2** 輪替並移除硬編碼的 JWT 密鑰與資料庫密碼，改走環境變數；config 缺密鑰時要 fail fast，不要有預設密鑰。
2. **S8** 統一錯誤回應格式（`{error, code, message}`），把 `internal/errors` 的 code 常數真正接上每個 handler —— 前端要靠穩定的 `code` 做條件分支（例如顯示「餘額不足」vs「請重新登入」），現在的格式因端點而異。
3. **S5** 加上 CORS 中介層並白名單前端 origin，否則瀏覽器端請求會被直接擋掉，前端團隊會誤以為是自己的 bug。
4. **M3／Q1** 先修好會 panic 的測試（連帶查一下有沒有 CI 在跑），否則之後每次改動都無法確定有沒有壓垮既有邏輯。
5. **M1** 修正轉帳鎖定順序（依 user ID 排序），這是資安/資金正確性議題，越晚修、之後有真實資料要遷移就越麻煩。
6. **S3** 把已經寫好的限流中介層實際掛到 `/auth/login`、`/users`、`/wallet/transfer`。
7. **Q6／Q7** 讓 `docker compose up` 真的能起完整可用的環境（含 API 本身），並修正綁定位址，這樣前端工程師/AI agent 才能在本機一鍵跑起整套系統做串接測試。
8. **M2** 至少在文件與啟動 log 明確警告「SQLite 模式不保證併發正確性」，如果 demo/前端測試環境預設用 SQLite，這點必須讓使用者知情。

---

## 4. 建議的後端延伸功能（依對 demo 的價值排序）

1. **多幣別錢包完整支援**（建立錢包 API、依幣別查詢/列出所有錢包、轉帳前校驗雙方幣別一致）—— README 已經「宣稱」有這個功能，實際做出來能立刻讓宣稱與現實一致，也是最容易在 demo 裡展示「這是真的加密貨幣錢包」的功能。做這個之前務必先修 M4。
2. **交易狀態機 + 冪等鍵（Idempotency Key）**——`Transaction.Status` 欄位已經定義了 `pending/processing/completed/failed/cancelled` 但目前永遠直接寫 `completed`，等於欄位形同虛設；加上冪等鍵可以直接展示「前端重試不會重複扣款」，是金流 demo 裡評審最愛問的題目之一，且能順便補上 M3/M1 修完後的完整測試案例。
3. **統一分頁 + 可查詢的交易列表 API 擴充**（篩選日期區間、幣別、方向 in/out）——現有 `PaginationRequest/Response` 已經設計得不錯，延伸出去對前端做交易紀錄頁面很直接有價值。
4. **Webhook / SSE 即時通知**——串接現有的 Kafka `tx.created` 事件，加一個訂閱端點（SSE 或 WebSocket）讓前端能即時看到轉入通知，比單純輪詢 API 更有「即時錢包」的展示效果，且能順便解決 Kafka 發送失敗目前只寫 log 沒有其他補救的可觀測性缺口（可以在這個功能裡順便加 outbox 表）。
5. **登出/Token 撤銷（搭配 Redis）**——`redis_addr` 設定已經存在但沒用到，做一個 Redis-backed 的 token 黑名單，同時可以把 S4/A4 提到的「多副本限流狀態不同步」一併解決，一魚兩吃。
6. **管理端/風控儀表板用的統計 API**（例如某使用者 24 小時內轉帳總額、大額轉帳告警）——展示對金融系統的風控意識，且不需要改動核心轉帳邏輯，風險低、demo 效果高。

---

## 5. 建議新增的 CLAUDE.md 內容草稿

```markdown
# CLAUDE.md

## 專案速覽
- Go + Gin + GORM，分層：handlers → services → repositories → db_conn。
- 金額一律使用 `github.com/shopspring/decimal`，禁止在任何金流路徑（Wallet.Balance、
  Transaction.Amount、BalanceHistory 相關欄位）使用 float32/float64。
- DB 驅動由 `config.yaml` 的 `db_driver` 控制：`postgres`（正式）／SQLite（開發 fallback）。
  **SQLite 模式目前不保證併發正確性**（GORM sqlite driver 會靜默丟棄 `FOR UPDATE`，
  見 docs/AUDIT.md #M2）——寫任何牽涉併發轉帳的程式碼前，先確認你是對著哪個 driver 測試的。

## 開發前必看
- 執行 `go build ./... && go vet ./... && go test ./... -race` 三件套再提 PR。
  截至上次稽核，`services` 套件的測試會 panic（docs/AUDIT.md #M3），修好之前新功能
  的 PR 請至少額外針對自己改到的路徑手動驗證，不要假設既有測試會擋住回歸。
- 不要提交編譯產物：`wallet-api`、`main/__debug_bin*`、`*.db` 一律不進版控。
  如果不小心 `go build` 產生了執行檔在專案目錄下，記得 `git status` 確認沒被加進暫存區。
- `config.yaml` 內含真實密鑰/密碼，這份檔案目前在版控中屬於歷史包袱，
  **修改前先跟專案owner確認是否已完成密鑰輪替**；新的密鑰一律走環境變數，不要寫回 yaml。

## 轉帳邏輯的兩條鐵律
1. 任何會同時鎖兩顆錢包的程式碼，鎖定順序永遠依 `user_id` 由小到大，不可依呼叫參數順序。
2. `Transfer()` 系列方法禁止繞過 `walletRepo.GetWalletByUserIDWithTx`（或其未來的
   currency-aware 版本）直接讀寫 `Wallet.Balance`。

## 錯誤回應規範（若尚未統一，這是目標狀態）
所有 handler 錯誤一律回傳 `models.ErrorResponse{Error, Code, Message}`，`Code` 從
`internal/errors` 的常數挑選，不要手刻 `gin.H{"error": ...}`。

## 明確禁止事項
- 不要在非 `_test.go` 的檔案裡 `import "testing"`。
- 不要新增只寫一半就沒接上路由/呼叫端的中介層或 util 函式；如果暫時用不到，先別提交。
- 不要修改 `.gitignore` 移除 README.md / docs/ 之外的既有規則，除非你知道自己在做什麼
  （目前這兩條規則本身是誤植，正在等待清理，見 docs/AUDIT.md #Q4）。
- 未經使用者確認，不要對 git 歷史做 filter-repo / force-push 之類的重寫操作。

## 常用指令
- 建置：`go build ./...`
- 靜態檢查：`go vet ./...`
- 測試（含競態檢測）：`go test ./... -race`
- 產生 Swagger：`swag init -g main/main.go`（目前輸出是 Swagger 2.0，不是 OpenAPI 3.0）
- 本機啟動基礎設施：`docker compose up -d`（目前只會起 zookeeper/kafka/postgres，
  API 本身要另外 `go run ./main` 或 `docker build && docker run`，見 docs/AUDIT.md #Q6）
```

---

*本報告所有行號依稽核當下（commit `80e6220`）的檔案內容為準；後續若有變動請重新核對。標註「需確認」的項目代表稽核者無法單靠讀程式碼得出結論，建議與產品/需求方或原作者確認意圖。*
