FROM golang:1.24-bullseye

WORKDIR /app

# 複製 go.mod / go.sum 並下載依賴
COPY go.mod go.sum ./
RUN go mod download

# 複製專案全部檔案
COPY . .

# 編譯執行檔，保留 CGO 支援以支援 SQLite
RUN go build -o wallet-api ./main

EXPOSE 8080

CMD ["./wallet-api"]
