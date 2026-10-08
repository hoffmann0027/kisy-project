// Zoom and pan for the media viewer, as plain arithmetic so it can be tested
// without a layout engine.
//
// The image sits centred in the stage and is drawn as
//   translate(x, y) scale(scale)
// around its own centre. Every point below is measured from the stage
// centre, in screen pixels.

export interface View {
  scale: number;
  x: number;
  y: number;
}

export interface Size {
  w: number;
  h: number;
}

export interface Point {
  x: number;
  y: number;
}

export const MIN_SCALE = 1;
export const MAX_SCALE = 5;
/** Where a double tap takes the picture, and back. */
export const DOUBLE_TAP_SCALE = 2.5;

export const IDENTITY: View = { scale: 1, x: 0, y: 0 };

export function clampScale(s: number): number {
  return Math.min(MAX_SCALE, Math.max(MIN_SCALE, s));
}

/**
 * Scale to `scale` keeping the image point under `at` exactly where it is —
 * the point between the fingers, or under the cursor, stays put.
 */
export function zoomTo(v: View, scale: number, at: Point): View {
  const s = clampScale(scale);
  const k = s / v.scale;
  return { scale: s, x: at.x - k * (at.x - v.x), y: at.y - k * (at.y - v.y) };
}

/**
 * Keep the picture covering what it can: no panning past its own edges, and
 * a picture smaller than the stage stays centred on that axis.
 */
export function clampPan(v: View, image: Size, stage: Size): View {
  const maxX = Math.max(0, (image.w * v.scale - stage.w) / 2);
  const maxY = Math.max(0, (image.h * v.scale - stage.h) / 2);
  return {
    scale: v.scale,
    x: Math.min(maxX, Math.max(-maxX, v.x)),
    y: Math.min(maxY, Math.max(-maxY, v.y)),
  };
}

/** Double tap: in to DOUBLE_TAP_SCALE at the tapped point, or back to fit. */
export function toggleZoom(v: View, at: Point): View {
  return v.scale > MIN_SCALE ? IDENTITY : zoomTo(v, DOUBLE_TAP_SCALE, at);
}

/** A horizontal swipe on an unzoomed picture: which way to turn, if any. */
export function swipeDirection(dx: number, dy: number, threshold = 60): -1 | 0 | 1 {
  if (Math.abs(dx) < threshold || Math.abs(dx) < Math.abs(dy) * 1.5) return 0;
  return dx < 0 ? 1 : -1;
}
