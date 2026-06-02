package ast

type TypeKind int

const (
	TK_PRIMITIVE TypeKind = iota
	TK_ARRAY
	TK_UNION
	TK_OBJECT
	TK_FUNCTION
)

type TypeAnnotation struct {
	Kind       TypeKind
	Name       string                // "number", "string", "boolean", "null", "undefined", "void", "any", "int", "float"
	ArrayOf    *TypeAnnotation       // for T[]
	Union      []*TypeAnnotation     // for T | U
	Properties map[string]*TypeAnnotation // for { k: T }
	ParamTypes []*TypeAnnotation     // for (a: T, b: U) => V
	ReturnType *TypeAnnotation
	Optional   bool                  // for T? or optional params
}

func (ta *TypeAnnotation) String() string {
	if ta == nil {
		return "any"
	}
	switch ta.Kind {
	case TK_PRIMITIVE:
		return ta.Name
	case TK_ARRAY:
		return ta.ArrayOf.String() + "[]"
	case TK_UNION:
		s := ""
		for i, u := range ta.Union {
			if i > 0 {
				s += " | "
			}
			s += u.String()
		}
		return s
	case TK_OBJECT:
		return "{...}"
	case TK_FUNCTION:
		return "fn"
	}
	return "any"
}
