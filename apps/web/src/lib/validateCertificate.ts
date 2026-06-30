/**
 * Pure validation helpers for IssueForm.
 * Mirrors the contract MAX_* constants — keep in sync with SkillPassCertificate.sol.
 * TDD: tests in validateCertificate.test.ts written first.
 */

import { isAddress } from 'viem'

// Contract caps (StringTooLong revert triggers if exceeded)
export const FIELD_LIMITS = {
  title: 200,
  recipientName: 100,
  issuerName: 100,
  description: 1000,
  metadataURI: 300,
} as const

export type CertificateFields = {
  readonly recipient: string
  readonly title: string
  readonly recipientName: string
  readonly issuerName: string
  readonly description: string
  readonly metadataURI: string
}

export type FieldErrors = Partial<Record<keyof CertificateFields, string>>

const ZERO_ADDRESS = '0x0000000000000000000000000000000000000000'

/** Validates an Ethereum recipient address. Returns null on pass, error string on fail. */
export function validateRecipient(value: string): string | null {
  if (!value) return 'Recipient address is required.'
  // strict: false → accepts any-case hex; viem's default rejects non-EIP-55 checksums
  if (!isAddress(value, { strict: false })) return 'Enter a valid Ethereum address (0x…).'
  if (value.toLowerCase() === ZERO_ADDRESS) return 'Recipient cannot be the zero address.'
  return null
}

/** Validates a string does not exceed maxLen. Returns null on pass, error string on fail. */
export function validateLength(value: string, maxLen: number): string | null {
  if (value.length > maxLen) return `Must be ${maxLen} characters or fewer (currently ${value.length}).`
  return null
}

/** Validates all IssueForm fields. Returns an object with only the failing fields. */
export function validateFields(fields: CertificateFields): FieldErrors {
  const errors: FieldErrors = {}

  const recipientErr = validateRecipient(fields.recipient)
  if (recipientErr) errors.recipient = recipientErr

  if (!fields.title.trim()) {
    errors.title = 'Certificate title is required.'
  } else {
    const titleErr = validateLength(fields.title, FIELD_LIMITS.title)
    if (titleErr) errors.title = titleErr
  }

  if (!fields.recipientName.trim()) {
    errors.recipientName = 'Recipient name is required.'
  } else {
    const nameErr = validateLength(fields.recipientName, FIELD_LIMITS.recipientName)
    if (nameErr) errors.recipientName = nameErr
  }

  if (!fields.issuerName.trim()) {
    errors.issuerName = 'Issuer name is required.'
  } else {
    const issuerErr = validateLength(fields.issuerName, FIELD_LIMITS.issuerName)
    if (issuerErr) errors.issuerName = issuerErr
  }

  if (!fields.description.trim()) {
    errors.description = 'Description is required.'
  } else {
    const descErr = validateLength(fields.description, FIELD_LIMITS.description)
    if (descErr) errors.description = descErr
  }

  // metadataURI is optional — only validate length if provided
  if (fields.metadataURI) {
    const uriErr = validateLength(fields.metadataURI, FIELD_LIMITS.metadataURI)
    if (uriErr) errors.metadataURI = uriErr
  }

  return errors
}
