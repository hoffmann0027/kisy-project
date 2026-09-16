import { existsSync, readFileSync } from "node:fs";
import { join } from "node:path";
import { describe, expect, it } from "vitest";

// The native incoming-call screen is wired together across four files that
// nothing type-checks against each other: an AndroidManifest, a Java plugin, a
// Java activity and a TypeScript bridge. Every mistake here compiles cleanly
// and fails only on a real phone, with a call that never rings — the one bug
// that is hardest to notice and most expensive to miss.
//
// __dirname is frontend/src/shared/config; three levels up is frontend/.
const ANDROID = join(__dirname, "..", "..", "..", "android", "app", "src", "main");
const CALLS = join(ANDROID, "java", "com", "kisy", "messenger", "calls");

const read = (...parts: string[]) => readFileSync(join(...parts), "utf8");

describe.runIf(existsSync(ANDROID))("android call wiring", () => {
  const manifest = () => read(ANDROID, "AndroidManifest.xml");

  it("routes Firebase to exactly one service", () => {
    const xml = manifest();
    // Both our service and the push plugin's answer com.google.firebase
    // .MESSAGING_EVENT. Firebase picks one, and which one depends on manifest
    // merge order — leaving both declared means calls ring on some builds and
    // not others.
    expect(xml).toMatch(/android:name=["']\.calls\.CallPushService["']/);
    expect(xml).toMatch(
      /com\.capacitorjs\.plugins\.pushnotifications\.MessagingService["'][\s\S]{0,120}?tools:node=["']remove["']/,
    );
  });

  it("asks for the permission that lets a call take over a locked screen", () => {
    expect(manifest()).toContain("android.permission.USE_FULL_SCREEN_INTENT");
  });

  it("lets the call screen appear on a sleeping, locked phone", () => {
    const xml = manifest();
    const activity = xml.slice(xml.indexOf(".calls.IncomingCallActivity"));
    // Without these the activity launches behind the keyguard: the phone
    // vibrates, the screen stays dark, and the call looks like it never came.
    expect(activity).toMatch(/android:showWhenLocked=["']true["']/);
    expect(activity).toMatch(/android:turnScreenOn=["']true["']/);
  });

  it("registers the plugin before the bridge starts", () => {
    const main = read(ANDROID, "java", "com", "kisy", "messenger", "MainActivity.java");
    const registered = main.indexOf("registerPlugin(KisyCallPlugin.class)");
    const superCall = main.indexOf("super.onCreate");
    expect(registered).toBeGreaterThan(-1);
    // Capacitor collects plugins while starting; one registered afterwards is
    // invisible to JavaScript, and every call goes unanswered.
    expect(registered).toBeLessThan(superCall);
  });

  it("agrees with the web bridge on the plugin name", () => {
    const plugin = read(CALLS, "KisyCallPlugin.java");
    const bridge = read(__dirname, "..", "lib", "nativeCall.ts");
    // The annotation may span lines once it declares permissions too.
    const native = /@CapacitorPlugin\(\s*name = "([^"]+)"/.exec(plugin)?.[1];
    const web = /registerPlugin<[^>]+>\("([^"]+)"\)/.exec(bridge)?.[1];
    expect(native).toBeTruthy();
    // A mismatch throws only when a call arrives, on a device, in the dark.
    expect(web).toBe(native);
  });

  it("exposes every method the web bridge calls", () => {
    const plugin = read(CALLS, "KisyCallPlugin.java");
    const bridge = read(__dirname, "..", "lib", "nativeCall.ts");
    const declared = [...plugin.matchAll(/@PluginMethod\s+public void (\w+)\(/g)].map((m) => m[1]);
    const used = [...bridge.matchAll(/KisyCall\.(\w+)\(/g)].map((m) => m[1]).filter((m) => m !== "addListener");
    // Guard against the loop below passing because it found nothing to check.
    expect(used.length).toBeGreaterThan(2);
    for (const method of new Set(used)) expect(declared).toContain(method);
  });

  it("may switch the call to the earpiece and blank the screen at the ear", () => {
    const xml = manifest();
    // Without MODIFY_AUDIO_SETTINGS the route calls are silently ignored and
    // the call stays on the loudspeaker; without WAKE_LOCK acquiring the
    // proximity lock throws.
    expect(xml).toContain("android.permission.MODIFY_AUDIO_SETTINGS");
    expect(xml).toContain("android.permission.WAKE_LOCK");
  });

  it("emits every event the web bridge listens for", () => {
    const java = read(CALLS, "KisyCallPlugin.java");
    const bridge = read(__dirname, "..", "lib", "nativeCall.ts");
    const emitted = [...java.matchAll(/notifyListeners\("(\w+)"/g)].map((m) => m[1]);
    const listened = [...bridge.matchAll(/KisyCall\.addListener\("(\w+)"/g)].map((m) => m[1]);
    expect(listened).toEqual(expect.arrayContaining(["callAction", "audioRoute"]));
    for (const event of listened) expect(emitted).toContain(event);
  });

  it("names the permissions the same on both sides", () => {
    const java = read(CALLS, "KisyCallPlugin.java");
    const bridge = read(__dirname, "..", "lib", "nativeCall.ts");
    const union = /export type NativePermission = ([^;]+);/.exec(bridge)?.[1] ?? "";
    const web = [...union.matchAll(/"(\w+)"/g)].map((m) => m[1]).sort();
    const native = [...java.matchAll(/static final String \w+ = "(\w+)";/g)].map((m) => m[1]).sort();
    expect(web).toEqual(["fullScreenIntent", "microphone", "notifications"]);
    // A name the plugin does not know resolves as a request for nothing.
    expect(native).toEqual(web);
  });
});
