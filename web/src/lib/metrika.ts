// Metrika goals. Each name is registered in the counter's settings as a
// "JavaScript event" goal; a name that is not registered there is dropped.
// ym itself is defined by public/analytics.js, and only on fxtun.ru.
const COUNTER = 108256538

export function reachGoal(name: 'signup' | 'copy_command') {
  if (typeof window !== 'undefined' && typeof window.ym === 'function') {
    window.ym(COUNTER, 'reachGoal', name)
  }
}

// Sign-up happens three ways (password, magic link, OAuth) and only the first
// one says so. An account this young when it first reaches the app was just
// created, whichever way it came in.
// ponytail: age heuristic; an explicit is_new flag from the API if it misfires.
const NEW_ACCOUNT_MS = 10 * 60 * 1000

export function isNewAccount(createdAt: string, now = Date.now()) {
  return now - Date.parse(createdAt) < NEW_ACCOUNT_MS
}
