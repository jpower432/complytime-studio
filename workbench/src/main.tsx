// SPDX-License-Identifier: Apache-2.0

import { render } from "preact";
import { App } from "./app";
import { ErrorBoundary } from "./components/error-boundary";
import "./global.css";

render(
  <ErrorBoundary>
    <App />
  </ErrorBoundary>,
  document.getElementById("root")!,
);
