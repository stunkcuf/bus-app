package main

import (
	"database/sql"
	"fmt"
	"log"
	
	_ "github.com/lib/pq"
)

func main() {
	dbURL := "postgresql://postgres:jTKOWGEzlprRGbkPBCgGxsgnwyLeGoDL@shortline.proxy.rlwy.net:40148/railway"
	
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	fmt.Println("=== CHECKING ROUTES ===\n")

	// Check routes table columns
	fmt.Println("1. Routes table structure:")
	rows, err := db.Query(`
		SELECT column_name, data_type, is_nullable
		FROM information_schema.columns
		WHERE table_name = 'routes'
		ORDER BY ordinal_position
	`)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		defer rows.Close()
		for rows.Next() {
			var col, dtype, nullable string
			rows.Scan(&col, &dtype, &nullable)
			fmt.Printf("   %s (%s) nullable=%s\n", col, dtype, nullable)
		}
	}

	fmt.Println("\n2. All routes in database:")
	routeRows, err := db.Query(`
		SELECT * FROM routes
	`)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		defer routeRows.Close()
		cols, _ := routeRows.Columns()
		fmt.Printf("   Columns: %v\n", cols)
		
		count := 0
		// Create a slice to hold the values
		values := make([]interface{}, len(cols))
		valuePtrs := make([]interface{}, len(cols))
		for i := range values {
			valuePtrs[i] = &values[i]
		}
		
		for routeRows.Next() {
			count++
			err := routeRows.Scan(valuePtrs...)
			if err != nil {
				fmt.Printf("   Error scanning: %v\n", err)
				continue
			}
			
			fmt.Printf("\n   Route %d:\n", count)
			for i, col := range cols {
				val := values[i]
				if val == nil {
					fmt.Printf("     %s: NULL\n", col)
				} else {
					switch v := val.(type) {
					case []byte:
						fmt.Printf("     %s: %s\n", col, string(v))
					default:
						fmt.Printf("     %s: %v\n", col, v)
					}
				}
			}
		}
		
		if count == 0 {
			fmt.Println("   No routes found!")
		} else {
			fmt.Printf("\n   Total: %d routes\n", count)
		}
	}

	// Check what the query in the handler is looking for
	fmt.Println("\n3. Routes with status='active':")
	activeRows, err := db.Query(`
		SELECT id, name, description 
		FROM routes 
		WHERE status = 'active'
		ORDER BY name
	`)
	if err != nil {
		fmt.Printf("   Error: %v\n", err)
		
		// Try without status filter
		fmt.Println("\n4. Routes without status filter:")
		allRows, err2 := db.Query(`
			SELECT route_id, route_name, description 
			FROM routes 
			ORDER BY route_name
		`)
		if err2 != nil {
			fmt.Printf("   Error: %v\n", err2)
		} else {
			defer allRows.Close()
			count := 0
			for allRows.Next() {
				var id, name string
				var desc sql.NullString
				allRows.Scan(&id, &name, &desc)
				count++
				fmt.Printf("   [%d] ID: %s, Name: %s\n", count, id, name)
				if desc.Valid {
					fmt.Printf("       Description: %s\n", desc.String)
				}
			}
			if count == 0 {
				fmt.Println("   No routes found!")
			}
		}
	} else {
		defer activeRows.Close()
		count := 0
		for activeRows.Next() {
			var id, name string
			var desc sql.NullString
			activeRows.Scan(&id, &name, &desc)
			count++
			fmt.Printf("   [%d] ID: %s, Name: %s\n", count, id, name)
			if desc.Valid {
				fmt.Printf("       Description: %s\n", desc.String)
			}
		}
		if count == 0 {
			fmt.Println("   No active routes found!")
		}
	}
}