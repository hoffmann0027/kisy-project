import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { notesApi } from "@shared/api/endpoints";
import type { Note } from "@shared/api/types";

export const noteKeys = { list: ["notes"] as const };

// useNotes loads the current user's personal notes (newest first). Only
// fetched while the notes panel is open.
export function useNotes(enabled: boolean) {
  return useQuery<Note[]>({
    queryKey: noteKeys.list,
    enabled,
    queryFn: async () => (await notesApi.list()).notes,
  });
}

export function useNoteMutations() {
  const qc = useQueryClient();
  const refresh = () => qc.invalidateQueries({ queryKey: noteKeys.list });

  return {
    createText: useMutation({ mutationFn: (text: string) => notesApi.createText(text), onSuccess: refresh }),
    createFile: useMutation({
      mutationFn: (a: { file: File; text?: string }) => notesApi.createFile(a.file, a.text),
      onSuccess: refresh,
    }),
    del: useMutation({ mutationFn: (id: string) => notesApi.del(id), onSuccess: refresh }),
  };
}
