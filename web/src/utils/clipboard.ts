/**
 * Safe clipboard helper with fallback for non-secure HTTP contexts
 */
export async function copyText(text: string): Promise<boolean> {
  if (!text) return true

  // Try modern navigator.clipboard first (requires HTTPS or localhost)
  if (typeof navigator !== 'undefined' && navigator.clipboard && typeof navigator.clipboard.writeText === 'function') {
    try {
      await navigator.clipboard.writeText(text)
      return true
    } catch {
      // Fall through to execCommand
    }
  }

  // Fallback for HTTP / older browsers
  try {
    const textArea = document.createElement('textarea')
    textArea.value = text
    textArea.style.position = 'fixed'
    textArea.style.left = '-9999px'
    textArea.style.top = '-9999px'
    textArea.setAttribute('readonly', '')
    document.body.appendChild(textArea)
    textArea.select()
    const successful = document.execCommand('copy')
    document.body.removeChild(textArea)
    return successful
  } catch {
    return false
  }
}

export async function readText(): Promise<string> {
  if (typeof navigator !== 'undefined' && navigator.clipboard && typeof navigator.clipboard.readText === 'function') {
    return await navigator.clipboard.readText()
  }
  throw new Error('Clipboard reading is not supported in this browser/environment')
}
