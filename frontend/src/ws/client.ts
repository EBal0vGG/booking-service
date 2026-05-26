import type { RealtimeMessage } from "@/src/types/api";

interface WsClientParams {
  baseUrl: string;
  onEvent: (event: RealtimeMessage) => void;
  onStatusChange?: (status: WsConnectionStatus) => void;
}

export type WsConnectionStatus =
  | "idle"
  | "connecting"
  | "connected"
  | "reconnecting"
  | "disconnected";

interface SubscribeMessage {
  type: "subscribe" | "unsubscribe";
  roomId: string;
}

export class RealtimeWsClient {
  private readonly baseUrl: string;
  private readonly onEvent: (event: RealtimeMessage) => void;
  private readonly onStatusChange?: (status: WsConnectionStatus) => void;

  private socket: WebSocket | null = null;
  private token: string | null = null;
  private reconnectTimer: ReturnType<typeof setTimeout> | null = null;
  private activeRoomId: string | null = null;
  private shouldReconnect = false;

  public constructor(params: WsClientParams) {
    this.baseUrl = params.baseUrl;
    this.onEvent = params.onEvent;
    this.onStatusChange = params.onStatusChange;
  }

  public connect(token: string): void {
    this.token = token;
    this.shouldReconnect = true;
    this.open("connecting");
  }

  public disconnect(): void {
    this.shouldReconnect = false;
    this.clearReconnect();
    if (this.socket) {
      this.socket.close();
      this.socket = null;
    }
    this.onStatusChange?.("disconnected");
  }

  public setRoomSubscription(roomId: string | null): void {
    if (this.activeRoomId === roomId) {
      return;
    }

    const previous = this.activeRoomId;
    this.activeRoomId = roomId;

    if (!this.socket || this.socket.readyState !== WebSocket.OPEN) {
      return;
    }

    if (previous) {
      this.send({ type: "unsubscribe", roomId: previous });
    }
    if (roomId) {
      this.send({ type: "subscribe", roomId });
    }
  }

  private open(initialStatus: "connecting" | "reconnecting"): void {
    if (!this.token) {
      return;
    }
    if (this.socket && this.socket.readyState <= WebSocket.OPEN) {
      return;
    }

    this.onStatusChange?.(initialStatus);
    const wsUrl = new URL(this.baseUrl);
    wsUrl.searchParams.set("token", this.token);

    const socket = new WebSocket(wsUrl.toString());
    this.socket = socket;

    socket.onopen = () => {
      this.onStatusChange?.("connected");
      if (this.activeRoomId) {
        this.send({ type: "subscribe", roomId: this.activeRoomId });
      }
    };

    socket.onmessage = (event) => {
      try {
        const parsed = JSON.parse(event.data) as RealtimeMessage;
        this.onEvent(parsed);
      } catch {
        this.onEvent({ type: "error", message: "invalid ws message" });
      }
    };

    socket.onclose = () => {
      this.socket = null;
      if (!this.shouldReconnect) {
        this.onStatusChange?.("disconnected");
        return;
      }
      this.scheduleReconnect();
    };

    socket.onerror = () => {
      socket.close();
    };
  }

  private scheduleReconnect(): void {
    this.clearReconnect();
    this.onStatusChange?.("reconnecting");
    this.reconnectTimer = setTimeout(() => {
      this.open("reconnecting");
    }, 1500);
  }

  private clearReconnect(): void {
    if (this.reconnectTimer) {
      clearTimeout(this.reconnectTimer);
      this.reconnectTimer = null;
    }
  }

  private send(message: SubscribeMessage): void {
    if (!this.socket || this.socket.readyState !== WebSocket.OPEN) {
      return;
    }
    this.socket.send(JSON.stringify(message));
  }
}
