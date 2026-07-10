import type { Config } from 'tailwindcss'

export default <Partial<Config>>{
  darkMode: 'class',
  content: [
    './app.vue',
    './components/**/*.{vue,js,ts}',
    './entities/**/*.{vue,js,ts}',
    './features/**/*.{vue,js,ts}',
    './layouts/**/*.vue',
    './pages/**/*.vue',
    './composables/**/*.{js,ts}',
    './lib/**/*.{js,ts}',
    './shared/**/*.{vue,js,ts}',
    './stores/**/*.{js,ts}',
    './widgets/**/*.{vue,js,ts}'
  ],
  theme: {
    borderRadius: {
      none: '0px',
      sm: '0px',
      DEFAULT: '0px',
      md: '0px',
      lg: '0px',
      xl: '0px',
      '2xl': '0px',
      '3xl': '0px',
      full: '0px'
    },
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
      fontFamily: {
        sans: ['Inter', 'Noto Sans SC', 'Segoe UI', 'ui-sans-serif', 'system-ui', 'sans-serif'],
        serif: ['Source Han Serif SC', 'Noto Serif SC', 'Songti SC', 'ui-serif', 'Georgia', 'serif'],
        mono: ['JetBrains Mono', 'Cascadia Code', 'ui-monospace', 'SFMono-Regular', 'monospace']
      }
    }
  },
  plugins: []
}
