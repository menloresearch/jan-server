import { ApiReferenceReact } from '@scalar/api-reference-react'
import '@scalar/api-reference-react/style.css'
import { useEffect, useState } from 'react'

const ScalarApiReference = () => {
  const [theme, setTheme] = useState('default')
  const [isDarkMode, setIsDarkMode] = useState(true)

  useEffect(() => {
    // Get initial theme
    const getCurrentTheme = () => {
      const theme = document.documentElement.getAttribute('data-theme')
      setIsDarkMode(theme === 'dark')
      return theme === 'light' ? 'default' : 'default'
    }

    setTheme(getCurrentTheme())

    // Watch for theme changes
    const observer = new MutationObserver((mutations) => {
      mutations.forEach((mutation) => {
        if (
          mutation.type === 'attributes' &&
          mutation.attributeName === 'data-theme'
        ) {
          setTheme(getCurrentTheme())
        }
      })
    })

    observer.observe(document.documentElement, {
      attributes: true,
      attributeFilter: ['data-theme'],
    })

    return () => observer.disconnect()
  }, [])

  return (
    <div
      id="scalar-api"
      className="scalar-container"
      style={{
        width: '100%',
        minHeight: 'calc(100vh - 80px)',
        position: 'relative',
        zIndex: 1,
      }}
    >
      <ApiReferenceReact
        configuration={{
          spec: {
            url: '/openapi/openapi.json',
          },
          theme: 'default',
          darkMode: isDarkMode,
          layout: 'modern',
          hideModels: false,
          hideDownloadButton: false,
          hideTestRequestButton: false,
          metaData: {
            title: '👋 Jan API Reference',
            description: "Jan's OpenAI-compatible API server documentation",
          },
          customCss: `
            /* Apply green accent color */
            :root {
              --scalar-color-accent: ${isDarkMode ? '#22c55e' : '#16a34a'} !important;
            }

            /* Minimal sidebar positioning fixes only */
            .scalar-api-reference .scalar-sidebar {
              position: sticky !important;
              top: 4rem !important;
              height: calc(100vh - 4rem) !important;
              overflow-y: auto !important;
            }

            /* Keep search bar visible */
            .scalar-api-reference .scalar-sidebar .scalar-search,
            .scalar-api-reference .scalar-sidebar-header {
              position: sticky !important;
              top: 0 !important;
              z-index: 10 !important;
              background: var(--scalar-background-2, #27272a) !important;
            }

            @media (max-width: 768px) {
              .scalar-api-reference .scalar-sidebar {
                top: 3.5rem !important;
                height: calc(100vh - 3.5rem) !important;
              }
            }
          `,
        }}
      />
      <style jsx>{`
        :global(.scalar-container) {
          --scalar-color-accent: ${isDarkMode
            ? '#22c55e'
            : '#16a34a'} !important;
        }

        /* Responsive adjustments */
        @media (max-width: 50rem) {
          .scalar-container {
            min-height: calc(100vh - 60px);
          }
        }
      `}</style>
    </div>
  )
}

export default ScalarApiReference
