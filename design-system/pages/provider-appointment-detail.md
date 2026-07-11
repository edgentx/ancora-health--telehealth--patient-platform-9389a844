---
page_id: provider-appointment-detail
auth_required: true
roles: [provider]
nav_section: none
master_hash: 89511053b2bd1d8c063091b963c7ee92af6d8b89db0bcc4eff4757a4867a7a9d
---
## Design System: Ancora Health — Telehealth & Patient Platform

### Pattern
- **Name:** Video-First Hero
- **Conversion Focus:** 86% higher engagement with video. Add captions for accessibility. Compress video for performance.
- **CTA Placement:** Overlay on video (center/bottom) + Bottom section
- **Color Strategy:** Dark overlay 60% on video. Brand accent for CTA. White text on dark.
- **Sections:** 1. Hero with video background, 2. Key features overlay, 3. Benefits section, 4. CTA

### Style
- **Name:** Dark Mode (OLED)
- **Keywords:** Dark theme, low light, high contrast, deep black, midnight blue, eye-friendly, OLED, night mode, power efficient
- **Best For:** Night-mode apps, coding platforms, entertainment, eye-strain prevention, OLED devices, low-light
- **Performance:** ⚡ Excellent | **Accessibility:** ✓ WCAG AAA

### Colors
| Role | Hex |
|------|-----|
| Primary | #2563EB |
| Secondary | #3B82F6 |
| CTA | #F97316 |
| Background | #F8FAFC |
| Text | #1E293B |

*Notes: Video pink on dark + timeline blue*

### Typography
- **Heading:** Space Grotesk
- **Body:** Space Grotesk
- **Mood:** neo brutalism, pop art, loud, bold, heavy, stickers, mechanical, high contrast, cream, gen-z
- **Best For:** Creative tools, Gen-Z marketing, e-commerce for youth culture, content portfolios, collage-style apps
- **Google Fonts:** https://fonts.google.com/share?selection.family=Space+Grotesk:wght@700
- **CSS Import:**
```css
@import url('https://fonts.googleapis.com/css2?family=Space+Grotesk:wght@700&display=swap');
```

### Key Effects
Minimal glow (text-shadow: 0 0 10px), dark-to-light transitions, low white emission, high readability, visible focus

### Avoid (Anti-patterns)
- Poor audio player
- Cluttered layout

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
   📄 design-system/ancora-health-—-telehealth-&-patient-platform/pages/provider-appointment-detail.md (Page Overrides)

📖 Usage: When building a page, check design-system/ancora-health-—-telehealth-&-patient-platform/pages/[page].md first.
   If exists, its rules override MASTER.md. Otherwise, use MASTER.md.
============================================================
