# Product

<!-- impeccable:product-schema 1 -->

## Platform

web

## Users

Primary users are e-commerce merchants, business owners, and media buyers in Laos who run Facebook, Instagram, and WhatsApp Ads. They manage daily product sales, ad spend, customer inquiries, and order deliveries in a multi-currency environment (Lao Kip LAK, Thai Baht THB, US Dollar USD).

## Product Purpose

Meta Super App exists to give Lao merchants a unified, automated command center that bridges advertising performance with real-world business outcomes. It eliminates wasted ad budget, automates routine ad set optimization (pausing losers and scaling winning "angel" ads), and provides instant clarity on customer demographics and buying patterns without requiring complex manual spreadsheets.

## Positioning

The only Lao-first, currency-aware Meta Ads Autopilot & Multi-Channel Commerce Operating System. Unlike generic analytics dashboards or foreign CRM tools, Meta Super App directly ties ad spend to actual Lao Kip order completions, filters out ghost inquiries and accidental clicks (such as Audience Network), and provides 1-click targeting blueprints ready to paste directly into Meta Ads Manager.

## Operating Context

- **Workflow:** Daily campaign monitoring, creative testing (Reels vs Feed), budget scaling, inquiry triage, order confirmation, and warehouse dispatch.
- **Environment:** Desktop browsers for deep analytics and management; tablet and mobile web for on-the-go status checks.
- **Ecosystem:** Meta Graph API (Marketing API, Webhooks, Page Messaging, WhatsApp Business Cloud API), BCEL One / local bank payment confirmations, local logistics/express shipping across Lao provinces.
- **Currencies:** Native support for Lao Kip (LAK), Thai Baht (THB), and US Dollar (USD) with dynamic exchange conversion.

## Capabilities and Constraints

- **Capabilities:**
  - Automated Ads Battle matrix with Angel Ad detection and Bad Ad kill switch.
  - Unified Product Audience Profile with 6-dimension breakdown (Age, Gender, OS Android vs iOS, Placements Reels vs Feeds, Lao Provinces, and 24-Hour Peak Buying Windows).
  - Video retention and creative diagnostics (Hook Rate 3s, Hold Rate ThruPlay, and drop-off funnel).
  - Ad Fatigue & Quality monitor (frequency tracking and Meta Quality Rankings).
  - Dynamic 1-Click Targeting Blueprint generator tailored per selected product.
  - Multi-channel automated reply flows (Facebook Messenger & WhatsApp Business).
  - Integrated Order and Warehouse inventory tracking.
- **Constraints:**
  - Zero manual CSV export/import needed for core media buying decisions.
  - Strict respect for Meta user privacy (demographic data is aggregated distribution, not personal PII).
  - Live data must load efficiently with clear loading and empty states.

## Brand Commitments

- **Tone & Voice:** Professional, direct, task-focused, and supportive.
- **Language:** Lao first (`lo`), with Thai (`th`) and English (`en`) fallbacks. Technical terms in media buying are accompanied by familiar English acronyms (CPA, CTR, ROAS, Reels, Feed, Android, iOS).
- **Aesthetic:** High-contrast, clean, modern, data-dense yet uncluttered dashboard UI that highlights actionable decisions.

## Evidence on Hand

- Production Go (Fiber) backend with Meta Graph API v20.0+ integration in `backend/`.
- Production Nuxt 3 (Vue 3, Vuetify 3, Vite, TypeScript) frontend in `frontend/`.
- Comprehensive product design standard in `DESIGN.md`.
- Live ad account connectors, order repositories, and automated rule engines.

## Product Principles

1. **Live Data Over Speculation:** Every metric, cost calculation, and recommendation is calculated from verified API insight rows and actual orders.
2. **One-Click Actionability:** Never present a problem without an immediate, low-friction solution (e.g. 1-click pause bad ads, 1-click scale +20%, 1-click copy targeting).
3. **Local Market Fidelity:** Interface explicitly supports Lao geography (provinces), local currencies (LAK/THB), and regional user behaviors (evening peak hours, Android dominance).
4. **Restraint Over Noise:** Highlight the critical 20% of metrics that drive 80% of net profit; hide extraneous raw API data behind progressive disclosure.

## Accessibility & Inclusion

- Multi-modal status indicators: Colors (green/amber/red) are always paired with icons and descriptive text labels so meaning is never communicated by color alone.
- Responsive layout adapting gracefully from mobile screens (360px) to ultra-wide desktop monitors (1680px+).
- Visible focus rings, semantic table structures, and clean text readability with natural Lao fonts.
