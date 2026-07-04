// One-off icon generator: rasterizes the SVG logo into the PNG sizes the PWA
// manifest and Apple touch icon need. Run via Docker so no host image tooling
// is required:
//
//   docker run --rm -v "$PWD":/w -w /w node:20-alpine \
//     sh -c "npm i sharp@0.33 --no-save --silent && node scripts/gen-icons.mjs"
//
// Commits the generated PNGs; re-run only when the SVG changes.
import sharp from "sharp";
import { readFileSync } from "node:fs";
import { fileURLToPath } from "node:url";

const pub = new URL("../public/", import.meta.url);
const rounded = readFileSync(new URL("favicon.svg", pub));
const maskable = readFileSync(new URL("icon-maskable.svg", pub));

const jobs = [
  { svg: rounded, size: 192, out: "icon-192.png" },
  { svg: rounded, size: 512, out: "icon-512.png" },
  { svg: maskable, size: 512, out: "icon-maskable-512.png" },
  { svg: maskable, size: 180, out: "apple-touch-icon.png" },
];

for (const { svg, size, out } of jobs) {
  await sharp(svg, { density: 384 })
    .resize(size, size, { fit: "contain", background: { r: 0, g: 0, b: 0, alpha: 0 } })
    .png()
    .toFile(fileURLToPath(new URL(out, pub)));
  console.log("wrote public/" + out + " (" + size + "px)");
}
