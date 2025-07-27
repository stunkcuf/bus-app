package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"
)

// ReportBuilder handles custom report generation
type ReportBuilder struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	DataSource  string                 `json:"data_source"`
	Filters     map[string]interface{} `json:"filters"`
	Fields      []ReportField          `json:"fields"`
	Grouping    []string               `json:"grouping"`
	Sorting     []SortConfig           `json:"sorting"`
	SortBy      string                 `json:"sort_by"`
	SortOrder   string                 `json:"sort_order"`
	ChartType   string                 `json:"chart_type"`
	ChartConfig *ChartConfig           `json:"chart_config,omitempty"`
	CreatedBy   string                 `json:"created_by"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

type ReportField struct {
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	Type        string `json:"type"`
	Aggregation string `json:"aggregation,omitempty"`
	Formula     string `json:"formula,omitempty"`
}

type SortConfig struct {
	Field     string `json:"field"`
	Direction string `json:"direction"`
}

type ChartConfig struct {
	Type   string            `json:"type"`
	XAxis  string            `json:"x_axis"`
	YAxis  string            `json:"y_axis"`
	Color  string            `json:"color,omitempty"`
	Title  string            `json:"title"`
	Config map[string]string `json:"config,omitempty"`
}

// Available data sources
var reportDataSources = map[string]DataSourceConfig{
	"fleet": {
		Name:        "Fleet Management",
		Description: "Bus and vehicle data",
		Table:       "buses",
		JoinTables: []JoinConfig{
			{Table: "route_assignments", On: "buses.bus_id = route_assignments.bus_id", Type: "LEFT"},
			{Table: "routes", On: "route_assignments.route_id = routes.id", Type: "LEFT"},
		},
		Fields: []FieldConfig{
			{Name: "bus_id", DisplayName: "Bus ID", Type: "string"},
			{Name: "model", DisplayName: "Model", Type: "string"},
			{Name: "status", DisplayName: "Status", Type: "string"},
			{Name: "current_mileage", DisplayName: "Current Mileage", Type: "number"},
			{Name: "routes.name", DisplayName: "Route Name", Type: "string"},
			{Name: "last_oil_change", DisplayName: "Last Oil Change", Type: "number"},
			{Name: "last_tire_service", DisplayName: "Last Tire Service", Type: "number"},
		},
	},
	"students": {
		Name:        "Student Management",
		Description: "Student roster and route assignments",
		Table:       "students",
		JoinTables: []JoinConfig{
			{Table: "routes", On: "students.route_id = routes.id", Type: "LEFT"},
		},
		Fields: []FieldConfig{
			{Name: "name", DisplayName: "Student Name", Type: "string"},
			{Name: "grade", DisplayName: "Grade", Type: "string"},
			{Name: "address", DisplayName: "Address", Type: "string"},
			{Name: "phone", DisplayName: "Phone", Type: "string"},
			{Name: "guardian_name", DisplayName: "Guardian", Type: "string"},
			{Name: "routes.name", DisplayName: "Route", Type: "string"},
			{Name: "driver", DisplayName: "Driver", Type: "string"},
			{Name: "pickup_time", DisplayName: "Pickup Time", Type: "time"},
			{Name: "dropoff_time", DisplayName: "Dropoff Time", Type: "time"},
			{Name: "active", DisplayName: "Active", Type: "boolean"},
		},
	},
	"trips": {
		Name:        "Trip Logs",
		Description: "Daily trip and mileage data",
		Table:       "trip_logs",
		Fields: []FieldConfig{
			{Name: "date", DisplayName: "Date", Type: "date"},
			{Name: "driver", DisplayName: "Driver", Type: "string"},
			{Name: "vehicle_id", DisplayName: "Vehicle ID", Type: "string"},
			{Name: "route", DisplayName: "Route", Type: "string"},
			{Name: "period", DisplayName: "Period", Type: "string"},
			{Name: "beginning_mileage", DisplayName: "Beginning Mileage", Type: "number"},
			{Name: "ending_mileage", DisplayName: "Ending Mileage", Type: "number"},
			{Name: "departure_time", DisplayName: "Departure Time", Type: "time"},
			{Name: "arrival_time", DisplayName: "Arrival Time", Type: "time"},
			{Name: "students_picked_up", DisplayName: "Students Picked Up", Type: "number"},
			{Name: "students_dropped_off", DisplayName: "Students Dropped Off", Type: "number"},
		},
	},
	"maintenance": {
		Name:        "Maintenance Records",
		Description: "Vehicle maintenance and costs",
		Table:       "bus_maintenance_logs",
		UnionTables: []string{"vehicle_maintenance_logs"},
		Fields: []FieldConfig{
			{Name: "date", DisplayName: "Date", Type: "date"},
			{Name: "vehicle_id", DisplayName: "Vehicle ID", Type: "string"},
			{Name: "maintenance_type", DisplayName: "Type", Type: "string"},
			{Name: "description", DisplayName: "Description", Type: "string"},
			{Name: "mileage", DisplayName: "Mileage", Type: "number"},
			{Name: "cost", DisplayName: "Cost", Type: "number"},
			{Name: "performed_by", DisplayName: "Performed By", Type: "string"},
		},
	},
	"mileage": {
		Name:        "Mileage Reports",
		Description: "Monthly mileage and fuel data",
		Table:       "mileage_reports",
		Fields: []FieldConfig{
			{Name: "vehicle_id", DisplayName: "Vehicle ID", Type: "string"},
			{Name: "month", DisplayName: "Month", Type: "number"},
			{Name: "year", DisplayName: "Year", Type: "number"},
			{Name: "beginning_mileage", DisplayName: "Beginning Mileage", Type: "number"},
			{Name: "ending_mileage", DisplayName: "Ending Mileage", Type: "number"},
			{Name: "total_miles", DisplayName: "Total Miles", Type: "number"},
			{Name: "driver", DisplayName: "Driver", Type: "string"},
		},
	},
}

type DataSourceConfig struct {
	Name        string
	Description string
	Table       string
	JoinTables  []JoinConfig
	UnionTables []string
	Fields      []FieldConfig
}

type JoinConfig struct {
	Table string
	On    string
	Type  string
}

type FieldConfig struct {
	Name        string
	DisplayName string
	Type        string
}

// reportBuilderHandler serves the report builder interface
func reportBuilderHandler(w http.ResponseWriter, r *http.Request) {
	session, err := GetSession(r)
	if err != nil {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	if session.Role != "manager" {
		SendError(w, ErrForbidden("Manager access required"))
		return
	}
	
	// Load saved reports
	savedReports, err := loadSavedReports(session.Username)
	if err != nil {
		log.Printf("Error loading saved reports: %v", err)
		savedReports = []map[string]interface{}{}
	}

	data := struct {
		User         *User                       `json:"user"`
		Role         string                      `json:"role"`
		CSRFToken    string                      `json:"csrf_token"`
		CSPNonce     string                      `json:"csp_nonce"`
		DataSources  map[string]DataSourceConfig `json:"data_sources"`
		SavedReports []map[string]interface{}    `json:"saved_reports"`
	}{
		User:         &User{Username: session.Username},
		Role:         session.Role,
		CSRFToken:    session.CSRFToken,
		CSPNonce:     GenerateCSPNonce(),
		DataSources:  reportDataSources,
		SavedReports: savedReports,
	}

	if err := templates.ExecuteTemplate(w, "report_builder.html", data); err != nil {
		LogError("Failed to execute report builder template", err)
		SendError(w, ErrInternal("Failed to render page", err))
		return
	}
}

// reportBuilderAPIHandler handles API requests for report building
func reportBuilderAPIHandler(w http.ResponseWriter, r *http.Request) {
	session, err := GetSession(r)
	if err != nil {
		SendError(w, ErrUnauthorized("Please log in"))
		return
	}

	if session.Role != "manager" {
		SendError(w, ErrForbidden("Manager access required"))
		return
	}

	switch r.Method {
	case "GET":
		handleGetReportData(w, r)
	case "POST":
		handleSaveReport(w, r)
	default:
		SendError(w, ErrMethodNotAllowed("Method not allowed"))
	}
}

// handleGetReportData generates and returns report data
func handleGetReportData(w http.ResponseWriter, r *http.Request) {
	dataSource := r.URL.Query().Get("data_source")
	if dataSource == "" {
		SendError(w, ErrBadRequest("Data source required"))
		return
	}

	sourceConfig, exists := reportDataSources[dataSource]
	if !exists {
		SendError(w, ErrBadRequest("Invalid data source"))
		return
	}

	// Parse filters
	filters := parseFilters(r.URL.Query())

	// Parse fields
	fieldsParam := r.URL.Query().Get("fields")
	var fields []string
	if fieldsParam != "" {
		fields = strings.Split(fieldsParam, ",")
	} else {
		// Default to all fields
		for _, field := range sourceConfig.Fields {
			fields = append(fields, field.Name)
		}
	}

	// Build query
	query, params := buildReportQuery(sourceConfig, fields, filters)
	if query == "" {
		SendError(w, ErrBadRequest("Invalid query parameters"))
		return
	}

	// Execute query
	rows, err := db.Query(query, params...)
	if err != nil {
		LogError("Failed to execute report query", err)
		SendError(w, ErrInternal("Failed to generate report", err))
		return
	}
	defer rows.Close()

	// Get column names
	columns, err := rows.Columns()
	if err != nil {
		SendError(w, ErrInternal("Failed to get columns", err))
		return
	}

	// Scan results
	var results []map[string]interface{}
	for rows.Next() {
		// Create slice to hold values
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		// Scan row
		if err := rows.Scan(valuePtrs...); err != nil {
			LogError("Failed to scan row", err)
			continue
		}

		// Convert to map
		row := make(map[string]interface{})
		for i, col := range columns {
			if values[i] != nil {
				row[col] = values[i]
			}
		}
		results = append(results, row)
	}

	response := struct {
		Success bool                     `json:"success"`
		Data    []map[string]interface{} `json:"data"`
		Columns []string                 `json:"columns"`
		Count   int                      `json:"count"`
	}{
		Success: true,
		Data:    results,
		Columns: columns,
		Count:   len(results),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleSaveReport saves a custom report configuration
func handleSaveReport(w http.ResponseWriter, r *http.Request) {
	session, _ := GetSession(r)
	
	var report ReportBuilder
	if err := json.NewDecoder(r.Body).Decode(&report); err != nil {
		SendError(w, ErrBadRequest("Invalid report data"))
		return
	}
	
	// Validate report
	if report.Name == "" {
		SendError(w, ErrBadRequest("Report name is required"))
		return
	}
	
	if report.DataSource == "" {
		SendError(w, ErrBadRequest("Data source is required"))
		return
	}
	
	// Convert fields and filters to JSON
	fieldsJSON, _ := json.Marshal(report.Fields)
	filtersJSON, _ := json.Marshal(report.Filters)
	chartConfigJSON, _ := json.Marshal(report.ChartConfig)
	
	// Save to database
	var reportID int
	err := db.QueryRow(`
		INSERT INTO saved_reports 
		(name, description, data_source, fields, filters, sort_by, sort_order, 
		 chart_type, chart_config, created_by, is_public)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING id`,
		report.Name,
		report.Description,
		report.DataSource,
		string(fieldsJSON),
		string(filtersJSON),
		report.SortBy,
		report.SortOrder,
		report.ChartType,
		string(chartConfigJSON),
		session.Username,
		false, // Default to private
	).Scan(&reportID)
	
	if err != nil {
		log.Printf("Error saving report: %v", err)
		SendError(w, ErrInternal("Failed to save report", err))
		return
	}
	
	response := struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
		ID      int    `json:"id"`
	}{
		Success: true,
		Message: "Report saved successfully",
		ID:      reportID,
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// loadSavedReports loads saved reports for a user
func loadSavedReports(username string) ([]map[string]interface{}, error) {
	query := `
		SELECT id, name, description, data_source, chart_type, 
		       created_at, updated_at, last_run, is_public
		FROM saved_reports
		WHERE created_by = $1 OR is_public = true
		ORDER BY updated_at DESC
	`
	
	rows, err := db.Query(query, username)
	if err != nil {
		return nil, fmt.Errorf("failed to load saved reports: %w", err)
	}
	defer rows.Close()
	
	var reports []map[string]interface{}
	for rows.Next() {
		var id int
		var name, dataSource string
		var description, chartType sql.NullString
		var createdAt, updatedAt time.Time
		var lastRun sql.NullTime
		var isPublic bool
		
		err := rows.Scan(&id, &name, &description, &dataSource, &chartType,
			&createdAt, &updatedAt, &lastRun, &isPublic)
		if err != nil {
			continue
		}
		
		report := map[string]interface{}{
			"id":          id,
			"name":        name,
			"description": description.String,
			"dataSource":  dataSource,
			"chartType":   chartType.String,
			"createdAt":   createdAt.Format("2006-01-02 15:04"),
			"updatedAt":   updatedAt.Format("2006-01-02 15:04"),
			"isPublic":    isPublic,
		}
		
		if lastRun.Valid {
			report["lastRun"] = lastRun.Time.Format("2006-01-02 15:04")
		}
		
		reports = append(reports, report)
	}
	
	return reports, nil
}

// buildReportQuery constructs SQL query based on configuration
// Returns the query string and the parameters to use with it
func buildReportQuery(source DataSourceConfig, fields []string, filters map[string]interface{}) (string, []interface{}) {
	// Validate fields and filters to prevent SQL injection
	for _, field := range fields {
		if !isValidFieldName(field) {
			log.Printf("Invalid field name: %s", field)
			return "", nil
		}
	}

	for fieldName := range filters {
		if !isValidFieldName(fieldName) {
			log.Printf("Invalid filter field name: %s", fieldName)
			return "", nil
		}
	}

	// Build SELECT clause
	selectClause := strings.Join(fields, ", ")

	// Build FROM clause with JOINs
	fromClause := source.Table
	for _, join := range source.JoinTables {
		fromClause += fmt.Sprintf(" %s JOIN %s ON %s", join.Type, join.Table, join.On)
	}

	// Build WHERE clause with parameterized queries
	var whereClauses []string
	var params []interface{}
	paramIndex := 1

	for field, value := range filters {
		if value != nil && value != "" {
			switch v := value.(type) {
			case string:
				if v != "" {
					whereClauses = append(whereClauses, fmt.Sprintf("%s ILIKE $%d", field, paramIndex))
					params = append(params, "%"+v+"%")
					paramIndex++
				}
			case []string:
				if len(v) > 0 {
					placeholders := make([]string, len(v))
					for i, val := range v {
						placeholders[i] = fmt.Sprintf("$%d", paramIndex)
						params = append(params, val)
						paramIndex++
					}
					whereClauses = append(whereClauses, fmt.Sprintf("%s IN (%s)", field, strings.Join(placeholders, ",")))
				}
			default:
				whereClauses = append(whereClauses, fmt.Sprintf("%s = $%d", field, paramIndex))
				params = append(params, v)
				paramIndex++
			}
		}
	}

	whereClause := "1=1"
	if len(whereClauses) > 0 {
		whereClause += " AND " + strings.Join(whereClauses, " AND ")
	}

	query := fmt.Sprintf("SELECT %s FROM %s WHERE %s", selectClause, fromClause, whereClause)

	// Handle UNION tables
	if len(source.UnionTables) > 0 {
		for _, unionTable := range source.UnionTables {
			// Replace table references in fields
			unionFields := make([]string, len(fields))
			for i, field := range fields {
				unionFields[i] = strings.Replace(field, source.Table, unionTable, 1)
			}

			unionQuery := fmt.Sprintf("SELECT %s FROM %s WHERE %s",
				strings.Join(unionFields, ", "),
				unionTable,
				strings.Replace(whereClause, source.Table, unionTable, -1))

			query += " UNION ALL " + unionQuery
		}
	}

	return query + " ORDER BY 1 LIMIT 1000", params
}

// isValidFieldName validates that a field name contains only allowed characters
// to prevent SQL injection through field names
func isValidFieldName(field string) bool {
	// Allow alphanumeric, underscore, dot (for table.column), and common aggregation functions
	for _, char := range field {
		if !((char >= 'a' && char <= 'z') ||
			(char >= 'A' && char <= 'Z') ||
			(char >= '0' && char <= '9') ||
			char == '_' || char == '.' || char == '(' || char == ')' || char == '*' || char == ' ') {
			return false
		}
	}
	
	// Additional check for common SQL injection patterns
	field = strings.ToLower(field)
	if strings.Contains(field, "--") || strings.Contains(field, "/*") || strings.Contains(field, "*/") ||
		strings.Contains(field, "xp_") || strings.Contains(field, "sp_") {
		return false
	}
	
	return true
}

// parseFilters extracts filter parameters from URL query
func parseFilters(values map[string][]string) map[string]interface{} {
	filters := make(map[string]interface{})

	for key, value := range values {
		if strings.HasPrefix(key, "filter_") {
			fieldName := strings.TrimPrefix(key, "filter_")
			if len(value) > 0 && value[0] != "" {
				filters[fieldName] = value[0]
			}
		}
	}

	return filters
}

// getReportDataSourcesHandler returns available data sources
func getReportDataSourcesHandler(w http.ResponseWriter, r *http.Request) {
	session, err := GetSession(r)
	if err != nil {
		SendError(w, ErrUnauthorized("Please log in"))
		return
	}

	if session.Role != "manager" {
		SendError(w, ErrForbidden("Manager access required"))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(reportDataSources)
}

// Chart type configurations
var chartTypes = map[string]ChartTypeConfig{
	"line": {
		Name:           "Line Chart",
		Description:    "Shows trends over time",
		RequiredAxes:   []string{"x", "y"},
		SupportedTypes: []string{"date", "number", "string"},
	},
	"bar": {
		Name:           "Bar Chart",
		Description:    "Compares categories",
		RequiredAxes:   []string{"x", "y"},
		SupportedTypes: []string{"string", "number"},
	},
	"pie": {
		Name:           "Pie Chart",
		Description:    "Shows proportions",
		RequiredAxes:   []string{"label", "value"},
		SupportedTypes: []string{"string", "number"},
	},
	"scatter": {
		Name:           "Scatter Plot",
		Description:    "Shows correlation between variables",
		RequiredAxes:   []string{"x", "y"},
		SupportedTypes: []string{"number"},
	},
}

type ChartTypeConfig struct {
	Name           string
	Description    string
	RequiredAxes   []string
	SupportedTypes []string
}

// reportChartTypesHandler returns available chart types
func reportChartTypesHandler(w http.ResponseWriter, r *http.Request) {
	_, err := GetSession(r)
	if err != nil {
		SendError(w, ErrUnauthorized("Please log in"))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(chartTypes)
}
