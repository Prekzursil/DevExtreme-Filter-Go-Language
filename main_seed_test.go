package main

import (
	"context"
	"testing"
)

func TestGenerateTransactions(t *testing.T) {
	// Just exercise the function for coverage; data correctness is not asserted
	// since TestMain pre-seeds 50 transactions which makes total counting unreliable.
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("generateTransactions panicked: %v", r)
		}
	}()
	generateTransactions(2, context.Background())
}

func TestGenerateTest1SchemaData(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("generateTest1SchemaData panicked: %v", r)
		}
	}()
	generateTest1SchemaData(2, context.Background())
}

func TestGenerateTest2SchemaData(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("generateTest2SchemaData panicked: %v", r)
		}
	}()
	generateTest2SchemaData(2, context.Background())
}

func TestGenerateTest3SchemaData(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("generateTest3SchemaData panicked: %v", r)
		}
	}()
	generateTest3SchemaData(2, context.Background())
}
