// Package enumtag provides a type-directed mechanism to encode arbitrary tagged
// enums as JSON.
//
// In abstract, a tagged enum is a type which can adopt one of a several
// defined shaps, called variants. Each variant has an associated "tag", a name
// that identifies it.
//
// 	enum shoppingCartEvent {
// 		itemAdded { itemID, quantity }
// 		itemRemoved { itemID, quantity }
// 		checkout
// 	}
//
// In JSON, the library encodes tagged enums as objects in which one of the
// fields holds the tag, and another field holds the value of the variant to
// which the tag corresponds.
//
// 	{"type": "item_added", "values": {"item_id": "xyz", "quantity": 2}}
//
// Alternatively, if the variant itself is an object/struct, its fields can be
// embedded in the enum object:
//
// 	{"type": "item_removed", "item_id": "xyz", "quantity": 2}
//
// In Go, the library expects each variant to be a concrete type, and the tagged
// enum is represented with a struct type with this shape:
//
// 	struct {
// 		Value <an interface type> <optional "enumvaluefield" tag>
// 		Variants [0]*struct {
// 			Variant1 <a concrete type> <"enumtag" tag; name of the field if omitted>
// 			Variant2 <a concrete type> <"enumtag" tag; name of the field if omitted>
// 			<an embedded type> <"enumtag" is not optional in this case>
// 			...
// 		} `enumtagfield:"<JSON field which holds the tag>"`
// 	}
//
// The Variants field is a struct (wrapped a zero-length array so as it doesn't
// occupy any memory; it's only needed to hold type information) in which each
// field represents a variant. The type is a concrete type to hold the variant's
// value, and its tag is defined in the "enumtag" struct tag. Variants must
// have an "enumtagfield" with the name of the JSON field in whose value the
// enum value's present variant's tag is set.
//
// The Value field's type must be an interface type. All variant types must
// implement it. When the present variant is identified (by looking at the field
// defined in "enumtagfield"), Value is set to a new value of its corresponding
// concrete type (as mapped in one of the Variants) and its JSON value (either
// stored in the field defined by "enumvaluefield", or as fields embedded in
// the enum's JSON object) is unmarshaled onto it.
//
// Typically, the enum type has MarshalJSON and UnmarshalJSON methods that just
// call this package's corresponding functions.
//
// 	type ShoppingCartEvent struct {
// 		Value interface{} `enumvaluefield:"value"`
// 		Variants [0]*struct {
// 			ItemAdded `enumtag:"item_added"`
// 			ItemRemoved `enumtag:"item_removed"`
// 			Checkout `enumtag:"checkout"`
// 		} `enumtagfield:"type"`
// 	}
//
// 	func (e ShoppingCartEvent) MarshalJSON() ([]byte, error) {
// 		return enumtag.MarshalJSON(e)
// 	}
//
// 	func (e *ShoppingCartEvent) UnmarshalJSON(data []byte) error {
// 		return enumtag.UnmarshalJSON(data, e)
// 	}
//
// (See the full ShoppingCartEvent example in the package documentation.)
//
// Struct tag reference
//
// "enumvaluefield": Defined in the Value field. The field in the enum's JSON
// object representation that holds the variant's value. If omitted, the
// variant's fields are embedded in the enum object itself, alongside the tag
// field.
//
// "enumtagfield": Defined in the Variants field. The field in the enum's JSON
// object representation that holds the variant's tag.
//
// "enumtag": Defined in each variant's struct field. The tag that identifies
// the variant, which is used as value of the field defined by
// "enumtagfield" in the enum's JSON object representation. Can be omitted if
// the field isn't anonymous (ie. isn't an embedded struct or interface), in
// which case the field's name is used.
package enumtag

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
)

// MarshalJSON marshals the given enum value, which must have the shape defined
// in the package's top-level documentation.
func MarshalJSON(v interface{}) ([]byte, error) {
	rv := reflect.Indirect(reflect.ValueOf(v))
	enum, err := reflectEnum(rv)
	if err != nil {
		return nil, errMalformedType(v, err)
	}

	tag, ok := enum.lookupTag()
	if !ok {
		return nil, fmt.Errorf("value type %v doesn't have an associated tag", enum.v.Type())
	}

	rawFields := enum.rawFields()
	variant := enum.v.Elem()
	if enum.valueIsEmbedded() {
		for i := 0; i < variant.NumField(); i++ {
			rawFields = append(rawFields, variant.Type().Field(i))
		}
	}

	raw := reflect.New(reflect.StructOf(rawFields)).Elem()
	raw.Field(0).Set(reflect.ValueOf(tag))
	if enum.valueIsEmbedded() {
		for i := 0; i < variant.NumField(); i++ {
			raw.Field(i + 1).Set(variant.Field(i))
		}
	} else {
		raw.Field(1).Set(enum.v)
	}

	return json.Marshal(raw.Interface())
}

// MarshalJSON unmarshals an enum value from JSON, which must have the shape
// defined in the package's top-level documentation.
func UnmarshalJSON(data []byte, dst interface{}) error {
	rdst := reflect.ValueOf(dst)
	if rdst.Type().Kind() != reflect.Ptr || rdst.IsNil() {
		return errMalformedType(dst, errors.New("tagenum: unmarshal destination must be a non-nil pointer"))
	}
	enum, err := reflectEnum(rdst.Elem())
	if err != nil {
		return errMalformedType(dst, fmt.Errorf("pointee type: %w", err))
	}

	raw := reflect.New(reflect.StructOf(enum.rawFields()))
	err = json.Unmarshal(data, raw.Interface())
	if err != nil {
		return fmt.Errorf("tagenum: unmarshaling enum tag from field %q: %w", enum.tagField, err)
	}

	tag := raw.Elem().Field(0).String()
	variantType, ok := enum.variants[tag]
	if !ok {
		return fmt.Errorf("tagenum: unknown tag %q for enum type %T", tag, dst)
	}

	value := reflect.New(variantType)
	valueData := data
	if !enum.valueIsEmbedded() {
		valueData = raw.Elem().Field(1).Interface().(json.RawMessage)
	}
	if len(valueData) > 0 {
		err = json.Unmarshal(valueData, value.Interface())
		if err != nil {
			return fmt.Errorf("tagenum: unmarshaling enum value into type %v: %w", variantType, err)
		}
	}
	enum.v.Set(value.Elem())

	return nil
}

type enumValue struct {
	v          reflect.Value
	variants   map[string]reflect.Type
	tagField   string
	valueField string
}

func (ev enumValue) rawFields() []reflect.StructField {
	fs := []reflect.StructField{{
		Name: "Tag",
		Type: reflect.TypeOf(""),
		Tag:  reflect.StructTag(fmt.Sprintf(`json:"%s"`, ev.tagField)),
	}}
	if !ev.valueIsEmbedded() {
		fs = append(fs, reflect.StructField{
			Name: "Value",
			Type: reflect.TypeOf(json.RawMessage(nil)),
			Tag:  reflect.StructTag(fmt.Sprintf(`json:"%s"`, ev.valueField)),
		})
	}
	return fs
}

func (ev enumValue) valueIsEmbedded() bool {
	return ev.valueField == "-"
}

func (ev enumValue) lookupTag() (string, bool) {
	for tag, variant := range ev.variants {
		if ev.v.Elem().Type() == variant {
			return tag, true
		}
	}
	return "", false
}

// Validate checks that the passed value has the shape expected by the
// MarshalJSON and UnmarshalJSON functions.
func Validate(v interface{}) error {
	_, err := reflectEnum(reflect.Indirect(reflect.ValueOf(v)))
	return err
}

func reflectEnum(v reflect.Value) (enum enumValue, err error) {
	if v.Kind() != reflect.Struct {
		return enumValue{}, errors.New("isn't a struct")
	}

	variantsField, ok := v.Type().FieldByName("Variants")
	if !ok {
		return enumValue{}, errors.New("struct doesn't have a Variants field")
	}
	enum.tagField, ok = variantsField.Tag.Lookup("enumtagfield")
	if !ok {
		return enumValue{}, errors.New(`Variants field doesn't have a "enumtagfield" tag`)
	}
	if enum.tagField == "-" {
		return enumValue{}, errors.New(`Variants "enumtagfield" value cannot be "-"`)
	}
	if variantsField.Type.Kind() != reflect.Array || variantsField.Type.Len() != 0 {
		return enumValue{}, fmt.Errorf("Variants's type %v isn't a zero-length array", variantsField.Type)
	}
	variants := variantsField.Type.Elem()
	if variants.Kind() != reflect.Ptr || variants.Elem().Kind() != reflect.Struct {
		return enumValue{}, fmt.Errorf("Variants's element type %v isn't a pointer to struct", variants)
	}
	variants = variants.Elem()
	enum.variants = make(map[string]reflect.Type)
	for i := 0; i < variants.NumField(); i++ {
		variant := variants.Field(i)
		tag, ok := variant.Tag.Lookup("enumtag")
		if !ok {
			if variant.Anonymous {
				return enumValue{}, fmt.Errorf(`Variants's %s field is anonymous and doesn't have an "enumtag" tag`, variant.Name)
			}
			tag = variant.Name
		}
		enum.variants[tag] = variant.Type
	}

	vField, ok := v.Type().FieldByName("Value")
	if !ok {
		return enumValue{}, errors.New("struct doesn't have a Value field")
	}
	enum.v = v.FieldByIndex(vField.Index)
	if enum.v.Kind() != reflect.Interface {
		return enumValue{}, fmt.Errorf("Value field's type %v isn't an interface", enum.v.Type())
	}
	for _, variant := range enum.variants {
		if !variant.Implements(enum.v.Type()) {
			return enumValue{}, fmt.Errorf("variant type %v can't be set to value of type %v", variant, enum.v.Type())
		}
	}
	enum.valueField, ok = vField.Tag.Lookup("enumvaluefield")
	if !ok {
		enum.valueField = "-"
	}

	return enum, nil
}

func errMalformedType(v interface{}, err error) error {
	return fmt.Errorf("enumtag: malformed enum type %T: %w", v, err)
}
