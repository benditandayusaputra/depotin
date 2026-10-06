export function pollWhileVisible(intervalMs: number, tick: () => void): () => void {
  let timer: ReturnType<typeof setInterval> | undefined;

  function sync() {
    const visible = document.visibilityState === 'visible';
    if (visible && timer === undefined) timer = setInterval(tick, intervalMs);
    if (!visible && timer !== undefined) {
      clearInterval(timer);
      timer = undefined;
    }
  }

  sync();
  document.addEventListener('visibilitychange', sync);
  return () => {
    clearInterval(timer);
    document.removeEventListener('visibilitychange', sync);
  };
}
