# Production Deployment Checklist

## 🚀 Pre-Deployment Checklist

### 1. Code Preparation
- [ ] All tests passing
- [ ] No `log.Fatal()` calls in production code
- [ ] Debug logging removed or set to appropriate level
- [ ] All TODO comments addressed
- [ ] Sensitive data removed from code

### 2. Security Checklist
- [ ] Environment variables for all secrets
- [ ] HTTPS enabled (TLS certificates)
- [ ] CSRF protection enabled
- [ ] Rate limiting configured
- [ ] Input validation on all forms
- [ ] SQL injection vulnerabilities fixed
- [ ] Password complexity requirements enforced
- [ ] Session timeout configured (24 hours max)

### 3. Database Preparation
- [ ] Production database backed up
- [ ] All migrations applied
- [ ] Indices created for performance
- [ ] Connection pool configured
- [ ] Prepared statements enabled

### 4. Environment Configuration
Create `.env.production`:
```env
# Database
DATABASE_URL=postgresql://user:password@host:5432/dbname?sslmode=require
DB_MAX_CONNECTIONS=25
DB_MAX_IDLE=5
DB_MAX_LIFETIME=300

# Server
PORT=5003
APP_ENV=production
SESSION_SECRET=<generate-secure-64-char-secret>

# Security
CSRF_SECRET=<generate-secure-32-char-secret>
ALLOWED_ORIGINS=https://yourdomain.com
SECURE_COOKIES=true

# Rate Limiting
RATE_LIMIT_REQUESTS=100
RATE_LIMIT_WINDOW=60

# Monitoring
SENTRY_DSN=<your-sentry-dsn>
LOG_LEVEL=info
LOG_FILE=/var/log/fleet/app.log

# Email (for notifications)
SMTP_HOST=smtp.sendgrid.net
SMTP_PORT=587
SMTP_USER=apikey
SMTP_PASS=<your-api-key>
ALERT_EMAIL=admin@yourdomain.com
```

### 5. Server Setup
- [ ] Go 1.19+ installed
- [ ] PostgreSQL 13+ running
- [ ] Redis installed (for future session store)
- [ ] Nginx configured as reverse proxy
- [ ] SSL certificates installed
- [ ] Firewall configured
- [ ] Backup system configured

## 📝 Nginx Configuration

Create `/etc/nginx/sites-available/fleet`:
```nginx
server {
    listen 80;
    server_name yourdomain.com;
    return 301 https://$server_name$request_uri;
}

server {
    listen 443 ssl http2;
    server_name yourdomain.com;

    ssl_certificate /etc/letsencrypt/live/yourdomain.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/yourdomain.com/privkey.pem;

    # Security headers
    add_header X-Frame-Options "SAMEORIGIN" always;
    add_header X-Content-Type-Options "nosniff" always;
    add_header X-XSS-Protection "1; mode=block" always;
    add_header Referrer-Policy "no-referrer-when-downgrade" always;
    add_header Content-Security-Policy "default-src 'self' https:; script-src 'self' 'unsafe-inline' 'unsafe-eval' https://cdn.jsdelivr.net; style-src 'self' 'unsafe-inline' https://cdn.jsdelivr.net; img-src 'self' data: https:;" always;

    # Rate limiting
    limit_req_zone $binary_remote_addr zone=general:10m rate=10r/s;
    limit_req_zone $binary_remote_addr zone=login:10m rate=5r/m;

    location / {
        limit_req zone=general burst=20 nodelay;
        
        proxy_pass http://localhost:5003;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        
        # WebSocket support (if needed)
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
    }

    location /login {
        limit_req zone=login burst=5 nodelay;
        proxy_pass http://localhost:5003;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }

    # Static files
    location /static/ {
        alias /var/www/fleet/static/;
        expires 30d;
        add_header Cache-Control "public, immutable";
    }
}
```

## 🚀 Deployment Steps

### 1. Build the Application
```bash
# On your local machine
GOOS=linux GOARCH=amd64 go build -o fleet-management *.go

# Or use Makefile
make build-production
```

### 2. Deploy to Server
```bash
# Copy files to server
scp fleet-management user@server:/var/www/fleet/
scp -r templates static user@server:/var/www/fleet/
scp .env.production user@server:/var/www/fleet/.env

# On the server
cd /var/www/fleet
chmod +x fleet-management
```

### 3. Setup systemd Service
Create `/etc/systemd/system/fleet.service`:
```ini
[Unit]
Description=Fleet Management System
After=postgresql.service
Wants=postgresql.service

[Service]
Type=simple
User=fleet
Group=fleet
WorkingDirectory=/var/www/fleet
ExecStart=/var/www/fleet/fleet-management
Restart=always
RestartSec=5
Environment="APP_ENV=production"

# Security
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/var/www/fleet/sessions.json
ReadWritePaths=/var/log/fleet

# Logging
StandardOutput=append:/var/log/fleet/output.log
StandardError=append:/var/log/fleet/error.log

[Install]
WantedBy=multi-user.target
```

### 4. Start the Service
```bash
# Reload systemd
sudo systemctl daemon-reload

# Enable auto-start
sudo systemctl enable fleet

# Start the service
sudo systemctl start fleet

# Check status
sudo systemctl status fleet
```

## 📊 Post-Deployment Verification

### 1. Health Checks
```bash
# Check application health
curl https://yourdomain.com/health

# Check API health
curl https://yourdomain.com/api/health
```

### 2. Test Critical Functions
- [ ] Login works (both manager and driver)
- [ ] Data displays correctly
- [ ] Forms submit without errors
- [ ] File uploads work
- [ ] Reports generate correctly

### 3. Monitor Logs
```bash
# Application logs
tail -f /var/log/fleet/app.log

# System logs
journalctl -u fleet -f

# Nginx logs
tail -f /var/log/nginx/access.log
tail -f /var/log/nginx/error.log
```

## 🔍 Monitoring Setup

### 1. Uptime Monitoring
- Setup Uptime Robot or similar for endpoint monitoring
- Monitor: `/health`, `/api/health`
- Alert on: Response time > 2s or status != 200

### 2. Application Monitoring
- Configure Sentry for error tracking
- Setup Prometheus + Grafana for metrics
- Configure alerts for:
  - High error rate
  - Slow queries
  - Memory usage > 80%
  - CPU usage > 80%

### 3. Database Monitoring
- Monitor connection count
- Track slow queries
- Alert on replication lag
- Monitor disk space

## 🔐 Security Hardening

### 1. Firewall Rules
```bash
# Allow only necessary ports
sudo ufw allow 22/tcp  # SSH
sudo ufw allow 80/tcp  # HTTP
sudo ufw allow 443/tcp # HTTPS
sudo ufw enable
```

### 2. Fail2ban Configuration
Create `/etc/fail2ban/jail.local`:
```ini
[fleet-login]
enabled = true
port = http,https
filter = fleet-login
logpath = /var/log/fleet/app.log
maxretry = 5
bantime = 3600
```

### 3. Regular Security Updates
```bash
# Create update script
#!/bin/bash
apt update
apt upgrade -y
systemctl restart fleet
```

## 📋 Maintenance Tasks

### Daily
- [ ] Check application logs for errors
- [ ] Monitor disk space
- [ ] Verify backups completed

### Weekly
- [ ] Review slow query logs
- [ ] Check for security updates
- [ ] Test backup restoration
- [ ] Review user activity logs

### Monthly
- [ ] Update dependencies
- [ ] Review and rotate logs
- [ ] Performance analysis
- [ ] Security audit

## 🚨 Rollback Plan

### If deployment fails:
1. Stop the new service: `sudo systemctl stop fleet`
2. Restore previous binary: `cp fleet-management.backup fleet-management`
3. Restore previous config: `cp .env.backup .env`
4. Start service: `sudo systemctl start fleet`
5. Verify functionality

### Database rollback:
1. Stop application
2. Restore from backup: `pg_restore -d fleet backup.dump`
3. Restart application

## 📞 Emergency Contacts

- **System Admin**: [Your contact]
- **Database Admin**: [DB contact]
- **On-call Developer**: [Dev contact]
- **Hosting Support**: [Railway/AWS/etc support]

## ✅ Final Checklist

Before going live:
- [ ] All environment variables set
- [ ] SSL certificate valid
- [ ] Monitoring configured
- [ ] Backups tested
- [ ] Load testing completed
- [ ] Security scan passed
- [ ] Documentation updated
- [ ] Team trained
- [ ] Support plan in place

🎉 Once all checks pass, your Fleet Management System is ready for production!