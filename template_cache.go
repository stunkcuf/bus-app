package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"os"
	"strings"
	"sync"
	"time"
)

// TemplateCache manages compiled templates with optimizations
type TemplateCache struct {
	templates map[string]*template.Template
	funcs     template.FuncMap
	mu        sync.RWMutex

	// Optimization settings
	enableCache   bool
	cacheTimeout  time.Duration
	precompileAll bool
	minifyHTML    bool

	// Template render cache for static content
	renderCache map[string]*CachedRender
	renderMu    sync.RWMutex
}

// CachedRender stores pre-rendered template output
type CachedRender struct {
	Content   []byte
	Hash      string
	CachedAt  time.Time
	ExpiresAt time.Time
}

// NewTemplateCache creates an optimized template cache
func NewTemplateCache(enableCache bool) *TemplateCache {
	return &TemplateCache{
		templates:     make(map[string]*template.Template),
		funcs:         getTemplateFuncs(),
		enableCache:   enableCache,
		cacheTimeout:  5 * time.Minute,
		precompileAll: true,
		minifyHTML:    true,
		renderCache:   make(map[string]*CachedRender),
	}
}

// LoadTemplates loads and compiles all templates
func (tc *TemplateCache) LoadTemplates(templateDir string) error {
	tc.mu.Lock()
	defer tc.mu.Unlock()

	// Base template functions
	baseFuncs := tc.funcs

	// Component templates
	componentFiles := []string{
		"components/navigation.html",
		"components/pagination.html",
		"components/dashboard_shortcuts.html",
		"components/progress_indicator.html",
	}
	
	// List of template files to load
	templateFiles := []string{
		// Authentication
		"login.html",
		"register.html",
		"registration_success.html",
		
		// Dashboards
		"manager_dashboard.html",
		"manager_dashboard_modern.html",
		"driver_dashboard.html",
		
		// Fleet Management
		"fleet.html",
		"fleet_modern.html",
		"company_fleet.html",
		"company_fleet_modern.html",
		"fleet_vehicles.html",
		"vehicle_maintenance.html",
		"maintenance_records.html",
		"service_records.html",
		"add_bus_wizard.html",
		
		// User Management
		"users.html",
		"approve_users.html",
		"driver_profile.html",
		
		// Student & Route Management
		"students.html",
		"students_modern.html",
		"add_student_wizard.html",
		"assign_routes.html",
		
		// ECSE Management
		"import_ecse.html",
		"view_ecse_reports.html",
		"view_ecse_student.html",
		"edit_ecse_student.html",
		"ecse_dashboard.html",
		"ecse_dashboard_modern.html",
		"ecse_student_details.html",
		
		// Mileage & Reports
		"import_mileage.html",
		"view_mileage_reports.html",
		"mileage-report-generator.html",
		"monthly_mileage_reports.html",
		
		// Fuel Management
		"fuel_records.html",
		"add_fuel_record.html",
		"fuel_analytics.html",
		
		// Export & Reports
		"export_templates.html",
		"scheduled_exports.html",
		"report_builder.html",
		
		// Error handling
		"error.html",
		
		// Components
		"components/pagination.html",
		
		// Future import system (not yet implemented)
		// "import.html",
		// "import_history.html",
		// "import_mapping.html",
		// "import_preview.html",
		// "import_result.html",
		// "import_details.html",
	}

	// Compile each template
	for _, file := range templateFiles {
		// Skip commented out templates
		if strings.HasPrefix(file, "//") {
			continue
		}
		
		tmpl := template.New(file).Funcs(baseFuncs)

		// First parse component templates
		for _, comp := range componentFiles {
			compPath := templateDir + "/" + comp
			_, err := tmpl.ParseFiles(compPath)
			if err != nil && !os.IsNotExist(err) {
				return fmt.Errorf("failed to parse component %s: %w", comp, err)
			}
		}

		// Parse the template file
		filePath := templateDir + "/" + file
		parsedTmpl, err := tmpl.ParseFiles(filePath)
		if err != nil {
			// Skip if file doesn't exist (for optional templates)
			if os.IsNotExist(err) {
				continue
			}
			return fmt.Errorf("failed to parse template %s: %w", file, err)
		}

		// Optimize template if enabled
		if tc.minifyHTML {
			// In production, we could add HTML minification here
			// For now, just store the compiled template
		}

		tc.templates[file] = parsedTmpl
	}

	return nil
}

// GetTemplate retrieves a compiled template
func (tc *TemplateCache) GetTemplate(name string) (*template.Template, error) {
	tc.mu.RLock()
	defer tc.mu.RUnlock()

	tmpl, exists := tc.templates[name]
	if !exists {
		return nil, fmt.Errorf("template %s not found", name)
	}

	// Clone the template to avoid concurrent access issues
	cloned, err := tmpl.Clone()
	if err != nil {
		return nil, fmt.Errorf("failed to clone template: %w", err)
	}

	return cloned, nil
}

// RenderCached renders a template with caching for static content
func (tc *TemplateCache) RenderCached(w io.Writer, name string, data interface{}, cacheKey string) error {
	// Check if caching is enabled and we have a cache key
	if tc.enableCache && cacheKey != "" {
		tc.renderMu.RLock()
		cached, exists := tc.renderCache[cacheKey]
		tc.renderMu.RUnlock()

		if exists && time.Now().Before(cached.ExpiresAt) {
			// Serve from cache
			_, err := w.Write(cached.Content)
			return err
		}
	}

	// Get the template
	tmpl, err := tc.GetTemplate(name)
	if err != nil {
		return err
	}

	// Render to buffer first
	var buf bytes.Buffer
	err = tmpl.Execute(&buf, data)
	if err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	content := buf.Bytes()

	// Cache the rendered content if enabled
	if tc.enableCache && cacheKey != "" {
		tc.renderMu.Lock()
		tc.renderCache[cacheKey] = &CachedRender{
			Content:   content,
			CachedAt:  time.Now(),
			ExpiresAt: time.Now().Add(tc.cacheTimeout),
		}
		tc.renderMu.Unlock()
	}

	// Write to output
	_, err = w.Write(content)
	return err
}

// Render renders a template without caching
func (tc *TemplateCache) Render(w io.Writer, name string, data interface{}) error {
	return tc.RenderCached(w, name, data, "")
}

// ClearCache clears the render cache
func (tc *TemplateCache) ClearCache() {
	tc.renderMu.Lock()
	tc.renderCache = make(map[string]*CachedRender)
	tc.renderMu.Unlock()
}

// ClearExpiredCache removes expired entries from the render cache
func (tc *TemplateCache) ClearExpiredCache() {
	tc.renderMu.Lock()
	defer tc.renderMu.Unlock()

	now := time.Now()
	for key, cached := range tc.renderCache {
		if now.After(cached.ExpiresAt) {
			delete(tc.renderCache, key)
		}
	}
}

// StartCacheCleaner starts a background goroutine to clean expired cache entries
func (tc *TemplateCache) StartCacheCleaner(interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for range ticker.C {
			tc.ClearExpiredCache()
		}
	}()
}

// getTemplateFuncs returns common template functions
func getTemplateFuncs() template.FuncMap {
	return template.FuncMap{
		"formatDate": func(date string) string {
			t, err := time.Parse("2006-01-02", date)
			if err != nil {
				return date
			}
			return t.Format("Jan 02, 2006")
		},
		"formatTime": func(timeStr string) string {
			t, err := time.Parse("15:04", timeStr)
			if err != nil {
				return timeStr
			}
			return t.Format("3:04 PM")
		},
		"formatDateTime": func(dt time.Time) string {
			return dt.Format("Jan 02, 2006 3:04 PM")
		},
		"truncate": func(s string, n int) string {
			if len(s) <= n {
				return s
			}
			return s[:n] + "..."
		},
		"add": func(a, b int) int {
			return a + b
		},
		"sub": func(a, b int) int {
			return a - b
		},
		"mul": func(a, b int) int {
			return a * b
		},
		"div": func(a, b int) int {
			if b == 0 {
				return 0
			}
			return a / b
		},
		"percentage": func(a, b int) float64 {
			if b == 0 {
				return 0
			}
			return float64(a) / float64(b) * 100
		},
		"statusClass": func(status string) string {
			switch status {
			case "active", "good", "operational":
				return "status-active"
			case "maintenance", "warning", "pending":
				return "status-warning"
			case "out of service", "critical", "inactive":
				return "status-inactive"
			default:
				return "status-unknown"
			}
		},
		"json": func(v interface{}) string {
			b, _ := json.Marshal(v)
			return string(b)
		},
		"substr": func(s string, start, length int) string {
			if start < 0 || start >= len(s) {
				return ""
			}
			end := start + length
			if end > len(s) {
				end = len(s)
			}
			return s[start:end]
		},
	}
}

// TemplateOptimizer provides template optimization utilities
type TemplateOptimizer struct {
	enableMinification bool
	enableCompression  bool
	cacheDuration      time.Duration
}

// NewTemplateOptimizer creates a new template optimizer
func NewTemplateOptimizer() *TemplateOptimizer {
	return &TemplateOptimizer{
		enableMinification: true,
		enableCompression:  true,
		cacheDuration:      5 * time.Minute,
	}
}

// OptimizeHTML removes unnecessary whitespace from HTML
func (to *TemplateOptimizer) OptimizeHTML(html []byte) []byte {
	if !to.enableMinification {
		return html
	}

	// Simple minification - remove extra whitespace
	// In production, use a proper HTML minifier
	result := bytes.ReplaceAll(html, []byte("  "), []byte(" "))
	result = bytes.ReplaceAll(result, []byte("\n\n"), []byte("\n"))
	result = bytes.ReplaceAll(result, []byte("\t"), []byte(""))

	return result
}

// PrecompileTemplates loads and compiles all templates at startup
func PrecompileTemplates(templateDir string) (*TemplateCache, error) {
	cache := NewTemplateCache(true)

	err := cache.LoadTemplates(templateDir)
	if err != nil {
		return nil, fmt.Errorf("failed to precompile templates: %w", err)
	}

	// Start cache cleaner
	cache.StartCacheCleaner(1 * time.Minute)

	return cache, nil
}
