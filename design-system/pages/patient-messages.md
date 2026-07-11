---
page_id: patient-messages
auth_required: true
roles: [patient]
nav_section: primary
master_hash: 89511053b2bd1d8c063091b963c7ee92af6d8b89db0bcc4eff4757a4867a7a9d
---
## Design System: Ancora Health — Telehealth & Patient Platform

### Pattern
- **Name:** Community/Forum Landing
- **Conversion Focus:** Show active community (member count, posts today). Highlight benefits. Preview content. Easy onboarding.
- **CTA Placement:** Join button prominent + After member showcase
- **Color Strategy:** Warm, welcoming. Member photos add humanity. Topic badges in brand colors. Activity indicators green.
- **Sections:** 1. Hero (community value prop), 2. Popular topics/categories, 3. Active members showcase, 4. Join CTA

### Style
- **Name:** Exaggerated Minimalism
- **Keywords:** Bold minimalism, oversized typography, high contrast, negative space, loud minimal, statement design
- **Best For:** Fashion, architecture, portfolios, agency landing pages, luxury brands, editorial
- **Performance:** ⚡ Excellent | **Accessibility:** ✓ WCAG AA

### Colors
| Role | Hex |
|------|-----|
| Primary | #2563EB |
| Secondary | #3B82F6 |
| CTA | #F97316 |
| Background | #F8FAFC |
| Text | #1E293B |

*Notes: Team red + championship gold [Accent adjusted from #FBBF24 for WCAG 3:1]*

### Typography
- **Heading:** Space Grotesk
- **Body:** Inter
- **Mood:** web3, bitcoin, defi, digital gold, fintech, crypto, trustless, luminescent, precision, dark
- **Best For:** DeFi protocols and wallets, NFT platforms, metaverse social apps, high-tech brand landing pages
- **Google Fonts:** https://fonts.google.com/share?selection.family=Inter:wght@400;500;600;700|JetBrains+Mono:wght@400;500|Space+Grotesk:wght@500;600;700
- **CSS Import:**
```css
@import url('https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600;700&family=JetBrains+Mono:wght@400;500&family=Space+Grotesk:wght@500;600;700&display=swap');
```

### Key Effects
font-size: clamp(3rem 10vw 12rem), font-weight: 900, letter-spacing: -0.05em, massive whitespace

### Avoid (Anti-patterns)
- Excessive decoration

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
   📄 design-system/ancora-health-—-telehealth-&-patient-platform/pages/patient-messages.md (Page Overrides)

📖 Usage: When building a page, check design-system/ancora-health-—-telehealth-&-patient-platform/pages/[page].md first.
   If exists, its rules override MASTER.md. Otherwise, use MASTER.md.
============================================================
