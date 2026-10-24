export interface WorkspacePlanInfo {
  hasPlan: boolean
  plan?: {
    id: string
    name: string
    price: number
    lockedPrice: number
    maxPages?: number
    status: string
    currentPeriodEnd: string
  }
  maxPages: number
  connectedPagesCount: number
  connectedPages: { id: string; name: string; category: string; pictureUrl?: string }[]
  totalAvailablePages: number
  step: 'plan' | 'page' | 'completed'
  isCompleted: boolean
}

export function useWorkspacePlan() {
  const planStatus = useState<WorkspacePlanInfo | null>('workspace.plan_status', () => null)
  const loading = useState<boolean>('workspace.plan_status_loading', () => false)
  const { activeAccount } = useAuth()
  const { l } = useLocale()

  async function fetchPlanStatus(force = false) {
    if (!activeAccount.value?.id) return null
    if (planStatus.value && !force) return planStatus.value

    loading.value = true
    try {
      const data = await $fetch<WorkspacePlanInfo>('/api/proxy/api/v1/onboarding/status')
      planStatus.value = data
      return data
    } catch (err) {
      console.warn('Failed to fetch workspace plan status:', err)
      return null
    } finally {
      loading.value = false
    }
  }

  const isFree = computed(() => {
    return planStatus.value?.plan?.id === 'free' || (!planStatus.value?.plan?.id && Boolean(planStatus.value?.hasPlan))
  })

  const isTrial = computed(() => {
    return planStatus.value?.plan?.status === 'trialing'
  })

  const isPro = computed(() => {
    return !isFree.value && (planStatus.value?.plan?.status === 'active' || planStatus.value?.plan?.status === 'trialing')
  })

  const planName = computed(() => {
    if (!planStatus.value?.hasPlan) {
      return l({ lo: 'ຍັງບໍ່ມີແຜນ', th: 'ยังไม่มีแพ็กเกจ', en: 'No Plan' })
    }
    if (isTrial.value) {
      return l({ lo: 'Pro (ທົດລອງ 7 ວັນ)', th: 'Pro (ทดลอง 7 วัน)', en: 'Pro (7-day Trial)' })
    }
    if (isFree.value) {
      return l({ lo: 'Free Plan (ຟຣີ)', th: 'Free Plan (ฟรี)', en: 'Free Plan' })
    }
    return planStatus.value?.plan?.name || 'Pro Plan'
  })

  const maxPages = computed(() => {
    if (isFree.value) return 1
    return planStatus.value?.maxPages || 10
  })

  const connectedPagesCount = computed(() => {
    return planStatus.value?.connectedPagesCount || 0
  })

  // Auto-refresh when active account changes
  watch(() => activeAccount.value?.id, (newId) => {
    if (newId) {
      fetchPlanStatus(true)
    } else {
      planStatus.value = null
    }
  })

  return {
    planStatus,
    loading,
    isFree,
    isTrial,
    isPro,
    planName,
    maxPages,
    connectedPagesCount,
    fetchPlanStatus,
    refreshPlanStatus: () => fetchPlanStatus(true),
  }
}
