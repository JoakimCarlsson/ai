// audio.ts — mic capture and PCM playback using Web Audio API.
//
// Exposes RMS level() on both the mic capture and the PCM player so the UI
// can drive its blob animation off real audio energy.

const SAMPLE_RATE = 16000;

export interface MicCapture {
  stop: () => Promise<void>;
  level: () => number;
}

/**
 * startMicCapture acquires the user's microphone, runs it through the PCM
 * worklet, and invokes onFrame for each 100ms frame of 16-bit PCM at 16 kHz.
 * The returned level() reports RMS energy of the live mic stream (0..1).
 */
export async function startMicCapture(
  onFrame: (frame: ArrayBuffer) => void,
): Promise<MicCapture> {
  const stream = await navigator.mediaDevices.getUserMedia({
    audio: {
      channelCount: 1,
      echoCancellation: true,
      noiseSuppression: true,
      autoGainControl: true,
    },
  });

  const ctx = new AudioContext();
  await ctx.audioWorklet.addModule("/pcm-worklet.js");

  const source = ctx.createMediaStreamSource(stream);
  const node = new AudioWorkletNode(ctx, "pcm-processor");
  node.port.onmessage = (event) => {
    onFrame(event.data as ArrayBuffer);
  };

  const analyser = ctx.createAnalyser();
  analyser.fftSize = 1024;
  analyser.smoothingTimeConstant = 0.7;

  source.connect(node);
  source.connect(analyser);

  return {
    stop: async () => {
      try {
        node.disconnect();
        source.disconnect();
        analyser.disconnect();
        stream.getTracks().forEach((t) => t.stop());
        await ctx.close();
      } catch {
        // ignore
      }
    },
    level: () => analyserRMS(analyser),
  };
}

/**
 * PCMPlayer queues 16 kHz mono PCM int16 frames received from the server and
 * schedules them on a single AudioContext for gapless playback. Audio is
 * routed through an analyser so level() can drive a UI animation.
 */
export class PCMPlayer {
  private ctx: AudioContext | null = null;
  private analyser: AnalyserNode | null = null;
  private nextStart = 0;

  enqueue(frame: ArrayBuffer): void {
    const samples = new Int16Array(frame);
    if (samples.length === 0) return;

    if (!this.ctx || !this.analyser) {
      this.ctx = new AudioContext({ sampleRate: SAMPLE_RATE });
      this.analyser = this.ctx.createAnalyser();
      this.analyser.fftSize = 1024;
      this.analyser.smoothingTimeConstant = 0.7;
      this.analyser.connect(this.ctx.destination);
      this.nextStart = this.ctx.currentTime;
    }
    const ctx = this.ctx;
    const analyser = this.analyser;

    const buffer = ctx.createBuffer(1, samples.length, SAMPLE_RATE);
    const channel = buffer.getChannelData(0);
    for (let i = 0; i < samples.length; i++) {
      const s = samples[i];
      channel[i] = s < 0 ? s / 0x8000 : s / 0x7fff;
    }

    const src = ctx.createBufferSource();
    src.buffer = buffer;
    src.connect(analyser);
    const startAt = Math.max(this.nextStart, ctx.currentTime);
    src.start(startAt);
    this.nextStart = startAt + buffer.duration;
  }

  level(): number {
    return this.analyser ? analyserRMS(this.analyser) : 0;
  }

  remainingTime(): number {
    if (!this.ctx) return 0;
    return Math.max(0, this.nextStart - this.ctx.currentTime);
  }

  async close(): Promise<void> {
    if (this.ctx) {
      try {
        await this.ctx.close();
      } catch {
        // ignore
      }
      this.ctx = null;
      this.analyser = null;
      this.nextStart = 0;
    }
  }

  /**
   * flush stops any currently scheduled audio immediately by closing the
   * AudioContext. The next enqueue() reinitializes a fresh context so
   * subsequent audio plays normally. Used to drop queued audio on barge-in.
   */
  async flush(): Promise<void> {
    await this.close();
  }
}

function analyserRMS(a: AnalyserNode, gain = 2.5): number {
  const buf = new Float32Array(a.fftSize);
  a.getFloatTimeDomainData(buf);
  let sum = 0;
  for (let i = 0; i < buf.length; i++) {
    sum += buf[i] * buf[i];
  }
  const rms = Math.sqrt(sum / buf.length);
  return Math.min(1, rms * gain);
}
