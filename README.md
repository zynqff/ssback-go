# Сборник стихов — Go Backend

## Локальный запуск

```bash
cp .env.example .env   # заполнить переменные
go mod tidy
go run cmd/server/main.go
```

## Деплой на Render

1. Залить репозиторий на GitHub
2. На [render.com](https://render.com) → **New Web Service** → подключить репо
3. Render автоматически подхватит `render.yaml`
4. В дашборде Render → **Environment** → добавить переменные из `.env.example`:
   - `SUPABASE_URL`
   - `SUPABASE_KEY`
   - `GROQ_API_KEY`
   - `GOOGLE_CLIENT_ID`
   - `GOOGLE_CLIENT_SECRET`
   - `SECRET_KEY` — любая случайная строка
   - `ADMIN_USERNAMES` — через запятую: `admin1,admin2`
   - `ADMIN_PASSWORDS` — через запятую: `pass1,pass2`

5. Deploy — готово.

## Google OAuth (мобильное приложение)

После деплоя в Google Cloud Console → OAuth 2.0 → Android Client:
- Package name: `com.sscollective.app`
- SHA-1: дебаг-ключ из `keytool -list -v -keystore ~/.android/debug.keystore`

## API эндпоинты

| Метод | Путь | Описание |
|-------|------|----------|
| POST | `/api/login` | Вход |
| POST | `/api/register` | Регистрация |
| GET | `/api/logout` | Выход |
| GET | `/auth/google/login` | Google OAuth (веб) |
| POST | `/api/google/mobile-auth` | Google OAuth (Flutter) |
| GET | `/api/me` | Текущий пользователь |
| POST | `/api/profile` | Обновить профиль |
| GET | `/api/poems` | Список стихов |
| POST | `/api/poems` | Добавить стих (админ) |
| PUT | `/api/poems/{title}` | Редактировать стих (админ) |
| DELETE | `/api/poems/{title}` | Удалить стих (админ) |
| POST | `/api/toggle_read` | Отметить прочитанным |
| POST | `/api/toggle_pin` | Закрепить стих |
| POST | `/api/ai/chat` | Чат с AI |
| POST | `/api/ai/verify_key` | Проверить ключ AI |
| POST | `/api/ai/generate_key` | Создать ключ AI (админ) |
| GET | `/api/ai/keys` | Список ключей (админ) |
| POST | `/api/ai/disable_key` | Отключить ключ (админ) |
