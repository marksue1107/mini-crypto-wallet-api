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

- [ ] S8：所有 handler 錯誤統一回傳 `ErrorResponse{error, code, message}`，`code` 使用 `internal/errors` 常數；middleware（驗證、限流、JWT）的錯誤也統一成相同格式
- [ ] S5：加入 `gin-contrib/cors`，允許的 origin 由環境變數 `CORS_ALLOWED_ORIGINS` 設定（逗號分隔），允許 `Authorization`、`Content-Type` 標頭，不對帶憑證請求使用 `*`
- [ ] A2：README 改為正確描述 Swagger 2.0；新增轉換步驟（`swagger2openapi` 或等效工具）產出 `api/docs/openapi.yaml`（OpenAPI 3.0），並在 README 寫明重新產生的指令
- [ ] A5：未使用的 validator 中介層與 decimal util，接上使用或刪除（驗證錯誤格式需符合 S8）
- [ ] S6：`/tx/:hash` 維持公開查詢（類區塊鏈瀏覽器設計），確認已掛限流，並在 README 與 Swagger 說明設計意圖
- [ ] 更新所有 Swagger 註解（含錯誤回應格式），重新產生文件

**驗收條件**
- `cd api && go test ./... -race` 全部通過
- 新增測試：抽查至少 4 個不同端點的錯誤回應，JSON 形狀完全一致且含 `code`
- CORS 測試：白名單 origin 的 preflight 成功，非白名單被拒
- `api/docs/openapi.yaml` 存在且 `openapi:` 欄位為 3.x
- 全 repo 搜尋 handler 中不再有手刻的 `gin.H{"error"`

**驗證紀錄**
（待填寫）

---

## 第 5 批：一鍵啟動與部署穩定性
對應：Q5、Q6、Q7、Q8、A3

- [ ] Q5：`api/Dockerfile` 改為 multi-stage，最終映像使用 distroless 或 alpine，以非 root 使用者執行
- [ ] Q7：監聽位址改為 `:8080`
- [ ] Q8：使用 `http.Server` + `Shutdown(ctx)` 實作 graceful shutdown，收到 SIGTERM/SIGINT 時等待請求完成並關閉 Kafka producer
- [ ] Q6：`docker-compose.yml` 新增 `api` 服務，`depends_on` 搭配 postgres、kafka 的健康檢查；環境變數用 `.env`（加入 `.gitignore`），另提供 `.env.example`
- [ ] A3：修正 README 的啟動步驟，改為 `cp .env.example .env && docker compose up`

**驗收條件**
- `cd api && go test ./... -race` 全部通過
- `docker compose build` 成功
- `docker compose up -d` 後，`curl -f localhost:8080/health` 與 `curl -f localhost:8080/ready` 皆成功
- 完整流程冒煙測試：建立兩個使用者 → 登入取得 token → 轉帳 → 查詢交易紀錄，全部回傳預期狀態碼
- `docker compose exec api id -u` 不是 0（若 distroless 無 shell，改用 `docker inspect` 確認 User 設定）
- 驗證完成後 `docker compose down`

**驗證紀錄**
（待填寫）

---

## 第 6 批：收尾
- [ ] 改寫根目錄 `CLAUDE.md`：只寫**長期有效的規則**（分層、decimal 鐵律、鎖定順序、錯誤格式、前後端邊界、常用指令、禁止事項），不寫「目前某處壞掉」這類會過時的狀態
- [ ] 更新 `README.md`：功能描述與實際程式碼一致（限流、Swagger 版本、啟動方式、SQLite 限制、`/tx/:hash` 設計）
- [ ] 產出 `docs/REMEDIATION_REPORT.md`：
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
（待填寫）

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
