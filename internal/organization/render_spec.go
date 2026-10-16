package organization

import "github.com/Glacius-Labs/Markitect/internal/model"

// Explicit tagged values preserve arbitrary-precision integers/decimal tuples,
// reference identities, enum distinctions and list order. This is a value view,
// not the original YAML spelling or a replacement canonical-model format.
func renderSpec(fields []model.Field) (map[string]any, error) {
	result := make(map[string]any, len(fields))
	for _, field := range fields {
		if _, exists := result[field.Name]; exists {
			return nil, renderInvalid("duplicate rendered spec field")
		}
		value, err := renderValue(field.Value)
		if err != nil {
			return nil, err
		}
		result[field.Name] = value
	}
	return result, nil
}

func renderValue(value model.Value) (any, error) {
	switch value.Kind() {
	case model.StringValue, model.EnumValue:
		text, _ := value.Text()
		return []any{value.Kind(), text}, nil
	case model.BooleanValue:
		boolean, _ := value.Boolean()
		return []any{value.Kind(), boolean}, nil
	case model.IntegerValue:
		integer, _ := value.Integer()
		return []any{value.Kind(), integer}, nil
	case model.NumberValue:
		number, _ := value.Number()
		return []any{value.Kind(), number.Coefficient, number.Exponent}, nil
	case model.ReferenceValue:
		id, _ := value.Reference()
		return []any{value.Kind(), id.Package, id.ID}, nil
	case model.ObjectValue:
		fields, _ := value.Fields()
		object, err := renderSpec(fields)
		return []any{value.Kind(), object}, err
	case model.ListValue:
		items, _ := value.Items()
		values := make([]any, len(items))
		for i, item := range items {
			value, err := renderValue(item)
			if err != nil {
				return nil, err
			}
			values[i] = value
		}
		return []any{value.Kind(), values}, nil
	default:
		return nil, renderInvalid("unsupported compiled spec value")
	}
}
