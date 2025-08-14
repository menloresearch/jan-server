package mcp

import (
	"reflect"
	"strings"

	mcpgo "github.com/mark3labs/mcp-go/mcp"
)

func ReflectToMCPOptions(description string, v interface{}) []mcpgo.ToolOption {
	t := reflect.TypeOf(v)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	var opts []mcpgo.ToolOption
	opts = append(opts, mcpgo.WithDescription(description))
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)

		// Get JSON field name
		jsonTag := f.Tag.Get("json")
		if jsonTag == "" || jsonTag == "-" {
			continue
		}
		name := strings.Split(jsonTag, ",")[0]

		// Parse jsonschema tag
		jsSchema := f.Tag.Get("jsonschema")
		required := strings.Contains(jsSchema, "required")
		desc := extractDescription(jsSchema)

		// Determine mcpgo arg type based on Go type
		baseType := f.Type
		if baseType.Kind() == reflect.Ptr {
			baseType = baseType.Elem()
		}

		var arg mcpgo.ToolOption
		switch baseType.Kind() {
		case reflect.String:
			if required {
				arg = mcpgo.WithString(name, mcpgo.Required(), mcpgo.Description(desc))
			} else {
				arg = mcpgo.WithString(name, mcpgo.Description(desc))
			}
		case reflect.Int:
			if required {
				arg = mcpgo.WithNumber(name, mcpgo.Required(), mcpgo.Description(desc))
			} else {
				arg = mcpgo.WithNumber(name, mcpgo.Description(desc))
			}
		case reflect.Bool:
			if required {
				arg = mcpgo.WithBoolean(name, mcpgo.Required(), mcpgo.Description(desc))
			} else {
				arg = mcpgo.WithBoolean(name, mcpgo.Description(desc))
			}
		default:
			continue
		}
		opts = append(opts, arg)
	}

	return opts
}

func extractDescription(tag string) string {
	parts := strings.Split(tag, ",")
	for _, p := range parts {
		if strings.HasPrefix(p, "description=") {
			return strings.TrimPrefix(p, "description=")
		}
	}
	return ""
}
