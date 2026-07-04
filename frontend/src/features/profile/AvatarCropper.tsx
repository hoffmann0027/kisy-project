import { useCallback, useEffect, useRef, useState, type PointerEvent as ReactPointerEvent } from "react";
import { Avatar, Button, Modal, toast } from "@shared/ui";

const FRAME = 260; // on-screen crop square, px
const OUTPUT = 256; // exported avatar size, px
const MAX_INPUT_BYTES = 8 * 1024 * 1024; // reject huge source files before decoding
const ACCEPT = ["image/jpeg", "image/png", "image/webp"];

interface Props {
  /** Current avatar for the trigger preview. */
  name: string;
  url?: string | null;
  size?: number;
  /** Uploads the cropped square as a JPEG blob. Should resolve when stored. */
  onUpload: (blob: Blob) => Promise<void>;
  /** Whether the current user may change this avatar. */
  disabled?: boolean;
}

// AvatarCropper renders an avatar that, when clicked (or a file is dropped on
// it), opens a square cropper: the image can be zoomed and panned inside a
// fixed square, then exported to a 256×256 JPEG for upload. Cropping/resizing
// happens client-side so the server only ever stores a small square image.
export function AvatarCropper({ name, url, size = 56, onUpload, disabled }: Props) {
  const [src, setSrc] = useState<string | null>(null); // object URL of chosen file
  const [busy, setBusy] = useState(false);
  const fileRef = useRef<HTMLInputElement>(null);

  const pick = (file: File | undefined) => {
    if (!file) return;
    if (!ACCEPT.includes(file.type)) {
      toast.error("Только JPEG, PNG или WebP");
      return;
    }
    if (file.size > MAX_INPUT_BYTES) {
      toast.error("Файл слишком большой (макс. 8 МБ)");
      return;
    }
    setSrc(URL.createObjectURL(file));
  };

  const close = () => {
    if (src) URL.revokeObjectURL(src);
    setSrc(null);
  };

  const upload = async (blob: Blob) => {
    setBusy(true);
    try {
      await onUpload(blob);
      toast.success("Аватар обновлён");
      close();
    } catch {
      toast.error("Не удалось загрузить аватар");
    } finally {
      setBusy(false);
    }
  };

  return (
    <>
      <button
        type="button"
        className="avatar-trigger"
        disabled={disabled}
        onClick={() => fileRef.current?.click()}
        onDragOver={(e) => e.preventDefault()}
        onDrop={(e) => {
          e.preventDefault();
          if (!disabled) pick(e.dataTransfer.files?.[0]);
        }}
        title={disabled ? "" : "Изменить аватар"}
      >
        <Avatar name={name} url={url} size={size} />
        {!disabled && <span className="avatar-trigger__hint">Изменить</span>}
      </button>
      <input
        ref={fileRef}
        type="file"
        accept={ACCEPT.join(",")}
        style={{ display: "none" }}
        onChange={(e) => {
          pick(e.target.files?.[0]);
          e.target.value = ""; // allow re-picking the same file
        }}
      />

      {src && <CropDialog src={src} busy={busy} onCancel={close} onConfirm={upload} />}
    </>
  );
}

function CropDialog({
  src,
  busy,
  onCancel,
  onConfirm,
}: {
  src: string;
  busy: boolean;
  onCancel: () => void;
  onConfirm: (blob: Blob) => void;
}) {
  const [img, setImg] = useState<HTMLImageElement | null>(null);
  const [zoom, setZoom] = useState(1);
  const [offset, setOffset] = useState({ x: 0, y: 0 });
  const drag = useRef<{ x: number; y: number } | null>(null);

  // Base scale makes the image "cover" the square at zoom 1.
  const baseScale = img ? FRAME / Math.min(img.naturalWidth, img.naturalHeight) : 1;
  const scale = baseScale * zoom;
  const dispW = img ? img.naturalWidth * scale : 0;
  const dispH = img ? img.naturalHeight * scale : 0;

  const clamp = useCallback(
    (o: { x: number; y: number }) => ({
      x: Math.min(0, Math.max(FRAME - dispW, o.x)),
      y: Math.min(0, Math.max(FRAME - dispH, o.y)),
    }),
    [dispW, dispH],
  );

  useEffect(() => {
    const el = new Image();
    el.onload = () => {
      setImg(el);
      // Center the image in the frame initially.
      const s = FRAME / Math.min(el.naturalWidth, el.naturalHeight);
      setOffset({ x: (FRAME - el.naturalWidth * s) / 2, y: (FRAME - el.naturalHeight * s) / 2 });
    };
    el.src = src;
  }, [src]);

  useEffect(() => {
    setOffset((o) => clamp(o));
  }, [zoom, clamp]);

  const onPointerDown = (e: ReactPointerEvent) => {
    drag.current = { x: e.clientX - offset.x, y: e.clientY - offset.y };
    (e.target as Element).setPointerCapture(e.pointerId);
  };
  const onPointerMove = (e: ReactPointerEvent) => {
    if (!drag.current) return;
    setOffset(clamp({ x: e.clientX - drag.current.x, y: e.clientY - drag.current.y }));
  };
  const onPointerUp = () => {
    drag.current = null;
  };

  const confirm = () => {
    if (!img) return;
    const canvas = document.createElement("canvas");
    canvas.width = OUTPUT;
    canvas.height = OUTPUT;
    const ctx = canvas.getContext("2d");
    if (!ctx) return;
    // The frame shows image-space [-offset/scale, (FRAME-offset)/scale].
    const sx = -offset.x / scale;
    const sy = -offset.y / scale;
    const sSize = FRAME / scale;
    ctx.drawImage(img, sx, sy, sSize, sSize, 0, 0, OUTPUT, OUTPUT);
    canvas.toBlob((blob) => blob && onConfirm(blob), "image/jpeg", 0.9);
  };

  return (
    <Modal open title="Обрезка аватара" onClose={onCancel}>
      <div className="cropper">
        <div
          className="cropper__frame"
          style={{ width: FRAME, height: FRAME }}
          onPointerDown={onPointerDown}
          onPointerMove={onPointerMove}
          onPointerUp={onPointerUp}
          onPointerLeave={onPointerUp}
        >
          {img && (
            <img
              src={src}
              alt=""
              draggable={false}
              style={{
                position: "absolute",
                left: offset.x,
                top: offset.y,
                width: dispW,
                height: dispH,
                maxWidth: "none",
                userSelect: "none",
              }}
            />
          )}
        </div>
        <label className="cropper__zoom">
          Масштаб
          <input
            type="range"
            min={1}
            max={3}
            step={0.01}
            value={zoom}
            onChange={(e) => setZoom(Number(e.target.value))}
          />
        </label>
        <div style={{ display: "flex", gap: 8, justifyContent: "flex-end" }}>
          <Button variant="ghost" onClick={onCancel} disabled={busy}>
            Отмена
          </Button>
          <Button variant="primary" onClick={confirm} loading={busy}>
            Сохранить
          </Button>
        </div>
      </div>
    </Modal>
  );
}
