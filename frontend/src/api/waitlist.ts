import { apiFetch } from "@/src/api/client";
import type { WaitlistEntry } from "@/src/types/api";

interface WaitlistResponse {
  entry: WaitlistEntry;
}

export function joinWaitlist(slotId: string): Promise<WaitlistResponse> {
  return apiFetch<WaitlistResponse>("/waitlist/join", {
    method: "POST",
    body: JSON.stringify({ slotId }),
  });
}

export function leaveWaitlist(waitlistId: string): Promise<WaitlistResponse> {
  return apiFetch<WaitlistResponse>(`/waitlist/${waitlistId}/leave`, {
    method: "POST",
  });
}
