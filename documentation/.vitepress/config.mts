import { defineConfig } from 'vitepress'
import { tabsMarkdownPlugin } from 'vitepress-plugin-tabs'

export default defineConfig({
    markdown: {
        config(md) {
            md.use(tabsMarkdownPlugin)
        }
    },
    title: "SQL Exporter",
    description: "Database-agnostic SQL exporter for Prometheus",
    srcDir: './pages',
    base: '/sql_exporter/',
    themeConfig: {
        search: { provider: 'local' },
        nav: [
            { text: 'Home', link: '/' },
            { text: 'Documentation', link: '/overview' }
        ],
        sidebar: {
            '/': [
                {
                    text: 'Introduction',
                    items: [
                        {
                            text: 'Overview', link: '/overview'
                        },
                        { text: 'Features', link: '/overview/features' },
                        { text: 'Architecture', link: '/overview/architecture' },
                        { text: 'Supported databases', link: '/overview/databases' },
                        { text: 'Deployment', link: '/overview/deploy.md' },
                    ]
                },
                {
                    text: 'Configuration',
                    items: [
                        { text: 'General', link: '/config/general' },
                        { text: 'Connection Strings', link: '/config/database_url' },
                        { text: 'Secrets', link: '/config/secrets' },
                        { text: 'Reload', link: '/config/reload' },
                        { text: 'Command-line Parameters', link: '/config/cmdline' },
                        { text: 'Environment Variables', link: '/config/env_vars' },
                        { text: 'Reference', link: '/config' },
                    ]

                },
                //                {
                //                    text: 'Deployment', link: '/deployment',
                //                    items: [
                //                        { text: 'Standalone', link: '/deploy/standalone' },
                //                        { text: 'Container', link: '/deploy/container' },
                //
                //                    ]
                //                },
                {
                    text: 'Advanced',
                    items: [
                        { text: 'Custom drivers', link: '/advanced/custom_drivers' },
                        { text: 'Run as a Windows service', link: '/advanced/windows_svc' },
                        { text: 'TLS and Basic Authentication', link: '/advanced/tls_basicauth' },
                        { text: 'Rate Limiting', link: '/advanced/rate_limiter' },
                        { text: 'Per-query Metrics', link: '/advanced/per_query_metrics' },
                        { text: 'Selective scraping', link: '/advanced/selective_jobs' },
                        { text: 'MySQL Custom Certificates', link: '/advanced/mysql_tls' },
                        { text: 'Scraping Special Backends', link: '/advanced/special_backend_scraping' },
                    ]
                },
                // TODO: Add recipes section
                // {
                //     text: 'Recipes',
                //     items: [
                //         { text: 'Runtime API Examples', link: '/recipes/api-examples' }
                //     ]
                // },
                { text: 'Community', link: '/community' },
                //                { text: 'Glossary', link: '/glossary' },
            ]
        },

        socialLinks: [
            { icon: 'github', link: 'https://github.com/burningalchemist/sql_exporter' }
        ]
    }
})
