import re
content = open("backend/internal/delivery/httpx/handler.go").read()

content = content.replace(
    'auth.Post("/api/v1/meta/whatsapp/signup/complete", requireAccount(h.auth, domain.ClaimPagesConnect), h.completeWhatsAppSignup)',
    'auth.Post("/api/v1/meta/whatsapp/signup/complete", requireAccount(h.auth, domain.ClaimPagesConnect), h.completeWhatsAppSignup)\n\tauth.Post("/api/v1/meta/whatsapp/manual", requireAccount(h.auth, domain.ClaimPagesConnect), h.metaWhatsAppManual)'
)

open("backend/internal/delivery/httpx/handler.go", "w").write(content)
