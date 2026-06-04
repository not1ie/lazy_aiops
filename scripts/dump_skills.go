//go:build ignore

package main

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/lazyautoops/lazy-auto-ops/plugins/ai"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func main() {
	db, err := gorm.Open(sqlite.Open("/db/lazy-auto-ops.db"), &gorm.Config{})
	if err != nil {
		log.Fatal("DB error:", err)
	}
	
	var list []ai.AISkill
	err = db.Find(&list).Error
	if err != nil {
		log.Fatal("Find error:", err)
	}
	
	fmt.Printf("Found %d skills\n", len(list))
	b, _ := json.MarshalIndent(list, "", "  ")
	fmt.Println(string(b))
}
