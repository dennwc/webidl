package ast

type Node interface {
	Base() *BaseNode
}

type BaseNode struct {
	Start    int          `json:"start"` // rune
	End      int          `json:"end"`   // rune
	Comments []string     `json:"comments,omitempty"`
	Errors   []*ErrorNode `json:"errors,omitempty"`
}

func (b *BaseNode) Base() *BaseNode {
	return b
}

// error occurred; value is text of error
type ErrorNode struct {
	BaseNode
	Message string `json:"message"`
}

type GenDecl interface {
	Node
	isGenDecl()
}

// The file root node
type File struct {
	BaseNode
	Declarations []GenDecl `json:"declarations,omitempty"`
}

// interface Foo { ... }
type Declaration struct {
	BaseNode
	Kind        string        `json:"kind"`
	Name        string        `json:"name"`
	ParentType  string        `json:"parent_type,omitempty"`
	Mixin       bool          `json:"mixin,omitempty"`
	Annotations []*Annotation `json:"annotations,omitempty"`
	Members     []*Member     `json:"members,omitempty"`
	CustomOps   []*CustomOp   `json:"custom_ops,omitempty"`
	Iterable    *Iterable     `json:"iterable,omitempty"`
}

func (Declaration) isGenDecl() {}

type Dictionary struct {
	BaseNode
	Name        string        `json:"name"`
	Annotations []*Annotation `json:"annotations,omitempty"`
	Members     []*Member     `json:"members,omitempty"`
}

func (Dictionary) isGenDecl() {}

// [Constructor], []
type Annotation struct {
	BaseNode
	Name       string       `json:"name"`
	Value      string       `json:"value,omitempty"`      // [A=B]
	Parameters []*Parameter `json:"parameters,omitempty"` // [A(X x, Y y)]
	Values     []string     `json:"values,omitempty"`     // [A=(a,b,c)]
}

// optional any SomeArg
type Parameter struct {
	BaseNode
	Type     *Type  `json:"type"`
	Optional bool   `json:"optional,omitempty"`
	Variadic bool   `json:"variadic,omitempty"`
	Name     string `json:"name"`
	Init     string `json:"init,omitempty"`
}

// Window implements ECMA262Globals
type Implementation struct {
	BaseNode
	Name   string `json:"name"`
	Source string `json:"source"`
}

func (Implementation) isGenDecl() {}

// Document includes DocumentOrShadowRoot
type Includes struct {
	BaseNode
	Name   string `json:"name"`
	Source string `json:"source"`
}

func (Includes) isGenDecl() {}

// readonly attribute something
type Member struct {
	BaseNode
	Name           string        `json:"name,omitempty"`
	Type           *Type         `json:"type,omitempty"`
	Init           string        `json:"init,omitempty"`
	Attribute      bool          `json:"attribute,omitempty"`
	Static         bool          `json:"static,omitempty"`
	Const          bool          `json:"const,omitempty"`
	Readonly       bool          `json:"readonly,omitempty"`
	Specialization string        `json:"specialization,omitempty"`
	Parameters     []*Parameter  `json:"parameters,omitempty"`
	Annotations    []*Annotation `json:"annotations,omitempty"`
}

type CustomOp struct {
	BaseNode
	Name string `json:"name"`
}

type Type struct {
	BaseNode
	Name       string  `json:"name,omitempty"`
	Any        bool    `json:"any,omitempty"`
	Nullable   bool    `json:"nullable,omitempty"`
	SequenceOf *Type   `json:"sequence_of,omitempty"`
	UnionOf    []*Type `json:"union_of,omitempty"`
}

type Iterable struct {
	BaseNode
	Type *Type `json:"type"`
}

type Callback struct {
	BaseNode
	Name       string       `json:"name"`
	Type       *Type        `json:"type,omitempty"`
	Parameters []*Parameter `json:"parameters,omitempty"`
}

func (Callback) isGenDecl() {}
