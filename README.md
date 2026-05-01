## PostgreSQL для проекта benches

## Установка

### macOS (Homebrew)
```bash
brew install postgresql
brew services start postgresql
```

### Установка PostGIS
```bash
# macOS
brew install postgis

# Ubuntu/Debian
sudo apt-get install postgresql-15-postgis-3
```

### Создание базы данных и пользователя
```bash
# Создаем пользователя postgres (если нет)
createuser -s postgres

# Создаем базу данных
createdb benches

# Включаем расширение PostGIS
psql -d benches -c "CREATE EXTENSION IF NOT EXISTS postgis;"
```

### Запуск миграций
```bash
# Применяем миграции для создания индексов и триггеров
psql -U postgres -d benches -f migrate.sql
```

## Переменные окружения

Создайте файл `.env`:
```
# База данных
PG_HOST=localhost
PG_PORT=5432
PG_USER=postgres
PG_PASSWORD=postgres
PG_DBNAME=benches

# Админ-панель
ADMIN_USERNAME=admin
ADMIN_PASSWORD=admin123
ADMIN_EMAIL=admin@example.com

# Порт сервера
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
Получить список **одобренных** скамеек (для публичной карты)

### POST /api/benches
Добавить новую скамейку (статус: pending)

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

## Админ-панель

Доступна по адресу: http://localhost:8080/admin/

### Функции:
- Выход на учётную запись (логин/пароль из `.env`)
- Просмотр списка ожидающих модерации скамеек
- Одобрение или отклонение заявок
- Автоматическое обновление списка

### API админки:
- `POST /api/admin/login` — вход
- `GET /api/admin/pending` — список ожидающих
- `POST /api/admin/bench/{id}/approve` — одобрить
- `POST /api/admin/bench/{id}/reject` — отклонить

## Docker (опционально)

```bash
docker run --name benches-db -e POSTGRES_PASSWORD=postgres -p 5432:5432 -d postgres:15
```

PORT=8080 go run main.go // back
python3 -m http.server 8000 // front
