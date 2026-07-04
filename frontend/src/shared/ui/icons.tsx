import type { ReactNode } from "react";

// Minimal stroked icon set (currentColor), 24x24.
interface IconProps {
  size?: number;
}

function svg(path: ReactNode, size = 22) {
  return (
    <svg width={size} height={size} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
      {path}
    </svg>
  );
}

export const Icon = {
  Chat: ({ size }: IconProps) => svg(<path d="M21 11.5a8.38 8.38 0 0 1-8.5 8.5 8.5 8.5 0 0 1-3.8-.9L3 21l1.9-5.7A8.5 8.5 0 1 1 21 11.5Z" />, size),
  Bell: ({ size }: IconProps) => svg(<><path d="M18 8a6 6 0 0 0-12 0c0 7-3 9-3 9h18s-3-2-3-9" /><path d="M13.7 21a2 2 0 0 1-3.4 0" /></>, size),
  Shield: ({ size }: IconProps) => svg(<path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10Z" />, size),
  Logout: ({ size }: IconProps) => svg(<><path d="M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4" /><path d="m16 17 5-5-5-5" /><path d="M21 12H9" /></>, size),
  Send: ({ size }: IconProps) => svg(<path d="m22 2-7 20-4-9-9-4 20-7Z" />, size),
  Plus: ({ size }: IconProps) => svg(<><path d="M12 5v14" /><path d="M5 12h14" /></>, size),
  Search: ({ size }: IconProps) => svg(<><circle cx="11" cy="11" r="8" /><path d="m21 21-4.3-4.3" /></>, size),
  Reply: ({ size }: IconProps) => svg(<><path d="M9 17 4 12l5-5" /><path d="M20 18v-2a4 4 0 0 0-4-4H4" /></>, size),
  Trash: ({ size }: IconProps) => svg(<><path d="M3 6h18" /><path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2" /></>, size),
  Smile: ({ size }: IconProps) => svg(<><circle cx="12" cy="12" r="10" /><path d="M8 14s1.5 2 4 2 4-2 4-2" /><path d="M9 9h.01M15 9h.01" /></>, size),
  Settings: ({ size }: IconProps) => svg(<><circle cx="12" cy="12" r="3" /><path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 1 1-2.83 2.83l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-4 0v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 1 1-2.83-2.83l.06-.06a1.65 1.65 0 0 0 .33-1.82 1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1 0-4h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 1 1 2.83-2.83l.06.06a1.65 1.65 0 0 0 1.82.33H9a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 4 0v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 1 1 2.83 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82V9c.24.58.78 1 1.51 1H21a2 2 0 0 1 0 4h-.09a1.65 1.65 0 0 0-1.51 1Z" /></>, size),
  Pin: ({ size }: IconProps) => svg(<><path d="M12 17v5" /><path d="M9 10.76V4a1 1 0 0 1 1-1h4a1 1 0 0 1 1 1v6.76a2 2 0 0 0 .59 1.41l1.7 1.7a1 1 0 0 1-.7 1.72H6.71a1 1 0 0 1-.7-1.72l1.7-1.7A2 2 0 0 0 9 10.76Z" /></>, size),
  Back: ({ size }: IconProps) => svg(<path d="m15 18-6-6 6-6" />, size),
  Copy: ({ size }: IconProps) => svg(<><rect x="9" y="9" width="13" height="13" rx="2" /><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1" /></>, size),
  Users: ({ size }: IconProps) => svg(<><path d="M16 21v-2a4 4 0 0 0-4-4H6a4 4 0 0 0-4 4v2" /><circle cx="9" cy="7" r="4" /><path d="M22 21v-2a4 4 0 0 0-3-3.87" /><path d="M16 3.13a4 4 0 0 1 0 7.75" /></>, size),
  Board: ({ size }: IconProps) => svg(<><rect x="3" y="3" width="18" height="18" rx="2" /><path d="M9 3v18M15 3v18" /></>, size),
  Calendar: ({ size }: IconProps) => svg(<><rect x="3" y="4" width="18" height="18" rx="2" /><path d="M16 2v4M8 2v4M3 10h18" /></>, size),
  Check: ({ size }: IconProps) => svg(<path d="m20 6-11 11-5-5" />, size),
};
