// SPDX-License-Identifier: Apache-2.0

import type { JSX } from "preact";

export function cardKeyHandler(callback: () => void): JSX.KeyboardEventHandler<HTMLElement> {
  return (e) => {
    if (e.key === "Enter" || e.key === " ") {
      e.preventDefault();
      callback();
    }
  };
}
