export type AppLocale = 'lo' | 'en'

const messages = {
  lo: {
    'nav.ads': 'ຈັດການໂຄສະນາ', 'nav.overview': 'ພາບລວມ', 'nav.chat': 'ສົນທະນາ',
    'nav.channels': 'ເພຈ ແລະ WhatsApp', 'nav.products': 'ສິນຄ້າ', 'nav.replies': 'ຊຸດຂໍ້ຄວາມ',
    'nav.automation': 'ຕອບກັບອັດຕະໂນມັດ', 'nav.guide': 'ຄູ່ມື Automation', 'nav.analytics': 'ລາຍງານ',
    'nav.customers': 'ລູກຄ້າ', 'nav.orders': 'ຄຳສັ່ງຊື້', 'nav.settings': 'ຕັ້ງຄ່າ',
    'common.workspace': 'ພື້ນທີ່ເຮັດວຽກ', 'common.signOut': 'ອອກຈາກລະບົບ', 'common.language': 'ພາສາ',
    'common.back': 'ກັບຄືນ', 'common.cancel': 'ຍົກເລີກ', 'common.continue': 'ດຳເນີນຕໍ່',
    'common.delete': 'ລຶບ', 'common.edit': 'ແກ້ໄຂ', 'common.details': 'ລາຍລະອຽດ', 'common.save': 'ບັນທຶກ', 'common.active': 'ເປີດໃຊ້', 'common.targets': 'ເປົ້າໝາຍ',
    'header.morning': 'ສະບາຍດີຕອນເຊົ້າ', 'header.afternoon': 'ສະບາຍດີຕອນບ່າຍ', 'header.evening': 'ສະບາຍດີຕອນແລງ',
    'auto.eyebrow': 'ສູນຈັດການ AUTOMATION', 'auto.title': 'ຈັດການຕອບກັບອັດຕະໂນມັດ',
    'auto.newTitle': 'ສ້າງ Automation ໃໝ່', 'auto.editTitle': 'ແກ້ໄຂ Automation', 'auto.subtitle': '1 Automation ສາມາດໃຊ້ກັບຫຼາຍໂພສ ຫຼື Campaign ໄດ້',
    'auto.new': 'ສ້າງ Automation', 'auto.list': 'ລາຍການ Automation', 'auto.connected': 'ເປົ້າໝາຍທີ່ເຊື່ອມແລ້ວ',
    'auto.search': 'ຄົ້ນຫາ Automation ຫຼືເປົ້າໝາຍ', 'auto.empty': 'ຍັງບໍ່ມີ Automation',
    'auto.emptyHelp': 'ສ້າງກົດ 1 ຄັ້ງ ແລ້ວເລືອກໂພສ ຫຼື Campaign ໄດ້ຫຼາຍລາຍການ',
    'auto.setupNeeded': 'ຕ້ອງຕຽມ: ສິນຄ້າ, ຊຸດຂໍ້ຄວາມທີ່ເປີດໃຊ້ ແລະ Facebook Page ທີ່ເຊື່ອມແລ້ວ',
    'auto.guide': 'ເບິ່ງຄູ່ມື', 'auto.firstOnly': 'ຕອບສະເພາະຂໍ້ຄວາມທຳອິດ', 'auto.everyMatch': 'ຕອບທຸກຄັ້ງທີ່ກົງເງື່ອນໄຂ',
    'auto.step.setup': 'ຕັ້ງຄ່າ', 'auto.step.trigger': 'ເລືອກເປົ້າໝາຍ', 'auto.step.response': 'ກຳນົດຄຳຕອບ',
    'auto.step.behavior': 'ຄວບຄຸມການສົ່ງ', 'auto.step.review': 'ກວດ ແລະບັນທຶກ',
    'auto.setupTitle': 'ຕັ້ງຊື່ ແລະເລືອກ Facebook Page', 'auto.setupHelp': 'ເລືອກ Page ກ່ອນ ເພື່ອໃຫ້ລະບົບໃຊ້ token ຖືກຕ້ອງ',
    'auto.name': 'ຊື່ Automation', 'auto.nameExample': 'ຕັ້ງຊື່ໃຫ້ຈື່ງ່າຍ ແລະຊອກຫາໄດ້ໄວ', 'auto.page': 'Facebook Page',
    'auto.triggerTitle': 'ໃຫ້ Automation ເຮັດວຽກຢູ່ໃສ?', 'auto.triggerHelp': 'ເລືອກໄດ້ຫຼາຍລາຍການ. Campaign ຈະຮອງຮັບ Ads ໃໝ່ພາຍໃນໂດຍອັດຕະໂນມັດ',
    'auto.posts': 'ໂພສ', 'auto.campaigns': 'Campaign', 'auto.adAccount': 'ບັນຊີໂຄສະນາ', 'auto.alreadyUsed': 'ຖືກໃຊ້ໃນ Automation ອື່ນແລ້ວ',
    'auto.responseTitle': 'ເລືອກສິນຄ້າ ແລະຊຸດຂໍ້ຄວາມ', 'auto.responseHelp': 'ຂໍ້ມູນສິນຄ້າຈະແທນຄ່າໃນຂໍ້ຄວາມ ແລະສົ່ງຕາມລຳດັບທີ່ກຳນົດ',
    'auto.product': 'ສິນຄ້າ', 'auto.replySet': 'ຊຸດຂໍ້ຄວາມ', 'auto.enabledMessages': 'ຂໍ້ຄວາມທີ່ເປີດໃຊ້',
    'auto.sendOrder': 'ຂໍ້ຄວາມ, ຮູບ, ວິດີໂອ ແລະສຽງ ຈະສົ່ງຈາກເທິງລົງລຸ່ມ',
    'auto.behaviorTitle': 'ປ້ອງກັນການຕອບຊ້ຳ', 'auto.behaviorHelp': 'ຄ່າປອດໄພຖືກເປີດໃຫ້ແລ້ວ ແຕ່ສາມາດປ່ຽນໄດ້',
    'auto.firstLabel': 'ຕອບສະເພາະຂໍ້ຄວາມທຳອິດຂອງລູກຄ້າ', 'auto.cooldown': 'ໄລຍະພັກກ່ອນຕອບອີກຄັ້ງ (ວິນາທີ)',
    'auto.cooldownHelp': '0 = ບໍ່ຈຳກັດໄລຍະພັກ; ການປ້ອງກັນຂໍ້ຄວາມທຳອິດຍັງເຮັດວຽກ',
    'auto.reviewTitle': 'ກວດທຸກຢ່າງກ່ອນເປີດໃຊ້', 'auto.reviewHelp': 'ລະບົບຈະບັນທຶກທຸກເປົ້າໝາຍພ້ອມກັນ ຫຼືບໍ່ບັນທຶກເລີຍຖ້າມີຂໍ້ຜິດພາດ',
    'auto.automation': 'Automation', 'auto.response': 'ຄຳຕອບ', 'auto.create': 'ສ້າງ ແລະເປີດໃຊ້', 'auto.update': 'ບັນທຶກການແກ້ໄຂ',
    'guide.title': 'ຄູ່ມືສ້າງ Automation', 'guide.subtitle': 'ຈາກການກຽມຂໍ້ມູນ ຫາການທົດສອບຂໍ້ຄວາມຈິງ',
  },
  en: {
    'nav.ads': 'Ads Manager', 'nav.overview': 'Overview', 'nav.chat': 'Live Chat', 'nav.channels': 'Pages & WhatsApp',
    'nav.products': 'Products', 'nav.replies': 'Reply Sets', 'nav.automation': 'Auto Replies', 'nav.guide': 'Automation Guide',
    'nav.analytics': 'Analytics', 'nav.customers': 'Customers', 'nav.orders': 'Orders', 'nav.settings': 'Settings',
    'common.workspace': 'Workspace', 'common.signOut': 'Sign out', 'common.language': 'Language', 'common.back': 'Back',
    'common.cancel': 'Cancel', 'common.continue': 'Continue', 'common.delete': 'Delete', 'common.edit': 'Edit', 'common.details': 'Details', 'common.save': 'Save', 'common.active': 'Active', 'common.targets': 'Targets',
    'header.morning': 'Good morning', 'header.afternoon': 'Good afternoon', 'header.evening': 'Good evening',
    'auto.eyebrow': 'AUTOMATION CENTER', 'auto.title': 'Auto Reply Manager', 'auto.newTitle': 'New Automation', 'auto.editTitle': 'Edit Automation',
    'auto.subtitle': 'One automation can cover many Posts or Campaigns.', 'auto.new': 'New Automation', 'auto.list': 'Automations',
    'auto.connected': 'Connected targets', 'auto.search': 'Search automation or target', 'auto.empty': 'No automations yet',
    'auto.emptyHelp': 'Create one workflow, then attach as many Posts or Campaigns as needed.',
    'auto.setupNeeded': 'Setup needed: create a Product, an enabled Reply Set, and connect a Facebook Page.',
    'auto.guide': 'View guide', 'auto.firstOnly': 'First message only', 'auto.everyMatch': 'Every matching message',
    'auto.step.setup': 'Setup', 'auto.step.trigger': 'Targets', 'auto.step.response': 'Response', 'auto.step.behavior': 'Behavior', 'auto.step.review': 'Review',
    'auto.setupTitle': 'Name it and choose a Facebook Page', 'auto.setupHelp': 'Choose the Page first so the correct token is used.',
    'auto.name': 'Automation name', 'auto.nameExample': 'Use a clear, searchable name', 'auto.page': 'Facebook Page',
    'auto.triggerTitle': 'Where should this automation run?', 'auto.triggerHelp': 'Select many targets. Campaigns automatically include future Ads inside them.',
    'auto.posts': 'Posts', 'auto.campaigns': 'Campaigns', 'auto.adAccount': 'Ad Account', 'auto.alreadyUsed': 'Already used by another automation',
    'auto.responseTitle': 'Choose Product and Reply Set', 'auto.responseHelp': 'Product data fills variables and enabled messages send in order.',
    'auto.product': 'Product', 'auto.replySet': 'Reply Set', 'auto.enabledMessages': 'enabled messages',
    'auto.sendOrder': 'Text, image, video and audio send from top to bottom.', 'auto.behaviorTitle': 'Prevent repeat replies',
    'auto.behaviorHelp': 'Safe defaults are enabled and can be changed.', 'auto.firstLabel': 'Reply only to the customer’s first matching message',
    'auto.cooldown': 'Cooldown before replying again (seconds)', 'auto.cooldownHelp': '0 disables cooldown; first-message protection still works.',
    'auto.reviewTitle': 'Review before activation', 'auto.reviewHelp': 'All selected targets are saved together, or none are saved if an error occurs.',
    'auto.automation': 'Automation', 'auto.response': 'Response', 'auto.create': 'Create & activate', 'auto.update': 'Save changes',
    'guide.title': 'Automation Guide', 'guide.subtitle': 'From preparing data to testing a real message.',
  },
} as const

export function useLocale() {
  const locale = useCookie<AppLocale>('app-locale', { default: () => 'lo', sameSite: 'lax', maxAge: 60 * 60 * 24 * 365 })
  const t = (key: string) => (messages[locale.value] as Record<string, string>)[key] || (messages.en as Record<string, string>)[key] || key
  const setLocale = (value: AppLocale) => { locale.value = value }
  useHead(() => ({ htmlAttrs: { lang: locale.value } }))
  return { locale, setLocale, t }
}
