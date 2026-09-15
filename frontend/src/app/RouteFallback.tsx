// Shown while a lazily-loaded screen's chunk is in flight. Deliberately a
// quiet shimmer rather than a spinner: on a fast connection it flashes for a
// frame or two, and a spinner appearing for 50ms reads as a glitch.
export function RouteFallback() {
  return (
    <div className="route-fallback" role="status" aria-label="Загрузка">
      <span className="route-fallback__bar" />
      <span className="route-fallback__bar route-fallback__bar--short" />
    </div>
  );
}
