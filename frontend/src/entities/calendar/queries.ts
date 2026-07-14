import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { calendarApi } from "@shared/api/endpoints";
import type { CalendarColor } from "@shared/api/types";

export const calendarKeys = {
  month: (groupId: string, fromISO: string, toISO: string) => ["calendar", groupId, fromISO, toISO] as const,
  group: (groupId: string) => ["calendar", groupId] as const,
};

export function useCalendarMonth(groupId: string, fromISO: string, toISO: string) {
  return useQuery({
    queryKey: calendarKeys.month(groupId, fromISO, toISO),
    queryFn: async () => calendarApi.list(groupId, fromISO, toISO),
  });
}

type EventBody = { title: string; startsAt: string; endsAt?: string | null; color: CalendarColor };

export function useCreateEvent(groupId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (body: EventBody) => calendarApi.create(groupId, body),
    onSuccess: () => qc.invalidateQueries({ queryKey: calendarKeys.group(groupId) }),
  });
}

export function useUpdateEvent(groupId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (args: { eventId: string; body: EventBody }) => calendarApi.update(args.eventId, args.body),
    onSuccess: () => qc.invalidateQueries({ queryKey: calendarKeys.group(groupId) }),
  });
}

export function useDeleteEvent(groupId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (eventId: string) => calendarApi.remove(eventId),
    onSuccess: () => qc.invalidateQueries({ queryKey: calendarKeys.group(groupId) }),
  });
}
