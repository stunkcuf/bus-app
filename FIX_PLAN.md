# Fleet Management System - Fix Plan

## Current Status
- 88.9% functionality working
- 1 critical error: User Management page (500 error)
- Driver features untested
- Some role-based access issues

## Priority 1: Fix User Management Page (CRITICAL)

### Issue
- Page returns 500 error: "Failed to get users"
- Cache might be nil or failing
- Already added fallback but still failing

### Fix Steps
1. **Debug the actual error**
   ```go
   // Add more detailed logging to see exact failure point
   // Check if loadUsersFromDB() is actually defined
   // Verify database connection when loading users
   ```

2. **Create dedicated fix**
   - Remove cache dependency entirely for this page
   - Direct database query only
   - Add proper error context

3. **Test immediately after fix**

## Priority 2: Complete Driver Testing

### Current Blockers
- Don't know driver passwords (bjmathis, MariaA1)
- Manager can't access driver pages (by design)

### Fix Steps
1. **Reset a driver password**
   ```go
   // Create password reset utility for bjmathis
   // Set to known password like "driver123"
   ```

2. **Test driver features**
   - Login as driver
   - Check maintenance alerts display
   - Verify student management
   - Test daily log entry

3. **Document any issues found**

## Priority 3: Fix Role-Based Access

### Issues
- Students page requires driver role (confusing)
- Managers can't see their drivers' students

### Fix Options
1. **Option A: Allow managers to access student pages**
   - Change requireRole("driver") to allow both roles
   - More logical for oversight

2. **Option B: Create manager-specific student view**
   - New route: /manager/students
   - Shows all students across all routes

## Priority 4: Data Integrity Check

### Verify
1. Fleet vehicles count (should be 54)
2. Route assignments are correct
3. All maintenance records accessible
4. ECSE data properly linked

### Fix Steps
1. Run comprehensive data validation
2. Create data cleanup script if needed
3. Add data integrity constraints

## Implementation Order

### Phase 1 - Immediate Fixes (Today)
1. [ ] Fix User Management page error
2. [ ] Create driver password reset tool
3. [ ] Test with driver account
4. [ ] Document findings

### Phase 2 - Access Control (Tomorrow)
1. [ ] Review all route permissions
2. [ ] Implement manager access to student data
3. [ ] Test all role combinations
4. [ ] Update documentation

### Phase 3 - Final Polish (Day 3)
1. [ ] Run full system test
2. [ ] Fix any remaining issues
3. [ ] Create user manual
4. [ ] Prepare for deployment

## Quick Fix for User Management

```go
// In handlers_missing.go, replace the entire manageUsersHandler with:
func manageUsersHandler(w http.ResponseWriter, r *http.Request) {
    user := getUserFromSession(r)
    if user == nil || user.Role != "manager" {
        http.Redirect(w, r, "/", http.StatusFound)
        return
    }

    // Direct database query - skip cache entirely
    var users []User
    query := `SELECT username, role, status, registration_date, created_at 
              FROM users ORDER BY created_at DESC`
    
    err := db.Select(&users, query)
    if err != nil {
        log.Printf("Error loading users from database: %v", err)
        // Don't fail - show empty list
        users = []User{}
    }

    data := map[string]interface{}{
        "User":      user,
        "Users":     users,
        "CSRFToken": getSessionCSRFToken(r),
    }

    renderTemplate(w, r, "users.html", data)
}
```

## Success Criteria
- [ ] All pages load without errors
- [ ] Both manager and driver roles tested
- [ ] 100% core functionality working
- [ ] Clear documentation provided
- [ ] System ready for production

## Estimated Time
- Phase 1: 1-2 hours
- Phase 2: 2-3 hours  
- Phase 3: 1-2 hours
- Total: 4-7 hours to complete

This plan will get us from 88.9% to 100% functionality!