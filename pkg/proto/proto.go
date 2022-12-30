package proto

import (
	"fmt"
	"strconv"
	"strings"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"
)

type ProtoDefinition struct {
	builder     strings.Builder
	indendation int
	pb          descriptorpb.FileDescriptorProto
	descriptor  protoreflect.FileDescriptor
	filename    string
}

// TODO add proto2 support

func (pd *ProtoDefinition) indent() {
	pd.indendation += 1
}

func (pd *ProtoDefinition) dedent() {
	pd.indendation -= 1
}

func (pd *ProtoDefinition) writeIndented(s string) {
	pd.builder.WriteString(strings.Repeat("    ", pd.indendation))
	pd.write(s)
}

func (pd *ProtoDefinition) write(s string) {
	pd.builder.WriteString(s)
}

func (pd *ProtoDefinition) String() string {
	return pd.builder.String()
}

func (pd *ProtoDefinition) Filename() string {
	return pd.filename
}

func (pd *ProtoDefinition) writeMethod(method protoreflect.MethodDescriptor) {
	// TODO need to handle method options
	pd.writeIndented("rpc ")
	pd.write(string(method.Name()))
	pd.write("(")
	if method.IsStreamingClient() {
		pd.write("streaming ")
	}
	pd.write(string(method.Input().Name()))
	pd.write(") returns (")
	if method.IsStreamingServer() {
		pd.write("streaming ")
	}
	pd.write(string(method.Output().Name()))
	pd.write(") {}\n")
}

func (pd *ProtoDefinition) writeService(service protoreflect.ServiceDescriptor) {
	// TODO need to handle service options
	pd.write("service ")
	pd.write(string(service.Name()))
	pd.write(" {\n")
	pd.indent()
	for i := 0; i < service.Methods().Len(); i++ {
		pd.writeMethod(service.Methods().Get(i))
	}
	pd.dedent()
	pd.writeIndented("}\n\n")
}

func (pd *ProtoDefinition) writeType(field protoreflect.FieldDescriptor) {
	kind := field.Kind().String()
	if kind == "message" {
		pd.write(string(field.Message().Name()))
	} else if kind == "map" {
		pd.write("map<")
		pd.writeType(field.MapKey())
		pd.write(", ")
		pd.writeType(field.MapValue())
		pd.write(">")
	} else {
		pd.write(kind)
	}
}

func (pd *ProtoDefinition) writeOneof(oneof protoreflect.OneofDescriptor) {
	// TODO need to handle oneof options
	pd.writeIndented("")
	pd.write("oneof ")
	pd.write(string(oneof.Name()))
	pd.write(" {\n")
	pd.indent()
	for i := 0; i < oneof.Fields().Len(); i++ {
		pd.writeField(oneof.Fields().Get(i))
	}
	pd.dedent()
	pd.writeIndented("}\n")
}

func (pd *ProtoDefinition) writeField(field protoreflect.FieldDescriptor) {
	// TODO need to handle options
	pd.writeIndented("")
	if field.HasOptionalKeyword() {
		pd.write("optional ")
	} else if field.Cardinality().String() == "repeated" {
		pd.write("repeated ")
	}
	pd.writeType(field)
	pd.write(" ")
	pd.write(string(field.Name()))
	pd.write(" = ")
	pd.write(strconv.Itoa(field.Index()))
	pd.write(";\n")
}

func (pd *ProtoDefinition) writeEnum(enum protoreflect.EnumDescriptor) {
	pd.write("enum ")
	pd.write(string(enum.Name()))
	pd.write(" {\n")
	// TODO need to handle enum options (allow_alias)
	pd.indent()
	for i := 0; i < enum.Values().Len(); i++ {
		value := enum.Values().Get(i)
		pd.writeIndented(string(value.Name()))
		pd.write(" = ")
		pd.write(fmt.Sprintf("%d", value.Number()))
		pd.write(";\n")
	}
	pd.dedent()
	pd.writeIndented("}\n\n")
}

func (pd *ProtoDefinition) writeMessage(message protoreflect.MessageDescriptor) {
	// TODO need to handle message options
	pd.write("message ")
	pd.write(string(message.Name()))
	pd.write(" {\n")
	pd.indent()

	for i := 0; i < message.ReservedNames().Len(); i++ {
		name := message.ReservedNames().Get(i)
		pd.writeIndented("reserved ")
		pd.write(string(name))
		pd.write(";\n")
	}

	for i := 0; i < message.ReservedRanges().Len(); i++ {
		reservedRange := message.ReservedRanges().Get(i)
		pd.writeIndented("reserved ")
		if reservedRange[0] == reservedRange[1] {
			pd.write(string(reservedRange[0]))
		} else {
			if reservedRange[0] > reservedRange[1] {
				reservedRange[1], reservedRange[0] = reservedRange[0], reservedRange[1]
			}
			pd.write(fmt.Sprintf("%d", reservedRange[0]))
			pd.write(" to ")
			pd.write(fmt.Sprintf("%d", reservedRange[1]))
		}
		pd.write(";\n")
	}

	for i := 0; i < message.Fields().Len(); i++ {
		field := message.Fields().Get(i)
		if field.ContainingOneof() == nil {
			pd.writeField(field)
		}
	}

	for i := 0; i < message.Oneofs().Len(); i++ {
		pd.writeOneof(message.Oneofs().Get(i))
	}
	pd.dedent()
	pd.writeIndented("}\n\n")
}

func (pd *ProtoDefinition) writeImport(fileImport protoreflect.FileImport) {
	pd.write("import ")
	if fileImport.IsPublic {
		pd.write("public ")
	}
	pd.write("\"")
	pd.write(fileImport.Path())
	pd.write("\";\n")
}

func (pd *ProtoDefinition) writeFileDescriptor() {
	pd.write("syntax = \"")
	pd.write(pd.descriptor.Syntax().String())
	pd.write("\"\n\n")

	pd.write("package ")
	pd.write(string(pd.descriptor.Package().Name()))
	pd.write(";\n\n")

	// TODO need to handle FileOptions
	filepb := protodesc.ToFileDescriptorProto(pd.descriptor)
	options := filepb.Options
	if options != nil {
	}

	for i := 0; i < pd.descriptor.Imports().Len(); i++ {
		pd.writeImport(pd.descriptor.Imports().Get(i))
	}

	if pd.descriptor.Imports().Len() > 0 {
		pd.write("\n")
	}

	for i := 0; i < pd.descriptor.Services().Len(); i++ {
		pd.writeService(pd.descriptor.Services().Get(i))
	}

	for i := 0; i < pd.descriptor.Messages().Len(); i++ {
		pd.writeMessage(pd.descriptor.Messages().Get(i))
	}

	for i := 0; i < pd.descriptor.Enums().Len(); i++ {
		pd.writeEnum(pd.descriptor.Enums().Get(i))
	}
}

func NewFromBytes(payload []byte) (*ProtoDefinition, error) {
	var pb descriptorpb.FileDescriptorProto
	err := proto.Unmarshal(payload, &pb)
	if err != nil {
		return nil, fmt.Errorf("Couldn't unmarshal proto: %w", err)
	}

	return NewFromDescriptor(pb)
}

func NewFromDescriptor(pb descriptorpb.FileDescriptorProto) (*ProtoDefinition, error) {
	fileOptions := protodesc.FileOptions{AllowUnresolvable: true}
	descriptor, err := fileOptions.New(&pb, &protoregistry.Files{})

	if err != nil {
		return nil, fmt.Errorf("Couldn't create FileDescriptor: %w", err)
	}

	pd := ProtoDefinition{
		pb:         pb,
		descriptor: descriptor,
		filename:   descriptor.Path(),
	}

	pd.writeFileDescriptor()

	return &pd, nil

}