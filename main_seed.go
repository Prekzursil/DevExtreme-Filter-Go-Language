// Test-data generators for the in-memory ent client. Split out of main.go
// so each file's qlty "high total complexity" stays below the smell
// threshold; the seed routines are pure ent.Create() calls and contribute
// most of main.go's per-function complexity.
package main

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"
)

func generateTransactions(count int, ctx context.Context) {
	locations := []string{"New York", "Los Angeles", "Chicago", "Houston", "Phoenix", "Philadelphia"}
	categories := []string{"Groceries", "Dining", "Food & Drink", "Income", "Shopping", "Bills", "Transportation", "Entertainment", "Housing", "Health"}
	types := []string{"Debit", "Credit"}
	for i := 0; i < count; i++ {
		baseDay := time.Now().AddDate(0, 0, -i)
		transactionDate := time.Date(baseDay.Year(), baseDay.Month(), baseDay.Day(), i%24, (i*13)%60, (i*7)%60, 0, time.UTC)
		client.Transaction.Create().
			SetAmount(float64((i+1)*10) + float64(i%10)*0.5).
			SetDate(transactionDate).
			SetName(fmt.Sprintf("Transaction %d", i+1)).
			SetLocation(locations[i%len(locations)]).
			SetCategory(categories[i%len(categories)]).
			SetType(types[i%len(types)]).
			SaveX(ctx)
	}
	log.Printf("Generated %d transactions", count)
}

func generateTest1SchemaData(count int, ctx context.Context) {
	for i := 0; i < count; i++ {
		client.Test1Schema.Create().
			SetFieldString(fmt.Sprintf("T1 String %d", i)).
			SetFieldInt(i * 100).
			SetFieldFloat(float64(i*10) + 0.55).
			SetFieldBool(i%2 == 0).
			SetFieldTime(time.Now().AddDate(0, -(i % 12), -(i % 28))).
			SetFieldText(fmt.Sprintf("This is some longer text for Test1Schema item #%d. It can contain multiple sentences.", i)).
			SaveX(ctx)
	}
	log.Printf("Generated %d Test1Schema records", count)
}

func generateTest2SchemaData(count int, ctx context.Context) {
	itemTypes := []string{"Gadget", "Widget", "Accessory", "Component", "Tool"}
	for i := 0; i < count; i++ {
		client.Test2Schema.Create().
			SetName(fmt.Sprintf("Item %c%d", 'A'+(i%26), i)).
			SetDescription(fmt.Sprintf("Detailed description of Item %c%d. Quality assured.", 'A'+(i%26), i)).
			SetQuantity(10 + (i * 3 % 50)).
			SetPrice(float64(20+(i*7%100)) + float64(i%100)/100.0).
			SetActive((i+1)%3 != 0).
			SetCreatedAt(time.Now().AddDate(0, 0, -(i * 2))).
			SetUpdatedAt(time.Now().AddDate(0, 0, -i)).
			SetItemType(itemTypes[i%len(itemTypes)]).
			SaveX(ctx)
	}
	log.Printf("Generated %d Test2Schema records", count)
}

func generateTest3SchemaData(count int, ctx context.Context) {
	tagOptions := [][]string{
		{"tech", "new", "featured"}, {"books", "classic"}, {"apparel", "sale", "cotton"},
		{"home", "decor"}, {"sports", "outdoor", "gear"},
	}
	for i := 0; i < count; i++ {
		client.Test3Schema.Create().
			SetSku(fmt.Sprintf("SKU-%04d-%c", i, 'A'+(i%26))).
			SetProductName(fmt.Sprintf("Complex Product %d", i)).
			SetShortDescription(fmt.Sprintf("Brief overview of CP%d.", i)).
			SetFullDescription(fmt.Sprintf("Extended narrative for Complex Product %d, detailing its features, benefits, and specifications. Built for performance and durability.", i)).
			SetCostPrice(float64(50+(i*12%200)) + float64(i%100)/100.0).
			SetRetailPrice(float64(100+(i*18%300)) + float64(i%100)/100.0).
			SetStockCount(50 + (i * 5 % 150)).
			SetIsActive((i)%5 != 0).
			SetPublishedAt(time.Now().AddDate(0, 0, -(i*3 + 5))).
			SetLastOrderedAt(time.Now().AddDate(0, 0, -(i*5 + 2))).
			SetTags(strings.Join(tagOptions[i%len(tagOptions)], ", ")).
			SaveX(ctx)
	}
	log.Printf("Generated %d Test3Schema records", count)
}
