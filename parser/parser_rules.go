// Copyright 2015 The Serulian Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
package parser

import (
	"github.com/dennwc/webidl/ast"
)

// Parse parses the given WebIDL source into a parse tree.
func Parse(input string) *ast.File {
	lexer := lex(input)

	config := parserConfig{
		ignoredTokenTypes: map[tokenType]struct{}{
			tokenTypeWhitespace: {},
			tokenTypeComment:    {},
		},
	}

	parser := buildParser(lexer, config, bytePosition(0))
	return parser.consumeTopLevel()
}

// consumeTopLevel attempts to consume the top-level constructs of a WebIDL file.
func (p *sourceParser) consumeTopLevel() *ast.File {
	n := &ast.File{}
	defer p.node(n)()

	// Start at the first token.
	p.consumeToken()

	if p.currentToken.kind == tokenTypeError {
		p.emitError("%s", p.currentToken.value)
		return n
	}

Loop:
	for !p.isToken(tokenTypeEOF) {
		switch {
		case p.isToken(tokenTypeLeftBracket) || p.isIdentifier("interface") ||
			p.isIdentifier("partial") || p.isIdentifier("callback") ||
			p.isIdentifier("dictionary") || p.isIdentifier("enum"):
			n.Declarations = append(n.Declarations, p.consumeDeclaration())
			continue
		case p.isToken(tokenTypeIdentifier):
			name := p.consumeIdentifier()
			if p.tryConsumeKeyword("implements") {
				impl := &ast.Implementation{Name: name}
				impl.Source = p.consumeIdentifier()
				n.Declarations = append(n.Declarations, impl)
				p.consume(tokenTypeSemicolon)
				continue
			} else if p.tryConsumeKeyword("includes") {
				impl := &ast.Includes{Name: name}
				impl.Source = p.consumeIdentifier()
				n.Declarations = append(n.Declarations, impl)
				p.consume(tokenTypeSemicolon)
				continue
			}
		}
		p.emitError("Unexpected token at root level: %v", p.currentToken)
		break Loop
	}

	return n
}

func (p *sourceParser) consumeInterfaceOrMixin(ann []*ast.Annotation, base *ast.Base, finish func()) ast.Decl {
	partial := p.tryConsumeKeyword("partial")
	p.consumeKeyword("interface")
	if p.tryConsumeKeyword("mixin") {
		return p.consumeMixin(ann, base, finish)
	}
	return p.consumeInterface(partial, false, ann, base, finish)
}
func (p *sourceParser) consumeInterface(partial, callback bool, ann []*ast.Annotation, base *ast.Base, finish func()) *ast.Interface {
	n := &ast.Interface{Annotations: ann, Partial: partial, Callback: callback}
	defer func() {
		finish()
		n.Base = *base
	}()

	n.Name = p.consumeIdentifier()

	if _, ok := p.tryConsume(tokenTypeColon); ok {
		n.Inherits = p.consumeIdentifier()
	}

	// {
	p.consume(tokenTypeLeftBrace)

loop:
	for {
		if p.isToken(tokenTypeRightBrace) {
			break
		}

		if p.isIdentifier("serializer") ||
			p.isIdentifier("jsonifier") ||
			p.isIdentifier("stringifier") {

			op := &ast.CustomOp{}
			finish := p.node(op)
			op.Name = p.consumeIdentifier()
			_, ok := p.consume(tokenTypeSemicolon)
			finish()

			n.CustomOps = append(n.CustomOps, op)

			if !ok {
				break loop
			}

			continue
		} else if p.isIdentifier("iterable") {
			p.consume(tokenTypeIdentifier)
			iter := &ast.Iterable{}
			finish := p.node(iter)
			p.consume(tokenTypeLeftTri)
			iter.Type = p.consumeType()
			p.consume(tokenTypeRightTri)
			finish()
			n.Iterable = iter
			_, ok := p.consume(tokenTypeSemicolon)
			if !ok {
				break loop
			}

			continue
		}
		n.Members = append(n.Members, p.consumeInterfaceMember())

		if _, ok := p.consume(tokenTypeSemicolon); !ok {
			p.emitError("Expected semicolon, got: %v", p.currentToken)
			break
		}
	}

	// };
	p.consume(tokenTypeRightBrace)
	p.consume(tokenTypeSemicolon)

	return n
}

func (p *sourceParser) consumeMixin(ann []*ast.Annotation, base *ast.Base, finish func()) *ast.Mixin {
	n := &ast.Mixin{Annotations: ann}
	defer func() {
		finish()
		n.Base = *base
	}()

	n.Name = p.consumeIdentifier()

	if _, ok := p.tryConsume(tokenTypeColon); ok {
		n.Inherits = p.consumeIdentifier()
	}

	// {
	p.consume(tokenTypeLeftBrace)

loop:
	for {
		if p.isToken(tokenTypeRightBrace) {
			break
		}

		if p.isIdentifier("serializer") || p.isIdentifier("jsonifier") {
			customOpNode := &ast.CustomOp{}
			finish := p.node(customOpNode)
			customOpNode.Name = p.currentToken.value

			p.consume(tokenTypeIdentifier)
			_, ok := p.consume(tokenTypeSemicolon)
			finish()

			n.CustomOps = append(n.CustomOps, customOpNode)

			if !ok {
				break loop
			}

			continue
		} else if p.isIdentifier("iterable") {
			p.consume(tokenTypeIdentifier)
			iter := &ast.Iterable{}
			finish := p.node(iter)
			p.consume(tokenTypeLeftTri)
			iter.Type = p.consumeType()
			p.consume(tokenTypeRightTri)
			finish()
			n.Iterable = iter
			_, ok := p.consume(tokenTypeSemicolon)
			if !ok {
				break loop
			}

			continue
		}
		n.Members = append(n.Members, p.consumeMixinMember())

		if _, ok := p.consume(tokenTypeSemicolon); !ok {
			p.emitError("Expected semicolon, got: %v", p.currentToken)
			break
		}
	}

	// };
	p.consume(tokenTypeRightBrace)
	p.consume(tokenTypeSemicolon)

	return n
}

func (p *sourceParser) consumeDictionary(ann []*ast.Annotation, base *ast.Base, finish func()) *ast.Dictionary {
	n := &ast.Dictionary{Annotations: ann}
	defer func() {
		finish()
		n.Base = *base
	}()
	p.consumeKeyword("dictionary")

	n.Name = p.consumeIdentifier()
	if _, ok := p.tryConsume(tokenTypeColon); ok {
		n.Inherits = p.consumeIdentifier()
	}

	// {
	p.consume(tokenTypeLeftBrace)
	for !p.isToken(tokenTypeRightBrace) {
		n.Members = append(n.Members, p.consumeMember(true))

		if _, ok := p.consume(tokenTypeSemicolon); !ok {
			p.emitError("Expected semicolon, got: %v", p.currentToken)
			break
		}
	}

	// };
	p.consume(tokenTypeRightBrace)
	p.consume(tokenTypeSemicolon)
	return n
}

func (p *sourceParser) consumeEnum(ann []*ast.Annotation, base *ast.Base, finish func()) *ast.Enum {
	n := &ast.Enum{Annotations: ann}
	defer func() {
		finish()
		n.Base = *base
	}()
	p.consumeKeyword("enum")
	n.Name = p.consumeIdentifier()

	// {
	p.consume(tokenTypeLeftBrace)
	for !p.isToken(tokenTypeRightBrace) {
		if len(n.Values) != 0 {
			if _, ok := p.tryConsume(tokenTypeComma); !ok {
				break
			}
		}
		n.Values = append(n.Values, p.consumeLiteral())
	}
	// };
	p.consume(tokenTypeRightBrace)
	p.consume(tokenTypeSemicolon)
	return n
}

// consumeDeclaration attempts to consume a declaration, with optional attributes.
func (p *sourceParser) consumeDeclaration() ast.Decl {
	base := &ast.Base{}
	finish := p.node(base)
	ann := p.tryConsumeAnnotations()
	switch {
	case p.isIdentifier("enum"):
		return p.consumeEnum(ann, base, finish)
	case p.isIdentifier("callback"):
		_ = p.consumeIdentifier()
		if p.tryConsumeKeyword("interface") {
			return p.consumeInterface(false, true, ann, base, finish)
		}
		name := p.consumeIdentifier()
		p.consume(tokenTypeEquals)
		ret := p.consumeType()
		par := p.consumeParameters()
		p.consume(tokenTypeSemicolon)
		finish()
		return &ast.Callback{Base: *base, Name: name, Return: ret, Parameters: par}
	case p.isIdentifier("interface") || p.isIdentifier("partial"):
		return p.consumeInterfaceOrMixin(ann, base, finish)
	case p.isIdentifier("dictionary"):
		return p.consumeDictionary(ann, base, finish)
	default:
		p.emitError("Expected interface or dictionary, got: %v", p.currentToken)
		// first, consume until '{'
		for !p.isToken(tokenTypeLeftBrace, tokenTypeEOF) {
			p.consumeToken()
		}
		// then consume until '}'
		for !p.isToken(tokenTypeRightBrace, tokenTypeEOF) {
			p.consumeToken()
		}
		p.consume(tokenTypeSemicolon)
		finish()
		return &ast.Interface{Base: *base}
	}
}

func (p *sourceParser) consumeInterfaceMember() ast.InterfaceMember {
	return p.consumeMember(false)
}

func (p *sourceParser) consumeMixinMember() ast.MixinMember {
	return p.consumeMember(false)
}

// consumeMember attempts to consume a member definition in a declaration.
func (p *sourceParser) consumeMember(dict bool) *ast.Member {
	n := &ast.Member{}
	defer p.node(n)()

	n.Annotations = p.tryConsumeAnnotations()
	n.Attribute = dict

	// getter/setter
	if p.isIdentifier("getter") || p.isIdentifier("setter") {
		n.Specialization = p.consumeIdentifier()
	} else if p.tryConsumeKeyword("stringifier") {
		n.Specialization = "stringifier"
	}

	if p.tryConsumeKeyword("const") {
		n.Const = true
	}

	if p.tryConsumeKeyword("static") {
		n.Static = true
	}

	if p.tryConsumeKeyword("readonly") {
		n.Readonly = true
	}

	if p.tryConsumeKeyword("required") {
		n.Required = true
	}

	if p.tryConsumeKeyword("attribute") {
		n.Attribute = true
	}

	if len(n.Annotations) == 0 {
		n.Annotations = p.tryConsumeAnnotations()
	}

	// Consume the type of the member.
	n.Type = p.consumeType()

	// Consume the member's name.
	n.Name, _ = p.tryConsumeIdentifier()

	// If not an attribute, consume the parameters of the member.
	if !n.Attribute && !n.Const {
		n.Parameters = p.consumeParameters()
	}
	n.Init = p.tryConsumeDefaultValue()
	return n
}

// tryConsumeAnnotations consumes any annotations found on the parent node.
func (p *sourceParser) tryConsumeAnnotations() (out []*ast.Annotation) {
	for {
		// [
		if _, ok := p.tryConsume(tokenTypeLeftBracket); !ok {
			return
		}

		for {
			// Foo()
			out = append(out, p.consumeAnnotationPart())

			// ,
			if _, ok := p.tryConsume(tokenTypeComma); !ok {
				break
			}
		}

		// ]
		if _, ok := p.consume(tokenTypeRightBracket); !ok {
			return
		}
	}
}

// consumeAnnotationPart consumes an annotation, as found within a set of brackets `[]`.
func (p *sourceParser) consumeAnnotationPart() *ast.Annotation {
	n := &ast.Annotation{}
	defer p.node(n)()

	// Consume the name of the annotation.
	n.Name = p.consumeIdentifier()

	// "="
	if _, ok := p.tryConsume(tokenTypeEquals); ok {
		// Consume (optional) value.

		// "("
		if list, ok := p.tryConsumeIdentifiersList(); ok {
			n.Values = list
		} else {
			n.Value = p.consumeIdentifier()
		}
	} else if p.isToken(tokenTypeLeftParen) {
		// Consume (optional) parameters.
		n.Parameters = p.consumeParameters()
	}

	return n
}

func (p *sourceParser) tryConsumeIdentifiersList() ([]string, bool) {
	// "("
	_, ok := p.tryConsume(tokenTypeLeftParen)
	if !ok {
		return nil, false
	}
	// identifier list
	var list []string
	for {
		list = append(list, p.consumeIdentifier())
		// ","
		if _, ok := p.tryConsume(tokenTypeComma); !ok {
			break
		}
	}
	// ")"
	p.consume(tokenTypeRightParen)
	return list, true
}

// expandedTypeKeywords defines the keywords that form the prefixes for expanded types:
// two-identifier type names.
var expandedTypeKeywords = map[string][]string{
	"unsigned":     {"short", "long"},
	"long":         {"long"},
	"unrestricted": {"float", "double"},
}

func (p *sourceParser) consumeType() ast.Type {
	base := &ast.Base{}
	finish := p.node(base)
	if p.tryConsumeKeyword("any") {
		finish()
		return &ast.AnyType{Base: *base}
	} else if p.tryConsumeKeyword("sequence") {
		seq := &ast.SequenceType{}
		p.consume(tokenTypeLeftTri)
		seq.Elem = p.consumeType()
		p.consume(tokenTypeRightTri)
		finish()
		seq.Base = *base
		return seq
	}
	if _, ok := p.tryConsume(tokenTypeLeftParen); ok {
		// "("
		var types []ast.Type
		for {
			types = append(types, p.consumeType())
			if !p.tryConsumeKeyword("or") {
				break
			}
		}
		// ")"
		p.consume(tokenTypeRightParen)
		finish()
		return &ast.UnionType{Base: *base, Types: types}
	}

	identifier := p.consumeIdentifier()
	typeName := identifier

	// If the identifier is the beginning of a possible expanded type name, check for the
	// secondary portion.
	if secondaries, ok := expandedTypeKeywords[identifier]; ok {
		for _, secondary := range secondaries {
			if p.isToken(tokenTypeIdentifier) && p.currentToken.value == secondary {
				typeName += " " + secondary
				p.consume(tokenTypeIdentifier)
				break
			}
		}
	}
	finish()
	tp := &ast.TypeName{Base: *base, Name: typeName}
	if _, ok := p.tryConsume(tokenTypeQuestionMark); ok {
		nl := &ast.NullableType{Base: tp.Base, Type: tp}
		nl.End++
		return nl
	}
	return tp
}

// consumeParameter attempts to consume a parameter.
func (p *sourceParser) consumeParameter() *ast.Parameter {
	n := &ast.Parameter{}
	defer p.node(n)()
	n.Annotations = p.tryConsumeAnnotations()

	// optional
	if p.tryConsumeKeyword("optional") {
		n.Optional = true
	}

	// Consume the parameter's type.
	n.Type = p.consumeType()
	if _, ok := p.tryConsume(tokenTypeVariadic); ok {
		n.Variadic = true
	}

	// Consume the parameter's name.
	n.Name = p.consumeIdentifier()

	n.Init = p.tryConsumeDefaultValue()

	return n
}

func (p *sourceParser) tryConsumeDefaultValue() *ast.Literal {
	if _, ok := p.tryConsume(tokenTypeEquals); ok {
		return p.consumeLiteral()
	}
	return nil
}

func (p *sourceParser) consumeDefaultValue() string {
	p.consume(tokenTypeEquals)
	return p.consumeIdentifier()
}

// consumeParameters attempts to consume a set of parameters.
func (p *sourceParser) consumeParameters() (out []*ast.Parameter) {
	p.consume(tokenTypeLeftParen)
	if _, ok := p.tryConsume(tokenTypeRightParen); ok {
		return
	}

	for {
		out = append(out, p.consumeParameter())
		if _, ok := p.tryConsume(tokenTypeRightParen); ok {
			return
		}

		if _, ok := p.consume(tokenTypeComma); !ok {
			return
		}
	}
}

// consumeImplementation attempts to consume an implementation definition.
func (p *sourceParser) consumeImplementation() *ast.Implementation {
	n := &ast.Implementation{}
	defer p.node(n)()

	// identifier
	n.Name = p.consumeIdentifier()

	// implements
	if !p.consumeKeyword("implements") {
		return n
	}

	// identifier
	n.Source = p.consumeIdentifier()

	// semicolon
	p.consume(tokenTypeSemicolon)
	return n
}

func (p *sourceParser) consumeIncludes() *ast.Includes {
	n := &ast.Includes{}
	defer p.node(n)()

	// identifier
	n.Name = p.consumeIdentifier()

	// implements
	if !p.consumeKeyword("includes") {
		return n
	}

	// identifier
	n.Source = p.consumeIdentifier()

	// semicolon
	p.consume(tokenTypeSemicolon)
	return n
}
