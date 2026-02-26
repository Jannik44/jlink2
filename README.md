# jlink2 🔗

A blazing-fast URL shortener with a **unique extra Feature**: custom link previews for WhatsApp, Discord, Telegram, Signal, and every other messenger! 🎨

Instead of just showing a raw URL, share a beautiful preview with your custom title and description in a snap. ✨ —no matter what's actually on the target page.

![Version](https://img.shields.io/badge/version-2.1-blue)
![Go](https://img.shields.io/badge/go-1.21+-00ADD8?logo=go&logoColor=white)
![Fiber](https://img.shields.io/badge/Fiber-v2-00ADD8?logo=go&logoColor=white)
![Bootstrap](https://img.shields.io/badge/Bootstrap-5-purple?logo=bootstrap&logoColor=white)
![SQLite](https://img.shields.io/badge/SQLite-3-blue?logo=sqlite&logoColor=white)

## 🌟 Custom Messenger Previews

When someone shares your jlink2 link in WhatsApp, Discord, Telegram, Signal, Twitter, or any other platform:

```
Instead of showing this:
┌─────────────────────────────────────┐
│ https://example.com/very/long/url   │
│ No preview available                │
└─────────────────────────────────────┘

```

<img width="252" height="97" alt="Screenshot_20260226-114416~2" src="https://github.com/user-attachments/assets/1945c2ea-a06b-4047-9ed7-939ea43fe63d" />

```
They see this:

┌─────────────────────────────────────┐
│ Check out this awesome thing! 🎁    │
│ Limited time offer - 50% off today  │
│ http://myurl.eu/mylink              │
└─────────────────────────────────────┘
```

<img width="201" height="91" alt="Screenshot_20260226-114757~2" src="https://github.com/user-attachments/assets/10a73a70-5568-486b-9945-469def99bf67" />

---

## ✨ Features

- 🎨 **Custom messenger previews** - Control exactly what title and description appear when links are shared on WhatsApp, Discord, Telegram, Signal, Twitter, etc. (works on any platform that reads HTML meta tags)
- 🚀 **Lightning-fast** - Built with [Fiber](https://gofiber.io/), one of the fastest Go web frameworks
- 🎯 **Custom or auto-generated slugs** - Create memorable short URLs or let the system create unique ones
- ⏰ **Link expiration** - Set expiration dates for temporary or campaign links
- 🛡️ **Security**:
  - Domain blacklist to prevent malicious URL redirects
  - Self-reference protection to avoid infinite redirect loops
  - Rate limiting (configurable requests per minute and links per day per IP)
  - XSS protection with full HTML escaping
  - Real client IP detection behind proxies
- 📊 **Minimal footprint** - ~11 MB memory usage, single Go binary, SQLite database
- 🐳 **Docker-first** - Pre-built Docker image, production-ready
- 💾 **SQLite database** - Lightweight, file-based, no external database needed
- 🌐 **REST API** - Simple JSON API for creating and managing links

---

## 🚀 Quick Start

### Prerequisites
- install Docker

### Installation

#### Using Docker: (Recommended) 🐳

```bash
docker run \
  -d \
  -p 80:3000 \
  -v jlink2-data:/app/data \ 
  --name jlink2 \
  jannik44/jlink2
```
note: you can also use -v /path/to/your/data/jlink2:/app/data if you want to access the data easily

## 📖 API-Usage

#### Note: In most cases you just want to use the webui (no coding skill needed, for end users)

### Create a Shortened Link with Customizable Social Media Preview

**Request:**
```bash
curl -X POST http://localhost:3000/create \
  -H "Content-Type: application/json" \
  -d '{
    "url": "https://example.com/very/long/url/path",
    "slug": "mylink",
    "title": "My Cool Link",
    "desc": "A description that shows in preview",
    "exp": "2026-12-31"
  }'
```
Note: Only set the fields you need, leave all others empty
**Response:**
```json
{
  "finalurl": "http://mydomain.eu/mylink"
}
```

### Access a Shortened Link

Simply visit or redirect to your short URL:
```
http://mydomain.eu/mylink
↓
https://example.com/very/long/url/path
```

---

## ⚙️ Configuration

Configuration is managed via `data/config.yaml`. On first run, a default config is created automatically.

### Configuration File Example

```yaml
# Minimum slug length (auto-extended if all combinations are taken)
defaultgeneratedsluglength: 2

# Domain for generating short URLs
domain: example.com

# Use HTTPS in generated URLs
uses-https: true

# Prevent self-referential links (avoid infinite redirect loops)
allow-self-reference: false

# Characters allowed in auto-generated slugs
slug-allowed-characters: "abcdefghijklmnopqrstuvwxyz123456789"

# Max links per IP per 24 hours
links-per-day: 50

# Max requests per IP per minute (rate limiting)
requests-per-minute: 200
```

### All Configuration Parameters

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `defaultgeneratedsluglength` | int | 2 | Minimum length for auto-generated slugs |
| `domain` | string | localhost | Domain used in generated short URLs |
| `uses-https` | bool | false | Whether to use HTTPS in generated URLs |
| `allow-self-reference` | bool | false | Allow links pointing to the same domain |
| `slug-allowed-characters` | string | abcdefghijklmnopqrstuvwxyz123456789 | Allowed characters for slug generation |
| `links-per-day` | int | 50 | Maximum links creation per IP per 24 hours |
| `requests-per-minute` | int | 200 | Maximum requests per IP per minute |

---

## 🛡️ Domain Blacklist

Edit `data/domain-blacklist.txt` to block certain domains from being used as redirect targets:

```bash
# Blacklist of domains that cannot be used as redirect targets
# One domain per line, case-insensitive
example.com
badsite.org
spam.net
```

Comments (lines starting with `#`) are ignored. Domains are case-insensitive.

---

## 📊 Database Schema

SQLite database is automatically initialized with this schema:

```sql
CREATE TABLE links (
  id          INTEGER PRIMARY KEY AUTOINCREMENT,
  url         TEXT NOT NULL,
  slug        TEXT NOT NULL UNIQUE,
  exp         TEXT,
  title       TEXT,
  desc        TEXT,
  created_ip  TEXT,
  created     TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);
```

---

## 🔍 Logging

All logs are written to both stdout and `data/logs/` directory as `YYYY-MM-DD_HH-MM-SS.log` files for easy debugging and monitoring.

---

## 🔐 Security Considerations

✅ **Client IP logging** - All link creations are logged with creator IP  
✅ **Rate limiting** - Prevent abuse with per-IP request throttling  
✅ **Blacklist support** - Block dangerous or spam domains  
✅ **XSS protection** - HTML entities escaped to prevent injection  
✅ **Self-reference protection** - Prevent redirect loop attacks

---

## 🤝 Contributing

Contributions are welcome! Please feel free to submit issues and pull requests to help improve jlink2.

---

## 📄 License & Credits

This project is provided as-is for educational, personal and commercial use. Contact me if you plan using this commercially as there might be special needs.

Made with ❤️ by [j44](https://github.com/jannik44)

---

Have questions? Found a bug? [Open an issue!](https://github.com/jannik44/jlink2/issues)
