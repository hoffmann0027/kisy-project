import { createBrowserRouter, Navigate, Outlet } from "react-router-dom";
import { LoginPage } from "@pages/auth/LoginPage";
import { RegisterPage } from "@pages/auth/RegisterPage";
import { MessengerPage } from "@pages/messenger/MessengerPage";
import { RequireAuth, RequireCEO, RequireRatingAccess, RedirectIfAuth } from "./guards";
import { useRealtime } from "./useRealtime";
import { CallProvider } from "@features/call/CallProvider";
import { TabBar } from "@widgets/tabbar/TabBar";
import { RouteFallback } from "./RouteFallback";
import { lazy, Suspense, useState } from "react";

// Everything below the messenger is loaded on demand. They are whole screens
// the app does not need to start — Rating drags in recharts, Admin its tables
// — and bundling them into the entry chunk meant the phone parsed all of it
// before it could paint the chat list.
const RatingPage = lazy(() => import("@pages/rating/RatingPage").then((m) => ({ default: m.RatingPage })));
const AdminPage = lazy(() => import("@pages/admin/AdminPage").then((m) => ({ default: m.AdminPage })));
const HubPage = lazy(() => import("@pages/hub/HubPage").then((m) => ({ default: m.HubPage })));
// The profile dialog is mounted on every authenticated screen but opened
// rarely, so its code (and the theme switcher's) waits for the first open.
const ProfileModal = lazy(() =>
  import("@features/profile/ProfileModal").then((m) => ({ default: m.ProfileModal })),
);

// AuthedLayout keeps the WebSocket connection alive across every authenticated
// route (messenger, rating, admin) so real-time events reach whichever page is
// open — the layout stays mounted while its child routes swap. CallProvider
// wraps it so incoming/ongoing calls surface on any page.
function AuthedLayout() {
  useRealtime();
  // The phone tab bar lives here, not inside a page, so it survives route
  // changes and shows up on every authenticated screen (it hides itself on
  // chat routes and on desktop). Profile is a dialog rather than a route, so
  // its state sits alongside.
  const [profileOpen, setProfileOpen] = useState(false);
  return (
    <CallProvider>
      <Suspense fallback={<RouteFallback />}>
        <Outlet />
      </Suspense>
      <TabBar onProfile={() => setProfileOpen(true)} />
      {/* Mounted only once opened: an unopened dialog should cost nothing. */}
      {profileOpen && (
        <Suspense fallback={null}>
          <ProfileModal open onClose={() => setProfileOpen(false)} />
        </Suspense>
      )}
    </CallProvider>
  );
}

export const router = createBrowserRouter([
  {
    path: "/login",
    element: (
      <RedirectIfAuth>
        <LoginPage />
      </RedirectIfAuth>
    ),
  },
  {
    path: "/register",
    element: (
      <RedirectIfAuth>
        <RegisterPage />
      </RedirectIfAuth>
    ),
  },
  {
    element: <AuthedLayout />,
    children: [
      { path: "/", element: <RequireAuth><MessengerPage /></RequireAuth> },
      { path: "/chat/:chatId", element: <RequireAuth><MessengerPage /></RequireAuth> },
      { path: "/communities", element: <RequireAuth><MessengerPage /></RequireAuth> },
      { path: "/group/:groupId", element: <RequireAuth><MessengerPage /></RequireAuth> },
      { path: "/hub", element: <RequireAuth><HubPage /></RequireAuth> },
      { path: "/rating", element: <RequireRatingAccess><RatingPage /></RequireRatingAccess> },
      { path: "/admin", element: <RequireCEO><AdminPage /></RequireCEO> },
    ],
  },
  { path: "*", element: <Navigate to="/" replace /> },
]);
