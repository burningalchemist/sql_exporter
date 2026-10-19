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
    srcDir: '.',
    base: '/sql_exporter/',
    themeConfig: {
        logo: '/assets/logo.webp',
        search: { provider: 'local' },
        nav: [
            { text: 'Home', link: '/' },
            { text: 'Documentation', link: 'pages/overview' }
        ],
        sidebar: {
            '/pages/': [
                {
                    text: 'Introduction',
                    items: [
                        {
                            text: 'Overview', link: 'pages/overview'
                        },
                        { text: 'Features', link: 'pages/overview/features' },
                        { text: 'Architecture', link: 'pages/overview/architecture' },
                        { text: 'Supported databases', link: 'pages/overview/databases' },
                        { text: 'Deployment', link: 'pages/overview/deploy.md' },
                    ]
                },
                {
                    text: 'Configuration',
                    items: [
                        { text: 'General', link: 'pages/config/general' },
                        { text: 'Connection Strings', link: 'pages/config/database_url' },
                        { text: 'Secrets', link: 'pages/config/secrets' },
                        { text: 'Reload', link: 'pages/config/reload' },
                        { text: 'Command-line Parameters', link: 'pages/config/cmdline' },
                        { text: 'Environment Variables', link: 'pages/config/env_vars' },
                        { text: 'Reference', link: 'pages/config' },
                    ]

                },
                //                {
                //                    text: 'Deployment', link: 'pages/deployment',
                //                    items: [
                //                        { text: 'Standalone', link: 'pages/deploy/standalone' },
                //                        { text: 'Container', link: 'pages/deploy/container' },
                //
                //                    ]
                //                },
                {
                    text: 'Advanced',
                    items: [
                        { text: 'Custom drivers', link: 'pages/advanced/custom_drivers' },
                        { text: 'Run as a Windows service', link: 'pages/advanced/windows_svc' },
                        { text: 'TLS and Basic Authentication', link: 'pages/advanced/tls_basicauth' },
                        { text: 'Rate Limiting', link: 'pages/advanced/rate_limiter' },
                        { text: 'Per-query Metrics', link: 'pages/advanced/per_query_metrics' },
                        { text: 'Selective scraping', link: 'pages/advanced/selective_jobs' },
                        { text: 'MySQL Custom Certificates', link: 'pages/advanced/mysql_tls' },
                        { text: 'Scraping Special Backends', link: 'pages/advanced/special_backend_scraping' },
                    ]
                },
                // TODO: Add recipes section
                // {
                //     text: 'Recipes',
                //     items: [
                //         { text: 'Runtime API Examples', link: 'pages/recipes/api-examples' }
                //     ]
                // },
                { text: 'Community', link: 'pages/community' },
                //                { text: 'Glossary', link: 'pages/glossary' },
            ]
        },

        socialLinks: [
            { icon: 'github', link: 'https://github.com/burningalchemist/sql_exporter' }
        ]
    }
})
