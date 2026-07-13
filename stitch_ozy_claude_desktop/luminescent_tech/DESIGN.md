---
name: Luminescent Tech
colors:
  surface: '#131313'
  surface-dim: '#131313'
  surface-bright: '#3a3939'
  surface-container-lowest: '#0e0e0e'
  surface-container-low: '#1c1b1b'
  surface-container: '#201f1f'
  surface-container-high: '#2a2a2a'
  surface-container-highest: '#353534'
  on-surface: '#e5e2e1'
  on-surface-variant: '#c6c9ac'
  inverse-surface: '#e5e2e1'
  inverse-on-surface: '#313030'
  outline: '#909379'
  outline-variant: '#454933'
  surface-tint: '#b7d300'
  primary: '#ffffff'
  on-primary: '#2c3400'
  primary-container: '#d1f107'
  on-primary-container: '#5c6b00'
  inverse-primary: '#566500'
  secondary: '#c8c6c5'
  on-secondary: '#313030'
  secondary-container: '#474746'
  on-secondary-container: '#b7b5b4'
  tertiary: '#ffffff'
  on-tertiary: '#2f3131'
  tertiary-container: '#e2e2e2'
  on-tertiary-container: '#636565'
  error: '#ffb4ab'
  on-error: '#690005'
  error-container: '#93000a'
  on-error-container: '#ffdad6'
  primary-fixed: '#d1f107'
  primary-fixed-dim: '#b7d300'
  on-primary-fixed: '#181e00'
  on-primary-fixed-variant: '#404c00'
  secondary-fixed: '#e5e2e1'
  secondary-fixed-dim: '#c8c6c5'
  on-secondary-fixed: '#1c1b1b'
  on-secondary-fixed-variant: '#474746'
  tertiary-fixed: '#e2e2e2'
  tertiary-fixed-dim: '#c6c6c7'
  on-tertiary-fixed: '#1a1c1c'
  on-tertiary-fixed-variant: '#454747'
  background: '#131313'
  on-background: '#e5e2e1'
  surface-variant: '#353534'
typography:
  headline-xl:
    fontFamily: Literata
    fontSize: 48px
    fontWeight: '700'
    lineHeight: 56px
    letterSpacing: -0.02em
  headline-lg:
    fontFamily: Literata
    fontSize: 32px
    fontWeight: '600'
    lineHeight: 40px
    letterSpacing: -0.01em
  headline-lg-mobile:
    fontFamily: Literata
    fontSize: 28px
    fontWeight: '600'
    lineHeight: 36px
  body-lg:
    fontFamily: Literata
    fontSize: 18px
    fontWeight: '400'
    lineHeight: 28px
  body-md:
    fontFamily: Literata
    fontSize: 16px
    fontWeight: '400'
    lineHeight: 24px
  label-md:
    fontFamily: Geist
    fontSize: 14px
    fontWeight: '500'
    lineHeight: 20px
    letterSpacing: 0.05em
  label-sm:
    fontFamily: Geist
    fontSize: 12px
    fontWeight: '500'
    lineHeight: 16px
rounded:
  sm: 0.25rem
  DEFAULT: 0.5rem
  md: 0.75rem
  lg: 1rem
  xl: 1.5rem
  full: 9999px
spacing:
  base: 8px
  gutter: 24px
  margin-desktop: 64px
  margin-mobile: 20px
  container-max: 1280px
---

## Brand & Style
This design system focuses on high-impact, technology-driven aesthetics that merge academic rigor with a high-energy, digital-first presence. The brand personality is authoritative yet provocative, designed to appeal to developers, researchers, and forward-thinking creators who value both deep legibility and bold visual statements.

The design style is a hybrid of **Minimalism** and **High-Contrast / Bold**. It utilizes vast dark voids to allow the vibrant primary accent to create "retina-burn" focal points. The interface feels lightweight and fast, stripping away unnecessary ornamentation to let the content and the striking color palette drive the user experience.

## Colors
The palette is centered around "Volt Green" (#d2f20b), a high-visibility primary color that serves as the singular beacon for all interactive and highlighted states. This color is set against a deep, near-black neutral base (#0a0a0a) to ensure maximum vibrance and professional tone.

- **Primary:** #d2f20b (Used for buttons, active icons, and critical calls to action).
- **Surface:** #1a1a1a (Used for cards and container backgrounds).
- **Background:** #0a0a0a (The base canvas).
- **On-Primary:** #000000 (Black text is required on the Volt Green background to maintain AA/AAA contrast).
- **Text-High:** #ffffff (Standard content).
- **Text-Low:** #888888 (Secondary metadata).

## Typography
This design system utilizes **Literata** for both headlines and body copy to provide a sophisticated, literary feel that grounds the high-contrast color scheme. The serif details offer a "human" counterpoint to the synthetic primary color.

**Geist** is introduced as a supporting monospace/technical font for labels, buttons, and metadata to reinforce the developer-centric, precise nature of the system. All labels should be rendered in Geist to differentiate functional UI elements from editorial content.

## Layout & Spacing
The layout follows a **Fixed Grid** model on desktop (12 columns) and a **Fluid Grid** on mobile (4 columns). The spacing rhythm is strictly based on an 8px base unit. 

- **Desktop:** 1280px max-width container with 24px gutters and 64px outer margins.
- **Tablet:** 8-column grid with 24px gutters and 32px outer margins.
- **Mobile:** 4-column fluid grid with 16px gutters and 20px outer margins.

Large vertical sections should be separated by 80px or 120px to maintain the "Minimalist" breathing room.

## Elevation & Depth
Depth is achieved through **Tonal Layers** rather than shadows. In a dark-mode environment, shadows are less effective; instead, we use progressively lighter shades of grey to indicate height.

1.  **Level 0 (Base):** #0a0a0a
2.  **Level 1 (Cards/Sections):** #1a1a1a
3.  **Level 2 (Modals/Popovers):** #262626

For high-priority interactive elements, use **low-contrast outlines** in #d2f20b at 20% opacity to create a subtle "glow" effect without the clutter of a traditional blur.

## Shapes
The shape language follows a "ROUND_EIGHT" philosophy (0.5rem base), providing a modern, balanced look that is neither too sharp nor too playful.

- **Standard Elements:** 8px (0.5rem) radius for buttons, input fields, and small cards.
- **Large Containers:** 16px (1rem) radius for primary content blocks and modals.
- **Outer Shells:** 24px (1.5rem) radius for the most significant layout containers.

## Components
- **Buttons:** Primary buttons use a solid #d2f20b background with black (#000000) Geist SemiBold text. Secondary buttons are ghost-style with a #d2f20b border and white text.
- **Inputs:** Dark backgrounds (#1a1a1a) with a subtle 1px border. On focus, the border changes to #d2f20b with no glow.
- **Chips:** Small, Geist-font labels with a #262626 background and #ffffff text. Active chips switch to #d2f20b background with black text.
- **Cards:** Subtle #1a1a1a backgrounds with 8px corner radii. No shadows. Use 24px internal padding for content.
- **Lists:** Separated by thin 1px lines in #262626. Interactive list items highlight with a #1a1a1a background on hover.
- **Status Indicators:** Success/Active states must use the primary #d2f20b color to maintain brand consistency.