// blob.ts — animated bar visualizer that tracks an audio level.
//
// Render 32 vertical bars whose heights animate based on a live level (0..1)
// plus a small amount of noise so they feel organic. Color shifts per mode.

export type BlobMode = "idle" | "user" | "assistant";

const NUM_BARS = 32;
const MIN_HEIGHT = 6;

export interface Blob {
  setMode: (mode: BlobMode) => void;
  setGetLevel: (fn: () => number) => void;
  destroy: () => void;
}

export function createBlob(host: HTMLElement): Blob {
  host.classList.add("blob");
  host.replaceChildren();

  const bars: HTMLDivElement[] = [];
  for (let i = 0; i < NUM_BARS; i++) {
    const bar = document.createElement("div");
    bar.className = "blob-bar";
    bar.style.height = `${MIN_HEIGHT}px`;
    host.appendChild(bar);
    bars.push(bar);
  }

  let mode: BlobMode = "idle";
  let getLevel: () => number = () => 0;
  let smoothed = 0;
  let t = 0;
  let raf = 0;
  let cancelled = false;

  setMode("idle");

  function setMode(next: BlobMode): void {
    mode = next;
    host.dataset.mode = next;
  }

  function setGetLevel(fn: () => number): void {
    getLevel = fn;
  }

  function tick(): void {
    if (cancelled) return;
    t += 0.05;
    const raw = mode === "idle" ? 0 : Math.max(0, Math.min(1, getLevel()));
    smoothed += (raw - smoothed) * 0.2;
    const level = smoothed;

    for (let i = 0; i < bars.length; i++) {
      const pos = (i - (NUM_BARS - 1) / 2) / ((NUM_BARS - 1) / 2);
      const centerWeight = Math.exp(-Math.pow(pos * 2.5, 2));
      const phase = i * 0.5;
      const noiseSig =
        Math.sin(t + phase) +
        Math.sin(t * 1.5 - phase * 0.8) +
        Math.sin(t * 0.3 + phase * 1.2);
      const noise = (noiseSig + 3) / 6;
      const dynamicBoost = level * 70 * centerWeight + level * 40 * noise;
      bars[i].style.height = `${MIN_HEIGHT + dynamicBoost}px`;
    }
    raf = requestAnimationFrame(tick);
  }
  raf = requestAnimationFrame(tick);

  return {
    setMode,
    setGetLevel,
    destroy: () => {
      cancelled = true;
      cancelAnimationFrame(raf);
    },
  };
}
