# Zosmed Hi-Fi Design — Screen Inventory

Source: `Vendor.html` from claude.ai/design project `zozmed`
(projectId `019dc928-acd7-7e0b-82e3-1b8f17387464`).

Vendor.html is the **master design canvas**. It composes artboard JSX components
(in `artboards/*.jsx` on the design project) into one scrollable board, wrapped in
design-tool chrome (`DesignCanvas`, `DCSection`, `DCArtboard`, `TweaksPanel`).
The **app screens** are the artboard components themselves. Design-tool chrome
(`design-canvas.jsx`, `tweaks-panel.jsx`) is NOT part of the app — ignore it.

Shared design system (saved locally in this folder):
- `tokens.css` — CSS variables: dark + lime theme (Direction A), Geist fonts.
- `primitives.jsx` — shared components: `Logo`, `Pill`, `Dot`, `I` (icon set),
  `Placeholder`, `Avatar`, `ZZ_LIME`.

## Full screen list (in canvas order)

| # | Section id        | Title                                  | Artboard component        | Source file                          | Canvas WxH   |
|---|-------------------|----------------------------------------|---------------------------|--------------------------------------|--------------|
| — | overview          | 00 · Brand & system                    | `BrandSheet`              | inline in Vendor.html                | 920 × 540    |
| 1 | landing           | 00 · Landing page                      | `LandingDark`             | artboards/landing-dark.jsx           | 1280 × 4480  |
| 2 | onboarding        | 00 · Onboarding (4 steps)              | `OnboardingDark`          | artboards/onboarding-dark.jsx        | 1280 × 820   |
| 3 | dashboard         | 01 · Dashboard                         | `DashboardDark`           | artboards/dashboard-dark.jsx         | 1440 × 1100  |
| 4 | inbox             | 02 · Inbox (threaded chat)             | `InboxDark`               | artboards/menu-details-dark.jsx?*    | 1440 × 1100  |
| 5 | workflow          | 03 · Workflows — Builder               | `WorkflowBuilderDark`     | artboards/workflow-dark.jsx          | 1440 × 1100  |
| 6 | wf-inspector      | 03 · Workflows — Editor inspector      | `WorkflowInspectorDark`   | artboards/workflow-dark.jsx?*        | 1440 × 1100  |
| 7 | workflow-runs     | 03 · Workflows — Runs                  | `WorkflowRunsDark`        | artboards/workflow-dark.jsx?*        | 1440 × 1100  |
| 8 | live-selling      | 03 · Workflows — Comment-to-Order      | `LiveSellingDark`         | artboards/*?*                        | 1440 × 1100  |
| 9 | ai-studio         | 04 · AI Studio                         | `AIStudioDark`            | artboards/*?*                        | 1440 × 1100  |
|10 | contacts          | 05 · Contacts                          | `ContactsDark`            | artboards/*?*                        | 1440 × 1100  |
|11 | contact-profile   | 05 · Contacts — Profile                | `ContactProfileDark`      | artboards/*?*                        | 1440 × 1100  |
|12 | analytics         | 06 · Analytics                         | `AnalyticsDark`           | artboards/analytics-dark.jsx         | 1440 × 1100  |
|13 | analytics-drill   | 06 · Analytics — Workflow drilldown    | `AnalyticsDrilldownDark`  | artboards/analytics-dark.jsx?*       | 1440 × 1100  |
|14 | templates         | 07 · Templates                         | `TemplatesDark`           | artboards/*?*                        | 1440 × 1100  |
|15 | safety            | 08 · Safety center                     | `SafetyDark`              | artboards/*?*                        | 1440 × 1100  |
|16 | notifications     | 09 · Notifications                     | `NotificationsDark`       | artboards/*?*                        | 1440 × 1100  |
|17 | members           | 10 · Team                              | `TeamMembersDark`         | artboards/*?*                        | 1440 × 1100  |
|18 | settings          | 11 · Settings                          | `SettingsDark`            | artboards/*?*                        | 1440 × 1100  |
|19 | billing           | 11 · Settings — Billing                | `BillingDark`             | artboards/*?*                        | 1440 × 1100  |
|20 | seller-kit        | 🇮🇩 Seller Kit                          | `IDCommerceKitDark`       | artboards/id-features-dark.jsx       | 1440 × 1240  |
|21 | creator-kit       | ✦ Creator Kit                          | `CreatorKitDark`          | artboards/kits-dark.jsx?*            | 1440 × 1240  |
|22 | booking-kit       | ◷ Booking Kit                          | `BookingKitDark`          | artboards/kits-dark.jsx?*            | 1440 × 1240  |
|23 | states            | 12 · Empty & error states              | `EmptyStatesDark`         | artboards/*?*                        | 1440 × 800   |

`*` = exact source file must be confirmed at implementation time by pulling the
component from the design project. Known artboard files on the project:
`landing-dark.jsx`, `dashboard-dark.jsx`, `workflow-dark.jsx`, `onboarding-dark.jsx`,
`analytics-dark.jsx`, `menu-details-dark.jsx`, `menu-deeper-dark.jsx`,
`menu-more-dark.jsx`, `id-features-dark.jsx`, `kits-dark.jsx`.
Several artboard files each export multiple screen components.

## How to pull a screen's source during implementation
Use the DesignSync MCP (`get_file`, projectId above) to fetch the relevant
`artboards/<file>.jsx`. These are React-in-Babel JSX with inline styles — they
must be ported to Next.js + TypeScript + Tailwind components, mapping inline
styles/hex to the `tokens.css` variables and `packages/ui` primitives.

## Tweaks (design-tool only, not app features)
accent (lime/cyan/orange/pink), density (compact/comfortable),
nodeStyle (sharp/soft), showGrid. Default ship: accent=lime, density=compact,
nodeStyle=soft. These are design knobs — implement the lime/compact/soft defaults.

## Sidebar order (from CLAUDE.md §9, matches canvas numbering)
dashboard · workflows · inbox · ai · contacts · analytics · safety ·
templates · settings · team · notifications
