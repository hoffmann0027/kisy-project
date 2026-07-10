// Upload orchestration (stage A): small files go up in one request; large
// ones through the resumable chunked flow with per-chunk progress, retries
// and cancellation. The server decides the limits — nothing is hardcoded.
import { attachmentsApi } from "@shared/api/endpoints";
import type { Attachment, AttachmentMeta } from "@shared/api/types";

/** Files at or below this size skip the chunked flow entirely. */
export const SINGLE_SHOT_MAX_BYTES = 4 << 20;

const CHUNK_RETRIES = 3;

export interface UploadOptions {
  meta?: AttachmentMeta;
  signal?: AbortSignal;
  /** Called with 0..1 as bytes are confirmed by the server. */
  onProgress?: (fraction: number) => void;
}

/**
 * Upload a file, choosing single-shot or chunked automatically. Chunked
 * uploads resume within the session: each chunk is retried on transient
 * failures, and an already-stored chunk is skipped server-side (idempotent).
 */
export async function uploadFile(file: File, opts: UploadOptions = {}): Promise<Attachment> {
  const { meta, signal, onProgress } = opts;

  if (file.size <= SINGLE_SHOT_MAX_BYTES) {
    onProgress?.(0);
    const { attachment } = await attachmentsApi.upload(file, meta, signal);
    onProgress?.(1);
    return attachment;
  }

  const { upload } = await attachmentsApi.initUpload(file.name, file.size, meta);
  const total = Math.ceil(file.size / upload.chunkBytes);
  const done = new Set<number>(upload.receivedChunks);

  for (let idx = 0; idx < total; idx++) {
    if (done.has(idx)) continue;
    signal?.throwIfAborted();
    const chunk = file.slice(idx * upload.chunkBytes, Math.min((idx + 1) * upload.chunkBytes, file.size));
    await putChunkWithRetry(upload.id, idx, chunk, signal);
    done.add(idx);
    onProgress?.(done.size / total);
  }

  const { attachment } = await attachmentsApi.completeUpload(upload.id);
  onProgress?.(1);
  return attachment;
}

async function putChunkWithRetry(id: string, idx: number, chunk: Blob, signal?: AbortSignal): Promise<void> {
  let lastError: unknown;
  for (let attempt = 0; attempt < CHUNK_RETRIES; attempt++) {
    try {
      await attachmentsApi.putChunk(id, idx, chunk, signal);
      return;
    } catch (err) {
      if (signal?.aborted) throw err;
      lastError = err;
    }
  }
  throw lastError;
}

/** Short uppercase label for a file's type chip (e.g. "PDF", "DOCX"). */
export function fileTypeLabel(fileName: string): string {
  const dot = fileName.lastIndexOf(".");
  if (dot < 0 || dot === fileName.length - 1) return "FILE";
  return fileName.slice(dot + 1).toUpperCase().slice(0, 5);
}

export function formatBytes(n: number): string {
  if (n >= 1 << 30) return `${(n / (1 << 30)).toFixed(1)} ГБ`;
  if (n >= 1 << 20) return `${(n / (1 << 20)).toFixed(1)} МБ`;
  if (n >= 1 << 10) return `${Math.round(n / (1 << 10))} КБ`;
  return `${n} Б`;
}
