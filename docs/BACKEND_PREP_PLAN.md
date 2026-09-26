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
- [ ] 成功時回傳建立的交易，複用 `models.TransactionResponse`，狀態碼維持 `200`
- [ ] 回應需包含 `hash`，讓前端可直接導向 Explorer
- [ ] 更新 Swagger 註解與 Postman collection 的對應斷言
- [ ] 測試：轉帳成功後回應中的 `hash` 與資料庫中該筆交易一致

### 1.2 拆細 `INVALID_AMOUNT`
- [ ] 在 `internal/errors/codes.go` 新增並改用：
  - `AMOUNT_NOT_POSITIVE`：金額為零或負數
  - `INVALID_DECIMALS`：小數位數超過該幣別的 `decimals`
  - `AMOUNT_EXCEEDS_LIMIT`：超過單筆上限
- [ ] 移除 `INVALID_AMOUNT`；若判斷保留較安全，保留常數但不再使用，並在程式碼註解說明
- [ ] 測試：三種情況各一條，斷言各自回傳正確的 code

### 1.3 拆細 `WALLET_NOT_FOUND`
- [ ] 新增並改用：
  - `SENDER_WALLET_NOT_FOUND`：轉出方在該幣別沒有錢包
  - `RECIPIENT_NOT_FOUND`：收款人不存在，或在該幣別沒有錢包
- [ ] `GET /wallet/{user_id}` 查無錢包時維持 `WALLET_NOT_FOUND`
- [ ] 測試：兩種情況各一條

### 1.4 `TransactionResponse` 補上 `currency_id`
- [ ] DTO 新增 `currency_id` 欄位，影響 `GET /tx/{hash}`、`GET /transactions/{user_id}`、以及 1.1 的轉帳回應
- [ ] 測試：三個端點的回應都含正確的 `currency_id`

**驗收條件**
- `cd api && go build ./... && go vet ./... && go test ./... -race` 全部通過
- 重新產生 Swagger 與 `openapi.yaml` 後 `git diff --exit-code docs/` 為乾淨
- `grep -rn 'INVALID_AMOUNT' api --include='*.go' | grep -v codes.go` 無輸出（或僅剩註解）
- `openapi.yaml` 中 `TransactionResponse` 含 `currency_id`

**驗證紀錄**
（待填寫）

---

## 第 2 批：前端需要的查詢端點

### 2.1 `GET /wallet/{user_id}/stats`
- [ ] 需 auth，並套用既有的 `RequireUserID` 水平越權檢查
- [ ] 查詢參數 `window`，第一版只支援 `24h`，其他值回 `INVALID_REQUEST`
- [ ] 回應：`{ window, transaction_count, total_sent, total_received, balance_change }`，金額欄位為字串格式的 decimal
- [ ] 統計由資料庫查詢直接計算，**不可**在應用層撈全部交易再加總
- [ ] 測試：無交易時全為零；有收有支時數值正確；超過 24 小時的交易不納入；別人的 user_id 回 403

### 2.2 `GET /currencies` 回應補上單筆上限
- [ ] `CurrencyResponse` 新增 `max_transfer_amount`，值取自後端設定
- [ ] `GET /currencies/{id}` 同步
- [ ] 測試：回應含該欄位且與設定值一致

### 2.3 `GET /users/lookup`
- [ ] 查詢參數 `username`，回傳 `{ id, username }`
- [ ] **只回傳 id 與 username，不可回傳 email 或任何其他欄位**（避免帳號列舉取得個資）
- [ ] 查無使用者回 `404` + `USER_NOT_FOUND`（此 code 已定義但未使用，正好啟用）
- [ ] 需 auth，並套用限流（比照一般端點）
- [ ] 測試：查得到、查不到、未帶 token 回 401、回應不含 email

**驗收條件**
- `cd api && go build ./... && go vet ./... && go test ./... -race` 全部通過
- 重新產生文件後 `git diff --exit-code docs/` 為乾淨
- 新端點的 E2E 測試：在 `api/e2e/` 新增對應測試，涵蓋 stats 正確性與 lookup 不洩漏 email
- `docker compose up -d --build` 後，用真實 HTTP 呼叫三個新端點皆回應正確

**驗證紀錄**
（待填寫）

---

## 第 3 批：更新文件與規格

- [ ] 更新 `docs/FRONTEND_SPEC.md`：
  - §4 錯誤對應表加入新的 code，移除已拆分的舊 code
  - §5 移除已實作的項目（5.6 統計、5.9 轉帳回應、5.10 上限、5.11 currency_id），5.2 改為「已支援，改用 `GET /users/lookup`」
  - §3.3 Transfer：移除查最新一筆交易的 workaround，改為直接使用轉帳回應中的 `hash`；限制說明列改為顯示實際上限數值
  - §3.2 Overview：24h 指標卡改為使用 stats API，移除「最新 100 筆」的免責說明
  - §3.1 Login：移除密碼強度條，僅保留「At least 8 characters」的規則勾選
  - §2.2 Token 儲存：改為 `localStorage`，並保留過期檢查與 401 全域處理
  - §3.4 Export：加上筆數上限保護，超過 1000 筆時提示使用者縮小範圍
  - 第 7 節改寫為「已確認的決定」，記錄每項的最終結論
- [ ] 更新 `CLAUDE.md`：加入前後端邊界規則（`web/` 只透過 HTTP 呼叫、型別自動生成禁止手寫、改 API 需同步更新 Swagger 並重新生成型別）
- [ ] 更新 `README.md`：新增端點說明，並補充本機同時啟動前後端時需設定 `CORS_ALLOWED_ORIGINS`
- [ ] 更新 `docs/BACKLOG.md`：把本次未做的項目（多幣別錢包列表、交易篩選參數、匯率估值、export 端點）補進去

**驗收條件**
- `cd api && go build ./... && go vet ./... && go test ./... -race` 全部通過
- `docs/FRONTEND_SPEC.md` 中搜尋「workaround」「最新 100 筆」「強度條」皆無殘留
- `git status` 乾淨，所有變更已 commit

**驗證紀錄**
（待填寫）

---

## 第 4 批：收尾

- [ ] push `feat/frontend-prep` 到 origin
- [ ] 產出 PR 標題與描述文字，直接輸出在回覆中並用 markdown code block 包起來，供使用者在 GitHub 網頁手動開 PR（`gh` 目前無法安裝）
- [ ] 使用者建立 PR 後，依其提供的網址查詢 CI 狀態並回報；失敗時分析原因，不自行修正後推送

**驗收條件**
- 分支已推送，PR 描述文字已輸出

**驗證紀錄**
（待填寫）

---

## 新發現
（執行過程中發現、但本計畫未涵蓋的問題，記在這裡）
