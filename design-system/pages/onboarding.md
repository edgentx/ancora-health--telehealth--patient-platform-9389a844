---
page_id: onboarding
auth_required: true
roles: []
nav_section: none
master_hash: 89511053b2bd1d8c063091b963c7ee92af6d8b89db0bcc4eff4757a4867a7a9d
---
## Design System: Ancora Health — Telehealth & Patient Platform

### Pattern
- **Name:** Before-After Transformation
- **Conversion Focus:** Visual proof of value. 45% higher conversion. Real results. Specific metrics. Guarantee offer.
- **CTA Placement:** After transformation reveal + Bottom
- **Color Strategy:** Contrast: muted/grey (before) vs vibrant/colorful (after). Success green for results.
- **Sections:** 1. Hero (problem state), 2. Transformation slider/comparison, 3. How it works, 4. Results CTA

### Style
- **Name:** User Behavior Analytics
- **Keywords:** Funnel visualization, user flow diagrams, conversion tracking, engagement metrics, user journey mapping, cohort analysis
- **Best For:** Conversion funnel analysis, user journey tracking, engagement analytics, cohort analysis, retention tracking
- **Performance:** ⚡ Good | **Accessibility:** ✓ WCAG AA

### Colors
| Role | Hex |
|------|-----|
| Primary | #2563EB |
| Secondary | #3B82F6 |
| CTA | #F97316 |
| Background | #F8FAFC |
| Text | #1E293B |

*Notes: Security blue + protected green [Accent adjusted from #22C55E for WCAG 3:1]*

### Typography
- **Heading:** IBM Plex Sans
- **Body:** IBM Plex Sans
- **Mood:** financial, trustworthy, professional, corporate, banking, serious
- **Best For:** Banks, finance, insurance, investment, fintech, enterprise
- **Google Fonts:** https://fonts.google.com/share?selection.family=IBM+Plex+Sans:wght@300;400;500;600;700
- **CSS Import:**
```css
@import url('https://fonts.googleapis.com/css2?family=IBM+Plex+Sans:wght@300;400;500;600;700&display=swap');
```

### Key Effects
Funnel animation (fill-down), flow diagram animations (connection draw), conversion pulse, engagement bar fill

### Avoid (Anti-patterns)
- Excessive decoration
- Color-only indicators

### Pre-Delivery Checklist
- [ ] No emojis as icons (use SVG: Heroicons/Lucide)
- [ ] cursor-pointer on all clickable elements
- [ ] Hover states with smooth transitions (150-300ms)
- [ ] Light mode: text contrast 4.5:1 minimum
- [ ] Focus states visible for keyboard nav
- [ ] prefers-reduced-motion respected
- [ ] Responsive: 375px, 768px, 1024px, 1440px


============================================================
✅ Design system persisted to design-system/ancora-health-—-telehealth-&-patient-platform/
   📄 design-system/ancora-health-—-telehealth-&-patient-platform/MASTER.md (Global Source of Truth)
   📄 design-system/ancora-health-—-telehealth-&-patient-platform/pages/onboarding.md (Page Overrides)

📖 Usage: When building a page, check design-system/ancora-health-—-telehealth-&-patient-platform/pages/[page].md first.
   If exists, its rules override MASTER.md. Otherwise, use MASTER.md.
============================================================
