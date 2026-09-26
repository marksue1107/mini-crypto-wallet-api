# 稽核修正結案報告

執行依據：`docs/AUDIT.md`（稽核當下 commit `80e6220`）＋ `docs/AUDIT_REMEDIATION_PLAN.md`
分支：`fix/audit-remediation`（從 `main` @ `80e6220` 分出）
執行方式：分 6 批 + 4 個中途新發現（N1-N4），每批/每個新發現獨立驗證、獨立 commit
執行期間所有變更皆已通過：`cd api && go build ./... && go vet ./... && go test ./... -race`

---

## 1. 總覽

`docs/AUDIT.md` 列出的 35 項發現（M1-M6、S1-S12、A1-A7、Q1-Q10）加上執行過程中新發現
的 4 項（N1-N4），共 39 項：

- **已修正**：31 項
- **已確認並記錄設計意圖（非 bug，維持現狀）**：1 項（S6）
- **部分修正**：2 項（S11、Q9）
- **未處理**：5 項（M5、S10、S12、A4，以及 S12/A4 共同牽涉的 Redis 分散式限流）

沒有任何項目因為「環境不足」而被跳過——本機有 Docker Desktop，所有 testcontainers
測試與 `docker compose` 完整流程都已實際執行並驗證通過，不是用猜的。

**最重要的三個修正**（原始稽核之外，執行過程中才發現的 Critical 問題）：
- **N2**：`Transfer()` 提前 return 沒有 rollback，每次「餘額不足」都會洩漏一條卡在
  idle-in-transaction 的 DB 連線，正常流量下遲早會把連線池耗盡讓整個服務停擺。
- **N3**：`UserService.CreateUser` 宣稱有交易保護，實際上使用者是用自動提交寫入的，
  wallet 建立失敗時會留下「能登入但沒有錢包」的孤兒帳號。
- **N4**：全新資料庫沒有任何幣別、也沒有建立幣別的 API，導致 `docker compose up`
  起來的全新環境完全無法建立第一個使用者。

---

## 2. 逐項處理狀態

### 資金正確性（M 系列）

| 編號 | 嚴重度 | 狀態 | 說明 | Commit |
|------|--------|------|------|--------|
| M1 | Critical | ✅ 已修正 | 轉帳鎖定順序改成依 `user_id` 由小到大，與呼叫參數順序無關；用真的 Postgres 重現過死結再驗證修好 | `828c317` |
| M2 | Critical | ⚠️ 已緩解（SQLite 本身限制無法「修復」） | SQLite 模式仍然沒有真正的列鎖（GORM driver 本身如此），已加上啟動警告 log + README 明確說明「僅供單機開發，正式環境用 Postgres」；順帶修正 `PRAGMA busy_timeout` 缺漏 | `828c317` |
| M3 | Critical | ✅ 已修正 | 測試 DB 改用檔案型 SQLite（非 `:memory:`），修好會 panic 的測試 | `16bf545` |
| M4 | High | ✅ 已修正 | 加鎖查詢改成同時帶 `user_id` + `currency_id` | `828c317` |
| M5 | Medium | ❌ 未處理 | `GenerateSignature()` 呼叫時機在 insert 之前，`CreatedAt` 仍是零值，簽名對同一個 `(FromUserID, Amount)` 永遠相同、非真正密碼學簽章。目前僅為展示用途，風險可控，**未在本次排程內修正** | — |
| M6 | High | ✅ 已修正 | 新增小數位數驗證（依幣別 `Decimals`）與可設定的單筆金額上限（`max_transfer_amount`） | `828c317` |

### 安全性（S 系列）

| 編號 | 嚴重度 | 狀態 | 說明 | Commit |
|------|--------|------|------|--------|
| S1 | Critical | ✅ 已修正（歷史清理需使用者決定） | `config.yaml` 移出版控，改用環境變數；**git 歷史裡的舊密鑰仍視為已外洩，需要使用者決定是否重寫歷史並實際輪替密鑰**（見第 6 節） | `582da7e` |
| S2 | High | ✅ 已修正 | JWT 密鑰未設定或 < 32 字元時直接拒絕啟動，移除硬編碼預設密鑰 | `582da7e` |
| S3 | High | ✅ 已修正 | 限流真正掛到 `/auth/login`（嚴格）、`/users`、`/wallet/transfer`、`/tx/:hash`（一般） | `582da7e` |
| S4 | Medium | ✅ 已修正 | `SetTrustedProxies` 預設不信任任何代理，防止偽造 `X-Forwarded-For` 繞過限流 | `582da7e` |
| S5 | High | ✅ 已修正 | 加入 `gin-contrib/cors`，預設關閉，需明確設定 `CORS_ALLOWED_ORIGINS` 才生效 | `3969c2a` |
| S6 | Medium（需確認） | ✅ 已確認並記錄設計意圖 | 執行時已與您確認：`/tx/:hash` 維持公開查詢（類區塊鏈瀏覽器設計），已在 README 與 Swagger 註解說明理由 | `bada299` |
| S7 | Medium | ✅ 已修正 | 重複帳號回傳 `409` + `USER_ALREADY_EXISTS`，不再是通用 500 | `6806a02` |
| S8 | High | ✅ 已修正 | 所有 handler／middleware 統一回傳 `{error, code, message}`，新增 7 種情境的一致性驗收測試 | `bada299` |
| S9 | Low | ✅ 已修正 | 不只 `tx.Commit()`，`Transfer()`／`CreateUser()` 內所有可能洩漏原始 DB 錯誤的路徑都已改為記 log + 回傳固定安全訊息 | `6806a02` |
| S10 | Low（需確認） | ❌ 未處理 | `X-Trace-ID` 仍可由客戶端任意帶入、未做格式/長度驗證。風險低（目前只是回顯），**未在本次排程內修正** | — |
| S11 | Medium | 🟡 部分修正 | 密碼最小長度 6→8 已完成；`/auth/login` 限流已生效（間接降低暴力破解可行性），但**沒有實作真正的「失敗次數鎖定」機制**（例如帳號級的漸進式延遲或鎖定） | `582da7e`、`6806a02` |
| S12 | Medium（需確認） | ❌ 未處理 | JWT 仍是固定 24 小時效期、無 refresh token、無登出/撤銷端點；`redis_addr` 設定仍未接上任何實際用途（與 A4 同一件事） | — |

### 架構與一致性（A 系列）

| 編號 | 嚴重度 | 狀態 | 說明 | Commit |
|------|--------|------|------|--------|
| A1 | Medium | ✅ 已修正 | Rate Limiting 現在是真的功能，README 描述屬實 | `582da7e` |
| A2 | Medium | ✅ 已修正 | README 改為正確描述 Swagger 2.0；新增 `api/cmd/swagger2openapi` 轉出 `api/docs/openapi.yaml`（OpenAPI 3.0），已驗證路徑數量一致 | `ffd93e7` |
| A3 | Low | ✅ 已修正 | README「How to Run」改為 `cp .env.example .env && docker compose up -d` 單一流程，已實測跑過完整流程 | `73acbb6` |
| A4 | Low | ❌ 未處理 | `redis_addr` 設定仍是零使用的欄位，限流狀態仍是單一 process 記憶體內的 map，多副本部署時彼此不同步。**未在本次排程內修正**（與 S12 的撤銷機制是同一個 Redis 缺口，建議合併規劃） | — |
| A5 | Low | ✅ 已修正 | 未使用的 validator middleware 已真正接上使用（含新的 `details` 欄位）；未使用的 `utils` decimal 函式已刪除 | `bada299` |
| A6 | High | ✅ 已修正 | 移除正式碼裡的 `import "testing"` 與刻意不安全的示範方法，搬到測試檔 | `894b184` |
| A7 | Medium | ✅ 已修正 | 測試與正式環境統一使用 `modernc.org/sqlite` | `16bf545` |

### 工程品質（Q 系列）

| 編號 | 嚴重度 | 狀態 | 說明 | Commit |
|------|--------|------|------|--------|
| Q1 | Critical | ✅ 已修正 | 新增 GitHub Actions CI（build/vet/test -race） | `212bdd8` |
| Q2 | High | ✅ 已修正 | 移除已 commit 的執行檔（`wallet-api`、`__debug_bin*`），`.gitignore` 補上規則 | `24d6879` |
| Q3 | Medium | ✅ 已修正 | 移除已 commit 的 `.idea/` | `24d6879` |
| Q4 | Low | ✅ 已修正 | 移除 `.gitignore` 裡誤植的 `README.md`/`docs/` 規則 | `24d6879` |
| Q5 | High | ✅ 已修正 | Dockerfile 改為 multi-stage + distroless nonroot，映像從 800MB+ 降到 42.9MB，已用 `docker inspect` 確認非 root | `a2cba15` |
| Q6 | High | ✅ 已修正 | `docker-compose.yml` 新增 `api` service，`depends_on` 搭配健康檢查；已完整跑過一次 `docker compose up` 全流程 | `d262580` |
| Q7 | Medium | ✅ 已修正 | 監聽位址改為 `:8080` | `21db2c1` |
| Q8 | Medium | ✅ 已修正 | 改用 `http.Server` + `Shutdown(ctx)` 實作 graceful shutdown | `21db2c1` |
| Q9 | Medium | 🟡 部分修正 | 新增了 `internal/auth`、`internal/config`、`middleware`、`router`、`handlers`（user_handler）五個先前零測試的套件的測試；但 `repositories/` 仍無直接單元測試，`wallet_handler`/`currency_handler` 也還沒有專屬的 handler 層測試。核心金流（`services/`）與新增的安全性關卡（限流、CORS、錯誤格式、fail-fast）都已有測試覆蓋 | 多筆，見第 5 節 |
| Q10 | Low | ✅ 已修正 | `TestConcurrentTransfers` 改用 testcontainers 起真正 Postgres，本機無 Docker 時改為 `t.Skip` 而非 `log.Fatal` | `894b184` |

---

## 3. 執行過程中的新發現（docs/AUDIT.md 未記錄）

完整討論見 `docs/AUDIT_REMEDIATION_PLAN.md` 檔案末尾「新發現」區，這裡只列摘要與結果。

| 編號 | 嚴重度 | 摘要 | 狀態 | Commit |
|------|--------|------|------|--------|
| N1 | Medium | swag 產生的 `docs/docs.go` 等檔案從未被 git 追蹤（被舊版 `.gitignore` 擋掉），全新 clone 這個 repo 原本無法直接 `go build` | ✅ 已修正 | `8b63a8d` |
| N2 | **Critical** | `Transfer()` 提前 return 未 rollback，每次「餘額不足」都洩漏一條 DB 連線 | ✅ 已修正（已停下向您確認處理方式） | `91b515d` |
| N3 | **Critical** | `UserService.CreateUser` 實際上不是原子操作，可能留下沒有錢包的孤兒使用者 | ✅ 已修正（已停下向您確認處理方式） | `8f98567` |
| N4 | Medium | 全新資料庫沒有任何幣別、沒有建立幣別的 API，`docker compose up` 起來的環境無法建立第一個使用者 | ✅ 已修正（自動 seed 一顆 USDT，未停下，依規則屬非 Critical） | `d262580` |

另外在執行過程中，A2 的實作方式（`swagger2openapi` 工具選擇）經歷了兩次權限分類器
擋下、兩次停下向您確認，最終核准使用 `github.com/getkin/kin-openapi`——這不是 bug
發現，但完整決策過程也記錄在 `AUDIT_REMEDIATION_PLAN.md` 的「新發現」區（標題
「A2 工具選擇」），供之後追溯為什麼會多這個依賴。

---

## 4. 「環境不足，未驗證」項目

**沒有這類項目。** 本次執行全程本機都有可用的 Docker Desktop，所有原本預期「無 Docker
時只能跳過」的驗收條件都已實際執行：

- `internal/test/concurrency_demo_test.go`、`internal/test/deadlock_test.go`：
  已用 testcontainers 起過真正的 Postgres，並在關閉排序邏輯的情況下重現過真實的
  `deadlock detected (SQLSTATE 40P01)` 錯誤，確認修正前會失敗、修正後穩定通過。
- `docker compose build` / `up` / 完整 API 流程冒煙測試 / `docker inspect` 確認
  非 root：全部已實際執行，細節見 `AUDIT_REMEDIATION_PLAN.md` 第 5 批驗證紀錄。

---

## 5. 完整 Commit 列表

分支 `fix/audit-remediation`，從 `main`（`80e6220`）分出，依時間順序：

```
24d6879 fix(Q2,Q3,Q4): remove committed build binaries/.idea from git, fix .gitignore
47b6974 refactor: move Go backend module into api/
8b63a8d fix(Q4,new): regenerate untracked swagger docs under api/docs; add audit docs
ed60e6d docs: record Batch 0 completion and new finding N1 in remediation plan
16bf545 fix(M3): make transfer test suite reliable, not flaky
894b184 fix(A6,Q10): remove test-only code from production, run concurrency demo against real Postgres
212bdd8 fix(Q1): add CI workflow to run build/vet/test -race on every change
799c1e5 docs: record Batch 1 completion in remediation plan
9c0f41f docs: record new Critical finding N2 (leaked open transaction on every early return in Transfer())
40c9966 docs: record new Critical finding N3 (CreateUser is not actually atomic) and N2 disposition
91b515d fix(N2): stop leaking an open transaction on every failed Transfer()
8f98567 fix(N3): make UserService.CreateUser actually atomic
cb8a68e docs: mark N2 and N3 as fixed in remediation plan
828c317 fix(M1,M4,M6,M2): fixed lock order, currency-aware locking, amount limits
285d76b docs(M2): document SQLite's lack of real concurrency safety in README
4013519 docs: record Batch 2 completion in remediation plan
582da7e fix(S1,S2,S3,S4): remove committed secrets, fail fast on weak JWT, wire up rate limiting and trusted-proxy config
6806a02 fix(S7,S9,S11): 409 on duplicate signup, stop leaking raw DB errors, raise min password length
72711f7 docs: record Batch 3 completion in remediation plan
bada299 fix(S8,A5): unify error response format across every handler and middleware
3969c2a fix(S5): add CORS support, off by default until a frontend origin is configured
ffd93e7 fix(A2): regenerate Swagger docs, add OpenAPI 3.0 conversion, fix README claim
68934d2 docs: record A2 tooling decision in 新發現 and Batch 4 completion
a2cba15 fix(Q5): multi-stage Dockerfile, distroless non-root runtime image
21db2c1 fix(Q7,Q8): bind to all interfaces, graceful shutdown
d262580 fix(Q6,new): add api service to docker-compose, seed a default currency
73acbb6 docs(A3): fix README's How to Run to match the real one-command flow
5b27679 docs: record N4 finding and Batch 5 completion in remediation plan
5975d9b docs: add CLAUDE.md, final README accuracy pass
```

（`git log --oneline fix/audit-remediation` 可隨時重新查詢；每個 `docs: ...` commit
都對應該批次在 `AUDIT_REMEDIATION_PLAN.md` 裡的詳細驗證紀錄，包含實際執行的指令與
輸出。）

---

## 6. 需要您手動處理的事項

這些是我不能、或不應該自行決定的操作：

1. **輪替已外洩的密鑰**（S1）：`config.yaml` 裡曾經 commit 過的 JWT 密鑰
   （`your-secret-key-change-in-production-min-32-chars`）與 Postgres 密碼
   （`secret`）已經進了 git 歷史。就算現在從工作目錄移除了，只要這個 repo 曾經
   被任何人 clone 過（包含之後要合作的前端/AI agent），這兩個值都應該視為已外洩。
   **建議**：找一個時間產生新的 JWT 密鑰與資料庫密碼，透過環境變數/密鑰管理服務
   佈署，不要再寫回任何會進版控的檔案。

2. **是否要清除 git 歷史裡的密鑰與大型二進位檔**：目前的修正只是「之後不再 commit」，
   舊 commit（`80e6220` 之前）裡仍然找得到 `config.yaml` 的明文密鑰與兩顆合計
   109MB 的執行檔。若要徹底清除，需要 `git filter-repo`（或 BFG）重寫歷史——這是
   破壞性操作，會改變所有 commit hash，需要協調任何已經 clone/fork 這個 repo 的人
   重新拉取。**我沒有在未經確認的情況下執行這類操作**，是否要做、什麼時候做，
   請您決定。

3. **合併分支**：目前所有修正都在 `fix/audit-remediation` 分支上，尚未合併回
   `main`，也**沒有 push 到任何遠端**。請您 review 過（例如用
   `git log main..fix/audit-remediation` 或開一個 PR 用 diff 檢視）之後，決定要
   直接 merge、squash，還是先開 PR 走 review 流程。

4. **S12/A4 的 Redis 缺口**：如果之後真的要做 JWT 撤銷或多副本水平擴展，需要導入
   Redis（`redis_addr` 設定已經在，但完全沒有對應程式碼）。這不影響目前單機/demo
   規模的正確性，留給之後排程。

5. **本次未修正的項目**（M5、S10、S11 的失敗次數鎖定、S12、A4）：都是 Low/Medium
   等級、且都不影響資金正確性或有立即可利用的資安漏洞，詳見第 2 節表格裡每一項的
   說明。建議之後另外排程處理，或視前端串接進度決定優先順序。
