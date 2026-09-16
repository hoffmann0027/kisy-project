import { useEffect, useState } from "react";
import { Button } from "@shared/ui";
import { useBackHandler } from "@shared/lib/backStack";
import { PERMISSION_COPY } from "./copy";
import { markOnboardingDone, onboardingDone } from "./platform";
import { actionFor, onboardingSteps, type AppPermission } from "./sequence";
import { usePermissionStates } from "./usePermissionStates";
import "./permissions.css";

// The first-run walk through the permissions a messenger with calls needs —
// one per screen, the reason before the system dialog, because a dialog that
// arrives unexplained is the one people refuse.
//
// Shown once per device. Everything here is also reachable later from the
// profile ("Разрешения"), so "Не сейчас" is never a dead end.

export function PermissionsOnboarding() {
  const [pending, setPending] = useState(false);
  useEffect(() => {
    let alive = true;
    void onboardingDone().then((done) => {
      if (alive) setPending(!done);
    });
    return () => {
      alive = false;
    };
  }, []);

  if (!pending) return null;
  return <Walkthrough onFinish={() => setPending(false)} />;
}

function Walkthrough({ onFinish }: { onFinish: () => void }) {
  const { states, platform, act } = usePermissionStates();
  // Which screens to show is decided once, from the states at the start: a
  // permission granted on its own screen must not make the list shift under
  // the person reading it.
  const [steps, setSteps] = useState<AppPermission[] | null>(null);
  const [index, setIndex] = useState(0);
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    if (states && steps === null) setSteps(onboardingSteps(platform, states));
  }, [platform, states, steps]);

  const permission = steps?.[index];
  const state = permission ? states?.[permission] : undefined;
  const last = !steps || index + 1 >= steps.length;

  const next = () => {
    if (last) {
      void markOnboardingDone();
      onFinish();
    } else {
      setIndex((i) => i + 1);
    }
  };

  // Granted — by the dialog, or in Settings while the app was away — moves on.
  // Nothing to ask on this device at all — done, and never mounted again.
  useEffect(() => {
    if (steps?.length === 0 || (permission && state === "granted")) next();
    // eslint-disable-next-line react-hooks/exhaustive-deps -- next() is derived from exactly these
  }, [steps, permission, state]);

  // Back means "not now", rather than leaving the app from under the screen.
  useBackHandler(!!permission, next);

  if (!permission || !state || !steps) return null;

  const copy = PERMISSION_COPY[permission];
  const action = actionFor(permission, state, platform);

  const run = async () => {
    setBusy(true);
    try {
      await act(permission);
    } finally {
      setBusy(false);
    }
  };

  let text = copy.why;
  if (action === "settings" || action === "instructions") text = copy.settingsHint(platform);
  else if (state === "prompt-with-rationale") text = `${copy.again} ${copy.why}`;

  return (
    <div className="perm-onboarding" role="dialog" aria-modal="true" aria-labelledby="perm-title">
      <div className="perm-card">
        <div className="perm-progress" aria-label={`Шаг ${index + 1} из ${steps.length}`}>
          {steps.map((p, i) => (
            <span key={p} className={"perm-progress__dot" + (i <= index ? " perm-progress__dot--on" : "")} />
          ))}
        </div>
        <div className="perm-card__icon">{copy.icon}</div>
        <h2 id="perm-title" className="perm-card__title">
          {copy.title}
        </h2>
        <p className="perm-card__text">{text}</p>
        <div className="perm-card__actions">
          {action === "request" && (
            <Button block loading={busy} onClick={() => void run()}>
              Разрешить
            </Button>
          )}
          {action === "settings" && (
            <Button block loading={busy} onClick={() => void run()}>
              Открыть настройки
            </Button>
          )}
          <Button variant="ghost" block onClick={next}>
            {last ? "Готово" : "Не сейчас"}
          </Button>
        </div>
      </div>
    </div>
  );
}
