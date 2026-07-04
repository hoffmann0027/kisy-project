import { createBrowserRouter, Navigate } from "react-router-dom";
import { LoginPage } from "@pages/auth/LoginPage";
import { RegisterPage } from "@pages/auth/RegisterPage";
import { MessengerPage } from "@pages/messenger/MessengerPage";
import { RatingPage } from "@pages/rating/RatingPage";
import { AdminPage } from "@pages/admin/AdminPage";
import { RequireAuth, RequireCEO, RedirectIfAuth } from "./guards";

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
    path: "/",
    element: (
      <RequireAuth>
        <MessengerPage />
      </RequireAuth>
    ),
  },
  {
    path: "/chat/:chatId",
    element: (
      <RequireAuth>
        <MessengerPage />
      </RequireAuth>
    ),
  },
  {
    path: "/group/:groupId",
    element: (
      <RequireAuth>
        <MessengerPage />
      </RequireAuth>
    ),
  },
  {
    path: "/rating",
    element: (
      <RequireAuth>
        <RatingPage />
      </RequireAuth>
    ),
  },
  {
    path: "/admin",
    element: (
      <RequireCEO>
        <AdminPage />
      </RequireCEO>
    ),
  },
  { path: "*", element: <Navigate to="/" replace /> },
]);
