const sharp = require("sharp");
const fs = require("fs");
const path = require("path");

const ROOT = "C:/Users/hamza/Desktop/ProjectS/Claude/messenger_kisy/kisy-project";
const SRC = path.join(ROOT, "design/logo-source.png");
const PUB = path.join(ROOT, "frontend/public");
const RES = path.join(ROOT, "frontend/android/app/src/main/res");

// Measured from the source: the mark (bubble + K) occupies x 263..1016,
// y 121..873. Centre it in a square with ~7% breathing room so the glow is
// not clipped when the tile is masked to a circle or a squircle.
const CX = 640, CY = 497, HALF = 405;
const CROP = { left: CX - HALF, top: CY - HALF, width: HALF * 2, height: HALF * 2 };

const DARK = { r: 11, g: 12, b: 20, alpha: 1 }; // --bg of the default "orbit" theme

const markOpaque = () => sharp(SRC).extract(CROP);

// The artwork is light drawn on black, so luminance IS the coverage: using it
// as alpha lifts the mark off its background and lets the glow fade out
// naturally instead of ending at a hard matte edge. The floor matters — the
// source has a faint vignette across the whole tile, and carrying it over as
// low-alpha dark pixels paints a visible grey box on the light themes.
const ALPHA_FLOOR = 18;
async function markAlpha() {
  const { data, info } = await markOpaque().raw().toBuffer({ resolveWithObject: true });
  const { width, height, channels } = info;
  const out = Buffer.alloc(width * height * 4);
  for (let i = 0, o = 0; i < data.length; i += channels, o += 4) {
    const r = data[i], g = data[i + 1], b = data[i + 2];
    out[o] = r; out[o + 1] = g; out[o + 2] = b;
    const luma = Math.max(r, g, b);
    out[o + 3] = luma <= ALPHA_FLOOR ? 0 : Math.round(((luma - ALPHA_FLOOR) / (255 - ALPHA_FLOOR)) * 255);
  }
  return sharp(out, { raw: { width, height, channels: 4 } }).png().toBuffer();
}

/** Mark scaled to `ratio` of the canvas, centred on `bg` (null = transparent). */
async function composed(alphaBuf, size, ratio, bg) {
  const inner = Math.round(size * ratio);
  const layer = await sharp(alphaBuf).resize(inner, inner).toBuffer();
  return sharp({
    create: { width: size, height: size, channels: 4, background: bg ?? { r: 0, g: 0, b: 0, alpha: 0 } },
  }).composite([{ input: layer, gravity: "centre" }]).png().toBuffer();
}

(async () => {
  const alpha = await markAlpha();
  const write = async (buf, file) => { await fs.promises.writeFile(file, buf); console.log("  ", path.relative(ROOT, file)); };

  console.log("web:");
  // In-app mark: transparent, so it sits on any of the seven themes.
  await write(await sharp(alpha).resize(512, 512).png().toBuffer(), path.join(PUB, "logo.png"));
  // Browser/OS tiles keep the artwork's own dark ground.
  for (const [file, size] of [["favicon.png", 48], ["apple-touch-icon.png", 180], ["icon-192.png", 192], ["icon-512.png", 512]]) {
    await write(await markOpaque().resize(size, size).png().toBuffer(), path.join(PUB, file));
  }
  // Maskable: Android crops to its own shape, so the mark must stay inside
  // the inner 60% safe zone.
  await write(await composed(alpha, 512, 0.6, DARK), path.join(PUB, "icon-maskable-512.png"));

  console.log("android launcher:");
  const densities = { mdpi: 48, hdpi: 72, xhdpi: 96, xxhdpi: 144, xxxhdpi: 192 };
  for (const [d, size] of Object.entries(densities)) {
    const dir = path.join(RES, "mipmap-" + d);
    await write(await composed(alpha, size, 0.86, DARK), path.join(dir, "ic_launcher.png"));
    // Round mask bites the corners: pull the mark in further.
    await write(await composed(alpha, size, 0.72, DARK), path.join(dir, "ic_launcher_round.png"));
    // Adaptive foreground: transparent, mark inside the 66% safe zone.
    const fg = Math.round(size * 2.25); // 108dp canvas for a 48dp icon
    await write(await composed(alpha, fg, 0.62, null), path.join(dir, "ic_launcher_foreground.png"));
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
    const layer = await sharp(alpha).resize(inner, inner).toBuffer();
    const buf = await sharp({ create: { width: w, height: h, channels: 4, background: DARK } })
      .composite([{ input: layer, gravity: "centre" }]).png().toBuffer();
    await write(buf, path.join(RES, dir, "splash.png"));
  }
})();
