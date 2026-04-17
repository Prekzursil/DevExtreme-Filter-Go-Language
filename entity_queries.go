package main

import (
	"context"
	"fmt"
	"log"
	"strings"

	"entgo.io/ent/dialect/sql"
)

var errUnsupportedEntity = fmt.Errorf("unsupported entity type")

func runEntityQuery(ctx context.Context, entity string, predicate PredicateFunc) (interface{}, error) {
	apply := func(s *sql.Selector) {
		if predicate != nil {
			s.Where(predicate)
		}
	}
	switch strings.ToLower(entity) {
	case "transaction":
		return queryTransactions(ctx, predicate, apply)
	case "test1schema":
		return queryTest1Schema(ctx, predicate, apply)
	case "test2schema":
		return queryTest2Schema(ctx, predicate, apply)
	case "test3schema":
		return queryTest3Schema(ctx, predicate, apply)
	}
	log.Print("Backend: Unsupported entity type for filtering (suppressed)")
	return nil, fmt.Errorf("%w: %s", errUnsupportedEntity, entity)
}

func queryTransactions(ctx context.Context, predicate PredicateFunc, apply func(*sql.Selector)) (interface{}, error) {
	query := client.Transaction.Query()
	if predicate != nil {
		query = query.Where(apply)
	}
	rows, err := query.All(ctx)
	if err != nil {
		return nil, err
	}
	dto := make([]Transaction, len(rows))
	for i, trx := range rows {
		dto[i] = Transaction{
			ID: trx.ID, Date: trx.Date, Amount: trx.Amount, Name: trx.Name,
			Location: trx.Location, Category: trx.Category, Type: trx.Type,
		}
	}
	return dto, nil
}

func queryTest1Schema(ctx context.Context, predicate PredicateFunc, apply func(*sql.Selector)) (interface{}, error) {
	query := client.Test1Schema.Query()
	if predicate != nil {
		query = query.Where(apply)
	}
	return query.All(ctx)
}

func queryTest2Schema(ctx context.Context, predicate PredicateFunc, apply func(*sql.Selector)) (interface{}, error) {
	query := client.Test2Schema.Query()
	if predicate != nil {
		query = query.Where(apply)
	}
	return query.All(ctx)
}

func queryTest3Schema(ctx context.Context, predicate PredicateFunc, apply func(*sql.Selector)) (interface{}, error) {
	query := client.Test3Schema.Query()
	if predicate != nil {
		query = query.Where(apply)
	}
	return query.All(ctx)
}
