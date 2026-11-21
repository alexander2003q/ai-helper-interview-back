# Backend

HTTP API для создания опросника технического интервью. В режиме `gigachat` теоретические вопросы генерируются по темам, описанию вакансии и уровню кандидата; для локальной разработки без ключа доступен режим `mock`.

## Запуск через Docker

Из корня проекта:

```bash
docker compose -f backend/docker-compose.yml up --build
```

API будет доступен на `http://localhost:8080`. Без дополнительной настройки сервер использует встроенные вопросы (`AI_PROVIDER=mock`).

## Подключение GigaChat

Создайте локальный файл окружения из примера:

```bash
cp backend/.env.example backend/.env
```

Вставьте в `backend/.env` новый Authorization Key из настроек проекта GigaChat API:

```env
AI_PROVIDER=gigachat
GIGACHAT_AUTHORIZATION_KEY=your_authorization_key
GIGACHAT_MODEL=GigaChat-2
GIGACHAT_SCOPE=GIGACHAT_API_PERS
```

Файл `.env` исключён из Git и Docker build context. Compose подключает его автоматически. Запуск:

```bash
docker compose -f backend/docker-compose.yml up --build
```

Не публикуйте вывод `docker compose config`: Compose подставляет в него значения из `.env`, включая секреты.

Backend сам обменивает Authorization Key на access token и обновляет его перед истечением срока действия. Сертификаты НУЦ Минцифры добавляются в доверенное хранилище Docker-образа во время сборки.

## Интерактивная документация API

В Docker Compose добавлен Swagger UI. После запуска откройте:

```text
http://localhost:8081
```

В документации описаны формат запроса, успешный ответ и все ошибки. Нажмите **Try it out**, измените пример JSON и затем нажмите **Execute**, чтобы отправить реальный запрос в backend. Swagger проксирует `/api` во внутренний API-контейнер, поэтому браузеру не требуется обращаться к отдельному порту или проходить CORS-проверку.

При использовании GigaChat ответ обычно требует несколько секунд. Для генерации крупных опросников HTTP-сервер ожидает ответ до 90 секунд.

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

Принимает настройки с первой страницы и возвращает готовый опросник. Сейчас генерируется только секция `theory`, а её размер берётся из `trainingConfig.theory.count` (от 1 до 20). Поля `code`, `experience` и `hints` пока принимаются для совместимости, но не отправляются в AI.

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
      "experience": {"enabled": true, "count": 3},
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
│   ├── provider/gigachat/           # OAuth и адаптер GigaChat API
│   ├── provider/memory/             # встроенный генератор для разработки
│   └── delivery/httpapi/            # HTTP handler, DTO и middleware
├── Dockerfile
└── docker-compose.yml
```

Зависимости направлены внутрь: HTTP-слой и AI-провайдеры зависят от domain/usecase, а usecase работает только через интерфейс из `ports`. GigaChat получает строгую JSON-схему ответа; полученные данные дополнительно проверяются перед передачей клиенту. На `429` и `5xx` выполняются повторные запросы, при `401` access token обновляется один раз. Наружу ошибки провайдера возвращаются как `502` без раскрытия ключей или внутренних ответов.

## Проверки

Тесты автоматически выполняются при сборке Docker-образа:

```bash
docker compose -f backend/docker-compose.yml build
```
