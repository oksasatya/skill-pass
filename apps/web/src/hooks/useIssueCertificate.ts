/**
 * useIssueCertificate — wraps writeContract → waitForTransactionReceipt.
 *
 * Parses the CertificateIssued log from the receipt to extract the tokenId.
 *
 * Status machine: idle → pending (wallet signing) → confirming (tx in mempool) → success | error
 *
 * Exposed: { issue, status, tokenId, txHash, error, reset }
 */

import { useState, useCallback, useEffect } from 'react'
import { useWriteContract, useWaitForTransactionReceipt } from 'wagmi'
import { parseEventLogs } from 'viem'
import { CONTRACT_ADDRESS, CONTRACT_ABI } from '@/lib/contract'
import type { CertificateFields } from '@/lib/validateCertificate'

export type IssueStatus = 'idle' | 'pending' | 'confirming' | 'success' | 'error'

export type UseIssueCertificateResult = {
  readonly issue: (fields: CertificateFields) => void
  readonly status: IssueStatus
  readonly tokenId: bigint | undefined
  readonly txHash: `0x${string}` | undefined
  readonly error: Error | null
  readonly reset: () => void
}

export function useIssueCertificate(): UseIssueCertificateResult {
  const [status, setStatus] = useState<IssueStatus>('idle')
  const [tokenId, setTokenId] = useState<bigint | undefined>(undefined)
  const [txHash, setTxHash] = useState<`0x${string}` | undefined>(undefined)
  const [issueError, setIssueError] = useState<Error | null>(null)

  const { writeContractAsync } = useWriteContract()

  const { data: receipt } = useWaitForTransactionReceipt({
    hash: txHash,
    query: { enabled: !!txHash && status === 'confirming' },
  })

  // When receipt arrives, parse the CertificateIssued log and transition to success
  useEffect(() => {
    if (!receipt || status !== 'confirming') return

    try {
      const logs = parseEventLogs({
        abi: CONTRACT_ABI,
        eventName: 'CertificateIssued',
        logs: receipt.logs,
      })
      const issuedLog = logs[0]
      if (issuedLog) {
        // parseEventLogs with untyped JSON ABI gives args as unknown — narrow manually
        const rawArgs = (issuedLog as unknown as { args: Record<string, unknown> }).args
        const rawTokenId = rawArgs?.['tokenId']
        if (typeof rawTokenId === 'bigint') {
          setTokenId(rawTokenId)
        }
      }
    } catch {
      // Log parse failure is non-fatal; tokenId stays undefined but tx succeeded
    }

    setStatus('success')
  }, [receipt, status])

  const issue = useCallback(
    (fields: CertificateFields) => {
      if (!CONTRACT_ADDRESS) {
        setIssueError(new Error('Contract not configured — set VITE_CONTRACT_ADDRESS.'))
        setStatus('error')
        return
      }

      setStatus('pending')
      setIssueError(null)
      setTokenId(undefined)
      setTxHash(undefined)

      writeContractAsync({
        address: CONTRACT_ADDRESS,
        abi: CONTRACT_ABI,
        functionName: 'issueCertificate',
        args: [
          fields.recipient as `0x${string}`,
          fields.title,
          fields.recipientName,
          fields.issuerName,
          fields.description,
          fields.metadataURI,
        ],
      })
        .then((hash) => {
          setTxHash(hash)
          setStatus('confirming')
        })
        .catch((err: unknown) => {
          setIssueError(err instanceof Error ? err : new Error(String(err)))
          setStatus('error')
        })
    },
    [writeContractAsync],
  )

  const reset = useCallback(() => {
    setStatus('idle')
    setTokenId(undefined)
    setTxHash(undefined)
    setIssueError(null)
  }, [])

  return { issue, status, tokenId, txHash, error: issueError, reset }
}
