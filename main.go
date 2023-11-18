package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/go-sql-driver/mysql"
	"golang.org/x/crypto/ssh/terminal"
	"github.com/pmezard/go-difflib/difflib"
)

func main() {
	// Prompt for MySQL connection details
	host := GetUserInput("Enter MySQL host (default: 127.0.0.1): ", "127.0.0.1")
	port := GetUserInput("Enter MySQL port (default: 3306): ", "3306")
	user := GetUserInput("Enter MySQL username (default: pterodactyl): ", "pterodactyl")

	// Prompt for MySQL password
	password := GetPasswordInput("Enter MySQL password: ")

	// Prompt for Pterodactyl database name
	pterodactylDBName := GetUserInput("Enter Pterodactyl database name (default: panel): ", "panel")

	// Build the MySQL DSN (Data Source Name)
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s", user, password, host, port, pterodactylDBName)

	// Connect to MySQL
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Fatal("Error connecting to MySQL:", err)
	}
	defer db.Close()

	// Check if the connection is successful
	err = db.Ping()
	if err != nil {
		log.Fatal("Error pinging MySQL server:", err)
	}

	fmt.Println("Connected to MySQL server successfully!")

	// Prompt for egg ID
	eggID := GetUserInput("Enter egg ID: ", "")

	// Query the startup field for the specified egg ID from the eggs table
	eggQuery := "SELECT startup FROM eggs WHERE id = ?"

	var eggStartup string
	err = db.QueryRow(eggQuery, eggID).Scan(&eggStartup)
	if err != nil {
		log.Fatal("Error retrieving startup value from eggs table:", err)
	}

	fmt.Printf("\nStartup value for Egg ID %s (from eggs table): %s\n", eggID, eggStartup)

	// Query servers with the specified egg ID
	serverQuery := "SELECT uuidShort, name, startup FROM servers WHERE egg_id = ?"

	rows, err := db.Query(serverQuery, eggID)
	if err != nil {
		log.Fatal("Error querying servers:", err)
	}
	defer rows.Close()

	fmt.Printf("Servers using Egg ID %s:\n\n", eggID)

	for rows.Next() {
		var uuidShort, name, serverStartup string
		if err := rows.Scan(&uuidShort, &name, &serverStartup); err != nil {
			log.Fatal("Error scanning row:", err)
		}

		// Compare startup value from eggs table with startup value from servers table
		if eggStartup == serverStartup {
			fmt.Printf("UUID Short: %s, Name: %s, Startup Matches\n", uuidShort, name)
		} else {
			fmt.Printf("UUID Short: %s, Name: %s, Startup Mismatch\n", uuidShort, name)

			// Show the difference between eggStartup and serverStartup
			diff := difflib.UnifiedDiff{
				A:        difflib.SplitLines(eggStartup),
				B:        difflib.SplitLines(serverStartup),
				FromFile: "Egg Startup",
				ToFile:   "Server Startup",
				Context:  3,
			}

			diffText, _ := difflib.GetUnifiedDiffString(diff)
			fmt.Println("Difference:\n", diffText)

			// Prompt the user to update the startup
			fmt.Print("Do you want to update the startup for this server? (y/n): ")
			var updateChoice string
			fmt.Scanln(&updateChoice)

			if updateChoice == "y" || updateChoice == "Y" {
				// Update the startup for the server with the latest one from the egg
				updateQuery := "UPDATE servers SET startup = ? WHERE uuidShort = ?"
				_, err := db.Exec(updateQuery, eggStartup, uuidShort)
				if err != nil {
					log.Fatal("Error updating startup for server:", err)
				}
				fmt.Printf("Startup updated for server with UUID Short %s\n", uuidShort)
			}
		}
	}
}


// GetUserInput prompts the user for input with a given prompt and provides a default value.
func GetUserInput(prompt, defaultValue string) string {
	fmt.Print(prompt)

	// Set default value
	if defaultValue != "" {
		fmt.Printf("(default: %s) ", defaultValue)
	}

	// Read user input
	var input string
	fmt.Scanln(&input)

	// Use default value if input is empty
	if input == "" {
		return defaultValue
	}

	return input
}

// GetPasswordInput prompts the user for a password without showing it on the screen.
func GetPasswordInput(prompt string) string {
	fmt.Print(prompt)

	// Read password without showing it
	password, err := terminal.ReadPassword(int(os.Stdin.Fd()))
	if err != nil {
		log.Fatal("Error reading password:", err)
	}

	// Print a new line after reading password
	fmt.Println()

	return string(password)
}