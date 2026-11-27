import { useEffect, useState } from 'react';

// A value that changes at sensor rate (60+ Hz). Canvas code subscribes and
// redraws on the next animation frame; React UI reads it through
// useThrottledStore so a vision packet does not re-render the whole tree.
export type LiveStore<T> = {
  get: () => T;
  set: (value: T) => void;
  subscribe: (listener: () => void) => () => void;
};

export function createLiveStore<T>(initial: T): LiveStore<T> {
  let value = initial;
  const listeners = new Set<() => void>();
  return {
    get: () => value,
    set: (next) => {
      if (Object.is(next, value)) return;
      value = next;
      listeners.forEach((listener) => listener());
    },
    subscribe: (listener) => {
      listeners.add(listener);
      return () => listeners.delete(listener);
    },
  };
}

// The store's value, re-rendering at most once per intervalMs.
export function useThrottledStore<T>(store: LiveStore<T>, intervalMs: number): T {
  const [value, setValue] = useState(store.get);
  useEffect(() => {
    let timer: ReturnType<typeof setTimeout> | null = null;
    setValue(store.get());
    const unsubscribe = store.subscribe(() => {
      if (timer !== null) return;
      timer = setTimeout(() => {
        timer = null;
        setValue(store.get());
      }, intervalMs);
    });
    return () => {
      unsubscribe();
      if (timer !== null) clearTimeout(timer);
    };
  }, [store, intervalMs]);
  return value;
}
