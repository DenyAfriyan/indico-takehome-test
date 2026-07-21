# Stockroom Frontend

React dashboard for the inventory reservation API. It provides live stock polling, reservation creation, a five-minute expiration countdown, purchase confirmation, and user-friendly backend error handling.

## Run locally

Start the Go backend from the repository root:

```bash
make run
```

Then start the frontend in a second terminal:

```bash
cd frontend
pnpm install
pnpm dev
```

Open `http://localhost:3000`.

The frontend connects to `http://localhost:8080` by default. Override it when needed:

```bash
VITE_API_BASE_URL=http://localhost:8080 pnpm dev
```

## Validate

```bash
pnpm build
```
