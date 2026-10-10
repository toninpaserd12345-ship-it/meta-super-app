# Meta Super App — Product Design Standard

This document is the source of truth for the production UI. It applies to the dashboard, Facebook Pages, WhatsApp, Ads, products, Reply Sets, Automations, live chat, settings, and every new screen.

## 1. Product model and information architecture

The interface must make this relationship visible wherever the user configures automation:

`Channel account → Page/phone → Post, Ad or Campaign → Automation → Product → Reply Set → Customer`

- Facebook Page and WhatsApp phone are channel identities, not products.
- A Product owns commercial information and its real product image.
- A Reply Set owns the ordered text, image, video, and audio response sequence.
- An Automation links one Page, one Product, one Reply Set, and one or more targets.
- A target is a Post, Ad, Campaign, Ad Set, or keyword. Never label a target as a product.
- Names shown in the UI come from live records. Do not display demo, assumed, generated, or hard-coded business data.

## 2. Core experience principles

1. **Show identity before identifiers.** Show the real Page image, product image, post creative, ad creative, or WhatsApp display identity when the API provides it. Put the ID in secondary text.
2. **One primary action per surface.** A screen or empty state has one visually dominant next action. Secondary actions use tonal or text treatment.
3. **Progressive disclosure.** Lists show identity, status, and relationship summaries. Technical details and raw API errors live behind “Technical details”.
4. **Live data is explicit.** Dynamic workspace data must be fetched from the source. Loading, last refresh, permission problems, and empty results are distinct states.
5. **Destructive actions explain impact.** Delete dialogs name the record and show linked Automations or other blockers before deletion.
6. **Lao first, with precise English fallback.** Lao copy is natural and task-oriented. Do not expose internal field names when a user-facing label exists.

## 3. Images and media

### Display rules

- Page: circular 48–56 px profile image; fallback to the Page initial or Facebook icon.
- WhatsApp: circular business/profile image when available; always show the formatted phone number and WABA/phone ID below it.
- Product: 4:3 or square image; use `object-fit: cover`; fallback to a package icon.
- Post: 64–96 px thumbnail in pickers and lists; open the permalink in a secondary action.
- Ad: 48–72 px creative thumbnail at ad level. Campaigns do not invent a cover; use the thumbnails of their Ads when expanded.
- Reply media: render image, video, and audio previews in sequence order. Broken media gets a typed fallback, never an empty white box.
- Chat media: preserve aspect ratio, cap visual size, and provide meaningful alternative text.

Every meaningful image has alt text derived from the real record name. Decorative artwork uses empty alt text. Images must not replace visible names, statuses, or actions.

### Freshness and caching

- Authenticated API responses, Page profile proxies, Graph media, WhatsApp account data, Ads, Posts, Products, Reply Sets, and Automation records use `Cache-Control: no-store`.
- Client code may coalesce identical requests that are currently in flight, but must not retain a timed response cache.
- A manual refresh always issues a new source request.
- Static versioned build assets (hashed JavaScript, CSS, local fonts, icons) may use long immutable caching. They are not business data.
- Uploaded media should be written with `Cache-Control: no-store` or a versioned object key. Updating an image must change its URL or version.

`no-store` is required for dynamic data because `no-cache` may still store a response and only forces validation.

## 4. Layout and responsive behavior

- Maximum content width: 1680 px; center the workspace on wide screens.
- Use a 12-column desktop grid, 8-column tablet grid, and 4-column mobile grid.
- Breakpoints: compact `< 640`, tablet `640–1023`, desktop `≥ 1024`.
- Use list-detail for Pages, WhatsApp accounts, Campaigns/Ads, and Automations when space permits. On compact screens, details open below the selected list item or on a separate view.
- Tables are used only when comparing at least three columns. On compact screens, convert rows to cards; never depend on horizontal scrolling for the primary action.
- Sticky action bars must not cover content or mobile navigation.

## 5. Visual system

### Typography

- Lao: **Noto Sans Lao** 400/500/600/700.
- Latin fallback: Inter, system UI.
- Page title: 24–31 px / 700.
- Section title: 18–22 px / 700.
- Body: 14–16 px / 400–500.
- Supporting text: minimum 12 px. Avoid 9–10 px for operational data.
- IDs may use monospace, but never as the only identity.

### Spacing and shape

- Use the 4 px spacing scale: 4, 8, 12, 16, 20, 24, 32, 40, 48.
- Interactive targets are at least 44 × 44 CSS px; primary row actions use 48 px height.
- Cards use 12–18 px radius; inputs and buttons use 9–12 px radius.
- Avoid nesting more than two bordered cards. Prefer spacing and section headings over extra containers.

### Color and status

- Use semantic tokens, never raw colors inside feature pages.
- Body text meets 4.5:1 contrast; large text and UI graphics meet at least 3:1.
- Status is communicated with icon + text + color. Color alone is never the only signal.
- Green means active/success, amber means attention or incomplete setup, red means failed/destructive, gray means inactive or unavailable.

## 6. Standard components

### Page header

Contains eyebrow, one H1/H2, one concise description, and the primary action aligned right. Actions stack full width on compact screens.

### Resource row/card

Order: image/avatar → name and secondary ID → relationship/status → actions. The whole row may open details, but edit/delete controls remain explicit and keyboard accessible.

### Loading, empty, error, and permission states

- Loading: skeletons matching the final layout; do not show an empty table.
- Empty: state what is missing, why it matters, and give one next action.
- Zero result: preserve filters and offer “Clear filters”.
- Permission error: explain the missing permission and show reconnect/setup action; raw Graph errors go under technical details.
- Network error: preserve current form input and provide Retry.

### Automation wizard

1. Name and Page.
2. Select live Post, Ad, Campaign, Ad Set, or keyword targets with images where available.
3. Select Product and Reply Set with a combined visual preview.
4. First-message and cooldown behavior.
5. Review the exact relationship and activate.

Users can return to completed steps without losing input. Disabled “Continue” always has nearby guidance describing what is missing.

## 7. Accessibility acceptance

- Full keyboard operation and visible focus rings.
- Logical heading order and landmarks.
- Inputs have persistent labels; errors are connected to their fields.
- Touch targets meet 44 × 44 px where possible and never fall below 24 × 24 px with adequate spacing.
- Motion respects `prefers-reduced-motion`.
- Text reflows at 200% zoom without clipped primary actions.

## 8. Definition of done for every screen

- [ ] Uses live scoped data and has no demo/assumed content.
- [ ] Shows available real images and a typed fallback.
- [ ] Covers loading, empty, zero result, error, permission, and success states.
- [ ] Has Lao copy and an English fallback.
- [ ] Works at 360, 768, 1024, and 1440 px widths.
- [ ] Primary action is visible without ambiguity.
- [ ] Keyboard, focus, contrast, labels, and target sizes pass review.
- [ ] Dynamic API and media responses are `no-store`.
- [ ] Delete actions show impact and are read-back verified.
- [ ] `scripts/check.sh full` passes before deployment.

## References

- Meta accessibility: https://developers.meta.com/vr/design/accessibility/
- Meta color and semantic usage: https://developers.meta.com/vr/design/styles_color/
- Meta accessibility and design systems: https://www.meta.com/design-at-meta/blog/accessibility-and-design-systems/
- Material canonical layouts: https://m3.material.io/foundations/layout/canonical-examples/overview
- Material typography: https://m3.material.io/styles/typography/applying-type
- Material data tables: https://m1.material.io/components/data-tables.html
- Shopify Polaris empty state: https://polaris-react.shopify.com/components/layout-and-structure/empty-state
- Carbon empty states: https://www.carbondesignsystem.com/building-blocks/core/patterns/empty-states
- WCAG target size minimum: https://www.w3.org/WAI/WCAG22/Understanding/target-size-minimum
- WCAG enhanced target size: https://www.w3.org/WAI/WCAG22/Understanding/target-size-enhanced
- MDN Cache-Control: https://developer.mozilla.org/en-US/docs/Web/HTTP/Reference/Headers/Cache-Control
- MDN Request cache: https://developer.mozilla.org/en-US/docs/Web/API/Request/cache
