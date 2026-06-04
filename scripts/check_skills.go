//go:build ignore

package main

import (
	"fmt"
	"log"

	"github.com/lazyautoops/lazy-auto-ops/plugins/ai"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func main() {
	db, err := gorm.Open(sqlite.Open("/db/lazy-auto-ops.db"), &gorm.Config{})
	if err != nil {
		log.Fatal(err)
	}
	var list []ai.AISkill
	if err := db.Find(&list).Error; err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Found %d skills\n", len(list))
	for _, s := range list {
		fmt.Printf("- %s (is_system: %v)\n", s.Name, s.IsSystem)
	}
}
