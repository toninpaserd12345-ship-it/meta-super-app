---
target: ads-battle
total_score: 21
max_score: 40
na_heuristics: 
p0_count: 1
p1_count: 2
target_identity: "file:/Users/tonxayinpaserd/Documents/ChatGPT/META-SUPPER-APP/frontend/pages/ads-battle.vue"
target_fingerprint: "sha256:5f896d2c0331877b1729abb515eefcc7df6329fcdaf91af580818c90306d0aab"
target_path: /Users/tonxayinpaserd/Documents/ChatGPT/META-SUPPER-APP/frontend/pages/ads-battle.vue
timestamp: 2026-10-20T09-22-01Z
slug: frontend-pages-ads-battle-vue
---
Method: dual-agent (A: f108d4cd-71ec-4b01-8d31-fd9b8a9b7399 · B: efedf103-ba81-44fd-ad5d-e88b13cadb04)

### Design Health Score

| # | Heuristic | Score | Key Issue |
|---|-----------|:-----:|-----------|
| 1 | Visibility of System Status | 2 | Page loads with a generic spinner instead of skeletons; bulk pausing freezes individual cards without row-level progress indicators. |
| 2 | Match Between System and Real World | 3 | Strong Lao market terminology (`ໂມງຄຳ`, `ແອດນາງຟ້າ`, `COD`); compromised by AI advice quoting USD spend (`$1 - $3`) and hardcoding "H9 MAX". |
| 3 | User Control and Freedom | 2 | No "Undo" for paused ads or budget scaling; no "Clear filters" button in zero-result states; unstyled browser `confirm()` dialogs trap the user. |
| 4 | Consistency and Standards | 2 | Inconsistent button sizing; mix of Vuetify components and custom HTML buttons; trilingual string localization applied haphazardly across modules. |
| 5 | Error Prevention | 1 | Daily budget estimated via `spend / days` rather than fetched from live Ad Set settings; no sanity check on minimum Meta budget limits; no confirmation breakdown on bulk pausing. |
| 6 | Recognition Rather Than Recall | 2 | Creative thumbnails and format tags aid visual recognition; however, hidden algorithmic rules defining "Angel" vs "Bad" ad force users to guess why an ad was categorized. |
| 7 | Flexibility and Efficiency | 3 | 1-click scale, 1-click pause, and table view cater to power users; however, table view lacks multi-select checkboxes and column sorting. |
| 8 | Aesthetic and Minimalist Design | 1 | Severe visual clutter: 15 sub-elements per card; Placement Battle card clogs viewport; ironic "Simple Design" tag in page header. |
| 9 | Error Recovery | 2 | Errors are trapped in generic warning banners without actionable guidance (e.g., token expired, missing permissions, or ad account disabled). |
| 10 | Help and Documentation | 3 | Direct link to Daily SOP (`/ads-playbook`); in-depth advisory guidance; tooltips rely on inaccessible HTML `title` attributes that fail on touch devices. |
| **Total** | | **21/40** | **Substantial Usability & Safety Deficit** |

---

### Design Specificity Verdict

**Verdict:** *Authentic Local Intent, But Structurally Overloaded & Operationally Risky.*

- **LLM Assessment:** The interface displays admirable *domain intent* for the Lao e-commerce landscape (LAK/THB/USD multi-currency zero-decimal awareness, Lao provincial delivery hubs, evening buying peaks 19:00–23:30, and COD closing scripts). However, it suffers from severe cognitive overload (15 distinct sub-widgets per ad card), a missing "Real Buyers" view promised in navigation, and hardcoded single-product references ("H9 MAX").
- **Deterministic Scan:** Impeccable CLI regex matching found 0 anti-patterns, but a full DOM/CSS cascade scan revealed **104 mechanical quality findings** (63 undersized/tiny text declarations `< 12px` reaching down to `8.5px`, 20 low-contrast instances under 4.5:1, 6 nested cards, 1 skipped heading `h1` -> `h3`, and 47 emojis standing in for real icons).
- **Visual Overlays:** Dev server was offline during scan; mechanical evidence was compiled via static AST & cascade evaluation.

---

### Overall Impression
A powerful, feature-rich command center with genuine local market empathy that currently tries to do everything at once. It forces the user to digest placement battles, video hook rates, ad fatigue diagnostics, and 9 filter controls simultaneously before taking action on their winning or losing ads.

---

### What's Working Well
1. **Visual Creative-to-Commerce Association:** Real ad creative thumbnails alongside format badges (Reels vs Feed) and linked product badges allow merchants to immediately identify winning or failing creatives without memorizing cryptic Meta Ad IDs.
2. **Deep Local Market Nuance in Diagnostics:** Strategic advice directly addresses real-world Lao e-commerce realities: distinguishing between 11:00–14:00 and 19:00–23:30 peak buying windows, warning against mobile game accidental clicks on Audience Network, and offering natural Lao COD closing scripts.
3. **Multi-Perspective View Switching:** Allowing merchants to alternate between visual Cards (for creative triage), a dense Data Table (for analytical scanning), and an Audience Profile (for demographic strategy) accommodates both quick checks and deep analysis.

---

### Priority Issues

#### [P0] Hardcoded Synthetic Demographics Masquerading as "100% Real Meta Data"
- **Why it matters:** `activeDemographics` calculates provincial splits (58% Vientiane), age groups, and 72% Android devices using hardcoded multipliers while the UI displays *"Meta Ads Insights ແທ້ 100%"*. Merchants risk allocating real advertising capital based on synthetic heuristics.
- **Fix:** Bind the Demographics view strictly to `demographicsReport.value` from the API. When Meta API reports insufficient sample size, show an honest empty/threshold state rather than fabricating numbers.
- **Suggested command:** `/impeccable harden`

#### [P1] Dangerous Daily Budget Scaling Estimation
- **Why it matters:** Estimating current daily budget as `spend / days` in `openScaleModal()` produces incorrect baselines for paused, recently launched, or unevenly pacing campaigns, risking catastrophic misbudgeting on live Meta ad sets.
- **Fix:** Retrieve the actual `daily_budget` directly from the Meta Ad Set Graph API record, enforce currency-aware minimum spend rules, and display the exact before/after monthly projection.
- **Suggested command:** `/impeccable optimize`

#### [P1] Extreme Card Visual Density (15+ Elements per Card)
- **Why it matters:** Stacking switches, badges, headlines, copywriting snippets, copy buttons, external links, 5 metrics, fatigue pills, hook rates, placement badges, and sparklines in a 360px card creates severe cognitive overload and decision paralysis.
- **Fix:** Apply aggressive progressive disclosure: retain only Identity, Status, the Core Profit Trio (Spend, Orders, Ad Net Profit), and 1 primary action. Relocate placement breakdowns, ad copy, and quality diagnostics into the preview drawer.
- **Suggested command:** `/impeccable distill`

#### [P2] Undersized Typography & Sub-12px Readability Deficit
- **Why it matters:** Over 40 CSS font-size declarations drop below the 12px readability floor (reaching as low as 8.5px and 9px for hours, badges, and metrics), causing extreme strain on mobile screens under Southeast Asian daylight glare.
- **Fix:** Normalize all secondary labels to a minimum of 12px (`--text-xs`), eliminate sub-10px micro-text, and increase touch targets to 44×44px.
- **Suggested command:** `/impeccable typeset`

#### [P2] Orphaned "Real Buyers" Feature & Ghost Navigation
- **Why it matters:** The view toggle claims `"👥 ລູກຄ້າຕົວຈິງ & Demographics"` and code compiles `filteredRealOrders`, but the template lacks the buyer table and sub-navigation entirely, hiding verified customer data from the user.
- **Fix:** Implement the sub-tab toggle (`demographics` vs `buyers`) in View 3, displaying the real order roster with phone numbers, provinces, order values, and an explicit Meta Custom Audience CSV export button.
- **Suggested command:** `/impeccable layout`

---

### Persona Red Flags

- **Somchai (Lao Agency Media Buyer, 20+ ads daily):** No multi-select checkboxes or column sorting in Table View; pausing 10 losing ads requires 10 individual clicks through modal prompts.
- **Noy (Small Retail Shop Owner in Vientiane, beginner):** Overwhelmed by 15+ concurrent metrics per card; intimidated by the ironic "Simple Design" badge; terrified of clicking "Scale (+20%)" because she cannot see the monthly bank deduction preview.
- **Alex (Cross-Border Buyer, USD/THB Accounts):** Static hardcoded currency conversion rates (22,000 LAK/USD, 620 LAK/THB) drift from actual bank settlement rates; AI advice quotes spend in USD even for LAK ad accounts.

---

### Minor Observations
1. **Duplicate closing tags:** Lines 2306–2307 contain duplicate closing tags (`</div></div>`).
2. **Native blocking modals:** `alert()` and `confirm()` freeze the browser thread and look unstyled on mobile web.
3. **Broken mobile hover:** Creative media overlay depends on `:hover`, which fails on touchscreens.
4. **Emoji as icons:** 47 distinct Unicode emojis are used as UI icons instead of consistent SVG/Material Design icons.

---

### Questions to Consider
1. *Why is the Placement Battle summary occupying the most valuable vertical screen space above the ads, when 90% of a merchant's daily workflow is simply identifying losing ad sets to pause and scaling winners?*
2. *If demographic metrics (age, gender, OS, provinces) are derived from hardcoded heuristic weights rather than live Meta breakdown data, does this feature empower merchants or give them false confidence based on fabricated numbers?*
3. *Why does the page label itself "Simple Design" in an H1 badge while exposing 15 data points per card, 9 filter controls, 3 view modes, and a 6-tab modal? Would true simplicity mean displaying fewer metrics with higher decision confidence?*
