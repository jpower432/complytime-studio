# Transparency Ledger for Certification Activities

**Status:** Exploratory

## Context

Evidence certification verdicts (schema validity, provenance, attestation integrity) are stored in a `certifications` table in ClickHouse. This table is operational metadata — it answers "what did the certifiers say?" for the UI and the agent. It does not provide tamper-evidence, cryptographic ordering, or independent verifiability.

ClickHouse is a mutable store. The same credentials that write evidence can rewrite certification rows. There is no structural guarantee that historical verdicts are preserved unaltered.

For a compliance platform, this gap matters. Auditors may need to verify that certification happened at a claimed time and was not retroactively modified.

## Problem

A `certifications` table in the application database is not a ledger. It lacks:

- **Append-only enforcement** — ClickHouse supports `ALTER DELETE` and mutations
- **Tamper evidence** — no hash chain between entries
- **Independent trust boundary** — same database, same credentials, same blast radius
- **External verifiability** — no third party can audit without trusting the DB operator

## Candidate: Trillian

[Trillian](https://github.com/google/trillian) is a transparent, append-only log backed by a Merkle tree. It provides cryptographic proof that:

- An entry was included in the log at a specific index
- The log has not been retroactively modified (consistency proofs)
- Any observer can verify inclusion without trusting the log operator

Trillian is the backend for Certificate Transparency (CT) logs, Sigstore Rekor, and Go module checksum databases.

| Property | ClickHouse table | Trillian log |
|---|---|---|
| Append-only | No | Yes (Merkle tree) |
| Tamper-evident | No | Yes (consistency proofs) |
| Independent trust | No | Yes (separate service, separate keys) |
| Cryptographic ordering | No | Yes (tree head signatures) |
| External verifiability | No | Yes (monitors/auditors can verify) |
| Query performance | High (OLAP) | Low (log, not a query engine) |

## Architecture if adopted

```
Evidence ingested
       │
       ▼
CertificationHandler
       │
       ├── INSERT to certifications table (ClickHouse, query surface)
       │
       └── Append to Trillian log (proof surface)
               │
               ▼
         Signed tree head (STH)
         Inclusion proof per entry
```

ClickHouse remains the query engine for the UI and agent. Trillian is the proof engine for auditors. Dual-write, independent trust boundaries.

Certification rows in ClickHouse gain a `log_index` column referencing the Trillian entry. Auditors can verify any certification verdict by requesting the inclusion proof from Trillian and checking it against the signed tree head.

## Alternatives

| Alternative | Trade-off |
|---|---|
| **Sigstore Rekor** | Hosted transparency log. Avoids running Trillian. Designed for software supply chain attestations. May not fit arbitrary certification verdicts without adaptation. |
| **S3 Object Lock** | Append-only at the storage layer. No Merkle tree, no inclusion proofs. Simpler but weaker guarantees. |
| **Application-layer hash chain** | Each row includes `hash(previous_row)`. Detectable tampering but no external witness. No independent verifiability. |
| **Do nothing** | Accept that the certifications table is operational, not evidentiary. Sufficient if auditors trust the platform operator. |

## Open Questions

- What is the deployment cost of Trillian (requires a backing database — MySQL or CockroachDB)?
- Is Rekor a better fit given the project's existing alignment with Sigstore for attestation bundles?
- What certification activities should be logged — only evidence certification, or also policy imports, assessment verdicts, audit log generation?
- What is the retention policy for the transparency log vs. ClickHouse?
- Who are the external verifiers / monitors in the compliance context?

## Decision

Deferred. The `certifications` table in ClickHouse is sufficient for the current scope. When external auditability becomes a requirement, revisit this document and evaluate Trillian or Rekor as the proof layer alongside ClickHouse as the query layer.
