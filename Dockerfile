FROM golang:1.24-alpine AS builder

WORKDIR /app

# Копируем файлы модулей
COPY go.mod go.sum ./
RUN go mod download

# Копируем весь проект
COPY . .

# Устанавливаем goose для миграций
RUN go install github.com/pressly/goose/v3/cmd/goose@latest

# Собираем приложение
RUN go build -o main .

FROM alpine:latest

WORKDIR /app

# Копируем бинарники и файлы
COPY --from=builder /go/bin/goose /usr/local/bin/goose
COPY --from=builder /app/main .
COPY --from=builder /app/sql ./sql
COPY --from=builder /app/.env ./

ENV PORT=8080
ENV DB_URL=postgres://user:password@db:5432/avito_backend?sslmode=disable

EXPOSE ${PORT}

CMD ["sh", "-c", "goose -dir ./sql/schema postgres \"$DB_URL\" up && ./main"]