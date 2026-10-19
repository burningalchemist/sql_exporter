import type { EnhanceAppContext } from 'vitepress'
import { enhanceAppWithTabs } from 'vitepress-plugin-tabs/client'
import DefaultTheme from 'vitepress/theme'
import { Pill, Notice, Underline } from '@theojs/lumen'
import '@theojs/lumen/style'
import './custom.css'

export default {
    extends: DefaultTheme,
    enhanceApp({ app }: EnhanceAppContext) {
        enhanceAppWithTabs(app),
            app.component('Pill', Pill),
            app.component('Underline', Underline),
            app.component('Notice', Notice)
    }
}
