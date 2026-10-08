import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { announcementsApi } from "@shared/api/endpoints";
import type { SendAnnouncementInput } from "@shared/api/types";

// Announcements: notifications written by levels 1-3 (internal/announcements).
// The server decides who may send and to whom; these helpers only keep the
// screen from offering what it would refuse.

export const announcementKeys = { list: ["announcements"] as const };

/** Lowest level that may write announcements; 1 is the CEO. */
export const ANNOUNCE_MAX_LEVEL = 3;

export function canAnnounce(roleLevel: number | null | undefined): boolean {
  return roleLevel != null && roleLevel >= 1 && roleLevel <= ANNOUNCE_MAX_LEVEL;
}

/**
 * The levels an author may address: their own and every one below it. The
 * same rule as starting a private chat — never upwards.
 */
export function addressableLevels(roleLevel: number): number[] {
  const out: number[] = [];
  for (let l = roleLevel; l <= 10; l++) out.push(l);
  return out;
}

/** Whether an author may address one person, by that person's level. */
export function canAddress(authorLevel: number, targetLevel: number | null | undefined): boolean {
  return targetLevel == null || targetLevel === 0 || targetLevel >= authorLevel;
}

export function useAnnouncements(enabled: boolean) {
  return useQuery({
    queryKey: announcementKeys.list,
    queryFn: async () => (await announcementsApi.list()).announcements,
    enabled,
  });
}

export function useSendAnnouncement() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (input: SendAnnouncementInput) => announcementsApi.send(input),
    onSuccess: () => qc.invalidateQueries({ queryKey: announcementKeys.list }),
  });
}

export function useRevokeAnnouncement() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => announcementsApi.revoke(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: announcementKeys.list }),
  });
}
