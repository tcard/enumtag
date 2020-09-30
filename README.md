# enumtag [![Build Status](https://secure.travis-ci.org/tcard/enumtag.svg?branch=master)](http://travis-ci.org/tcard/enumtag) [![GoDoc](https://godoc.org/github.com/tcard/enumtag?status.svg)](https://godoc.org/github.com/tcard/enumtag)

Package enumtag provides a type-directed mechanism to encode arbitrary tagged enums as JSON.

[Check out the package documentation for details.](https://godoc.org/github.com/tcard/enumtag)

```go
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
```