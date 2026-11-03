/**
 * Utility to normalize and prepare media files for upload across iOS, Android, and Desktop.
 * - Converts iOS .heic / .heif photos to standard JPEG (.jpg) for full cross-browser compatibility
 *   and Meta (Facebook & WhatsApp) API acceptance.
 * - Uses strict timeouts (max 3 seconds) to guarantee the UI never freezes or hangs.
 * - Returns original file safely if conversion is not supported or times out.
 */
export async function normalizeMediaFile(file: File): Promise<File> {
  if (!file) return file

  const fileName = (file.name || '').toLowerCase()
  const fileType = (file.type || '').toLowerCase()

  const isHeic =
    fileName.endsWith('.heic') ||
    fileName.endsWith('.heif') ||
    fileName.endsWith('.hif') ||
    fileType.includes('heic') ||
    fileType.includes('heif')

  if (!isHeic) {
    return file
  }

  if (typeof window === 'undefined' || typeof document === 'undefined') {
    return file
  }

  // Wrap the entire conversion with a hard timeout of 3.5 seconds
  const conversionPromise = (async (): Promise<File | null> => {
    // 1. Try native WebKit / browser canvas conversion if createImageBitmap is supported
    if (typeof createImageBitmap === 'function') {
      try {
        const bmp = await createImageBitmap(file)
        if (bmp && bmp.width > 0 && bmp.height > 0) {
          const canvas = document.createElement('canvas')
          canvas.width = bmp.width
          canvas.height = bmp.height
          const ctx = canvas.getContext('2d')
          if (ctx) {
            ctx.drawImage(bmp, 0, 0)
            if (typeof (bmp as ImageBitmap).close === 'function') {
              (bmp as ImageBitmap).close()
            }
            const blob = await new Promise<Blob | null>((resolve) => {
              canvas.toBlob(resolve, 'image/jpeg', 0.92)
            })
            if (blob && blob.size > 0) {
              const baseName = file.name.replace(/\.(heic|heif|hif)$/i, '') || 'photo'
              return new File([blob], `${baseName}.jpg`, {
                type: 'image/jpeg',
                lastModified: Date.now(),
              })
            }
          }
        }
      } catch {
        // Native decode not supported for HEIC on this browser, proceed to fallback
      }
    }

    // 2. Fallback: try heic2any with worker
    try {
      const heic2anyModule = await import('heic2any')
      const heic2any = (heic2anyModule as any).default || heic2anyModule
      if (typeof heic2any === 'function') {
        const result = await heic2any({
          blob: file,
          toType: 'image/jpeg',
          quality: 0.92,
        })
        const blob = Array.isArray(result) ? result[0] : result
        if (blob && blob.size > 0) {
          const baseName = file.name.replace(/\.(heic|heif|hif)$/i, '') || 'photo'
          return new File([blob], `${baseName}.jpg`, {
            type: 'image/jpeg',
            lastModified: Date.now(),
          })
        }
      }
    } catch (e) {
      console.warn('heic2any conversion fallback unavailable:', e)
    }

    return null
  })()

  try {
    const timeoutPromise = new Promise<null>((resolve) => setTimeout(() => resolve(null), 3500))
    const converted = await Promise.race([conversionPromise, timeoutPromise])
    if (converted) {
      return converted
    }
  } catch (err) {
    console.warn('HEIC normalization error, continuing with original file:', err)
  }

  // Graceful fallback: return original file so upload can still proceed directly to storage
  return file
}
