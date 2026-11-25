# Backend

HTTP API для создания опросника технического интервью.

## Запуск через Docker

Из корня проекта:

```bash
docker compose -f backend/docker-compose.yml up --build
```

API будет доступен на `http://localhost:8080`. Сервер использует встроенный генератор вопросов.

## Интерактивная документация API

В Docker Compose добавлен Swagger UI. После запуска откройте:

```text
http://localhost:8081
```

В документации описаны формат запроса, успешный ответ и все ошибки. Нажмите **Try it out**, измените пример JSON и затем нажмите **Execute**, чтобы отправить реальный запрос в backend. Swagger проксирует `/api` во внутренний API-контейнер, поэтому браузеру не требуется обращаться к отдельному порту или проходить CORS-проверку.

OpenAPI-описание хранится в `api/openapi.yaml`. Если нужно изменить порт документации, одновременно добавьте новый origin в CORS:

```bash
SWAGGER_PORT=18081 \
CORS_ALLOWED_ORIGINS=http://localhost:5173,http://localhost:18081 \
docker compose -f backend/docker-compose.yml up --build
```

Если порт `8080` занят, его можно переопределить:

```bash
BACKEND_PORT=18080 docker compose -f backend/docker-compose.yml up --build
```

Остановка:

```bash
docker compose -f backend/docker-compose.yml down
```

## API

### `POST /api/v1/questionnaires`

Принимает настройки с первой страницы и возвращает моковую тренировку. Количество вопросов берётся из `trainingConfig.theory`, количество задач — из `trainingConfig.code`. Число подсказок у задачи равно `hints.perTask`, а `attemptsPerTask` возвращается вместе с опросником. Хотя бы одна из секций должна быть включена и иметь count больше нуля.

```bash
curl -X POST http://localhost:8080/api/v1/questionnaires \
  -H 'Content-Type: application/json' \
  -d '{
    "topics": ["React"],
    "vacancy": "React, TypeScript, REST API",
    "level": "middle",
    "responseFormat": "text",
    "trainingConfig": {
      "theory": {"enabled": true, "count": 8},
      "code": {"enabled": true, "count": 2},
      "hints": {"enabled": true, "perTask": 2},
      "attemptsPerTask": 3
    }
  }'
```

Пока `correctAnswerId` и `explanation` возвращаются вместе с вопросом, поскольку фронтенд работает полностью локально. При подключении реального прохождения их следует перенести в отдельный endpoint проверки ответа.

## Архитектура

```text
backend/
├── cmd/api/                         # composition root и запуск сервера
├── internal/
│   ├── config/                      # конфигурация окружения
│   ├── domain/                      # бизнес-модели и валидация
│   ├── ports/                       # интерфейс генератора вопросов
│   ├── usecase/                     # сценарии приложения
│   ├── provider/memory/             # встроенный генератор для разработки
│   └── delivery/httpapi/            # HTTP handler, DTO и middleware
├── Dockerfile
└── docker-compose.yml
```

Зависимости направлены внутрь: HTTP-слой и генератор вопросов зависят от domain/usecase, а usecase работает только через интерфейс из `ports`.

## Проверки

Тесты автоматически выполняются при сборке Docker-образа:

```bash
docker compose -f backend/docker-compose.yml build
```
