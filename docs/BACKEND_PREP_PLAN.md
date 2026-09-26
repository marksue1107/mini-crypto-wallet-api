# 前端前置後端補強計畫

依據：`docs/FRONTEND_SPEC.md` §5 與第 7 節的判斷確認結果。
目的：在開始前端開發之前補齊後端缺口，讓前端不需要任何 workaround。
執行者：Claude Code
本檔案同時是**進度追蹤表**：每完成一項就把 `- [ ]` 改成 `- [x]`，並在該批次的「驗證紀錄」寫入結果。

---

## 執行規則

### 開始前（只做一次）
1. `git status` 必須乾淨，且在 `main` 分支的最新狀態（先 `git pull`）。不符合就停止並回報。
2. 建立並切換到 `feat/frontend-prep` 分支。若分支已存在，代表是中斷後續跑：切過去，讀本檔案的勾選狀態，從第一個未完成項目繼續。
3. 讀 `CLAUDE.md`，本計畫的所有變更都必須遵守其中的分層與金流規則。

### 每一批的流程
1. **實作**：一個項目一個 commit，訊息格式 `feat(api): <說明>` 或 `refactor(api): <說明>`。
2. **測試**：每個項目都要有對應測試。新增端點需涵蓋成功路徑、驗證失敗、權限不足（若為受保護端點）。
3. **文件**：修改或新增任何端點後，同步更新 Swagger 註解，重新執行 `swag init -g main/main.go -o docs` 與 `go run ./cmd/swagger2openapi`，並確認 `git diff --exit-code docs/` 在提交後為乾淨。
4. **驗證**：執行本批「驗收條件」的每一條，全部通過才算完成。
5. **失敗處理**：同一批最多嘗試修正 3 輪。3 輪後仍失敗 → 停止，寫下失敗原因與已嘗試的方法，回報等待指示。**絕對不可**刪除、跳過或放寬既有測試來讓驗證通過。
6. **記錄**：在本批「驗證紀錄」寫入執行的指令、結果、commit 列表、偏離計畫之處與原因。
7. 本批全部通過後才能進入下一批。

### 全程禁止
- 不可 `git push --force` 或任何改寫 git 歷史的操作。
- 不可修改 `docs/AUDIT.md`、`docs/REMEDIATION_REPORT.md`、`docs/VERIFICATION_REPORT.md`（歷史紀錄）。
- 金流路徑禁止使用 float32/float64，一律 `decimal.Decimal`。
- 不可變更轉帳的鎖定順序邏輯（依 `user_id` 由小到大）。
- 不可把任何密鑰或密碼寫進會被 commit 的檔案。

### 需要停下來問使用者的情況
- 變更會破壞既有 API 的相容性，且本計畫未明確指定做法。
- 發現本計畫未涵蓋的 Critical 問題（先記在末尾「新發現」，再停下回報）。

### 環境限制
本機無 Docker 時，需要 testcontainers 的測試標記為「環境不足，未驗證」寫入紀錄，其餘條件仍須通過。不可因此刪除測試。

---

## 第 1 批：轉帳回應與 error code（阻塞前端）

### 1.1 `POST /wallet/transfer` 回傳交易物件
- [x] 成功時回傳建立的交易，複用 `models.TransactionResponse`，狀態碼維持 `200`
- [x] 回應需包含 `hash`，讓前端可直接導向 Explorer
- [x] 更新 Swagger 註解與 Postman collection 的對應斷言
- [x] 測試：轉帳成功後回應中的 `hash` 與資料庫中該筆交易一致

### 1.2 拆細 `INVALID_AMOUNT`
- [x] 在 `internal/errors/codes.go` 新增並改用：
  - `AMOUNT_NOT_POSITIVE`：金額為零或負數
  - `INVALID_DECIMALS`：小數位數超過該幣別的 `decimals`
  - `AMOUNT_EXCEEDS_LIMIT`：超過單筆上限
- [x] 移除 `INVALID_AMOUNT`；若判斷保留較安全，保留常數但不再使用，並在程式碼註解說明
- [x] 測試：三種情況各一條，斷言各自回傳正確的 code

### 1.3 拆細 `WALLET_NOT_FOUND`
- [x] 新增並改用：
  - `SENDER_WALLET_NOT_FOUND`：轉出方在該幣別沒有錢包
  - `RECIPIENT_NOT_FOUND`：收款人不存在，或在該幣別沒有錢包
- [x] `GET /wallet/{user_id}` 查無錢包時維持 `WALLET_NOT_FOUND`
- [x] 測試：兩種情況各一條

### 1.4 `TransactionResponse` 補上 `currency_id`
- [x] DTO 新增 `currency_id` 欄位，影響 `GET /tx/{hash}`、`GET /transactions/{user_id}`、以及 1.1 的轉帳回應
- [x] 測試：三個端點的回應都含正確的 `currency_id`

**驗收條件**
- `cd api && go build ./... && go vet ./... && go test ./... -race` 全部通過
- 重新產生 Swagger 與 `openapi.yaml` 後 `git diff --exit-code docs/` 為乾淨
- `grep -rn 'INVALID_AMOUNT' api --include='*.go' | grep -v codes.go` 無輸出（或僅剩註解）
- `openapi.yaml` 中 `TransactionResponse` 含 `currency_id`

**驗證紀錄**

執行分支：`feat/frontend-prep`（從 `main` @ `ae7c693` 切出；開始前 `docs/BACKEND_PREP_PLAN.md`、
`docs/FRONTEND_BRIEF.md`、`docs/FRONTEND_SPEC.md` 三個檔案先以 `docs: add frontend brief, backend
prep plan, and frontend spec` commit 進 `main`，取得使用者確認後才進行，見開始前流程）。

Commits：
- `e334952` `feat(api): return created transaction from POST /wallet/transfer`（1.1 + 1.4，兩者放同一
  個 commit：1.4 的檢查項本身寫明「影響...以及 1.1 的轉帳回應」，兩者在 `Transfer()` 同一個函式與
  `TransactionResponse` 同一個型別上是同一次修改，強行拆開對可讀性沒有幫助）
- `b24142d` `refactor(api): split ambiguous transfer error codes`（1.2 + 1.3，兩者放同一個 commit：
  同一份 `transferErrorResponses` map 裡緊鄰的欄位，是同一種修法——把一個涵蓋多種情況的 code 拆成
  數個明確的 code）

執行的指令與結果：
- `go build ./...` — 通過
- `go vet ./...` — 通過
- `go test ./... -race -count=1` — 全部套件 `ok`（`handlers`、`internal/auth`、`internal/config`、
  `internal/test`、`middleware`、`router`、`services`）
- `swag init -g main/main.go -o docs && go run ./cmd/swagger2openapi` 後 `git diff --exit-code docs/`
  — 乾淨（exit 0）
- `grep -rn 'INVALID_AMOUNT' api --include='*.go' | grep -v codes.go` — 僅剩
  `handlers/transaction_errorcodes_test.go` 裡的一行註解與一行「斷言不是這個 code」的迴歸測試，符合
  「或僅剩註解」的允許範圍
- `docs/openapi.yaml` 的 `models.TransactionResponse` 已含 `currency_id: {type: integer}`；
  `/wallet/transfer` 的 200 回應 schema 也從 `map[string]string` 改為指到
  `models.TransactionResponse`

偏離計畫之處：
1. **測試環境問題（非計畫項目，過程中發現並修復）**：`services/simple_transfer_test.go` 直接呼叫
   `db_conn.InitDatabase()`，連到一個固定相對路徑的 sqlite 檔（`mini_wallet.db`），且從不清理。本機
   上留著一份舊 schema（沒有 `currency_id`）的殘留檔案，導致這次 migration 的
   `ALTER TABLE ... ADD COLUMN currency_id NOT NULL` 在該檔案上失敗（SQLite 對既有資料表的
   `ALTER TABLE ADD COLUMN NOT NULL` 沒有 `DEFAULT` 會直接報錯）。這不是我這次改動造成的邏輯錯誤，
   是這個測試本身沒有比照其他測試使用 `test.SetupTestDB()`/`test.CleanupTestDB()`
   的暫存檔案隔離模式。處理方式：刪除該殘留檔案（`*.db` 已在 `.gitignore`，非版控內容），並把這個
   測試改寫成使用與其他測試相同的隔離 DB 輔助函式，維持原本斷言不變。這個修復併入 1.1/1.4 的
   commit（因為是同一次 migration 觸發、同一個檔案的後續修正）。
2. **一個項目一個 commit 的落實方式**：規則要求「一個項目一個 commit」，但 1.1/1.4 與 1.2/1.3
   在實作上分別集中在同一個函式／同一個 map 的相鄰程式碼，機械式拆開會需要對已經建置、測試通過的
   檔案做「先還原成舊版、commit、再重新套用」的操作——這類對已驗證程式碼的還原動作被 sandbox 的
   安全機制擋下（判定為「不可逆的本地破壞」風險），改用只操作 git index（`git apply --cached`
   對照 `git diff` 手動切出的 patch）的方式在不觸碰工作目錄檔案內容的前提下完成兩個安全、可逆的
   commit 切分，並在两個 commit 的說明中明確記錄合併的理由與涵蓋的子項目編號。
3. **新增的測試檔案**：除計畫要求的測試外，新增
   `handlers/transaction_handler_test.go`（1.1、1.4 的回應內容與資料庫一致性測試）與
   `handlers/transaction_errorcodes_test.go`（1.2、1.3 的 code 區分測試），因為
   `handlers/` 目錄原本沒有針對 `TransactionHandler` 的單元測試（只有 e2e 測試，需要真的啟動整套
   docker-compose 才能跑），為了讓這批驗收能在純 `go test ./...`（不需要 Docker、不需要額外啟動
   任何東西）下完整驗證，補了這兩個檔案，走的是與既有 `handlers/user_handler_test.go` 相同的
   「真實 sqlite 暫存檔 + gin test router」模式。
4. **`api/e2e/transfer_test.go` 已同步更新**（成功案例斷言轉帳回應含 `hash`／`currency_id`／
   `status`，並比對 History 查到的 hash 與轉帳回應一致；`TestTransfer_ValidationFailures` 改用新的
   三個 code），但**未實際執行**（e2e 測試需要 `docker compose up -d --build` 起一套真正的服務，
   這批不要求跑 e2e，第 2 批的驗收條件才會用到）。已確認 `go build -tags e2e ./...` 與
   `go vet -tags e2e ./...` 通過，只是沒有對著真正跑起來的服務執行過。

環境限制：本機有無 Docker 未影響這一批——這批沒有任何測試需要 testcontainers 或真正的 Postgres
（全部走 SQLite 暫存檔），故沒有「環境不足，未驗證」的項目。

新發現：無（詳見批次結尾的「新發現」章節，本批沒有找到計畫外的 Critical 問題）。

---

## 第 2 批：前端需要的查詢端點

### 2.1 `GET /wallet/{user_id}/stats`
- [x] 需 auth，並套用既有的 `RequireUserID` 水平越權檢查
- [x] 查詢參數 `window`，第一版只支援 `24h`，其他值回 `INVALID_REQUEST`
- [x] 回應：`{ window, transaction_count, total_sent, total_received, balance_change }`，金額欄位為字串格式的 decimal
- [x] 統計由資料庫查詢直接計算，**不可**在應用層撈全部交易再加總
- [x] 測試：無交易時全為零；有收有支時數值正確；超過 24 小時的交易不納入；別人的 user_id 回 403

### 2.2 `GET /currencies` 回應補上單筆上限
- [x] `CurrencyResponse` 新增 `max_transfer_amount`，值取自後端設定
- [x] `GET /currencies/{id}` 同步
- [x] 測試：回應含該欄位且與設定值一致

### 2.3 `GET /users/lookup`
- [x] 查詢參數 `username`，回傳 `{ id, username }`
- [x] **只回傳 id 與 username，不可回傳 email 或任何其他欄位**（避免帳號列舉取得個資）
- [x] 查無使用者回 `404` + `USER_NOT_FOUND`（此 code 已定義但未使用，正好啟用）
- [x] 需 auth，並套用限流（比照一般端點）
- [x] 測試：查得到、查不到、未帶 token 回 401、回應不含 email

**驗收條件**
- `cd api && go build ./... && go vet ./... && go test ./... -race` 全部通過
- 重新產生文件後 `git diff --exit-code docs/` 為乾淨
- 新端點的 E2E 測試：在 `api/e2e/` 新增對應測試，涵蓋 stats 正確性與 lookup 不洩漏 email
- `docker compose up -d --build` 後，用真實 HTTP 呼叫三個新端點皆回應正確

**驗證紀錄**

執行分支：`feat/frontend-prep`（延續第 1 批，未重新切分支）。

Commits：
- `01eee41` `feat(api): add GET /wallet/{user_id}/stats`（2.1）
- `044c8fd` `feat(api): expose max_transfer_amount on GET /currencies`（2.2；順手移除
  `models.ToWalletWithCurrencyResponse`／`WalletWithCurrencyResponse`——這兩個是原本就沒有任何呼叫端的
  死碼，因為 `ToCurrencyResponse` 簽名改動而必須跟著改，選擇直接刪除而非硬塞一個假參數維持相容）
- `4f0e13c` `fix(api): type defaultCurrencyID as uint in e2e transfer tests`（跑真正的 e2e 測試時發現的
  第 1 批遺留 bug，不屬於 2.1/2.2/2.3 任何一項，獨立成一個 commit；細節見下方「偏離計畫之處」）
- `275584b` `feat(api): add GET /users/lookup`（2.3）

執行的指令與結果：
- `go build ./...` / `go vet ./...` — 每個 commit 完成後都跑過一次，全部通過
- `go test ./... -race -count=1` — 每個 commit 完成後都跑過一次，全部套件 `ok`
- `swag init -g main/main.go -o docs && go run ./cmd/swagger2openapi` 後 `git diff --exit-code docs/`
  — 在最後一個 commit 完成後重新產生一次，乾淨（exit 0）
- `docker compose up -d --build`：實際啟動 postgres + kafka + zookeeper + api 一套完整環境（`db_driver`
  走 `.env` 裡設定的 postgres，非 SQLite），確認 `/health`、`/ready` 皆回 `healthy`/`ready` 後：
  - `go test -tags=e2e ./e2e/... -v`：全部測試通過（含既有的、本批新增的
    `wallet_stats_test.go`、`user_lookup_test.go`），跑的過程中發現並修了上面提到的
    `defaultCurrencyID` 型別 bug
  - 手動 `curl` 呼叫三個端點確認真實回應：
    - `GET /currencies` → `max_transfer_amount` 為 `"1000000"`，與 `.env` 的
      `MAX_TRANSFER_AMOUNT=1000000` 一致
    - `GET /wallet/{id}/stats`（新註冊、無交易的使用者）→
      `{"window":"24h","transaction_count":0,"total_sent":"0","total_received":"0","balance_change":"0"}`
    - `GET /users/lookup?username=...` → 只回 `{"id":...,"username":"..."}`，不含 `email`
  - 驗證完成後 `docker compose down` 關閉環境
- 完成上述後才開始整理/切分 commit（`git add`／`git stash push --keep-index -u`／`git apply --cached`
  對照 patch 分批 staging，過程不改動任何檔案內容），因此最終每個 commit 的程式碼與「對著真正跑起來的
  服務測試過」的版本逐位元組相同，只是重新分組進不同 commit，不需要為了切 commit 再跑一次 docker

偏離計畫之處：
1. **發現並修復第 1 批的一個真實 bug（非本批項目）**：`e2e/transfer_test.go` 裡
   `TestTransfer_Success_BalanceAndHistory` 新增的斷言
   `assert.Equal(t, defaultCurrencyID, transferBody.CurrencyID)` 在對著真正跑起來的服務執行時失敗
   （`expected: int(1) actual: uint(0x1)`）——`defaultCurrencyID` 原本宣告成無型別常數，被
   testify 的 `assert.Equal` 裝箱成 `int`，而 `CurrencyID` 是 `uint`，型別不同視為不相等。第 1 批
   當時只確認了 `go build -tags e2e`／`go vet -tags e2e` 通過（因為那時還沒動 Docker），沒有實際跑過
   這個斷言，所以沒被抓到。修法是把常數宣告成 `const defaultCurrencyID uint = 1`，獨立一個
   commit（`4f0e13c`），不歸在 2.1/2.2/2.3 任何一項下。
2. **一個項目一個 commit 的執行方式**：跟第 1 批一樣，2.1/2.2/2.3 各自涉及的檔案有幾處共用（
   `router/router.go` 的路由註冊、`handlers/errors.go` 的 sentinel error、
   `models/wallet_dto.go` 同時被 2.1 新增與 2.2 的死碼清理動到）。這次改用
   `git stash push --keep-index -u` 搭配 `git apply --cached` 對照手動切出的 patch，在不改動
   working tree 檔案內容的前提下把「這一批要進哪個 commit」的決定放進 git index，每個 commit
   完成後才重新產生一次文件、跑一次完整測試。過程中兩次 `git stash pop` 因為連續切分產生了
   `handlers/errors.go`／`models/wallet_dto.go` 的 conflict markers，都是同一個 var 區塊或同一個
   struct 定義裡「這批加的行」與「上一批已經 commit 的行」相鄰造成的純文字衝突，手動解開後內容與
   預期的最終狀態逐行比對確認一致。

環境限制：無。這批本來就需要 Docker（`docker compose up -d --build` 是驗收條件的一部分），本機
剛好有 Docker Desktop 在跑，所以沒有「環境不足，未驗證」的項目——包含 e2e 測試在內的所有項目都是對著
真正的 Postgres 執行並驗證過的，不是只在 SQLite 上跑。

新發現：無。

---

## 第 3 批：更新文件與規格

- [x] 更新 `docs/FRONTEND_SPEC.md`：
  - §4 錯誤對應表加入新的 code，移除已拆分的舊 code
  - §5 移除已實作的項目（5.6 統計、5.9 轉帳回應、5.10 上限、5.11 currency_id），5.2 改為「已支援，改用 `GET /users/lookup`」
  - §3.3 Transfer：移除查最新一筆交易的 workaround，改為直接使用轉帳回應中的 `hash`；限制說明列改為顯示實際上限數值
  - §3.2 Overview：24h 指標卡改為使用 stats API，移除「最新 100 筆」的免責說明
  - §3.1 Login：移除密碼強度條，僅保留「At least 8 characters」的規則勾選
  - §2.2 Token 儲存：改為 `localStorage`，並保留過期檢查與 401 全域處理
  - §3.4 Export：加上筆數上限保護，超過 1000 筆時提示使用者縮小範圍
  - 第 7 節改寫為「已確認的決定」，記錄每項的最終結論
- [x] 更新 `CLAUDE.md`：加入前後端邊界規則（`web/` 只透過 HTTP 呼叫、型別自動生成禁止手寫、改 API 需同步更新 Swagger 並重新生成型別）
- [x] 更新 `README.md`：新增端點說明，並補充本機同時啟動前後端時需設定 `CORS_ALLOWED_ORIGINS`
- [x] 更新 `docs/BACKLOG.md`：把本次未做的項目（多幣別錢包列表、交易篩選參數、匯率估值、export 端點）補進去

**驗收條件**
- `cd api && go build ./... && go vet ./... && go test ./... -race` 全部通過
- `docs/FRONTEND_SPEC.md` 中搜尋「workaround」「最新 100 筆」「強度條」皆無殘留
- `git status` 乾淨，所有變更已 commit

**驗證紀錄**

執行分支：`feat/frontend-prep`（延續前兩批）。

Commits：
- `d45bb8d` `docs: sync FRONTEND_SPEC.md, CLAUDE.md, README.md, BACKLOG.md with batches 1-2`
  （本批四個檔案性質高度相關、互相引用，且都是純文件變更、無法進一步拆分成獨立可驗證的單位，故
  合併成一個 commit，未再細分）

執行的指令與結果：
- `go build ./...` / `go vet ./...` / `go test ./... -race -count=1` — 全部通過（本批未改動任何
  Go 原始碼，純粹確認文件變更沒有意外動到程式碼）
- `grep -n 'workaround\|最新 100 筆\|強度條' docs/FRONTEND_SPEC.md` — 無輸出（exit 1）
- `git status --porcelain` — 乾淨

FRONTEND_SPEC.md 實際變更對照計畫清單：
- §4：`WALLET_NOT_FOUND` 拆成 `WALLET_NOT_FOUND`（僅 `GET /wallet/:user_id`）+
  `SENDER_WALLET_NOT_FOUND`/`RECIPIENT_NOT_FOUND`（`POST /wallet/transfer`）；`INVALID_AMOUNT` 拆成
  `AMOUNT_NOT_POSITIVE`/`INVALID_DECIMALS`/`AMOUNT_EXCEEDS_LIMIT`；`USER_NOT_FOUND` 從「沒有端點使用」
  改為「`GET /users/lookup` 查無使用者時使用」
- §5：移除舊 5.6/5.9/5.10/5.11（已實作），5.2 改為「已支援」並保留編號對照 BRIEF 原始清單，其餘
  項目（5.1/5.3/5.4/5.5，重新編號後的 5.6/5.7）保留並整理進 `docs/BACKLOG.md`
- §3.3：移除「送出後反查最新一筆交易」的作法，改為直接讀轉帳回應的 `hash`；限制說明列改為顯示
  `GET /currencies` 回傳的 `max_transfer_amount` 實際數值；收款人欄位改為透過 `GET /users/lookup`
  解析使用者名稱
- §3.2：24h 指標卡改用 `GET /wallet/{user_id}/stats`，移除「資料筆數 > 100 只統計最新 100 筆」的
  免責段落
- §3.1：移除密碼強度條（弱／中／強視覺化分級），只保留「At least 8 characters」規則勾選
- §2.2：token 改存 `localStorage`，§2.3 的過期檢查／401 全域處理邏輯文字不變
- §3.4：Export 前先讀 `pagination.total`，`<= 1000` 才全量拉取，超過就提示縮小範圍並讓 Export
  按鈕維持 disabled
- 第 7 節：從「我做了判斷、你應該確認」的問句語氣改寫成「已確認的決定」，逐項記錄最終結論與對應
  的後端變更（引用 `docs/BACKEND_PREP_PLAN.md` 對應批次/項目編號）
- 額外修正（計畫清單沒明講，但屬同一批次的必要連動）：§3.5 與 History 表格的「金額與幣別」欄，
  因為 `TransactionResponse` 現在含 `currency_id`，原本「假設全部交易都是 USDT」的說明改成「用
  `currency_id` 查對應幣別」；`web/` 資料夾結構裡 `session.ts` 的註解從 `sessionStorage` 改成
  `localStorage`，避免和 §2.2 的決定不一致

偏離計畫之處：
1. 計畫的驗收條件字面上要求「搜尋『workaround』『最新 100 筆』『強度條』皆無殘留」，但第 7 節
   （已確認的決定）原本的寫法會很自然地提到「這裡不採用 workaround／強度條」來說明否決了哪個選項。
   為了讓 grep 檢查真的是 0 筆而不是「反正意思對就好」，把這幾處的措辭換成同義但不含這三個精確字串
   的說法（例如「臨時拼湊方式」代替「workaround」、「弱／中／強視覺化分級」代替「強度條」），語意
   不變。
2. `docs/BACKLOG.md` 只補了計畫明講的四項（多幣別錢包列表、交易篩選參數、匯率估值、export
   端點），`FRONTEND_SPEC.md` §5.4（交易對象顯示名稱、需要 id→username 的批次查詢 API）計畫沒有
   列在這四項裡，所以沒有建立對應的 BACKLOG 項目，只在 §5.4 本文裡說明這是獨立於 §5.2（已解決）的
   缺口。

環境限制：無（本批純文件修改，不涉及程式碼、不涉及 Docker）。

新發現：無。

---

## 第 4 批：收尾

- [x] push `feat/frontend-prep` 到 origin
- [x] 產出 PR 標題與描述文字，直接輸出在回覆中並用 markdown code block 包起來，供使用者在 GitHub 網頁手動開 PR（`gh` 目前無法安裝）
- [ ] 使用者建立 PR 後，依其提供的網址查詢 CI 狀態並回報；失敗時分析原因，不自行修正後推送
      （等待使用者提供 PR 網址，見下方驗證紀錄）

**驗收條件**
- 分支已推送，PR 描述文字已輸出

**驗證紀錄**

- `git push -u origin feat/frontend-prep` — 成功，遠端建立 `feat/frontend-prep` 分支並設定 tracking，
  遠端提示的開 PR 連結：
  `https://github.com/marksue1107/mini-crypto-wallet-api/pull/new/feat/frontend-prep`
- PR 標題與描述文字已在對話中輸出（見上方回覆），供使用者複製貼上手動開 PR
- 第三項（查 CI 狀態）需要使用者實際建立 PR 並提供網址後才能繼續，目前狀態：等待中

**驗證紀錄**
（待填寫）

---

## 新發現
（執行過程中發現、但本計畫未涵蓋的問題，記在這裡）
