# 前端規格書

輸入來源：`docs/FRONTEND_BRIEF.md`（視覺與設計決定）、`api/docs/openapi.yaml`（API contract）、
`api/router/router.go`（路由與中介層行為）。本文件只定義規格，不含任何前端程式碼。

實際核對 API 行為時，除了 `openapi.yaml` 之外，也讀了以下原始碼以確認 schema 沒寫出來的細節（error
code 對應、驗證規則、資料排序等）：`internal/errors/codes.go`、`handlers/*.go`、
`services/transaction_service.go`、`services/user_service.go`、`middleware/{auth,validator,ratelimit}.go`、
`models/*.go`、`db_conn/conn.go`、`repositories/transaction_repository.go`。凡是規格中提到「後端目前如何」
的敘述，都是照這些檔案目前的實作寫的，不是猜測。

---

## 1. 視覺規範

原樣沿用 BRIEF 第 2 節，未調整任何數值。

### 色票
| 用途 | 色碼 |
|---|---|
| 頁面底色 | `#0B0E11` |
| 卡片／區塊底色 | `#161A1E` |
| 次要按鈕、頭像底色 | `#1E2329` |
| 分隔線、外框 | `#23282F` |
| 主要文字 | `#EAECEF` |
| 次要文字、標籤 | `#848E9C` |
| 停用文字 | `#4A5057` |
| 主色（CTA、連結、當前分頁） | `#F0B90B`（黃） |
| 收入、成功 | `#0ECB81`（綠） |
| 支出、失敗 | `#F6465D`（紅） |

狀態區塊底色用主色的低透明度疊加（如 `#0ECB8119`、`#F6465D14`），外框用 `40` 透明度（如 `#F6465D40`）。

### 字體
- 介面文字：系統無襯線字體。
- **所有數字一律使用等寬字體**：餘額、金額、時間、hash、ID、error code。
- 字級：主要標題 15px、區塊標題 13–14px、內文 13px、標籤與輔助說明 11–12px。餘額主數字 32px，
  Explorer 金額 28px，指標卡數字 18–20px。
- 字重只用 400 與 500，不使用更粗的字重。

### 數字呈現規則
- 金額一律補滿至幣別的小數位數（USDT/BTC/ETH 為 8 位），例如 `250.00000000`。
- 千分位加逗號。
- 收入前綴 `+` 並用綠色，支出前綴 `−`（U+2212，非減號 hyphen）並用紅色。
- 時間精確到秒；Explorer 頁顯示到毫秒並標註時區。
- hash 一律縮寫顯示（前 6 後 4，如 `0x7f3a2b…c21e`），可複製完整值。

> 核對：目前資料庫只 seed 了一顆幣別 `USDT`，`decimals = 8`（`db_conn/conn.go`），所以「8 位小數」
> 這個假設與後端現況一致。小數位數不應該在前端寫死 `8`，而是永遠讀 `GET /currencies` 回傳的
> `decimals` 欄位——現在剛好都是 8，但邏輯上要走這個欄位，之後加新幣別才不用改前端程式碼。

### 元件樣式
- 圓角：卡片 8px、輸入框與按鈕 6px、小徽章 4–5px。
- 外框：`0.5px solid #23282F`；輸入框聚焦時外框改為 `#F0B90B`。
- 主要按鈕：黃底 `#F0B90B`、深色文字 `#0B0E11`；次要按鈕：`#1E2329` 底、淺色文字。
- **每個畫面最多一顆黃色主要按鈕**，其餘用次要樣式。
- 收支方向除了顏色，一律搭配箭頭圖示（收入 `arrow-down-left`、支出 `arrow-up-right`），不可只靠顏色
  區分。
- 導覽列當前分頁：文字用主要文字色，下方 2px 黃色底線。

### 文案
- 介面文字用英文，句首大寫、其餘小寫（sentence case），不用全大寫或每字大寫。
- **所有錯誤訊息同時顯示人類可讀的說明與後端回傳的 error code**（code 用等寬字體、次要文字色）。

---

## 2. 路由表與登入狀態

### 2.1 路由表

| 路徑 | 頁面 | 登入需求 | 說明 |
|---|---|---|---|
| `/login` | Login / Register | 僅限未登入 | 已登入時直接 redirect 到 `/` |
| `/` | Overview | 需登入 | |
| `/transfer` | Transfer | 需登入 | |
| `/history` | History | 需登入 | |
| `/explorer` | Explorer | **不需要** | 見 2.4 |
| `/explorer/:hash` | Explorer（帶結果） | 不需要 | 允許直接分享連結，載入後自動查詢該 hash |

導覽列（Overview / Transfer / History / Explorer）與右側使用者資訊只在已登入狀態渲染；未登入時
`/explorer` 用獨立的最簡 header（左側 logo + Public 徽章、右側一個「Sign in」連結），不顯示完整導覽列。

### 2.2 Token 儲存

- `POST /auth/login` 回傳 `{ token, user_id, username, expires_in }`（`expires_in` 目前固定為
  86400 秒／24 小時，對應 `router.go` 裡 `auth.NewJWTManager(jwtSecret, 24*time.Hour)`）。
- 後端目前**沒有** cookie-based session（沒有 `Set-Cookie`、沒有 refresh token endpoint），token
  只透過 JSON body 回傳，且每次 API 呼叫都要手動帶 `Authorization: Bearer <token>`。這代表 httpOnly
  cookie 那種比較安全的存法在現有後端下做不到，除非後端改動。
- 前端儲存策略：token + `user_id` + `username` 存在 `localStorage`，並在記憶體中（例如 React context /
  TanStack Query 的 auth store）保留一份供 API client 攔截器讀取。這是刻意的決定（見第 7 節）：
  demo 情境下「關閉分頁就要重新登入」的體驗成本，高於「沒有 revoke 機制的 bearer token 多存活一段
  時間」的風險，換取重新整理瀏覽器／關閉分頁再打開都不用重新登入。§2.3 的過期檢查與 401 全域處理
  邏輯不受影響——token 是否還在有效期內一律由 `expires_at` 判斷，不是由存放位置決定。
- 同時存 `expires_at = login 當下時間 + expires_in * 1000`（毫秒 timestamp），供 2.3 的過期判斷用。

### 2.3 過期處理

- 路由層：每次進入需登入頁面前，檢查 `expires_at` 是否已過期；過期就直接清空 session 並導向
  `/login?redirect=<原路徑>`，不送出任何 API 請求。
- API 層：任何請求收到 `401 UNAUTHORIZED`（token 缺失／格式錯／過期／簽章錯，這四種在後端目前
  共用同一個 code，見 §4）一律視為「session 失效」，清空 session、導向 `/login?redirect=<原路徑>`，
  並在登入頁顯示一次性提示「Session expired, please sign in again」。
- 登出是純前端行為：JWT 是 stateless 的，後端沒有任何 revoke／黑名單機制，「登出」只是清掉前端存的
  token，該 token 本身在到期前仍然有效（如果被別人取得仍可使用）。這是現有設計的限制，前端無法補，
  只需要不要讓文案暗示「登出後 token 立即失效」。

### 2.4 受保護路由（Explorer 的特殊性）

`GET /tx/:hash` 是 router.go 裡明確標成公開、不需要 auth 的端點（設計上比照區塊鏈 explorer：知道
hash 就能查）。因此 Explorer 頁面本身**不強制登入**：
- 已登入：走完整版面（含導覽列、當前分頁底線在 Explorer）。
- 未登入：走精簡版面，但功能完全相同（搜尋、結果卡片都可用），並在頁尾強調「Public」與「不需登入」
  的說明（BRIEF 3.5 已經要求）。

其餘三頁（Overview / Transfer / History）呼叫的 API 全部要求 `Authorization: Bearer`
（`protected` route group in router.go），且 handler 內用 `RequireUserID` 檢查 path 上的
`user_id` 必須等於 token 裡的使用者 id，否則回 `403 FORBIDDEN`。前端只要永遠用登入回傳的
`user_id` 去打這些 API（不要讓使用者手動輸入自己的 user_id），正常操作下不會觸發 403；403 仍要有
通用錯誤處理（見 §4），避免萬一觸發時整頁空白。

---

## 3. 頁面規格

### 3.1 Login / Register

**版面結構**：單一頁面卡片置中，卡片內兩個分頁（Sign in / Create account）切換，不換路由。

**Sign in 分頁**
- 欄位：Username、Password。
- 主按鈕「Sign in」。
- 失敗訊息固定文案「Invalid username or password」+ code `INVALID_CREDENTIALS`（不論帳號不存在或
  密碼錯誤，後端本來就回同一個錯誤，見 `services/user_service.go` 的
  `ErrInvalidCredentials` 註解——這是刻意設計，防止帳號列舉，前端不用也不能做更細的區分）。
- 底部文案：「Sessions last 24 hours.」

**Create account 分頁**
- 欄位：Username（3–50 字）、Email、Password（至少 8 碼）。
- Username／Email 即時（debounce，例如 500ms）呼叫 `POST /users` 之前**不**做重複性預檢查——後端
  沒有「查詢是否已存在」的獨立端點，重複與否只能在送出當下由 `409 USER_ALREADY_EXISTS` 得知。故
  「即時顯示錯誤」實際上是「送出後即時顯示」，不是打字時即時查。欄位失焦（blur）時做的是純前端格式
  檢查（長度、email 格式），不是唯一性檢查。
- 密碼規則提示：後端目前只驗證 `min=8`（`models/user_dto.go` 的 `binding:"required,min=8"`），沒有
  大小寫／數字／符號等規則。**不做任何弱／中／強視覺化分級**（見第 7 節第 4 項），只保留一個規則
  勾選項「At least 8 characters」，輸入時即時打勾／取消，如實反映後端真正會擋的唯一規則，不暗示任何
  後端不存在的複雜度要求。
- 主按鈕「Create account」。
- 成功後：後端 `POST /users` **不會**回傳 token（只回 `UserResponse`），需要另外呼叫
  `POST /auth/login` 才能拿到 token。前端在註冊成功後，直接用同一組 username/password 自動呼叫
  登入 API，成功就直接導向 `/`；如果自動登入失敗（理論上不該發生）才退回 Sign in 分頁並帶出剛剛的
  username。
- 頁面底部註明：「Your account and wallet are created together in a single atomic transaction.」
  （對應 `services/user_service.go` 的 `CreateUser` 是同一個 DB transaction 建立 user + wallet）。

**狀態**
- 載入中：按鈕內 spinner + disable，不用整頁骨架屏（這頁沒有資料區塊）。
- 錯誤：見 §4 對應表（`INVALID_REQUEST` 含欄位級 `details`、`USER_ALREADY_EXISTS`、
  `INVALID_CREDENTIALS`、`RATE_LIMIT_EXCEEDED`）。
- 這頁沒有「空狀態」與「未登入導向登入頁」的情境（本身就是登入頁）。

**API 對應**

| 動作 | API | 欄位對應 |
|---|---|---|
| 註冊 | `POST /users` | `username`,`email`,`password` → `UserCreateRequest`；成功回 `UserResponse{id,username,email,created_at}` |
| 登入 | `POST /auth/login` | `username`,`password` → `LoginRequest`；成功回 `LoginResponse{token,user_id,username,expires_in}` |

---

### 3.2 Overview（錢包總覽）

**版面結構**（由上到下）
1. 總餘額區：32px 等寬數字 + 眼睛圖示（隱藏金額，純前端狀態，不呼叫 API）+ 24 小時變化 + 右側
   Transfer（主按鈕）/ Deposit（次按鈕）。
2. 三張指標卡：總交易數、24h 支出、24h 收入。
3. Assets 區塊：幣別列表。
4. Recent activity：最近數筆交易 + 右上角「View all」→ `/history`。

**內容與 API 對應**

| 區塊 | 資料來源 | 說明 |
|---|---|---|
| 總餘額 | `GET /wallet/{user_id}` → `balance` | 目前只有一顆錢包（見 §5-1），故「總餘額」= 這顆錢包的 balance，不是加總。 |
| Deposit 按鈕 | 無對應 API | 純 UI 佔位，點擊顯示「Deposits aren't supported yet」的提示，不導向任何頁面（後端沒有入金端點）。 |
| 指標卡：總交易數 | `GET /transactions/{user_id}` 的 `pagination.total` | 呼叫一次 `page=1&page_size=1` 即可拿到 `total`，不用抓全部資料。 |
| 指標卡：24h 支出／收入、總餘額旁的「24 小時變化」 | `GET /wallet/{user_id}/stats`（`?window=24h`，也是預設值）→ `total_sent`/`total_received`/`balance_change` | 由資料庫直接聚合計算的精確值，不是前端從交易紀錄估算的（見下方）。「24 小時變化」直接顯示 `balance_change`（帶 `+`/`−` 前綴的絕對金額，不是百分比）。 |
| Assets 列表 | `GET /wallet/{user_id}` + `GET /currencies/{currency_id}` | 只有一列：幣別圖示（前端自備的靜態圖示，依 `code` 對應）、`name`、`balance`、估值欄位隱藏（見 §5-3）。 |
| Recent activity | `GET /transactions/{user_id}?page=1&page_size=5` | 每筆：方向圖示（比較 `from_user_id`/`to_user_id` 與目前登入的 `user_id`）、對象顯示 `user #{對方 user_id}`（見 §5-4）、`hash` 縮寫、`amount`、`created_at`。 |

**互動行為**
- 眼睛圖示：切換金額顯示為 `••••••••` 或實際數字，純前端 state，不持久化（每次重新整理預設顯示）。
- Transfer / Deposit 按鈕：導向 `/transfer`；Deposit 見上方說明。
- Recent activity 每列 hash：點擊導向 `/explorer/:hash`。
- View all：導向 `/history`。

**狀態**
- 載入中：總餘額、三張指標卡、Assets、Recent activity 各自獨立骨架屏（不要求同時完成才顯示，各區塊
  各自 loading/ready）。
- 空狀態：Recent activity 若 `pagination.total === 0`（帳號剛註冊、沒有任何交易），顯示「No
  transactions yet」+ 「Send your first transfer」CTA 連到 `/transfer`；三張指標卡在這種情況下數字
  顯示 `0`，不特別做空狀態插圖。Assets 永遠至少一列（註冊即建立錢包），不會空。
- 錯誤：任一區塊的 API 失敗都在該區塊內顯示錯誤卡片（說明 + code + Retry 按鈕），不影響其他區塊。
- 401：全域攔截器處理（見 2.3），不需要這頁自己處理。

---

### 3.3 Transfer（轉帳）

**版面結構**（由上到下）
1. 幣別選擇（下拉）。
2. 收款人輸入框。
3. 金額輸入框：上方右側顯示可用餘額；框內右側 MAX 快捷鍵 + 幣別標示。
4. 限制說明列：「Up to 8 decimal places」+ 單筆上限（顯示 `GET /currencies` 回傳的
   `max_transfer_amount` 實際數值，見下方）。
5. 摘要區塊：轉帳後餘額（= 目前餘額 − 輸入金額，前端即時計算）、手續費（固定顯示 `0`，見 §5.7）。
6. 主按鈕「Review transfer」→ 開啟確認彈窗（二次確認，顯示幣別／收款人／金額／轉帳後餘額／手續費
   摘要 + 「Confirm transfer」/「Back」）→ 確認後才真正呼叫 API。

**欄位與驗證**

| 欄位 | 前端驗證（送出前） | 對應後端行為 |
|---|---|---|
| 幣別 | 必選，選項來自 `GET /currencies`（只顯示 `is_active === true`） | `currency_id` |
| 收款人 | 必填；輸入使用者名稱，欄位失焦（blur）時呼叫 `GET /users/lookup?username=` 解析成 `to_user_id` 並顯示對方名稱做確認；查無此人時欄位下方顯示「No user found with that username」，不阻擋繼續輸入但送出前必須先解析成功 | `to_user_id` |
| 金額 | > 0；小數位數 ≤ 所選幣別的 `decimals`；不超過 `max_transfer_amount`；收款人與自己相同時直接前端擋下 | `amount` |

**收款人欄位**：轉帳收的是 `to_user_id`，但 `GET /users/lookup?username=` 讓前端可以先把使用者輸入
的名稱解析成 id 並顯示對方使用者名稱做二次確認，不用再讓使用者直接輸入裸的 user_id（見 §5.2，此項
已由後端支援並實作）。

**單筆上限**：載入頁面時透過 `GET /currencies` 取得所選幣別的 `max_transfer_amount`，直接顯示在限制
說明列（例如「Up to 8 decimal places · Max 1,000,000 USDT per transfer」），並在前端送出前就用這個
數值擋下超額輸入，不用等送出後才由 `AMOUNT_EXCEEDS_LIMIT` 錯誤告知（見 §4）。

**確認彈窗**：符合 BRIEF「必須有確認步驟，不可一鍵直接送出」的要求。彈窗內不重新驗證，直接沿用送出
前的欄位值；「Confirm transfer」點擊後才呼叫 `POST /wallet/transfer`，呼叫期間彈窗內按鈕顯示
loading，兩顆按鈕都 disable，避免重複送出。

**成功後顯示交易 hash**：`POST /wallet/transfer` 成功時直接回傳建立的交易物件
（`TransactionResponse{id,from_user_id,to_user_id,currency_id,amount,hash,signature,status,created_at}`），
前端直接用回應裡的 `hash` 顯示並連結到 `/explorer/:hash`，不需要額外呼叫任何 API 去猜測剛剛建立的
是哪一筆交易。

**錯誤狀態涵蓋**（送出 `POST /wallet/transfer` 之後）：`INSUFFICIENT_BALANCE`、
`AMOUNT_NOT_POSITIVE`／`INVALID_DECIMALS`／`AMOUNT_EXCEEDS_LIMIT`（三者都已由前端預先驗證擋下，仍
保留伺服器端錯誤作為 fallback，見 §4）、`SAME_ACCOUNT_TRANSFER`（前端已預先擋下，此為 fallback）、
`SENDER_WALLET_NOT_FOUND`（理論上不該出現，代表自己這個幣別沒有錢包）、`RECIPIENT_NOT_FOUND`
（收款人不存在，或在這個幣別沒有錢包——理論上也不該出現，因為收款人已經先透過 lookup 解析過，除非
兩次操作之間收款人被刪除）、`NOT_FOUND`（幣別在選單載入之後被停用的極端情況）。所有錯誤都在彈窗內
或原表單上方以錯誤 banner 呈現，不關閉彈窗，讓使用者可以修改金額重試。

**狀態**
- 載入中：幣別選單與可用餘額各自骨架屏；沒有資料可用時（例如錢包 API 失敗）整個表單 disable 並顯示
  區塊錯誤 + Retry。
- 空狀態：不適用（表單頁）。
- 錯誤：見上。

**API 對應**

| 動作 | API |
|---|---|
| 載入幣別（含小數位數、單筆上限） | `GET /currencies` |
| 載入可用餘額 | `GET /wallet/{user_id}` |
| 解析收款人名稱 | `GET /users/lookup?username=` |
| 送出轉帳（回應含 `hash`） | `POST /wallet/transfer` |

---

### 3.4 History（交易紀錄）

**版面結構**
1. 篩選列：All / Sent / Received 分頁（Sent、Received 停用，見 §5-5）、幣別下拉（停用）、日期區間
   （停用）、右側 Export。
2. 表格：方向圖示、對象與方向文字、hash、金額與幣別、時間。
3. 分頁列：「Showing {start}–{end} of {total}」+ 頁碼控制（上一頁／下一頁／頁碼）。

**表格欄位對應**

| 欄位 | 資料來源 | 呈現 |
|---|---|---|
| 方向圖示 | 比較 `from_user_id`/`to_user_id` 與目前 `user_id` | `arrow-up-right`（紅，Sent）或 `arrow-down-left`（綠，Received） |
| 對象與方向文字 | 同上 | 「Sent to user #{to_user_id}」/「Received from user #{from_user_id}」，見 §5-4 |
| hash | `hash` | 縮寫顯示，點擊複製、再點導向 `/explorer/:hash` |
| 金額與幣別 | `amount` + `currency_id`（查對應幣別的 `decimals`/`code`） | `+`/`−` 前綴、綠/紅、依 `currency_id` 對應幣別的小數位數補滿（目前系統仍只有 USDT，實際顯示不變） |
| 時間 | `created_at` | 時:分:秒 + 日期 |

**分頁**：用 `GET /transactions/{user_id}?page={n}&page_size=20` 對應
`PaginationResponse{page,page_size,total,total_pages}`；「Showing 1–20 of 147」由前端算
`(page-1)*page_size+1` 到 `min(page*page_size, total)`。

**Export**：後端沒有匯出 API（BRIEF §5-7 已明訂由前端處理，見 §5.6）。做法：點擊 Export 前，先用
`GET /transactions/{user_id}?page=1&page_size=1` 讀一次 `pagination.total`：
- `total <= 1000`：用既有的分頁 API 依序把**全部**頁面撈完（`page_size=100` 一次撈滿上限以減少請求
  數），組成 CSV 後在瀏覽器下載，不是只匯出目前這一頁。
- `total > 1000`：**不**發動全量拉取，改為彈出提示「This account has {total} transactions — narrow
  the date range or currency filter before exporting」（實際文案視 §5.5 篩選功能是否已上線調整），
  Export 按鈕維持 disabled 直到符合上限。這個 1000 筆的上限是前端自訂的安全閥，避免帳號交易量變大時
  觸發幾十次循序分頁請求造成瀏覽器/使用者體驗卡頓；門檻本身沒有對應到任何後端限制，純粹是前端判斷
  （見第 7 節），之後如果門檻不合適可以再調整，或等後端補一個真正的匯出端點（見 §5.6）後直接拿掉
  這個限制。

**狀態**
- 載入中：表格骨架屏（列狀骨架，不轉圈圈）。
- 空狀態：`total === 0` 時顯示「No transactions yet」，不顯示表格與分頁列。
- 錯誤：表格區塊整塊替換為錯誤卡片 + Retry，篩選列仍可操作（但停用的篩選項目本來就不可互動）。
- 401：全域攔截器處理。

---

### 3.5 Explorer（公開交易查詢）

**版面結構**
1. 右上角「Public」徽章。
2. 搜尋框（輸入 hash）+ 說明文字「Anyone with the hash can look up this transaction — no sign-in
   required.」
3. 結果卡片（查到資料才顯示）：
   - 狀態標頭：圖示 + 完成時間（毫秒 + 時區）+ 狀態徽章。
   - From / To 兩側對照，箭頭連接。
   - 置中大金額（28px）。
   - 明細列表：hash（可複製完整值）、signature（旁註「Demo signature — not real cryptography」）、
     幣別與小數位、交易 ID。
4. 頁尾說明：「Only the transaction itself is public. Balances and account details are not
   exposed.」

**狀態欄位的現實**：`TransactionResponse.status` 的 schema 允許
`pending/processing/completed/failed/cancelled`（`models/transaction.go` 的欄位註解），但目前唯一
會建立交易的路徑（`services/transaction_service.go` 的 `Transfer()`）永遠寫死 `Status: "completed"`，
沒有任何程式碼路徑會產生其他狀態。狀態徽章元件要照 schema 支援全部五種狀態的樣式（用色票的成功/失敗
色系分流），但**目前實際上只會看到 `completed`**——這不是前端要處理的空缺，只是現況記錄，之後後端如果
真的有非同步/失敗的交易，徽章元件已經準備好了。

**互動行為**
- 輸入 hash 後 Enter 或按鈕觸發 `GET /tx/{hash}`；也支援 `/explorer/:hash` 直連時自動觸發。
- From/To 顯示 `user #{id}`（無名稱 API，見 §5-4）。
- signature／hash 都可點擊複製到剪貼簿。

**狀態**
- 初始（尚未搜尋）：顯示搜尋框 + 說明文字，不顯示錯誤或空狀態卡片。
- 載入中：結果卡片位置顯示骨架屏。
- 找不到（`404 TRANSACTION_NOT_FOUND`）：卡片位置顯示「No transaction found for this hash」+ code，
  不算「錯誤」（不提供 Retry，因為重試同一個 hash 沒有意義），只提示檢查輸入。
- 錯誤（其他狀態碼，如 `429 RATE_LIMIT_EXCEEDED`）：一般錯誤卡片 + Retry。
- 這頁不需要「未登入導向登入頁」——它本來就不需要登入（見 2.4）。

**API 對應**

| 動作 | API | 欄位對應 |
|---|---|---|
| 查詢交易 | `GET /tx/{hash}` | `TransactionResponse{id,from_user_id,to_user_id,currency_id,amount,hash,signature,status,created_at}` |

> `TransactionResponse` 現在含 `currency_id`，Explorer／History／Overview 顯示金額時可以（也應該）用
> `currency_id` 查對應幣別的 `decimals`／`code` 來格式化，而不是像先前那樣假設全部交易都是 USDT。
> 目前系統仍然只 seed 了 USDT 一種幣別，所以實際顯示結果不變，但邏輯上要走 `currency_id`，之後加新
> 幣別交易才不用改前端程式碼。

---

## 4. 錯誤處理策略

所有錯誤回應統一格式 `{ error, code, message }`（部分 400 額外帶 `details: FieldValidationError[]`）。
前端一律依 `code` 分流處理，`message`／`error` 只用來顯示（畫面同時顯示訊息與 code，等寬字體、次要
文字色），不用字串比對做邏輯判斷。

| Code | HTTP | 會出現在哪些 API | 人類可讀訊息（前端顯示文案） | 前端處理 |
|---|---|---|---|---|
| `INVALID_REQUEST` | 400 | 所有有 request body/query 的端點（綁定/驗證失敗） | 「Please check the highlighted fields.」，若有 `details` 則逐欄位顯示（欄位名 + 規則說明） | 表單內逐欄標紅，不彈全域錯誤 |
| `UNAUTHORIZED` | 401 | 所有受保護端點（缺 token／格式錯／過期／簽章錯，四種狀況共用同一個 code） | 「Session expired, please sign in again.」 | 全域攔截器：清 session、導向 `/login?redirect=...`（見 2.3） |
| `FORBIDDEN` | 403 | `GET /wallet/:user_id`、`POST /wallet/transfer`、`GET /transactions/:user_id`（存取別人的資源） | 「You don't have access to this resource.」 | 一般錯誤卡片；正常操作流程下不應出現（見 2.4），出現時當作異常上報／記錄，不特殊導頁 |
| `NOT_FOUND` | 404 | `GET /currencies/:id`、轉帳時幣別不存在 | 「Not found.」 | 一般錯誤卡片 |
| `INTERNAL_ERROR` | 500 | 各端點的未預期失敗 | 「Something went wrong on our end. Please try again.」 | 一般錯誤卡片 + Retry |
| `RATE_LIMIT_EXCEEDED` | 429 | `POST /users`（60/min）、`POST /auth/login`（10/min，較嚴格）、`POST /wallet/transfer`（60/min）、`GET /tx/:hash`（60/min） | 「Too many requests. Please wait a moment and try again.」 | 顯示錯誤，按鈕短暫 disable（可選：讀 `Retry-After` header 顯示倒數，非必要） |
| `USER_ALREADY_EXISTS` | 409 | `POST /users` | 「That username or email is already registered.」 | 註冊表單內標示對應欄位 |
| `INVALID_CREDENTIALS` | 401 | `POST /auth/login` | 固定文案「Invalid username or password」（不透露帳號是否存在，見 3.1） | 表單內錯誤，不導頁（跟 `UNAUTHORIZED` 是不同 code，不要共用處理邏輯） |
| `WALLET_NOT_FOUND` | 404 | `GET /wallet/:user_id`（只有一種情況，不會混淆） | 「Wallet not found.」 | 一般錯誤卡片 |
| `SENDER_WALLET_NOT_FOUND` | 404 | `POST /wallet/transfer`（自己這個幣別沒有錢包，理論上不該出現） | 「You don't hold a wallet for this currency.」 | 表單內錯誤 |
| `RECIPIENT_NOT_FOUND` | 404 | `POST /wallet/transfer`（收款人不存在，或收款人在這個幣別沒有錢包） | 「Recipient not found, or doesn't hold this currency.」 | 表單內錯誤，收款人欄位標紅（正常流程下應該先被 §3.3 的 `GET /users/lookup` 擋下，這裡是 fallback） |
| `INSUFFICIENT_BALANCE` | 400 | `POST /wallet/transfer` | 「Insufficient balance for this transfer.」 | 表單內錯誤，金額欄位標紅 |
| `AMOUNT_NOT_POSITIVE` | 400 | `POST /wallet/transfer`（金額為零或負數） | 「Amount must be greater than zero.」 | 表單內錯誤（正常流程下應該先被前端驗證擋下，這裡是 fallback） |
| `INVALID_DECIMALS` | 400 | `POST /wallet/transfer`（小數位數超過該幣別支援的位數） | 「Amount has too many decimal places for this currency.」 | 同上，fallback |
| `AMOUNT_EXCEEDS_LIMIT` | 400 | `POST /wallet/transfer`（超過 `GET /currencies` 回傳的 `max_transfer_amount`） | 「Amount exceeds the per-transfer limit.」 | 同上，fallback |
| `SAME_ACCOUNT_TRANSFER` | 400 | `POST /wallet/transfer` | 「You can't transfer to your own account.」 | 前端已預先擋下，此為 fallback |
| `TRANSACTION_NOT_FOUND` | 404 | `GET /tx/:hash` | 「No transaction found for this hash.」 | Explorer 專用的「查無結果」，非一般錯誤（見 3.5） |
| `TRANSACTION_FAILED` | 500 | `POST /wallet/transfer`（非預期失敗，例如 DB commit 失敗） | 「Transfer failed. Please try again.」 | 一般錯誤 + Retry（重試等同重新送出，需回到表單而非重打同一個請求） |
| `USER_NOT_FOUND` | 404 | `GET /users/lookup`（查無此使用者名稱） | 「No user found with that username.」 | Transfer 頁收款人欄位下方顯示，見 §3.3；非全域錯誤 |

**關於錯誤 code 的設計**：先前 `WALLET_NOT_FOUND`／`INVALID_AMOUNT` 各自涵蓋兩到三種、後端訊息文字
不同但 code 相同的情況，靠 `message` 原文區分——這在第一版規格裡是已知妥協，現在後端已經拆成上面
這幾個各自明確、彼此互斥的 code（`docs/BACKEND_PREP_PLAN.md` 第 1 批），前端可以（也應該）純粹依
`code` 分流，不需要再解析 `message` 文字內容。`message`／`error` 欄位維持只用來顯示（BRIEF 要求同時
顯示訊息與 code），不參與任何判斷邏輯。

---

## 5. 後端目前不支援的功能

> 這一節是 BRIEF §5 原始清單裡，`docs/BACKEND_PREP_PLAN.md` 兩批補強**之後仍然沒有處理**的部分（已
> 處理的項目：轉帳回應含交易物件與 `hash`、error code 拆細、`GET /wallet/{user_id}/stats`、
> `GET /currencies` 的 `max_transfer_amount`、`GET /users/lookup`——這些已經反映在第 1–4 節裡，不再
> 出現在這裡）。其中影響範圍夠大、值得追蹤的四項（5.1、5.3、5.5、5.6）已整理進 `docs/BACKLOG.md`
> （`[FE1]`–`[FE4]`）；5.4 是獨立的小缺口（見該節說明，不算在這四項裡）；5.7 不是缺口，是產品決定。

### 5.1 列出使用者所有幣別錢包（`docs/BACKLOG.md` [FE1]）
第一版只顯示單一錢包（`GET /wallet/{user_id}` 本身也只回一顆）。Assets 區塊保留清單版面，但目前
永遠只有一列。

**建議新增 API**：`GET /wallets` （帶 auth，回傳目前登入使用者名下所有錢包）
```
GET /wallets
→ 200: [ { id, user_id, currency_id, balance, created_at }, ... ]
```

### 5.2 以使用者名稱查詢收款人 —— 已支援
~~轉帳收的是 `to_user_id`，沒有依名稱查詢的端點~~。已由 `GET /users/lookup?username=` 支援（見
`docs/BACKEND_PREP_PLAN.md` 第 2 批 2.3），細節見 §3.3——這裡保留編號只是為了跟 BRIEF 原始清單
對照，不代表還有缺口。

### 5.3 估值（USDT value）（`docs/BACKLOG.md` [FE3]）
沒有匯率來源。第一版隱藏估值欄位（Assets 區塊、指標卡都不顯示任何以外幣計價的估值）。

**建議新增 API**：`GET /rates?base=USDT` 或類似的匯率端點；由於這是展示用途，静態/固定匯率表即可，
不必接真實匯率源。

### 5.4 交易對象顯示名稱（`docs/BACKLOG.md` [FE2] 的相關項目，見下方說明）
交易紀錄回傳 `from_user_id`/`to_user_id`，不含名稱。第一版一律顯示 `user #{id}`。注意
`GET /users/lookup` 是「username → id」，不是「id → username」，不能直接拿來反解交易紀錄裡的
`user_id`，所以這裡仍然是一個獨立的缺口，不是 §5.2 的副產品。

**建議新增 API**：批次查詢，例如 `GET /users/batch?ids=1,2,3` 回傳 `[{id, username}, ...]`，避免
前端在交易列表裡對每一筆交易各自打一次單筆查詢。

### 5.5 Sent / Received、幣別、日期篩選（`docs/BACKLOG.md` [FE2]）
後端 `GET /transactions/{user_id}` 目前只吃 `page`/`page_size`。第一版篩選列僅 All 可互動，其餘
（Sent/Received tab、幣別下拉、日期區間）在畫面上顯示但 disabled，附 tooltip「Coming soon」。

**建議新增 API**：`GET /transactions/{user_id}?direction=sent|received&currency_id=&from=&to=`
（`direction` 由後端依 `from_user_id`/`to_user_id` 篩，比前端拿全部資料再篩省流量）。

### 5.6 Export（`docs/BACKLOG.md` [FE4]）
由前端將分頁資料轉成 CSV（見 §3.4，第一版做法是把所有分頁依序拉完再匯出，並加上 1000 筆的安全上限，
超過就提示使用者縮小範圍，不再嘗試全量拉取）。

**建議新增 API**（若未來資料量變大）：`GET /transactions/{user_id}/export?format=csv`，由後端直接
串流輸出，不受前端分頁上限（`page_size` 上限 100）與 §3.4 那個 1000 筆前端安全閥影響。

### 5.7 手續費
系統無手續費概念，固定顯示 `0` 並標示為「Zero fees」。無需新增 API（沒有對應的 `docs/BACKLOG.md`
項目，這不是缺口，是產品決定）。

---

## 6. 技術選型與 `web/` 資料夾結構

技術選型照 BRIEF 第 6 節，不調整：
- Vite + React + TypeScript
- Tailwind CSS + shadcn/ui（覆寫為本文件第 1 節的深色色票）
- TanStack Query（資料存取與快取）
- React Router
- API client／型別由 `api/docs/openapi.yaml` 自動生成（orval 或 openapi-typescript），禁止手寫
- MSW（開發期 mock、測試）
- Vitest + Testing Library（單元）、Playwright（E2E／截圖驗收）

### 資料夾結構（`web/` 與 `api/` 平行，位於 repo 根目錄）

```
web/
├── go.mod                  # 最小 stub，避免 Go 工具掃描這個目錄（BRIEF §6 要求）
├── package.json
├── vite.config.ts
├── tailwind.config.ts
├── tsconfig.json
├── index.html
├── public/
├── src/
│   ├── api/
│   │   ├── generated/      # orval/openapi-typescript 產出，不手動編輯，隨 openapi.yaml 重新生成
│   │   └── client.ts       # axios/fetch instance：baseURL、Authorization header 攔截器、401 全域處理
│   ├── auth/
│   │   ├── session.ts      # token/user_id/expires_at 的 localStorage 存取
│   │   └── AuthProvider.tsx
│   ├── routes/
│   │   ├── router.tsx      # React Router 路由表（對應本文件 §2.1）
│   │   ├── login/
│   │   ├── overview/
│   │   ├── transfer/
│   │   ├── history/
│   │   └── explorer/
│   ├── components/
│   │   ├── ui/             # shadcn/ui 覆寫元件（深色主題）
│   │   ├── amount/         # 金額格式化、等寬數字、+/− 前綴＋箭頭圖示的共用元件
│   │   ├── hash/           # hash 縮寫＋複製元件（Overview/History/Explorer 共用）
│   │   ├── error-state/    # 通用錯誤卡片＋Retry
│   │   ├── empty-state/    # 通用空狀態
│   │   └── skeleton/       # 骨架屏元件
│   ├── lib/
│   │   ├── currency.ts     # 依 GET /currencies 的 decimals 做金額補零/驗證的共用邏輯
│   │   └── csv.ts          # History Export 用的 CSV 組裝
│   └── mocks/               # MSW handlers，依 openapi.yaml 的 schema 對齊
├── tests/
│   ├── unit/
│   └── e2e/
└── README.md
```

`web/` 只透過 HTTP 呼叫後端；本機開發時後端需設定 `CORS_ALLOWED_ORIGINS`（例如
`http://localhost:5173`）才能讓瀏覽器放行跨來源請求（見 `router/router.go`，未設定時 CORS
中介層完全不掛載）——這是啟動前端開發環境時要記得的後端設定，不是前端程式碼能處理的事。

---

## 7. 已確認的決定

第一版規格（本文件較早的版本）在以下 8 個地方做了判斷、列為「你應該確認」的項目。這些已經全部
討論並確認，結論記錄在這裡；本文件第 1–6 節已經照著這些結論寫，不再用「如果你希望…我會改」的
問句語氣。

1. **註冊後自動登入** — 採用。後端 `POST /users` 不回傳 token，前端在註冊成功後自動用同一組帳密
   呼叫 `POST /auth/login`，成功直接進 `/`（見 §3.1）。
2. **Transfer 成功後的 hash** — 不採用「送出後立刻反查最新一筆交易」的臨時拼湊方式，改為後端直接
   回傳交易物件。`POST /wallet/transfer` 已經改為成功時回傳完整的 `TransactionResponse`（含
   `hash`），前端直接用回應裡的值，不需要額外呼叫任何 API（見 §3.3，對應後端變更：
   `docs/BACKEND_PREP_PLAN.md` 第 1 批 1.1）。
3. **`WALLET_NOT_FOUND`／`INVALID_AMOUNT` 這兩個 code 各自代表多種情況** — 拆細，不靠 `message`
   原文區分。後端已經拆成 `SENDER_WALLET_NOT_FOUND`/`RECIPIENT_NOT_FOUND` 與
   `AMOUNT_NOT_POSITIVE`/`INVALID_DECIMALS`/`AMOUNT_EXCEEDS_LIMIT`，前端純粹依 `code` 分流（見
   §4，對應後端變更：`docs/BACKEND_PREP_PLAN.md` 第 1 批 1.2/1.3）。
4. **密碼複雜度的視覺化提示** — 移除弱／中／強分級，只保留「At least 8 characters」的規則勾選
   （見 §3.1）。後端目前確實只驗證這一條規則，分級顯示會暗示不存在的複雜度要求，不採用。
5. **Explorer 對未登入使用者也開放，且用不同（無導覽列）版面** — 採用（見 §2.1/§2.4）。
6. **History 的 Export** — 匯出全部分頁，但加上筆數上限保護：`total <= 1000` 才真的把所有分頁拉完
   匯出，超過就提示使用者縮小範圍、不再嘗試全量拉取（見 §3.4）。
7. **24 小時統計** — 改由後端新增 `GET /wallet/{user_id}/stats` API 計算，不用前端從交易紀錄估算
   （見 §3.2，對應後端變更：`docs/BACKEND_PREP_PLAN.md` 第 2 批 2.1）。原本「資料筆數超過分頁上限時
   只統計近期部分並加免責文字」的限制因此不再適用——新 API 是資料庫聚合查詢，不受分頁上限影響。
8. **Token 儲存** — 改存 `localStorage`，保留 §2.3 的過期檢查與 401 全域處理邏輯不變（見 §2.2）。
