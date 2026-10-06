const BEEP_DURATION_MS = 150;
const BEEP_FREQUENCY_HZ = 880;

let context: AudioContext | null = null;

function unlock(): void {
  context ??= new AudioContext();
}

export function armBeepOnFirstGesture(): () => void {
  window.addEventListener('click', unlock, { once: true });
  return () => window.removeEventListener('click', unlock);
}

export function beep(): void {
  if (!context) return;
  const oscillator = context.createOscillator();
  const gain = context.createGain();
  oscillator.frequency.value = BEEP_FREQUENCY_HZ;
  gain.gain.value = 0.2;
  oscillator.connect(gain);
  gain.connect(context.destination);
  oscillator.start();
  oscillator.stop(context.currentTime + BEEP_DURATION_MS / 1000);
}
