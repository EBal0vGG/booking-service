import { ApiError } from "@/src/api/client";

export function formatUtcDateTime(value: string): string {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return value;
  }
  return date.toISOString().replace("T", " ").replace(".000Z", " UTC");
}

export function formatUtcTimeRange(start: string, end: string): string {
  const startDate = new Date(start);
  const endDate = new Date(end);
  if (Number.isNaN(startDate.getTime()) || Number.isNaN(endDate.getTime())) {
    return `${start} - ${end}`;
  }
  const startText = startDate.toISOString().slice(11, 16);
  const endText = endDate.toISOString().slice(11, 16);
  return `${startText} - ${endText} UTC`;
}

export function toUserErrorMessage(error: unknown): string {
  if (!(error instanceof ApiError)) {
    if (error instanceof Error) {
      return error.message;
    }
    return "Unknown error";
  }

  switch (error.code) {
    case "SLOT_ALREADY_BOOKED":
      return "Слот уже забронирован.";
    case "SLOT_RESERVED":
      return "Слот временно зарезервирован для пользователя из очереди.";
    case "INVALID_REQUEST":
      return error.message;
    case "FORBIDDEN":
      return "Недостаточно прав для операции.";
    case "SCHEDULE_EXISTS":
      return "Расписание уже создано.";
    case "WAITLIST_ALREADY_JOINED":
      return "Вы уже состоите в waitlist для этого слота.";
    default:
      return `${error.code}: ${error.message}`;
  }
}
