# Git 歷史清理分析（History Cleanup Analysis）

**這份文件只做分析與準備，沒有執行任何會改寫歷史的指令**（符合驗證計畫 V6
「只產生報告，不改寫歷史」的限制）。所有指令都需要使用者自行決定時機後手動執行。

**產出時間**：2026-09-26　**分析對象**：本機 repo 完整 git 歷史（含 `origin/main`
與 `fix/audit-remediation` 兩個分支能到達的所有 commit）

---

## 🔴 緊急：這不是「歷史清理」而已，是目前正在進行式的外洩

在開始分析歷史之前，必須先指出一個比「清理歷史」更急迫的事實：

- `git remote -v` 確認 `origin` 是 `git@github.com:marksue1107/mini-crypto-wallet-api.git`。
- 用 GitHub 公開 API 查詢（`curl https://api.github.com/repos/marksue1107/mini-crypto-wallet-api`）
  確認 `"private": false`——**這個 repo 現在是公開的**。
- `git show origin/main:api/config.yaml` 確認：**此時此刻**，`origin/main` 的
  最新一次 commit 樹裡，`api/config.yaml` 仍然存在，內容包含
  `postgres_dsn: ...password=secret...` 與
  `jwt_secret: your-secret-key-change-in-production-min-32-chars`。
- `git ls-tree -r origin/main` 確認 `wallet-api`（約 55MB 編譯後二進位檔）與
  `main/__debug_bin2424212485`（約 54MB，Delve debug 二進位檔）也仍然存在於
  `origin/main` 目前的樹裡。

**換句話說：這些密鑰與二進位檔不是「曾經外洩過、現在已經修好只是歷史還留著」，
而是「現在打開 GitHub 網頁還看得到」。** 本輪的 `fix/audit-remediation`
分支已經修好這些問題（見下方 commit 對照），但這些修正**還沒有 merge 回
`main` 也還沒 push**，所以對任何能看到這個公開 repo 的人來說，問題依然存在。

這件事跟「Go/No-Go 是否可以 merge」是兩個獨立的決策：
1. 不管要不要 merge 這個分支，`origin/main` 上現存的密鑰都應該視為已外洩，
   **及早輪替**（換掉真正在用的 Postgres 密碼、JWT 密鑰——即使目前這兩個值
   看起來像是佔位符字串，只要曾經或正在被任何真實環境使用，就要換掉）。
2. 若要徹底移除「歷史紀錄裡查得到」的痕跡（而不是只清掉未來的 commit），
   才需要用到下面的 `git filter-repo` 清理流程，而且清理完成後**必須
   force-push**，這件事的影響（見下方「注意事項」）需要使用者自行評估時機
   （例如是否有其他協作者已經 clone/fork 這個 repo）。

---

## 目前工作樹與 `main..fix/audit-remediation` 掃描結果

對 `git diff main..fix/audit-remediation`（本輪修正引入的所有變更）掃描
`password=`、`secret=`、`api_key`、私鑰標頭等樣式：**沒有發現新增的真實密鑰**。
比對到的字串全部屬於下列兩類，皆非外洩：

- 測試檔案裡的假密鑰（如 `"test-secret-at-least-32-characters-long"`），僅用於
  單元測試建構 `config.AppConfig`，不對應任何真實服務。
- 本輪修正**移除**的舊字串（`diff` 裡的 `-` 行），也就是下一節要清理的
  歷史內容本身。

目前工作樹（`git status`、`git ls-files`）沒有任何 `.env`、`*.pem`、`*.key`、
`id_rsa` 等敏感檔案被追蹤。`.env` 存在於磁碟但 `git check-ignore -v .env`
確認會被 `.gitignore` 規則擋下，不會被誤 commit。

---

## 完整 git 歷史掃描結果：需要移除的路徑

| 路徑 | 出現的 commit | 移除的 commit | 內容 |
|---|---|---|---|
| `config.yaml`（搬移前，repo 根目錄） | `eb7472e`（新增）、`bfc4da1`（修改） | `47b6974` 把它改名搬到 `api/config.yaml`（同一個 blob 歷史延續，不是刪除） | `postgres_dsn` 內含 `password=secret`；`bfc4da1` 之後開始含 `jwt_secret: your-secret-key-change-in-production-min-32-chars` |
| `api/config.yaml`（搬移後） | `47b6974`（改名承接）、`828c317`（修改，加了 `max_transfer_amount`） | `582da7e`（刪除，改用 `config.yaml.example`） | 同上，密碼與 JWT 密鑰內容延續到這裡 |
| `wallet-api`（repo 根目錄，編譯後二進位檔） | `2f62904`（新增，54.9MB） | `24d6879`（刪除） | 編譯產物，不含密鑰，但佔用大量歷史空間 |
| `main/__debug_bin2424212485`（Delve debug 二進位檔） | `2f62904`（新增，51.6MB） | `24d6879`（刪除） | 同上 |

掃描方法：`git log --all --full-history --diff-filter=A -- <path>` 找出每個
路徑第一次出現的 commit；`git rev-list --objects --all | git cat-file
--batch-check` 依 blob 大小排序找出所有歷史中曾出現過的大型物件（top 15
中只有上述兩個二進位檔明顯異常，其餘都是 `go.sum`／`docs/*.md` 等正常成長
的文字檔，不需清理）；`git log --all -p` 對全歷史掃描 AWS key
（`AKIA[0-9A-Z]{16}`）、私鑰標頭（`BEGIN (RSA|EC|OPENSSH|PGP) PRIVATE
KEY`）、憑證標頭等樣式，**沒有其他發現**。

**沒有遺漏**：以上 4 個路徑（對應 2 個 blob 歷史脈絡：config.yaml 搬移前後
算同一份內容的延續、2 個二進位檔）就是全部需要從歷史移除的目標。

---

## Repo 大小

- 目前 `.git` 目錄：**52 MiB**（`git count-objects -vH`：371 個尚未打包的
  loose object 共 51.93 MiB，加上一個很小的 pack）。這個數字之所以還這麼大，
  是因為 `wallet-api`／`__debug_bin*` 這兩個大型 blob 雖然已經從
  `fix/audit-remediation` 分支的最新 commit 裡刪除，但**仍然被
  `origin/main`（尚未清理）引用**，git 不會主動丟掉任何 ref 還能到達的物件。
- 預估清理後：扣掉這兩個二進位檔（約 105MB 未壓縮，實際 loose object 壓縮後
  約數十 MB），`.git` 應可降到 **1MB 以下量級**（`api/go.sum`、
  `docs/AUDIT_REMEDIATION_PLAN.md` 等歷史版本的文字檔本身占用很小）。實際
  數字需清理後用 `git count-objects -vH` 重新量測，且必須先在所有 clone
  端都執行過 `git gc --prune=now` 才會真的釋放磁碟空間。

---

## `git-filter-repo` 安裝狀態

`which git-filter-repo` 回報**未安裝**。依計畫規則**不自行安裝**，僅記錄：
安裝方式一般是 `pip install git-filter-repo` 或 `brew install git-filter-repo`
（macOS），由使用者自行決定是否安裝與何時安裝。

---

## 完整清理步驟（僅供參考，使用者自行決定時機後手動執行）

### 步驟 0：前置確認
```bash
# 確認沒有其他人正在這個 repo 上工作（clone/fork/PR），因為這個清理會改寫
# 所有 commit hash，任何已存在的 clone 之後 pull 都會衝突
gh repo list --json forks 2>/dev/null || echo "需自行到 GitHub 網頁確認 fork/clone 狀況"

# 確認 working tree 乾淨、沒有進行中的 worktree
git status
git worktree list
```

### 步驟 1：備份（務必先做，此操作不可逆）
```bash
# 完整備份整個 repo（含 .git），存放在 repo 之外的路徑
cp -R /Users/marksue/projects/mini-crypto-wallet-api /Users/marksue/projects/mini-crypto-wallet-api.backup-$(date +%Y%m%d)

# 或者：直接對 .git 做 bundle 備份，之後可用 git clone 復原任一分支/tag
git bundle create /Users/marksue/mini-crypto-wallet-api-full-backup-$(date +%Y%m%d).bundle --all
```

### 步驟 2：安裝並執行 git-filter-repo
```bash
pip install git-filter-repo   # 或 brew install git-filter-repo

cd /Users/marksue/projects/mini-crypto-wallet-api

# 先跑分析報告（唯讀，不改寫任何東西），確認上面列出的路徑清單無誤
git filter-repo --analyze
# 報告產生在 .git/filter-repo/analysis/，可打開 path-all-sizes.txt 交叉核對

# 正式移除歷史中的 4 個目標路徑（--invert-paths 表示「移除清單內的，其餘全留」）
git filter-repo --force \
  --path config.yaml \
  --path api/config.yaml \
  --path wallet-api \
  --path 'main/__debug_bin2424212485' \
  --invert-paths
```
> `filter-repo` 執行後會自動移除 `origin` remote（這是它的安全設計，避免
> 改寫完歷史後不小心對舊的 remote 做一般 push）。

### 步驟 3：驗證兩個分支都還能 build 與 test
```bash
cd /Users/marksue/projects/mini-crypto-wallet-api/api
go build ./...
go vet ./...
go test ./...
go test ./... -race

git checkout fix/audit-remediation
go build ./... && go vet ./... && go test ./... -race
git checkout main
go build ./... && go vet ./... && go test ./... -race
```

### 步驟 4：加回 remote
```bash
cd /Users/marksue/projects/mini-crypto-wallet-api
git remote add origin git@github.com:marksue1107/mini-crypto-wallet-api.git
```

### 步驟 5：推送（會改寫遠端歷史，務必確認團隊/協作者都已知會）
```bash
# 所有分支與 tag 的 hash 都已改變，一般 push 會被拒絕，必須強制推送
git push origin --force --all
git push origin --force --tags
```

### 步驟 6：清理後驗證
```bash
# 確認歷史中已經找不到這些檔案
git log --all --full-history --oneline -- config.yaml api/config.yaml wallet-api 'main/__debug_bin2424212485'
# 預期：完全沒有輸出

# 確認任何 commit 的任何版本都不再含有那組密鑰字串
git log --all -p | grep -c "your-secret-key-change-in-production-min-32-chars"
git log --all -p | grep -c "password=secret"
# 預期：兩者皆為 0

# 確認 repo 大小已下降
git gc --prune=now --aggressive
git count-objects -vH
du -sh .git
```

---

## 注意事項（風險與必要配套動作）

1. **所有 commit hash 都會改變**——包含 `main` 與 `fix/audit-remediation`
   兩個分支上的每一個 commit。任何已經 clone 這個 repo 的人（包含自己
   的其他機器）都需要重新 clone，不能用 `git pull`（會產生大量衝突/重複
   歷史）。
2. **必須強制推送**（`git push --force`）才能讓遠端反映清理後的歷史；
   在推送之前，遠端（GitHub）上舊的、含密鑰的 commit 依然完整存在且公開
   可見。
3. **必須在推送 `fix/audit-remediation` 分支或開 PR 之前執行**這個清理
   ——否則清理的意義就只剩下本機，遠端還是看得到。
4. GitHub 本身可能還保留這些 blob 的快取（例如透過 PR diff 頁面、
   `raw.githubusercontent.com` 連結、或第三方曾經 fork/clone 過），
   `git filter-repo` 只能清掉這個 repo 自己的 git 物件庫，清理完成後建議
   額外聯絡 GitHub Support 請求清除快取的物件（尤其是曾經公開過的密鑰）。
5. **不論是否執行這個歷史清理，都應該獨立進行「輪替」**：換掉任何跟
   `password=secret`、`your-secret-key-change-in-production-min-32-chars`
   相關的真實憑證（若這兩個值從未在任何真實環境使用過，風險本來就低，
   但無法從 git 歷史本身確認這件事，應以「已外洩」為前提處理）。
6. 這份文件本身、以及 `docs/VERIFICATION_REPORT.md`、`docs/AUDIT.md` 等
   會被一起清理（它們是清理**之後**才存在的檔案，不受影響），但清理過程
   一定要保留這些文件在清理後的樹狀結構中——`--invert-paths` 只精準排除
   上面 4 個路徑，不會動到其他任何檔案。
