import type { MetaAdAccount } from '~/types/automation'

export const vatRatesByCountry: Record<string, { rate: number; name: string; flag: string }> = {
  LA: { rate: 10, name: 'ລາວ (ສປປ ລາວ)', flag: '🇱🇦' },
  TH: { rate: 7, name: 'ໄທ (Thailand)', flag: '🇹🇭' },
  VN: { rate: 5, name: 'ຫວຽດນາມ (Vietnam)', flag: '🇻🇳' },
  SG: { rate: 9, name: 'ສິງກະໂປ (Singapore)', flag: '🇸🇬' },
  MY: { rate: 8, name: 'ມາເລເຊຍ (Malaysia)', flag: '🇲🇾' },
  ID: { rate: 11, name: 'ອິນໂດເນເຊຍ (Indonesia)', flag: '🇮🇩' },
  PH: { rate: 12, name: 'ຟີລິບປິນ (Philippines)', flag: '🇵🇭' },
  KH: { rate: 10, name: 'ກຳປູເຈຍ (Cambodia)', flag: '🇰🇭' },
  JP: { rate: 10, name: 'ຍີ່ປຸ່ນ (Japan)', flag: '🇯🇵' },
  AU: { rate: 10, name: 'ອົດສະຕຣາລີ (Australia)', flag: '🇦🇺' },
  GB: { rate: 20, name: 'ອັງກິດ (United Kingdom)', flag: '🇬🇧' },
  US: { rate: 0, name: 'ສະຫະລັດ (United States)', flag: '🇺🇸' },
}

export function useAdVat() {
  /**
   * Auto-detect VAT rate based on Ad Account country and taxId status
   */
  function detectVatRate(account?: MetaAdAccount | null): { rate: number; countryCode: string; countryName: string; flag: string; isTaxExempt: boolean } {
    if (!account) {
      return { rate: 7, countryCode: 'TH', countryName: 'Default', flag: '🌐', isTaxExempt: false }
    }

    // If business has valid tax ID registered in Meta, VAT on invoice is 0% (Reverse Charge)
    if (account.taxId && account.taxIdStatus === 2) {
      const code = (account.businessCountryCode || 'TH').toUpperCase()
      const c = vatRatesByCountry[code]
      return {
        rate: 0,
        countryCode: code,
        countryName: c?.name || code,
        flag: c?.flag || '🌐',
        isTaxExempt: true,
      }
    }

    const code = (account.businessCountryCode || '').toUpperCase()
    if (code && vatRatesByCountry[code]) {
      const c = vatRatesByCountry[code]
      return {
        rate: c.rate,
        countryCode: code,
        countryName: c.name,
        flag: c.flag,
        isTaxExempt: false,
      }
    }

    // Fallback by currency
    if (account.currency === 'THB') return { rate: 7, countryCode: 'TH', countryName: 'Thailand', flag: '🇹🇭', isTaxExempt: false }
    if (account.currency === 'LAK') return { rate: 10, countryCode: 'LA', countryName: 'Laos', flag: '🇱🇦', isTaxExempt: false }
    if (account.currency === 'VND') return { rate: 5, countryCode: 'VN', countryName: 'Vietnam', flag: '🇻🇳', isTaxExempt: false }

    return { rate: 7, countryCode: 'TH', countryName: 'Default', flag: '🌐', isTaxExempt: false }
  }

  /**
   * Calculate Net Spend, VAT Amount, and Gross Spend
   */
  function computeVat(netSpend: number | null | undefined, vatRate: number) {
    const net = Number(netSpend) || 0
    const vat = net * (vatRate / 100)
    const gross = net + vat
    return {
      netSpend: net,
      vatRate,
      vatAmount: vat,
      grossSpend: gross,
    }
  }

  return {
    detectVatRate,
    computeVat,
    vatRatesByCountry,
  }
}
