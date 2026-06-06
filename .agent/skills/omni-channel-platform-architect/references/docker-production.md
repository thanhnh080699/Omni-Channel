# Docker Production

## Services

Use Docker Compose as a production-like baseline:

- `api`
- `frontend`
- `mongo`
- `rabbitmq`
- `redis`
- `worker-inbound`
- `worker-outbound`
- `worker-media`
- `cdn`
- `nginx` or another reverse proxy when needed

## Requirements

- Dedicated network.
- Volumes for MongoDB, RabbitMQ, Redis if persistence is desired, and CDN media folder.
- Healthchecks for stateful services and HTTP services.
- Restart policy.
- `.env.example` with placeholders only.
- No real secrets in compose or env examples.

## Environment Placeholders

Use names like:

```env
APP_ENV=production
API_PORT=8080
FRONTEND_PORT=3000
MONGO_URI=mongodb://mongo:27017/omni_channel
RABBITMQ_URL=amqp://omni:CHANGE_ME@rabbitmq:5672/
REDIS_ADDR=redis:6379
JWT_SECRET=CHANGE_ME_JWT_SECRET
CDN_BASE_URL=http://cdn:8081
CDN_API_KEY=CHANGE_ME_CDN_API_KEY
```

## Operational Notes

- API and workers should wait for MongoDB, RabbitMQ, and Redis health.
- Workers should be horizontally scalable if idempotency and locks are correct.
- Nginx can route `/api`, `/webhooks`, `/socket`, and frontend paths.
- Store channel credentials in a secret manager or encrypted configuration, not raw MongoDB/plain env when possible.

## Compose Example

Use `scripts/docker-compose.example.yml` as the canonical example file for this skill.
