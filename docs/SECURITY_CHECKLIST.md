# Security Checklist for Fleet Management System

## Before Pushing to GitHub

### 🔒 Critical Security Items

- [ ] **NO hardcoded passwords** in any Go files
- [ ] **NO .sql files** with production data
- [ ] **NO .env files** with real credentials  
- [ ] **NO admin creation scripts** with passwords
- [ ] **NO Excel/CSV files** with real student data
- [ ] **NO compiled executables** (.exe files)

### ✅ Safe to Commit

- [x] All Go source code (without hardcoded credentials)
- [x] HTML templates
- [x] Static files (CSS, JS)
- [x] Markdown documentation
- [x] .gitignore file
- [x] .env.example (template only)
- [x] go.mod and go.sum
- [x] Dockerfile (if no secrets)

### 🔍 Security Features Implemented

1. **Password Security**
   - Bcrypt hashing (cost factor 12)
   - No plain text storage
   - Password strength requirements

2. **Session Management**
   - Secure session cookies
   - HTTPOnly, Secure, SameSite flags
   - 24-hour expiration
   - CSRF token protection

3. **Input Validation**
   - HTML tag stripping
   - SQL injection prevention (parameterized queries)
   - File upload restrictions
   - Rate limiting on login

4. **Security Headers**
   - Content Security Policy (CSP)
   - X-Frame-Options: DENY
   - X-Content-Type-Options: nosniff
   - Strict-Transport-Security (HSTS)
   - Referrer-Policy: strict-origin-when-cross-origin

### 🚀 Deployment Security (Railway)

1. **Environment Variables to Set:**
   ```
   DATABASE_URL     (provided by Railway)
   ADMIN_USERNAME   (default: admin)
   ADMIN_PASSWORD   (MUST set this!)
   SESSION_SECRET   (random string)
   APP_ENV         (production)
   ```

2. **First Run:**
   - Admin user created automatically if ADMIN_PASSWORD is set
   - Use utilities/reset_password.go locally to change passwords

3. **Regular Maintenance:**
   - Review user accounts monthly
   - Check for unusual login patterns
   - Update dependencies regularly
   - Monitor error logs

### ⚠️ Common Mistakes to Avoid

1. **DON'T** commit after running admin creation scripts
2. **DON'T** test with real student data locally
3. **DON'T** share database URLs in issues/commits
4. **DON'T** disable CSRF protection for convenience
5. **DON'T** log sensitive data (passwords, sessions)

### 📝 Git Commands for Safety

```bash
# Check what will be committed
git status
git diff --staged

# Remove sensitive file from staging
git reset HEAD <sensitive-file>

# Remove file from Git history (if accidentally committed)
git filter-branch --tree-filter 'rm -f passwords.sql' HEAD

# Check for secrets in commit history
git log -p | grep -i password
```

### 🔐 Quick Security Audit

```bash
# Find potential secrets in codebase
grep -r "password.*=" --include="*.go" .
grep -r "secret" --include="*.go" .
find . -name "*.sql" -o -name "*.env"

# Check Git for sensitive files
git ls-files | grep -E "password|secret|admin|.env|.sql"
```

Remember: **When in doubt, don't commit it!**