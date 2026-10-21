# UI audit — Omnichannel Workbench

Audit scope: all 24 Nuxt routes, shared app shell, cards, tables, forms, dialogs, empty/loading/error states, icons, responsive navigation, and operational dashboards.

## Design decision

The product now follows one enterprise operations language: neutral work surfaces, a dark navigation rail, indigo primary actions, semantic status colors, compact data density, 44 px interaction targets, and Noto Sans Lao/Thai localization. Decorative gradients, glass effects, floating card stacks, rainbow icon tiles, oversized labels, and ornamental emoji are not part of the product language.

## Route coverage

| Area | Routes reviewed | Result |
| --- | --- | --- |
| Shell and access | `/`, `/login`, dynamic section route | Unified navigation, hierarchy, focus, and responsive behavior |
| Meta connection | `/meta-pages`, OAuth loading/completion | Clear connection states, real Page imagery, accessible actions |
| Messaging | `/live-chat`, `/replies`, `/replies/:id`, `/auto-replies`, `/automation-guide` | Operational layouts, consistent sequence editor and previews |
| Commerce | `/products`, `/orders`, `/warehouses`, `/leads` | Dense data views, visible primary actions, standardized status treatment |
| Ads | `/ads`, `/ads-battle`, `/ads-intelligence`, `/ads-playbook` | Unified KPI strip, filters, tables, decision states, and detail panels |
| Settings | `/settings`, `/settings/general`, `/settings/team`, `/settings/notifications`, `/settings/billing` | Consistent settings navigation, forms, notification grouping, and pricing |

## Fixes applied

- Replaced page-specific visual themes with shared design tokens.
- Normalized hero, KPI, card, table, form, modal, badge, icon, and empty-state geometry.
- Removed decorative gradient/glass/shadow treatments from operational UI.
- Standardized typography and raised tiny labels to readable sizes.
- Added accessible labels to icon-only actions and verified image alternatives.
- Kept emoji only where it is actual customer-message content, not interface decoration.
- Added responsive two-column/one-column KPI behavior and full-width mobile actions.
- Added `scripts/ui-audit.sh` so regressions can be checked without manually spending review time.

## Verification

Run:

```bash
./scripts/ui-audit.sh
cd frontend && npm run typecheck
```

The audit script blocks missing image alternatives and unlabeled icon buttons. Raw color, gradient, and inline-style counts remain visible as inventory because charts, dynamic status colors, media previews, and data-driven dimensions legitimately require them.
