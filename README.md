# jlink2 🚀
Overview 🌟

jlink2 is a super-lightweight URL shortener written in Go. It creates short links and lets you customize the preview title and description in a snap. ✨
## Features 🎉
- Single Go binary, zero external dependencies 💪
- Auto-generated slugs with adjustable minimum length 🔢
- Custom link preview (title & description) for sharing in style 🖼️
- expiry settings 📅
- Domain and HTTPS settings for perfectly formatted URLs 🌐
- Optional blacklist and self-reference protection 🚫
- Real client IP detection behind your reverse proxy 🕵️‍♂️

## Installation 🛠️

```
docker run \
-d \ # detached mode, runs in background
-p80:3000 \ # public port 80, remove if the container should only be accessible from your proxy/ docker network
-v /path/to/your/data/jlink2:/app/data \ # persistent storage for your data, can also be a volume
--name jlink2 \ # the name of the docker container
jannik44/jlink2 # the image to use
```
