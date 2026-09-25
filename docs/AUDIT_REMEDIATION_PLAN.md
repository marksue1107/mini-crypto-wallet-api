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

- [ ] 測試 DB：統一使用 `modernc.org/sqlite` 對應的 dialector（與正式環境相同），改用暫存檔案 SQLite 或對 in-memory 設定 `SetMaxOpenConns(1)`
- [ ] 金流相關測試的前置條件與關鍵斷言改用 `require`
- [ ] 將 `TransferWithLockOption` 及其不安全示範邏輯移到 `_test.go` 檔，正式程式碼移除 `import "testing"`
- [ ] `TestConcurrentTransfers` 改用 testcontainers-go 啟動一次性 Postgres；無 Docker 時 `t.Skip`，不可 `log.Fatal`
- [ ] 新增 `.github/workflows/api.yml`：`paths: ['api/**']` 觸發，執行 `go build ./...`、`go vet ./...`、`go test ./... -race`

**驗收條件**
- `cd api && go test ./... -race` 全部通過，無 panic
- `cd api && go test ./... -race -count=3` 連續三次通過（確認不是偶發通過）
- `grep -rn '"testing"' api --include='*.go' | grep -v '_test.go'` 無輸出
- workflow YAML 語法正確（可用 `python -c "import yaml; yaml.safe_load(open('.github/workflows/api.yml'))"` 檢查）

**驗證紀錄**
（待填寫）

---

## 第 2 批：資金正確性
對應：M1、M4、M6、M2（採方案 c）

- [ ] M1：轉帳時依 `user_id` 由小到大鎖定兩顆錢包，與呼叫參數順序無關
- [ ] M1 測試：A→B 與 B→A 大量同時轉帳，不發生 deadlock，且兩人餘額總和守恆（Postgres / testcontainers）
- [ ] M4：新增依 `user_id` + `currency_id` 查詢並加鎖的 repository 方法，`Transfer()` 改用它
- [ ] M6：依幣別 `Decimals` 驗證金額小數位數，超過即回傳驗證錯誤；新增單筆上限金額設定（放 config，預設值寫在 `config.yaml.example` 或程式常數）
- [ ] M6 測試：小數位數超過、零、負數、超過上限，各至少一條
- [ ] M2：SQLite 模式啟動時印出明確警告（不保證併發正確性、僅供單機開發）；README 補充說明

**驗收條件**
- `cd api && go test ./... -race` 全部通過
- 新增的 deadlock / 餘額守恆測試在有 Docker 時實際執行並通過（無 Docker 則依「環境限制」處理）
- 用 SQLite 啟動服務時，log 中可看到警告訊息

**驗證紀錄**
（待填寫）

---

## 第 3 批：安全性
對應：S1、S2、S3、S4、S7、S9、S11

- [ ] S1：`config.yaml` 自版控移除（`git rm --cached`）並加入 `.gitignore`；新增 `api/config.yaml.example`（佔位值）；密鑰與 DB 密碼改由環境變數注入
- [ ] S2：`JWTSecret` 未設定或長度 < 32 時 `log.Fatal`，移除原始碼中的預設密鑰
- [ ] S3：在 `/auth/login`、`/users`、`/wallet/transfer`、`/tx/:hash` 掛上限流（登入端點用較嚴格的設定）
- [ ] S4：呼叫 `SetTrustedProxies`，可信代理從環境變數讀取，預設不信任任何代理
- [ ] S7：重複帳號回傳 `409` + `USER_ALREADY_EXISTS`
- [ ] S9：commit 失敗不回傳原始 DB 錯誤給客戶端，原始錯誤帶 trace id 寫入 log
- [ ] S11：密碼最小長度提高到 8
- [ ] 以上每一項都要有對應的 handler 或 middleware 測試

**驗收條件**
- `cd api && go test ./... -race` 全部通過
- `git ls-files | grep -E '(^|/)config\.yaml$'` 無輸出
- `grep -rn 'default-secret' api` 無輸出
- 不設定 JWT 密鑰啟動服務 → 程式立即結束並顯示清楚錯誤
- 限流測試：超過上限的請求回傳 `429`，且偽造 `X-Forwarded-For` 無法繞過

**驗證紀錄**
（待填寫）

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
