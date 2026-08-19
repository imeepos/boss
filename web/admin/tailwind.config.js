/** @type {import('tailwindcss').Config} */
export default {
  darkMode: ['class', '[data-theme="dark"]'],
  content: [
    './index.html',
    './src/**/*.{ts,tsx}',
  ],
  theme: {
    extend: {
      colors: {
        brand: {
          navy: {
            950: 'var(--color-brand-navy-950)',
            900: 'var(--color-brand-navy-900)',
          },
          blue: {
            900: 'var(--color-brand-blue-900)',
            800: 'var(--color-brand-blue-800)',
            700: 'var(--color-brand-blue-700)',
            300: 'var(--color-brand-blue-300)',
          },
          gold: {
            600: 'var(--color-brand-gold-600)',
            500: 'var(--color-brand-gold-500)',
            300: 'var(--color-brand-gold-300)',
            50: 'var(--color-brand-gold-50)',
          },
        },
        surface: {
          primary: 'var(--color-surface-primary)',
          input: 'var(--color-surface-input)',
          disabled: 'var(--color-surface-disabled)',
          dark: 'var(--color-surface-dark)',
        },
        text: {
          primary: 'var(--color-text-primary)',
          secondary: 'var(--color-text-secondary)',
          tertiary: 'var(--color-text-tertiary)',
          muted: 'var(--color-text-muted)',
          placeholder: 'var(--color-text-placeholder)',
          'on-dark': 'var(--color-text-on-dark)',
          link: 'var(--color-text-link)',
          input: 'var(--color-text-input)',
        },
        border: {
          DEFAULT: 'var(--color-border-default)',
          hover: 'var(--color-border-hover)',
          focus: 'var(--color-border-focus)',
          error: 'var(--color-border-error)',
        },
        shell: {
          topbar: 'var(--shell-topbar-bg)',
          'side-bg': 'var(--shell-side-bg)',
          'content-bg': 'var(--shell-content-bg)',
          'content-text': 'var(--shell-content-text)',
          heading: 'var(--shell-heading)',
          'card-bg': 'var(--shell-card-bg)',
          'menu-text': 'var(--shell-menu-text)',
          'menu-hover': 'var(--shell-menu-hover-bg)',
          'menu-active': 'var(--shell-menu-active-bg)',
        },
        /* shadcn 标准语义颜色（映射到 CSS 变量）*/
        background: 'hsl(var(--background))',
        foreground: 'hsl(var(--foreground))',
        card: 'hsl(var(--card))',
        'card-foreground': 'hsl(var(--card-foreground))',
        popover: 'hsl(var(--popover))',
        'popover-foreground': 'hsl(var(--popover-foreground))',
        primary: {
          DEFAULT: 'hsl(var(--primary))',
          foreground: 'hsl(var(--primary-foreground))',
        },
        secondary: {
          DEFAULT: 'hsl(var(--secondary))',
          foreground: 'hsl(var(--secondary-foreground))',
        },
        muted: {
          DEFAULT: 'hsl(var(--muted))',
          foreground: 'hsl(var(--muted-foreground))',
        },
        accent: {
          DEFAULT: 'hsl(var(--accent))',
          foreground: 'hsl(var(--accent-foreground))',
        },
        destructive: {
          DEFAULT: 'hsl(var(--destructive))',
          foreground: 'hsl(var(--destructive-foreground))',
        },
        success: 'var(--color-success)',
        danger: 'var(--color-danger)',
      },
      borderRadius: {
        sm: 'var(--radius-sm)',
        md: 'var(--radius-md)',
        lg: 'var(--radius-lg)',
      },
      fontFamily: {
        base: ['"PingFang SC"', '"Microsoft YaHei"', '"Noto Sans SC"', '"Helvetica Neue"', 'Arial', 'sans-serif'],
        brand: ['Inter', 'Arial', '"Helvetica Neue"', 'sans-serif'],
      },
      boxShadow: {
        panel: 'var(--shadow-panel)',
        card: 'var(--shadow-card)',
        focus: 'var(--shadow-focus)',
      },
      keyframes: {
        'accordion-down': {
          from: { height: '0' },
          to: { height: 'var(--radix-accordion-content-height)' },
        },
        'accordion-up': {
          from: { height: 'var(--radix-accordion-content-height)' },
          to: { height: '0' },
        },
      },
      animation: {
        'accordion-down': 'accordion-down 0.2s ease-out',
        'accordion-up': 'accordion-up 0.2s ease-out',
      },
    },
  },
  plugins: [require('tailwindcss-animate')],
}