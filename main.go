package main

import (
	"github.com/sunkaimr/taskcube/cmd"
)

// @title
// @version         v1.0
// @description     taskcube
// @termsOfService  http://swagger.io/terms/
// @schemes http https
// @host localhost:8080
// @BasePath /taskcube/api/v1
func main() {
	cmd.Execute()
}
