import { createBrowserRouter, Navigate, Outlet } from "react-router-dom";
import { LoginPage } from "@pages/auth/LoginPage";
import { RegisterPage } from "@pages/auth/RegisterPage";
import { MessengerPage } from "@pages/messenger/MessengerPage";
import { RatingPage } from "@pages/rating/RatingPage";
import { AdminPage } from "@pages/admin/AdminPage";
import { RequireAuth, RequireCEO, RequireRatingAccess, RedirectIfAuth } from "./guards";
import { useRealtime } from "./useRealtime";
import { CallProvider } from "@features/call/CallProvider";
import { HubPage } from "@pages/hub/HubPage";
import { TabBar } from "@widgets/tabbar/TabBar";
import { ProfileModal } from "@features/profile/ProfileModal";
import { useState } from "react";

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
      <Outlet />
      <TabBar onProfile={() => setProfileOpen(true)} />
      <ProfileModal open={profileOpen} onClose={() => setProfileOpen(false)} />
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
