# 📘 ປຶ້ມຄູ່ມືລະບົບສະບັບສົມບູນ: ສະຖາປັດຕະຍະກຳ, ການຕິດຕັ້ງ, ແລະ ການຂະຫຍາຍລະບົບ
**META-SUPER-APP: Production End-to-End System Manual & Operations Guide**

---

## 📑 ສາລະບານ (Table of Contents)
1. [ພາບລວມຂອງລະບົບ ແລະ ຟັງຊັນຫຼັກ (System Overview)](#1-ພາບລວມຂອງລະບົບ-ແລະ-ຟັງຊັນຫຼັກ)
2. [ສະຖາປັດຕະຍະກຳດ້ານເທັກນິກ (Technical Architecture)](#2-ສະຖາປັດຕະຍະກຳດ້ານເທັກນິກ)
3. [ງົບປະມານ RAM 1GB ແລະ ຄວາມປອດໄພ (Memory & Safety Budget)](#3-ງົບປະມານ-ram-1gb-ແລະ-ຄວາມປອດໄພ)
4. [ຄູ່ມືການຕິດຕັ້ງ Server ຕັ້ງແຕ່ເລີ່ມຕົ້ນ (Zero-to-Hero Server Setup)](#4-ຄູ່ມືການຕິດຕັ້ງ-server-ຕັ້ງແຕ່ເລີ່ມຕົ້ນ)
5. [ການຕັ້ງຄ່າ Meta Developer App & Webhooks](#5-ການຕັ້ງຄ່າ-meta-developer-app--webhooks)
6. [ການ Deploy Frontend ເທິງ Cloudflare Pages](#6-ການ-deploy-frontend-ເທິງ-cloudflare-pages)
7. [ລະບົບ Backup ອັດຕະໂນມັດຂຶ້ນ Cloudflare R2](#7-ລະບົບ-backup-ອັດຕະໂນມັດຂຶ້ນ-cloudflare-r2)
8. [ຄູ່ມືການຂະຫຍາຍລະບົບ ແລະ ຍ້າຍ Database (Scaling to Server 2)](#8-ຄູ່ມືການຂະຫຍາຍລະບົບ-ແລະ-ຍ້າຍ-database)
9. [ຄູ່ມືການແກ້ໄຂບັນຫາ ແລະ ຄຳສັ່ງປະຈຳວັນ (Cheat Sheet & Troubleshooting)](#9-ຄູ່ມືການແກ້ໄຂບັນຫາ-ແລະ-ຄຳສັ່ງປະຈຳວັນ)
10. [ແຜນທີ່ຄວາມຮູ້ທີ່ຕ້ອງສຶກສາຕໍ່ຍອດ (Knowledge Roadmap)](#10-ແຜນທີ່ຄວາມຮູ້ທີ່ຕ້ອງສຶກສາຕໍ່ຍອດ)
11. [ວັດຈະນານຸກົມ ແລະ ຄຳອະທິບາຍທຸກໂຕອັກສອນ (Complete Line-by-Line Technical Dictionary)](#11-ວັດຈະນານຸກົມ-ແລະ-ຄຳອະທິບາຍທຸກໂຕອັກສອນ)

---

## 1. ພາບລວມຂອງລະບົບ ແລະ ຟັງຊັນຫຼັກ

**META-SUPER-APP** ແມ່ນລະບົບ Multi-tenant Omni-channel Commerce & Messaging Platform ທີ່ອອກແບບມາເພື່ອຮອງຮັບທຸລະກິດ ແລະ ຮ້ານຄ້າອອນລາຍຢ່າງຄົບວົງຈອນ:

* **Workspace & Multi-tenancy**: 1 ບັນຊີຜູ້ໃຊ້ສາມາດມີຫຼາຍ Workspace (ຮ້ານຄ້າ), ແຍກຂໍ້ມູນ, ລູກຄ້າ, ສິນຄ້າ, ແລະ ອໍເດີ ອອກຈາກກັນ 100%.
* **Meta Graph API Integration**: ເຊື່ອມຕໍ່ Facebook Pages, Instagram, WhatsApp Cloud API, ແລະ Meta Ads Manager.
* **Auto-Reply & AI Automation**: ລະບົບຕອບກັບອັດຕະໂນມັດຕາມຄີເວີດ, ດັກຈັບເລກໂທລະສັບ, ປິດການຂາຍ, ແລະ ສະຫຼຸບອໍເດີອັດຕະໂນມັດ.
* **Realtime Live Chat**: ຮັບສົ່ງຂໍ້ຄວາມແຊັດແບບ Instant ຜ່ານ WebSockets ພ້ອມລະບົບ SeqID ເພື່ອປ້ອງກັນຂໍ້ຄວາມຫຼຸດຫາຍ.
* **Warehouse & Order Management**: ຄຸ້ມຄອງສາງສິນຄ້າ (Stock Cut), ຕັດສະຕັອກທັນທີເມື່ອມີອໍເດີ, ພ້ອມອອກບິນ ແລະ ເຊັກສະລິບ.
* **Meta Conversions API (CAPI)**: ສົ່ງ Event ການຊື້-ຂາຍ (Purchase/Lead) ກັບຄືນຫາ Meta Ads ແບບ Server-to-Server ເພື່ອໃຫ້ຍິງ Ads ໄດ້ແມ້ນຢຳ.

---

## 2. ສະຖາປັດຕະຍະກຳດ້ານເທັກນິກ

ລະບົບຖືກອອກແບບດ້ວຍແນວຄິດ **Decoupled Edge-First Architecture** ເພື່ອໃຫ້ສາມາດແລ່ນໄດ້ຢ່າງສະບາຍເທິງ Server ຂະໜາດນ້ອຍ (1 vCPU, 1 GB RAM):

```mermaid
flowchart TD
    classDef edgeStyle fill:#f38020,stroke:#d35400,stroke-width:2px,color:#fff;
    classDef hostStyle fill:#2c3e50,stroke:#1a252f,stroke-width:3px,color:#fff;
    classDef proxyStyle fill:#2980b9,stroke:#1f618d,stroke-width:2px,color:#fff;
    classDef appStyle fill:#27ae60,stroke:#1e8449,stroke-width:2px,color:#fff;
    classDef cacheStyle fill:#e74c3c,stroke:#c0392b,stroke-width:2px,color:#fff;
    classDef coreDbStyle fill:#34495e,stroke:#1c2833,stroke-width:2px,color:#fff;
    classDef chatDbStyle fill:#1a5276,stroke:#154360,stroke-width:2px,color:#fff;
    classDef r2Style fill:#16a085,stroke:#117a65,stroke-width:2px,color:#fff;

    subgraph ZoneEdge["🌐 1. EDGE NETWORK & SEPARATE FRONTEND"]
        Users["👥 Shop Owners & Team\n(Browser Dashboard)"]:::edgeStyle
        CFPages["⚡ Cloudflare Pages (Frontend)\n- Domain: app.yourdomain.com\n- 300+ Edge Data Centers\n- ໃຊ້ RAM Server ຫຼັກ = 0 MB!"]:::edgeStyle
        MetaHook["📡 Meta Webhooks\n(Facebook, WhatsApp, Ads)"]:::edgeStyle
        CF_DNS["🛡️ Cloudflare DNS & Proxy\n- Domain: api.yourdomain.com\n- SSL & Anti-DDoS Protection"]:::edgeStyle
    end

    subgraph ZoneDroplet["🖥️ 2. DIGITALOCEAN DROPLET (1 vCPU, 1 GB RAM, 25 GB SSD)"]
        subgraph OSLevel["Linux Kernel & Memory Safety"]
            SWAP["💾 2 GB SSD Swapfile (OOM Killer Insurance)"]
        end

        subgraph IngressLayer["Reverse Proxy (Port 80 / 443)"]
            Caddy["🔌 Caddy 2 Reverse Proxy\n- Auto Let's Encrypt SSL\n- RAM: ~30MB"]:::proxyStyle
        end

        subgraph DockerNet["🐳 Internal Docker Network (meta-internal-net)"]
            GoBackend["⚡ Go Backend API (Fiber)\n- GOMEMLIMIT=200MiB\n- High Concurrency (10,000+ Req/s)\n- Dual DB Pool: Core + Chat"]:::appStyle
            Redis["⚡ Redis 7 In-Memory Cache\n- maxmemory: 128MB\n- Deduplication & Distributed Locks"]:::cacheStyle

            subgraph DualPostgres["Dual-Engine PostgreSQL 16 (Isolated Containers)"]
                CoreDB[("🗄️ Container: postgres-core\n- Host: postgres-core:5432\n- DB: meta_core_db\n- Users, Orders, Products, Rules")]:::coreDbStyle
                ChatDB[("💬 Container: postgres-chat\n- Host: postgres-chat:5432\n- DB: meta_chat_db\n- sync_commit=off (10x ໄວ)\n- chat_message_models (SeqID)")]:::chatDbStyle
            end
        end
    end

    subgraph ZoneOffsite["☁️ 3. OFFSITE DISASTER RECOVERY & MEDIA"]
        CloudflareR2["📦 Cloudflare R2 Storage\n- Encrypted Daily DB Dumps (.sql.zst)\n- Slips & Product Images\n- 0$ Egress Bandwidth Fees"]:::r2Style
    end

    Users --> CFPages
    CFPages -->|API Requests| CF_DNS
    MetaHook --> CF_DNS
    CF_DNS --> Caddy
    Caddy --> GoBackend
    GoBackend <--> Redis
    GoBackend -->|DATABASE_URL| CoreDB
    GoBackend -->|CHAT_DATABASE_URL| ChatDB
    CoreDB -.->|Daily Backup| CloudflareR2
    ChatDB -.->|Daily Backup| CloudflareR2
```

---

## 3. ງົບປະມານ RAM 1GB ແລະ ຄວາມປອດໄພ

Server ຂະໜາດ 1GB RAM ຕ້ອງມີການກຳນົດຂອບເຂດ Memory Limits ຢ່າງເຄັ່ງຄັດ ເພື່ອປ້ອງກັນ **Linux OOM Killer**:

| Service / Container | RAM ທີ່ໃຊ້ປົກກະຕິ | Memory Limit (ສູງສຸດ) | ຄຳອະທິບາຍ |
| :--- | :--- | :--- | :--- |
| **Ubuntu OS & System Services** | ~200 MB | ~250 MB | Linux Kernel, SSH, Systemd |
| **Caddy 2 Reverse Proxy** | ~25 MB | 40 MB | ຮັບ SSL, HTTP/2, Reverse Proxy ໄວ ແລະ ເບົາຫຼາຍ |
| **Go Backend API (Fiber)** | ~100 MB | 200 MB | ຕັ້ງຄ່າ `GOMEMLIMIT=200MiB`, `GOGC=80` |
| **Redis 7 In-Memory Cache** | ~60 MB | 128 MB | ຕັ້ງ `maxmemory 128mb`, ນະໂຍບາຍ `allkeys-lru` |
| **PostgreSQL Core Container** | ~120 MB | 200 MB | `shared_buffers = 128MB`, `max_connections = 40` |
| **PostgreSQL Chat Container** | ~120 MB | 200 MB | `shared_buffers = 128MB`, `sync_commit = off`, `max_conns = 60` |
| **Swap Space (ເທິງ SSD)** | — | **2,048 MB (2GB)** | ເປັນກັນຊົນສຳຮອງ ເວລາ Traffic ຂຶ້ນກະທັນຫັນ |
| **ລວມ Physical RAM ທີ່ໃຊ້** | **~625 MB** | **~1,018 MB** | **ປອດໄພ 100% ພາຍໃຕ້ 1GB RAM!** |

---

## 4. ຄູ່ມືການຕິດຕັ້ງ Server ຕັ້ງແຕ່ເລີ່ມຕົ້ນ

### ຂັ້ນຕອນທີ 1: ສ້າງ SWAP File 2GB ເທິງ Server
Run ຄຳສັ່ງນີ້ຜ່ານ SSH ໃນ Server Droplet ທັນທີ:
```bash
# 1. ສ້າງໄຟລ໌ Swap ຂະໜາດ 2GB
sudo fallocate -l 2G /swapfile
sudo chmod 600 /swapfile
sudo mkswap /swapfile
sudo swapon /swapfile

# 2. ບັນທຶກໃຫ້ເປີດອັດຕະໂນມັດທຸກຄັ້ງທີ່ Reboot
echo '/swapfile none swap sw 0 0' | sudo tee -a /etc/fstab

# 3. ຕັ້ງຄ່າ Swappiness ເພື່ອບອກໃຫ້ Linux ໃຊ້ RAM ແທ້ກ່ອນ
sudo sysctl vm.swappiness=20
echo 'vm.swappiness=20' | sudo tee -a /etc/sysctl.conf
```

### ຂັ້ນຕອນທີ 2: ຕິດຕັ້ງ Docker & UFW Firewall
```bash
# ອັບເດດລະບົບ ແລະ ຕິດຕັ້ງ Docker
sudo apt update && sudo apt upgrade -y
curl -fsSL https://get.docker.com | sh
sudo usermod -aG docker $USER

# ຕັ້ງຄ່າ Firewall ເປີດສະເພາະ Port ທີ່ຈຳເປັນ
sudo ufw default deny incoming
sudo ufw default allow outgoing
sudo ufw allow 22/tcp   # SSH
sudo ufw allow 80/tcp   # HTTP
sudo ufw allow 443/tcp  # HTTPS
sudo ufw enable
```

### ຂັ້ນຕອນທີ 3: ສ້າງໂຄງສ້າງໂຟນເດີໂປຣເຈັກ
```bash
sudo mkdir -p /opt/meta-app/scripts
sudo chown -R $USER:$USER /opt/meta-app
cd /opt/meta-app
```

### ຂັ້ນຕອນທີ 4: ສ້າງໄຟລ໌ `docker-compose.yml`
ສ້າງໄຟລ໌ `/opt/meta-app/docker-compose.yml`:
```yaml
version: '3.8'

networks:
  meta-internal-net:
    driver: bridge

volumes:
  postgres_core_data:
  postgres_chat_data:
  redis_data:
  caddy_data:
  caddy_config:

services:
  # 1. Caddy Reverse Proxy
  caddy:
    image: caddy:2-alpine
    container_name: meta-proxy
    restart: always
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - ./Caddyfile:/etc/caddy/Caddyfile:ro
      - caddy_data:/data
      - caddy_config:/config
    networks:
      - meta-internal-net
    depends_on:
      - backend

  # 2. Go Backend API
  backend:
    image: ghcr.io/toninpaserd12345-ship-it/meta-super-app/backend:latest
    container_name: meta-backend
    restart: always
    environment:
      - PORT=8080
      - GOMEMLIMIT=200MiB
      - GOGC=80
      - DATABASE_URL=postgres://meta_user:CoreDbSecretPass987@postgres-core:5432/meta_core_db?sslmode=disable
      - CHAT_DATABASE_URL=postgres://meta_user:ChatDbSecretPass654@postgres-chat:5432/meta_chat_db?sslmode=disable
      - REDIS_URL=redis://redis:6379/0
      - META_APP_ID=${META_APP_ID}
      - META_APP_SECRET=${META_APP_SECRET}
      - META_REDIRECT_URI=https://api.yourdomain.com/api/v1/meta/oauth/callback
      - META_WHATSAPP_CONFIG_ID=${META_WHATSAPP_CONFIG_ID:-991544757321935}
      - R2_ACCESS_KEY=${R2_ACCESS_KEY}
      - R2_SECRET_KEY=${R2_SECRET_KEY}
      - R2_BUCKET=${R2_BUCKET}
    deploy:
      resources:
        limits:
          memory: 220M
    networks:
      - meta-internal-net
    depends_on:
      postgres-core:
        condition: service_healthy
      postgres-chat:
        condition: service_healthy
      redis:
        condition: service_healthy

  # 3. Redis In-Memory Engine
  redis:
    image: redis:7-alpine
    container_name: meta-redis
    restart: always
    command: redis-server --appendonly yes --maxmemory 128mb --maxmemory-policy allkeys-lru
    deploy:
      resources:
        limits:
          memory: 140M
    volumes:
      - redis_data:/data
    networks:
      - meta-internal-net
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 5s
      timeout: 3s
      retries: 5

  # 4. PostgreSQL Core Database
  postgres-core:
    image: postgres:16-alpine
    container_name: meta-postgres-core
    restart: always
    environment:
      POSTGRES_USER: meta_user
      POSTGRES_PASSWORD: CoreDbSecretPass987
      POSTGRES_DB: meta_core_db
    command: >
      postgres
      -c shared_buffers=128MB
      -c work_mem=4MB
      -c maintenance_work_mem=32MB
      -c max_connections=40
      -c synchronous_commit=on
    deploy:
      resources:
        limits:
          memory: 220M
    volumes:
      - postgres_core_data:/var/lib/postgresql/data
    networks:
      - meta-internal-net
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U meta_user -d meta_core_db"]
      interval: 5s
      timeout: 3s
      retries: 5

  # 5. PostgreSQL Chat Database
  postgres-chat:
    image: postgres:16-alpine
    container_name: meta-postgres-chat
    restart: always
    environment:
      POSTGRES_USER: meta_user
      POSTGRES_PASSWORD: ChatDbSecretPass654
      POSTGRES_DB: meta_chat_db
    command: >
      postgres
      -c shared_buffers=128MB
      -c work_mem=4MB
      -c maintenance_work_mem=32MB
      -c max_connections=60
      -c synchronous_commit=off
      -c wal_writer_delay=50ms
    deploy:
      resources:
        limits:
          memory: 220M
    volumes:
      - postgres_chat_data:/var/lib/postgresql/data
    networks:
      - meta-internal-net
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U meta_user -d meta_chat_db"]
      interval: 5s
      timeout: 3s
      retries: 5
```

### ຂັ້ນຕອນທີ 5: ສ້າງໄຟລ໌ `Caddyfile`
ສ້າງໄຟລ໌ `/opt/meta-app/Caddyfile`:
```caddy
api.yourdomain.com {
    encode zstd gzip

    reverse_proxy backend:8080 {
        header_up Host {host}
        header_up X-Real-IP {remote}
        header_up X-Forwarded-For {remote}
        header_up X-Forwarded-Proto {scheme}
    }
}
```

### ຂັ້ນຕອນທີ 6: ເປີດໃຊ້ງານລະບົບ
```bash
# ສ້າງໄຟລ໌ .env ສຳລັບເກັບ Secrets
cat <<EOF > /opt/meta-app/.env
META_APP_ID=your_facebook_app_id
META_APP_SECRET=your_facebook_app_secret
R2_ACCESS_KEY=your_cloudflare_r2_access_key
R2_SECRET_KEY=your_cloudflare_r2_secret_key
R2_BUCKET=meta-super-app-backups
EOF

# Start Containers ທັງໝົດ
docker compose up -d
```

---

## 5. ການຕັ້ງຄ່າ Meta Developer App & Webhooks

ເພື່ອໃຫ້ລະບົບເຊື່ອມຕໍ່ກັບ Facebook, WhatsApp, ແລະ Messenger ໄດ້ສົມບູນ:

1. ເຂົ້າໄປທີ່ **[developers.facebook.com](https://developers.facebook.com)** -> ເລືອກແອັບຂອງທ່ານ:
2. **Facebook Login Settings**:
   * Valid OAuth Redirect URIs: `https://api.yourdomain.com/api/v1/meta/oauth/callback`
3. **Webhooks Setup**:
   * Callback URL: `https://api.yourdomain.com/api/v1/meta/webhook`
   * Verify Token: ຄ່າທີ່ກຳນົດໃນ Backend (ຫຼື Default verify string)
   * Subscription Fields ທີ່ຕ້ອງຕິກ: `messages`, `messaging_postbacks`, `feed`, `leadgen`.
4. **App Permissions ທີ່ຕ້ອງຂໍ**:
   * `pages_show_list`, `pages_read_engagement`, `pages_manage_metadata`, `pages_manage_posts`, `pages_messaging`, `whatsapp_business_messaging`.

---

## 6. ການ Deploy Frontend ເທິງ Cloudflare Pages

1. **ເຂົ້າ Cloudflare Dashboard** -> ເມນູ **Workers & Pages** -> **Create Application** -> **Pages**.
2. **Connect to Git**: ເລືອກ Repository `META-SUPPER-APP`.
3. **Build Settings**:
   * Framework Preset: `Nuxt.js`
   * Root Directory: `frontend`
   * Build Command: `npm run build` ຫຼື `npx nuxi generate`
   * Output Directory: `frontend/.output/public` (ຫຼື `dist`)
4. **Environment Variables**:
   * `NUXT_PUBLIC_API_BASE`: `https://api.yourdomain.com`
5. **Custom Domain**:
   * ຜູກ Custom Domain ເປັນ `app.yourdomain.com`

---

## 7. ລະບົບ Backup ອັດຕະໂນມັດຂຶ້ນ Cloudflare R2

ສ້າງໄຟລ໌ Script `/opt/meta-app/scripts/backup.sh`:
```bash
#!/bin/sh
set -e

TIMESTAMP=$(date +%Y%m%d_%H%M%S)
BACKUP_DIR="/tmp/backups"
mkdir -p "$BACKUP_DIR"

echo "=== Starting Dual Database Backup at $(date) ==="

# 1. Backup Core DB (Zero-Lock Snapshot, Nice low priority)
nice -n 19 ionice -c 3 docker exec meta-postgres-core pg_dump -U meta_user -d meta_core_db -Fc | zstd -3 > "$BACKUP_DIR/core_$TIMESTAMP.sql.zst"

# 2. Backup Chat DB (Zero-Lock Snapshot)
nice -n 19 ionice -c 3 docker exec meta-postgres-chat pg_dump -U meta_user -d meta_chat_db -Fc | zstd -3 > "$BACKUP_DIR/chat_$TIMESTAMP.sql.zst"

# 3. Upload to Cloudflare R2 ຜ່ານ rclone
rclone copy "$BACKUP_DIR" "r2:${R2_BUCKET}/database_backups/$TIMESTAMP"

# 4. Clean local files
rm -rf "$BACKUP_DIR"

echo "=== Backup Completed successfully ==="
```
ຕັ້ງ Crontab ໃຫ້ແລ່ນທຸກມື້ຕອນ 03:00 ໂມງເຊົ້າ:
```bash
(crontab -l 2>/dev/null; echo "0 3 * * * /opt/meta-app/scripts/backup.sh >> /var/log/backup.log 2>&1") | crontab -
```

---

## 8. ຄູ່ມືການຂະຫຍາຍລະບົບ ແລະ ຍ້າຍ Database

ເມື່ອຮ້ານຄ້າ ແລະ ຂໍ້ຄວາມແຊັດເພີ່ມຂຶ້ນຈົນຮອດ 5,000+ ຮ້ານຄ້າ, ທ່ານສາມາດຍ້າຍ `postgres-chat` ໄປ Server ໃໝ່ (Server 2) ພາຍໃນ 5 ນາທີ:

### ຄຳສັ່ງຍ້າຍຂໍ້ມູນສົດຜ່ານ Network Pipe (Zero Disk Waste):
```bash
# Run ເທິງ Server 1: ດູດຂໍ້ມູນຈາກ Server 1 ສົ່ງຕົງເຂົ້າ Server 2 ແບບ Stream
docker exec meta-postgres-chat pg_dump -U meta_user -d meta_chat_db -Fc | \
ssh root@<SERVER_2_PRIVATE_IP> "docker exec -i meta-postgres-chat pg_restore -U meta_user -d meta_chat_db --clean"
```

### ສະຫຼັບ Backend ໄປຫາ Server 2:
ແກ້ໄຂ `docker-compose.yml` ຢູ່ **Server 1**:
```yaml
- CHAT_DATABASE_URL=postgres://meta_user:ChatDbSecretPass654@<SERVER_2_PRIVATE_IP>:5432/meta_chat_db?sslmode=disable
```
ສັ່ງ Restart Backend:
```bash
docker compose up -d backend
```
ຫຼັງຈາກນັ້ນປິດ ແລະ ລຶບ `postgres-chat` ອອກຈາກ Server 1 ເພື່ອຄືນ RAM ທັນທີ!

---

## 9. ຄູ່ມືການແກ້ໄຂບັນຫາ ແລະ ຄຳສັ່ງປະຈຳວັນ

### ຄຳສັ່ງກວດສອບສະຖານະລະບົບ:
```bash
# ເບິ່ງວ່າ Container ໃດແລ່ນຢູ່
docker ps

# ເບິ່ງການກິນ RAM / CPU ແບບ Realtime ຂອງແຕ່ລະ Container
docker stats

# ເບິ່ງ Log ຂອງ Backend ສົດໆ
docker logs -f meta-backend --tail 100

# ເບິ່ງ Log ຂອງຖານຂໍ້ມູນ
docker logs -f meta-postgres-core --tail 50
docker logs -f meta-postgres-chat --tail 50
```

### ວິທີແກ້ໄຂບັນຫາທີ່ພົບບ່ອນ (Troubleshooting):
* **ຖ້າ Server ຊ້າ ຫຼື RAM ເກືອບເຕັມ**:
  ກວດເບິ່ງດ້ວຍ `free -m`. ຖ້າ Swap ຖືກນຳໃຊ້ເກີນ 1GB, ໃຫ້ກວດສອບ `docker stats` ວ່າ Container ໃດກິນ RAM ເກີນ Limit.
* **ຖ້າ Facebook Webhook ບໍ່ຕອບສະໜອງ**:
  1. ກວດສອບ Caddy: `docker logs meta-proxy --tail 50` ວ່າ Meta ສົ່ງ HTTP 200 ມາຫຼືບໍ່.
  2. ກວດສອບ Redis: `docker exec -it meta-redis redis-cli ping` ຕ້ອງຕອບ `PONG`.

---

## 10. ແຜນທີ່ຄວາມຮູ້ທີ່ຕ້ອງສຶກສາຕໍ່ຍອດ

| ລຳດັບ | ຫົວຂໍ້ | ຈຸດປະສົງ | ແຫຼ່ງຮຽນຮູ້ທີ່ແນະນຳ |
| :--- | :--- | :--- | :--- |
| **1** | **Linux & Swap** | ຄຸ້ມຄອງ Server 1GB ບໍ່ໃຫ້ດັບ, ເບິ່ງ log | [linuxjourney.com](https://linuxjourney.com/) |
| **2** | **Docker & Compose** | Start, Stop, Restart, ແລະ ອັບເດດແອັບ | [roadmap.sh/docker](https://roadmap.sh/docker) |
| **3** | **Postgres & Backup** | Backup/Restore ຂໍ້ມູນ, ເຂົ້າໃຈ B-Tree Index | [postgresqltutorial.com](https://www.postgresqltutorial.com/) |
| **4** | **Meta Graph API** | ເຂົ້າໃຈ Webhook, Page Access Token | [developers.facebook.com/docs](https://developers.facebook.com/docs/) |
| **5** | **Cloudflare Pages/R2** | ເກັບໄຟລ໌ສິນຄ້າ, Deploy Edge Frontend | [developers.cloudflare.com](https://developers.cloudflare.com/) |

---

## 11. ວັດຈະນານຸກົມ ແລະ ຄຳອະທິບາຍທຸກໂຕອັກສອນ

ພາກນີ້ແມ່ນການອະທິບາຍ **ທຸກຕົວອັກສອນ, ທຸກຕົວຫຍໍ້, ແລະ ທຸກພາຣາມິເຕີ** ທີ່ປາກົດໃນຄູ່ມືນີ້ ເພື່ອໃຫ້ທ່ານເຂົ້າໃຈຄວາມໝາຍ ແລະ ເຫດຜົນທາງເທັກນິກຢ່າງເລິກເຊິ່ງ:

### 11.1 ຕົວຫຍໍ້ລະດັບ Server & ເຄືອຂ່າຍ
* **vCPU (Virtual Central Processing Unit)**: ແກນສະໝອງປະມວນຜົນຈຳລອງຂອງ Server (1 vCPU = 1 Thread ປະມວນຜົນ).
* **RAM (Random Access Memory)**: ໜ່ວຍຄວາມຈຳຫຼັກທີ່ເຮັດວຽກໄວທີ່ສຸດ ແຕ່ເປັນຂໍ້ມູນຊົ່ວຄາວ (ຖ້າໄຟດັບ ຫຼື Server ຣີບູດ ຂໍ້ມູນໃນ RAM ຈະຫາຍ).
* **SSD (Solid State Drive)**: ໜ່ວຍຄວາມຈຳຖາວອນຄວາມໄວສູງທີ່ໃຊ້ເກັບໄຟລ໌ລະບົບ ແລະ ຖານຂໍ້ມູນ (ຂໍ້ມູນບໍ່ມີວັນຫາຍ).
* **SWAP**: ການເອົາພື້ນທີ່ SSD ມາຈຳລອງເປັນ RAM ສຳຮອງ ເວລາ RAM ແທ້ເຕັມ ເພື່ອປ້ອງກັນ Server ດັບ.
* **OOM (Out Of Memory / OOM Killer)**: ກົນໄກປ້ອງກັນຕົວຂອງ Linux Kernel: ເວລາ RAM ເຕັມ 100%, ມັນຈະສັ່ງ "ຂ້າ" ໂປຣແກຣມທີ່ກິນ RAM ຫຼາຍທີ່ສຸດຖິ້ມທັນທີ ເພື່ອຮັກສາໃຫ້ Server ຍັງເປີດຢູ່ໄດ້.
* **SSL / TLS (Secure Sockets Layer / Transport Layer Security)**: ລະບົບເຂົ້າລະຫັດຂໍ້ມູນລະຫວ່າງ Browser ກັບ Server ເພື່ອປ້ອງກັນການດັກຈັບລະຫັດຜ່ານ (ສັນຍະລັກກຸນແຈ `https://`).
* **DNS (Domain Name System)**: ລະບົບປ່ຽນຊື່ເວັບໄຊ (ເຊັ່ນ `api.yourdomain.com`) ໃຫ້ກາຍເປັນເລກ IP ຂອງ Server.
* **WAF (Web Application Firewall)**: ກຳແພງໄຟກວດກາ ແລະ ປ້ອງກັນ Hacker, Bot ກໍ່ກວນ, ແລະ ການຍິງຖະຫຼົ່ມ (DDoS Attack).
* **TTFB (Time To First Byte)**: ໄລຍະເວລາຕັ້ງແຕ່ Browser ສົ່ງຄຳຂໍ ຈົນຮອດເວລາທີ່ Server ສົ່ງຂໍ້ມູນ byte ທຳອິດກັບມາ (ຍິ່ງໜ້ອຍຍິ່ງໂຫຼດໄວ).
* **Egress Bandwidth**: ປະລິມານຂໍ້ມູນທີ່ຖືກສົ່ງອອກຈາກ Cloud ໄປຫາຜູ້ໃຊ້ (Cloudflare R2 ບໍ່ຄິດຄ່ານີ້ = ຟຣີ $0).

---

### 11.2 ຄຳສັ່ງ Linux Setup (ແຍກແຕ່ລະຕົວອັກສອນ)
* **`sudo` (Superuser DO)**: ຂໍສິດທິຜູ້ດູແລລະບົບສູງສຸດ (Root) ເພື່ອແລ່ນຄຳສັ່ງລະດັບ System.
* **`fallocate -l 2G /swapfile`**:
  * `fallocate`: ຄຳສັ່ງຈອງພື້ນທີ່ Disk ລ່ວງໜ້າແບບທັນທີ.
  * `-l 2G`: **L**ength = ຂະໜາດ **2 Gigabytes**.
  * `/swapfile`: ຊື່ໄຟລ໌ Swap ທີ່ຕັ້ງຢູ່ Root directory.
* **`chmod 600 /swapfile`**:
  * `chmod`: **Ch**ange **Mod**e = ປ່ຽນສິດການເຂົ້າເຖິງໄຟລ໌.
  * `600`: ເລກ `6` (ອ່ານ/ຂຽນໄດ້ສະເພາະ Root ເທົ່ານັ້ນ), ເລກ `00` (ຄົນອື່ນທັງໝົດຫ້າມອ່ານ ແລະ ຫ້າມແຕະຕ້ອງ ເພື່ອຄວາມປອດໄພ).
* **`mkswap /swapfile`**: **M**a**k**e **Swap** = ຟໍແມັດໄຟລ໌ນີ້ໃຫ້ກາຍເປັນພື້ນທີ່ Swap ຂອງ Linux.
* **`swapon /swapfile`**: **Swap ON** = ເປີດໃຊ້ງານ Swap ນີ້ເຂົ້າສູ່ Memory pool ທັນທີ.
* **`/etc/fstab` (File Systems Table)**: ໄຟລ໌ຕາຕະລາງລະບົບທີ່ບອກ Linux ວ່າເວລາ Reboot ຂຶ້ນມາ ໃຫ້ mount partition ແລະ swap ໃດແດ່ອັດຕະໂນມັດ.
* **`sysctl vm.swappiness=20`**:
  * `sysctl`: ປັບແຕ່ງຄ່າ Kernel ຂອງລະບົບປະຕິບັດການ Linux.
  * `vm.swappiness`: ຄ່າຄວາມດຸໝັ່ນໃນການເອົາ Swap ມາໃຊ້ (ເລກ 0–100). ເຮົາຕັ້ງ `20` ເພື່ອບອກວ່າ **"ໃຫ້ໃຊ້ RAM ແທ້ຈົນຮອດ 80% ກ່ອນ ຈຶ່ງຄ່ອຍເອົາ Swap ມາຊ່ວຍ"** ເພື່ອໃຫ້ Server ແລ່ນໄວທີ່ສຸດ.
* **`ufw` (Uncomplicated Firewall)**:
  * `default deny incoming`: ບລັອກທຸກ Port ທີ່ເຊື່ອມຕໍ່ເຂົ້າມາ Server ໂດຍເລີ່ມຕົ້ນ.
  * `allow 22/tcp`: ເປີດ Port 22 ສຳລັບ **SSH**.
  * `allow 80/tcp`: ເປີດ Port 80 ສຳລັບ **HTTP**.
  * `allow 443/tcp`: ເປີດ Port 443 ສຳລັບ **HTTPS** (SSL).

---

### 11.3 ພາຣາມິເຕີໃນ Docker Compose (`docker-compose.yml`)
* **`version: '3.8'`**: ມາດຕະຖານໂຄງສ້າງໄຟລ໌ Docker Compose.
* **`services:`**: ລາຍຊື່ Container ທັງໝົດທີ່ຕ້ອງການເປີດໃຊ້.
* **`image: postgres:16-alpine`**:
  * `postgres:16`: ເວີຊັນ PostgreSQL 16.
  * `-alpine`: ເປັນ Linux ລຸ້ນພິເສດທີ່ມີຂະໜາດນ້ອຍຈິ໋ວ (ພຽງ ~15MB ແທນທີ່ຈະເປັນ 500MB) **ປະຢັດ RAM ແລະ SSD ທີ່ສຸດ**.
* **`container_name:`**: ຊື່ສະເພາະຂອງ Container ເພື່ອໃຫ້ສັ່ງການ ຫຼື ເບິ່ງ log ໄດ້ງ່າຍ.
* **`restart: always`**: ຖ້າ Container ດັບ ຫຼື Server ຖືກ Reboot, **ໃຫ້ Docker ເປີດມັນຄືນມາໃໝ່ອັດຕະໂນມັດ 24/7**.
* **`ports: - "80:80"`**:
  * ເລກໜ້າ `80`: Port ເທິງ Server ຕົວຈິງ (Host).
  * ເລກຫຼັງ `80`: Port ພາຍໃນ Container.
* **`environment:`**: ຕົວປ່ຽນສະພາບແວດລ້ອມ (Environment Variables) ທີ່ສົ່ງເຂົ້າໄປໃຫ້ App.
* **`volumes: postgres_core_data:/var/lib/postgresql/data`**:
  * ເອົາໂຟນເດີຖານຂໍ້ມູນຂ້າງໃນ Container ມາຜູກຕິດກັບ Host SSD **ເພື່ອວ່າເວລາອັບເດດ ຫຼື ລຶບ Container, ຂໍ້ມູນຈະບໍ່ມີວັນສູນຫາຍ**.
* **`networks: meta-internal-net`**: ສ້າງວົງ Lan ພາຍໃນ Docker ເພື່ອໃຫ້ Backend ລົມກັບ Postgres ໄດ້ **ໂດຍຄົນພາຍນອກບໍ່ສາມາດແຮັກເຂົ້າມາຫາ Database ໄດ້**.
* **`depends_on: condition: service_healthy`**: ສັ່ງໃຫ້ Backend ລໍຖ້າໃຫ້ Database ກວດສອບຕົວເອງຈົນພ້ອມເຮັດວຽກ 100% ກ່ອນ ຈຶ່ງຄ່ອຍ Start Backend ຂຶ້ນມາ (ປ້ອງກັນ Backend crash).
* **`deploy: resources: limits: memory: 220M`**: ບອກ Docker ວ່າ **ຫ້າມ Container ນີ້ໃຊ້ RAM ເກີນ 220MB ເດັດຂາດ** ເພື່ອປ້ອງກັນບໍ່ໃຫ້ແຍ່ງ RAM ຂອງ Container ອື່ນ.

---

### 11.4 ຄ່າປັບແຕ່ງ PostgreSQL & Backend Engine
* **`shared_buffers=128MB`**: ຈຳນວນ RAM ທີ່ມອບໃຫ້ PostgreSQL ໃຊ້ເປັນ Cache ເກັບຂໍ້ມູນຕາຕະລາງທີ່ຖືກເອີ້ນໃຊ້ເລື້ອຍໆ.
* **`work_mem=4MB`**: RAM ທີ່ມອບໃຫ້ແຕ່ລະ Query ໃຊ້ເວລາຄົ້ນຫາ, ຈັດລຽງ (SORT), ຫຼື JOIN ຕາຕະລາງ.
* **`maintenance_work_mem=32MB`**: RAM ສຳລັບວຽກດູແລລະບົບ ເຊັ່ນ: ການສ້າງ Index, ການເຮັດ VACUUM.
* **`max_connections=40/60`**: ຈຳນວນ Client ທີ່ອະນຸຍາດໃຫ້ເຊື່ອມຕໍ່ຖານຂໍ້ມູນພ້ອມກັນສູງສຸດ.
* **`synchronous_commit=on` (Core DB)**: ໂໝດປອດໄພສູງສຸດ: ທຸກໆອໍເດີ ແລະ ເງິນ ຕ້ອງຂຽນລົງ SSD ສຳເລັດແທ້ໆກ່ອນ ຈຶ່ງຈະຕອບຕົກລົງ (ປ້ອງກັນຂໍ້ມູນເສຍ).
* **`synchronous_commit=off` (Chat DB)**: **(ໄຮໄລ້ສຳລັບ Chat!)**: ບໍ່ຕ້ອງລໍຖ້າ SSD ຂຽນແລ້ວໆ, ໃຫ້ຂຽນລົງ RAM Cache ກ່ອນ ແລ້ວຕອບທັນທີ ເຮັດໃຫ້ **ຮັບຂໍ້ຄວາມແຊັດໄດ້ໄວຂຶ້ນ 10 ເທົ່າ!**
* **`wal_writer_delay=50ms`**: ສັ່ງໃຫ້ຖານຂໍ້ມູນດຶງຂໍ້ຄວາມແຊັດຈາກ RAM ໄປຂຽນລົງ SSD ທຸກໆ 50 ມິລລິວິນາທີ.
* **`WAL` (Write-Ahead Logging)**: ບັນທຶກປະຫວັດການຂຽນລ່ວງໜ້າ ເພື່ອໃຊ້ກູ້ຄືນຂໍ້ມູນເວລາໄຟດັບ.
* **`GOMEMLIMIT=200MiB`**: ບອກ Go Runtime ວ່າ: **"ເຈົ້າໃຊ້ RAM ໄດ້ບໍ່ເກີນ 200MB ເດີ!"** ຖ້າໃກ້ຈະເກີນ ໃຫ້ບັງຄັບເກັບຂີ້ເຫຍື້ອ (Garbage Collector) ທັນທີ ເພື່ອບໍ່ໃຫ້ Linux ຂ້າ.
* **`GOGC=80`**: ຄວາມໄວໃນການເຄຼຍ RAM ຂີ້ເຫຍື້ອ (ຕັ້ງ 80 ເພື່ອໃຫ້ເຄຼຍ RAM ໄວຂຶ້ນກວ່າປົກກະຕິ).
* **`sslmode=disable`**: ປິດ SSL ລະຫວ່າງ Backend ກັບ Database ຍ້ອນທັງສອງຢູ່ໃນວົງ Lan ດຽວກັນພາຍໃນ Server (ຊ່ວຍປະຢັດ CPU).

---

### 11.5 ຄ່າປັບແຕ່ງ Redis & Caddy Reverse Proxy
* **`appendonly yes`**: ບັນທຶກທຸກຄຳສັ່ງຂອງ Redis ລົງ Disk ປ້ອງກັນຂໍ້ມູນ Cache ຫາຍ.
* **`maxmemory 128mb`**: ຈຳກັດ RAM ຂອງ Redis ໄວ້ບໍ່ເກີນ 128 MB.
* **`maxmemory-policy allkeys-lru` (Least Recently Used)**: ຖ້າ RAM 128MB ເຕັມ, ໃຫ້ Redis **ລຶບ Key ທີ່ເກົ່າທີ່ສຸດ ແລະ ບໍ່ຄ່ອຍມີຄົນໃຊ້ອອກອັດຕະໂນມັດ** ເພື່ອໃຫ້ມີບ່ອນວ່າງສຳລັບ Key ໃໝ່.
* **`reverse_proxy backend:8080`**: ຮັບ Traffic ຈາກພາຍນອກ ແລ້ວສົ່ງຕໍ່ໄປຫາ Go Backend ທີ່ Port 8080.
* **`encode zstd gzip`**: ບີບອັດຂໍ້ມູນ JSON ດ້ວຍ Zstandard ແລະ Gzip ກ່ອນສົ່ງຫາ Browser (ເຮັດໃຫ້ເວັບໂຫຼດໄວຂຶ້ນ 3 ເທົ່າ ແລະ ປະຢັດ Bandwidth).
* **`header_up Host {host}`**: ສົ່ງຊື່ Domain ຕົ້ນທາງໄປໃຫ້ Backend ຮູ້.
* **`header_up X-Real-IP {remote}`**: **(ສຳຄັນຫຼາຍ)** ສົ່ງເລກ IP ແທ້ຂອງລູກຄ້າໄປໃຫ້ Backend.

---

### 11.6 ຕົວອັກສອນກ່ຽວກັບ Meta (Facebook) & Cloudflare
* **`META_APP_ID`**: ລະຫັດປະຈຳຕົວຂອງ Facebook Developer Application.
* **`META_APP_SECRET`**: ລະຫັດລັບສູງສຸດຂອງ Facebook App.
* **`OAuth 2.0`**: ມາດຕະຖານການ Login ດ້ວຍ Facebook ໂດຍທີ່ລູກຄ້າບໍ່ຕ້ອງບອກ Password ໃຫ້ເຮົາ.
* **`Redirect URI`**: ລິ້ງທີ່ Facebook ຈະສົ່ງຄົນກັບຄືນມາຫຼັງຈາກກົດອະນຸຍາດ Login ສຳເລັດ.
* **`Webhook`**: ລະບົບທີ່ Facebook ຍິງແຈ້ງເຕືອນເຂົ້າມາຫາ Server ເຮົາທັນທີ ເວລາມີຄົນທັກແຊັດ ຫຼື ຄອມເມັ້ນ.
* **`MID` (Message ID)**: ລະຫັດປະຈຳຂໍ້ຄວາມແຕ່ລະອັນຂອງ Facebook (ເຮົາເອົາໃສ່ Redis ເພື່ອປ້ອງກັນບໍ່ໃຫ້ Bot ຕອບລູກຄ້າຊ້ຳ 2 ເທື່ອ).
* **`CAPI` (Conversions API)**: ລະບົບຍິງຂໍ້ມູນການຂາຍຈາກ Server ຂອງເຮົາກົງໄປຫາ Facebook Ads ເພື່ອຫາລູກຄ້າທີ່ຕົງກຸ່ມເປົ້າໝາຍ.
* **`R2`**: ລະບົບເກັບໄຟລ໌ Cloud Storage ຂອງ Cloudflare (ຄືກັບ AWS S3 ແຕ່ບໍ່ຄິດຄ່າດາວໂຫຼດ Bandwidth).

---

### 11.7 ຄຳສັ່ງ Backup & Streaming Pipe
```bash
nice -n 19 ionice -c 3 docker exec meta-postgres-chat pg_dump -U meta_user -d meta_chat_db -Fc | \
ssh root@<SERVER_2_PRIVATE_IP> "docker exec -i meta-postgres-chat pg_restore -U meta_user -d meta_chat_db --clean"
```
* **`nice -n 19`**:
  * `nice`: ຄຳສັ່ງກຳນົດລະດັບຄວາມສຳຄັນຂອງ CPU.
  * `-n 19`: ເລກ `19` ແມ່ນ **ລະດັບຄວາມສຳຄັນຕ່ຳສຸດ (Low Priority)** ໝາຍຄວາມວ່າ: ຖ້າມີຄົນກຳລັງຊື້ເຄື່ອງ ຫຼື ແຊັດຢູ່, ໃຫ້ CPU ໄປບໍລິການລູກຄ້າກ່ອນ! ຖ້າ CPU ຫວ່າງ ຈຶ່ງຄ່ອຍມາ Backup.
* **`ionice -c 3`**:
  * `ionice`: ກຳນົດລະດັບຄວາມສຳຄັນຂອງການອ່ານ/ຂຽນ Hard Disk (I/O).
  * `-c 3`: ໂໝດ **Idle** = ໃຫ້ Hard Disk ຫວ່າງແທ້ໆ ຈຶ່ງຄ່ອຍອ່ານຂໍ້ມູນ Backup (ປ້ອງກັນບໍ່ໃຫ້ Disk ຄ້າງ).
* **`pg_dump -Fc`**:
  * `-F`: Format.
  * `c`: **C**ustom Archive Format = ບີບອັດເປັນ Binary ພິເສດຂອງ Postgres (ຂະໜາດນ້ອຍ ແລະ Restore ໄວທີ່ສຸດ).
* **`|` (Pipe Operator)**: **ທໍ່ສົ່ງຕໍ່** = ຈັບຂໍ້ມູນຈາກຝັ່ງຊ້າຍ ຍັດສົ່ງຕໍ່ໃຫ້ຝັ່ງຂວາທັນທີເທິງ RAM ໂດຍບໍ່ຕ້ອງເສຍເວລາ Save ລົງ Disk.
* **`pg_restore --clean`**:
  * `--clean`: ສັ່ງໃຫ້ລຶບ (Drop) ຕາຕະລາງເກົ່າຖິ້ມກ່ອນ ແລ້ວສ້າງໃໝ່ ເພື່ອບໍ່ໃຫ້ເກີດ Error ຂໍ້ມູນຊ້ຳຊ້ອນ.

---
*ເອກະສານນີ້ຖືກສ້າງຂຶ້ນເພື່ອເປັນຄູ່ມືມາດຕະຖານການປະຕິບັດງານ (Standard Operating Procedure) ຂອງ META-SUPER-APP.*
