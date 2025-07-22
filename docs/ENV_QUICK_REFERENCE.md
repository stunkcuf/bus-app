# Environment Variables Quick Reference

## Essential Variables (Production)

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `DATABASE_URL` | ✅ Yes | none | PostgreSQL connection string |
| `APP_ENV` | ⚠️ Recommended | `development` | `production` or `development` |
| `ADMIN_PASSWORD` | ✅ Yes | none | Admin user password |
| `SESSION_SECRET` | ✅ Yes | none | Session encryption key |
| `PORT` | No | `5000` | HTTP server port |

## Quick Setup Commands

### Development
```bash
# Copy template
cp .env.example .env

# Edit with your values
nano .env

# Generate secure session secret
openssl rand -hex 32

# Start application
make dev
```

### Production
```bash
export APP_ENV=production
export DATABASE_URL="postgresql://user:pass@host:port/db?sslmode=require"
export ADMIN_PASSWORD="SecurePassword123!"
export SESSION_SECRET="$(openssl rand -hex 32)"
```

## Common Issues

| Issue | Check | Solution |
|-------|-------|----------|
| Database connection fails | `DATABASE_URL` | Verify connection string |
| Admin login not working | `ADMIN_PASSWORD` | Set strong password |
| Sessions not persisting | `SESSION_SECRET` | Set 32+ character secret |
| Wrong port | `PORT` | Check port availability |

## Security Checklist

- [ ] `ADMIN_PASSWORD` set and strong (12+ chars)
- [ ] `SESSION_SECRET` set and random (32+ chars) 
- [ ] `APP_ENV=production` for production
- [ ] Database uses SSL (`sslmode=require`)
- [ ] `.env` files not committed to git

## All Variables

| Variable | Type | Default | Required | Description |
|----------|------|---------|----------|-------------|
| `APP_ENV` | string | `development` | No | Application environment |
| `PORT` | int | `5000` | No | HTTP server port |
| `LOG_LEVEL` | string | `INFO` | No | Logging verbosity |
| `DATABASE_URL` | string | none | Yes | PostgreSQL connection |
| `PGHOST` | string | none | No* | DB hostname |
| `PGPORT` | int | `5432` | No* | DB port |
| `PGUSER` | string | none | No* | DB username |
| `PGPASSWORD` | string | none | No* | DB password |
| `PGDATABASE` | string | none | No* | DB name |
| `ADMIN_USERNAME` | string | `admin` | No | Admin username |
| `ADMIN_PASSWORD` | string | none | Yes** | Admin password |
| `SESSION_SECRET` | string | none | Yes** | Session key |
| `SESSION_STORE_FILE` | string | `sessions.json` | No | Session storage file |
| `DISABLE_COMPRESSION` | bool | `false` | No | Disable HTTP compression |
| `NODE_ENV` | string | `development` | No | Frontend build mode |

*Required if `DATABASE_URL` not set  
**Required in production