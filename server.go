package asyncgo

import (
	"errors"
	"fmt"

	"github.com/RubenRibGarcia/asyncgo/spec"
)

type serversItem []*server

// Servers adds one or more servers to the document.
func Servers(s ...*server) Item { return serversItem(s) }

func (s serversItem) apply(b *builder) error {
	if b.doc.Servers == nil {
		b.doc.Servers = map[string]*spec.Server{}
	}
	var errs []error
	for _, sv := range s {
		if err := sv.apply(b); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

type server struct {
	name string
	s    spec.Server
}

// Server declares a server (broker) with a name, protocol, and host.
func Server(name, protocol, host string) *server {
	return &server{name: name, s: spec.Server{Protocol: protocol, Host: host}}
}

func (s *server) ProtocolVersion(v string) *server { s.s.ProtocolVersion = v; return s }
func (s *server) Description(d string) *server     { s.s.Description = d; return s }

// Tags attaches tags for logical grouping and categorization of the server.
func (s *server) Tags(tags ...spec.Tag) *server {
	s.s.Tags = append(s.s.Tags, tags...)
	return s
}

// ExternalDocs attaches additional external documentation for the server.
func (s *server) ExternalDocs(d spec.ExternalDocs) *server {
	s.s.ExternalDocs = &d
	return s
}

// Variable declares a server URL variable.
func (s *server) Variable(name string, v spec.ServerVariable) *server {
	if s.s.Variables == nil {
		s.s.Variables = map[string]*spec.ServerVariable{}
	}
	s.s.Variables[name] = &v
	return s
}

// Security declares the security schemes a client can use with this server.
// Every scheme must be declared via SecuritySchemes(...).
func (s *server) Security(schemes ...*securityScheme) *server {
	for _, sc := range schemes {
		s.s.Security = append(s.s.Security, securitySchemeRef(sc))
	}
	return s
}

func (s *server) apply(b *builder) error {
	var errs []error
	if s.name == "" {
		errs = append(errs, fmt.Errorf("server.name: is required"))
	}
	if s.s.Protocol == "" {
		errs = append(errs, fmt.Errorf("server.%s.protocol: is required", s.name))
	}
	if s.s.Host == "" {
		errs = append(errs, fmt.Errorf("server.%s.host: is required", s.name))
	}
	if _, dup := b.doc.Servers[s.name]; dup && s.name != "" {
		errs = append(errs, fmt.Errorf("server.%s: duplicate name", s.name))
	} else {
		b.doc.Servers[s.name] = &s.s
	}
	return errors.Join(errs...)
}
