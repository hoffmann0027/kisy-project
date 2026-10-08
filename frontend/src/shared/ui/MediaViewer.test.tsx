import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { MediaViewer } from "./MediaViewer";

// A user's suggestion: zooming into pictures in chats was clumsy. The viewer
// now zooms like a phone's gallery — pinch, double tap, drag, wheel, keys —
// and a swipe turns the page. jsdom has no layout, so these pin down the
// gestures, not the pixels (the arithmetic is in zoomPan.test.ts).

// jsdom has no PointerEvent: without it the events carry no coordinates and
// no pointer id. A MouseEvent with a pointerId is all the viewer reads.
if (!("PointerEvent" in window)) {
  class PointerEventPolyfill extends MouseEvent {
    pointerId: number;
    constructor(type: string, init: PointerEventInit = {}) {
      super(type, init);
      this.pointerId = init.pointerId ?? 0;
    }
  }
  Object.defineProperty(window, "PointerEvent", { value: PointerEventPolyfill, configurable: true });
}

const items = [
  { id: "1", url: "/api/v1/attachments/1", fileName: "one.jpg" },
  { id: "2", url: "/api/v1/attachments/2", fileName: "two.jpg" },
];

function setup(index = 0) {
  const onClose = vi.fn();
  const onIndexChange = vi.fn();
  render(<MediaViewer items={items} index={index} onClose={onClose} onIndexChange={onIndexChange} />);
  const stage = screen.getByTestId("mviewer-stage");
  const img = screen.getByAltText(items[index].fileName);
  return { onClose, onIndexChange, stage, img };
}

const scaleOf = (img: HTMLElement) => Number(/scale\(([\d.]+)\)/.exec(img.style.transform)?.[1] ?? NaN);

function tap(target: HTMLElement, x = 100, y = 100, id = 1) {
  fireEvent.pointerDown(target, { pointerId: id, clientX: x, clientY: y });
  fireEvent.pointerUp(target, { pointerId: id, clientX: x, clientY: y });
}

describe("the media viewer", () => {
  it("zooms in on a double tap and back out on the next", () => {
    const { img, onClose } = setup();
    expect(scaleOf(img)).toBe(1);
    tap(img);
    tap(img);
    expect(scaleOf(img)).toBeGreaterThan(2);
    tap(img);
    tap(img);
    expect(scaleOf(img)).toBe(1);
    expect(onClose).not.toHaveBeenCalled();
  });

  it("closes on a tap beside the picture, not on the picture", () => {
    const { img, stage, onClose } = setup();
    tap(img);
    expect(onClose).not.toHaveBeenCalled();
    tap(stage, 5, 5, 2);
    expect(onClose).toHaveBeenCalledTimes(1);
  });

  it("zooms with two fingers", () => {
    const { img, stage } = setup();
    fireEvent.pointerDown(img, { pointerId: 1, clientX: 100, clientY: 100 });
    fireEvent.pointerDown(stage, { pointerId: 2, clientX: 150, clientY: 100 });
    fireEvent.pointerMove(stage, { pointerId: 2, clientX: 250, clientY: 100 });
    expect(scaleOf(img)).toBeCloseTo(3);
    fireEvent.pointerUp(stage, { pointerId: 2, clientX: 250, clientY: 100 });
    fireEvent.pointerUp(img, { pointerId: 1, clientX: 100, clientY: 100 });
    expect(scaleOf(img)).toBeCloseTo(3);
  });

  it("turns to the next picture on a swipe, and not while zoomed", () => {
    const { img, onIndexChange } = setup();
    fireEvent.pointerDown(img, { pointerId: 1, clientX: 300, clientY: 200 });
    fireEvent.pointerMove(img, { pointerId: 1, clientX: 150, clientY: 210 });
    fireEvent.pointerUp(img, { pointerId: 1, clientX: 150, clientY: 210 });
    expect(onIndexChange).toHaveBeenCalledWith(1);

    onIndexChange.mockClear();
    tap(img);
    tap(img); // zoomed in: a drag now moves the picture instead
    fireEvent.pointerDown(img, { pointerId: 1, clientX: 300, clientY: 200 });
    fireEvent.pointerMove(img, { pointerId: 1, clientX: 150, clientY: 210 });
    fireEvent.pointerUp(img, { pointerId: 1, clientX: 150, clientY: 210 });
    expect(onIndexChange).not.toHaveBeenCalled();
  });

  it("answers + − 0 on the keyboard", () => {
    const { img } = setup();
    const dialog = screen.getByRole("dialog");
    fireEvent.keyDown(dialog, { key: "+" });
    expect(scaleOf(img)).toBeCloseTo(1.5);
    fireEvent.keyDown(dialog, { key: "-" });
    expect(scaleOf(img)).toBeCloseTo(1);
    fireEvent.keyDown(dialog, { key: "+" });
    fireEvent.keyDown(dialog, { key: "0" });
    expect(scaleOf(img)).toBe(1);
  });
});
