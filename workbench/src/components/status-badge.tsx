// SPDX-License-Identifier: Apache-2.0
const labels: Record<string, string> = {
  submitted: "Submitted", working: "Working", "input-required": "Needs Input",
  ready: "Ready", accepted: "Accepted", cancelled: "Cancelled",
  completed: "Completed", failed: "Failed", disconnected: "Disconnected",
};
export function StatusBadge({ status }: { status: string }) {
  return <span class={`badge badge-${status}`}>{labels[status] || status}</span>;
}
