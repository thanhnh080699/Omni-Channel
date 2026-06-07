import amqp from "amqplib";
import type { InboundEnvelope } from "./contracts.js";

const INBOUND_EXCHANGE = "inbound.exchange";
const INBOUND_ROUTING_KEY = "inbound.dispatcher";

export class RabbitInboundPublisher {
  private connection?: Awaited<ReturnType<typeof amqp.connect>>;
  private channel?: Awaited<ReturnType<Awaited<ReturnType<typeof amqp.connect>>["createChannel"]>>;

  constructor(private readonly url: string) {}

  async connect() {
    this.connection = await amqp.connect(this.url);
    this.channel = await this.connection.createChannel();
    await this.channel.assertExchange(INBOUND_EXCHANGE, "direct", { durable: true });
  }

  async publish(envelope: InboundEnvelope) {
    if (!this.channel) throw new Error("rabbit channel is not connected");
    this.channel.publish(INBOUND_EXCHANGE, INBOUND_ROUTING_KEY, Buffer.from(JSON.stringify(envelope)), {
      contentType: "application/json",
      persistent: true,
      timestamp: Date.now(),
    });
  }

  async close() {
    await this.channel?.close();
    await this.connection?.close();
  }
}
