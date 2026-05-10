// pcm-worklet.js
//
// AudioWorkletProcessor that captures mic input at the AudioContext's native
// sample rate (typically 48 kHz), downsamples to 16 kHz mono, converts to
// signed 16-bit little-endian PCM, and ships it to the main thread as
// transferable ArrayBuffers.
//
// The downsample uses simple decimation. For higher fidelity speech, add a
// low-pass before the decimation. For STT this is fine in practice.

const TARGET_SAMPLE_RATE = 16000;
const FRAME_MS = 100;
const SAMPLES_PER_FRAME = (TARGET_SAMPLE_RATE * FRAME_MS) / 1000; // 1600

class PCMProcessor extends AudioWorkletProcessor {
  constructor() {
    super();
    this.ratio = sampleRate / TARGET_SAMPLE_RATE;
    this.buffer = new Int16Array(SAMPLES_PER_FRAME);
    this.bufferIndex = 0;
    this.sampleCarry = 0;
  }

  process(inputs) {
    const input = inputs[0];
    if (!input || input.length === 0) {
      return true;
    }
    const channel = input[0];
    if (!channel) {
      return true;
    }

    for (let i = 0; i < channel.length; i++) {
      this.sampleCarry += 1;
      if (this.sampleCarry < this.ratio) {
        continue;
      }
      this.sampleCarry -= this.ratio;

      let sample = channel[i];
      if (sample > 1) sample = 1;
      if (sample < -1) sample = -1;
      this.buffer[this.bufferIndex++] = sample < 0
        ? Math.round(sample * 0x8000)
        : Math.round(sample * 0x7fff);

      if (this.bufferIndex >= SAMPLES_PER_FRAME) {
        const out = new ArrayBuffer(this.buffer.byteLength);
        new Int16Array(out).set(this.buffer);
        this.port.postMessage(out, [out]);
        this.bufferIndex = 0;
      }
    }
    return true;
  }
}

registerProcessor("pcm-processor", PCMProcessor);
