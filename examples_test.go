package enumtag_test

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/tcard/enumtag"
)

type ItemAdded struct {
	ItemID   string `json:"item_id"`
	Quantity int    `json:"quantity"`
}

type ItemRemoved struct {
	ItemID   string `json:"item_id"`
	Quantity int    `json:"quantity"`
}

type Checkout struct{}

type ShoppingCartEvent struct {
	Value    interface{} `enumvaluefield:"value"`
	Variants [0]*struct {
		ItemAdded   `enumtag:"item_added"`
		ItemRemoved `enumtag:"item_removed"`
		Checkout    `enumtag:"checkout"`
	} `enumtagfield:"type"`
}

func (e ShoppingCartEvent) MarshalJSON() ([]byte, error) {
	return enumtag.MarshalJSON(e)
}

func (e *ShoppingCartEvent) UnmarshalJSON(data []byte) error {
	return enumtag.UnmarshalJSON(data, e)
}

func Example_shoppingCartEvent() {
	for _, jsonEvent := range []string{
		`{"type": "item_removed", "value": {"item_id": "foo", "quantity": 2}}`,
		`{"type": "checkout"}`,
		`{"type": "item_added", "value": {"item_id": "xyz", "quantity": 5}}`,
	} {
		var event ShoppingCartEvent
		err := json.Unmarshal([]byte(jsonEvent), &event)
		if err != nil {
			panic(err)
		}

		// Which event did we get? Let's see.
		switch e := event.Value.(type) {
		case ItemAdded:
			fmt.Printf("Got ItemAdded; item ID: %v; quantity: %v\n", e.ItemID, e.Quantity)
		case ItemRemoved:
			fmt.Printf("Got ItemRemoved; item ID: %v; quantity: %v\n", e.ItemID, e.Quantity)
		case Checkout:
			fmt.Println("Got Checkout")
		}
	}

	// Output:
	// Got ItemRemoved; item ID: foo; quantity: 2
	// Got Checkout
	// Got ItemAdded; item ID: xyz; quantity: 5
}

type Number struct {
	Value int `json:"value"`
}

func (e Number) String() string { return strconv.Itoa(e.Value) }

type Add struct {
	Left  Expr `json:"left"`
	Right Expr `json:"right"`
}

func (e Add) String() string { return fmt.Sprintf("(%v + %v)", e.Left, e.Right) }

type Sub struct {
	Left  Expr `json:"left"`
	Right Expr `json:"right"`
}

func (e Sub) String() string { return fmt.Sprintf("(%v - %v)", e.Left, e.Right) }

type Expr struct {
	Value    interface{ String() string }
	Variants [0]*struct {
		Number `enumtag:"number"`
		Add    `enumtag:"add"`
		Sub    `enumtag:"sub"`
	} `enumtagfield:"type"`
}

func (e Expr) String() string { return e.Value.String() }

func (e Expr) MarshalJSON() ([]byte, error) {
	return enumtag.MarshalJSON(e)
}

func (e *Expr) UnmarshalJSON(b []byte) error {
	return enumtag.UnmarshalJSON(b, e)
}

// Say we want to represent simple arithmetic expressions as JSON. An arithmetic
// expression can be a number, or an addition or substraction of two other
// expressions.
//
// In Go, we have Number, Add, and Sub types to represent each case, and a
// Expr enum type that can be any of those.
func Example_expr() {
	var expr Expr
	json.Unmarshal([]byte(`
		{
			"type": "add",
			"left": {
				"type": "number",
				"value": 3
			},
			"right": {
				"type": "sub",
				"left": {
					"type": "number",
					"value": 5
				},
				"right": {
					"type": "number",
					"value": 2
				}
			}
		}
	`), &expr)
	fmt.Println(expr)
	// Output:
	// (3 + (5 - 2))
}
