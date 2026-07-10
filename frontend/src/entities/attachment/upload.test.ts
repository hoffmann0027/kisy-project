// @vitest-environment node
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { Attachment } from "@shared/api/types";

const served: Attachment = {
  id: "att-1",
  fileName: "big.bin",
  mimeType: "application/octet-stream",
  sizeBytes: 25,
  isImage: false,
  url: "/api/v1/attachments/att-1",
  kind: "file",
};

const api = vi.hoisted(() => ({
  upload: vi.fn(),
  initUpload: vi.fn(),
  putChunk: vi.fn(),
  completeUpload: vi.fn(),
}));

vi.mock("@shared/api/endpoints", () => ({ attachmentsApi: api }));

import { uploadFile, fileTypeLabel, SINGLE_SHOT_MAX_BYTES } from "./upload";

// Node 20+ File works in the node test environment.
function makeFile(size: number): File {
  return new File([new Uint8Array(size)], "big.bin");
}

describe("uploadFile", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    api.upload.mockImplementation(async () => ({ attachment: served }));
    api.completeUpload.mockImplementation(async () => ({ attachment: served }));
    api.initUpload.mockImplementation(async (_name: string, size: number) => ({
      upload: { id: "sess-1", chunkBytes: 10, declaredBytes: size, expiresAt: "", receivedChunks: [] },
    }));
    api.putChunk.mockImplementation(async (_id: string, index: number) => ({ stored: true, index }));
  });

  it("small files go single-shot", async () => {
    await uploadFile(makeFile(100));
    expect(api.upload).toHaveBeenCalledOnce();
    expect(api.initUpload).not.toHaveBeenCalled();
  });

  it("large files are chunked with progress", async () => {
    const progress: number[] = [];
    const file = makeFile(SINGLE_SHOT_MAX_BYTES + 5); // chunkBytes=10 from mock
    await uploadFile(file, { onProgress: (f) => progress.push(f) });

    expect(api.initUpload).toHaveBeenCalledOnce();
    const totalChunks = Math.ceil(file.size / 10);
    expect(api.putChunk).toHaveBeenCalledTimes(totalChunks);
    // Chunk indexes are sequential and complete.
    const indexes = api.putChunk.mock.calls.map((c) => c[1]);
    expect(indexes).toEqual([...Array(totalChunks).keys()]);
    expect(api.completeUpload).toHaveBeenCalledOnce();
    expect(progress.at(-1)).toBe(1);
  });

  it("resumes: chunks the server already has are skipped", async () => {
    api.initUpload.mockImplementationOnce(async (_name: string, size: number) => ({
      upload: { id: "sess-1", chunkBytes: 10, declaredBytes: size, expiresAt: "", receivedChunks: [0, 1] },
    }));
    const file = makeFile(SINGLE_SHOT_MAX_BYTES + 5);
    await uploadFile(file);
    const indexes = api.putChunk.mock.calls.map((c) => c[1]);
    expect(indexes).not.toContain(0);
    expect(indexes).not.toContain(1);
    expect(indexes[0]).toBe(2);
  });

  it("retries a transiently failing chunk", async () => {
    let failures = 2;
    api.putChunk.mockImplementation(async (_id: string, index: number) => {
      if (index === 1 && failures-- > 0) throw new Error("network blip");
      return { stored: true, index };
    });
    await uploadFile(makeFile(SINGLE_SHOT_MAX_BYTES + 5));
    expect(api.completeUpload).toHaveBeenCalledOnce();
  });

  it("gives up after repeated chunk failures", async () => {
    api.putChunk.mockImplementation(async () => {
      throw new Error("down");
    });
    await expect(uploadFile(makeFile(SINGLE_SHOT_MAX_BYTES + 5))).rejects.toThrow("down");
    expect(api.completeUpload).not.toHaveBeenCalled();
  });
});

describe("fileTypeLabel", () => {
  it("labels by extension, capped and uppercased", () => {
    expect(fileTypeLabel("report.pdf")).toBe("PDF");
    expect(fileTypeLabel("deck.pptx")).toBe("PPTX");
    expect(fileTypeLabel("noext")).toBe("FILE");
    expect(fileTypeLabel("trailing.")).toBe("FILE");
  });
});
