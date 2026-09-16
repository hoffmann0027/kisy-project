import { describe, expect, it } from "vitest";
import { actionFor, onboardingSteps, permissionsFor } from "./sequence";

describe("which screens the onboarding shows", () => {
  it("asks a phone for all three, notifications first and the call screen last", () => {
    expect(
      onboardingSteps("native", { notifications: "prompt", microphone: "prompt", fullScreenIntent: "denied" }),
    ).toEqual(["notifications", "microphone", "fullScreenIntent"]);
  });

  it("asks a browser only about notifications", () => {
    // Whatever a web build might report for the rest, it has no way to act on it.
    expect(permissionsFor("web")).toEqual(["notifications"]);
    expect(
      onboardingSteps("web", { notifications: "prompt", microphone: "prompt", fullScreenIntent: "denied" }),
    ).toEqual(["notifications"]);
  });

  it("skips what is already granted", () => {
    // Android before 13 has notifications on by default, and before 14 the
    // full-screen call screen needs no permission — those phones see one screen.
    expect(
      onboardingSteps("native", { notifications: "granted", microphone: "prompt", fullScreenIntent: "granted" }),
    ).toEqual(["microphone"]);
  });

  it("skips what the browser does not support, leaving nothing to show", () => {
    expect(onboardingSteps("web", { notifications: "unsupported" })).toEqual([]);
  });

  it("shows nothing when everything is already in place", () => {
    expect(
      onboardingSteps("native", { notifications: "granted", microphone: "granted", fullScreenIntent: "granted" }),
    ).toEqual([]);
  });

  it("keeps a permission refused for good, so the screen can offer Settings", () => {
    expect(onboardingSteps("native", { notifications: "denied", microphone: "granted", fullScreenIntent: "granted" })).toEqual([
      "notifications",
    ]);
  });

  it("never asks for the camera", () => {
    expect(permissionsFor("native")).not.toContain("camera");
  });
});

describe("what a screen's button does", () => {
  it("shows the system dialog while the system still has one to show", () => {
    expect(actionFor("microphone", "prompt", "native")).toBe("request");
    // Refused once: the person may change their mind after reading why.
    expect(actionFor("microphone", "prompt-with-rationale", "native")).toBe("request");
    expect(actionFor("notifications", "prompt", "web")).toBe("request");
  });

  it("opens Settings instead of asking in circles once refused for good", () => {
    expect(actionFor("microphone", "denied", "native")).toBe("settings");
    expect(actionFor("notifications", "denied", "native")).toBe("settings");
  });

  it("explains instead of asking in a browser that was refused for good", () => {
    expect(actionFor("notifications", "denied", "web")).toBe("instructions");
  });

  it("always sends the full-screen call screen to Settings — Android has no dialog for it", () => {
    expect(actionFor("fullScreenIntent", "denied", "native")).toBe("settings");
    expect(actionFor("fullScreenIntent", "prompt", "native")).toBe("settings");
  });

  it("is done once granted", () => {
    expect(actionFor("notifications", "granted", "native")).toBe("done");
    expect(actionFor("fullScreenIntent", "granted", "native")).toBe("done");
  });
});
