---
page_id: patient-provider-profile
auth_required: true
roles: [patient]
nav_section: none
master_hash: 89511053b2bd1d8c063091b963c7ee92af6d8b89db0bcc4eff4757a4867a7a9d
---
## Design System: Ancora Health — Telehealth & Patient Platform

### Pattern
- **Name:** Real-Time / Operations Landing
- **Conversion Focus:** For ops/security/iot products. Demo or sandbox link. Trust signals.
- **CTA Placement:** Primary CTA in nav + After metrics
- **Color Strategy:** Dark or neutral. Status colors (green/amber/red). Data-dense but scannable.
- **Sections:** 1. Hero (product + live preview or status), 2. Key metrics/indicators, 3. How it works, 4. CTA (Start trial / Contact)

### Style
- **Name:** Soft UI Evolution
- **Keywords:** Evolved soft UI, better contrast, modern aesthetics, subtle depth, accessibility-focused, improved shadows, hybrid
- **Best For:** Modern enterprise apps, SaaS platforms, health/wellness, modern business tools, professional, hybrid
- **Performance:** ⚡ Excellent | **Accessibility:** ✓ WCAG AA+

### Colors
| Role | Hex |
|------|-----|
| Primary | #2563EB |
| Secondary | #3B82F6 |
| CTA | #F97316 |
| Background | #F8FAFC |
| Text | #1E293B |

*Notes: Calendar blue + available green*

### Typography
- **Heading:** Russo One
- **Body:** Chakra Petch
- **Mood:** gaming, bold, action, esports, competitive, energetic
- **Best For:** Gaming, esports, action games, competitive sports, entertainment
- **Google Fonts:** https://fonts.google.com/share?selection.family=Chakra+Petch:wght@300;400;500;600;700|Russo+One
- **CSS Import:**
```css
@import url('https://fonts.googleapis.com/css2?family=Chakra+Petch:wght@300;400;500;600;700&family=Russo+One&display=swap');
```

### Key Effects
Improved shadows (softer than flat, clearer than neumorphism), modern (200-300ms), focus visible, WCAG AA/AAA

### Avoid (Anti-patterns)
- Complex shadows
- 3D effects

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
   📄 design-system/ancora-health-—-telehealth-&-patient-platform/pages/patient-provider-profile.md (Page Overrides)

📖 Usage: When building a page, check design-system/ancora-health-—-telehealth-&-patient-platform/pages/[page].md first.
   If exists, its rules override MASTER.md. Otherwise, use MASTER.md.
============================================================
