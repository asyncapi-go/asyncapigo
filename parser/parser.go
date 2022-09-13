package parser

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"log"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	"github.com/asyncapi-go/asyncapigo/model"

	"github.com/KyleBanks/depth"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"golang.org/x/tools/go/ast/inspector"
)

var infoExp = regexp.MustCompile(`[/ ]*@([a-z.]*) (.*)`)
var handlerMatchParamNameExp = regexp.MustCompile(`(?mU)@([a-z].*) `)

// Parse comments like " // @param value"
var handlerMatchSimpleValueExp = regexp.MustCompile(`(?m)@[a-z].* (.*)`)

// parse comments like " // @param value:field1=2;field2='some value';"
var handlerMatchValueWithParamsExp = regexp.MustCompile(`@[a-z].* ([a-zA-Z]*): *(.*)`)

var modelMatchJsonFieldNameExp = regexp.MustCompile(`(?U)json:"(.*)["|,]`)

var modelParseTagExp = regexp.MustCompile(`(?U)([a-zA-Z0-9]*):"(.*)"`)

type Parser struct {
	basePath   string
	api        *model.AsyncApi
	models     map[string]*ast.TypeSpec
	needModels map[string]struct{}
}

func New(basePath string, api *model.AsyncApi) *Parser {
	return &Parser{
		basePath:   basePath,
		api:        api,
		models:     make(map[string]*ast.TypeSpec),
		needModels: make(map[string]struct{}),
	}
}

func (p *Parser) Parse(path string, parseDependencies int) error {
	astPackages, err := parser.ParseDir(
		token.NewFileSet(),
		path,
		nil,
		parser.ParseComments,
	)
	if err != nil {
		log.Fatalf("parse file: %v", err)
	}

	for _, astPackage := range astPackages {
		if parseDependencies > 0 {
			var t depth.Tree
			t.ResolveInternal = true
			t.MaxDepth = 4

			err = t.Resolve(path)
			if err != nil {
				log.Printf("pkg %s cannot find all dependencies, %s\n", path, err)
			}
			for i := 0; i < len(t.Root.Deps); i++ {
				if strings.HasPrefix(t.Root.Deps[i].Raw.Dir, p.basePath) {
					relPath := strings.Replace(t.Root.Deps[i].Raw.Dir, p.basePath, ".", 1)
					err = p.Parse(relPath, parseDependencies-1)
					if err != nil {
						return err
					}
				}
			}
		}

		for _, astInFile := range astPackage.Files {
			for _, commentGroup := range astInFile.Comments {
				for _, comment := range commentGroup.List {
					parsed := infoExp.FindStringSubmatch(comment.Text)
					if len(parsed) != 3 {
						continue
					}
					fieldName := parsed[1]
					fieldValue := parsed[2]
					fillFieldString(reflect.ValueOf(&p.api.Info), fieldName, fieldValue)
				}
			}

			i := inspector.New([]*ast.File{astInFile})
			iFilter := []ast.Node{&ast.GenDecl{}, &ast.FuncDecl{}}
			i.Nodes(
				iFilter, func(node ast.Node, push bool) (proceed bool) {
					switch decl := node.(type) {
					case *ast.FuncDecl:
						err := p.parseFunc(decl)
						if err != nil {
							fmt.Println("\tparse function warning:", err)
						}
					case *ast.GenDecl:
						if decl, ok := decl.Specs[0].(*ast.TypeSpec); ok {
							p.models[fmt.Sprintf("%s.%s", astPackage.Name, decl.Name)] = decl
						}
					}
					return false
				},
			)
		}
	}

	needParse := true
	for needParse {
		needParse = false
		for name := range p.needModels {
			oldLen := len(p.needModels)
			decl, ok := p.models[name]
			if !ok {
				continue
			}
			p.api.Components.Schemas[name] = p.parseModel(decl)
			if oldLen < len(p.needModels) {
				needParse = true
			}
			delete(p.needModels, name)
		}
	}

	return nil
}

func (p *Parser) parseFunc(decl *ast.FuncDecl) error {
	if decl.Doc == nil {
		return nil
	}

	if !(len(decl.Doc.List) > 0 && strings.Contains(decl.Doc.List[0].Text, "asyncApi")) {
		return nil
	}

	fmt.Println("parse function", decl.Name.String())

	var message model.Message
	var queueName string
	var channelOperation string
	for _, comment := range decl.Doc.List {
		matches := handlerMatchParamNameExp.FindStringSubmatch(comment.Text)
		if len(matches) < 1 {
			continue
		}

		paramName := matches[1]
		switch paramName {
		case "queue":
			queueName = parseCommentSimpleValue(comment.Text)
		case "header":
			headerName, headerParams := parseCommentValueWithParams(comment.Text)
			message.Headers.Type = "object"
			if message.Headers.Properties == nil {
				message.Headers.Properties = make(map[string]model.Object)
			}
			if headerParams["required"] == "true" {
				message.Headers.Required = append(message.Headers.Required, headerName)
			}
			message.Headers.Properties[headerName] = model.Object{
				Type:        headerParams["type"],
				Description: headerParams["description"],
				Example:     parseStringIfPossible(headerParams["example"]),
				Format:      headerParams["format"],
			}
		case "payload":
			payloadName := parseCommentSimpleValue(comment.Text)
			message.Payload.Ref = fmt.Sprintf("#/components/schemas/%s", payloadName)
			p.needModels[payloadName] = struct{}{}
		case "tags", "tag":
			values := strings.Split(strings.Split(comment.Text, paramName+" ")[1], " ")
			message.Tags = make([]model.Tag, len(values))
			for i, value := range values {
				message.Tags[i] = model.Tag{Name: value}
			}
		case "operation":
			channelOperation = parseCommentSimpleValue(comment.Text)
		default:
			fillFieldString(reflect.ValueOf(&message), paramName, strings.Split(comment.Text, paramName+" ")[1])
		}
	}

	var messageName string

	switch channelOperation {
	case "subscribe":
		messageName = fmt.Sprintf("%s.subscribe.%s", queueName, decl.Name)
		if _, ok := p.api.Channels[queueName]; ok {
			channel := p.api.Channels[queueName]
			channel.Subscribe.Message.OneOf = append(
				channel.Subscribe.Message.OneOf,
				model.Object{Ref: fmt.Sprintf("#/components/messages/%s", messageName)},
			)
			p.api.Channels[queueName] = channel
		} else {
			p.api.Channels[queueName] = model.Channel{
				Subscribe: model.ChannelAction{
					Message: model.
						Object{OneOf: []model.Object{{Ref: fmt.Sprintf("#/components/messages/%s", messageName)}}},
				},
			}
		}
	default:
		messageName = fmt.Sprintf("%s.publish.%s", queueName, decl.Name)
		if _, ok := p.api.Channels[queueName]; ok {
			channel := p.api.Channels[queueName]
			channel.Publish.Message.OneOf = append(
				channel.Publish.Message.OneOf,
				model.Object{Ref: fmt.Sprintf("#/components/messages/%s", messageName)},
			)
			p.api.Channels[queueName] = channel
		} else {
			p.api.Channels[queueName] = model.Channel{
				Publish: model.ChannelAction{
					Message: model.
						Object{OneOf: []model.Object{{Ref: fmt.Sprintf("#/components/messages/%s", messageName)}}},
				},
			}
		}
	}

	if p.api.Components.Messages == nil {
		p.api.Components.Messages = make(map[string]model.Message)
	}

	p.api.Components.Messages[messageName] = message
	return nil
}

func (p *Parser) parseModel(decl *ast.TypeSpec) model.Object {
	fmt.Println("parse model", decl.Name.String())
	object := model.Object{}

	switch expr := decl.Type.(type) {
	case *ast.StructType:
		object.Type = "object"
		object.Properties = make(map[string]model.Object)
		for _, field := range expr.Fields.List {
			property := model.Object{}
			p.fillPropertyType(&property, field.Type)
			name := getJsonNameFromTags(field.Tag.Value, field.Names[0].Name)
			p.fillPropertyTagValues(&object, name, &property, field.Tag.Value)
			object.Properties[name] = property
		}
	case *ast.Ident:
		object.Type = mapType(expr.Name)

	}

	return object
}

func fillFieldString(objectValue reflect.Value, fieldPath string, fieldValue string) {
	fieldName := strings.Split(fieldPath, ".")[0]
	field := objectValue.Elem().FieldByName(cases.Title(language.Und).String(fieldName))

	if field.IsValid() {
		switch field.Kind() {
		case reflect.Struct:
			if strings.Contains(fieldPath, ".") {
				fillFieldString(
					field.Addr(),
					strings.Replace(fieldPath, fieldName+".", "", 1),
					fieldValue,
				)
			}
		default:
			field.SetString(fieldValue)
		}
	}
}

func parseCommentSimpleValue(comment string) string {
	values := handlerMatchSimpleValueExp.FindStringSubmatch(comment)
	if len(values) > 1 {
		return values[1]
	}
	return ""
}

func parseCommentValueWithParams(comment string) (string, map[string]string) {
	values := handlerMatchValueWithParamsExp.FindStringSubmatch(comment)
	if len(values) < 3 {
		if len(values) > 1 {
			return values[1], nil
		}
		return "", nil
	}
	name := values[1]
	fields := make(map[string]string)

	rawFields := values[2]
	rawFieldsSplit := strings.Split(rawFields, ";")
	for _, rawField := range rawFieldsSplit {
		rawField = strings.TrimSpace(rawField)
		keyValueSplit := strings.Split(rawField, "=")
		if len(keyValueSplit) != 2 {
			continue
		}
		key := keyValueSplit[0]
		value := keyValueSplit[1]
		value = strings.Trim(value, `"'`)
		fields[key] = value
	}

	return name, fields
}

func (p *Parser) fillPropertyType(property *model.Object, expr ast.Expr) {
	switch expr := expr.(type) {
	case *ast.Ident:
		property.Type = mapType(expr.Name)
	case *ast.SelectorExpr:
		property.Ref = fmt.Sprintf("#/components/schemas/%s.%s", expr.X.(*ast.Ident).Name, expr.Sel.Name)
		p.needModels[fmt.Sprintf("%s.%s", expr.X.(*ast.Ident).Name, expr.Sel.Name)] = struct{}{}
	case *ast.ArrayType:
		property.Type = "array"
		if mapType(expr.Elt.(*ast.Ident).Name) == "" {
			p.needModels[expr.Elt.(*ast.Ident).Name] = struct{}{}
			p.models[expr.Elt.(*ast.Ident).Name] = expr.Elt.(*ast.Ident).Obj.Decl.(*ast.TypeSpec)
			property.Items = []model.Object{{Ref: fmt.Sprintf("#/components/schemas/%s", expr.Elt.(*ast.Ident).Name)}}
		} else {
			property.Items = []model.Object{{Type: mapType(expr.Elt.(*ast.Ident).Name)}}
		}
	default:
		return
	}
}

func (p *Parser) fillPropertyTagValues(root *model.Object, fieldName string, property *model.Object, tag string) {
	matches := modelParseTagExp.FindAllStringSubmatch(tag, -1)
	for _, match := range matches {
		if len(match) != 3 {
			continue
		}
		key := match[1]
		value := match[2]
		switch key {
		case "example":
			property.Example = parseStringIfPossible(value)
		case "description":
			property.Description = value
		case "validate":
			fillObjectWithValidateTagValue(root, fieldName, property, value)
		}
	}
}

func fillObjectWithValidateTagValue(root *model.Object, fieldName string, property *model.Object, validate string) {
	values := strings.Split(validate, ",")
	for _, value := range values {
		switch strings.Split(strings.TrimSpace(value), "=")[0] {
		case "required":
			if len(root.Required) > 0 {
				root.Required = append(root.Required, fieldName)
			} else {
				root.Required = []string{fieldName}
			}
		case "min", "gt":
			property.Minimum = parseStringIfPossible(strings.Split(strings.TrimSpace(value), "=")[1])
		case "max", "lt":
			property.Maximum = parseStringIfPossible(strings.Split(strings.TrimSpace(value), "=")[1])
		case "oneof", "oneOf":
			values := strings.Split(strings.Split(strings.TrimSpace(value), "=")[1], " ")
			property.Enum = make([]any, len(values))
			for i, s := range values {
				property.Enum[i] = parseStringIfPossible(s)
			}
		}
	}
}

func getJsonNameFromTags(tag string, defaultName string) string {
	matches := modelMatchJsonFieldNameExp.FindStringSubmatch(tag)
	if len(matches) > 1 {
		return matches[1]
	}
	return defaultName

}

func parseStringIfPossible(value string) any {
	valueInt, err := strconv.Atoi(value)
	if err == nil {
		return valueInt
	}

	valueFloat, err := strconv.ParseFloat(value, 64)
	if err == nil {
		return valueFloat
	}

	return value
}
