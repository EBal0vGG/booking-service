"use client";

import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useRef,
  useState,
} from "react";

import { ApiError } from "@/src/api/client";
import {
  cancelReservation,
  confirmReservation,
  listMyActiveReservations,
} from "@/src/api/reservations";
import { useAuth } from "@/src/auth/AuthProvider";
import type { RealtimeMessage } from "@/src/types/api";
import { RealtimeWsClient, type WsConnectionStatus } from "@/src/ws/client";

const WS_URL = process.env.NEXT_PUBLIC_WS_URL ?? "ws://localhost:8080/ws";

interface Toast {
  id: string;
  message: string;
}

interface ActiveReservationPrompt {
  reservationId: string;
  roomId?: string;
  slotId: string;
  expiresAt: string;
}

interface RealtimeContextValue {
  status: WsConnectionStatus;
  lastEvent: RealtimeMessage | null;
  toasts: Toast[];
  activeReservation: ActiveReservationPrompt | null;
  setRoomSubscription: (roomId: string | null) => void;
  removeToast: (id: string) => void;
  dismissReservationPrompt: () => void;
  confirmActiveReservation: () => Promise<void>;
  cancelActiveReservation: () => Promise<void>;
  reloadActiveReservations: () => Promise<void>;
}

const RealtimeContext = createContext<RealtimeContextValue | undefined>(undefined);

function createToast(message: string): Toast {
  return {
    id: `${Date.now()}-${Math.random().toString(36).slice(2, 8)}`,
    message,
  };
}

function toMessage(error: unknown): string {
  if (error instanceof ApiError) {
    return `${error.code}: ${error.message}`;
  }
  if (error instanceof Error) {
    return error.message;
  }
  return "Unknown error";
}

export function RealtimeProvider({ children }: { children: React.ReactNode }) {
  const { session } = useAuth();
  const sessionUserID = session?.userId;
  const clientRef = useRef<RealtimeWsClient | null>(null);

  const [status, setStatus] = useState<WsConnectionStatus>("idle");
  const [lastEvent, setLastEvent] = useState<RealtimeMessage | null>(null);
  const [toasts, setToasts] = useState<Toast[]>([]);
  const [activeReservation, setActiveReservation] =
    useState<ActiveReservationPrompt | null>(null);

  const pushToast = useCallback((message: string) => {
    setToasts((current) => [...current, createToast(message)]);
  }, []);

  const loadActiveReservations = useCallback(async () => {
    if (!session?.token) {
      return;
    }
    try {
      const response = await listMyActiveReservations();
      const nearest = response.reservations[0];
      if (!nearest) {
        setActiveReservation(null);
        return;
      }
      setActiveReservation((current) => {
        if (current?.reservationId === nearest.id) {
          return current;
        }
        return {
          reservationId: nearest.id,
          slotId: nearest.slotId,
          expiresAt: nearest.expiresAt,
        };
      });
    } catch (error) {
      if (error instanceof ApiError && (error.status === 401 || error.status === 403)) {
        return;
      }
      pushToast(`Не удалось загрузить активные резервы: ${toMessage(error)}`);
    }
  }, [pushToast, session?.token]);

  const handleIncomingEvent = useCallback(
    (event: RealtimeMessage) => {
      setLastEvent(event);

      if (
        event.type === "waitlist_slot_reserved" &&
        sessionUserID &&
        event.userId === sessionUserID &&
        event.reservationId &&
        event.expiresAt
      ) {
        setActiveReservation({
          reservationId: event.reservationId,
          roomId: event.roomId,
          slotId: event.slotId ?? "",
          expiresAt: event.expiresAt,
        });
        pushToast("Для вас создан временный резерв. Подтвердите или отмените его.");
      }

      if (
        event.type === "reservation_expired" &&
        sessionUserID &&
        event.userId === sessionUserID
      ) {
        setActiveReservation((current) =>
          current?.reservationId === event.reservationId ? null : current,
        );
        pushToast("Резерв истек.");
      }

      if (event.type === "error" && event.message) {
        pushToast(`WS error: ${event.message}`);
      }
    },
    [pushToast, sessionUserID],
  );

  useEffect(() => {
    const client = new RealtimeWsClient({
      baseUrl: WS_URL,
      onEvent: handleIncomingEvent,
      onStatusChange: setStatus,
    });
    clientRef.current = client;

    return () => {
      client.disconnect();
      clientRef.current = null;
    };
  }, [handleIncomingEvent]);

  useEffect(() => {
    if (!clientRef.current) {
      return;
    }
    if (!session?.token) {
      const timeout = window.setTimeout(() => {
        setActiveReservation(null);
      }, 0);
      clientRef.current.disconnect();
      return () => window.clearTimeout(timeout);
    }
    clientRef.current.connect(session.token);
  }, [session?.token]);

  useEffect(() => {
    if (!session?.token) {
      return;
    }
    const timeout = window.setTimeout(() => {
      void loadActiveReservations();
    }, 0);
    return () => window.clearTimeout(timeout);
  }, [loadActiveReservations, session?.token]);

  const setRoomSubscription = useCallback((roomId: string | null) => {
    clientRef.current?.setRoomSubscription(roomId);
  }, []);

  const removeToast = useCallback((id: string) => {
    setToasts((current) => current.filter((item) => item.id !== id));
  }, []);

  const dismissReservationPrompt = useCallback(() => {
    setActiveReservation(null);
  }, []);

  const confirmActiveReservation = useCallback(async () => {
    if (!activeReservation) {
      return;
    }
    try {
      await confirmReservation(activeReservation.reservationId);
      pushToast("Резерв подтверждён, бронь создана.");
      if (activeReservation.roomId) {
        setLastEvent({
          type: "slot_booked",
          roomId: activeReservation.roomId,
          slotId: activeReservation.slotId,
          reservationId: activeReservation.reservationId,
          timestamp: new Date().toISOString(),
        });
      }
      setActiveReservation(null);
      await loadActiveReservations();
    } catch (error) {
      pushToast(`Не удалось подтвердить резерв: ${toMessage(error)}`);
    }
  }, [activeReservation, loadActiveReservations, pushToast]);

  const cancelActiveReservationAction = useCallback(async () => {
    if (!activeReservation) {
      return;
    }
    try {
      await cancelReservation(activeReservation.reservationId);
      pushToast("Резерв отменён.");
      if (activeReservation.roomId) {
        setLastEvent({
          type: "slot_available",
          roomId: activeReservation.roomId,
          slotId: activeReservation.slotId,
          reservationId: activeReservation.reservationId,
          timestamp: new Date().toISOString(),
        });
      }
      setActiveReservation(null);
      await loadActiveReservations();
    } catch (error) {
      pushToast(`Не удалось отменить резерв: ${toMessage(error)}`);
    }
  }, [activeReservation, loadActiveReservations, pushToast]);

  const value = useMemo<RealtimeContextValue>(
    () => ({
      status,
      lastEvent,
      toasts,
      activeReservation,
      setRoomSubscription,
      removeToast,
      dismissReservationPrompt,
      confirmActiveReservation,
      cancelActiveReservation: cancelActiveReservationAction,
      reloadActiveReservations: loadActiveReservations,
    }),
    [
      activeReservation,
      cancelActiveReservationAction,
      confirmActiveReservation,
      dismissReservationPrompt,
      loadActiveReservations,
      lastEvent,
      removeToast,
      setRoomSubscription,
      status,
      toasts,
    ],
  );

  return <RealtimeContext.Provider value={value}>{children}</RealtimeContext.Provider>;
}

export function useRealtime(): RealtimeContextValue {
  const value = useContext(RealtimeContext);
  if (!value) {
    throw new Error("useRealtime must be used inside RealtimeProvider");
  }
  return value;
}
