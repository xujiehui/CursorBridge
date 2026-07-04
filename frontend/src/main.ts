import { createApp } from 'vue'
import App from './App.vue'
import './styles.css'

const app = createApp(App)

app.config.errorHandler = (err) => {
  const root = document.querySelector('#app')
  if (root) {
    const message = err instanceof Error ? err.message : String(err)
    const stack = err instanceof Error && err.stack ? `\n${err.stack}` : ''
    setTimeout(() => {
      root.innerHTML = `<main class="shell"><div class="error">前端渲染失败：${escapeHTML(message + stack)}</div></main>`
    }, 0)
  }
  console.error(err)
}

app.mount('#app')

function escapeHTML(value: string) {
  return value.replace(/[&<>"']/g, (char) => {
    const entities: Record<string, string> = {
      '&': '&amp;',
      '<': '&lt;',
      '>': '&gt;',
      '"': '&quot;',
      "'": '&#039;'
    }
    return entities[char]
  })
}
