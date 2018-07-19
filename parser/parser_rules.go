// Copyright 2015 The Serulian Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
package parser

import "github.com/dennwc/webidl/ast"

// Parse parses the given WebIDL source into a parse tree.
func Parse(source InputSource, input string) *ast.File {
	lexer := lex(source, input)

	config := parserConfig{
		ignoredTokenTypes: map[tokenType]bool{
			tokenTypeWhitespace: true,
			tokenTypeComment:    true,
		},

		isCommentToken: func(kind tokenType) bool {
			return kind == tokenTypeComment
		},

		keywordTokenType: tokenTypeKeyword,
		errorTokenType:   tokenTypeError,
		eofTokenType:     tokenTypeEOF,
	}

	parser := buildParser(lexer, config, bytePosition(0), input)
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
	for {
		switch {

		case p.isToken(tokenTypeLeftBracket) || p.isKeyword("interface") || p.isKeyword("dictionary"):
			n.Declarations = append(n.Declarations, p.consumeDeclaration())

		case p.isToken(tokenTypeIdentifier):
			name := p.consumeIdentifier()
			if name == "callback" { // cannot be a keyword, because it's a common name for identifiers
				cb := p.consumeCallbackType()
				cb.Name = name
				n.Declarations = append(n.Declarations, cb)
				p.consume(tokenTypeSemicolon)
				continue
			} else if p.tryConsumeKeyword("implements") {
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
			p.emitError("Unexpected token at root level: %v", p.currentToken.kind)
			break Loop

		default:
			p.emitError("Unexpected token at root level: %v", p.currentToken.kind)
			break Loop
		}

		if p.isToken(tokenTypeEOF) {
			break Loop
		}
	}

	return n
}

// consumeDeclaration attempts to consume a declaration, with optional attributes.
func (p *sourceParser) consumeDeclaration() *ast.Declaration {
	decl := &ast.Declaration{}
	defer p.node(decl)()

	// Consume any annotations.
	decl.Annotations = p.tryConsumeAnnotations()

	// Consume the type of declaration.
	if p.tryConsumeKeyword("interface") {
		decl.Kind = "interface"
		if p.tryConsumeKeyword("mixin") {
			decl.Mixin = true
		}
	} else if p.tryConsumeKeyword("dictionary") {
		decl.Kind = "dictionary"
	} else {
		p.consumeKeyword("interface")
		return decl
	}

	// Consume the name of the declaration.
	decl.Name = p.consumeIdentifier()

	// Check for (optional) inheritance.
	if _, ok := p.tryConsume(tokenTypeColon); ok {
		decl.ParentType = p.consumeIdentifier()
	}

	// {
	p.consume(tokenTypeLeftBrace)

	// Members and custom operations (if any).
	isDict := decl.Kind == "dictionary"
loop:
	for {
		if p.isToken(tokenTypeRightBrace) {
			break
		}

		if p.isKeyword("serializer") || p.isKeyword("jsonifier") {
			customOpNode := &ast.CustomOp{}
			finish := p.node(customOpNode)
			customOpNode.Name = p.currentToken.value

			p.consume(tokenTypeKeyword)
			_, ok := p.consume(tokenTypeSemicolon)
			finish()

			decl.CustomOps = append(decl.CustomOps, customOpNode)

			if !ok {
				break loop
			}

			continue
		} else if p.isKeyword("iterable") {
			p.consume(tokenTypeKeyword)
			iter := &ast.Iterable{}
			finish := p.node(iter)
			p.consume(tokenTypeLeftTri)
			iter.Type = p.consumeType()
			p.consume(tokenTypeRightTri)
			finish()
			decl.Iterable = iter
			_, ok := p.consume(tokenTypeSemicolon)
			if !ok {
				break loop
			}

			continue
		}
		decl.Members = append(decl.Members, p.consumeMember(isDict))

		if _, ok := p.consume(tokenTypeSemicolon); !ok {
			p.emitError("Expected semicolon, got: %v", p.currentToken.kind)
			break
		}
	}

	// };
	p.consume(tokenTypeRightBrace)
	p.consume(tokenTypeSemicolon)
	return decl
}

// consumeMember attempts to consume a member definition in a declaration.
func (p *sourceParser) consumeMember(dict bool) *ast.Member {
	n := &ast.Member{}
	defer p.node(n)()

	// annotations
	n.Annotations = p.tryConsumeAnnotations()
	n.Attribute = dict

	// getter/setter
	if p.isKeyword("getter") || p.isKeyword("setter") {
		consumed, _ := p.consume(tokenTypeKeyword)
		n.Specialization = consumed.value
	}

	if p.tryConsumeKeyword("const") {
		n.Const = true
	}

	// static readonly attribute
	if p.tryConsumeKeyword("static") {
		n.Static = true
	}

	if p.tryConsumeKeyword("readonly") {
		n.Readonly = true
	}

	if p.tryConsumeKeyword("attribute") {
		n.Attribute = true
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

func (p *sourceParser) consumeCallbackType() *ast.Callback {
	cb := &ast.Callback{}
	defer p.node(cb)()
	cb.Type = p.consumeType()
	cb.Parameters = p.consumeParameters()
	return cb
}

// expandedTypeKeywords defines the keywords that form the prefixes for expanded types:
// two-identifier type names.
var expandedTypeKeywords = map[string][]string{
	"unsigned":     {"short", "long"},
	"long":         {"long"},
	"unrestricted": {"float", "double"},
}

// consumeType attempts to consume a type (identifier (with optional ?) or 'any').
func (p *sourceParser) consumeType() *ast.Type {
	n := &ast.Type{}
	defer p.node(n)()
	if p.tryConsumeKeyword("any") {
		n.Any = true
		return n
	} else if p.tryConsumeKeyword("sequence") {
		p.consume(tokenTypeLeftTri)
		n.SequenceOf = p.consumeType()
		p.consume(tokenTypeRightTri)
		return n
	}
	if _, ok := p.tryConsume(tokenTypeLeftParen); ok {
		// "("
		var types []*ast.Type
		for {
			types = append(types, p.consumeType())
			if !p.tryConsumeKeyword("or") {
				break
			}
		}
		// ")"
		p.consume(tokenTypeRightParen)
		n.UnionOf = types
		return n
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
	n.Name = typeName

	if _, ok := p.tryConsume(tokenTypeQuestionMark); ok {
		n.Nullable = true
	}
	return n
}

// consumeParameter attempts to consume a parameter.
func (p *sourceParser) consumeParameter() *ast.Parameter {
	n := &ast.Parameter{}
	defer p.node(n)()

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

func (p *sourceParser) tryConsumeDefaultValue() string {
	if _, ok := p.tryConsume(tokenTypeEquals); ok {
		return p.consumeIdentifier()
	}
	return ""
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
