import { App } from "@capacitor/app";
import { isNative, setAppVersionHeader } from "./native";

// The packaged app tells the server which build it is, so the admin panel can
// say how many people still run an old one ("Обновления"). The browser has no
// build of its own — the page is always the deployed one — and sends nothing.

const VERSION = /^[0-9A-Za-z._+-]{1,64}$/;
const BUILD = /^[0-9]{1,15}$/;

/** "<versionName> (<versionCode>)", or null for anything the server would refuse. */
export function formatAppVersion(version: string, build: string): string | null {
  return VERSION.test(version) && BUILD.test(build) ? `${version} (${build})` : null;
}

/** Reads the build once at start-up; every later request carries it. */
export async function initAppVersion(): Promise<void> {
  if (!isNative()) return;
  try {
    const info = await App.getInfo();
    setAppVersionHeader(formatAppVersion(info.version, info.build));
  } catch {
    // Not knowing the version costs a number on a dashboard, nothing more.
  }
}
