# Frontend MVP

Frontend для ручного тестирования backend-flow:

`auth -> rooms -> schedule -> slots -> booking -> waitlist -> reservation -> realtime`

## Local run

```bash
cp .env.example .env.local
npm install
npm run dev
```

Откройте [http://localhost:3000](http://localhost:3000).

## Environment

- `NEXT_PUBLIC_API_URL` (default `http://localhost:8080`)
- `NEXT_PUBLIC_WS_URL` (default `ws://localhost:8080/ws`)

## Main pages

- `/login`
- `/register`
- `/rooms`
- `/rooms/[roomId]`
- `/bookings`
- `/admin/bookings`

## Docker

```bash
docker compose up --build frontend
```
