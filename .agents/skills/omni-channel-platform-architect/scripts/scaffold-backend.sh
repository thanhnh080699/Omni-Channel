#!/usr/bin/env bash
set -euo pipefail

TARGET_DIR="${1:-backend}"
MODULE_NAME="${GO_MODULE_NAME:-omni-channel/backend}"

mkdir -p \
  "$TARGET_DIR/cmd/api" \
  "$TARGET_DIR/cmd/worker" \
  "$TARGET_DIR/internal/auth" \
  "$TARGET_DIR/internal/rbac" \
  "$TARGET_DIR/internal/channel" \
  "$TARGET_DIR/internal/conversation" \
  "$TARGET_DIR/internal/message" \
  "$TARGET_DIR/internal/queue" \
  "$TARGET_DIR/internal/socket" \
  "$TARGET_DIR/internal/media" \
  "$TARGET_DIR/internal/config" \
  "$TARGET_DIR/internal/database" \
  "$TARGET_DIR/pkg"

if [ ! -f "$TARGET_DIR/go.mod" ]; then
  cat > "$TARGET_DIR/go.mod" <<EOF
module $MODULE_NAME

go 1.22
EOF
fi

if [ ! -f "$TARGET_DIR/cmd/api/main.go" ]; then
  cat > "$TARGET_DIR/cmd/api/main.go" <<'EOF'
package main

import (
	"context"
	"fmt"
	"time"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_ = ctx
	fmt.Println("omni-channel api placeholder")
}
EOF
fi

if [ ! -f "$TARGET_DIR/cmd/worker/main.go" ]; then
  cat > "$TARGET_DIR/cmd/worker/main.go" <<'EOF'
package main

import "fmt"

func main() {
	fmt.Println("omni-channel worker placeholder")
}
EOF
fi

if [ ! -f "$TARGET_DIR/.env.example" ]; then
  cat > "$TARGET_DIR/.env.example" <<'EOF'
APP_ENV=local
API_PORT=8080
MONGO_URI=mongodb://localhost:27017/omni_channel
RABBITMQ_URL=amqp://guest:guest@localhost:5672/
REDIS_ADDR=localhost:6379
JWT_SECRET=CHANGE_ME_JWT_SECRET
CDN_BASE_URL=http://localhost:8081
CDN_API_KEY=CHANGE_ME_CDN_API_KEY
EOF
fi

echo "Backend skeleton created at $TARGET_DIR"
