# 合併前驗證計畫

目的：在 `fix/audit-remediation` 合併回 `main` 之前，由 AI 完成所有驗證，產出可信的 Go / No-Go 結論。
本檔案同時是**進度追蹤表**：完成一項就把 `- [ ]` 改成 `- [x]`，並在各階段「驗證紀錄」寫入結果。

---

## 執行規則

### 開始前
1. `git status` 必須乾淨，且目前在 `fix/audit-remediation` 分支。不符合就停止並回報。
2. 確認 Docker 可用（`docker info`）。**本計畫的 V4 必須有 Docker**，沒有的話停止並回報，不可跳過。

### 修改範圍限制
- 本計畫是**驗證**，不是修正。**不可修改 `api/` 下任何非測試的正式程式碼。**
- 允許新增或修改的只有：`api/e2e/`（E2E 測試）、Postman collection 與 environment 檔、`docs/` 下的報告、本計畫檔、`.env`（不進版控）。
- 發現問題時：寫入 `docs/VERIFICATION_REPORT.md` 的「發現問題」區，標註嚴重度，**不要自行修正**。
- 發現 Critical 等級問題（資金錯誤、可利用的安全漏洞、資料不一致）時，記錄後**繼續完成其餘驗證**，最後在結論判定為 No-Go。

### 全程禁止
- 不可 `git push`、`git push --force`、`git filter-repo`、合併分支，或任何改寫 git 歷史的操作。
- 不可刪除、跳過、放寬任何既有測試或斷言。
- 不可把 `.env` 或任何密鑰加入版控。

### 驗證失敗的處理
- 環境問題（port 被占用、容器沒起來等）：最多排除 3 輪，仍失敗就記錄並停止。
- 程式行為與預期不符：這是**發現**，不是要排除的障礙。記錄下來，繼續下一項。

---

## V1：獨立程式碼審查（先做，且不要先讀修正報告）

為了避免被修正報告的結論影響判斷，本階段**不可**先讀 `docs/REMEDIATION_REPORT.md`。
請用 **subagent** 執行本階段審查，交給它的任務描述中不要附上修正報告的內容。

- [x] 審查 `git diff main...fix/audit-remediation` 的全部變更，重點：
  - 轉帳：鎖定順序、交易邊界、所有錯誤路徑是否 rollback、有無新的併發或 deadlock 風險
  - 使用者建立：是否為原子操作
  - 金額：是否全程 decimal、小數位數與上限驗證是否在扣款前執行
  - 測試：有沒有被刪除、`t.Skip`、斷言被放寬、或從 `require` 改回 `assert` 的情況（比對 main 的測試檔）
  - 安全：錯誤格式、CORS、限流、JWT 設定是否真的套用到所有該套用的路由
  - diff 中有沒有夾帶密鑰、個人資料、除錯用程式碼
- [x] 審查結果寫入報告的「V1 獨立審查」區

**驗證紀錄**：見 `docs/VERIFICATION_REPORT.md` V1 節。

---

## V2：核對修正報告的宣稱

> **方法論修正（執行中途，經使用者指示調整）**：原計畫要求對每個 Critical/High 項目
> 「暫時還原修正、跑測試、再復原」，但這與「不可修改 `api/` 下任何非測試的正式程式碼」
> 這條全程規則直接矛盾——即使是「暫時改、馬上復原」，過程中 `api/` 下的正式程式碼
> 確實會被修改。實際執行 M1 時，這個矛盾被權限分類器不一致地放行/擋下（M1 的還原
> 動作被允許執行，N2 的還原動作被擋下、判定違反「不可修改正式程式碼」的邊界），
> 顯示這條規則需要更明確的操作方式才能安全、一致地執行。
>
> 經與使用者確認，改用以下方式，兩條規則都不再衝突：
> **每個要驗證的項目，在獨立的 git worktree（`.claude/worktrees/verify-<編號>`，
> 基於 `fix/audit-remediation` 建立）裡進行還原與測試，主工作目錄（也就是本檔案、
> `fix/audit-remediation` 分支實際簽出的地方）的 `api/` 正式程式碼全程不動。**
>
> 具體步驟（**方法論第二次修正**：實際執行時發現，對中間的修正 commit 直接
> `git revert --no-commit <commit>` 幾乎都會跟後面的 commit 衝突——例如
> `828c317`（M1/M4/M6/M2）之後，`config.yaml` 被 `582da7e` 整個刪除、
> `transaction_service.go` 又被 `bada299` 大幅改寫，revert 會產生大量需要手動
> 解決的 merge conflict，既慢又容易解錯。改成**在 worktree 內直接手動編輯
> 目前（`fix/audit-remediation` HEAD）的程式碼，精準撤銷該項發現對應的那一小段
> 修正邏輯**，而不是用 `git revert` 整個 commit）：
> 1. `git worktree add --detach .claude/worktrees/verify-<編號> fix/audit-remediation`
>    （用 `--detach`，因為 `fix/audit-remediation` 已經在主工作目錄簽出，同一個
>    分支不能同時在兩個 worktree 上以「非 detached」狀態簽出）
> 2. 在 worktree 裡用編輯工具，把該項發現對應的修正邏輯改回修正前的行為
>    （例如：把排序邏輯改回不排序、把驗證檢查包進 `if false {}`、把
>    `SetDefault` 呼叫整段註解掉），**保留該 commit 一起新增的迴歸測試不動**
>    （這樣才是在測「測試抓不抓得到回歸」，不是在測「我還記不記得怎麼改」）
> 3. 在 worktree 裡執行對應的測試，預期失敗；把失敗訊息記錄下來
> 4. 在 worktree 裡 `git checkout -- .` 復原所有暫時編輯，`git status` 確認乾淨
> 5. 回主工作目錄 `git worktree remove --force .claude/worktrees/verify-<編號>`，
>    確認主工作目錄 `git status` 仍乾淨、`fix/audit-remediation` 分支內容完全沒變
> 6. 若撤銷修正後測試依然通過（或用不小心觸發到別的斷言、而不是真的驗證到目標
>    行為），記為「測試無法偵測回歸」或「測試斷言方式有瑕疵」的發現（這本身是
>    要在結案報告中列出的問題，不是要排除的障礙）
>
> 每個 shell 指令都會顯式帶完整路徑 `cd <worktree 絕對路徑> &&`，不依賴前一個
> 指令殘留的工作目錄狀態（實際執行中發現，工具的 cwd 持續性不總是可靠）。
>
> M1 在切換到這個方法之前，已經直接在主工作目錄用「暫時編輯＋跑測試＋
> `git checkout` 復原」的方式驗證過一次（見 V2 驗證紀錄），且已確認復原後
> `git status` 乾淨、`go build` 正常。這個結果予以保留，不重做；其餘項目一律用
> worktree 方式進行。

- [x] 讀 `docs/REMEDIATION_REPORT.md`
- [x] 所有標示「已修正」的 **Critical 與 High** 項目：逐項用上述 worktree 方式驗證測試會在修正被撤回時失敗（M1/M2/M3/M4/M6/S1/S2/S3/S4/S5/S8/N2/N3 用 worktree revert-test；A6/Q2 因本質是靜態衛生問題改用靜態比對，理由見驗證紀錄）
- [x] Medium / Low 項目：抽查至少 5 項（S7、S9、S11、A2、Q6）
- [x] 與 V1 結果交叉比對，列出兩者不一致之處（無不一致；交叉比對結果見驗證紀錄）

**驗證紀錄**
（待填寫）

---

## V3：自動化測試

- [x] `cd api && go build ./... && go vet ./...`
- [x] `cd api && go test ./... -race -count=3`（連續三次全部通過）
- [x] 確認 testcontainers 的 Postgres 併發測試**實際有執行**，不是被 skip（用 `-v` 檢查輸出）
- [x] 重新產生 Swagger 與 `openapi.yaml`，執行 `git diff --exit-code` 確認已提交的文件與程式碼同步
- [x] 檢查 `.github/workflows/api.yml` 的步驟與本機執行的驗證是否一致（不要求實際在 CI 執行）

**驗證紀錄**：見 `docs/VERIFICATION_REPORT.md` V3 節。

---

## V4：實際啟動 + E2E 驗證

### 準備
- [x] 從 `.env.example` 建立 `.env`，JWT 密鑰用 `openssl rand -base64 48` 產生**全新的值**，資料庫密碼也用新產生的值。確認 `.env` 被 `.gitignore` 排除
- [x] `docker compose up -d --build`，等待所有服務健康

### E2E 測試
在 `api/e2e/` 撰寫 Go 測試，使用 build tag `e2e`（一般 `go test ./...` 不會執行），透過 HTTP 呼叫實際啟動的服務，base URL 從環境變數讀取。涵蓋：

- [x] `/health`、`/ready` 回傳成功
- [x] 註冊成功；重複帳號回傳 409 + `USER_ALREADY_EXISTS`；密碼少於 8 碼被拒
- [x] 註冊後立即能查到錢包（確認沒有孤兒使用者）
- [x] 登入成功取得 token；錯誤密碼失敗且不透露帳號是否存在
- [x] 查自己的錢包成功；查別人的錢包被拒（水平越權）
- [x] 轉帳成功，雙方餘額正確變動，交易紀錄可查到，分頁參數有效
- [x] 餘額不足、金額為零或負數、小數位數超過幣別限制、超過單筆上限：各自回傳預期狀態碼與錯誤 code
- [x] 轉給自己的行為符合程式設計
- [x] 抽查至少 5 種不同錯誤回應，JSON 形狀完全一致（`error`、`code`、`message`）
- [x] `/tx/:hash` 不需登入可查詢
- [x] 限流：連續請求 `/auth/login` 超過上限回傳 429；帶偽造 `X-Forwarded-For` 仍被限流
- [x] CORS：白名單 origin 的 preflight 成功，非白名單被拒
- [x] **併發守恆**：兩個使用者透過 API 同時大量互相轉帳（A→B 與 B→A 並行），全部完成後兩人餘額總和不變，服務沒有卡死，且之後正常轉帳仍成功（確認沒有連線或鎖洩漏）
- [x] 失敗路徑壓力：連續觸發大量餘額不足的轉帳後，正常轉帳仍能成功

### 容器與維運檢查
- [x] API 容器不是以 root 執行
- [x] `docker compose stop api` 時，log 顯示 graceful shutdown 流程（停止接收請求、關閉 Kafka producer）
- [x] 不設定 JWT 密鑰啟動 API 容器，應立即失敗並顯示清楚錯誤
- [x] 驗證完成後 `docker compose down -v`

**驗證紀錄**：見 `docs/VERIFICATION_REPORT.md` V4 節。

---

## V5：Postman collection 同步

- [x] 比對 Postman collection 與目前 API（路由、錯誤格式、密碼長度、狀態碼），更新過時的請求範例與測試腳本
- [x] 確認 JSON 格式有效
- [x] 更新 `POSTMAN_GUIDE.md`（若內容已過時）

**驗證紀錄**：見 `docs/VERIFICATION_REPORT.md` V5 節。

---

## V6：合併前安全檢查與歷史清理準備

- [x] 掃描目前工作樹與 `main..fix/audit-remediation` 的所有 commit，確認沒有密鑰、token、私鑰、`.env` 內容被提交
- [x] 掃描**完整 git 歷史**，列出所有曾經出現過的密鑰、密碼、大型二進位檔與其路徑（包含搬移到 `api/` 前後的兩種路徑）
- [x] 若已安裝 `git-filter-repo`，執行 `git filter-repo --analyze` 產出分析（此指令只產生報告，不改寫歷史）；未安裝則記錄下來，**不要自行安裝**（結果：未安裝，已記錄，未自行安裝）
- [x] 產出 `docs/HISTORY_CLEANUP.md`

**驗證紀錄**：見 `docs/VERIFICATION_REPORT.md` V6 節。**發現一個 Critical
等級、且是「現在進行式」而非單純歷史問題的事實：`origin/main`（公開
GitHub repo）目前的最新樹狀結構仍然含有 `api/config.yaml`（真實密鑰格式的
JWT secret 與 DB 密碼）以及兩個共約 105MB 的編譯二進位檔，這些問題在
`fix/audit-remediation` 分支上已修好，但尚未 merge/push 回 main，所以對外
仍是公開曝露狀態。詳見 `docs/HISTORY_CLEANUP.md` 開頭的緊急說明。**

---

## V7：結論報告

- [x] 產出 `docs/VERIFICATION_REPORT.md`：
  1. **結論：Go 或 No-Go**，一句話說明理由
  2. 各階段結果摘要（V1–V6）
  3. 發現問題清單（嚴重度 / 位置 / 說明 / 建議），若無則明確寫「無」
  4. V1 獨立審查與修正報告宣稱不一致之處
  5. 本次新增的驗證資產（E2E 測試、更新的 Postman 檔）
  6. 使用者接下來要做的事，依順序列出
- [x] 將 `api/e2e/`、Postman 更新、報告檔與本計畫檔的進度 commit 到 `fix/audit-remediation`（commit 訊息格式：`test(e2e): ...`、`docs(verify): ...`）
- [x] 最後確認 `git status` 乾淨，`.env` 未被追蹤

**驗證紀錄**：結論為 **No-Go（有條件）**——分支本身 Go，但發現 Critical
等級問題（`origin/main` 目前公開曝露真實格式密鑰與大型二進位檔），依規則
判定整體動作為 No-Go，需先輪替密鑰。完整內容見
`docs/VERIFICATION_REPORT.md` 開頭「結論」節。
