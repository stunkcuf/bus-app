package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"strings"
)

func main() {
	// Create cookie jar to handle session cookies
	jar, err := cookiejar.New(nil)
	if err != nil {
		log.Fatal("Failed to create cookie jar:", err)
	}

	client := &http.Client{
		Jar: jar,
	}

	baseURL := "http://localhost:5000"
	if len(os.Args) > 1 {
		baseURL = os.Args[1]
	}

	// Step 1: Login
	loginURL := baseURL + "/"
	loginData := url.Values{
		"username": {"admin"},
		"password": {"SecureAdminPass123!"},
	}

	fmt.Println("Logging in as admin...")
	resp, err := client.PostForm(loginURL, loginData)
	if err != nil {
		log.Fatal("Failed to login:", err)
	}
	defer resp.Body.Close()

	// Check if login was successful (should redirect to dashboard)
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusSeeOther {
		log.Fatal("Login failed with status:", resp.StatusCode)
	}

	// Step 2: Fetch manager dashboard
	fmt.Println("\nFetching manager dashboard...")
	dashboardURL := baseURL + "/manager-dashboard"
	resp, err = client.Get(dashboardURL)
	if err != nil {
		log.Fatal("Failed to fetch dashboard:", err)
	}
	defer resp.Body.Close()

	dashboardHTML, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal("Failed to read dashboard response:", err)
	}

	// Save dashboard HTML
	err = os.WriteFile("manager_dashboard_actual.html", dashboardHTML, 0644)
	if err != nil {
		log.Fatal("Failed to save dashboard HTML:", err)
	}
	fmt.Println("Manager dashboard HTML saved to manager_dashboard_actual.html")

	// Extract key content from dashboard
	dashboardContent := string(dashboardHTML)
	fmt.Println("\nManager Dashboard Key Content:")
	extractContent(dashboardContent, "h1", "Main heading")
	extractContent(dashboardContent, "h2", "Section headings")
	extractContent(dashboardContent, ".metric-label", "Metric labels")
	extractContent(dashboardContent, ".quick-action span", "Quick action labels")

	// Step 3: Fetch students page
	fmt.Println("\n\nFetching students page...")
	studentsURL := baseURL + "/students"
	resp, err = client.Get(studentsURL)
	if err != nil {
		log.Fatal("Failed to fetch students page:", err)
	}
	defer resp.Body.Close()

	studentsHTML, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal("Failed to read students response:", err)
	}

	// Save students HTML
	err = os.WriteFile("students_page_actual.html", studentsHTML, 0644)
	if err != nil {
		log.Fatal("Failed to save students HTML:", err)
	}
	fmt.Println("Students page HTML saved to students_page_actual.html")

	// Extract key content from students page
	studentsContent := string(studentsHTML)
	fmt.Println("\nStudents Page Key Content:")
	extractContent(studentsContent, "h1", "Main heading")
	extractContent(studentsContent, "h2", "Section headings")
	extractContent(studentsContent, ".stat-label", "Stat labels")
	extractContent(studentsContent, ".btn-primary", "Primary buttons")
}

func extractContent(html, selector, description string) {
	// Simple extraction based on common patterns
	fmt.Printf("\n%s (%s):\n", description, selector)
	
	switch selector {
	case "h1":
		start := strings.Index(html, "<h1")
		if start >= 0 {
			end := strings.Index(html[start:], "</h1>")
			if end >= 0 {
				h1Content := html[start : start+end+5]
				// Clean up the content
				h1Text := extractText(h1Content)
				fmt.Printf("  - %s\n", h1Text)
			}
		}
	case "h2":
		parts := strings.Split(html, "<h2")
		for i := 1; i < len(parts); i++ {
			end := strings.Index(parts[i], "</h2>")
			if end >= 0 {
				h2Content := "<h2" + parts[i][:end+5]
				h2Text := extractText(h2Content)
				if h2Text != "" && !strings.Contains(h2Text, "sr-only") {
					fmt.Printf("  - %s\n", h2Text)
				}
			}
		}
	case ".metric-label", ".stat-label":
		className := strings.TrimPrefix(selector, ".")
		parts := strings.Split(html, `class="`+className+`"`)
		for i := 1; i < len(parts); i++ {
			start := strings.Index(parts[i], ">")
			if start >= 0 {
				end := strings.Index(parts[i][start:], "<")
				if end >= 0 {
					text := strings.TrimSpace(parts[i][start+1 : start+end])
					if text != "" {
						fmt.Printf("  - %s\n", text)
					}
				}
			}
		}
	case ".quick-action span":
		parts := strings.Split(html, `class="quick-action"`)
		for i := 1; i < len(parts); i++ {
			spanStart := strings.Index(parts[i], "<span>")
			if spanStart >= 0 {
				spanEnd := strings.Index(parts[i][spanStart:], "</span>")
				if spanEnd >= 0 {
					text := strings.TrimSpace(parts[i][spanStart+6 : spanStart+spanEnd])
					if text != "" {
						fmt.Printf("  - %s\n", text)
					}
				}
			}
		}
	case ".btn-primary":
		parts := strings.Split(html, `class="btn btn-primary"`)
		for i := 1; i < len(parts); i++ {
			start := strings.Index(parts[i], ">")
			if start >= 0 {
				end := strings.Index(parts[i][start:], "</")
				if end >= 0 {
					text := extractText(parts[i][start+1 : start+end])
					if text != "" && !strings.Contains(text, "{{") {
						fmt.Printf("  - %s\n", text)
					}
				}
			}
		}
	}
}

func extractText(html string) string {
	// Remove HTML tags and clean up text
	text := html
	// Remove everything between < and >
	for {
		start := strings.Index(text, "<")
		if start == -1 {
			break
		}
		end := strings.Index(text[start:], ">")
		if end == -1 {
			break
		}
		text = text[:start] + " " + text[start+end+1:]
	}
	
	// Clean up whitespace
	text = strings.TrimSpace(text)
	text = strings.ReplaceAll(text, "\n", " ")
	text = strings.ReplaceAll(text, "\t", " ")
	for strings.Contains(text, "  ") {
		text = strings.ReplaceAll(text, "  ", " ")
	}
	
	return text
}