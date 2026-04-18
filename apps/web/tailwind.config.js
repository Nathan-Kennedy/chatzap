/** @type {import('tailwindcss').Config} */
export default {
    darkMode: ["class"],
    content: ["./index.html", "./src/**/*.{ts,tsx,js,jsx}"],
  theme: {
  	extend: {
  		colors: {
        background: '#0A0A0F',
        sidebar: {
          DEFAULT: '#0F0F17',
        },
        card: {
          DEFAULT: '#15151F',
          hover: '#1C1C28',
          foreground: '#F1F5F9',
        },
        input: '#2e2e3a',
        ring: '#7C3AED',
        border: '#ffffff0f',
        primary: {
          DEFAULT: '#7C3AED',
          hover: '#6D28D9',
          foreground: '#F1F5F9'
        },
        accent: {
          cyan: '#06B6D4',
          DEFAULT: '#1C1C28',
          foreground: '#F1F5F9'
        },
        success: '#10B981',
        warning: '#F59E0B',
        danger: '#EF4444',
        text: {
          primary: '#F1F5F9',
          secondary: '#94A3B8',
          muted: '#475569'
        },
        foreground: '#F1F5F9',
        muted: {
          DEFAULT: '#15151F',
          foreground: '#94A3B8'
        },
        popover: {
          DEFAULT: '#15151F',
          foreground: '#F1F5F9'
        },
        secondary: {
          DEFAULT: '#1C1C28',
          foreground: '#F1F5F9'
        },
        destructive: {
          DEFAULT: '#EF4444',
          foreground: '#F1F5F9'
        }
  		},
  		borderRadius: {
  			lg: '0.75rem',
  			md: '0.5rem',
  			sm: '0.25rem'
  		}
  	}
  },
  plugins: [],
}
