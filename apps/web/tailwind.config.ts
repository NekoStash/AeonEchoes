import type { Config } from 'tailwindcss'

export default <Partial<Config>>{
  darkMode: 'class',
  content: [
    './app.vue',
    './components/**/*.{vue,js,ts}',
    './layouts/**/*.vue',
    './pages/**/*.vue',
    './composables/**/*.{js,ts}',
    './lib/**/*.{js,ts}',
    './stores/**/*.{js,ts}'
  ],
  theme: {
    extend: {
      colors: {
        border: 'hsl(var(--border))',
        input: 'hsl(var(--input))',
        ring: 'hsl(var(--ring))',
        background: 'hsl(var(--background))',
        foreground: 'hsl(var(--foreground))',
        primary: {
          DEFAULT: 'hsl(var(--primary))',
          foreground: 'hsl(var(--primary-foreground))'
        },
        secondary: {
          DEFAULT: 'hsl(var(--secondary))',
          foreground: 'hsl(var(--secondary-foreground))'
        },
        destructive: {
          DEFAULT: 'hsl(var(--destructive))',
          foreground: 'hsl(var(--destructive-foreground))'
        },
        muted: {
          DEFAULT: 'hsl(var(--muted))',
          foreground: 'hsl(var(--muted-foreground))'
        },
        accent: {
          DEFAULT: 'hsl(var(--accent))',
          foreground: 'hsl(var(--accent-foreground))'
        },
        popover: {
          DEFAULT: 'hsl(var(--popover))',
          foreground: 'hsl(var(--popover-foreground))'
        },
        card: {
          DEFAULT: 'hsl(var(--card))',
          foreground: 'hsl(var(--card-foreground))'
        },
        archive: {
          ink: '#070A12',
          panel: '#0D1324',
          glass: '#121A2E',
          line: '#27344E',
          cyan: '#67E8F9',
          violet: '#A78BFA',
          gold: '#F8D16C',
          rose: '#FB7185'
        }
      },
      borderRadius: {
        xl: '1rem',
        '2xl': '1.25rem'
      },
      boxShadow: {
        'star-glow': '0 0 40px rgba(103, 232, 249, 0.16)',
        'violet-glow': '0 0 34px rgba(167, 139, 250, 0.14)'
      },
      backgroundImage: {
        'star-grid': 'radial-gradient(circle at 1px 1px, rgba(148,163,184,.22) 1px, transparent 0)',
        'archive-radial': 'radial-gradient(circle at 20% 20%, rgba(103,232,249,.16), transparent 30%), radial-gradient(circle at 80% 0%, rgba(167,139,250,.14), transparent 28%), linear-gradient(135deg, #070A12 0%, #0D1324 55%, #080B14 100%)'
      },
      fontFamily: {
        sans: ['Inter', 'ui-sans-serif', 'system-ui', 'sans-serif'],
        serif: ['Literata', 'ui-serif', 'Georgia', 'serif'],
        mono: ['JetBrains Mono', 'ui-monospace', 'SFMono-Regular', 'monospace']
      }
    }
  },
  plugins: []
}
