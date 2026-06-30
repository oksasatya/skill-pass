/**
 * TDD: validateCertificate — tests written before implementation.
 *
 * Pure validation functions for IssueForm:
 *   - validateRecipient: must be a valid EVM address
 *   - validateLength: must not exceed the contract MAX_* caps
 *   - FIELD_LIMITS: exported caps mirroring the contract constants
 */

import { describe, it, expect } from 'vitest'
import {
  FIELD_LIMITS,
  validateRecipient,
  validateLength,
  validateFields,
} from './validateCertificate'

describe('FIELD_LIMITS', () => {
  it('exports the correct contract-mirrored caps', () => {
    expect(FIELD_LIMITS.title).toBe(200)
    expect(FIELD_LIMITS.recipientName).toBe(100)
    expect(FIELD_LIMITS.issuerName).toBe(100)
    expect(FIELD_LIMITS.description).toBe(1000)
    expect(FIELD_LIMITS.metadataURI).toBe(300)
  })
})

describe('validateRecipient', () => {
  it('returns null for a valid lowercase 0x address', () => {
    expect(validateRecipient('0x1234567890abcdef1234567890abcdef12345678')).toBeNull()
  })

  it('returns null for a checksummed EIP-55 address', () => {
    expect(validateRecipient('0xAbCdEf1234567890AbCdEf1234567890AbCdEf12')).toBeNull()
  })

  it('returns an error for an empty string', () => {
    expect(validateRecipient('')).not.toBeNull()
  })

  it('returns an error for a zero address', () => {
    expect(validateRecipient('0x0000000000000000000000000000000000000000')).not.toBeNull()
  })

  it('returns an error for an invalid address (too short)', () => {
    expect(validateRecipient('0x1234')).not.toBeNull()
  })

  it('returns an error for a non-hex address', () => {
    expect(validateRecipient('not-an-address')).not.toBeNull()
  })

  it('returns an error for an address missing the 0x prefix', () => {
    expect(validateRecipient('1234567890abcdef1234567890abcdef12345678')).not.toBeNull()
  })
})

describe('validateLength', () => {
  it('returns null when value is within the limit', () => {
    expect(validateLength('hello', 10)).toBeNull()
  })

  it('returns null when value equals the limit', () => {
    expect(validateLength('a'.repeat(100), 100)).toBeNull()
  })

  it('returns an error when value exceeds the limit by 1', () => {
    expect(validateLength('a'.repeat(101), 100)).not.toBeNull()
  })

  it('includes the limit in the error message', () => {
    const err = validateLength('a'.repeat(201), 200)
    expect(err).toContain('200')
  })

  it('returns null for an empty string', () => {
    expect(validateLength('', 100)).toBeNull()
  })
})

describe('validateFields', () => {
  const validFields = {
    recipient: '0x1234567890abcdef1234567890abcdef12345678' as `0x${string}`,
    title: 'Go Bootcamp Certificate',
    recipientName: 'Alice',
    issuerName: 'Hacktiv8',
    description: 'Completed the Go backend bootcamp.',
    metadataURI: '',
  }

  it('returns no errors for valid fields', () => {
    const errors = validateFields(validFields)
    expect(Object.keys(errors)).toHaveLength(0)
  })

  it('returns recipient error for zero address', () => {
    const errors = validateFields({
      ...validFields,
      recipient: '0x0000000000000000000000000000000000000000' as `0x${string}`,
    })
    expect(errors.recipient).toBeDefined()
  })

  it('returns title error when title exceeds 200 chars', () => {
    const errors = validateFields({ ...validFields, title: 'a'.repeat(201) })
    expect(errors.title).toBeDefined()
  })

  it('returns title error for empty title', () => {
    const errors = validateFields({ ...validFields, title: '' })
    expect(errors.title).toBeDefined()
  })

  it('returns recipientName error when too long', () => {
    const errors = validateFields({ ...validFields, recipientName: 'a'.repeat(101) })
    expect(errors.recipientName).toBeDefined()
  })

  it('returns issuerName error when too long', () => {
    const errors = validateFields({ ...validFields, issuerName: 'a'.repeat(101) })
    expect(errors.issuerName).toBeDefined()
  })

  it('returns description error when exceeds 1000 chars', () => {
    const errors = validateFields({ ...validFields, description: 'a'.repeat(1001) })
    expect(errors.description).toBeDefined()
  })

  it('returns metadataURI error when exceeds 300 chars', () => {
    const errors = validateFields({ ...validFields, metadataURI: 'https://'.padEnd(301, 'x') })
    expect(errors.metadataURI).toBeDefined()
  })

  it('allows empty metadataURI (optional field)', () => {
    const errors = validateFields({ ...validFields, metadataURI: '' })
    expect(errors.metadataURI).toBeUndefined()
  })

  it('returns multiple errors when multiple fields are invalid', () => {
    const errors = validateFields({
      ...validFields,
      title: '',
      recipientName: 'a'.repeat(101),
    })
    expect(errors.title).toBeDefined()
    expect(errors.recipientName).toBeDefined()
  })
})
