import { createApp } from 'vue'
import { createPinia } from 'pinia'
import { createHead } from '@unhead/vue/client'
import App from './App.vue'
import router from './router'
import { i18n } from './i18n'
import './styles.css'

const app = createApp(App)
const head = createHead()

app.use(head)
app.use(createPinia())
app.use(router)
app.use(i18n)

app.mount('#app')
