// audio.ts — mic capture and PCM playback using Web Audio API.

const SAMPLE_RATE = 16000;

export interface MicCapture {
  stop: () => Promise<void>;
}

/**
 * startMicCapture acquires the user's microphone, runs it through the PCM
 * worklet, and invokes onFrame for each 100ms frame of 16-bit PCM at 16 kHz.
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
  source.connect(node);

  return {
    stop: async () => {
      try {
        node.disconnect();
        source.disconnect();
        stream.getTracks().forEach((t) => t.stop());
        await ctx.close();
      } catch {
        // ignore
      }
    },
  };
}

/**
 * PCMPlayer queues 16 kHz mono PCM int16 frames received from the server and
 * schedules them on a single AudioContext for gapless playback.
 */
export class PCMPlayer {
  private ctx: AudioContext | null = null;
  private nextStart = 0;

  enqueue(frame: ArrayBuffer): void {
    if (!this.ctx) {
      this.ctx = new AudioContext({ sampleRate: SAMPLE_RATE });
      this.nextStart = this.ctx.currentTime;
    }
    const ctx = this.ctx;

    const samples = new Int16Array(frame);
    if (samples.length === 0) return;

    const buffer = ctx.createBuffer(1, samples.length, SAMPLE_RATE);
    const channel = buffer.getChannelData(0);
    for (let i = 0; i < samples.length; i++) {
      const s = samples[i];
      channel[i] = s < 0 ? s / 0x8000 : s / 0x7fff;
    }

    const src = ctx.createBufferSource();
    src.buffer = buffer;
    src.connect(ctx.destination);
    const startAt = Math.max(this.nextStart, ctx.currentTime);
    src.start(startAt);
    this.nextStart = startAt + buffer.duration;
  }

  async close(): Promise<void> {
    if (this.ctx) {
      try {
        await this.ctx.close();
      } catch {
        // ignore
      }
      this.ctx = null;
      this.nextStart = 0;
    }
  }
}
