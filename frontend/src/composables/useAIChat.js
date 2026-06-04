import { useRouter } from 'vue-router'

/**
 * Navigate to AI chat with context pre-filled.
 * The AI page will auto-submit the context as the first message.
 */
export function useAIChat() {
  const router = useRouter()

  function goAIAnalysis(context) {
    const params = new URLSearchParams()
    if (context.type) params.set('type', context.type)
    if (context.title) params.set('title', context.title)
    if (context.id) params.set('id', context.id)
    if (context.summary) params.set('summary', context.summary)

    // Store richer context in sessionStorage so AI page can pick it up
    sessionStorage.setItem('ai_context_hint', JSON.stringify({
      path: router.currentRoute.value?.fullPath || '',
      title: context.title || '',
      type: context.type || '',
      id: context.id || '',
      summary: context.summary || ''
    }))

    router.push('/ai?' + params.toString())
  }

  return { goAIAnalysis }
}
