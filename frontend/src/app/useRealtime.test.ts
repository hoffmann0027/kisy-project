import { QueryClient } from "@tanstack/react-query";
import { describe, expect, it } from "vitest";
import type { GroupViewer } from "@shared/api/types";
import { groupKeys } from "@entities/group/queries";
import { chatKeys } from "@entities/chat/queries";
import { refreshAfterGap } from "./useRealtime";

// Events sent while the socket was down are not replayed. A tablet that slept
// while its account joined a community kept the answer "not a member" and
// showed the community without its member list until the app was restarted.

describe("after a gap in the socket", () => {
  it("re-reads where the account stands in its groups", () => {
    const qc = new QueryClient();
    const stranger: GroupViewer = { member: false, role: "", canPost: false, canUseWorkspace: false };
    qc.setQueryData(groupKeys.viewer("c1"), stranger);
    qc.setQueryData(groupKeys.list, []);
    qc.setQueryData(chatKeys.list, []);

    refreshAfterGap(qc);

    expect(qc.getQueryState(groupKeys.viewer("c1"))?.isInvalidated).toBe(true);
    expect(qc.getQueryState(groupKeys.list)?.isInvalidated).toBe(true);
    expect(qc.getQueryState(chatKeys.list)?.isInvalidated).toBe(true);
  });
});
