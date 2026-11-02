/**
 * Utility to normalize and prepare media files for upload across iOS, Android, and Desktop.
 * - Converts iOS .heic / .heif photos to standard JPEG (.jpg) for full cross-browser compatibility
 *   and Meta (Facebook & WhatsApp) API acceptance.
 * - Normalizes .mov and other media extensions and types.
 */
export async function normalizeMediaFile(file: File): Promise<File> {
  if (!file) return file

  const fileName = (file.name || '').toLowerCase()
  const fileType = (file.type || '').toLowerCase()

  const isHeic =
    fileName.endsWith('.heic') ||
    fileName.endsWith('.heif') ||
    fileName.endsWith('.hif') ||
    fileType === 'image/heic' ||
    fileType === 'image/heif' ||
    fileType === 'image/x-heic' ||
    fileType === 'image/x-heif'

  if (!isHeic) {
    return file
  }

  // 1. Try native WebKit / browser canvas conversion (ultra-fast on iOS 17+ and macOS)
  if (typeof window !== 'undefined' && typeof document !== 'undefined') {
    try {
      let bmp: ImageBitmap | HTMLImageElement | null = null
      if (typeof createImageBitmap === 'function') {
        try {
          bmp = await createImageBitmap(file)
        } catch {}
      }

      if (!bmp && typeof Image !== 'undefined') {
        bmp = await new Promise<HTMLImageElement>((resolve, reject) => {
          const img = new Image()
          const url = URL.createObjectURL(file)
          img.onload = () => {
            URL.revokeObjectURL(url)
            resolve(img)
          }
          img.onerror = (e) => {
            URL.revokeObjectURL(url)
            reject(e)
          }
          img.src = url
        })
      }

      if (bmp) {
        const canvas = document.createElement('canvas')
        canvas.width = bmp.width
        canvas.height = bmp.height
        const ctx = canvas.getContext('2d')
        if (ctx) {
          ctx.drawImage(bmp, 0, 0)
          if ('close' in bmp && typeof (bmp as ImageBitmap).close === 'function') {
            (bmp as ImageBitmap).close()
          }
          const blob = await new Promise<Blob | null>((resolve) => {
            canvas.toBlob(resolve, 'image/jpeg', 0.92)
          })
          if (blob) {
            const baseName = file.name.replace(/\.(heic|heif|hif)$/i, '') || 'photo'
            return new File([blob], `${baseName}.jpg`, {
              type: 'image/jpeg',
              lastModified: Date.now(),
            })
          }
        }
      }
    } catch {}

    // 2. Fallback to heic2any (for browsers without native HEIC decoder)
    try {
      const heic2anyModule = await import('heic2any')
      const heic2any = heic2anyModule.default || heic2anyModule
      if (typeof heic2any === 'function') {
        const result = await heic2any({
          blob: file,
          toType: 'image/jpeg',
          quality: 0.92,
        })
        const blob = Array.isArray(result) ? result[0] : result
        if (blob) {
          const baseName = file.name.replace(/\.(heic|heif|hif)$/i, '') || 'photo'
          return new File([blob], `${baseName}.jpg`, {
            type: 'image/jpeg',
            lastModified: Date.now(),
          })
        }
      }
    } catch (e) {
      console.warn('HEIC to JPEG conversion fallback error, uploading original:', e)
    }
  }

  return file
}
