/**
 * truncateAddress: "0x1234567890...abcdef" → "0x1234…cdef"
 * First 6 chars (0x + 4 hex) + ellipsis + last 4 chars.
 * Returns the address unchanged if it is too short to truncate (< 11 chars).
 */
export function truncateAddress(addr: `0x${string}`): string {
  // ponytail: < 11 = "0x" + fewer than 9 hex chars — no room for first6 + last4 overlap
  if (addr.length < 11) return addr
  return `${addr.slice(0, 6)}…${addr.slice(-4)}`
}
