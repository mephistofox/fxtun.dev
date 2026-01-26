/** @type {import('tailwindcss').Config} */
export default {
  darkMode: ['class'],
  content: [
    './index.html',
    './src/**/*.{vue,js,ts,jsx,tsx}',
  ],
  theme: {
    container: {
      center: true,
      padding: '2rem',
      screens: {
        '2xl': '1400px',
      },
    },
    extend: {
      colors: {
        border: 'hsl(var(--border))',
        input: 'hsl(var(--input))',
        ring: 'hsl(var(--ring))',
        background: 'hsl(var(--background))',
        foreground: 'hsl(var(--foreground))',
        primary: {
          DEFAULT: 'hsl(var(--primary))',
          foreground: 'hsl(var(--primary-foreground))',
          dim: 'hsl(var(--primary-dim))',
        },
        secondary: {
          DEFAULT: 'hsl(var(--secondary))',
          foreground: 'hsl(var(--secondary-foreground))',
        },
        destructive: {
          DEFAULT: 'hsl(var(--destructive))',
          foreground: 'hsl(var(--destructive-foreground))',
        },
        muted: {
          DEFAULT: 'hsl(var(--muted))',
          foreground: 'hsl(var(--muted-foreground))',
        },
        accent: {
          DEFAULT: 'hsl(var(--accent))',
          foreground: 'hsl(var(--accent-foreground))',
        },
        popover: {
          DEFAULT: 'hsl(var(--popover))',
          foreground: 'hsl(var(--popover-foreground))',
        },
        card: {
          DEFAULT: 'hsl(var(--card))',
          foreground: 'hsl(var(--card-foreground))',
        },
        surface: {
          DEFAULT: 'hsl(var(--surface))',
          elevated: 'hsl(var(--surface-elevated))',
        },
        // Protocol type colors
        'type-http': 'hsl(var(--type-http))',
        'type-tcp': 'hsl(var(--type-tcp))',
        'type-udp': 'hsl(var(--type-udp))',
        // Glow colors
        'glow-primary': 'hsl(var(--glow-primary))',
        'glow-accent': 'hsl(var(--glow-accent))',
      },
      borderRadius: {
        lg: 'var(--radius)',
        md: 'calc(var(--radius) - 2px)',
        sm: 'calc(var(--radius) - 4px)',
        xl: 'calc(var(--radius) + 4px)',
        '2xl': 'calc(var(--radius) + 8px)',
        '3xl': '1.5rem',
      },
      fontFamily: {
        display: ['Clash Display', 'ui-sans-serif', 'system-ui', 'sans-serif'],
        sans: ['Satoshi', 'ui-sans-serif', 'system-ui', 'sans-serif'],
        mono: ['JetBrains Mono', 'ui-monospace', 'SFMono-Regular', 'monospace'],
      },
      fontSize: {
        'display-xl': ['clamp(3rem, 8vw, 6rem)', { lineHeight: '1', letterSpacing: '-0.02em', fontWeight: '800' }],
        'display-lg': ['clamp(2.5rem, 6vw, 4rem)', { lineHeight: '1.1', letterSpacing: '-0.02em', fontWeight: '700' }],
        'display-md': ['clamp(1.75rem, 4vw, 2.5rem)', { lineHeight: '1.2', letterSpacing: '-0.01em', fontWeight: '700' }],
      },
      keyframes: {
        'accordion-down': {
          from: { height: 0 },
          to: { height: 'var(--radix-accordion-content-height)' },
        },
        'accordion-up': {
          from: { height: 'var(--radix-accordion-content-height)' },
          to: { height: 0 },
        },
        'float': {
          '0%, 100%': { transform: 'translateY(0px)' },
          '50%': { transform: 'translateY(-10px)' },
        },
        'glow-pulse': {
          '0%, 100%': {
            boxShadow: '0 0 5px hsl(var(--primary) / 0.3), 0 0 10px hsl(var(--primary) / 0.2)',
          },
          '50%': {
            boxShadow: '0 0 20px hsl(var(--primary) / 0.6), 0 0 40px hsl(var(--primary) / 0.4)',
          },
        },
        'fade-in-up': {
          from: { opacity: 0, transform: 'translateY(30px)' },
          to: { opacity: 1, transform: 'translateY(0)' },
        },
        'fade-in-left': {
          from: { opacity: 0, transform: 'translateX(-30px)' },
          to: { opacity: 1, transform: 'translateX(0)' },
        },
        'fade-in-right': {
          from: { opacity: 0, transform: 'translateX(30px)' },
          to: { opacity: 1, transform: 'translateX(0)' },
        },
        'slide-in-right': {
          from: { opacity: 0, transform: 'translateX(20px)' },
          to: { opacity: 1, transform: 'translateX(0)' },
        },
        'scale-in': {
          from: { opacity: 0, transform: 'scale(0.95)' },
          to: { opacity: 1, transform: 'scale(1)' },
        },
        'blur-in': {
          from: { opacity: 0, filter: 'blur(10px)' },
          to: { opacity: 1, filter: 'blur(0)' },
        },
        'gradient-x': {
          '0%, 100%': { backgroundPosition: '0% 50%' },
          '50%': { backgroundPosition: '100% 50%' },
        },
        'data-flow': {
          '0%': { transform: 'translateX(-100%)' },
          '100%': { transform: 'translateX(100%)' },
        },
        'pulse-ring': {
          '0%': { transform: 'scale(1)', opacity: '1' },
          '100%': { transform: 'scale(1.5)', opacity: '0' },
        },
        'particle-float': {
          '0%, 100%': { transform: 'translate(0, 0) scale(1)', opacity: '0.5' },
          '25%': { transform: 'translate(10px, -20px) scale(1.1)', opacity: '1' },
          '50%': { transform: 'translate(20px, 0) scale(1)', opacity: '0.7' },
          '75%': { transform: 'translate(10px, 20px) scale(0.9)', opacity: '0.5' },
        },
        'scan-line': {
          '0%': { transform: 'translateY(-100%)' },
          '100%': { transform: 'translateY(100%)' },
        },
        'text-shimmer': {
          '0%': { backgroundPosition: '-200% center' },
          '100%': { backgroundPosition: '200% center' },
        },
      },
      animation: {
        'accordion-down': 'accordion-down 0.2s ease-out',
        'accordion-up': 'accordion-up 0.2s ease-out',
        'float': 'float 6s ease-in-out infinite',
        'glow-pulse': 'glow-pulse 3s ease-in-out infinite',
        'fade-in-up': 'fade-in-up 0.8s ease-out forwards',
        'fade-in-left': 'fade-in-left 0.8s ease-out forwards',
        'fade-in-right': 'fade-in-right 0.8s ease-out forwards',
        'slide-in-right': 'slide-in-right 0.5s ease-out forwards',
        'scale-in': 'scale-in 0.6s ease-out forwards',
        'blur-in': 'blur-in 0.6s ease-out forwards',
        'gradient-x': 'gradient-x 15s ease infinite',
        'data-flow': 'data-flow 2s linear infinite',
        'pulse-ring': 'pulse-ring 2s ease-out infinite',
        'particle-float': 'particle-float 8s ease-in-out infinite',
        'scan-line': 'scan-line 4s linear infinite',
        'text-shimmer': 'text-shimmer 3s ease-in-out infinite',
      },
      backgroundImage: {
        'gradient-radial': 'radial-gradient(var(--tw-gradient-stops))',
        'gradient-conic': 'conic-gradient(from 180deg at 50% 50%, var(--tw-gradient-stops))',
        'hero-pattern': 'linear-gradient(135deg, hsl(var(--primary) / 0.05) 0%, transparent 50%, hsl(var(--accent) / 0.05) 100%)',
        'grid-pattern': `linear-gradient(hsl(var(--border)) 1px, transparent 1px),
                         linear-gradient(90deg, hsl(var(--border)) 1px, transparent 1px)`,
        'noise': `url("data:image/svg+xml,%3Csvg viewBox='0 0 200 200' xmlns='http://www.w3.org/2000/svg'%3E%3Cfilter id='noise'%3E%3CfeTurbulence type='fractalNoise' baseFrequency='0.65' numOctaves='3' stitchTiles='stitch'/%3E%3C/filter%3E%3Crect width='100%' height='100%' filter='url(%23noise)'/%3E%3C/svg%3E")`,
      },
      backgroundSize: {
        'grid-40': '40px 40px',
        'grid-60': '60px 60px',
      },
      boxShadow: {
        'glow-sm': '0 0 10px hsl(var(--primary) / 0.3)',
        'glow-md': '0 0 20px hsl(var(--primary) / 0.4), 0 0 40px hsl(var(--primary) / 0.2)',
        'glow-lg': '0 0 30px hsl(var(--primary) / 0.5), 0 0 60px hsl(var(--primary) / 0.3)',
        'glow-accent': '0 0 20px hsl(var(--accent) / 0.4), 0 0 40px hsl(var(--accent) / 0.2)',
        'inner-glow': 'inset 0 0 20px hsl(var(--primary) / 0.1)',
      },
    },
  },
  plugins: [require('tailwindcss-animate')],
}
