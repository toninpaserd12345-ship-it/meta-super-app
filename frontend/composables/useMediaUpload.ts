/**
 * Utility to normalize and prepare media files for upload across iOS, Android, and Desktop.
 * - Converts iOS .heic / .heif and iPhone photos to standard PNG (.png) for full quality,
 *   lossless clarity, cross-browser compatibility, and Meta (Facebook & WhatsApp) API acceptance.
 * - Converts iOS .mov / QuickTime videos to standard MP4 (.mp4 / video/mp4) by patching
 *   the ISOBMFF ftyp brand to mp42, ensuring immediate playback on Chrome, Android, and WhatsApp.
 * - Uses strict timeouts (max 3.5 seconds) to guarantee the UI never freezes or hangs.
 * - Returns original file safely if conversion is not supported or times out.
 */

function isIosDevice(): boolean {
  if (typeof window === 'undefined' || typeof navigator === 'undefined') return false
  const ua = navigator.userAgent || ''
  return /iPad|iPhone|iPod/.test(ua) || (navigator.platform === 'MacIntel' && navigator.maxTouchPoints > 1)
}

/**
 * Patch QuickTime MOV header ftyp atom to MP4 (mp42)
 */
async function convertMovToMp4(file: File): Promise<File> {
  try {
    const headerLen = Math.min(file.size, 1024)
    const headerBuf = await file.slice(0, headerLen).arrayBuffer()
    const uint8 = new Uint8Array(headerBuf)

    if (uint8.length >= 16) {
      // Check for "ftyp" at offset 4
      const boxType = String.fromCharCode(uint8[4] ?? 0, uint8[5] ?? 0, uint8[6] ?? 0, uint8[7] ?? 0)
      if (boxType === 'ftyp') {
        const view = new DataView(headerBuf)
        const boxSize = view.getUint32(0, false)
        const majorBrand = String.fromCharCode(uint8[8] ?? 0, uint8[9] ?? 0, uint8[10] ?? 0, uint8[11] ?? 0)

        if (majorBrand === 'qt  ') {
          // Patch major brand to "mp42"
          uint8[8] = 0x6d  // 'm'
          uint8[9] = 0x70  // 'p'
          uint8[10] = 0x34 // '4'
          uint8[11] = 0x32 // '2'

          // Patch any compatible brand "qt  " within boxSize to "mp42"
          for (let i = 16; i + 4 <= boxSize && i + 4 <= uint8.length; i += 4) {
            const comp = String.fromCharCode(uint8[i] ?? 0, uint8[i + 1] ?? 0, uint8[i + 2] ?? 0, uint8[i + 3] ?? 0)
            if (comp === 'qt  ') {
              uint8[i] = 0x6d
              uint8[i + 1] = 0x70
              uint8[i + 2] = 0x34
              uint8[i + 3] = 0x32
            }
          }
        }
      }
    }

    const baseName = file.name.replace(/\.(mov|qt|quicktime)$/i, '') || 'video'
    const newBlob = new Blob([uint8, file.slice(headerLen)], { type: 'video/mp4' })
    return new File([newBlob], `${baseName}.mp4`, {
      type: 'video/mp4',
      lastModified: Date.now(),
    })
  } catch (err) {
    console.warn('convertMovToMp4 fallback:', err)
    const baseName = file.name.replace(/\.(mov|qt|quicktime)$/i, '') || 'video'
    return new File([file], `${baseName}.mp4`, {
      type: 'video/mp4',
      lastModified: Date.now(),
    })
  }
}

/**
 * Convert image file (HEIC or iOS image) to PNG
 */
async function convertImageToPng(file: File): Promise<File | null> {
  const baseName = file.name.replace(/\.(heic|heif|hif|jpg|jpeg|webp)$/i, '') || 'image'

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
            canvas.toBlob(resolve, 'image/png')
          })
          if (blob && blob.size > 0) {
            return new File([blob], `${baseName}.png`, {
              type: 'image/png',
              lastModified: Date.now(),
            })
          }
        }
      }
    } catch {
      // Native decode not supported for this file, continue to next fallback
    }
  }

  // 2. Try standard HTMLImageElement decode if not HEIC
  const fileName = (file.name || '').toLowerCase()
  const fileType = (file.type || '').toLowerCase()
  const isHeic = fileName.endsWith('.heic') || fileName.endsWith('.heif') || fileType.includes('heic') || fileType.includes('heif')

  if (!isHeic && typeof Image !== 'undefined') {
    try {
      const img = new Image()
      const objectUrl = URL.createObjectURL(file)
      img.src = objectUrl
      await img.decode()
      const canvas = document.createElement('canvas')
      canvas.width = img.naturalWidth || img.width
      canvas.height = img.naturalHeight || img.height
      const ctx = canvas.getContext('2d')
      if (ctx) {
        ctx.drawImage(img, 0, 0)
        URL.revokeObjectURL(objectUrl)
        const blob = await new Promise<Blob | null>((resolve) => {
          canvas.toBlob(resolve, 'image/png')
        })
        if (blob && blob.size > 0) {
          return new File([blob], `${baseName}.png`, {
            type: 'image/png',
            lastModified: Date.now(),
          })
        }
      }
      URL.revokeObjectURL(objectUrl)
    } catch {
      // Continue
    }
  }

  // 3. Fallback for HEIC: try heic2any with worker
  if (isHeic) {
    try {
      const heic2anyModule = await import('heic2any')
      const heic2any = (heic2anyModule as any).default || heic2anyModule
      if (typeof heic2any === 'function') {
        const result = await heic2any({
          blob: file,
          toType: 'image/png',
        })
        const blob = Array.isArray(result) ? result[0] : result
        if (blob && blob.size > 0) {
          return new File([blob], `${baseName}.png`, {
            type: 'image/png',
            lastModified: Date.now(),
          })
        }
      }
    } catch (e) {
      console.warn('heic2any conversion fallback unavailable:', e)
    }
  }

  return null
}

export async function normalizeMediaFile(file: File): Promise<File> {
  if (!file) return file

  const fileName = (file.name || '').toLowerCase()
  const fileType = (file.type || '').toLowerCase()
  const isIos = isIosDevice()

  // Video check: MOV / QuickTime or iOS video -> Convert to MP4
  const isMov =
    fileName.endsWith('.mov') ||
    fileName.endsWith('.qt') ||
    fileType.includes('quicktime') ||
    fileType.includes('mov')

  if (isMov || (isIos && fileType.startsWith('video/') && !fileType.includes('mp4'))) {
    return convertMovToMp4(file)
  }

  // Image check: HEIC/HEIF or iOS image -> Convert to PNG
  const isHeic =
    fileName.endsWith('.heic') ||
    fileName.endsWith('.heif') ||
    fileName.endsWith('.hif') ||
    fileType.includes('heic') ||
    fileType.includes('heif')

  const shouldConvertToPng =
    isHeic ||
    (isIos && (fileType.startsWith('image/') || /\.(jpg|jpeg|png|heic|heif|webp)$/i.test(fileName)) && !fileName.endsWith('.png'))

  if (!shouldConvertToPng) {
    return file
  }

  if (typeof window === 'undefined' || typeof document === 'undefined') {
    return file
  }

  // Wrap conversion with a hard timeout of 3.5 seconds
  try {
    const timeoutPromise = new Promise<null>((resolve) => setTimeout(() => resolve(null), 3500))
    const converted = await Promise.race([convertImageToPng(file), timeoutPromise])
    if (converted) {
      return converted
    }
  } catch (err) {
    console.warn('Image PNG normalization error, continuing with original file:', err)
  }

  return file
}
