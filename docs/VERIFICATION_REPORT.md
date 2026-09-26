# 合併前驗證報告

依據：`docs/PRE_MERGE_VERIFICATION_PLAN.md`（若該檔案不存在，指使用者於對話中提供的
「合併前驗證計畫」）
分支：`fix/audit-remediation`（比較基準：`main`）
本檔案在驗證進行過程中逐步填寫，各節「驗證紀錄」與計畫檔的勾選狀態同步更新。

---

## 結論

### 1. Go / No-Go

**分支本身：Go**（`fix/audit-remediation` 的程式碼品質、修正正確性、測試覆蓋
都通過驗證，可以合併）。
**整體動作：No-Go（有條件）**——依計畫規則，發現 Critical 等級問題時結論須
判定為 No-Go；本次驗證發現的 Critical 問題**不是**這個分支的程式碼缺陷，而是
V6 發現的「公開 GitHub repo 的 `origin/main` 現在的 HEAD 就含有真實格式的
JWT 密鑰、資料庫密碼與 105MB 編譯二進位檔」——這是可被任何人利用的現行外洩，
必須先處理完才能視為安全地完成「合併並推送」這整個動作。

一句話：**這個分支的修正是對的、測試是可信的，但在按下 push／merge 之前，
必須先輪替 `origin/main` 上已經公開曝露的密鑰**（詳見下方發現清單第 1 項與
`docs/HISTORY_CLEANUP.md`）；密鑰輪替與（可選的）歷史清理完成後，此分支即可
安全合併。

### 2. 各階段結果摘要（V1–V6）

| 階段 | 結果 | 摘要 |
|---|---|---|
| V1 獨立程式碼審查 | ✅ 通過 | 全新 subagent（未看過修正報告）獨立審查 diff，確認 `main` 上的鎖定順序、rollback 洩漏、使用者建立非原子、限流未掛載、JWT 硬編碼 fallback 等問題確實存在且已被正確修正；額外標記 3 個既有、非本次引入的 Low 問題。 |
| V2 核對修正報告宣稱 | ✅ 通過 | 對所有 Critical/High 項目（M1-M4、M6、S1-S3、S5、S8、N2、N3 等）在隔離 git worktree 中做「撤銷修正→跑測試預期失敗→還原」的真實驗證，全數證實測試能偵測到對應回歸；額外發現 4 點修正報告未提到的細節（S4/A6 缺乏自動化回歸防護、9 處潛在 panic 風險的測試寫法、M3 迴歸只在整包測試才穩定重現）。 |
| V3 自動化測試 | ✅ 通過 | `go build`/`go vet`/`go test ./... -race` 連續 3 次全數通過；確認 testcontainers 測試真的有執行（非被跳過）；Swagger→OpenAPI 轉換無 drift；CI workflow 步驟與本機指令一致。 |
| V4 真實啟動 + E2E | ✅ 通過 | 全新 `.env`、`docker compose up --build` 完整啟動；新增 7 個檔案、20 個 top-level 測試的 E2E 套件（build tag 隔離），涵蓋健康檢查、認證、橫向越權、轉帳成功/失敗、併發、限流、CORS、graceful shutdown、JWT fail-fast、非 root 執行，全數通過。 |
| V5 Postman 同步 | ✅ 通過 | 路由 100% 比對一致；修正 3 處明確過時的測試斷言（Register 回應欄位、Health/Readiness 回應內容）並用真實服務重放驗證；強化 3 處負向測試以檢查 S8 新增的穩定 `code` 欄位；更新 `POSTMAN_GUIDE.md` 過時的啟動說明。 |
| V6 安全掃描與歷史清理準備 | 🔴 發現 Critical | 本分支本身未新增任何密鑰；但掃描發現 `origin/main`（公開 repo）**目前的 HEAD** 仍含真實格式密鑰與大型二進位檔（本分支已修正但尚未 push）。已產出 `docs/HISTORY_CLEANUP.md`（只分析、未執行任何改寫歷史或推送的指令）。 |

### 3. 發現問題清單

| 編號 | 嚴重度 | 位置 | 說明 | 建議 |
|---|---|---|---|---|
| F1 | **Critical** | `origin/main`（公開 GitHub repo）目前 HEAD 的 `api/config.yaml` | 真實格式的 `postgres_dsn` 密碼（`password=secret`）與 `jwt_secret`（`your-secret-key-change-in-production-min-32-chars`）目前仍公開可見；`wallet-api`／`main/__debug_bin2424212485` 共約 105MB 編譯二進位檔同樣仍在 HEAD 上。 | 立刻輪替任何跟這兩個值相關的真實憑證；儘快把 `fix/audit-remediation` 合併並 push 到 `main`（可讓「未來」的 HEAD 不再含這些檔案）；若要徹底清除「歷史」中的痕跡，依 `docs/HISTORY_CLEANUP.md` 的步驟執行 `git filter-repo` 並強制推送（需使用者自行決定時機，本次驗證未執行）。 |
| F2 | Medium | `config.yaml` / `api/config.yaml` 的 git 歷史（commit `eb7472e`、`bfc4da1`、`47b6974`、`828c317`） | 同 F1 的密鑰內容在歷史上共出現於 4 個 commit，`582da7e` 之後的 HEAD 已不含此檔，但歷史紀錄本身仍查得到。 | 同 F1，需要 `git filter-repo` 才能真正從歷史移除。 |
| F3 | Low | `middleware/validator.go`（`HandleValidationError`） | 驗證錯誤分支手刻 `gin.H{...}` 而非直接複用 `models.ErrorResponse` struct；欄位內容正確，純風格瑕疵。（V1 發現，既有、非本次引入） | 可選：改用 `models.NewErrorResponse` 建構，與其餘 handler 一致。 |
| F4 | Low | `middleware/ratelimit.go`（`RateLimiter.limiters`） | 每個來源 IP 的 `rate.Limiter` 永不清除，長期會隨不同 IP 數量無上限成長。（V1 發現，既有、非本次引入） | 可選：加上 LRU 或定期清除機制。 |
| F5 | Low | `models/transaction.go`（`GenerateHash`） | 使用 `time.Now().UnixNano()` 而非密碼學亂數產生 hash 的一部分。（V1 發現，既有、非本次引入，與本次修正無關） | 可選：若未來要當作真正防偽依據，改用 CSPRNG 或 HMAC。 |
| F6 | Low | `middleware/ratelimit.go` + `router.go`（S4） | S4（`SetTrustedProxies`）目前只有中介層單元測試，沒有透過 `SetupRouter()` 組出真正 router 再驗證的整合測試；撤銷 `SetTrustedProxies` 呼叫後其餘測試仍全數通過，代表沒有測試會抓到這個特定回歸。（V2 發現） | 可選：加一個透過 `SetupRouter()` 組出真正 router、偽造 `X-Forwarded-For` 驗證 `ClientIP()` 行為的整合測試。 |
| F7 | Low | `services/transaction_service.go`（A6 對應的既有問題） | 移除 `import "testing"`／`TransferWithLockOption` 的修正沒有對應 CI 檢查，若日後又混入類似程式碼，`go build`/`vet`/`test` 都不會失敗。（V2 發現） | 可選：在 CI 加一個 grep 檢查非 `_test.go` 檔案是否 import `testing`。 |
| F8 | Low | `services/transaction_service_test.go:192-193,225-226,254-255,279-280,307-308,340-341,363-364,386-387,412-413` | 9 處 `assert.Error(t, err)` 後立即對 `err.Error()` 解參考，與 M3 原本修的問題同一種模式（`err` 意外為 `nil` 時整個測試 binary 會 panic）；本次驗證 M4/M6 時因程式碼被撤銷而親身重現。（V2 發現） | 建議：9 處全改為 `require.Error(t, err)`。 |
| F9 | 資訊性（非缺陷） | `services` package 測試 | M3 的迴歸測試必須整個 package 一起跑才會穩定重現，單獨用 `-run` 篩選單一測試名稱不會重現。不影響 CI（CI 本來就整包跑），但可能誤導只跑單一測試的人。（V2 發現） | 無強制動作，僅供之後除錯時留意。 |
| F10 | Low | `kafka_client.KafkaProducer.Close()` | 成功關閉時不輸出任何 log，只有失敗才 log，所以從 log 只能間接確認 Kafka producer 已關閉。（V4 發現） | 可選：成功關閉時補一行 log，純觀測性改善。 |

**F3-F10 均為 Low 或資訊性，且皆為既有問題或測試強化建議，不影響本次合併的
Go/No-Go 判斷。唯一影響判斷的是 F1（連帶 F2）。**

### 4. V1 獨立審查 與 修正報告 的不一致之處

**沒有發現不一致**。V1（全新、未看過修正報告的 subagent）與 V2（逐項核對）
的結論高度一致，修正報告沒有誇大或遺漏任何已完成項目。V1 額外標記的 3 個
既有 Low 問題（上表 F3-F5）修正報告確實沒提到，但這些問題**在稽核範圍
（`docs/AUDIT.md`）之外**，不算修正報告的疏漏，只是驗證過程中的額外收穫。

### 5. 本次新增的驗證資產

- **E2E 測試套件**（`api/e2e/`，7 個檔案、20 個 top-level 測試函式，
  `//go:build e2e` 隔離，一般 build/test 不受影響）：涵蓋健康檢查、註冊/登入、
  橫向越權存取控制、轉帳成功與 5 種驗證失敗情境、併發轉帳（餘額守恆+無死結）、
  限流、CORS、公開端點存取控制。
- **Postman collection 修正**：3 處過時斷言更新（Register 回應欄位、
  Health/Readiness 回應內容），3 處負向測試強化（加入 S8 的 `code` 欄位檢查）。
- **`POSTMAN_GUIDE.md` 更新**：啟動說明改為同時涵蓋 docker-compose 與本機
  執行兩種路徑，補充 JWT 密鑰長度不足會 fail-fast 的說明。
- **`docs/VERIFICATION_PLAN.md`**：完整記錄本次驗證的方法論（含 V2 從
  「git revert」改為「worktree 內直接編輯」的方法演進與原因）。
- **`docs/HISTORY_CLEANUP.md`**：完整的歷史清理分析與可執行步驟（未執行）。

### 6. 使用者接下來要做的事（依順序）

1. **立刻輪替**跟 `origin/main` 上曝露的 `password=secret`（Postgres）與
   `jwt_secret: your-secret-key-change-in-production-min-32-chars` 相關的
   任何真實憑證——不論是否採信這兩個值「看起來只是佔位符」，只要曾經或正在
   任何真實環境使用，都應視為已外洩。
2. 決定是否要執行 `docs/HISTORY_CLEANUP.md` 的 `git filter-repo` 歷史清理
   （會改寫所有 commit hash、需要強制推送，請先評估是否有其他協作者/clone）。
   若暫不執行歷史清理，至少應盡快完成第 3 步，讓「未來」的 HEAD 不再曝露。
3. 合併 `fix/audit-remediation` 到 `main` 並 push（分支本身的品質已通過本次
   全部驗證）。
4. 檢視 F3-F10 的 Low/資訊性建議，依優先順序自行排入後續維護。
5. 若計畫進一步接前端，参考 `CLAUDE.md` 的分層/金流規則與 `docs/AUDIT.md`
   第 3 節「接前端前必須先修的項目」（本輪已全數完成）。

---

## V1：獨立程式碼審查

執行方式：透過全新（非 fork）subagent 進行，**任務描述中未附上** `docs/REMEDIATION_REPORT.md`
或 `docs/AUDIT.md`/`docs/AUDIT_REMEDIATION_PLAN.md` 的內容，並明確指示不要主動讀取這些
檔案，避免結論被既有修正報告影響。Subagent 只被告知：這是一個 Go 寫的加密貨幣錢包
後端 API，要對 `git diff main...fix/audit-remediation` 做獨立、懷疑論式的審查，並給了
六個具體檢查面向（金流正確性、使用者建立原子性、金額驗證、測試完整性比對 main、
安全性、diff 衛生）。

### 獨立審查結論（逐字保留 subagent 回報）

**總結**：`main` 與 `fix/audit-remediation` 有共同歷史，但分支把整個 module 搬進了
`api/` 子目錄，單純 `git diff` 會把每個檔案顯示成「新增」；用 `-M --find-renames=40%`
才能看到真正的內容差異。整體而言，這個分支修好了 `main` 上幾個真實、嚴重的缺陷：
寫死的 JWT 密鑰 fallback、沒掛載的限流、非決定性的錢包鎖定順序（死結風險）、交易
rollback 缺口（一般錯誤路徑會洩漏 idle-in-transaction 連線）、使用者建立路徑在
包住錢包建立的交易之外寫入 User（真實的孤兒帳號風險）。這些都已經對照 `main`
「實際的舊程式碼」（不是只看註解描述）驗證過，並且完整測試套件（含兩個用
Docker/testcontainers 跑真正 Postgres 的併發與死結測試）在 `-race` 下全數通過。
測試檔案的 diff 只有「加強」——`assert` 改 `require`（修掉原本可能的 nil
dereference 風險）、新增迴歸測試、修掉測試基礎設施裡一個重要的 SQLite `:memory:`
跨連線隔離 bug——沒有刪除測試、沒有新增「非環境判斷」的 skip、沒有斷言被放寬。
殘留、較低嚴重度的問題：`main` 的 git 歷史裡仍有一組真實（雖然明顯是示範用）的
資料庫密碼與 JWT 密鑰字串，這個分支無法回溯清除；`RateLimiter.limiters` map
沒有上限成長機制（`main` 上就存在，非本次引入）；還有一兩個較小的錯誤格式/
一致性小瑕疵。

### 發現清單（Subagent 原始回報）

| 嚴重度 | 領域 | 位置 | 說明 |
|--------|------|------|------|
| Critical（既有、已修正） | 安全 | `main:router/router.go`（舊版，分支已移除） | `main` 在 `JWT_SECRET` 未設定時會 fallback 成寫死的預設密鑰；分支已改成驗證強度不足就拒絕啟動 |
| Critical（既有、已修正） | 金流 | `main:services/transaction_service.go`（舊版 `Transfer`） | 鎖定順序依呼叫參數（`fromID`、`toID`）而非固定順序，可能死結；已改成依 `user_id` 排序，並用真的 testcontainers Postgres 驗證過 |
| High（既有、已修正） | 金流 | `main:services/transaction_service.go`（舊版 `Transfer`） | `defer utils.RollbackIfPanic(tx)` 只在 panic 時 rollback，一般 `return err`（餘額不足、錢包不存在、DB 更新失敗）都會洩漏 idle-in-transaction 連線；已改成 `committed` flag + 無條件 deferred rollback，並有 `TestTransfer_Fail_DoesNotLeakConnection` 驗證連線確實歸還池 |
| High（既有、已修正） | 使用者建立原子性 | `main:services/user_service.go` 約第 85 行 | `CreateUser` 呼叫時未帶交易控制代碼，User 那筆是自動提交，wallet 建立失敗時救不回來，形成孤兒帳號；已修正為 `tx` 貫穿整個流程，並有對應迴歸測試 |
| High（既有、已修正） | 安全 | `main:router/router.go` | 限流中介層在 `main` 上完全沒有掛載到任何路由；分支已掛上 `generalLimiter`/`loginLimiter`，並呼叫 `SetTrustedProxies`（預設不信任任何代理）避免偽造 `X-Forwarded-For` 繞過 |
| Medium（既有、已修正） | 金流 | `main:repositories/wallet_repository.go` 的 `GetWalletByUserIDWithTx` | 鎖定只用 `user_id`，事後才在應用層比對 `CurrencyID`，多幣別情境下不保證鎖到正確的那一列；已改成查詢本身就同時帶 `user_id`+`currency_id` |
| Medium（殘留） | Diff 衛生／密鑰 | `main:config.yaml`（git 歷史） | 真實的資料庫密碼與 JWT 密鑰範例字串曾經直接 commit；分支已修正「之後不再這樣做」的模式，但**無法回溯清除歷史**，這件事需要另外處理 |
| Low | 安全介面 | `middleware/validator.go`（`HandleValidationError`） | 驗證錯誤分支仍是手刻 `gin.H{...}`，而不是直接複用 `models.ErrorResponse` struct 再加欄位；欄位名稱實際上是對的，純屬程式碼風格瑕疵 |
| Low | Diff 衛生 | `middleware/ratelimit.go`（`RateLimiter.limiters`） | 每個 IP 的 `rate.Limiter` 記錄永遠不會被清除，長期下來 map 會隨不同來源 IP 數量無上限成長。**這不是本次分支引入的**，`main` 上邏輯完全相同，只是修正完之後依然存在，值得記錄 |
| Low | Diff 衛生 | `models/transaction.go` 的 `GenerateHash`（未變動） | 用 `time.Now().UnixNano()`（非密碼學亂數）當雜湊輸入的一部分；以現在的流量來說實務上唯一，但不是密碼學不可預測。與本次修正無關，僅供參考 |

### 六個檢查面向的具體回答（摘要）

1. **金流正確性**：確認鎖定順序、rollback 涵蓋所有提前 return、加鎖查詢同時帶
   currency_id，皆與宣稱相符，且用真的 Postgres 測試驗證過。
2. **使用者建立原子性**：確認 `main` 上的 bug 真實存在（`CreateUser` 沒有帶
   `tx`），分支的修法（`IUser.CreateUser` 加上 `tx ...*gorm.DB`）正確解決問題，
   有針對性的迴歸測試。
3. **金額驗證**：全程 `decimal.Decimal`，找不到 `float64` 用在金流上；小數位數與
   上限檢查都在 `tx.Begin()` 之前執行，符合「扣款前驗證」的要求。
4. **測試完整性比對 main**：**沒有**發現刪除測試、非環境判斷的新 skip、或斷言被
   放寬的情況；相反地，多處 `assert` 改成 `require`、新增多個迴歸測試、修掉測試
   基礎設施本身的一個 bug（SQLite `:memory:` 跨連線隔離）。
5. **安全性**：錯誤格式統一、CORS 沒有 `*` 與 credentials 並用的情況、限流真的掛
   到路由上且信任代理預設關閉、JWT 找不到任何殘留的硬編碼 fallback。
6. **Diff 衛生**：沒有找到真實密鑰、個人資料或除錯用程式碼；`main` 歷史裡的舊密鑰
   仍是需要另外處理的殘留風險。

**結論**：V1 獨立審查與後續 V2 要核對的修正報告內容**高度一致**，沒有發現修正
報告誇大或遺漏的情況；額外標記了三個修正報告未提及、但嚴重度為 Low 且非本次
引入的既有小問題（見上表最後三列），會在 V7 結論報告的「發現問題」區一併列出。

**驗證紀錄**：Subagent 執行了 67 次工具呼叫、耗時約 7.5 分鐘，實際跑過
`go test ./... -race`（含 testcontainers Postgres 測試）並確認通過，逐一對照
`main` 分支的實際舊程式碼（非僅讀取註解或 commit message）。

---

## V2：核對修正報告的宣稱

方法論見 `docs/VERIFICATION_PLAN.md` V2 節（worktree 隔離 + 直接手動編輯，理由與
細節都寫在那裡，這裡只記結果）。以下逐項記錄「撤銷修正 → 跑測試 → 復原」的結果。

### M1（Critical）— 已於方法論調整前，在主工作目錄直接驗證（結果保留）
把 `services/transaction_service.go` 的 `lowID, highID := fromID, toID` 排序邏輯
暫時改回不排序，跑 `TestTransfer_NoDeadlock_BidirectionalConcurrentTransfers`
（testcontainers 真 Postgres）：**測試失敗**，重現真實的
`ERROR: deadlock detected (SQLSTATE 40P01)`。`git checkout -- .` 復原，`go build`
正常，`git status` 乾淨。**結論：測試確實會在修正被撤回時失敗。**

### M2（Critical）— worktree `verify-M1-M4-M6-M2`
撤銷：`internal/test/test_helpers.go` 裡設定 `PRAGMA busy_timeout` 的那段程式碼。
測試：`TestTransfer_ConcurrentTransfers_NoRaceCondition`（SQLite）。
連續跑 3 次：**2 次 FAIL**（`database is locked (5) (SQLITE_BUSY)`）、1 次
PASS——這正是這個 bug 原本的行為特徵（併發寫入時機沒對齊才會炸，本來就是機率性的，
不是每次都會炸）。**結論：測試會在修正被撤回時（機率性地）失敗，符合這個 bug
本身機率性的特質。**

### M4（High）— worktree `verify-M1-M4-M6-M2`
撤銷：`GetWalletByUserIDAndCurrencyWithTx` 的 SQL 查詢條件從
`user_id = ? AND currency_id = ?` 改回只用 `user_id = ?`（模擬修正前「先撈任一筆
再事後比對幣別」的行為，但這次連事後比對都拿掉，因為修正後的程式碼已經不再需要
事後比對）。測試：`TestTransfer_Fail_InvalidCurrency`（Alice 持有 USDT 錢包、Bob
持有 BTC 錢包，嘗試用 USDT 的 currency_id 轉帳）。**測試失敗**——而且失敗方式比
預期更嚴重：`err` 變成 `nil`（原本應該要因為幣別不符而報錯，撤銷修正後這筆
「跨幣別」轉帳直接靜默成功了），導致測試斷言 `err.Error()` 時對 nil 解參考、
整個測試 binary panic。**結論：測試確實會在修正被撤回時失敗，而且暴露的問題比
原始稽核描述的更嚴重（不只是「誤報找不到錢包」，是「跨幣別轉帳可能靜默成功」）。**
**副帶發現（見下方「V1/V2 交叉比對」）**：這個測試用 `assert.Error(t, err)` 緊接著
`assert.Contains(t, err.Error(), ...)`，`err` 意外為 `nil` 時會讓整個測試 binary
panic，而不是乾淨地讓這一個測試失敗——這跟 M3 原本要修的問題是同一個模式，但
`transaction_service_test.go` 裡其實還有 8 處一樣的寫法沒有跟著改成 `require`
（見下方 V2 額外發現）。

### M6（High）— worktree `verify-M1-M4-M6-M2`
撤銷：金額驗證裡的「超過上限」與「小數位數過多」兩個檢查（各自包進
`if false { ... }`）。測試：`TestTransfer_Fail_ExceedsMaxAmount`、
`TestTransfer_Fail_TooManyDecimalPlaces`。**兩個測試都失敗**（一樣是
`assert.Error` 對 nil `err` 呼叫 `.Error()` 導致 panic，見上方同一個副帶發現）。
**結論：測試確實會在修正被撤回時失敗。**

### S1（Critical）— worktree `verify-S1-S2-S3-S4`
撤銷：`internal/config/loader.go` 裡所有 `viper.SetDefault(...)` 呼叫。測試：
`TestLoadConfig_FromEnvOnly_NoConfigFile`、`TestLoadConfig_DefaultsWhenNothingSet`。
**兩個測試都失敗**（`JWTSecret`/`DBDriver`/`AppEnv` 全部讀回空字串，而不是預期的
環境變數值或預設值）。**結論：測試確實會在修正被撤回時失敗。**

### S2（Critical）— worktree `verify-S1-S2-S3-S4`
撤銷：`router/router.go` 的 JWT 密鑰驗證，改回「空字串就 fallback 成硬編碼密鑰」。
測試：`TestSetupRouter_FailsFastOnWeakJWTSecret`（子行程重跑測試二進位檔，預期
非 0 結束碼）。**測試失敗**（子行程以 exit code 0 正常結束，因為現在不會再因為
密鑰太弱而拒絕啟動）。**結論：測試確實會在修正被撤回時失敗。**

### S3（High）— worktree `verify-S1-S2-S3-S4`
撤銷：`router.go` 裡 `generalLimiter`/`loginLimiter` 從所有路由拿掉。測試：
`TestSetupRouter_LoginIsRateLimited`。**測試失敗**（連續打 20 次
`/auth/login` 只拿到 400，從未看到 429）。**結論：測試確實會在修正被撤回時
失敗。**

### S4（Medium，非 High，但與 S1-S3 同一個 worktree 一併驗證）— worktree `verify-S1-S2-S3-S4`
撤銷：`router.go` 裡呼叫 `r.SetTrustedProxies(...)` 的整段程式碼直接拿掉（等於
回到 Gin 預設「信任所有代理」的行為）。**執行 `TestSetupRouter_CORS`、
`TestErrorResponses_ConsistentShapeAcrossEndpoints` 這兩個現有測試：全部依然
PASS，沒有任何測試偵測到這個回歸。**
**結論：⚠️「測試無法偵測回歸」——這是一個要記錄的發現。** 追查原因：現有專門
驗證 S4（`middleware/ratelimit_test.go` 的
`TestRateLimitMiddleware_ClientIPNotSpoofableViaXFF_WhenNoTrustedProxies`）
是自己組一個獨立的 gin engine、手動呼叫 `r.SetTrustedProxies(nil)`，**沒有透過
`router.SetupRouter()` 走完整的組裝流程**，所以只證明了「如果有正確呼叫
`SetTrustedProxies(nil)`，行為是對的」，沒有證明「`SetupRouter` 真的有呼叫它」。
兩者都撤銷、都復原後 `git status` 確認乾淨。

**（worktree 已於驗證後用 `git checkout -- .` 復原、`git worktree remove --force`
移除，主工作目錄全程未變動，`go build`/`git status` 皆確認正常。）**

### M3（Critical）— worktree `verify-M3`
撤銷：`internal/test/test_helpers.go` 的 `SetupTestDB` 改回 `sqlite.Open(":memory:")`
（而不是唯一暫存檔案）。
- 先只跑 `-run TestTransfer_Success_ValidTransfer`（單一測試、`-count=1`）：
  **意外地 PASS**（連跑 5 次都 PASS）。追查原因：這個特定的 M3 bug 需要「同一個
  `*gorm.DB` 的連線池被迫借出第二條實體連線」才會觸發，而 M4 那次重寫（見上方）
  順便把 `Transfer()` 內部「先做兩次非交易查詢、再做兩次交易內查詢」的舊寫法
  拿掉了，改成全程都在 `tx` 裡查詢——單獨跑一個測試時，整個程式從頭到尾都是
  循序、非併發的資料庫存取，連線池幾乎不會被迫開兩條連線，所以這個特定測試
  剛好不會踩到。
- 改成跑整個 `services` package（`go test ./services/... -race -count=1`，
  不加 `-run`）：**4 個測試確實失敗**——`TestTransfer_ConcurrentTransfers_
  NoRaceCondition`（`SQL logic error: no such table: wallets`，代表撈到一個
  完全沒 migrate 過的空白 `:memory:` 資料庫）、`TestCreateUser_Success`、
  `TestCreateUser_Fail_DuplicateUsername`、`TestCreateUser_Fail_DuplicateEmail`
  （三個都是 `no currency available`，代表撈不到前面測試建立的幣別）。
  再用 `-count=5` 重跑整包，同樣的 4 個測試持續失敗。
- **結論：測試會在修正被撤回時失敗，但必須是「跑整個 `services` package」
  （也就是 `go test ./...`／CI 實際的跑法），不能只挑單一測試名稱用 `-run`
  執行——這個細節值得記錄，但不影響「CI 會抓到這個回歸」的結論，因為 CI 本來
  就是跑整個 package，不是挑單一測試。**

### S5（High）— worktree `verify-S5`
撤銷：`router.go` 裡掛載 `cors.New(...)` 中介層的整段程式碼拿掉。測試：
`TestSetupRouter_CORS`。**子測試 `allows_a_configured_origin` 失敗**（預期
`Access-Control-Allow-Origin: https://app.example.com`，實際完全沒有這個標頭，
因為根本沒有掛 CORS 中介層）；另外兩個子測試（`disabled_by_default`、
`rejects_a_non-whitelisted_origin`）維持 PASS——這符合預期，因為「CORS 完全
沒掛」跟「CORS 有掛但沒設定/拒絕非白名單」這兩種狀態，從這兩個子測試個別看
剛好觀察不出差異，只有「白名單 origin 應該被允許」這個案例能分辨兩者。
**結論：測試確實會在修正被撤回時失敗。**

### S8（High）— worktree `verify-S8`
撤銷：`internal/errors/respond.go` 的 `RespondError` 改回只回傳
`gin.H{"error": err.Error()}`（拿掉 `code`、`message` 欄位，模擬修正前「每個
端點自己刻錯誤格式」裡最陽春的那一種）。測試：
`TestErrorResponses_ConsistentShapeAcrossEndpoints`。**全部 7 個子測試都
失敗**（`code` 欄位全部變成空字串，跟預期的 `INVALID_REQUEST`／
`USER_ALREADY_EXISTS`／`FORBIDDEN` 等等對不上；`message` 欄位也變空）。
**結論：測試確實會在修正被撤回時失敗，而且是一次性測到全部 7 種情境，證明
`RespondError` 這個統一收斂點的設計本身就讓「錯誤格式一致性」這件事很好驗證。**

**（以上三組 worktree 皆已用 `git checkout -- .` 復原、`git worktree remove
--force` 移除，主工作目錄全程未變動，`go build`/`git status` 皆確認正常。）**

### N2（Critical，新發現）— worktree `verify-N2`
撤銷：`Transfer()` 的 `committed` flag 收尾機制改回舊版
`defer func() { if r := recover(); r != nil { tx.Rollback(); panic(r) } }()`
（等效於已刪除的 `utils.RollbackIfPanic`，因為該函式已在 N3 修正時被刪除，
在 worktree 裡用等效邏輯內聯重建，不影響驗證結果）。測試：
`TestTransfer_Fail_DoesNotLeakConnection`。**測試失敗**（`sqlDB.Stats().InUse`
變成 `1`，預期是 `0`——連線確實洩漏了，一如 N2 原始發現描述）。
**結論：測試確實會在修正被撤回時失敗。**

### N3（Critical，新發現）— worktree `verify-N3`
撤銷：`UserService.CreateUser` 呼叫 `s.userRepo.CreateUser(user)` 時不再傳入
`tx`（改用 repository 內部預設的非交易連線，模擬修正前的自動提交行為）。測試：
`TestCreateUser_Fail_NoCurrency_DoesNotOrphanUser`。**測試失敗**（刻意讓資料庫
沒有任何幣別、逼 wallet 建立失敗後，`userRepo.GetUserByUsername("orphan")`
**真的查得到**這個孤兒使用者，測試斷言「應該查不到」因而失敗）。
**結論：測試確實會在修正被撤回時失敗。**

**（N2、N3 兩組 worktree 皆已用 `git checkout -- .` 復原、`git worktree remove
--force` 移除，主工作目錄全程未變動，`go build`/`git status` 皆確認正常。）**

### A6（High）— 靜態比對（非 revert-test，理由見下）
`import "testing"` 與 `TransferWithLockOption`（帶不安全轉帳邏輯的 exported
method）在 `main` 的 `services/transaction_service.go` 裡確實存在
（第 11 行、第 166 行），`fix/audit-remediation` 目前的版本用
`grep -rn '"testing"' api --include='*.go' | grep -v _test.go` 確認整個
`api/` 下沒有任何非測試檔案匯入 `testing`。**沒有用 worktree 做 revert-test**，
因為 A6 本質上是「正式程式碼裡不該出現什麼」的靜態衛生問題，不是一段會在
執行期產生可觀察行為差異的邏輯——就算把 `import "testing"` 加回去，
`go build`/`go vet`/`go test` 全部都還是會正常通過（Go 語言合法允許正式程式碼
匯入 `testing` 套件），**沒有任何自動化機制會擋下這個回歸**。
**結論：⚠️「沒有自動化迴歸保護」——這是一個要記錄的發現**（跟 S4 不同：S4 是
「有測試但測試沒測到整合路徑」，A6 是「這類問題本質上不是 go test 能抓的，
需要額外的 lint/grep 檢查步驟，而 `.github/workflows/api.yml` 目前沒有這一步」）。

### Q2（High）— 靜態比對 + 防護機制驗證
`git ls-tree -r --name-only main` 確認 `main/__debug_bin2424212485`、
`wallet-api` 兩個檔案確實在 `main` 的版控歷史裡；同樣指令對
`fix/audit-remediation` 確認兩者都不存在。**與 A6 不同，Q2 有可驗證的持續性
防護機制**：在主工作目錄建立同名的假檔案（`api/wallet-api`、
`api/main/__debug_bin999`）後執行 `git status --short`，**兩個檔案都沒有出現在
輸出裡**，確認 `.gitignore` 的規則（`/api/wallet-api`、`**/__debug_bin*`）
真的會擋下未來意外用 `git add -A` 重新加入這類檔案（驗證後已刪除這兩個測試用
假檔案）。**結論：不只是「這次移除了」，而是「之後也很難不小心加回去」，
防護機制是持續性的。**

### Medium / Low 抽查（至少 5 項，全部用 `git show main:<path>` 對照舊版程式碼 + 執行現有測試）

| 編號 | 對照結果 |
|------|----------|
| S7 | `main` 的 `handlers/user_handler.go` 建立使用者失敗時**只有** `http.StatusInternalServerError` 這一條路徑，沒有任何 409 分支。現在的版本有 `errors.Is(err, services.ErrUserAlreadyExists)` → `409` + `USER_ALREADY_EXISTS`。`TestCreateUser_Fail_DuplicateUsername`、`TestCreateUser_Fail_DuplicateEmail`、`TestCreateUser_Fail_DuplicateUsername_Returns409` 三個測試現況皆 PASS。 |
| S9 | `main` 的 `Transfer()` 有 6 處 `return err`／`return commitDB.Error` 直接把原始 GORM 錯誤往外拋；現在的版本全部改成回傳 `ErrTransferFailed` 這個固定的安全訊息，原始錯誤改寫進 log。 |
| S11 | `main` 的 `models/user_dto.go` 密碼驗證是 `binding:"required,min=6"`；現在是 `min=8`。`TestCreateUser_Fail_PasswordTooShort`（7 碼密碼）現況 PASS。 |
| A2 | `main` 的 README 寫「Swagger (OpenAPI 3.0 docs)」；現在的版本正確描述「Swagger 2.0 + 另外轉出的 OpenAPI 3.0」，且 `api/docs/openapi.yaml` 實際存在、`openapi:` 欄位為 `3.0.3`（V3 已驗證過）。 |
| Q6 | `main` 的 `docker-compose.yml` 只有 3 個 service（`zookeeper`/`kafka`/`postgres`）；現在有 4 個，新增的 `api` service 已在 V4 實際 `docker compose up -d --build` 跑通過完整流程。 |

以上 5 項全部與修正報告的宣稱一致，沒有發現誇大或遺漏。

---

## V2 額外發現（執行過程中發現，修正報告沒提到）

1. **⚠️ S4 沒有整合層級的迴歸測試**（詳見上方 S4 小節）。現有的 S4 測試只驗證
   `middleware.RateLimitMiddleware` 這個中介層單獨運作時是對的，沒有驗證
   `router.SetupRouter()` 真的有呼叫 `SetTrustedProxies`。撤銷 `router.go`
   裡的 `SetTrustedProxies` 呼叫後，`TestSetupRouter_CORS`、
   `TestErrorResponses_ConsistentShapeAcrossEndpoints` 都還是全數通過。
   **建議**：在 `router_test.go` 加一個會透過 `SetupRouter()` 組出真正的
   router、再檢查偽造 `X-Forwarded-For` 不會影響 `ClientIP()` 的整合測試。

2. **⚠️ A6 沒有任何自動化機制防止回歸**（詳見上方 A6 小節）。`import "testing"`
   出現在非 `_test.go` 檔案裡，`go build`/`go vet`/`go test` 都不會失敗，
   純粹是本次人工 grep 才發現、人工修正。**建議**：在
   `.github/workflows/api.yml` 加一個 lint 步驟，例如
   `! grep -rln '"testing"' --include='*.go' . | grep -v _test.go`，讓這類
   回歸真的會讓 CI 變紅。

3. **`transaction_service_test.go` 有 9 處 `assert.Error(t, err)` 緊接著
   `err.Error()` 解參考的寫法**（詳見上方 M4/M6 小節；精確行號：
   192-193、225-226、254-255、279-280、307-308、340-341、363-364、
   386-387、412-413）。這是跟 M3 原本要修的問題同一個模式（`err` 意外為
   `nil` 時整個測試 binary 會 panic，而不是乾淨地讓單一測試失敗），M3
   當時只修了「成功路徑」解參考 `*Wallet` 的地方，這 9 個「失敗路徑」解參考
   `err.Error()` 的地方沒有一併修正。這次驗證 M4/M6 時剛好因為程式碼被撤銷、
   `err` 真的變成 `nil`，親身重現了這個 panic。**建議**：這 9 處全部改成
   `require.Error(t, err)`。

4. **M3 的迴歸測試無法用單一 `-run` 挑選特定測試名稱重現**（詳見上方 M3
   小節），必須整個 `services` package 一起跑才會穩定重現。不影響「CI 會
   抓到回歸」的結論（CI 本來就是整包跑），但如果之後有人想「只跑跟我這次
   改動有關的測試」來省時間，可能會因為只跑單一測試而誤以為 M3 沒有回歸。

### V1 / V2 交叉比對
V1（獨立審查）與 V2（逐項核對）的結論**高度一致**，沒有發現修正報告誇大或
遺漏的情況。V1 額外標記的三個既有小問題（`middleware/validator.go` 手刻
`gin.H`、`RateLimiter.limiters` 無上限成長、`GenerateHash` 用
`UnixNano` 非密碼學亂數）V2 沒有重新驗證（V1 已標注這些是 Low 嚴重度、
且非本次分支引入的既有問題），予以保留、一併列入 V7 結論報告的發現清單。
V2 額外發現的 4 點（上方列表）V1 沒有提到，屬於 V2 在做「撤銷再測試」這種更
主動的驗證方式時才會浮現的問題類型（V1 是純程式碼閱讀比對，不會執行
「刻意撤銷後跑測試」這種操作）。

---

## V3：自動化測試

**驗證紀錄**
- `cd api && go build ./...`：通過，無輸出。
- `cd api && go vet ./...`：通過，無輸出。
- `cd api && go test ./... -race -count=3`：連續三次全部 `ok`（`handlers`、
  `internal/auth`、`internal/config`、`internal/test`、`middleware`、`router`、
  `services` 皆有測試且通過；`db_conn`、`docs`、`cmd/swagger2openapi`、
  `kafka_client`、`main`、`models`、`repositories`、`repositories/entity`、
  `utils` 目前無測試檔，`go test` 回報 `[no test files]`，非失敗）。
- `go test ./internal/test/... -race -v -count=1`：確認
  `TestConcurrentTransfers`、`TestTransfer_NoDeadlock_BidirectionalConcurrentTransfers`
  皆為 `--- PASS`（非 `--- SKIP`），本機 Docker Desktop 有啟動，testcontainers
  真的起了 Postgres 容器並執行完整測試邏輯，不是被跳過。
- 重新執行 `swag init -g main/main.go -o docs`，`git diff --exit-code` 對
  `api/docs/docs.go`、`api/docs/swagger.json`、`api/docs/swagger.yaml` 三個檔案
  皆為 exit code 0（無差異）——已提交的 Swagger 2.0 文件與目前程式碼註解完全同步。
- 重新執行 `go run ./cmd/swagger2openapi`，`git diff --exit-code` 對
  `api/docs/openapi.yaml` 同樣 exit code 0——OpenAPI 3.0 轉檔結果也與已提交版本
  一致。
- 檢查 `.github/workflows/api.yml`：三個步驟（`go build ./...`、`go vet ./...`、
  `go test ./... -race -count=1`）與本機執行的驗證指令一致（本機额外用
  `-count=3` 加強信心，CI 用 `-count=1`，屬合理差異，非不一致）；
  `go-version-file: api/go.mod` 會動態讀取 module 宣告的 Go 版本
  （目前 `go 1.26.0`），不會因為版本寫死而漂移。未實際在 GitHub Actions 上跑過
  （計畫未要求）。

---

## V4：實際啟動 + E2E 驗證

### 準備
- `.env`：從 `.env.example` 複製，`JWT_SECRET` 用 `openssl rand -base64 48` 產生
  全新亂數值，`POSTGRES_PASSWORD` 用 `openssl rand -base64 24`（去掉 `/+=`）產生
  全新亂數值，兩者都跟本次對話中任何先前手動測試用過的值不同。額外把
  `CORS_ALLOWED_ORIGINS` 設成 `https://app.example.com`（而不是留空），才能真的
  測到「允許清單內 origin 通過、清單外被拒」這個邏輯，不是只測「CORS 完全關閉」
  這個預設狀態。`git status .env` 確認未被追蹤，`git check-ignore -v .env`
  確認命中 `.gitignore:27:.env` 規則。
- `docker compose up -d --build`：全部服務啟動，`postgres`／`kafka_client` 皆回報
  `Healthy` 後 `mini-wallet-api` 才啟動；`curl -f /health`、`curl -f /ready`
  皆 200。

### E2E 測試
新增 `api/e2e/`（`//go:build e2e`，一般 `go build ./...`/`go test ./...` 確認會
完全跳過這個套件——`go test ./e2e/...`（不帶 tag）回報
`warning: "./e2e/..." matched no packages`）。共 7 個檔案、20 個 top-level
測試函式（含多個 `t.Run` 子測試），黑箱透過真實 HTTP 打已經跑起來的服務，涵蓋計畫
列出的每一項：

- `TestHealthAndReady`
- `TestRegister_Success_DuplicateRejected_ShortPasswordRejected`（含密碼 < 8 碼）
- `TestRegister_WalletExistsImmediately`（確認註冊後立即查得到錢包，無孤兒使用者）
- `TestLogin_WrongPassword_And_UnknownUser_SameGenericError`
- `TestWallet_HorizontalAccessControl`（含完全沒帶 token 的情況）
- `TestTransfer_Success_BalanceAndHistory`（餘額變動、交易紀錄、分頁欄位）
- `TestTransfer_ValidationFailures`（餘額不足／零／負數／小數位數超過／超過單筆上限，
  5 個子測試，各自驗證獨立的 error code）
- `TestTransfer_SameAccount`
- `TestTransfer_RequiresOwnAccountAsSender`（水平越權：不能用別人的帳號當 from）
- `TestErrorResponseShape_Consistency`（8 種錯誤情境，含登入失敗，格式一致性）
- `TestTxByHash_PublicAccess`
- `TestRateLimit_Login`（含偽造 `X-Forwarded-For` 仍被限流）
- `TestCORS`（白名單 origin 通過、非白名單被拒且不會退回 `*`）
- `TestConcurrency_ConservationAndNoDeadlock`（見下方獨立說明）
- `TestFailurePathStress_ThenNormalTransferSucceeds`

**執行中發現並修正的 E2E 測試設計問題（非產品 bug，屬允許修改的 `api/e2e/` 範圍）**：
第一次執行時，從 `TestFailurePathStress_ThenNormalTransferSucceeds` 開始，後面幾乎
所有測試都因為 `POST /auth/login` 回傳 429 而失敗。追查後確認這**不是產品缺陷**：
登入端點刻意設定了很嚴格的限流（10 req/min、burst 5——這正是 S3 要修的功能，
現在正確生效中），而這個 E2E 套件在很短時間內對同一個來源 IP（測試執行機器）
累積呼叫了遠超過 5 次 `/auth/login`（每個測試各自註冊新使用者並登入），burst
額度在前幾個測試就用完，之後的呼叫在還沒等到 token 補充（每 6 秒補 1 個）前就
接連失敗。這是測試套件本身在短時間內大量呼叫登入端點所致，不是 API 錯誤地限制了
正常使用情境。修法：在 `helpers_test.go` 新增 `loginRaw`/`login`，對 429 自動重試
（間隔對齊限流器實際補充速率，約 6.5 秒，最多 12 次），模擬正常客戶端遇到 429
時應有的退避行為；`TestRateLimit_Login` 本身刻意**不**使用這個重試版本，直接呼叫
`doRaw`，才能真的觀察到未重試的原始 429。修正後完整套件（20 個測試函式）全數通過，
總耗時約 108 秒（含多處因限流退避而產生的等待）。

**併發守恆測試**（`TestConcurrency_ConservationAndNoDeadlock`）：兩個帳號透過真實
HTTP 請求同時互轉（15 輪、共 30 個並發請求，A→B 與 B→A 各 15 個），有 30 秒逾時
保護（用 `select` + `time.After`，若卡住會讓測試明確失敗而非無限掛起）。實測
2.27 秒內全部完成，沒有卡住；轉帳結束後兩人餘額總和與轉帳前完全相同（用
`decimal.Decimal` 精確比較，非字串近似比對）；且轉帳結束後立刻再打一次正常轉帳，
確認回應正常（200 或 400 皆屬預期，只要不是逾時或 5xx），佐證沒有連線或鎖洩漏。
`TestFailurePathStress_ThenNormalTransferSucceeds` 則是連續觸發 20 次「餘額不足」
的轉帳失敗後，確認第 21 次正常轉帳依然成功——這是 N2 修正（連線洩漏）的 HTTP 層級
迴歸測試，補足原本只在 Go 單元測試層級（`TestTransfer_Fail_DoesNotLeakConnection`）
的驗證。

**踩到一個測試撰寫本身的正確性問題並修正**：`TestConcurrency_ConservationAndNoDeadlock`
最初在用 `sync.WaitGroup` 產生的 goroutine 裡直接呼叫會用到 `require.*`（間接呼叫
`t.FailNow()`）的 `do()` 輔助函式——這違反 Go testing 套件「`FailNow` 只能從執行
測試函式本身的 goroutine 呼叫」的規則，屬於我自己撰寫 E2E 測試時引入的問題（不是
產品程式碼問題）。修法：新增不帶 `*testing.T`、只回傳 `(httpResult, error)` 的
`doRaw`，goroutine 內一律用它，結果透過 channel 傳回主測試 goroutine 之後才用
`require`/`assert` 斷言。

### 容器與維運檢查
- **非 root 執行**：`docker inspect mini-wallet-api --format '{{.Config.User}}'`
  回報 `65532`；`docker exec mini-wallet-api id -u` 因為 distroless 沒有 shell
  而失敗（`exec: "id": executable file not found in $PATH`），依計畫的備案改用
  `docker inspect` 佐證。
- **Graceful shutdown**：`docker compose stop api`（送 `SIGTERM`）後，容器 log
  依序出現 `🛑 shutdown signal received, draining in-flight requests...` 與
  `✅ server stopped cleanly`，證實有進入 `srv.Shutdown(ctx)` 的排水流程。
  **小發現（Low，未修正，超出本輪驗證允許修改的範圍）**：`kafka_client.
  KafkaProducer.Close()` 成功關閉時不會印出任何 log（只有失敗時才 log），
  所以從 log 沒辦法「直接」看到 Kafka producer 被關閉這件事，只能從程式碼
  （`main.go` 的 `defer producer.Close()`）與「沒有任何錯誤 log」間接確認。
  建議之後可以加一行成功時的 log，純屬可觀測性小改善，不影響正確性。
- **JWT 密鑰缺失拒絕啟動**：用同一個 image、接上真正的 Postgres/Kafka（DB 連線
  正常成功），但**不帶** `JWT_SECRET` 直接 `docker run`，容器印出
  `❌ invalid JWT secret: jwt secret must be at least 32 characters (...)`
  並以 exit code 1 結束，符合「立即失敗並顯示清楚錯誤」的要求。
- 驗證完成後 `docker compose down -v`：所有容器、network、`pg_data` volume
  皆已清除。

---

## V5：Postman collection 同步

### 範圍比對
逐一列出 collection 內 6 個資料夾、19 個請求的完整路徑與方法，與目前
`router.go` 的路由表逐條比對：**路由完全一致**，沒有缺漏、多餘或改名的端點。

### 發現的過時斷言（已修正）
比對每個請求的 test script 與目前實際回應格式，發現 3 處測試腳本已經對不上
目前的 API 回應，全部屬於「測試腳本過時」而非「API 有問題」：

| 請求 | 舊斷言（錯誤） | 實際回應 | 修正後斷言 |
|---|---|---|---|
| Register Alice / Register Bob | `response.message === "user created"`、讀取 `response.user_id` | `models.UserResponse{id, username, email, created_at}`，無 `message` 欄位，欄位是 `id` 不是 `user_id` | 改讀 `response.id`；成功時另外斷言 `password` 欄位不存在；額外處理 409 情境並檢查 `code === "USER_ALREADY_EXISTS"` |
| Health Check | `response.status === 'ok'` | `{"status":"healthy","service":"mini-crypto-wallet-api"}` | 改為 `response.status === 'healthy'` |
| Readiness Check | `response.database === 'connected'`（該欄位不存在） | `{"status":"ready","service":"mini-crypto-wallet-api"}` | 改為 `response.status === 'ready'` |

修正後三項都用真正啟動的服務（`docker compose up`，全新 `.env`）以 `curl`
逐一重放對應的請求本文，確認回應與新斷言完全吻合，然後 `docker compose
down -v` 清除。

### 強化的斷言（對齊 S8 統一錯誤格式）
以下 3 個既有的負向測試原本只檢查舊的 `error` 字串內容（不算錯，但沒有驗證
S8 新增的穩定 `code` 欄位），予以強化：

| 請求 | 新增斷言 |
|---|---|
| `[Negative] Get Other User's Wallet (403)` | `code === 'FORBIDDEN'` |
| `[Negative] Transfer to Same Account (400)` | `code === 'SAME_ACCOUNT_TRANSFER'` |
| `[Negative] Insufficient Balance (400)` | `code === 'INSUFFICIENT_BALANCE'` |

同樣用真正啟動的服務重放這三個請求的實際本文（含 collection 原本設定的
`amount: 999999`，確認會落在「餘額不足」而非「超過單筆上限」分支），確認
回應 code 值與新斷言完全吻合。

### 未變更但已核對過的部分
- Login Alice / Login Bob 的 test script（`response.token`、`response.user_id`、
  `response.username`、`response.expires_in`）與 `models.LoginResponse` 欄位
  一致，不需修正。
- Transfer 成功、Transaction History 相關 test script 與目前回應格式
  （`{"message":"transfer successful"}`、`TransactionListResponse`、
  `TransactionResponse`）一致，不需修正。
- Collection 內密碼範例（`password123`、`password456`，11 碼）已滿足 S11 的
  `min=8` 規則，不需修正。
- `Mini-Crypto-Wallet.postman_environment.json`：變數名稱與預設值跟 collection
  的實際使用方式一致，沒有發現問題，未修改。

### JSON 格式驗證
`python3 -c "import json; json.load(open('Mini-Crypto-Wallet-API.postman_collection.json'))"`
成功解析，格式有效。`git diff --stat` 顯示改動範圍精準（22 行新增、11 行刪除），
沒有意外改動其他請求。

### `POSTMAN_GUIDE.md` 更新
發現以下過時內容並修正：
- 啟動指令 `go run main/main.go` 沒有標明要在 `api/` 目錄下執行，且完全沒提到
  `docker-compose`（本輪修正後的推薦啟動方式）——改為同時列出 docker-compose
  方式（含 `.env` 設定步驟）與本機直接執行方式（含 `cd api` 與
  `config.yaml.example` 複製步驟）。
- Troubleshooting 段落引用 `config.yaml` 但沒說明它已經是 gitignored、需要從
  `config.yaml.example` 複製——已補充。
- 新增一條 troubleshooting 項目說明 JWT 密鑰長度不足會導致服務拒絕啟動
  （對應本輪修正的 fail-fast 行為，原本文件完全沒提到）。
- 其餘章節（Collection Structure、Environment Variables、Test Scripts
  Included 範例程式碼、Collection Metadata）逐一核對，內容與目前 collection
  一致，不需修改。

**結論**：Postman collection 與 API 目前行為已同步，3 處明確錯誤已修正並用
真實服務驗證過，3 處負向測試已強化以展示 S8 的統一錯誤格式。無 Critical
或 High 等級發現。

---

## V6：合併前安全檢查與歷史清理準備

### 目前工作樹與本分支變更掃描
`git ls-files` 掃描敏感檔名樣式（`.env`、`config.yaml`、`*.pem`、`*.key`、
`id_rsa` 等）：無結果。`git diff main..fix/audit-remediation` 全文掃描
`password=`／`secret=`／`api_key`／私鑰標頭：比對到的字串全部屬於（a）測試檔
內的假密鑰常數（如 `"test-secret-at-least-32-characters-long"`，僅供單元測試
建構 config 物件用）、或（b）本輪修正**移除**的舊字串（也就是下面歷史掃描要
處理的目標本身）。**本分支本身沒有新增任何真實密鑰**。`.env` 檔存在於磁碟但
`git check-ignore -v .env` 確認會被正確擋下，不會被誤 commit。

### 完整 git 歷史掃描
用 `git log --all --full-history --diff-filter=A/D` 找出路徑生命週期、
`git rev-list --objects --all | git cat-file --batch-check` 依大小排序找出
所有曾出現過的大型物件、`git log --all -p` 對全歷史做 AWS key／私鑰標頭／
憑證標頭樣式掃描，找到以下需要清理的路徑（詳細版本、逐 commit 紀錄見
`docs/HISTORY_CLEANUP.md`）：

1. `config.yaml`（搬移前）／`api/config.yaml`（搬移後）——含明文
   `postgres_dsn` 密碼（`password=secret`）與 `jwt_secret:
   your-secret-key-change-in-production-min-32-chars`，即 S1 描述的洩漏源頭。
2. `wallet-api`（根目錄，約 55MB 編譯二進位檔）與
   `main/__debug_bin2424212485`（約 54MB Delve debug 二進位檔）——不含密鑰，
   但佔用大量歷史空間（對應 Q2 的清理對象）。

沒有找到 AWS key、私鑰、憑證等其他類型的洩漏。

**`git-filter-repo`**：`which git-filter-repo` 確認未安裝，依規則記錄後
**未自行安裝**。

### 🔴 額外發現（Critical，但性質特殊，需特別說明）
在掃描歷史的過程中，額外核對了「這些問題目前在遠端的實際狀態」，發現：

- `git remote -v` 確認 `origin` 指向公開 GitHub repo
  `marksue1107/mini-crypto-wallet-api`；用 GitHub 公開 API 確認
  `"private": false`。
- `git show origin/main:api/config.yaml` 確認：**`origin/main` 目前最新一次
  commit 的樹狀結構裡，這個含密鑰的檔案依然存在**——不是「歷史裡查得到」，
  是「現在的 HEAD 就有」。
- `git ls-tree -r origin/main` 確認上述兩個編譯二進位檔也依然存在於
  `origin/main` 目前的樹裡。

**這件事跟「`fix/audit-remediation` 這個分支準不準備好 merge」是兩個不同的
問題**：
- 本分支（`fix/audit-remediation`）在 S1／Q2 的修正（commit `582da7e`、
  `24d6879`）已經把這些檔案從**這個分支的最新狀態**移除，本分支的樹狀結構
  是乾淨的，前面 V1-V5 的驗證結論不受影響。
- 但這些修正**還沒有 merge 回 `main`，也還沒 push**，所以站在
  「這個公開 repo 現在讓外界看到什麼」的角度，密鑰跟大型二進位檔**現在
  依然公開曝露中**，而且會持續曝露到 merge+push 完成為止。

換句話說，**及早完成 merge 並 push 到 `main`，反而是讓這個 Critical 曝露
狀態盡快結束的手段之一**（push 之後 `origin/main` 的 HEAD 就不會再有這些
檔案了）；但 push 只能移除「未來」看到的內容，不會移除「已經在
`origin/main` 歷史裡」的內容——要做到後者，需要額外執行
`docs/HISTORY_CLEANUP.md` 裡的 `git filter-repo` + 強制推送流程，而且不管
有沒有做歷史清理，都應該**盡快輪替**任何跟這兩個外洩值相關的真實憑證。

### 產出
`docs/HISTORY_CLEANUP.md`：完整記錄上述掃描結果、需移除路徑清單、
`.git` 目前大小（52 MiB）與清理後預估、完整的 `git filter-repo` 執行步驟
（備份 → filter-repo → 驗證兩分支 build/test → 加回 remote → 強制推送）、
清理後驗證指令、以及風險注意事項（commit hash 全部改變、需強制推送、
必須在 push/PR 之前執行、GitHub 可能仍保留快取物件需另外聯絡 support）。
**只產生分析與計畫，沒有執行任何會改寫歷史或推送的指令。**