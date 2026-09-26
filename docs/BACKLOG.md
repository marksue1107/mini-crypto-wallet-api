# Backlog

未處理與部分修正的稽核/驗證發現，整理自 `docs/AUDIT.md`、`docs/REMEDIATION_REPORT.md`、
`docs/VERIFICATION_REPORT.md`。每一項的格式設計成可以直接複製貼上開一個 GitHub
Issue（標題 → Issue 標題，其餘內容 → Issue 內文）。目前因為 `gh` CLI 無法在本機
安裝，所以先整理在這裡，之後要開 Issue 時直接複製即可。

`[FE*]` 開頭的項目來源不同：整理自 `docs/FRONTEND_SPEC.md` §5「後端目前不支援的
功能」，是前端規格制定過程中發現的**功能缺口**，不是稽核發現的缺陷——
`docs/BACKEND_PREP_PLAN.md` 已經把其中影響前端第一版最深的四項（轉帳回應缺
`hash`、error code 過於籠統、缺統計/上限查詢 API）處理掉了，這裡列的是**還沒
處理、前端第一版會繞過或隱藏**的部分。

---

## [M5] Transaction.Signature 不是真正的密碼學簽章

**嚴重度**：Medium
**位置**：`api/models/transaction.go:45-49`（`GenerateSignature()`）；呼叫點
`api/services/transaction_service.go:175`（在 `CreateTransaction` insert 之前呼叫）

**問題描述**：
`GenerateSignature()` 在 GORM insert 之前呼叫，此時 `t.CreatedAt` 還是零值，所以
同一個 `(FromUserID, Amount)` 組合每次產生的「簽名」其實完全相同（`SIG-<FromUserID>-<Amount>-<CreatedAt.UnixNano()>`
中的 `CreatedAt` 恆為零值），而且整個機制沒有私鑰、沒有 HMAC，不是任何形式的
密碼學簽章，純粹是展示用字串。目前程式碼註解已誠實寫「simulated signature for
demonstration」，風險可控，但若未來把 `Signature` 當作完整性/防偽依據，就會出
真正的問題。

**建議修法**：
- 若只是 demo 用途：在 API 文件（Swagger 註解、README）明確標註「非真實簽章，
  僅供展示」。
- 若要延伸成真正的交易簽章功能：改為對 hash 做 HMAC-SHA256（使用伺服器密鑰，
  例如複用/衍生自 `JWT_SECRET` 或獨立的 `SIGNING_SECRET`），或走非對稱簽章
  （ed25519/ECDSA），且必須在 `hash`/`ID` 產生**之後**才計算，不能在 insert 前
  用還是零值的欄位。

---

## [S10] X-Trace-ID 標頭未做格式/長度驗證

**嚴重度**：Low（需確認）
**位置**：`api/middleware/trace.go:8-20`

**問題描述**：
客戶端可自帶 `X-Trace-ID` 標頭，內容原封不動被塞進 log（`TraceIDKey`）與回應
標頭（`c.Header(TraceIDHeader, traceID)`），沒有任何格式或長度驗證。目前只是
回顯，風險低，但如果之後把 trace id 寫進會被下游系統解析的 log pipeline（例如
ELK、結構化查詢），就有 log injection 或格式混亂的疑慮。

**建議修法**：
驗證格式（例如要求是合法 UUID）或加上長度上限；不符合格式時忽略客戶端帶的值，
改為伺服器自行產生一個新的 trace id。

---

## [S11] 登入端點沒有帳號層級的失敗次數鎖定（部分修正）

**嚴重度**：Medium
**位置**：`api/handlers/user_handler.go`（`Login`）；密碼規則
`api/models/user_dto.go:11`

**問題描述**：
已完成的部分：密碼最小長度已從 6 提高到 8（`min=8`），`/auth/login` 也已經在
`router.go` 掛上比一般端點更嚴格的限流（10 req/min、burst 5，依 client IP）。
但這只是**依 IP 節流**，沒有實作**依帳號**的失敗次數鎖定或漸進式延遲——攻擊者
若控制多個來源 IP（或走殭屍網路），仍然可以對單一帳號做分散式暴力破解，不會被
目前的限流機制擋下。

**建議修法**：
在 `UserService.Login`（或更上層）針對「使用者名稱」而非「來源 IP」記錄連續
失敗次數，超過門檻後對該帳號本身加上鎖定時間（例如連續 5 次失敗鎖定 15
分鐘），並在鎖定期間回傳明確但不洩漏帳號是否存在的錯誤訊息。若要支援多副本
部署，這個計數狀態建議跟 [S12]/[A4] 的 Redis 需求一起規劃。

---

## [S12] JWT 沒有撤銷/登出/refresh 機制

**嚴重度**：Medium（需確認）
**位置**：`api/internal/auth/jwt.go` 全檔；全 repo 搜尋不到任何 logout 端點或
token 黑名單邏輯

**問題描述**：
JWT 固定 24 小時效期、沒有 refresh token、沒有登出端點、沒有撤銷機制。使用者
登出後舊 token 在效期內依然完全有效；帳號被盜用時也無法主動讓已核發的 token
失效。`config.yaml.example` 裡的 `redis_addr` 設定暗示原本規劃要用 Redis 做這件
事，但目前完全沒有對應程式碼——純屬尚未實作的規劃欄位。

**建議修法**：
若要做 token 撤銷，導入 Redis 存黑名單（`jti` 或 `user_id+issued_at` 為 key，
TTL 設為剩餘效期），登出端點把當前 token 加入黑名單，`AuthMiddleware` 驗證時
多查一次黑名單。這個 Redis 依賴可以跟 [A4] 提到的跨副本限流狀態共用同一個
Redis 實例規劃。若近期不做，至少要在 README／API 文件寫清楚「目前沒有登出/
撤銷能力」這個已知限制，避免前端誤以為登出端點存在。

---

## [A4] Redis 設定是死欄位，限流狀態無法跨副本同步

**嚴重度**：Low
**位置**：`api/internal/config/config.go:9`（`RedisAddr` 欄位）；
`api/middleware/ratelimit.go:20`（`limiters map[string]*rate.Limiter`，純
process 記憶體內狀態）

**問題描述**：
`redis_addr` 有設定欄位、有註解，但全 repo 搜尋不到任何 Redis client
程式碼——沒有任何地方真的連上 Redis。目前的限流狀態（`RateLimiter.limiters`）
是存在單一 process 記憶體裡的 map，如果之後要水平擴展成多個 API 副本，各副本
的限流計數完全不會同步，等於每個副本各自有一份獨立的配額，實質上限流上限會
隨副本數量等比例放大，可能被繞過設計意圖。

**建議修法**：
若要橫向擴展：導入 Redis 做分散式限流（例如 `redis_rate` 套件，或自製
token-bucket in Redis 的 Lua script），跟 [S12] 的 token 黑名單共用同一個
Redis 實例。若目前規模（單副本/demo）不需要橫向擴展：從設定與文件移除
`redis_addr`，避免讓人誤以為這是已啟用的功能。

---

## [Q9] repositories/ 與部分 handler 仍無專屬單元測試（部分修正）

**嚴重度**：Medium
**位置**：`api/repositories/`（`git ls-files 'api/repositories/*_test.go'` 無結果）；
`api/handlers/`（目前只有 `user_handler_test.go`，`wallet_handler.go`、
`currency_handler.go`、`transaction_handler.go` 都沒有專屬 handler 層測試）

**問題描述**：
本輪已經幫先前零測試的套件（`internal/auth`、`internal/config`、
`middleware`、`router`、`handlers/user_handler`）補上測試，核心金流
（`services/`）與新增的安全性關卡（限流、CORS、統一錯誤格式、JWT fail-fast）
也都有測試覆蓋。但 `repositories/` 這一層（`wallet_repository.go`、
`user_repository.go`、`transaction_repository.go`、`currency_repository.go`、
`balance_history_repository.go`）目前完全沒有直接針對 GORM 查詢邏輯的單元測試
——這一層目前只透過 `services/` 的測試間接覆蓋，若 repository 本身的查詢條件寫錯
（例如 [M4] 那種「查詢忘記帶 currency_id」的錯誤），不一定能被現有測試抓到。
`wallet_handler`/`currency_handler`/`transaction_handler` 同樣沒有 handler 層
的直接測試，目前只被 `api/e2e/` 的黑箱測試間接覆蓋。

**建議修法**：
針對 `repositories/` 補上使用真實（或 testcontainers）資料庫的整合測試，特別是
帶鎖定（`FOR UPDATE`）與多條件查詢（如 `GetWalletByUserIDAndCurrencyWithTx`）
的方法，確保之後任何人改查詢條件時，測試會直接反映在 repository 層而不必依賴
E2E。`wallet_handler`/`currency_handler`/`transaction_handler` 可以參考現有
`user_handler_test.go` 的模式（mock service 層）補上對應測試。

---

## [F3] 驗證錯誤回應手刻 gin.H，未複用 ErrorResponse struct

**嚴重度**：Low（既有問題，非本輪引入，V1 獨立審查發現）
**位置**：`api/middleware/validator.go:70`（`HandleValidationError`）

**問題描述**：
驗證錯誤分支目前是手刻 `gin.H{...}` 組出回應，而不是直接建構/複用
`models.ErrorResponse` struct 再加上 `details` 欄位。實際回傳的欄位名稱（`error`、
`code`、`message`、`details`）內容是對的，跟其他 handler 的錯誤格式一致，純屬
程式碼風格上的不一致，沒有造成行為錯誤。

**建議修法**：
改成先用 `models.NewErrorResponse(...)` 建構基礎錯誤物件，再疊加 `details`
欄位，與 `internal/errors.RespondError` 的其餘呼叫點風格一致，方便之後維護。

---

## [F4] RateLimiter.limiters map 無上限成長

**嚴重度**：Low（既有問題，非本輪引入，V1 獨立審查發現）
**位置**：`api/middleware/ratelimit.go:20`（`limiters map[string]*rate.Limiter`）

**問題描述**：
每個來源 IP（或其他限流 key）對應的 `rate.Limiter` 一旦建立就永遠留在
`limiters` map 裡，沒有任何清除機制。長時間運行、面對大量不同來源 IP（例如被
掃描或有大量真實使用者）時，這個 map 會隨著出現過的 IP 數量無上限成長，屬於
一種慢性記憶體洩漏。

**建議修法**：
加上 LRU 淘汰（例如 `hashicorp/golang-lru`）或定期背景清除超過一段時間沒有
活動的 entry；若之後導入 [A4] 的 Redis 分散式限流，這個問題也會隨之解決（狀態
搬到 Redis，可以設 TTL 自動過期）。

---

## [F5] GenerateHash 使用 UnixNano 而非密碼學亂數

**嚴重度**：Low（既有問題，非本輪引入，與本次修正無關，V1 獨立審查發現）
**位置**：`api/models/transaction.go:40`（`GenerateHash`）

**問題描述**：
`GenerateHash()` 用 `time.Now().UnixNano()`（非密碼學亂數）當作 hash 輸入的一
部分，來源是系統時鐘而非 CSPRNG。以目前的流量規模來說,實務上唯一性沒有問題,
但這不是密碼學意義上的不可預測值，如果之後把這個 hash 當作需要抵抗刻意碰撞
攻擊的識別碼，會有風險。

**建議修法**：
若未來需要更強的不可預測性，改用 `crypto/rand` 產生的亂數（或額外拌入）取代
/補強 `UnixNano()`，或改用資料庫自動遞增 ID + UUID 的組合。

---

## [F9] M3 迴歸測試只在整包執行時才穩定重現（資訊性，非缺陷）

**嚴重度**：資訊性（不影響 CI，V2 驗證發現）
**位置**：`api/services/`（測試套件本身的執行方式，非特定程式碼行）

**問題描述**：
[M3] 修正的迴歸測試（SQLite 測試資料庫從 `:memory:` 改為檔案型）若用
`go test -run <單一測試名稱>` 只挑一個測試執行，不會穩定重現撤銷修正後的失敗；
必須整個 `services` package 一起跑（`go test ./services/...`）才會穩定重現。
這不影響「CI 會抓到回歸」的結論——CI 本來就是整包執行——但如果之後有人想「只跑
跟我這次改動有關的單一測試」來省時間，可能會因為只跑單一測試而誤以為這個
迴歸點已經修好、沒有問題。

**建議修法**：
無強制動作。若要更嚴謹，可以在測試檔案或 `CLAUDE.md` 加一句提醒：「這個
package 的測試之間有共享狀態依賴（SQLite 連線池行為），除錯時建議整包執行,
不要只用 `-run` 篩選單一測試」。

---

## [F10] KafkaProducer.Close() 成功時不輸出 log

**嚴重度**：Low（V4 驗證發現）
**位置**：`api/kafka_client/producer.go:57`（`Close`）

**問題描述**：
`docker compose stop api` 時可以從 log 看到 graceful shutdown 的排水流程
（`shutdown signal received` → `server stopped cleanly`），但 `KafkaProducer.
Close()` 成功關閉時完全不輸出任何 log（目前的實作只有失敗時才 log 錯誤）。
這代表從 log 沒辦法「直接」確認 Kafka producer 真的被正常關閉，只能從「沒有
出現任何錯誤 log」加上讀程式碼（`main.go` 的 `defer producer.Close()`）間接
推論。純屬可觀測性小改善，不影響正確性。

**建議修法**：
在 `Close()` 成功關閉時補一行 info 等級的 log（例如
`log.Println("✅ Kafka producer closed cleanly")`），讓 graceful shutdown 的
每個步驟在 log 裡都有直接對應的證據。

---

## [FE1] 沒有列出使用者所有幣別錢包的 API

**嚴重度**：Low（功能缺口，非缺陷）
**位置**：`api/handlers/wallet_handler.go`（`GetWallet` 底層呼叫
`GetWalletByUserID(userID)`，只用 `user_id` 查詢，不帶 `currency_id`）

**問題描述**：
資料庫 schema 已經支援一個使用者對每個幣別各有一顆錢包（`wallets` 表以
`(user_id, currency_id)` 為鍵），但沒有任何端點可以「列出某使用者名下所有
錢包」。`GET /wallet/{user_id}` 目前只用 `user_id` 查詢單一筆，一旦使用者
真的持有超過一顆錢包，回傳的會是 GORM 預設排序下的任一筆，行為未定義。
`docs/FRONTEND_SPEC.md` §3.2/§5 因此第一版的 Assets 區塊只顯示單一錢包。

**建議修法**：
新增 `GET /wallets`（帶 auth，回傳當前登入使用者名下所有錢包），多幣別功能
真的啟用後前端才能顯示完整的資產列表。若要保留 `GET /wallet/{user_id}`
相容，需要求帶 `currency_id` query 參數並改用
`GetWalletByUserIDAndCurrency`，不要繼續依賴「只用 user_id 查詢」的隱含假設。

---

## [FE2] 交易紀錄查詢不支援方向／幣別／日期篩選

**嚴重度**：Low（功能缺口，非缺陷）
**位置**：`api/handlers/transaction_handler.go`（`GetTransactions`）；
`api/repositories/transaction_repository.go`
（`GetTransactionsByUserIDWithPagination` 只接受 `offset`/`limit`）

**問題描述**：
History 頁需要的 Sent/Received、幣別、日期區間篩選，後端完全沒有對應查詢
參數，只能撈全部（受限於 `page_size` 上限 100）再由前端自行篩選，資料量一大
就會失真。`docs/FRONTEND_SPEC.md` §3.4/§5 因此第一版篩選列只有 All 可互動，
其餘選項先停用。

**建議修法**：
`GetTransactionsByUserIDWithPagination` 加上 `direction`/`currency_id`/
`from`/`to` 參數，直接在 SQL `WHERE` 子句篩選，而不是先撈出來再讓前端或
應用層過濾。

---

## [FE3] 沒有匯率／估值來源

**嚴重度**：Low（功能缺口，非缺陷）
**位置**：無對應程式碼（`models/currency.go` 沒有匯率欄位，也沒有匯率相關的表）

**問題描述**：
Overview 頁的 Assets 區塊理想上想顯示「以 USDT 計價的估值」，但系統裡沒有
任何匯率資料，也沒有整合任何外部匯率來源。`docs/FRONTEND_SPEC.md` §3.2/§5
因此第一版直接隱藏估值欄位。

**建議修法**：
新增一個固定／靜態的匯率表（demo 用途不需要接真實匯率源）＋
`GET /rates?base=USDT` 端點，回傳目前系統內每個幣別對 `base` 的匯率。

---

## [FE4] 沒有伺服器端的交易紀錄匯出功能

**嚴重度**：Low（功能缺口，非缺陷）
**位置**：無對應程式碼

**問題描述**：
History 頁的 Export 只能由前端把目前已下載的分頁資料轉成 CSV；資料量大時
前端得先把所有分頁依序抓完才能匯出「全部」，對後端是多次重複請求、對使用者
是額外等待時間，且沒有一個明確的筆數上限保護（`docs/FRONTEND_SPEC.md` §3.4
的第一版做法是前端設一個安全上限，超過就提示使用者縮小範圍，而不是真的把
所有資料都拉下來）。

**建議修法**：
新增 `GET /transactions/{user_id}/export?format=csv`，由後端直接查詢並
串流輸出 CSV，不受前端分頁上限影響，也不需要前端發出多次請求。
