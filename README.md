# PostgreSQL для проекта benches

## Установка

### macOS (Homebrew)
```bash
brew install postgresql
brew services start postgresql
```

### Создание базы данных и пользователя
```bash
# Создаем пользователя postgres (если нет)
createuser -s postgres

# Создаем базу данных
createdb benches

# Или через psql:
# psql -U postgres
# CREATE DATABASE benches;
```

## Переменные окружения

Создайте файл `.env`:
```
PG_HOST=localhost
PG_PORT=5432
PG_USER=postgres
PG_PASSWORD=postgres
PG_DBNAME=benches
PORT=8080
```

## Запуск бэкэнда

```bash
# Установка зависимостей
go mod tidy

# Запуск сервера
go run main.go
```

Сервер запустится на http://localhost:8080

## API

### GET /api/benches
Получить список всех скамеек

### POST /api/benches
Добавить новую скамейку

```json
{
  "name": "Скамейка у парка",
  "comment": "Тенистое место",
  "email": "user@example.com",
  "latitude": 55.7558,
  "longitude": 37.6173,
  "photo": "data:image/png;base64..."
}
```

## Docker (опционально)

```bash
docker run --name benches-db -e POSTGRES_PASSWORD=postgres -p 5432:5432 -d postgres:15
```
