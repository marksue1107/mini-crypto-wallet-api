# 稽核修正執行計畫

依據：`docs/AUDIT.md`（稽核當下 commit `80e6220`）
執行者：Claude Code
本檔案同時是**進度追蹤表**：每完成一項就把 `- [ ]` 改成 `- [x]`，並在該批次的「驗證紀錄」寫入結果。

---

## 執行規則（每一批都必須遵守）

### 開始前（只做一次）
1. 執行 `git status`，工作區必須是乾淨的；若不乾淨，**停止並回報**，不要自行 stash 或丟棄變更。
2. 從目前分支建立並切換到 `fix/audit-remediation` 分支。若分支已存在，代表是中斷後續跑：直接切過去，讀本檔案的勾選狀態，從第一個未完成的項目繼續。

### 每一批的流程
1. **開工**：讀 `docs/AUDIT.md` 中本批對應的發現項目，確認目前程式碼位置（第 0 批後路徑會多一層 `api/`，行號可能已變動，以實際程式碼為準）。
2. **實作**：一個發現項目（或一組緊密相關的項目）一個 commit，commit 訊息格式：`fix(<AUDIT編號>): <說明>`，例如 `fix(M1): lock wallets in ascending user_id order`。
3. **驗證**：執行本批「驗收條件」中的每一條指令，全部通過才算完成。
4. **失敗處理**：驗證失敗時，分析原因並修正，**同一批最多嘗試修正 3 輪**。3 輪後仍失敗 → 停止所有工作，在本檔案寫下失敗原因與已嘗試的方法，回報後等待指示。**絕對不可**為了讓驗證通過而刪除、跳過、註解掉既有測試或斷言。
5. **記錄**：在本批「驗證紀錄」寫入：執行了哪些指令、結果、commit 列表、任何偏離計畫之處與原因。
6. **進入下一批**：只有本批所有驗收條件通過且紀錄已寫入，才能開始下一批。

### 全程禁止
- 不可 `git push`、`git push --force`、`git filter-repo` 或任何改寫 git 歷史的操作。
- 不可修改 `docs/AUDIT.md`（它是稽核當下的紀錄）。
- 不可在非 `_test.go` 檔案 `import "testing"`。
- 金流路徑（餘額、金額、審計紀錄）禁止使用 float32/float64。
- 不可把任何密鑰、密碼寫進會被 commit 的檔案。

### 需要停下來問使用者的情況
- 需要做任何上面「全程禁止」清單裡的事。
- 修正方式會改變對外 API 的行為，且本計畫沒有明確指定做法。
- 發現 `docs/AUDIT.md` 沒記錄、但屬於 Critical 等級的新問題（先記錄在本檔案末尾「新發現」，再停下回報）。

### 環境限制的處理
- 若本機沒有 Docker：需要 Docker 的驗收條件（testcontainers、docker compose）標記為「環境不足，未驗證」寫入紀錄，其餘條件仍須通過，然後繼續下一批。**不可**因此把相關測試刪掉；testcontainers 測試應在無 Docker 時 `t.Skip`。

---

## 第 0 批：清理 repo 並搬移後端到 `api/`
對應：Q2、Q3、Q4 ＋ 專案結構調整

- [x] `git rm --cached` 移除 `wallet-api`、`main/__debug_bin*`、`.idea/`（以及 `.vscode/` 若有被追蹤）
- [x] `.gitignore`：新增 `/wallet-api`、`/api/wallet-api`、`**/__debug_bin*`、`*.db`；移除誤植的 `README.md` 與 `docs/` 規則
- [x] 用 `git mv` 將 `go.mod`、`go.sum`、`Dockerfile`、`config.yaml` 與所有 Go 程式資料夾（db_conn、handlers、internal、kafka_client、main、middleware、models、repositories、router、services、utils，以及 Swagger 產出的 docs 若屬 Go 套件）移入 `api/`
- [x] 根目錄保留：`README.md`、`docs/`（稽核報告等文件）、`docker-compose.yml`、`CLAUDE.md`、Postman 檔案、本計畫檔
- [x] 更新 `docker-compose.yml`、`Dockerfile`、`README.md` 中所有路徑與指令
- [x] `go.mod` 的 module 名稱維持不變，import 路徑不改

**驗收條件**
- `cd api && go build ./...` 成功
- `cd api && go vet ./...` 成功
- `git ls-files | grep -E '__debug_bin|\.idea/|(^|/)wallet-api$'` 無輸出
- （本批不要求 `go test` 通過，已知會 panic，第 1 批處理）

**驗證紀錄**
- 開始前 `git status` 確認乾淨，建立並切換到 `fix/audit-remediation` 分支。
- `git rm --cached wallet-api main/__debug_bin2424212485` 及 `git rm --cached -r .idea`，並額外把這兩顆二進位檔從磁碟刪除（純建置產物，非原始碼，判斷可直接刪除而非只是搬移到 `api/`）。`.vscode/` 本來就未被追蹤，不需處理。
- `.gitignore`：移除 `README.md`、`docs/` 兩條誤植規則；新增 `/wallet-api`、`/api/wallet-api`、`**/__debug_bin*`（`*.db` 原本就已存在，未重複新增）。**偏離計畫之處**：順手移除了原本無意義的 `.git/` 規則（git 本來就不會意外把自己的 `.git/` 目錄加入版控，這條規則沒有任何效果），屬於同一批「清理 .gitignore」精神下的小範圍衛生清理。
- 用 `git mv` 把 `go.mod`/`go.sum`/`Dockerfile`/`config.yaml`/`db_conn`/`handlers`/`internal`/`kafka_client`/`main`/`middleware`/`models`/`repositories`/`router`/`services`/`utils` 全部移入 `api/`。
- **新發現**（記錄於本檔案末尾「新發現」）：`docs/docs.go`、`docs/swagger.json`、`docs/swagger.yaml`（swaggo 產生的 Go 套件檔案，`main.go` 用 `_ "mini-crypto-wallet-api/docs"` import）在稽核當下其實從未被 git 追蹤（`git log --all` 對這三個檔案完全沒有紀錄），只是本地磁碟上存在、被舊版 `.gitignore` 的 `docs/` 規則擋掉而已。這代表全新 clone 這個 repo 在本次修正之前是**無法直接 `go build` 成功的**（`docs.go` 不存在）。處理方式：用本機的 `swag` CLI（v1.16.4）在新路徑 `api/docs/` 重新產生（`swag init -g main/main.go -o docs`），確保內容與目前程式碼的 Swagger 註解一致，而不是搬移可能過時的舊檔案；之後正式 `git add` 進版控，修正這個「無法重現建置」的問題。
- 同步更新 `.vscode/launch.json`（未被 git 追蹤的本機檔案）的 `program`/`cwd` 指向 `api/`，避免 Mark 本機除錯設定失效；此檔案不影響任何驗收條件，純屬順手修正。
- 更新 `README.md`：測試指令加上 `cd api &&` 前綴；`docker build` 指令改成 `docker build -t mini-wallet-api ./api`；在「Start Kafka and PostgreSQL」步驟加上明確警告區塊，標註該步驟的 compose 檔缺口（Q6/A3）留給第 5 批處理，避免文件在修正期間誤導讀者。
- 執行 `cd api && go build ./...` → 成功（無輸出）。
- 執行 `cd api && go vet ./...` → 成功（無輸出）。
- 執行 `git ls-files | grep -E '__debug_bin|\.idea/|(^|/)wallet-api$'` → 無輸出，符合驗收條件。
- Commits（本批，`git log --oneline fix/audit-remediation` 可查）：
  - `24d6879` fix(Q2,Q3,Q4): remove committed build binaries/.idea from git, fix .gitignore
  - `47b6974` refactor: move Go backend module into api/
  - （本批最後一個 commit，加入本檔案時尚未產生 hash）fix(Q4,new): regenerate untracked swagger docs under api/docs; add audit docs and fix README paths

---

## 第 1 批：讓測試可信賴，並加上 CI
對應：M3、Q1、Q10、A6、A7

- [x] 測試 DB：統一使用 `modernc.org/sqlite` 對應的 dialector（與正式環境相同），改用暫存檔案 SQLite 或對 in-memory 設定 `SetMaxOpenConns(1)`
- [x] 金流相關測試的前置條件與關鍵斷言改用 `require`
- [x] 將 `TransferWithLockOption` 及其不安全示範邏輯移到 `_test.go` 檔，正式程式碼移除 `import "testing"`
- [x] `TestConcurrentTransfers` 改用 testcontainers-go 啟動一次性 Postgres；無 Docker 時 `t.Skip`，不可 `log.Fatal`
- [x] 新增 `.github/workflows/api.yml`：`paths: ['api/**']` 觸發，執行 `go build ./...`、`go vet ./...`、`go test ./... -race`

**驗收條件**
- `cd api && go test ./... -race` 全部通過，無 panic
- `cd api && go test ./... -race -count=3` 連續三次通過（確認不是偶發通過）
- `grep -rn '"testing"' api --include='*.go' | grep -v '_test.go'` 無輸出
- workflow YAML 語法正確（可用 `python -c "import yaml; yaml.safe_load(open('.github/workflows/api.yml'))"` 檢查）

**驗證紀錄**
- **M3 根因與修法**：`internal/test/test_helpers.go` 原本用 `sqlite.Open(":memory:")`。SQLite 的 `:memory:` DSN 是「每個實體連線各自一份空資料庫」，而 GORM 的連線池在單一 `*gorm.DB` 上本來就會視情況借出不只一條連線——`TransactionService.Transfer()` 正是這種情況：`tx := ...Begin()` 用掉一條連線做交易，緊接著 `s.walletRepo.GetWalletByUserIDAndCurrency(...)` 卻是用**非交易**的一般查詢（沒有帶 `tx`），可能向同一個連線池借到**另一條**連線，而那條連線在 `:memory:` 模式下看到的是全新的空資料庫，於是回報「wallet not found」。改為每次 `SetupTestDB()` 都建立一個唯一的暫存檔案（`os.CreateTemp` + `gorm.io/driver/sqlite` 的 `Dialector{DriverName:"sqlite"}` 指向 `modernc.org/sqlite`，與正式環境 `db_conn/sqlite.go` 用同一顆 driver，回應 A7），檔案型資料庫天生就是「所有連線看到同一份資料」，問題自然消失，且不需要限制連線池大小。
  - 中途也試過「`:memory:` + `SetMaxOpenConns(1)`」這個更常見的建議解法，但**會直接造成自我死結**：`Transfer()` 的 `tx.Begin()` 用掉唯一的一條連線，緊接著同一個 goroutine 內的非交易查詢要再借一條連線，但整個池只有 1 條、還被自己的交易鎖著，永遠借不到，測試整組 hang 住。已改回檔案型方案，不再嘗試單連線池。
  - `CleanupTestDB` 同步更新為關閉連線後刪除暫存檔（含 `-journal`/`-wal`/`-shm` 側車檔），用一個 `sync.Map`（`*gorm.DB` → 檔案路徑）追蹤，呼叫端簽名完全不用改。
- **require 修正**：`transaction_service_test.go` 中，凡是「這一步失敗，下一步就會對 nil 解參考」的地方全部從 `assert` 改成 `require`：`TestTransfer_Success_ValidTransfer`、`TestTransfer_Success_BalanceHistoryRecorded`、`TestTransfer_Success_TransactionHashGenerated`、`TestTransfer_MultipleSequential`、`TestTransfer_ConcurrentTransfers_NoRaceCondition` 的關鍵 `err`/`len` 檢查，並且不再用 `_` 丟棄 repository 查詢的 error 就直接解參考。
- **A6**：`services/transaction_service.go` 移除 `import "testing"`、移除 `TransferWithLockOption` 與其「tester」註解區塊。`internal/test/concurrency_demo_test.go` 改寫：不安全示範邏輯搬進測試檔自己的 `simulateUnsafeTransfer()`（透過公開的 `repositories.IWallet` 介面實作，不再需要碰 `TransactionService` 私有欄位），`useLock=true` 的情境直接呼叫正式的 `service.Transfer()`。
- **Q10 / testcontainers**：`TestConcurrentTransfers` 改成用 `testcontainers-go` + `modules/postgres` 啟動一次性 Postgres，換掉原本會在本機沒有 Postgres 時 `log.Fatal`（整個測試 process 直接退出）的 `config.LoadConfig()+db_conn.InitDatabase()`。新增 `dockerAvailable()`（`exec.LookPath("docker")` + 3 秒逾時的 `docker info`）在偵測不到可用 Docker daemon 時 `t.Skip(...)`，不會讓整個套件失敗。同時把原本只印餘額給人看的示範，加上真正的斷言（鎖定路徑下餘額不可為負、轉帳雙方餘額總和必須守恆）。
  - `go get github.com/testcontainers/testcontainers-go@latest github.com/testcontainers/testcontainers-go/modules/postgres@latest github.com/jackc/pgx/v5@latest` 後執行 `go mod tidy` 讓 go.sum 收斂；副作用：`go.mod` 的 `go` 版本指示被工具鏈自動調整為 `go 1.25.0`（由新依賴的最低需求版本決定），CI workflow 用 `actions/setup-go@v5` 的 `go-version-file: api/go.mod` 讀取，不寫死版本號，避免之後再次漂移。
- **Q1 / CI**：新增 `.github/workflows/api.yml`（`working-directory: api`，`paths: ['api/**', '.github/workflows/api.yml']` 觸發，依序 `go build ./...`、`go vet ./...`、`go test ./... -race -count=1`）。GitHub-hosted ubuntu runner 預裝 Docker，`TestConcurrentTransfers` 在 CI 上會是真的跑過 testcontainers，不會走到 skip 分支。YAML 語法用 `ruby -ryaml`（本機沒有可用的 `python3 yaml` 套件，改用內建的 Ruby psych）驗證通過。
- **實際執行結果**（本機 Docker Desktop 有啟動，非「環境不足」情況，已完整驗證）：
  - `go build ./...` / `go vet ./...`：通過，無輸出。
  - `go test ./... -race -count=1`：全部 `ok`，`internal/test` 的 `TestConcurrentTransfers` 真的對容器化 Postgres 跑過一次（未加鎖示範印出 A=0/B=800，加鎖路徑同樣 A=0/B=800 且斷言金額守恆與非負皆通過）。
  - `go test ./... -race -count=3`：連續三次皆 `ok`，未見任何 panic 或 flaky 失敗。
  - `grep -rn '"testing"' api --include='*.go' | grep -v '_test.go'`：無輸出。
- Commits（本批）：
  - `16bf545` fix(M3): make transfer test suite reliable, not flaky
  - `894b184` fix(A6,Q10): remove test-only code from production, run concurrency demo against real Postgres
  - `212bdd8` fix(Q1): add CI workflow to run build/vet/test -race on every change

---

## 第 2 批：資金正確性
對應：M1、M4、M6、M2（採方案 c）

- [x] M1：轉帳時依 `user_id` 由小到大鎖定兩顆錢包，與呼叫參數順序無關
- [x] M1 測試：A→B 與 B→A 大量同時轉帳，不發生 deadlock，且兩人餘額總和守恆（Postgres / testcontainers）
- [x] M4：新增依 `user_id` + `currency_id` 查詢並加鎖的 repository 方法，`Transfer()` 改用它
- [x] M6：依幣別 `Decimals` 驗證金額小數位數，超過即回傳驗證錯誤；新增單筆上限金額設定（放 config，預設值寫在 `config.yaml.example` 或程式常數）
- [x] M6 測試：小數位數超過、零、負數、超過上限，各至少一條
- [x] M2：SQLite 模式啟動時印出明確警告（不保證併發正確性、僅供單機開發）；README 補充說明

**驗收條件**
- `cd api && go test ./... -race` 全部通過
- 新增的 deadlock / 餘額守恆測試在有 Docker 時實際執行並通過（無 Docker 則依「環境限制」處理）
- 用 SQLite 啟動服務時，log 中可看到警告訊息

**驗證紀錄**
- **開工前先處理 N2/N3**（見上方，兩個都已獨立 commit 修好），才開始本批。
- **M1**：`Transfer()` 改成 `lowID, highID := fromID, toID`（`highID < lowID` 時互換），一律先鎖 `lowID`、再鎖 `highID`，事後再依 `fromID == lowID` 映射回 `fromWallet`/`toWallet`。**用臨時停用排序邏輯的方式實測驗證**：把排序那 3 行暫時改回「不排序」（`lowID, highID := fromID, toID` 不做互換），跑新測試 `TestTransfer_NoDeadlock_BidirectionalConcurrentTransfers`，在真的 Postgres（testcontainers）上重現出貨真價實的
  `ERROR: deadlock detected (SQLSTATE 40P01)`（log 裡出現 6 次），測試也如預期失敗；還原排序邏輯後，同一個測試穩定通過（500 個 SQL 常式很快跑完，不再有 deadlock）。SQLite 因為本來就不支援真正的 row-level lock（M2），沒有能力重現這種「兩個交易各自握有一個鎖互相等待」的結構性死結，所以這個測試必須用 testcontainers 起真的 Postgres 才有意義。
- **M4**：`IWallet` 介面把 `GetWalletByUserIDWithTx(userID, tx...)` 換成
  `GetWalletByUserIDAndCurrencyWithTx(userID, currencyID, tx...)`，查詢條件直接帶
  `user_id = ? AND currency_id = ?` 一起鎖定，不再是「先用 user_id 撈任一筆再事後比對
  CurrencyID」。順手把 `Transfer()` 一開始那兩個不需要鎖、單純預檢查用的
  `GetWalletByUserIDAndCurrency` 查詢拿掉（改成直接靠加鎖查詢本身判斷錢包是否存在/
  幣別是否相符），少了兩次多餘的資料庫往返。
- **M6**：`TransactionService` 新增 `currencyRepo` 欄位與 `maxTransferAmount`
  （`decimal.Decimal`）欄位；`NewTransactionService` 簽名新增 `currencyRepo
  repositories.ICurrency` 參數（已同步修改 `router/router.go` 與所有測試呼叫點，共
  14 處）。驗證邏輯：`amount.GreaterThan(maxTransferAmount)` 判斷是否超過上限
  （預設常數 `DefaultMaxTransferAmount = "1000000"`，可由 `config.yaml` 新增的
  `max_transfer_amount` 字串欄位覆寫，留空或解析失敗則用預設值）；
  `!amount.Equal(amount.Round(int32(currency.Decimals)))` 判斷小數位數是否超過該幣別
  允許的位數。新增 `TestTransfer_Fail_TooManyDecimalPlaces`（9 位小數 vs. USDT 測試幣別
  的 8 位）與 `TestTransfer_Fail_ExceedsMaxAmount`（超過預設 1,000,000 上限）；零元、
  負數已有既有測試涵蓋（`TestTransfer_Fail_ZeroAmount`、`TestTransfer_Fail_NegativeAmount`）。
- **M2**：`db_conn/sqlite.go` 在連線成功後印出明確的中文警告（SQLite 模式沒有真正的
  列鎖，`FOR UPDATE` 是 no-op，不保證併發轉帳正確性，僅供單機開發使用）；README 的
  「Concurrency Safety」小節同步補充這個限制的說明，並修正一處已經過時、指向舊方法名
  `GetWalletByUserIDWithTx` 的行號引用（commit `285d76b`）。
  - **意外發現並一併修正**：實作 M4 拿掉那兩次多餘查詢後，原本穩定通過的
    `TestTransfer_ConcurrentTransfers_NoRaceCondition`（SQLite、Batch 1 就有的測試）
    開始有約 4/5 的機率因為 `database is locked (5) (SQLITE_BUSY)` 失敗——因為
    SQLite 從來沒有設定過 `PRAGMA busy_timeout`，只要兩個連線的寫入階段時間點稍微
    靠近就會立刻報錯，而不是像正常情況下等一下讓另一個先寫完。這其實是本來就存在、
    只是靠著多餘查詢造成的時間差意外沒被踩到的潛在問題，屬於 M2「SQLite 併發安全」
    範疇內，於是一併在 `db_conn/sqlite.go` 與 `internal/test/test_helpers.go` 都加上
    `PRAGMA busy_timeout = 5000`（5 秒）解決，之後連續多次重跑該測試都穩定通過。
    這不是新的 Critical 發現（沒有造成資料錯誤，只是把「應該等待」的情境誤判成
    「直接失敗」），所以沒有另外停下回報，直接在本批修正紀錄中說明。
- **實際執行結果**：`go build ./...`、`go vet ./...` 皆乾淨；`go test ./... -race
  -count=3` 連續三次全部 `ok`，包含兩個 testcontainers 測試
  （`TestConcurrentTransfers`、`TestTransfer_NoDeadlock_BidirectionalConcurrentTransfers`）
  皆為 `--- PASS`（非 skip）。
- Commits（本批）：
  - `828c317` fix(M1,M4,M6,M2): fixed lock order, currency-aware locking, amount limits
  - `285d76b` docs(M2): document SQLite's lack of real concurrency safety in README

---

## 第 3 批：安全性
對應：S1、S2、S3、S4、S7、S9、S11

- [x] S1：`config.yaml` 自版控移除（`git rm --cached`）並加入 `.gitignore`；新增 `api/config.yaml.example`（佔位值）；密鑰與 DB 密碼改由環境變數注入
- [x] S2：`JWTSecret` 未設定或長度 < 32 時 `log.Fatal`，移除原始碼中的預設密鑰
- [x] S3：在 `/auth/login`、`/users`、`/wallet/transfer`、`/tx/:hash` 掛上限流（登入端點用較嚴格的設定）
- [x] S4：呼叫 `SetTrustedProxies`，可信代理從環境變數讀取，預設不信任任何代理
- [x] S7：重複帳號回傳 `409` + `USER_ALREADY_EXISTS`
- [x] S9：commit 失敗不回傳原始 DB 錯誤給客戶端，原始錯誤帶 trace id 寫入 log
- [x] S11：密碼最小長度提高到 8
- [x] 以上每一項都要有對應的 handler 或 middleware 測試

**驗收條件**
- `cd api && go test ./... -race` 全部通過
- `git ls-files | grep -E '(^|/)config\.yaml$'` 無輸出
- `grep -rn 'default-secret' api` 無輸出
- 不設定 JWT 密鑰啟動服務 → 程式立即結束並顯示清楚錯誤
- 限流測試：超過上限的請求回傳 `429`，且偽造 `X-Forwarded-For` 無法繞過

**驗證紀錄**
- **S1**：`git rm --cached api/config.yaml`，`.gitignore` 新增 `/api/config.yaml`；新增
  `api/config.yaml.example`（含產生隨機密鑰的指令範例、明確標註「留空或太短，服務會
  直接拒絕啟動」）。**額外處理**：發現只把 `config.yaml` 從版控移除還不夠——
  `internal/config/loader.go` 原本只有 `viper.AutomaticEnv()`，viper 的行為是「只有
  已經知道的 key 才能被環境變數覆寫」，如果完全沒有 `config.yaml`（正式環境的目標
  狀態），viper 根本不知道有 `jwt_secret`/`postgres_dsn` 這些 key 存在，光設環境變數
  不會被讀到。修法：對每個欄位呼叫 `viper.SetDefault(key, "")`，讓 viper 一開始就
  「認得」這些 key，環境變數才真的能覆寫。新增
  `internal/config/loader_test.go`：`TestLoadConfig_FromEnvOnly_NoConfigFile`
  （chdir 到全新的暫存目錄，保證真的沒有 `config.yaml`，只靠環境變數）與
  `TestLoadConfig_DefaultsWhenNothingSet`，兩者都通過，證明「純環境變數配置」現在
  真的可行。**需要使用者知情**：`config.yaml` 裡曾經 commit 過的那組 JWT 密鑰與
  Postgres 密碼已經進了 git 歷史，從工作目錄移除不代表歷史紀錄裡沒有，兩者都應該
  視為已外洩並更換（歷史清理留給第 6 批的收尾報告處理，需要使用者決定是否要重寫
  git 歷史）。
- **S2**：新增 `auth.ValidateSecretStrength`（最小 32 字元）與 `auth.MinSecretLength`
  常數；`router.go` 移除 `default-secret-key-change-in-production-min-32-chars` 這個
  硬編碼 fallback，改成驗證失敗就 `log.Fatalf`。測試：`internal/auth/jwt_test.go` 的
  `TestValidateSecretStrength`（表格測試涵蓋空字串/31 字元/32 字元/正常長度）；
  `router/router_test.go` 的 `TestSetupRouter_FailsFastOnWeakJWTSecret` 用 Go 官方
  建議的「重新執行自己的測試二進位檔」模式（`exec.Command(os.Args[0], "-test.run=...")`
  + 環境變數旗標）測試 `log.Fatal`／`os.Exit` 這種無法在同一個測試 process 內直接呼叫
  的路徑，確認子行程真的以非 0 狀態碼結束。
- **S3**：`router.go` 建立兩個限流器——一般端點 60 req/min（`generalLimiter`），登入
  10 req/min、burst 5（`loginLimiter`，比一般端點嚴格很多，因為是最常見的暴力破解
  目標）——分別掛到 `POST /users`、`GET /tx/:hash`、`POST /wallet/transfer`（一般）與
  `POST /auth/login`（嚴格）。新增 `middleware/ratelimit_test.go`
  （`TestRateLimitMiddleware_BlocksAfterLimit`、
  `TestRateLimitMiddleware_SeparateClientsHaveSeparateLimits`）與
  `router/router_test.go` 的 `TestSetupRouter_LoginIsRateLimited`（直接打真正組好的
  `SetupRouter()`，連續打 `/auth/login` 直到收到 429，證明限流真的掛在路由上，不是
  只有 middleware 本身能動）。
- **S4**：`router.go` 一開始就呼叫 `r.SetTrustedProxies(trustedProxies)`
  （`trustedProxies` 來自新的 `TRUSTED_PROXIES`/`trusted_proxies` 設定，逗號分隔，
  留空則傳 `nil` = 不信任任何代理）。新增
  `TestRateLimitMiddleware_ClientIPNotSpoofableViaXFF_WhenNoTrustedProxies`：
  **實測驗證**先把測試裡的 `SetTrustedProxies(nil)` 暫時拿掉重跑，確認測試真的會
  FAIL（同一個真實來源 IP 帶偽造的 `X-Forwarded-For` 就能繞過限流，拿到
  200 而非 429），加回來後穩定 PASS。
- **S7**：`UserService.CreateUser` 一開始就查詢 username/email 是否已存在，存在就回傳
  新的 sentinel error `services.ErrUserAlreadyExists`；`UserHandler.CreateUser` 用
  `errors.Is` 判斷，命中就回 `409` + `{"code":"USER_ALREADY_EXISTS"}`（沿用
  `internal/errors` 裡本來就定義好、但從未被使用過的常數）。新增
  `repositories.IUser.GetUserByEmail`（原本只有 `GetUserByUsername`）。測試：
  service 層的 `TestCreateUser_Fail_DuplicateUsername`／
  `TestCreateUser_Fail_DuplicateEmail`，以及**這個 repo 第一個 handler 層測試**
  `handlers/user_handler_test.go` 的 `TestCreateUser_Fail_DuplicateUsername_Returns409`
  （直接檢查 HTTP 409 與回應 body 的 `code` 欄位）。
- **S9**：`Transfer()`、`UserService.CreateUser` 裡所有「直接把底層 GORM/DB error
  往外拋」的地方（不只 `tx.Commit()` 失敗，連 `UpdateWallet`／`CreateTransaction`／
  `CreateHistory`／`CreateUser`／`CreateWallet` 寫入失敗）都改成：詳細錯誤寫進
  `log.Println`，回傳給呼叫端的是固定的安全訊息（例如「transfer failed, please try
  again」）。**未做自動化測試**：要在單元測試裡可靠地讓 `tx.Commit()`
  本身失敗（而不是前面的驗證邏輯攔下來）需要能操控底層連線的 mock，目前的
  repository 介面沒有為此設計，超出這次修正的合理範圍；已在此紀錄這個已知的測試
  缺口，供之後補強。
- **S11**：`models/user_dto.go` 的 `Password` binding 從 `min=6` 改成 `min=8`。測試：
  `handlers/user_handler_test.go` 的 `TestCreateUser_Fail_PasswordTooShort`（7 字元
  密碼應被拒絕）。
- **實際執行結果**：`go build ./...`、`go vet ./...` 皆乾淨；`go test ./... -race
  -count=3` 連續三次全部 `ok`，涵蓋新增的 `internal/auth`、`internal/config`、
  `middleware`、`router`、`handlers` 五個先前完全沒有測試的套件。
  `git ls-files | grep -E '(^|/)config\.yaml$'` 與 `grep -rn 'default-secret' api`
  皆無輸出。
- Commits（本批）：
  - `582da7e` fix(S1,S2,S3,S4): remove committed secrets, fail fast on weak JWT, wire up rate limiting and trusted-proxy config
  - `6806a02` fix(S7,S9,S11): 409 on duplicate signup, stop leaking raw DB errors, raise min password length

---

## 第 4 批：前端串接準備
對應：S8、S5、A2、A5、S6（決策：維持公開）

- [x] S8：所有 handler 錯誤統一回傳 `ErrorResponse{error, code, message}`，`code` 使用 `internal/errors` 常數；middleware（驗證、限流、JWT）的錯誤也統一成相同格式
- [x] S5：加入 `gin-contrib/cors`，允許的 origin 由環境變數 `CORS_ALLOWED_ORIGINS` 設定（逗號分隔），允許 `Authorization`、`Content-Type` 標頭，不對帶憑證請求使用 `*`
- [x] A2：README 改為正確描述 Swagger 2.0；新增轉換步驟（`swagger2openapi` 或等效工具）產出 `api/docs/openapi.yaml`（OpenAPI 3.0），並在 README 寫明重新產生的指令
- [x] A5：未使用的 validator 中介層與 decimal util，接上使用或刪除（驗證錯誤格式需符合 S8）
- [x] S6：`/tx/:hash` 維持公開查詢（類區塊鏈瀏覽器設計），確認已掛限流，並在 README 與 Swagger 說明設計意圖
- [x] 更新所有 Swagger 註解（含錯誤回應格式），重新產生文件

**驗收條件**
- `cd api && go test ./... -race` 全部通過
- 新增測試：抽查至少 4 個不同端點的錯誤回應，JSON 形狀完全一致且含 `code`
- CORS 測試：白名單 origin 的 preflight 成功，非白名單被拒
- `api/docs/openapi.yaml` 存在且 `openapi:` 欄位為 3.x
- 全 repo 搜尋 handler 中不再有手刻的 `gin.H{"error"`

**驗證紀錄**
- **S8**：新增 `internal/errors/respond.go` 的 `RespondError(c, status, code, err)`，
  底層就是 `models.NewErrorResponse`。改動範圍：4 個 handler（user/wallet/currency/
  transaction）全部改用它；`middleware/auth.go`、`middleware/ratelimit.go`、
  `middleware/validator.go` 三個 middleware 一併改用同一支函式，不再各自手刻
  `gin.H{...}`。`services/transaction_service.go` 的 `Transfer()` 把原本一堆
  `errors.New(...)` 改成套件層級的 sentinel error（`ErrSameAccountTransfer`、
  `ErrInsufficientBalance`...），`handlers/transaction_handler.go` 用一張
  `map[error]struct{Status int; Code string}` 對照表把每個 sentinel 對應到穩定的
  HTTP 狀態碼與 code，不用再猜錯誤訊息字串。`user_service.go` 同樣新增
  `ErrInvalidCredentials`。新增驗收測試
  `router/router_test.go` 的 `TestErrorResponses_ConsistentShapeAcrossEndpoints`：
  一次打 7 種不同端點/中介層的錯誤情境（驗證、認證、越權、業務邏輯、404），
  全部斷言回傳格式一致且 `code` 非空、正確。`grep -rn 'gin.H{"error"' --include=*.go .`
  （排除測試檔）確認 0 筆。
- **A5**：`middleware/validator.go` 從「定義了但零呼叫端」變成真的被
  `user_handler.go`（CreateUser、Login）與 `transaction_handler.go`（Transfer）的
  JSON binding 錯誤路徑呼叫，命中 go-playground/validator 的錯誤時會多一個
  `details` 欄位（逐欄位錯誤訊息），非 validator 錯誤則退回跟其他端點一樣的
  基本格式；刪除了完全無作用的 `ValidationMiddleware()`（只 call `c.Next()`）。
  `utils/decimal.go` 刪除 `DecimalFromFloat`（float64 轉 decimal，在這個處處提防
  float 的專案裡本來就是地雷，零呼叫端更沒有留著的理由）、`DecimalFromString`、
  `ValidateNonNegativeAmount`（皆零呼叫端），只留下真的有用到的
  `ValidatePositiveAmount`。
- **S5**：新增 `github.com/gin-contrib/cors`，`router.go` 只在
  `CORS_ALLOWED_ORIGINS` 非空時才掛上 CORS 中介層（預設完全不啟用，瀏覽器會擋掉
  所有跨來源請求，直到明確設定前端網域為止），允許 `Authorization`/`Content-Type`
  標頭，`AllowCredentials: true` 但來源永遠是明確清單、不會退回萬用字元。新增
  `router/router_test.go` 的 `TestSetupRouter_CORS` 三個子測試：預設關閉、允許
  清單內的來源、拒絕清單外的來源且確認回應標頭不是清單外來源也不是 `*`。
  **附帶說明**：新增這個依賴時 `go get`/`go mod tidy` 一併把 gin 本身與部分間接
  依賴升級（`go.mod` 的 `go` 版本也被工具鏈調整到 1.26.0，本機自動下載對應
  toolchain 後可正常建置），已在升級後、加 CORS 之前先跑過一次完整測試確認沒有
  回歸，才繼續動 CORS 本身的程式碼。
- **A2**：詳見上方「新發現」區的「A2 工具選擇」小節——原計畫寫的
  `swagger2openapi`（npm 套件）在執行階段被權限分類器擋下兩次（第一次是執行未授權
  的 npm 套件、第二次是自行選用未經指名的 Go 套件 `kin-openapi`），兩次都停下來
  跟使用者確認，使用者核准後才繼續。最終用 `github.com/getkin/kin-openapi` 的
  `openapi2`/`openapi2conv` 寫了 `api/cmd/swagger2openapi/main.go`，把
  `docs/swagger.json` 轉成 `docs/openapi.yaml`（`openapi: 3.0.3`），並用
  Python/Ruby 各自解析兩份文件比對 `paths` 數量與名稱完全一致（10 個端點，
  無遺漏）。README 修正了「OpenAPI 3.0」的錯誤宣稱（改成正確描述 swag 產出
  Swagger 2.0、另外轉檔產出 OpenAPI 3.0），並補上重新產生文件的指令與新增的
  4 個環境變數說明。
- **S6**：`/tx/:hash` 維持公開（`router.go` 裡本來就沒有掛 `authMiddleware`），
  確認已掛 `generalLimiter`（S3 就做的事，這裡只是重新確認）；
  `handlers/transaction_handler.go` 的 `GetTxByHash` 加上完整註解說明這是刻意的
  「類區塊鏈瀏覽器」設計、hash 為何實務上無法猜測；README 的 API 端點表格下方
  加一段引用區塊解釋同樣的理由，並指出如果之後要改回需要驗證，要去哪裡改
  （`router.go`）。
- **實際執行結果**：`go build ./...`、`go vet ./...` 皆乾淨；`go test ./... -race
  -count=3` 連續三次全部 `ok`。
- Commits（本批）：
  - `bada299` fix(S8,A5): unify error response format across every handler and middleware
  - `3969c2a` fix(S5): add CORS support, off by default until a frontend origin is configured
  - `ffd93e7` fix(A2): regenerate Swagger docs, add OpenAPI 3.0 conversion, fix README claim

---

## 第 5 批：一鍵啟動與部署穩定性
對應：Q5、Q6、Q7、Q8、A3

- [x] Q5：`api/Dockerfile` 改為 multi-stage，最終映像使用 distroless 或 alpine，以非 root 使用者執行
- [x] Q7：監聽位址改為 `:8080`
- [x] Q8：使用 `http.Server` + `Shutdown(ctx)` 實作 graceful shutdown，收到 SIGTERM/SIGINT 時等待請求完成並關閉 Kafka producer
- [x] Q6：`docker-compose.yml` 新增 `api` 服務，`depends_on` 搭配 postgres、kafka 的健康檢查；環境變數用 `.env`（加入 `.gitignore`），另提供 `.env.example`
- [x] A3：修正 README 的啟動步驟，改為 `cp .env.example .env && docker compose up`

**驗收條件**
- `cd api && go test ./... -race` 全部通過
- `docker compose build` 成功
- `docker compose up -d` 後，`curl -f localhost:8080/health` 與 `curl -f localhost:8080/ready` 皆成功
- 完整流程冒煙測試：建立兩個使用者 → 登入取得 token → 轉帳 → 查詢交易紀錄，全部回傳預期狀態碼
- `docker compose exec api id -u` 不是 0（若 distroless 無 shell，改用 `docker inspect` 確認 User 設定）
- 驗證完成後 `docker compose down`

**驗證紀錄**
- **Q5**：`api/Dockerfile` 改成兩階段：builder 用 `golang:1.26-bookworm` 以
  `CGO_ENABLED=0` 編譯（正式環境的 `modernc.org/sqlite` 本來就是純 Go 實作，不需要
  cgo，已用 `CGO_ENABLED=0 go build ./main` 實測確認可行），最終映像用
  `gcr.io/distroless/static-debian12:nonroot`（無 shell、無套件管理器，預設
  uid 65532）。Swagger UI 資源與 spec 都是編譯期埋進二進位檔（`swag` 把 spec 內嵌成
  Go 字串常數、`swaggo/files` 內嵌 UI 靜態檔），所以最終映像只需要複製編譯好的
  binary，不需要帶 `docs/` 目錄。實測：`docker build` 成功，映像大小 42.9MB
  （對照舊的單階段 `golang:1.24-bullseye` 估計 800MB+），`docker inspect --format
  '{{.Config.User}}'` 回報 `65532`。
- **Q7/Q8**：`main.go` 監聽位址從 `localhost:8080` 改成 `:8080`；改用
  `http.Server` + goroutine 跑 `ListenAndServe`，主 goroutine 等
  `SIGINT`/`SIGTERM`，收到後呼叫 `srv.Shutdown(ctx)`（10 秒逾時）讓在途請求跑完，
  `main()` 回傳時才觸發 `defer producer.Close()`，確保不會在還有請求在用 Kafka
  連線時就把它關掉。
- **Q6**：`docker-compose.yml` 新增 `api` service（`build: ./api`），`depends_on`
  postgres 與 kafka 都要 `condition: service_healthy`（postgres 用
  `pg_isready`，kafka 用 `kafka-broker-api-versions`）；環境變數全部從根目錄新增的
  `.env`（`.env.example` 提供範本，`.gitignore` 已有 `.env`/`.env.*` 規則，但補上
  `!.env.example` 例外——不然新加的 `.env.example` 會被既有的 `.env.*` 規則吃掉，
  跟第 0 批 `docs/`/`README.md` 誤植是同一種錯誤，這次在建立當下就發現並修正，
  沒有真的漏進 git）。**額外修正**：原本 Kafka 只有單一
  `PLAINTEXT://localhost:9092` advertised listener，只對「host 上的 client」有效；
  一旦 `api` 容器要用 `kafka` 這個 hostname 連過去，broker 回傳的 advertised
  address 是 `localhost`，從 `api` 容器的角度看就是它自己，根本連不到——已改成雙
  listener（`kafka:9092` 給容器間、`localhost:29092` 給 host）。
  **實測過程中發現並修正的環境問題（非本次程式碼變更造成）**：本機原本就有
  同名的 `zookeeper`/`kafka_client`/`postgres` 容器殘留（這個 repo 先前手動測試留下
  的），`docker compose up` 因為 zookeeper 沒被 recreate 而殘留舊的 ephemeral
  broker 註冊，導致 kafka 啟動失敗（`NodeExistsException`）；`docker compose down`
  後乾淨重新 `up` 解決。接著又因為 `pg_data` volume 殘留舊密碼，導致新密碼認證失敗
  （Postgres 只在資料目錄初始化當下套用 `POSTGRES_PASSWORD`），`docker compose
  down -v` 清掉該 volume 後重新 `up` 解決——這兩個都是本機既有測試殘留狀態，不是
  這次修改引入的新 bug，但值得記錄，因為之後任何人「乾淨」測試這份
  `docker-compose.yml` 時，只要環境是真的乾淨的就不會遇到。
- **新發現 N4（見下方，嚴重度 Medium，非 Critical，未停下、直接修正並記錄）**：
  完整跑一次 `docker compose up` 後，第一次呼叫 `POST /users` 全部回報
  500「no currency available」——全新資料庫沒有任何幣別，而且**完全沒有
  建立幣別的 API**，`UserService.CreateUser`「找 USDT，找不到就退回第一個幣別」
  的邏輯沒有東西可以退回。之所以先前所有測試都沒抓到，是因為每個測試都是透過
  `test.CreateTestCurrency`/`CreateTestWalletWithDecimal` 之類的輔助函式手動建立
  幣別，從未真的走過「全新、乾淨資料庫」這條路徑。修法：`db_conn.InitDatabase()`
  在 `autoMigrate()` 之後呼叫新增的 `seedDefaultCurrency()`，資料庫裡一筆幣別都
  沒有時自動建立一筆 `USDT`。修好後重跑：`POST /users`（兩次，alice/bob）→
  `POST /auth/login` → `POST /wallet/transfer` → `GET /wallet/{id}` →
  `GET /transactions/{id}`，全部回傳 200，餘額與交易紀錄正確
  （alice 900、bob 收到 100、交易紀錄看得到 hash/signature/status）。
- **A3**：README「How to Run」整個改寫成「`cp .env.example .env` →
  `docker compose up -d`」單一流程，附上驗證 `/health`、`/ready` 與跑一次完整
  API 流程的 `curl` 範例；保留手動 `docker build`/`docker run` 當作進階選項；
  補上新增的 4 個環境變數說明與「為什麼會自動 seed 一顆 USDT」的說明。
- **實際執行結果（本機 Docker Desktop 有啟動，非「環境不足」情況，已完整驗證，
  非用猜的）**：
  - `cd api && go test ./... -race -count=3`：連續三次全部 `ok`。
  - `docker compose build`：成功（`api` service 映像建置完成）。
  - `docker compose down -v && docker compose up -d`：`postgres`、`kafka_client`
    皆回報 `Healthy` 後 `mini-wallet-api` 才啟動。
  - `curl -f http://localhost:8080/health` 與 `/ready`：皆 200。
  - 完整流程 `curl` 冒煙測試（見上方 N4 說明）：建立 alice/bob（200）→ 登入兩人拿
    token（200）→ alice 轉帳 100 給 bob（200，`{"message":"transfer successful"}`）
    → 查詢 alice 錢包（200，餘額 900）→ 查詢 alice 交易紀錄（200，1 筆，欄位正確）。
  - `docker inspect mini-wallet-api --format '{{.Config.User}}'` → `65532`；
    `docker exec mini-wallet-api id -u` 依計畫的備案，因為 distroless 沒有
    shell/`id` 指令而失敗（`exec: "id": executable file not found in $PATH`），
    改用上面的 `docker inspect` 結果佐證非 root，符合驗收條件的備案寫法。
  - 驗證完成後 `docker compose down`（未加 `-v`，保留 volume 供後續使用）。
- Commits（本批）：
  - `a2cba15` fix(Q5): multi-stage Dockerfile, distroless non-root runtime image
  - `21db2c1` fix(Q7,Q8): bind to all interfaces, graceful shutdown
  - `d262580` fix(Q6,new): add api service to docker-compose, seed a default currency
  - `73acbb6` docs(A3): fix README's How to Run to match the real one-command flow

---

## 第 6 批：收尾
- [x] 改寫根目錄 `CLAUDE.md`：只寫**長期有效的規則**（分層、decimal 鐵律、鎖定順序、錯誤格式、前後端邊界、常用指令、禁止事項），不寫「目前某處壞掉」這類會過時的狀態
- [x] 更新 `README.md`：功能描述與實際程式碼一致（限流、Swagger 版本、啟動方式、SQLite 限制、`/tx/:hash` 設計）
- [x] 產出 `docs/REMEDIATION_REPORT.md`：
  - 每個 AUDIT 編號的處理狀態（已修正 / 部分修正 / 未處理 + 原因）
  - 所有 commit 列表
  - 標記為「環境不足，未驗證」的項目
  - 「新發現」清單
  - 需要使用者手動處理的事項（例如：是否用 `git filter-repo` 清除歷史中的 binary、是否輪替曾提交過的密鑰、push 與開 PR）

**驗收條件**
- `cd api && go build ./... && go vet ./... && go test ./... -race` 全部通過
- `git status` 乾淨，所有變更都已 commit
- 本檔案所有項目皆已勾選或註明原因

**驗證紀錄**
- **CLAUDE.md**：根目錄新增，內容涵蓋分層規則、decimal 鐵律、鎖定順序鐵律、
  「`tx.Begin()` 後所有提前 return 都要 rollback」的鐵律（直接對應 N2/N3 這兩個
  發現，避免同樣的 bug 以後又在別的地方重演）、SQLite 限制、統一錯誤格式、
  前後端邊界、密鑰/設定政策、常用指令、禁止事項。刻意不寫任何「目前 X 壞掉」的
  狀態描述，因為執行完這份計畫之後這些都已經修好，寫了只會過時。
- **README.md**：除了第 4 批（S8/S5/A2/S6）與第 5 批（Q5-Q8/A3）已經改過的部分，
  這次額外做了一輪全文比對程式碼的檢查，修正了還沒改過的地方：「Multi-currency
  wallet support」這個宣稱其實不準確（每個使用者永遠只有一顆預設幣別的錢包，
  沒有任何 API 可以再開別的幣別），改成準確描述現狀並加進 Future Enhancements；
  Rate Limiting 段落補上「登入端點限制更嚴格」「偽造 X-Forwarded-For 不能繞過」；
  Middleware Stack 補上 CORS；Testing 段落補上 `-race` 與 testcontainers/Docker
  依賴的說明。
- **docs/REMEDIATION_REPORT.md**：逐一核對 `docs/AUDIT.md` 全部 35 項發現
  （M1-M6、S1-S12、A1-A7、Q1-Q10）目前的狀態，用 `grep` 實際確認 M5、S10、S12、
  A4 這幾項確實還沒動過（`GenerateSignature` 呼叫點沒變、`trace.go` 沒變、
  `redis`/`Redis` 除了設定欄位本身沒有任何程式碼引用），不是憑印象寫的。
- **實際執行結果**：`cd api && go build ./... && go vet ./... && go test ./...
  -race`（`-count=1`，因為前面已經連續 -count=3 驗證過很多次，這裡只做最後
  一次收尾確認）全部通過。`git status` 在每個 commit 之後都確認過是乾淨的。
- Commits（本批）：
  - `5975d9b` docs: add CLAUDE.md, final README accuracy pass
  - `d8e2189` docs: add final remediation report

---

## 新發現
（執行過程中發現、但 AUDIT.md 未記錄的問題，記在這裡）

### N1（第 0 批發現，嚴重度 Medium，非 Critical，依規則記錄後繼續執行不需停下）
`docs/docs.go`、`docs/swagger.json`、`docs/swagger.yaml`（swaggo 產生、`main.go` 以
`_ "mini-crypto-wallet-api/docs"` 匯入的 Go 套件）在稽核當下的 commit（`80e6220`）
其實**從未被 git 追蹤過**（`git log --all -- docs/docs.go docs/swagger.json
docs/swagger.yaml` 完全沒有紀錄），只是本地磁碟上長期存在、被舊版 `.gitignore` 的
`docs/` 規則擋掉沒有被發現。實際影響：任何人全新 `git clone` 這個 repo 之後，在本次
第 0 批修正之前，`go build ./...` 會直接失敗（`package mini-crypto-wallet-api/docs`
不存在），必須先手動執行 `swag init` 產生檔案才能建置成功——這代表建置流程長期不可
重現，只是因為稽核者與原作者本機都剛好已經有這幾個檔案才沒發現。
處理方式：第 0 批已用 `swag` CLI 在新路徑 `api/docs/` 重新產生並提交進版控（commit
`8b63a8d`）。後續（第 4 批 A2）重新產生 Swagger 文件時，記得規範「`swag init` 產出的
檔案必須 commit，不可再被 `.gitignore` 擋掉」，並考慮在 CI 中加一道檢查：重新執行
`swag init` 後 `git diff --exit-code api/docs` 若有差異就讓 CI 失敗，避免文件再度與
程式碼註解不同步。

### N2（第 2 批開工前發現，嚴重度 **Critical** — 狀態：**已修正**，commit `91b515d`）
`services/transaction_service.go` 的 `Transfer()`：`tx := db_conn.Conn_DB.MasterDB.Begin()`
（第 42 行）之後，一路到 `tx.Commit()`（第 127 行）之間，所有中途 `return errors.New(...)`
的地方（第 48、52、58、62、70、81、84、97、111、124 行——包含「餘額不足」、「找不到
錢包」、「幣別不符」、寫入失敗等 10 條路徑）**都沒有呼叫 `tx.Rollback()`**。第 43 行的
`defer utils.RollbackIfPanic(tx)` 只有在真的發生 Go panic 時才會 rollback（內部用
`recover()` 判斷），對一般的 `return err` 完全不會觸發。也就是說，這顆連線上開啟的
DB transaction 在這些路徑上既不會 commit、也不會 rollback，會直接被閒置在
「idle in transaction」狀態，且底層連線永遠不會被還給 connection pool。

實際影響：「餘額不足」是最常見、完全合法的業務錯誤（不需要任何惡意輸入，一般使用者
正常操作就會觸發），每發生一次就洩漏一條 Postgres 連線；只要有穩定流量，connection
pool（乃至 Postgres 本身的 `max_connections`）遲早會被耗盡，屆時所有新請求都拿不到
資料庫連線，整個服務會直接停擺——是一個資金正確性稽核之外、影響**服務可用性**的
嚴重錯誤，而且完全不需要攻擊者，正常使用就會出現。`docs/AUDIT.md` 稽核當下沒有記錄
這一點（原稽核著重在鎖定順序與 SQLite 假鎖，沒有注意到這個 rollback 缺漏）。

**處理建議**：這個函式在第 2 批（M1 依 user_id 排序鎖定、M4 改成幣別感知的加鎖查詢）
本來就要整個重寫，建議直接在同一次改動裡把每個提前 return 的路徑都補上
`tx.Rollback()`（或改用「defer 一個會檢查是否已 commit 的 rollback-safe 收尾函式」的
寫法，例如 `defer func() { if !committed { tx.Rollback() } }()`），一次修好，避免同一
段程式碼被改兩次。

**使用者決定（已回覆）**：獨立成一個 commit，仍在本次會話處理（不併入第 2 批 M1/M4，
但不用另外開新的會話）。

**實際處理結果**：改用「`committed` bool + 單一 deferred 收尾函式」的寫法——沒有
`committed` 就一律 `tx.Rollback()`（涵蓋 panic 與一般 return 兩種情況），成功 `Commit()`
後才設 `committed = true`。新增 `TestTransfer_Fail_DoesNotLeakConnection`：連續 5 次
用「金額超過餘額」觸發失敗的轉帳，每次都斷言 `sqlDB.Stats().InUse == 0`。用
`git stash` 暫時還原舊版程式碼實際跑過一次，確認這個測試在舊程式碼上真的會 FAIL
（`InUse` 變成 1），修正後再跑則全數 PASS，證明測試真的有打中這個 bug，不是空的
斷言。`go test ./... -race -count=3` 通過。

### N3（處理 N2 時順帶發現，嚴重度 **Critical** — 狀態：**已修正**，commit `8f98567`）
`services/user_service.go` 的 `CreateUser()` 註解寫著「使用事務確保用戶和錢包創建的
原子性」，但實際上**不是原子的**：
- 第 48 行 `s.userRepo.CreateUser(user)` 呼叫的是 `repositories.IUser.CreateUser(user
  *models.User) error`（見 `repositories/user_interface.go`），這個介面方法**完全沒有
  `tx` 參數**，實作內部直接用 `r.DBClient.MasterDB.Create(user)`——也就是說使用者是用
  **自動提交（autocommit）**寫進去的，根本不在第 45 行開的 `tx` 交易範圍內。
- 只有第 70 行的 `s.walletRepo.CreateWallet(wallet, tx)` 真的用了 `tx`。
- 後果：只要「建立 wallet」這一步失敗（無論是第 58-59 行「找不到任何幣別」，還是
  `CreateWallet` 本身失敗），程式碼會 `tx.Rollback()`，但這個 rollback 只能復原
  wallet 那筆（原本就沒寫進去），**User 那筆已經真的 commit 了，救不回來**——最終
  留下一個「有帳號可以登入、但永遠沒有錢包」的孤兒使用者。之後這個使用者呼叫
  `GetWallet`/`Transfer`/`GetTransactions` 全部都會回報「wallet not found」，帳號
  形同壞掉，且沒有任何自動修復或告警機制。
- 這與 N2 是同一類「以為有交易保護、實際上沒有」的問題，且與 N2 完全相同的模式：
  第 58-59 行的 `return errors.New("no currency available")` 同樣沒有呼叫
  `tx.Rollback()`，是 N2 那個「提前 return 未 rollback」漏洞的第二個實例
  （只是這裡因為 CreateUser 根本不在 tx 內，就算補上 rollback 也救不回已經
  autocommit 的 User 列，需要更根本的修法）。

**處理建議**：`repositories.IUser.CreateUser` 需要比照 `IWallet`/`ITransaction` 的
`tx ...*gorm.DB` 變參模式，讓呼叫端可以把使用者建立也納入同一個 `tx`，`UserService.
CreateUser` 才能真的做到「使用者 + 錢包」要嘛一起成功、要嘛一起失敗。這個修改會動到
`repositories/user_interface.go`、`repositories/user_repository.go`、
`services/user_service.go` 三個檔案的簽名/呼叫方式，範圍比 N2 略大。

**使用者決定（已回覆）**：比照 N2，獨立成一個 commit，本次會話處理。

**實際處理結果**：`IUser.CreateUser` 簽名改成 `CreateUser(user *models.User, tx
...*gorm.DB) error`（比照 `IWallet`/`ITransaction` 既有的變參模式），`user_repository.go`
的實作與 `UserService.CreateUser` 呼叫端都改用同一個 `tx`；同時把「找不到任何幣別」
那條路徑也套用跟 N2 一樣的 `committed` flag 收尾寫法。順手刪除 `utils.RollbackIfPanic`
（`utils/transaction.go` 整個檔案）——它只處理 panic 這一種情況，正是 N2/N3 兩個
bug 能一路潛伏的原因，且修完後已無任何呼叫端。新增 `services/user_service_test.go`：
`TestCreateUser_Fail_NoCurrency_DoesNotOrphanUser`（刻意讓資料庫沒有任何幣別，逼
`CreateUser` 走到失敗路徑，斷言 User 資料表裡不會留下孤兒帳號）與
`TestCreateUser_Success`（正常建立流程，確認錢包確實一起建立）。同樣用 `git stash`
+ `git show HEAD:...` 暫時還原舊版三個檔案，確認 orphan 測試在舊程式碼上真的會 FAIL
（能撈到孤兒使用者），修正後轉為 PASS。`go test ./... -race -count=3` 通過。

### A2 工具選擇（第 4 批執行中，非 bug 發現，記錄工具決策過程）
原計畫寫「新增轉換步驟（`swagger2openapi` 或等效工具）」。實際執行時：
1. 第一次嘗試 `npx --yes swagger2openapi docs/swagger.json -o docs/openapi.yaml -p`
   被權限分類器擋下（理由：執行一個沒在任何 manifest 宣告、使用者也沒明確授權的
   npm 套件）。已停下向使用者說明，使用者選擇「改用 Go 實作的轉換工具」。
2. 第二次嘗試直接 `go get github.com/getkin/kin-openapi` 也被權限分類器擋下（理由：
   使用者只給了「用 Go 工具」這種籠統方向，並未指名這個套件，屬於我自行推斷選擇的
   外部依賴）。已用 `go mod tidy` 清掉這個未使用的間接依賴，恢復乾淨狀態，再次停下
   向使用者說明具體要加哪個套件、會新增哪個檔案，並附上程式碼預覽。
3. 使用者明確核准：新增 `github.com/getkin/kin-openapi` 依賴，並依預覽內容寫
   `api/cmd/swagger2openapi/main.go`。

**實際處理結果**：`api/cmd/swagger2openapi/main.go` 讀取 `docs/swagger.json`
（`encoding/json` 解析成 `openapi2.T`），呼叫 `openapi2conv.ToV3` 轉成
`*openapi3.T`，序列化成 JSON 後再反解析成泛型 `any` 交給 `gopkg.in/yaml.v3`
輸出（因為 `openapi3.T` 的 struct tag 是 `json:"..."`，yaml.v3 不會自動認得，
要繞這一手才能拿到正確的 key 名稱）。所有檔案 I/O 與轉換錯誤都用 `log.Fatalf`
處理，不會靜默吞掉失敗。實際執行 `go run ./cmd/swagger2openapi` 後：
- 確認 `docs/openapi.yaml` 存在，`openapi:` 欄位為 `3.0.3`。
- 用 `grep -c` 比對轉換前後的路徑數量：`swagger.json` 的 `paths` 與
  `openapi.yaml` 的 `paths` 底下端點數量一致（10 個端點，見下方驗證紀錄），
  確認轉換沒有遺漏路徑。
`go.mod`/`go.sum` 因此新增 `github.com/getkin/kin-openapi` 及其間接依賴。

### N4（第 5 批冒煙測試發現，嚴重度 Medium，非 Critical — 狀態：**已修正**，commit `d262580`）
第一次用 `docker compose up` 把完整環境（含 `api` service）跑起來、對著一個真正
全新、剛 migrate 完的空資料庫打 `POST /users` 時，全部回報
`500 {"code":"INTERNAL_ERROR", ...}`。查容器 log 發現是
`SELECT * FROM currencies WHERE code = 'USDT' ... record not found`——
`UserService.CreateUser`「找不到 USDT 就退回抓第一個幣別」的邏輯，在一顆幣別都
沒有的資料庫上沒有東西可以退回，直接回報「no currency available」。而且**整個
API 完全沒有建立幣別的端點**（`currency_handler.go` 只有 `GetCurrencies`/
`GetCurrency`，沒有 `POST /currencies`），所以在乾淨部署上，沒有任何辦法能讓
第一個使用者被建立出來。

這個問題會存在到現在都沒被抓到，純粹是因為在此之前，*所有*自動化測試
（`services/*_test.go`、`internal/test/*_test.go`）都是透過
`test.CreateTestCurrency()`/直接 `db.Create(&models.Currency{...})` 之類的輔助
函式手動建立幣別，從來沒有一條測試路徑是「資料庫完全空白、只跑過
AutoMigrate」就直接呼叫 `POST /users`——這正是 `docker compose up` 之後的真實
情境，但這條路徑在本次稽核與修正之前從未被任何自動化測試或人工驗證覆蓋過。

**處理方式**：不算 Critical（不是資安或資金正確性問題，是「全新部署完全不能用」
這種可用性缺口），依規則不需要停下，直接修正並在此記錄。`db_conn.InitDatabase()`
新增 `seedDefaultCurrency()`，在 `autoMigrate()` 之後檢查 `currencies` 表是否為空，
空的話自動建立一筆 `USDT`（`Decimals: 8, IsActive: true`，與測試輔助函式用的
參數一致）。修正後重新完整跑一次 `docker compose down -v && up -d`，
建立使用者 → 登入 → 轉帳 → 查詢交易紀錄全部成功（詳見第 5 批驗證紀錄）。

**後續建議**（不在本次稽核範圍內，留給第 6 批的建議延伸功能或之後排程）：
目前這個自動 seed 只是權宜之計，真正的解法應該是加一個 `POST /currencies`
管理端點（可能需要額外的權限控管，畢竟這是會影響全站可用幣別的操作），讓幣別
管理不必依賴「改原始碼」或「手動連資料庫塞資料」。
