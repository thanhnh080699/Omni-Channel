import http from "node:http";
import { URL } from "node:url";
import { WhatsAppBaileysClient } from "./baileysClient.js";
import type { OutboundPayload } from "./contracts.js";
import { RabbitInboundPublisher } from "./rabbit.js";
import { SessionStore } from "./sessionStore.js";

const port = Number(process.env.WHATSAPP_ADAPTER_PORT || "19090");
const rabbitURL = process.env.RABBITMQ_URL || "amqp://guest:guest@localhost:5672/";
const sessionsDir = process.env.WHATSAPP_SESSION_DIR || "sessions";
const apiBaseURL = process.env.API_BASE_URL || "http://localhost:18080";
const webhookSharedSecret = process.env.WEBHOOK_SHARED_SECRET || "";

const sessions = new SessionStore();
const publisher = new RabbitInboundPublisher(rabbitURL);
const client = new WhatsAppBaileysClient(sessions, publisher, sessionsDir, apiBaseURL, webhookSharedSecret);

await publisher.connect();

const server = http.createServer(async (req, res) => {
  try {
    const url = new URL(req.url || "/", `http://${req.headers.host}`);
    if (req.method === "GET" && url.pathname === "/health") {
      return json(res, 200, { status: "UP", service: "whatsapp-adapter" });
    }
    if (req.method === "POST" && url.pathname.startsWith("/connect/")) {
      const accountId = decodeURIComponent(url.pathname.split("/")[2] || "");
      const snapshot = await client.connect(accountId);
      return json(res, 202, snapshot);
    }
    if (req.method === "GET" && url.pathname.startsWith("/session/")) {
      const accountId = decodeURIComponent(url.pathname.split("/")[2] || "");
      return json(res, 200, sessions.snapshot(accountId));
    }
    if (req.method === "GET" && url.pathname.startsWith("/typing/")) {
      const [, , accountId = "", ...jidParts] = url.pathname.split("/");
      return json(res, 200, await client.typing(decodeURIComponent(accountId), decodeURIComponent(jidParts.join("/"))));
    }
    if (req.method === "GET" && url.pathname.startsWith("/on-whatsapp/")) {
      const [, , accountId = "", phone = ""] = url.pathname.split("/");
      const exists = await client.onWhatsApp(decodeURIComponent(accountId), decodeURIComponent(phone));
      return json(res, 200, { exists });
    }
    if (req.method === "GET" && url.pathname.startsWith("/avatar/")) {
      const [, , accountId = "", ...jidParts] = url.pathname.split("/");
      const urlStr = await client.profilePictureUrl(decodeURIComponent(accountId), decodeURIComponent(jidParts.join("/")));
      return json(res, 200, { url: urlStr });
    }
    if (req.method === "POST" && url.pathname.startsWith("/disconnect/")) {
      const accountId = decodeURIComponent(url.pathname.split("/")[2] || "");
      await client.disconnect(accountId);
      return json(res, 202, sessions.snapshot(accountId));
    }
    if (req.method === "POST" && url.pathname.startsWith("/reset-session/")) {
      const accountId = decodeURIComponent(url.pathname.split("/")[2] || "");
      await client.resetSession(accountId);
      return json(res, 202, sessions.snapshot(accountId));
    }
    if (req.method === "POST" && url.pathname === "/send") {
      const payload = await readJSON<OutboundPayload>(req);
      const result = await client.send(payload);
      return json(res, 202, { status: "sent", channel_message_id: result?.key.id || "" });
    }
    if (req.method === "POST" && url.pathname === "/resync") {
      await client.resync(url.searchParams.get("account_id") || undefined);
      return json(res, 202, { status: "resynced" });
    }
    return json(res, 404, { error: "not found" });
  } catch (error) {
    const message = error instanceof Error ? error.message : "request failed";
    return json(res, 500, { error: message });
  }
});

server.listen(port, () => {
  console.log(`whatsapp-adapter listening on :${port}`);
});

async function readJSON<T>(req: http.IncomingMessage): Promise<T> {
  const chunks: Buffer[] = [];
  for await (const chunk of req) {
    chunks.push(Buffer.isBuffer(chunk) ? chunk : Buffer.from(chunk));
  }
  return JSON.parse(Buffer.concat(chunks).toString("utf8")) as T;
}

function json(res: http.ServerResponse, status: number, payload: unknown) {
  res.statusCode = status;
  res.setHeader("Content-Type", "application/json");
  res.end(JSON.stringify(payload));
}
