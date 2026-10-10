import re
content = open("backend/internal/delivery/httpx/router.go").read()

content = content.replace(
    'meta.Post("/whatsapp/signup/complete", h.metaWhatsAppSignupComplete)',
    'meta.Post("/whatsapp/signup/complete", h.metaWhatsAppSignupComplete)\n\t\tmeta.Post("/whatsapp/manual", h.metaWhatsAppManual)'
)

open("backend/internal/delivery/httpx/router.go", "w").write(content)
