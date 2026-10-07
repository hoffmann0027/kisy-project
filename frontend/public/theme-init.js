// Apply the persisted theme before first paint to avoid a flash of the
// default theme for returning users (mirrors useThemeStore, key kisy-theme).
// A file of its own, not an inline <script>: the CSP allows no inline script,
// so inline it never ran.
try {
  var t = JSON.parse(localStorage.getItem("kisy-theme") || "null");
  var theme = (t && t.state && t.state.theme) || "orbit";
  if (["orbit", "glass", "luce", "aurora", "cyber", "xp", "matrix"].indexOf(theme) === -1) theme = "orbit";
  document.documentElement.setAttribute("data-theme", theme);
} catch (e) {}
