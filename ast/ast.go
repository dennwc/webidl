package ast

type Node interface {
	NodeBase() *Base
}

type Base struct {
	Start    int          `json:"start"` // rune
	End      int          `json:"end"`   // rune
	Comments []string     `json:"comments,omitempty"`
	Errors   []*ErrorNode `json:"errors,omitempty"`
}

func (b *Base) NodeBase() *Base {
	return b
}

// error occurred; value is text of error
type ErrorNode struct {
	Base
	Message string `json:"message"`
}

type Decl interface {
	Node
	isGenDecl()
}

// The file root node
type File struct {
	Base
	Declarations []Decl `json:"declarations,omitempty"`
}

// interface Foo { ... }
type Declaration struct {
	Base
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
	Base
	Name        string        `json:"name"`
	Annotations []*Annotation `json:"annotations,omitempty"`
	Members     []*Member     `json:"members,omitempty"`
}

func (Dictionary) isGenDecl() {}

// [Constructor], []
type Annotation struct {
	Base
	Name       string       `json:"name"`
	Value      string       `json:"value,omitempty"`      // [A=B]
	Parameters []*Parameter `json:"parameters,omitempty"` // [A(X x, Y y)]
	Values     []string     `json:"values,omitempty"`     // [A=(a,b,c)]
}

// optional any SomeArg
type Parameter struct {
	Base
	Type     Type   `json:"type"`
	Optional bool   `json:"optional,omitempty"`
	Variadic bool   `json:"variadic,omitempty"`
	Name     string `json:"name"`
	Init     string `json:"init,omitempty"`
}

// Window implements ECMA262Globals
type Implementation struct {
	Base
	Name   string `json:"name"`
	Source string `json:"source"`
}

func (Implementation) isGenDecl() {}

// Document includes DocumentOrShadowRoot
type Includes struct {
	Base
	Name   string `json:"name"`
	Source string `json:"source"`
}

func (Includes) isGenDecl() {}

// readonly attribute something
type Member struct {
	Base
	Name           string        `json:"name,omitempty"`
	Type           Type          `json:"type,omitempty"`
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
	Base
	Name string `json:"name"`
}

type TypeName struct {
	Base
	Name string
}

func (*TypeName) isType() {}

type Iterable struct {
	Base
	Type Type `json:"type"`
}

type Callback struct {
	Base
	Name       string       `json:"name"`
	Return     Type         `json:"type,omitempty"`
	Parameters []*Parameter `json:"parameters,omitempty"`
}

func (Callback) isGenDecl() {}

type Type interface {
	Node
	isType()
}

type AnyType struct {
	Base
}

func (*AnyType) isType() {}

type SequenceType struct {
	Base
	Elem Type
}

func (*SequenceType) isType() {}

type UnionType struct {
	Base
	Types []Type
}

func (*UnionType) isType() {}

type NullableType struct {
	Base
	Type Type
}

func (*NullableType) isType() {}
