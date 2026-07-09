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
        surface: {
          DEFAULT: 'hsl(var(--surface))',
          foreground: 'hsl(var(--surface-foreground))',
          muted: 'hsl(var(--surface-muted))',
          elevated: 'hsl(var(--surface-elevated))',
          raised: 'hsl(var(--surface-raised))',
          sunken: 'hsl(var(--surface-sunken))'
        },
        state: {
          info: {
            DEFAULT: 'hsl(var(--state-info))',
            foreground: 'hsl(var(--state-info-foreground))',
            surface: 'hsl(var(--state-info-surface))',
            border: 'hsl(var(--state-info-border))'
          },
          success: {
            DEFAULT: 'hsl(var(--state-success))',
            foreground: 'hsl(var(--state-success-foreground))',
            surface: 'hsl(var(--state-success-surface))',
            border: 'hsl(var(--state-success-border))'
          },
          warning: {
            DEFAULT: 'hsl(var(--state-warning))',
            foreground: 'hsl(var(--state-warning-foreground))',
            surface: 'hsl(var(--state-warning-surface))',
            border: 'hsl(var(--state-warning-border))'
          },
          danger: {
            DEFAULT: 'hsl(var(--state-danger))',
            foreground: 'hsl(var(--state-danger-foreground))',
            surface: 'hsl(var(--state-danger-surface))',
            border: 'hsl(var(--state-danger-border))'
          }
        },
        primary: {
          DEFAULT: 'hsl(var(--primary))',
          foreground: 'hsl(var(--primary-foreground))'
        },
        secondary: {
          DEFAULT: 'hsl(var(--secondary))',
          foreground: 'hsl(var(--secondary-foreground))'
        },
        destructive: {
          DEFAULT: 'hsl(var(--state-danger))',
          foreground: 'hsl(var(--state-danger-foreground))'
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
      spacing: {
        'density-xs': 'var(--density-xs)',
        'density-sm': 'var(--density-sm)',
        'density-md': 'var(--density-md)',
        'density-lg': 'var(--density-lg)',
        'density-xl': 'var(--density-xl)',
        'layout-gutter': 'var(--layout-gutter)'
      },
      width: {
        sidebar: 'var(--layout-width-sidebar)',
        'sidebar-collapsed': 'var(--layout-width-sidebar-collapsed)'
      },
      height: {
        topbar: 'var(--layout-height-topbar)'
      },
      maxWidth: {
        'layout-page': 'var(--layout-width-page)',
        'layout-readable': 'var(--layout-width-readable)',
        'layout-panel': 'var(--layout-width-panel)'
      },
      minHeight: {
        main: 'calc(100dvh - var(--layout-height-topbar))'
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
