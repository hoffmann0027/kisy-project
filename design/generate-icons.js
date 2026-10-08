const sharp = require("sharp");
const fs = require("fs");
const path = require("path");

// The repository root, wherever it was cloned. It used to be one developer's
// absolute Windows path, so the script ran on exactly one machine (audit C-10).
const ROOT = path.resolve(__dirname, "..");
const SRC = path.join(ROOT, "design/logo-source.png");
const PUB = path.join(ROOT, "frontend/public");
const RES = path.join(ROOT, "frontend/android/app/src/main/res");

// The October 2026 artwork is a clear-glass tile in the manner of iOS 26:
// a glowing glass frame around the orange "K" bubble, with nothing behind it —
// the background shows through the tile. The whole tile is the icon now (the
// earlier artwork was light on black, and only the bubble was cut out of it).
//
// One thing is added: a faint frosted pane inside the frame. Without it the
// tile reads as an empty outline on a busy wallpaper; with it, as a sheet of
// glass. It is brighter at the top and fades downwards, like light on glass,
// and stays low enough that whatever is behind still shows through.
const FROST_TOP = 0.16;
const FROST_BOTTOM = 0.05;
// Pixels at least this opaque are the frame: the flood fill that finds the
// outside stops at them.
const FRAME_ALPHA = 200;

const DARK = { r: 11, g: 12, b: 20, alpha: 1 }; // --bg of the default "orbit" theme
const CLEAR = { r: 0, g: 0, b: 0, alpha: 0 };

/** The artwork with its frosted pane, trimmed to the tile and square. */
async function glassTile() {
  const { data, info } = await sharp(SRC).ensureAlpha().raw().toBuffer({ resolveWithObject: true });
  const { width: W, height: H } = info;

  // Outside = everything reachable from the border without crossing the
  // frame. What is left is the tile: the frame and the glass inside it.
  const outside = new Uint8Array(W * H);
  const stack = [];
  const push = (x, y) => {
    if (x < 0 || y < 0 || x >= W || y >= H) return;
    const i = y * W + x;
    if (outside[i] || data[i * 4 + 3] >= FRAME_ALPHA) return;
    outside[i] = 1;
    stack.push(i);
  };
  for (let x = 0; x < W; x++) { push(x, 0); push(x, H - 1); }
  for (let y = 0; y < H; y++) { push(0, y); push(W - 1, y); }
  while (stack.length) {
    const i = stack.pop();
    const x = i % W, y = (i - x) / W;
    push(x + 1, y); push(x - 1, y); push(x, y + 1); push(x, y - 1);
  }

  let top = H, bottom = 0, left = W, right = 0, inside = 0;
  for (let y = 0; y < H; y++) {
    for (let x = 0; x < W; x++) {
      if (outside[y * W + x]) continue;
      inside++;
      if (y < top) top = y;
      if (y > bottom) bottom = y;
      if (x < left) left = x;
      if (x > right) right = x;
    }
  }
  // A frame with a gap would let the fill run inside and find no tile at all.
  if (inside < W * H * 0.3) throw new Error(`tile not found (inside ${inside} of ${W * H} px): is the frame closed?`);

  // The pane, under the artwork: source-over of the art onto frosted white.
  const out = Buffer.alloc(W * H * 4);
  for (let y = 0; y < H; y++) {
    const t = (y - top) / Math.max(1, bottom - top);
    const frost = FROST_TOP + (FROST_BOTTOM - FROST_TOP) * Math.min(1, Math.max(0, t));
    for (let x = 0; x < W; x++) {
      const i = y * W + x, o = i * 4;
      const sa = data[o + 3] / 255;
      const ba = outside[i] ? 0 : frost;
      const a = sa + ba * (1 - sa);
      if (a === 0) continue;
      for (let c = 0; c < 3; c++) out[o + c] = Math.round((data[o + c] * sa + 255 * ba * (1 - sa)) / a);
      out[o + 3] = Math.round(a * 255);
    }
  }

  // Square crop around the tile, with room for the glow outside the frame.
  const glow = Math.round(Math.max(right - left, bottom - top) * 0.03);
  const side = Math.max(right - left, bottom - top) + 1 + glow * 2;
  const cx = Math.round((left + right) / 2), cy = Math.round((top + bottom) / 2);
  const x0 = cx - Math.floor(side / 2), y0 = cy - Math.floor(side / 2);
  const ext = {
    left: Math.max(0, -x0), top: Math.max(0, -y0),
    right: Math.max(0, x0 + side - W), bottom: Math.max(0, y0 + side - H),
  };
  return sharp(out, { raw: { width: W, height: H, channels: 4 } })
    .extend({ ...ext, background: CLEAR })
    .extract({ left: x0 + ext.left, top: y0 + ext.top, width: side, height: side })
    .png()
    .toBuffer();
}

/** The tile scaled to `ratio` of a square canvas, centred on `bg`. */
async function composed(tile, size, ratio, bg) {
  const inner = Math.round(size * ratio);
  const layer = await sharp(tile).resize(inner, inner).toBuffer();
  return sharp({ create: { width: size, height: size, channels: 4, background: bg } })
    .composite([{ input: layer, gravity: "centre" }]).png().toBuffer();
}

(async () => {
  const tile = await glassTile();
  const write = async (buf, file) => { await fs.promises.writeFile(file, buf); console.log("  ", path.relative(ROOT, file)); };

  console.log("web:");
  // In the app and in the browser the glass stays clear: whatever is behind
  // it — any of the seven themes, a tab strip — shows through.
  await write(await sharp(tile).resize(512, 512).png().toBuffer(), path.join(PUB, "logo.png"));
  for (const [file, size] of [["favicon.png", 48], ["icon-192.png", 192], ["icon-512.png", 512]]) {
    await write(await sharp(tile).resize(size, size).png().toBuffer(), path.join(PUB, file));
  }
  // iOS fills a transparent home-screen icon with black, and the system then
  // rounds it: the tile on the app's own dark ground looks deliberate instead.
  await write(await composed(tile, 180, 0.9, DARK), path.join(PUB, "apple-touch-icon.png"));
  // Maskable: Android crops to its own shape and fills nothing in, so this
  // one keeps a ground; the tile stays inside the 80% safe circle.
  await write(await composed(tile, 512, 0.6, DARK), path.join(PUB, "icon-maskable-512.png"));

  console.log("store:");
  // Google Play wants a 512x512 icon with no transparency and draws its own
  // rounded mask over it: the tile on the app's dark ground, full bleed.
  await write(await composed(tile, 512, 0.86, DARK), path.join(ROOT, "design/play-icon-512.png"));

  console.log("android launcher:");
  const densities = { mdpi: 48, hdpi: 72, xhdpi: 96, xxhdpi: 144, xxxhdpi: 192 };
  for (const [d, size] of Object.entries(densities)) {
    const dir = path.join(RES, "mipmap-" + d);
    // Before Android 8 the launcher shows the bitmap as it is: clear glass.
    await write(await composed(tile, size, 0.92, CLEAR), path.join(dir, "ic_launcher.png"));
    await write(await composed(tile, size, 0.82, CLEAR), path.join(dir, "ic_launcher_round.png"));
    // Adaptive (8+): a transparent background colour and this foreground.
    // 108dp canvas; the most aggressive mask is a 72dp circle, and a rounded
    // square of side s with ~22% corners reaches 0.616·s from the centre — so
    // s ≤ 0.54 keeps the frame whole under every mask.
    const fg = Math.round(size * 2.25);
    await write(await composed(tile, fg, 0.54, CLEAR), path.join(dir, "ic_launcher_foreground.png"));
  }

  console.log("splash:");
  const splashes = [
    ["drawable", 480, 320], ["drawable-land-mdpi", 480, 320], ["drawable-land-hdpi", 800, 480],
    ["drawable-land-xhdpi", 1280, 720], ["drawable-land-xxhdpi", 1600, 960], ["drawable-land-xxxhdpi", 1920, 1280],
    ["drawable-port-mdpi", 320, 480], ["drawable-port-hdpi", 480, 800], ["drawable-port-xhdpi", 720, 1280],
    ["drawable-port-xxhdpi", 960, 1600], ["drawable-port-xxxhdpi", 1280, 1920],
  ];
  for (const [dir, w, h] of splashes) {
    const inner = Math.round(Math.min(w, h) * 0.38);
    const layer = await sharp(tile).resize(inner, inner).toBuffer();
    const buf = await sharp({ create: { width: w, height: h, channels: 4, background: DARK } })
      .composite([{ input: layer, gravity: "centre" }]).png().toBuffer();
    await write(buf, path.join(RES, dir, "splash.png"));
  }
})().catch((e) => {
  console.error(e);
  process.exit(1);
});
