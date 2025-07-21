# Fleet Management System - Test Content Guide

## Manager Dashboard Page (`/manager-dashboard`)

### Expected Content to Test For:

1. **Page Title/Header**: The page doesn't have a visible h1, it uses the navigation component with a page description

2. **Section Headings (h2)**:
   - "Fleet Overview"
   - "Quick Actions"
   - "Recent Activity"
   - "Maintenance Alerts" (conditional - only if there are alerts)

3. **Metric Labels** (in Fleet Overview section):
   - "Total Buses"
   - "Active Drivers"
   - "Total Students"
   - "Active Routes"

4. **Quick Action Buttons**:
   - "Manage Fleet"
   - "Assign Routes"
   - "Manage Students"
   - "ECSE Students"
   - "Import ECSE"
   - "Import Wizard"
   - "View Reports"
   - "Approve Users"

## Students Page (`/students`)

### Expected Content to Test For:

1. **Page Title (h1)**:
   - "Student Management"

2. **Section Headings (h2)**:
   - "Student Statistics" (screen-reader only - has class "sr-only")
   - "All Students"

3. **Statistics Labels**:
   - "Total Students"
   - "Active Students"
   - "Morning Pickups"
   - "Afternoon Dropoffs"

4. **Buttons**:
   - "Add Student" (in header)
   - "Edit" buttons for each student
   - "Remove" buttons for each student

5. **Student Card Content** (for each student):
   - Student name
   - "Active" or "Inactive" status
   - "Pickup:" and "Dropoff:" times
   - "Guardian:" label
   - Phone numbers
   - Location badges

## Test Code Examples

Instead of looking for "Statistics" or generic terms, your E2E tests should look for these specific strings:

```javascript
// For Manager Dashboard
await expect(page.locator('text=Fleet Overview')).toBeVisible();
await expect(page.locator('text=Quick Actions')).toBeVisible();
await expect(page.locator('text=Total Buses')).toBeVisible();
await expect(page.locator('text=Manage Fleet')).toBeVisible();

// For Students Page
await expect(page.locator('h1:has-text("Student Management")')).toBeVisible();
await expect(page.locator('text=All Students')).toBeVisible();
await expect(page.locator('text=Total Students')).toBeVisible();
await expect(page.locator('text=Add Student')).toBeVisible();
```

## Important Notes:

1. The manager dashboard doesn't have a visible h1 heading - it uses a navigation component
2. The "Student Statistics" heading on the students page has `sr-only` class (screen reader only)
3. Some content is conditional (like Maintenance Alerts) and may not always appear
4. The actual metric values (numbers) are dynamic and should not be tested for specific values
5. Both pages use icon fonts (Bootstrap Icons) which appear as `<i>` tags with classes like `bi-bus-front`