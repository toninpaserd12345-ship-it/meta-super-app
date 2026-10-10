# Ads Performance Dashboard — Research and Product Contract

Updated: 2026-10-10

## 1. Current state

The existing Ads page already loads these Meta objects:

- Ad accounts, separated into Personal and Business accounts.
- Campaign identity, status, objective, Ad Set count, and Ad count.
- Ads inside a campaign, including status and creative thumbnail.
- Automation links between a Campaign/Ad, Product, and Reply Set.

It does **not** currently load Ads Insights. Spend, impressions, reach, clicks,
CTR, CPC, CPM, frequency, leads, messages, purchases, and ROAS therefore cannot
yet be shown accurately.

## 2. What Meta can provide

Use the Marketing API Insights edge. Reporting is available at four levels:

| Level | Endpoint/parameter | Dashboard use |
| --- | --- | --- |
| Account | `GET /act_{ad_account_id}/insights?level=account` | Overall store advertising health |
| Campaign | `GET /act_{ad_account_id}/insights?level=campaign` | Compare campaign performance |
| Ad Set | `GET /act_{ad_account_id}/insights?level=adset` | Compare audience, budget, and delivery groups |
| Ad | `GET /act_{ad_account_id}/insights?level=ad` | Compare creatives and individual ads |

Recommended base fields:

```text
account_id,account_name,
campaign_id,campaign_name,
adset_id,adset_name,
ad_id,ad_name,
date_start,date_stop,
impressions,reach,frequency,
clicks,inline_link_clicks,
spend,cpm,cpc,ctr,
actions,cost_per_action_type,
action_values,purchase_roas
```

`actions`, `cost_per_action_type`, and `action_values` are arrays keyed by
`action_type`. The backend must normalize them; the frontend must never guess
an array position. Useful action types should be mapped into stable internal
metrics such as:

- `lead`
- `onsite_conversion.lead_grouped`
- `messaging_conversation_started_7d`
- `onsite_conversion.messaging_first_reply`
- `link_click`
- `landing_page_view`
- `add_to_cart`
- `initiate_checkout`
- `purchase`

Meta may return different action types according to objective, destination,
pixel/CAPI setup, attribution settings, and API version. Preserve unknown action
types in raw diagnostic data, but do not show them as zero or as a conversion.

## 3. Dashboard information architecture

### Header controls

1. Workspace/store selector (existing global selector).
2. Ad account selector with a clear `Personal` or `Business` badge.
3. Date preset: Today, Yesterday, Last 7 days, Last 30 days, This month, Custom.
4. Compare toggle: previous period.
5. Attribution label showing the setting used by the report.
6. Last synchronized time and a manual Refresh action.

### Summary cards

Show only metrics supported by the selected account and objective:

- Amount spent
- Results (with the result type written explicitly)
- Cost per result
- Reach
- Impressions
- CTR (link)
- CPC (link)
- CPM
- Leads or Messaging conversations started
- Purchases and purchase ROAS when conversion tracking exists

Never show a bare `Results` number without its action type. A lead, message,
purchase, and link click are not interchangeable.

### Trend chart

- Default: Spend and Results by day (`time_increment=1`).
- Optional series: Reach, Impressions, CTR, Cost per result, Leads, Messages,
  Purchases, ROAS.
- Use the ad account currency and time zone returned by Meta.
- Missing data must render as `—`, not `0`, unless Meta explicitly returned zero.

### Performance table

The table must drill down in this order:

```text
Campaign → Ad Set → Ad → Creative
```

Recommended columns: Status, Delivery, Budget, Spend, Results, Cost/result,
Reach, Impressions, Frequency, CTR, CPC, CPM, Automation, and Product/Reply Set.
Users can choose columns, sort, search, and export CSV. On phones the row becomes
a card with the four primary metrics and an expandable details section.

### Lead and messaging panel

- Aggregated lead/message counts come from Ads Insights.
- Actual Lead Ads form submissions require `leads_retrieval` and access to the
  Page/form. This is a separate permission and a separate feature from the
  aggregated performance dashboard.
- Messenger/WhatsApp replies received by this application should be joined from
  local webhook data using `ad_id`, `campaign_id`, Page/phone, and workspace.
- Show a funnel: Ad result → conversation started → first reply → qualified lead
  → order, but never claim later stages unless the application recorded them.

## 4. Permissions and access behavior

- `ads_read`: read Ads structure and performance for ad accounts the person can
  access.
- `business_management`: discover Business Portfolio assets and client/owned ad
  accounts. It does not replace access to the ad account itself.
- `leads_retrieval`: required only when retrieving the people and answers from a
  Lead Ads form.
- App Development mode limits real use to people who have an app role. Production
  access for other businesses requires the corresponding App Review/advanced
  access and may require Business Verification.

Every request must resolve the token from the active workspace. No Ad Account,
Insight, WABA, Page, or AI summary may be shared between workspaces.

## 5. Backend API contract

Add one normalized endpoint rather than letting the browser call Meta directly:

```http
GET /api/v1/meta/ad-accounts/{id}/insights
  ?level=account|campaign|adset|ad
  &since=YYYY-MM-DD
  &until=YYYY-MM-DD
  &timeIncrement=all_days|1
  &compare=false|true
```

Example response shape:

```json
{
  "currency": "USD",
  "timezone": "Asia/Vientiane",
  "attribution": "account_setting",
  "dateStart": "2026-10-01",
  "dateStop": "2026-10-10",
  "generatedAt": "2026-10-10T12:00:00Z",
  "items": [
    {
      "entityType": "campaign",
      "entityId": "123",
      "entityName": "Campaign A",
      "spend": 25.5,
      "impressions": 12000,
      "reach": 9000,
      "frequency": 1.33,
      "linkClicks": 240,
      "linkCtr": 2.0,
      "linkCpc": 0.10625,
      "cpm": 2.125,
      "results": { "type": "lead", "value": 18, "cost": 1.4167 },
      "leads": 18,
      "messagingConversations": null,
      "purchases": null,
      "purchaseValue": null,
      "purchaseRoas": null
    }
  ],
  "warnings": []
}
```

All money and ratios must be parsed into numbers on the backend. Retain the raw
Meta response only in protected short-lived diagnostics; never return access
tokens or raw personal lead data from this endpoint.

## 6. Fetching, caching, and rate limits

- Frontend: one query per `workspace + adAccount + level + date range`; deduplicate
  concurrent requests. Do not independently fetch the same report in every card.
- Backend: cache normalized Insights for 5 minutes. `Refresh` invalidates only the
  selected report key.
- Historical completed days can be retained longer; today should be refreshed.
- Use cursor pagination for entity lists.
- Use Meta asynchronous Insights jobs for large date ranges, ad-level exports, or
  breakdown reports; poll the report run, then page through its results.
- Do not call Insights on every component mount or window focus.
- Return `generatedAt`, `isStale`, and warnings so the UI can explain freshness.

## 7. AI analysis contract

The AI is an explanation layer, not the source of truth. Calculations remain
deterministic in the backend.

Give the AI only:

- Workspace-safe normalized metrics.
- Selected date range and comparison range.
- Currency, time zone, attribution label, objective, and delivery status.
- Metric definitions and warnings from this contract.

The AI response should contain:

1. A plain-language summary in the user's selected language (Lao/Thai/English).
2. What improved or declined, with exact numbers and periods.
3. Likely causes clearly labelled as hypotheses.
4. Three prioritized actions.
5. Data-quality warnings and missing tracking.

The AI must not:

- Invent unavailable conversions, revenue, or demographics.
- Treat correlation as causation.
- compare periods of different length without saying so.
- Recommend increasing budget solely because CTR is high.
- Claim that an ad is profitable when purchase value/ROAS is unavailable.
- expose one store's data to another store.

Recommended UI: a `Explain performance` button opens an analysis drawer. Show
the exact data range, generated time, and a `Why?` link next to each recommendation.

## 8. Delivery phases

### Phase 1 — Reliable reporting

- Account/Campaign/Ad Set/Ad Insights endpoint.
- Date selector, summary cards, trend chart, performance table.
- Workspace isolation, 5-minute server cache, error and permission states.
- Lao/Thai/English metric labels and definitions.

### Phase 2 — Automation attribution

- Join campaign/ad metrics with Product and Reply Set automations.
- Show messages received and replies sent from local webhook events.
- Add first-message, qualified-lead, and order funnel when locally recorded.

### Phase 3 — AI analyst

- Deterministic comparison payload.
- Multilingual explanation and prioritized recommendations.
- Feedback buttons and audit log of the metrics used for each explanation.

### Phase 4 — Lead CRM

- Request `leads_retrieval` only when this feature is ready.
- Secure Lead Ads form ingestion, assignment, status, follow-up, and retention.

## 9. Acceptance tests

- Two workspaces using the same Meta user never see each other's ad accounts,
  cache entries, reports, or AI summaries.
- Personal and Business ad accounts are visibly different.
- Changing date range updates every card, chart, table, and AI context together.
- Account totals reconcile with Meta Ads Manager for the same dates, time zone,
  attribution setting, and selected columns.
- Missing purchases render as unavailable, not zero.
- Meta permission, expired-token, rate-limit, and async-report states have clear
  recovery actions.
- Desktop, tablet, and mobile layouts remain usable without a clipped primary
  action.

## Primary references

- Meta Marketing API Insights documentation:
  https://developers.facebook.com/docs/marketing-api/insights
- Meta Ads Insights reference:
  https://developers.facebook.com/docs/marketing-api/reference/ads-insights
- Meta official Marketing API Postman workspace:
  https://www.postman.com/meta/facebook-marketing-api/overview
- Meta official Python Business SDK, Insights implementation:
  https://github.com/facebook/facebook-python-business-sdk/blob/main/facebook_business/adobjects/ad.py
- Meta official generated Ads Insights field list:
  https://github.com/facebook/facebook-python-business-sdk/blob/main/facebook_business/adobjects/adsinsights.py
- Meta permissions reference:
  https://developers.facebook.com/docs/permissions
