export const apiHost = import.meta.env.DEV ? "http://localhost:3000" : "";

export function isTauriRuntime(): boolean {
  return (
    typeof window !== "undefined" &&
    typeof (window as Window & { __TAURI_INTERNALS__?: unknown }).__TAURI_INTERNALS__ === "object"
  );
}
