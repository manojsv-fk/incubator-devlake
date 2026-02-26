# Local Setup — FourKites Engineering Signals

## Prerequisites
- [Rancher Desktop](https://rancherdesktop.io/) (or Docker Desktop) with at least 30 GB of container storage

## First-time Setup

**1. Create the `.env` file**
```bash
cp env.example .env
```
Set `ENCRYPTION_SECRET` in `.env` to a random hex string — run `openssl rand -hex 16` to generate one.

**2. Fix Docker disk space (Rancher Desktop only)**

Rancher Desktop uses a small in-memory disk by default. Run once to allocate 30 GB from your Mac:
```bash
rdctl shell sudo sh -c "
  dd if=/dev/zero bs=1 count=0 seek=30G of=/Users/$USER/.docker-storage.img
  mkfs.ext4 /Users/$USER/.docker-storage.img
  mount -o loop /Users/$USER/.docker-storage.img /var/lib/docker
  rc-service docker restart
"
```
> After every Rancher Desktop restart, re-run just the last two lines (mount + restart).

**3. Start the stack**
```bash
docker compose -f docker-compose-dev.yml up -d mysql grafana devlake config-ui
```

## Services

| Service | URL | Credentials |
|---|---|---|
| DevLake Config UI | http://localhost:4000 | — |
| Grafana | http://localhost:3002 | admin / admin |
| DevLake API | http://localhost:8080 | — |

## Adding a GitHub Repo

1. Go to http://localhost:4000 → **Projects → New Project**
2. Add a GitHub connection with a PAT (scopes: `repo`, `read:org`)
3. Select your repo as the data scope
4. Click **Collect Data**

## Stop / Start
```bash
docker compose -f docker-compose-dev.yml down
docker compose -f docker-compose-dev.yml up -d mysql grafana devlake config-ui
```
