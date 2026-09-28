// Query keys for the report queue, in their own file so the tab and its test
// can share them without importing the component.
export const reportsKeys = {
  queue: (status: string) => ["admin", "reports", status] as const,
  counts: ["admin", "reports", "counts"] as const,
};
