package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"
)

// Budget models
type Budget struct {
	ID              int       `json:"id" db:"id"`
	Name            string    `json:"name" db:"name"`
	FiscalYear      int       `json:"fiscal_year" db:"fiscal_year"`
	TotalAmount     float64   `json:"total_amount" db:"total_amount"`
	AllocatedAmount float64   `json:"allocated_amount" db:"allocated_amount"`
	SpentAmount     float64   `json:"spent_amount" db:"spent_amount"`
	Status          string    `json:"status" db:"status"` // draft, active, closed
	CreatedBy       string    `json:"created_by" db:"created_by"`
	CreatedAt       time.Time `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time `json:"updated_at" db:"updated_at"`
}

type BudgetCategory struct {
	ID              int     `json:"id" db:"id"`
	BudgetID        int     `json:"budget_id" db:"budget_id"`
	CategoryName    string  `json:"category_name" db:"category_name"`
	CategoryType    string  `json:"category_type" db:"category_type"` // fuel, maintenance, insurance, salaries, etc.
	AllocatedAmount float64 `json:"allocated_amount" db:"allocated_amount"`
	SpentAmount     float64 `json:"spent_amount" db:"spent_amount"`
	Description     string  `json:"description" db:"description"`
}

type BudgetTransaction struct {
	ID              int       `json:"id" db:"id"`
	BudgetID        int       `json:"budget_id" db:"budget_id"`
	CategoryID      int       `json:"category_id" db:"category_id"`
	TransactionDate time.Time `json:"transaction_date" db:"transaction_date"`
	Amount          float64   `json:"amount" db:"amount"`
	TransactionType string    `json:"transaction_type" db:"transaction_type"` // expense, adjustment
	Description     string    `json:"description" db:"description"`
	VehicleID       *string   `json:"vehicle_id" db:"vehicle_id"`
	ReferenceID     *string   `json:"reference_id" db:"reference_id"` // Links to fuel_records, maintenance_records, etc.
	ReferenceType   *string   `json:"reference_type" db:"reference_type"`
	CreatedBy       string    `json:"created_by" db:"created_by"`
	CreatedAt       time.Time `json:"created_at" db:"created_at"`
}

type BudgetAlert struct {
	ID          int       `json:"id" db:"id"`
	BudgetID    int       `json:"budget_id" db:"budget_id"`
	CategoryID  *int      `json:"category_id" db:"category_id"`
	AlertType   string    `json:"alert_type" db:"alert_type"` // threshold, overspend, projection
	ThresholdPct float64  `json:"threshold_pct" db:"threshold_pct"`
	Message     string    `json:"message" db:"message"`
	IsActive    bool      `json:"is_active" db:"is_active"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
}

// Budget summary for dashboard
type BudgetSummary struct {
	Budget          Budget                     `json:"budget"`
	Categories      []BudgetCategorySummary    `json:"categories"`
	RecentExpenses  []BudgetTransaction        `json:"recent_expenses"`
	Alerts          []BudgetAlert              `json:"alerts"`
	MonthlyTrend    []MonthlyBudgetTrend       `json:"monthly_trend"`
}

type BudgetCategorySummary struct {
	BudgetCategory
	PercentUsed    float64 `json:"percent_used"`
	Remaining      float64 `json:"remaining"`
	ProjectedTotal float64 `json:"projected_total"`
}

type MonthlyBudgetTrend struct {
	Month       string  `json:"month"`
	Budgeted    float64 `json:"budgeted"`
	Spent       float64 `json:"spent"`
	Categories  map[string]float64 `json:"categories"`
}

// Budget handlers

// budgetDashboardHandler shows the budget overview
func budgetDashboardHandler(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil || user.Role != "manager" {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	// Get current fiscal year budget
	currentYear := time.Now().Year()
	if time.Now().Month() < 7 { // Fiscal year starts in July
		currentYear--
	}

	budget, err := getCurrentBudget(currentYear)
	if err != nil && err != sql.ErrNoRows {
		log.Printf("Error loading budget: %v", err)
		http.Error(w, "Failed to load budget", http.StatusInternalServerError)
		return
	}

	var summary *BudgetSummary
	if budget != nil {
		summary, err = getBudgetSummary(budget.ID)
		if err != nil {
			log.Printf("Error loading budget summary: %v", err)
		}
	}

	data := map[string]interface{}{
		"Title":         "Budget Management",
		"User":          user,
		"Budget":        budget,
		"Summary":       summary,
		"CurrentYear":   currentYear,
		"CSRFToken":     getSessionCSRFToken(r),
		"CSPNonce":      r.Context().Value("cspNonce"),
	}

	renderTemplate(w, r, "budget_dashboard.html", data)
}

// budgetCreateHandler creates a new budget
func budgetCreateHandler(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil || user.Role != "manager" {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	if r.Method == "GET" {
		data := map[string]interface{}{
			"Title":     "Create Budget",
			"User":      user,
			"CSRFToken": getSessionCSRFToken(r),
			"CSPNonce":  r.Context().Value("cspNonce"),
		}
		renderTemplate(w, r, "budget_create.html", data)
		return
	}

	if r.Method == "POST" {
		if !validateCSRF(r) {
			http.Error(w, "Invalid CSRF token", http.StatusForbidden)
			return
		}

		// Parse form
		fiscalYear, _ := strconv.Atoi(r.FormValue("fiscal_year"))
		totalAmount, _ := strconv.ParseFloat(r.FormValue("total_amount"), 64)

		// Create budget
		budgetID, err := createBudget(Budget{
			Name:        fmt.Sprintf("FY %d Budget", fiscalYear),
			FiscalYear:  fiscalYear,
			TotalAmount: totalAmount,
			Status:      "draft",
			CreatedBy:   user.Username,
		})

		if err != nil {
			log.Printf("Error creating budget: %v", err)
			http.Error(w, "Failed to create budget", http.StatusInternalServerError)
			return
		}

		// Create default categories
		defaultCategories := []struct {
			name     string
			catType  string
			percent  float64
		}{
			{"Fuel", "fuel", 0.30},
			{"Maintenance & Repairs", "maintenance", 0.25},
			{"Insurance", "insurance", 0.15},
			{"Driver Salaries", "salaries", 0.20},
			{"Parts & Supplies", "supplies", 0.05},
			{"Other Expenses", "other", 0.05},
		}

		for _, cat := range defaultCategories {
			err = createBudgetCategory(BudgetCategory{
				BudgetID:        budgetID,
				CategoryName:    cat.name,
				CategoryType:    cat.catType,
				AllocatedAmount: totalAmount * cat.percent,
			})
			if err != nil {
				log.Printf("Error creating category %s: %v", cat.name, err)
			}
		}

		http.Redirect(w, r, fmt.Sprintf("/budget/edit/%d", budgetID), http.StatusFound)
	}
}

// budgetEditHandler edits budget allocations
func budgetEditHandler(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil || user.Role != "manager" {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	// Extract budget ID from URL
	budgetID, err := strconv.Atoi(r.URL.Path[len("/budget/edit/"):])
	if err != nil {
		http.Error(w, "Invalid budget ID", http.StatusBadRequest)
		return
	}

	budget, err := getBudgetByID(budgetID)
	if err != nil {
		http.Error(w, "Budget not found", http.StatusNotFound)
		return
	}

	if r.Method == "GET" {
		categories, err := getBudgetCategories(budgetID)
		if err != nil {
			log.Printf("Error loading categories: %v", err)
		}

		data := map[string]interface{}{
			"Title":      "Edit Budget",
			"User":       user,
			"Budget":     budget,
			"Categories": categories,
			"CSRFToken":  getSessionCSRFToken(r),
			"CSPNonce":   r.Context().Value("cspNonce"),
		}
		renderTemplate(w, r, "budget_edit.html", data)
		return
	}

	if r.Method == "POST" {
		if !validateCSRF(r) {
			http.Error(w, "Invalid CSRF token", http.StatusForbidden)
			return
		}

		// Update budget status if requested
		if status := r.FormValue("status"); status != "" {
			err = updateBudgetStatus(budgetID, status)
			if err != nil {
				log.Printf("Error updating budget status: %v", err)
			}
		}

		// Update category allocations
		categories, _ := getBudgetCategories(budgetID)
		totalAllocated := 0.0

		for _, cat := range categories {
			allocStr := r.FormValue(fmt.Sprintf("category_%d", cat.ID))
			if allocStr != "" {
				alloc, err := strconv.ParseFloat(allocStr, 64)
				if err == nil {
					err = updateCategoryAllocation(cat.ID, alloc)
					if err != nil {
						log.Printf("Error updating category %d: %v", cat.ID, err)
					}
					totalAllocated += alloc
				}
			}
		}

		// Update total allocated
		err = updateBudgetAllocated(budgetID, totalAllocated)
		if err != nil {
			log.Printf("Error updating budget allocated: %v", err)
		}

		http.Redirect(w, r, "/budget", http.StatusFound)
	}
}

// budgetExpenseHandler records an expense
func budgetExpenseHandler(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil {
		SendError(w, ErrUnauthorized("Authentication required"))
		return
	}

	if r.Method != "POST" {
		SendError(w, ErrMethodNotAllowed("Only POST method allowed"))
		return
	}

	var expense struct {
		CategoryID      int     `json:"category_id"`
		Amount          float64 `json:"amount"`
		Description     string  `json:"description"`
		VehicleID       string  `json:"vehicle_id"`
		TransactionDate string  `json:"transaction_date"`
	}

	if err := json.NewDecoder(r.Body).Decode(&expense); err != nil {
		SendError(w, ErrBadRequest("Invalid request format"))
		return
	}

	// Get category and budget info
	category, err := getBudgetCategoryByID(expense.CategoryID)
	if err != nil {
		SendError(w, ErrNotFound("Category not found"))
		return
	}

	// Parse date
	transDate, err := time.Parse("2006-01-02", expense.TransactionDate)
	if err != nil {
		transDate = time.Now()
	}

	// Create transaction
	transaction := BudgetTransaction{
		BudgetID:        category.BudgetID,
		CategoryID:      expense.CategoryID,
		TransactionDate: transDate,
		Amount:          expense.Amount,
		TransactionType: "expense",
		Description:     expense.Description,
		VehicleID:       &expense.VehicleID,
		CreatedBy:       user.Username,
	}

	err = recordBudgetTransaction(transaction)
	if err != nil {
		SendError(w, ErrDatabase("Failed to record expense", err))
		return
	}

	// Check for budget alerts
	checkBudgetAlerts(category.BudgetID, expense.CategoryID)

	SendJSON(w, http.StatusOK, map[string]string{
		"message": "Expense recorded successfully",
	})
}

// budgetReportHandler generates budget reports
func budgetReportHandler(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil || user.Role != "manager" {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	budgetID, _ := strconv.Atoi(r.URL.Query().Get("budget_id"))
	reportType := r.URL.Query().Get("type")
	if reportType == "" {
		reportType = "summary"
	}

	var data interface{}
	var err error

	switch reportType {
	case "summary":
		data, err = generateBudgetSummaryReport(budgetID)
	case "variance":
		data, err = generateBudgetVarianceReport(budgetID)
	case "projection":
		data, err = generateBudgetProjectionReport(budgetID)
	case "category":
		categoryID, _ := strconv.Atoi(r.URL.Query().Get("category_id"))
		data, err = generateCategoryDetailReport(budgetID, categoryID)
	}

	if err != nil {
		log.Printf("Error generating report: %v", err)
		http.Error(w, "Failed to generate report", http.StatusInternalServerError)
		return
	}

	// Return JSON for API calls
	if r.Header.Get("Accept") == "application/json" {
		SendJSON(w, http.StatusOK, data)
		return
	}

	// Render report template
	templateData := map[string]interface{}{
		"Title":      fmt.Sprintf("Budget Report - %s", reportType),
		"User":       user,
		"ReportType": reportType,
		"ReportData": data,
		"CSPNonce":   r.Context().Value("cspNonce"),
	}

	renderTemplate(w, r, "budget_report.html", templateData)
}

// Database operations

func getCurrentBudget(fiscalYear int) (*Budget, error) {
	var budget Budget
	err := db.QueryRow(`
		SELECT id, name, fiscal_year, total_amount, allocated_amount, spent_amount, 
			   status, created_by, created_at, updated_at
		FROM budgets
		WHERE fiscal_year = $1 AND status != 'closed'
		ORDER BY created_at DESC
		LIMIT 1
	`, fiscalYear).Scan(&budget.ID, &budget.Name, &budget.FiscalYear, 
		&budget.TotalAmount, &budget.AllocatedAmount, &budget.SpentAmount,
		&budget.Status, &budget.CreatedBy, &budget.CreatedAt, &budget.UpdatedAt)
	
	if err != nil {
		return nil, err
	}
	return &budget, nil
}

func getBudgetByID(id int) (*Budget, error) {
	var budget Budget
	err := db.QueryRow(`
		SELECT id, name, fiscal_year, total_amount, allocated_amount, spent_amount, 
			   status, created_by, created_at, updated_at
		FROM budgets
		WHERE id = $1
	`, id).Scan(&budget.ID, &budget.Name, &budget.FiscalYear, 
		&budget.TotalAmount, &budget.AllocatedAmount, &budget.SpentAmount,
		&budget.Status, &budget.CreatedBy, &budget.CreatedAt, &budget.UpdatedAt)
	
	if err != nil {
		return nil, err
	}
	return &budget, nil
}

func createBudget(budget Budget) (int, error) {
	var id int
	err := db.QueryRow(`
		INSERT INTO budgets (name, fiscal_year, total_amount, status, created_by)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`, budget.Name, budget.FiscalYear, budget.TotalAmount, 
		budget.Status, budget.CreatedBy).Scan(&id)
	
	return id, err
}

func updateBudgetStatus(id int, status string) error {
	_, err := db.Exec(`
		UPDATE budgets 
		SET status = $1, updated_at = CURRENT_TIMESTAMP
		WHERE id = $2
	`, status, id)
	return err
}

func updateBudgetAllocated(id int, allocated float64) error {
	_, err := db.Exec(`
		UPDATE budgets 
		SET allocated_amount = $1, updated_at = CURRENT_TIMESTAMP
		WHERE id = $2
	`, allocated, id)
	return err
}

func getBudgetCategories(budgetID int) ([]BudgetCategory, error) {
	var categories []BudgetCategory
	rows, err := db.Query(`
		SELECT id, budget_id, category_name, category_type, 
			   allocated_amount, spent_amount, description
		FROM budget_categories
		WHERE budget_id = $1
		ORDER BY category_name
	`, budgetID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var cat BudgetCategory
		err := rows.Scan(&cat.ID, &cat.BudgetID, &cat.CategoryName, 
			&cat.CategoryType, &cat.AllocatedAmount, &cat.SpentAmount, 
			&cat.Description)
		if err != nil {
			continue
		}
		categories = append(categories, cat)
	}

	return categories, nil
}

func getBudgetCategoryByID(id int) (*BudgetCategory, error) {
	var cat BudgetCategory
	err := db.QueryRow(`
		SELECT id, budget_id, category_name, category_type, 
			   allocated_amount, spent_amount, description
		FROM budget_categories
		WHERE id = $1
	`, id).Scan(&cat.ID, &cat.BudgetID, &cat.CategoryName, 
		&cat.CategoryType, &cat.AllocatedAmount, &cat.SpentAmount, 
		&cat.Description)
	
	if err != nil {
		return nil, err
	}
	return &cat, nil
}

func createBudgetCategory(category BudgetCategory) error {
	_, err := db.Exec(`
		INSERT INTO budget_categories 
		(budget_id, category_name, category_type, allocated_amount, description)
		VALUES ($1, $2, $3, $4, $5)
	`, category.BudgetID, category.CategoryName, category.CategoryType,
		category.AllocatedAmount, category.Description)
	
	return err
}

func updateCategoryAllocation(id int, amount float64) error {
	_, err := db.Exec(`
		UPDATE budget_categories 
		SET allocated_amount = $1
		WHERE id = $2
	`, amount, id)
	return err
}

func recordBudgetTransaction(transaction BudgetTransaction) error {
	// Start transaction
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Insert transaction
	_, err = tx.Exec(`
		INSERT INTO budget_transactions 
		(budget_id, category_id, transaction_date, amount, transaction_type,
		 description, vehicle_id, reference_id, reference_type, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`, transaction.BudgetID, transaction.CategoryID, transaction.TransactionDate,
		transaction.Amount, transaction.TransactionType, transaction.Description,
		transaction.VehicleID, transaction.ReferenceID, transaction.ReferenceType,
		transaction.CreatedBy)
	
	if err != nil {
		return err
	}

	// Update category spent amount
	_, err = tx.Exec(`
		UPDATE budget_categories 
		SET spent_amount = spent_amount + $1
		WHERE id = $2
	`, transaction.Amount, transaction.CategoryID)
	
	if err != nil {
		return err
	}

	// Update budget spent amount
	_, err = tx.Exec(`
		UPDATE budgets 
		SET spent_amount = spent_amount + $1, updated_at = CURRENT_TIMESTAMP
		WHERE id = $2
	`, transaction.Amount, transaction.BudgetID)
	
	if err != nil {
		return err
	}

	return tx.Commit()
}

func getBudgetSummary(budgetID int) (*BudgetSummary, error) {
	budget, err := getBudgetByID(budgetID)
	if err != nil {
		return nil, err
	}

	categories, err := getBudgetCategories(budgetID)
	if err != nil {
		return nil, err
	}

	// Calculate category summaries
	var categorySummaries []BudgetCategorySummary
	for _, cat := range categories {
		summary := BudgetCategorySummary{
			BudgetCategory: cat,
			Remaining:      cat.AllocatedAmount - cat.SpentAmount,
		}
		
		if cat.AllocatedAmount > 0 {
			summary.PercentUsed = (cat.SpentAmount / cat.AllocatedAmount) * 100
		}
		
		// Project based on current spending rate
		monthsElapsed := float64(time.Now().Month() - 6) // Fiscal year starts in July
		if monthsElapsed < 1 {
			monthsElapsed = 1
		}
		monthlyRate := cat.SpentAmount / monthsElapsed
		summary.ProjectedTotal = monthlyRate * 12
		
		categorySummaries = append(categorySummaries, summary)
	}

	// Get recent expenses
	recentExpenses, err := getRecentBudgetTransactions(budgetID, 10)
	if err != nil {
		log.Printf("Error loading recent expenses: %v", err)
	}

	// Get active alerts
	alerts, err := getActiveBudgetAlerts(budgetID)
	if err != nil {
		log.Printf("Error loading alerts: %v", err)
	}

	// Get monthly trend
	monthlyTrend, err := getMonthlyBudgetTrend(budgetID)
	if err != nil {
		log.Printf("Error loading monthly trend: %v", err)
	}

	return &BudgetSummary{
		Budget:         *budget,
		Categories:     categorySummaries,
		RecentExpenses: recentExpenses,
		Alerts:         alerts,
		MonthlyTrend:   monthlyTrend,
	}, nil
}

func getRecentBudgetTransactions(budgetID int, limit int) ([]BudgetTransaction, error) {
	var transactions []BudgetTransaction
	
	rows, err := db.Query(`
		SELECT id, budget_id, category_id, transaction_date, amount,
			   transaction_type, description, vehicle_id, reference_id,
			   reference_type, created_by, created_at
		FROM budget_transactions
		WHERE budget_id = $1
		ORDER BY transaction_date DESC, created_at DESC
		LIMIT $2
	`, budgetID, limit)
	
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var t BudgetTransaction
		err := rows.Scan(&t.ID, &t.BudgetID, &t.CategoryID, &t.TransactionDate,
			&t.Amount, &t.TransactionType, &t.Description, &t.VehicleID,
			&t.ReferenceID, &t.ReferenceType, &t.CreatedBy, &t.CreatedAt)
		if err != nil {
			continue
		}
		transactions = append(transactions, t)
	}

	return transactions, nil
}

func getActiveBudgetAlerts(budgetID int) ([]BudgetAlert, error) {
	var alerts []BudgetAlert
	
	rows, err := db.Query(`
		SELECT id, budget_id, category_id, alert_type, threshold_pct,
			   message, is_active, created_at
		FROM budget_alerts
		WHERE budget_id = $1 AND is_active = true
		ORDER BY created_at DESC
	`, budgetID)
	
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var a BudgetAlert
		err := rows.Scan(&a.ID, &a.BudgetID, &a.CategoryID, &a.AlertType,
			&a.ThresholdPct, &a.Message, &a.IsActive, &a.CreatedAt)
		if err != nil {
			continue
		}
		alerts = append(alerts, a)
	}

	return alerts, nil
}

func getMonthlyBudgetTrend(budgetID int) ([]MonthlyBudgetTrend, error) {
	rows, err := db.Query(`
		SELECT 
			TO_CHAR(transaction_date, 'YYYY-MM') as month,
			SUM(amount) as total,
			c.category_name,
			SUM(CASE WHEN t.category_id = c.id THEN t.amount ELSE 0 END) as category_total
		FROM budget_transactions t
		JOIN budget_categories c ON c.budget_id = $1
		WHERE t.budget_id = $1 AND t.transaction_type = 'expense'
		GROUP BY TO_CHAR(transaction_date, 'YYYY-MM'), c.category_name
		ORDER BY month DESC
		LIMIT 12
	`, budgetID)
	
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	trendMap := make(map[string]*MonthlyBudgetTrend)
	
	for rows.Next() {
		var month, categoryName string
		var total, categoryTotal float64
		
		err := rows.Scan(&month, &total, &categoryName, &categoryTotal)
		if err != nil {
			continue
		}
		
		if _, exists := trendMap[month]; !exists {
			budget, _ := getBudgetByID(budgetID)
			monthlyBudget := budget.TotalAmount / 12
			
			trendMap[month] = &MonthlyBudgetTrend{
				Month:      month,
				Budgeted:   monthlyBudget,
				Spent:      total,
				Categories: make(map[string]float64),
			}
		}
		
		if categoryTotal > 0 {
			trendMap[month].Categories[categoryName] = categoryTotal
		}
	}

	// Convert map to slice
	var trend []MonthlyBudgetTrend
	for _, t := range trendMap {
		trend = append(trend, *t)
	}

	return trend, nil
}

// Alert checking
func checkBudgetAlerts(budgetID, categoryID int) {
	// Check category threshold
	if categoryID > 0 {
		category, err := getBudgetCategoryByID(categoryID)
		if err == nil && category.AllocatedAmount > 0 {
			percentUsed := (category.SpentAmount / category.AllocatedAmount) * 100
			
			// Check thresholds
			thresholds := []struct {
				percent float64
				message string
			}{
				{90, fmt.Sprintf("Category '%s' has reached 90%% of allocated budget", category.CategoryName)},
				{75, fmt.Sprintf("Category '%s' has reached 75%% of allocated budget", category.CategoryName)},
			}
			
			for _, threshold := range thresholds {
				if percentUsed >= threshold.percent {
					// Check if alert already exists
					var exists bool
					err := db.QueryRow(`
						SELECT EXISTS(
							SELECT 1 FROM budget_alerts 
							WHERE budget_id = $1 AND category_id = $2 
							AND alert_type = 'threshold' AND threshold_pct = $3
						)
					`, budgetID, categoryID, threshold.percent).Scan(&exists)
					
					if err == nil && !exists {
						// Create alert
						_, err = db.Exec(`
							INSERT INTO budget_alerts 
							(budget_id, category_id, alert_type, threshold_pct, message, is_active)
							VALUES ($1, $2, 'threshold', $3, $4, true)
						`, budgetID, categoryID, threshold.percent, threshold.message)
						
						// Send notification
						if notificationTriggers != nil {
							go sendBudgetAlertNotification(budgetID, threshold.message)
						}
					}
					break // Only create one alert per check
				}
			}
		}
	}

	// Check overall budget threshold
	budget, err := getBudgetByID(budgetID)
	if err == nil && budget.TotalAmount > 0 {
		percentUsed := (budget.SpentAmount / budget.TotalAmount) * 100
		
		if percentUsed >= 90 {
			message := fmt.Sprintf("Budget has reached 90%% of total allocation ($%.2f of $%.2f)", 
				budget.SpentAmount, budget.TotalAmount)
			
			// Check if alert exists
			var exists bool
			err := db.QueryRow(`
				SELECT EXISTS(
					SELECT 1 FROM budget_alerts 
					WHERE budget_id = $1 AND category_id IS NULL 
					AND alert_type = 'threshold' AND threshold_pct = 90
				)
			`, budgetID).Scan(&exists)
			
			if err == nil && !exists {
				_, err = db.Exec(`
					INSERT INTO budget_alerts 
					(budget_id, alert_type, threshold_pct, message, is_active)
					VALUES ($1, 'threshold', 90, $2, true)
				`, budgetID, message)
				
				if notificationTriggers != nil {
					go sendBudgetAlertNotification(budgetID, message)
				}
			}
		}
	}
}

func sendBudgetAlertNotification(budgetID int, message string) {
	// Get managers to notify
	recipients, _ := notificationTriggers.getManagerRecipients()
	
	notification := Notification{
		Type:     NotifySystemAlert,
		Priority: "high",
		Subject:  "Budget Alert",
		Message:  message,
		Data: map[string]interface{}{
			"budget_id": budgetID,
			"timestamp": time.Now(),
		},
		Channels:   []string{"email", "in-app"},
		Recipients: recipients,
	}
	
	if notificationSystem != nil {
		notificationSystem.Send(notification)
	}
}

// Report generation functions
func generateBudgetSummaryReport(budgetID int) (map[string]interface{}, error) {
	summary, err := getBudgetSummary(budgetID)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"summary":     summary,
		"generated":   time.Now(),
		"report_type": "summary",
	}, nil
}

func generateBudgetVarianceReport(budgetID int) (map[string]interface{}, error) {
	categories, err := getBudgetCategories(budgetID)
	if err != nil {
		return nil, err
	}

	var variances []map[string]interface{}
	for _, cat := range categories {
		variance := map[string]interface{}{
			"category":   cat.CategoryName,
			"budgeted":   cat.AllocatedAmount,
			"spent":      cat.SpentAmount,
			"variance":   cat.AllocatedAmount - cat.SpentAmount,
			"variance_pct": 0.0,
		}
		
		if cat.AllocatedAmount > 0 {
			variance["variance_pct"] = ((cat.AllocatedAmount - cat.SpentAmount) / cat.AllocatedAmount) * 100
		}
		
		variances = append(variances, variance)
	}

	return map[string]interface{}{
		"variances":   variances,
		"generated":   time.Now(),
		"report_type": "variance",
	}, nil
}

func generateBudgetProjectionReport(budgetID int) (map[string]interface{}, error) {
	summary, err := getBudgetSummary(budgetID)
	if err != nil {
		return nil, err
	}

	// Calculate projections
	monthsElapsed := float64(time.Now().Month() - 6)
	if monthsElapsed < 1 {
		monthsElapsed = 1
	}
	monthsRemaining := 12 - monthsElapsed

	projections := map[string]interface{}{
		"current_spent":     summary.Budget.SpentAmount,
		"monthly_rate":      summary.Budget.SpentAmount / monthsElapsed,
		"projected_total":   (summary.Budget.SpentAmount / monthsElapsed) * 12,
		"projected_surplus": summary.Budget.TotalAmount - ((summary.Budget.SpentAmount / monthsElapsed) * 12),
		"months_remaining":  monthsRemaining,
	}

	return map[string]interface{}{
		"budget":      summary.Budget,
		"projections": projections,
		"categories":  summary.Categories,
		"generated":   time.Now(),
		"report_type": "projection",
	}, nil
}

func generateCategoryDetailReport(budgetID, categoryID int) (map[string]interface{}, error) {
	category, err := getBudgetCategoryByID(categoryID)
	if err != nil {
		return nil, err
	}

	// Get all transactions for this category
	rows, err := db.Query(`
		SELECT id, transaction_date, amount, description, vehicle_id,
			   reference_id, reference_type, created_by, created_at
		FROM budget_transactions
		WHERE category_id = $1
		ORDER BY transaction_date DESC
	`, categoryID)
	
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var transactions []map[string]interface{}
	for rows.Next() {
		var t BudgetTransaction
		err := rows.Scan(&t.ID, &t.TransactionDate, &t.Amount, &t.Description,
			&t.VehicleID, &t.ReferenceID, &t.ReferenceType, &t.CreatedBy, &t.CreatedAt)
		if err != nil {
			continue
		}
		
		transaction := map[string]interface{}{
			"id":          t.ID,
			"date":        t.TransactionDate,
			"amount":      t.Amount,
			"description": t.Description,
			"vehicle_id":  t.VehicleID,
			"created_by":  t.CreatedBy,
		}
		transactions = append(transactions, transaction)
	}

	return map[string]interface{}{
		"category":     category,
		"transactions": transactions,
		"total_count":  len(transactions),
		"generated":    time.Now(),
		"report_type":  "category_detail",
	}, nil
}

// Integration with existing expense tracking
func linkFuelExpenseToBudget(fuelRecordID int) error {
	// Get fuel record details
	var fuel struct {
		VehicleID string
		Cost      float64
		Date      time.Time
		Driver    string
	}
	
	err := db.QueryRow(`
		SELECT vehicle_id, cost, date, driver
		FROM fuel_records
		WHERE id = $1
	`, fuelRecordID).Scan(&fuel.VehicleID, &fuel.Cost, &fuel.Date, &fuel.Driver)
	
	if err != nil {
		return err
	}

	// Get current budget
	fiscalYear := fuel.Date.Year()
	if fuel.Date.Month() < 7 {
		fiscalYear--
	}
	
	budget, err := getCurrentBudget(fiscalYear)
	if err != nil || budget == nil {
		return nil // No active budget
	}

	// Find fuel category
	var categoryID int
	err = db.QueryRow(`
		SELECT id FROM budget_categories
		WHERE budget_id = $1 AND category_type = 'fuel'
		LIMIT 1
	`, budget.ID).Scan(&categoryID)
	
	if err != nil {
		return nil // No fuel category
	}

	// Create budget transaction
	refID := strconv.Itoa(fuelRecordID)
	refType := "fuel_record"
	
	transaction := BudgetTransaction{
		BudgetID:        budget.ID,
		CategoryID:      categoryID,
		TransactionDate: fuel.Date,
		Amount:          fuel.Cost,
		TransactionType: "expense",
		Description:     fmt.Sprintf("Fuel for vehicle %s", fuel.VehicleID),
		VehicleID:       &fuel.VehicleID,
		ReferenceID:     &refID,
		ReferenceType:   &refType,
		CreatedBy:       fuel.Driver,
	}

	return recordBudgetTransaction(transaction)
}

func linkMaintenanceExpenseToBudget(maintenanceRecordID int) error {
	// Get maintenance record details
	var maintenance struct {
		VehicleID string
		Cost      float64
		Date      time.Time
		Service   string
	}
	
	err := db.QueryRow(`
		SELECT vehicle_id, cost, service_date, service_type
		FROM maintenance_records
		WHERE id = $1
	`, maintenanceRecordID).Scan(&maintenance.VehicleID, &maintenance.Cost, 
		&maintenance.Date, &maintenance.Service)
	
	if err != nil {
		return err
	}

	// Get current budget
	fiscalYear := maintenance.Date.Year()
	if maintenance.Date.Month() < 7 {
		fiscalYear--
	}
	
	budget, err := getCurrentBudget(fiscalYear)
	if err != nil || budget == nil {
		return nil // No active budget
	}

	// Find maintenance category
	var categoryID int
	err = db.QueryRow(`
		SELECT id FROM budget_categories
		WHERE budget_id = $1 AND category_type = 'maintenance'
		LIMIT 1
	`, budget.ID).Scan(&categoryID)
	
	if err != nil {
		return nil // No maintenance category
	}

	// Create budget transaction
	refID := strconv.Itoa(maintenanceRecordID)
	refType := "maintenance_record"
	
	transaction := BudgetTransaction{
		BudgetID:        budget.ID,
		CategoryID:      categoryID,
		TransactionDate: maintenance.Date,
		Amount:          maintenance.Cost,
		TransactionType: "expense",
		Description:     fmt.Sprintf("%s for vehicle %s", maintenance.Service, maintenance.VehicleID),
		VehicleID:       &maintenance.VehicleID,
		ReferenceID:     &refID,
		ReferenceType:   &refType,
		CreatedBy:       "system",
	}

	return recordBudgetTransaction(transaction)
}