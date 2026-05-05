# View Consolidation

## Summary

Restructure the ComplyTime Studio UI around a compliance program model, replacing the flat posture/policies/evidence navigation with a program-centric workflow: Dashboard, Programs, Policies, Inventory, Evidence, Reviews.

## Motivation

The current UI is policy-centric — users see posture cards per policy but lack a higher-order grouping for regulatory obligations (FedRAMP, PCI-DSS, ISO 27001). Compliance analysts manage **programs** that bundle multiple policies under a single framework with shared applicability and lifecycle. Without this abstraction, the UI cannot answer "How compliant is my FedRAMP Moderate program?" without mental aggregation.

## Goals

1. Programs as the top-level organizational unit for compliance obligations
2. Unified import for all Gemara artifact types (policy bundles, guidance, mappings)
3. Policy recommendation engine using mapping overlap and evidence quality
4. Standalone Inventory view for target-level compliance visibility
5. Cross-program Reviews queue replacing the inbox-centric review model
6. Stripe-inspired design tokens for professional aesthetics
7. Writer role for content management without admin privileges

## Non-Goals

- Multi-tenant program isolation (single-tenant for now)
- Automated program creation from regulatory feeds
- Real-time collaboration on review edits
