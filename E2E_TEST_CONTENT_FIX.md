# E2E Test Content Fix Guide

## Problem Summary
The E2E tests are failing because they're looking for content that doesn't exist on the pages. The tests are looking for generic terms like "Statistics" when the actual pages use more specific terms.

## Manager Dashboard (`/manager-dashboard`)

### What the test is looking for (WRONG):
- "Statistics" - This text doesn't exist on the page

### What the test SHOULD look for (CORRECT):
```javascript
// Section headings
await expect(page.locator('text=Fleet Overview')).toBeVisible();
await expect(page.locator('text=Quick Actions')).toBeVisible();
await expect(page.locator('text=Recent Activity')).toBeVisible();

// Metric labels in Fleet Overview section
await expect(page.locator('text=Total Buses')).toBeVisible();
await expect(page.locator('text=Active Drivers')).toBeVisible();
await expect(page.locator('text=Total Students')).toBeVisible();
await expect(page.locator('text=Active Routes')).toBeVisible();

// Quick action buttons
await expect(page.locator('text=Manage Fleet')).toBeVisible();
await expect(page.locator('text=Assign Routes')).toBeVisible();
await expect(page.locator('text=Manage Students')).toBeVisible();
```

## Students Page (`/students`)

### What the test SHOULD look for:
```javascript
// Page heading
await expect(page.locator('h1:has-text("Student Management")')).toBeVisible();

// Section heading
await expect(page.locator('h2:has-text("All Students")')).toBeVisible();

// Statistics labels
await expect(page.locator('.stat-label:has-text("Total Students")')).toBeVisible();
await expect(page.locator('.stat-label:has-text("Active Students")')).toBeVisible();
await expect(page.locator('.stat-label:has-text("Morning Pickups")')).toBeVisible();
await expect(page.locator('.stat-label:has-text("Afternoon Dropoffs")')).toBeVisible();

// Action button
await expect(page.locator('a.btn-primary:has-text("Add Student")')).toBeVisible();
```

## Key Differences to Note:

1. **Manager Dashboard** has NO visible h1 heading - it uses a navigation component with page description
2. **Manager Dashboard** does NOT have a section called "Statistics" - it has "Fleet Overview"
3. **Students Page** has a hidden h2 "Student Statistics" with class `sr-only` (screen reader only)
4. **Students Page** statistics are shown with class `stat-label`, not generic text

## Complete Test Example:

```javascript
test('Manager can view dashboard content', async ({ page }) => {
  // Login first
  await page.goto('/');
  await page.fill('input[name="username"]', 'admin');
  await page.fill('input[name="password"]', 'SecureAdminPass123!');
  await page.click('button[type="submit"]');
  
  // Wait for navigation
  await page.waitForURL('**/manager-dashboard');
  
  // Check dashboard content
  await expect(page.locator('h2:has-text("Fleet Overview")')).toBeVisible();
  await expect(page.locator('h2:has-text("Quick Actions")')).toBeVisible();
  
  // Check metrics exist (don't check specific values)
  await expect(page.locator('.metric-label:has-text("Total Buses")')).toBeVisible();
  await expect(page.locator('.metric-label:has-text("Active Drivers")')).toBeVisible();
});

test('Manager can access student management', async ({ page }) => {
  // Assume already logged in
  await page.goto('/students');
  
  // Check page loaded
  await expect(page.locator('h1:has-text("Student Management")')).toBeVisible();
  
  // Check statistics section exists
  await expect(page.locator('.stat-label:has-text("Total Students")')).toBeVisible();
  
  // Check main section
  await expect(page.locator('h2:has-text("All Students")')).toBeVisible();
  
  // Check add button
  await expect(page.locator('a:has-text("Add Student")')).toBeVisible();
});
```

## Files to Update:
1. Any E2E test files that check for dashboard content
2. Any E2E test files that check for student page content
3. Update selectors to match actual HTML structure
4. Remove checks for non-existent content like generic "Statistics" text