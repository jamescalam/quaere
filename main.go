// Command quaere is a terminal SQL client with a Beekeeper-style layout: a
// left sidebar of saved connections and their tables, a query editor across the
// top of the main area, and a results grid below. It speaks PostgreSQL, either
// via a direct connection string or a Google Cloud SQL instance authenticated
// with IAM (Application Default Credentials).
package main

import (
	"github.com/aurelio-labs/quaere/internal/cmd"
)

func main() {
	cmd.Execute()
}
