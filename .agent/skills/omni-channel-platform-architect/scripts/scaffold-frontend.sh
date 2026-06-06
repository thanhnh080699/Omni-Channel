#!/usr/bin/env bash
set -euo pipefail

TARGET_DIR="${1:-frontend}"

mkdir -p \
  "$TARGET_DIR/src/app/login" \
  "$TARGET_DIR/src/app/dashboard" \
  "$TARGET_DIR/src/app/inbox/[conversationId]" \
  "$TARGET_DIR/src/app/admin/users" \
  "$TARGET_DIR/src/app/admin/roles" \
  "$TARGET_DIR/src/app/admin/teams" \
  "$TARGET_DIR/src/app/admin/channels" \
  "$TARGET_DIR/src/app/reports" \
  "$TARGET_DIR/src/components" \
  "$TARGET_DIR/src/features/inbox" \
  "$TARGET_DIR/src/features/admin" \
  "$TARGET_DIR/src/lib" \
  "$TARGET_DIR/src/store" \
  "$TARGET_DIR/src/types" \
  "$TARGET_DIR/src/utils"

if [ ! -f "$TARGET_DIR/package.json" ]; then
  cat > "$TARGET_DIR/package.json" <<'EOF'
{
  "name": "omni-channel-frontend",
  "private": true,
  "scripts": {
    "dev": "next dev",
    "build": "next build",
    "start": "next start",
    "lint": "next lint"
  },
  "dependencies": {
    "next": "latest",
    "react": "latest",
    "react-dom": "latest"
  },
  "devDependencies": {
    "@types/node": "latest",
    "@types/react": "latest",
    "@types/react-dom": "latest",
    "typescript": "latest"
  }
}
EOF
fi

if [ ! -f "$TARGET_DIR/src/types/chat.ts" ]; then
  cat > "$TARGET_DIR/src/types/chat.ts" <<'EOF'
export type MessageStatus =
  | "pending"
  | "sending"
  | "sent"
  | "delivered"
  | "read"
  | "failed"
  | "cancelled";

export type AttachmentStatus =
  | "pending"
  | "processing"
  | "ready"
  | "failed"
  | "expired";
EOF
fi

if [ ! -f "$TARGET_DIR/.env.example" ]; then
  cat > "$TARGET_DIR/.env.example" <<'EOF'
NEXT_PUBLIC_API_BASE_URL=http://localhost:8080
NEXT_PUBLIC_SOCKET_URL=ws://localhost:8080/socket
EOF
fi

echo "Frontend skeleton created at $TARGET_DIR"
