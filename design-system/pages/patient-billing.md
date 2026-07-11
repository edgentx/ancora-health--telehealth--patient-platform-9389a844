---
page_id: patient-billing
auth_required: true
roles: [patient]
nav_section: primary
master_hash: 89511053b2bd1d8c063091b963c7ee92af6d8b89db0bcc4eff4757a4867a7a9d
---
## Design System: Ancora Health — Telehealth & Patient Platform

### Pattern
- **Name:** Marketplace / Directory
- **Conversion Focus:** Search bar is the CTA. Reduce friction to search. Popular searches suggestions.
- **CTA Placement:** Hero Search Bar + Navbar 'List your item'
- **Color Strategy:** Search: High contrast. Categories: Visual icons. Trust: Blue/Green.
- **Sections:** 1. Hero (Search focused), 2. Categories, 3. Featured Listings, 4. Trust/Safety, 5. CTA (Become a host/seller)

### Style
- **Name:** Minimalism & Swiss Style
- **Keywords:** Clean, simple, spacious, functional, white space, high contrast, geometric, sans-serif, grid-based, essential
- **Best For:** Enterprise apps, dashboards, documentation sites, SaaS platforms, professional tools
- **Performance:** ⚡ Excellent | **Accessibility:** ✓ WCAG AAA

### Colors
| Role | Hex |
|------|-----|
| Primary | #2563EB |
| Secondary | #3B82F6 |
| CTA | #F97316 |
| Background | #F8FAFC |
| Text | #1E293B |

*Notes: Navy professional + paid green*

### Typography
- **Heading:** EB Garamond
- **Body:** Crimson Text
- **Mood:** academic, old-school, university, research, serious, traditional
- **Best For:** University sites, archives, research papers, history
- **Google Fonts:** https://fonts.google.com/share?selection.family=Crimson+Text:wght@400;600;700|EB+Garamond:wght@400;500;600;700;800
- **CSS Import:**
```css
@import url('https://fonts.googleapis.com/css2?family=Crimson+Text:wght@400;600;700&family=EB+Garamond:wght@400;500;600;700;800&display=swap');
```

### Key Effects
Subtle hover (200-250ms), smooth transitions, sharp shadows if any, clear type hierarchy, fast loading

### Avoid (Anti-patterns)
- Excessive decoration
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
   📄 design-system/ancora-health-—-telehealth-&-patient-platform/pages/patient-billing.md (Page Overrides)

📖 Usage: When building a page, check design-system/ancora-health-—-telehealth-&-patient-platform/pages/[page].md first.
   If exists, its rules override MASTER.md. Otherwise, use MASTER.md.
============================================================
