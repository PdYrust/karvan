import { act, useEffect } from 'react';
import { createRoot, type Root } from 'react-dom/client';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { useSubToggle } from './use-sub-toggle';

type Toggle = ReturnType<typeof useSubToggle>;

let root: Root;
let host: HTMLDivElement;
let current: Toggle;

function Harness({ onRender }: { onRender: (t: Toggle) => void }) {
  const toggle = useSubToggle();
  useEffect(() => {
    onRender(toggle);
  });
  return null;
}

beforeEach(() => {
  vi.useFakeTimers();
  host = document.createElement('div');
  root = createRoot(host);
  act(() =>
    root.render(
      <Harness
        onRender={(t) => {
          current = t;
        }}
      />,
    ),
  );
});

afterEach(() => {
  act(() => root.unmount());
  vi.useRealTimers();
});

/** A press on the trigger as the browser sends it: pointer down, then click. */
function press(menuReacts?: () => void) {
  act(() => current.trigger.onPointerDownCapture());
  act(() => current.trigger.onClickCapture());
  if (menuReacts) act(menuReacts);
  act(() => vi.runAllTimers());
}

const open = (reason: string) => () => current.root.onOpenChange(true, { reason });

// React's act() needs this flag outside a testing library.
(globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }).IS_REACT_ACT_ENVIRONMENT = true;

describe('useSubToggle', () => {
  it('closes on a second press after a press opened it', () => {
    press(open('trigger-press'));
    expect(current.root.open).toBe(true);
    press(open('trigger-press'));
    expect(current.root.open).toBe(false);
  });

  it('treats a hover opening right after a press as the answer to it', () => {
    press();
    act(open('trigger-hover'));
    expect(current.root.open).toBe(true);
    press();
    expect(current.root.open).toBe(false);
  });

  it('keeps a submenu the pointer opened on the first press, and closes it on the next', () => {
    act(() => vi.advanceTimersByTime(1000));
    act(open('trigger-hover'));
    press();
    expect(current.root.open).toBe(true);
    press();
    expect(current.root.open).toBe(false);
  });

  it('opens again on the press after it closed', () => {
    press(open('trigger-press'));
    press(open('trigger-press'));
    press(open('trigger-press'));
    expect(current.root.open).toBe(true);
  });

  it('follows the menu when it closes the submenu itself', () => {
    press(open('trigger-press'));
    act(() => current.root.onOpenChange(false, { reason: 'outside-press' }));
    expect(current.root.open).toBe(false);
  });
});
