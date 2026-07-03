import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { groupsApi } from "@shared/api/endpoints";
import type { Group } from "@shared/api/types";

export const groupKeys = { list: ["groups"] as const };

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

export function useGroupMembers(groupId: string | null) {
  return useQuery({
    queryKey: ["group-members", groupId],
    enabled: !!groupId,
    queryFn: async () => (await groupsApi.members(groupId as string)).members,
  });
}

export function useAddMember() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (args: { groupId: string; userId: string }) =>
      groupsApi.addMember(args.groupId, args.userId),
    onSuccess: (_res, args) => qc.invalidateQueries({ queryKey: ["group-members", args.groupId] }),
  });
}
