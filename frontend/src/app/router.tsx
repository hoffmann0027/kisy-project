import { createBrowserRouter } from "react-router-dom";
import { StatusPage } from "@pages/status/StatusPage";

export const router = createBrowserRouter([
  {
    path: "/",
    element: <StatusPage />,
  },
]);
