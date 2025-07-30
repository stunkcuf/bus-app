package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"strings"

	_ "github.com/lib/pq"
	"github.com/joho/godotenv"
)

func main() {
	fmt.Println("🔍 CHECKING SERVICE RECORDS DATA (SIMPLE)")
	fmt.Println("=" + strings.Repeat("=", 60))

	// Load environment
	godotenv.Load("../.env")

	// Connect to database
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL not set")
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer db.Close()

	// Count records
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM service_records").Scan(&count)
	if err != nil {
		log.Fatal("Failed to count records:", err)
	}
	
	fmt.Printf("\n📈 Total Records: %d\n", count)

	// Sample some records with all fields
	fmt.Println("\n📝 Sample Records (first 10):")
	rows, err := db.Query(`
		SELECT id, 
		       COALESCE(unnamed_0, '') as f0,
		       COALESCE(unnamed_1, '') as f1, 
		       COALESCE(unnamed_2, '') as f2,
		       COALESCE(unnamed_3, '') as f3,
		       COALESCE(unnamed_4, '') as f4,
		       COALESCE(unnamed_5, '') as f5,
		       COALESCE(unnamed_6, '') as f6
		FROM service_records 
		LIMIT 10
	`)
	if err != nil {
		log.Fatal("Failed to query records:", err)
	}
	defer rows.Close()

	recordNum := 0
	for rows.Next() {
		var id int
		var f0, f1, f2, f3, f4, f5, f6 string
		err = rows.Scan(&id, &f0, &f1, &f2, &f3, &f4, &f5, &f6)
		if err != nil {
			continue
		}
		
		recordNum++
		fmt.Printf("\n  Record #%d (ID: %d):\n", recordNum, id)
		
		fields := []string{f0, f1, f2, f3, f4, f5, f6}
		hasData := false
		for i, field := range fields {
			if field != "" && field != " " {
				// Truncate long fields
				display := field
				if len(display) > 50 {
					display = display[:50] + "..."
				}
				fmt.Printf("    Field %d: %s\n", i, display)
				hasData = true
			}
		}
		
		if !hasData {
			fmt.Println("    (Empty record)")
		}
	}

	// Check for records with any data
	var recordsWithData int
	err = db.QueryRow(`
		SELECT COUNT(*) FROM service_records 
		WHERE unnamed_0 IS NOT NULL AND unnamed_0 != ''
		   OR unnamed_1 IS NOT NULL AND unnamed_1 != ''
		   OR unnamed_2 IS NOT NULL AND unnamed_2 != ''
		   OR unnamed_3 IS NOT NULL AND unnamed_3 != ''
		   OR unnamed_4 IS NOT NULL AND unnamed_4 != ''
	`).Scan(&recordsWithData)
	
	if err == nil {
		fmt.Printf("\n✅ Records with data: %d (%.1f%%)\n", recordsWithData, float64(recordsWithData)/float64(count)*100)
	}

	// Try to identify what each column might contain
	fmt.Println("\n🔬 Analyzing column patterns...")
	for i := 0; i <= 6; i++ {
		var sample string
		query := fmt.Sprintf(`
			SELECT unnamed_%d FROM service_records 
			WHERE unnamed_%d IS NOT NULL AND unnamed_%d != ''
			LIMIT 1
		`, i, i, i)
		
		err = db.QueryRow(query).Scan(&sample)
		if err == nil && sample != "" {
			if len(sample) > 50 {
				sample = sample[:50] + "..."
			}
			fmt.Printf("  Column %d sample: %s\n", i, sample)
		}
	}
}