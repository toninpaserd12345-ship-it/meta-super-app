import re
content = open("frontend/nuxt.config.ts").read()

content = content.replace(
    "connect-src 'self' https://www.facebook.com https://web.facebook.com https://graph.facebook.com",
    "connect-src 'self' https://www.facebook.com https://web.facebook.com https://graph.facebook.com https://connect.facebook.net"
)

open("frontend/nuxt.config.ts", "w").write(content)
