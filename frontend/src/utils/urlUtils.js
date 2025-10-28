/**
 * Normalizes a URL by ensuring it has a protocol
 * @param {string} url - The URL to normalize
 * @returns {string} - The normalized URL
 */
export const normalizeUrl = (url) => {
  if (!url) return ''

  const trimmedUrl = url.trim()

  // If it already has a protocol, return as-is
  if (trimmedUrl.startsWith('http://') || trimmedUrl.startsWith('https://')) {
    return trimmedUrl
  }

  // Otherwise, prepend https://
  return `https://${trimmedUrl}`
}

/**
 * Validates if a string is a valid URL
 * @param {string} url - The URL to validate
 * @returns {boolean} - True if valid URL
 */
export const isValidUrl = (url) => {
  try {
    new URL(normalizeUrl(url))
    return true
  } catch (e) {
    return false
  }
}
