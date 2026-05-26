"use client";

import { useEffect } from "react";

import { useRealtime } from "@/src/ws/RealtimeProvider";

export function ToastStack() {
  const { toasts, removeToast } = useRealtime();

  useEffect(() => {
    if (toasts.length === 0) {
      return;
    }
    const timers = toasts.map((toast) =>
      window.setTimeout(() => {
        removeToast(toast.id);
      }, 6000),
    );
    return () => {
      timers.forEach((timer) => window.clearTimeout(timer));
    };
  }, [toasts, removeToast]);

  if (toasts.length === 0) {
    return null;
  }

  return (
    <div className="fixed right-4 top-4 z-50 flex w-80 flex-col gap-2">
      {toasts.map((toast) => (
        <div
          key={toast.id}
          className="rounded-lg border border-zinc-700 bg-zinc-900 px-3 py-2 text-sm text-zinc-100 shadow-lg"
        >
          <p>{toast.message}</p>
          <button
            type="button"
            className="mt-2 text-xs text-zinc-300 hover:text-white"
            onClick={() => removeToast(toast.id)}
          >
            Close
          </button>
        </div>
      ))}
    </div>
  );
}
