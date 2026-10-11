export const useNavDrawer = () => {
  const isOpen = useState<boolean>('navDrawerOpen', () => false)
  const toggle = () => {
    isOpen.value = !isOpen.value
  }
  const close = () => {
    isOpen.value = false
  }
  const open = () => {
    isOpen.value = true
  }
  return { isOpen, toggle, close, open }
}
