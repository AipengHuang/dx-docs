const LEGACY_PRODUCT_NAME_RE = /weknora/gi

/**
 * Final user-visible brand guard for generated or persisted assistant text.
 * Prompt rules remain the primary control; this protects streaming output,
 * restored history, copy actions, and embedded chat from model regressions.
 */
export function sanitizeUserVisibleBrandText(value: unknown): string {
  const text = typeof value === 'string' ? value : String(value || '')
  return text.replace(LEGACY_PRODUCT_NAME_RE, '帝显')
}
