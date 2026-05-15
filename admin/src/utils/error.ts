export function getErrorMessage(err: unknown, fallback = 'An error occurred'): string {
  const error = err as { response?: { data?: { error?: string } }; message?: string }
  return error.response?.data?.error || error.message || fallback
}
