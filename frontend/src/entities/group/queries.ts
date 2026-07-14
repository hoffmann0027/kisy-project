import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { groupsApi } from "@shared/api/endpoints";
import type { Group, GroupRole, JoinPolicy, PostPolicy } from "@shared/api/types";

export const groupKeys = {
  list: ["groups"] as const,
  directory: ["groups", "directory"] as const,
  members: (groupId: string) => ["group-members", groupId] as const,
  requests: (groupId: string) => ["group-requests", groupId] as const,
  viewer: (groupId: string) => ["group-viewer", groupId] as const,
};

export function useGroups() {
  return useQuery({
    queryKey: groupKeys.list,
    queryFn: async () => (await groupsApi.list()).groups,
  });
}

export function useCreateGroup() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (args: { name: string; minRoleLevel: number; description?: string }) =>
      groupsApi.create(args.name, args.minRoleLevel, args.description),
    onSuccess: ({ group }) => {
      qc.setQueryData<Group[]>(groupKeys.list, (prev) => (prev ? [group, ...prev] : [group]));
    },
  });
}

export function useUpdateGroupLevel() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (args: { groupId: string; minRoleLevel: number }) =>
      groupsApi.updateLevel(args.groupId, args.minRoleLevel),
    onSuccess: ({ group }) => {
      qc.setQueryData<Group[]>(groupKeys.list, (prev) =>
        prev?.map((g) => (g.id === group.id ? group : g)),
      );
    },
  });
}

export function useDeleteGroup() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (groupId: string) => groupsApi.remove(groupId),
    onSuccess: (_res, groupId) => {
      qc.setQueryData<Group[]>(groupKeys.list, (prev) => prev?.filter((g) => g.id !== groupId));
    },
  });
}

export function useGroupMembers(groupId: string | null) {
  return useQuery({
    queryKey: groupKeys.members(groupId ?? ""),
    enabled: !!groupId,
    queryFn: async () => (await groupsApi.members(groupId as string)).members,
  });
}

export function useAddMember() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (args: { groupId: string; userId: string }) =>
      groupsApi.addMember(args.groupId, args.userId),
    onSuccess: (_res, args) => qc.invalidateQueries({ queryKey: groupKeys.members(args.groupId) }),
  });
}

// --- Stage N: access settings, directory, join, requests, roles ---

/** The caller's own membership/role/post-right in a group. */
export function useGroupViewer(groupId: string | null) {
  return useQuery({
    queryKey: groupKeys.viewer(groupId ?? ""),
    enabled: !!groupId,
    queryFn: async () => (await groupsApi.viewer(groupId as string)).viewer,
  });
}

export function useUpdateGroupSettings() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (args: { groupId: string; joinPolicy: JoinPolicy; postPolicy: PostPolicy }) =>
      groupsApi.updateSettings(args.groupId, args.joinPolicy, args.postPolicy),
    onSuccess: ({ group }) => {
      qc.setQueryData<Group[]>(groupKeys.list, (prev) => prev?.map((g) => (g.id === group.id ? group : g)));
      qc.invalidateQueries({ queryKey: groupKeys.viewer(group.id) });
    },
  });
}

/** Groups the caller may join (cleared, not a member). */
export function useGroupDirectory(enabled: boolean) {
  return useQuery({
    queryKey: groupKeys.directory,
    enabled,
    queryFn: async () => (await groupsApi.directory()).groups,
  });
}

export function useJoinGroup() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (groupId: string) => groupsApi.join(groupId),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: groupKeys.directory });
      qc.invalidateQueries({ queryKey: groupKeys.list });
    },
  });
}

/** Pending join requests for a group (editor/owner/CEO only). */
export function useGroupRequests(groupId: string | null, enabled: boolean) {
  return useQuery({
    queryKey: groupKeys.requests(groupId ?? ""),
    enabled: enabled && !!groupId,
    queryFn: async () => (await groupsApi.listRequests(groupId as string)).requests,
  });
}

export function useDecideRequest() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (args: { groupId: string; userId: string; approve: boolean }) =>
      args.approve
        ? groupsApi.approveRequest(args.groupId, args.userId)
        : groupsApi.rejectRequest(args.groupId, args.userId),
    onSuccess: (_res, args) => {
      qc.invalidateQueries({ queryKey: groupKeys.requests(args.groupId) });
      qc.invalidateQueries({ queryKey: groupKeys.members(args.groupId) });
    },
  });
}

export function useSetMemberRole() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (args: { groupId: string; userId: string; role: GroupRole }) =>
      groupsApi.setMemberRole(args.groupId, args.userId, args.role),
    onSuccess: (_res, args) => qc.invalidateQueries({ queryKey: groupKeys.members(args.groupId) }),
  });
}
