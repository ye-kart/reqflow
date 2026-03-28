package plugin

import (
	"context"
	"sort"
	"testing"
)

// --- Fake implementations for testing ---

type fakeProtocol struct {
	name string
}

func (f *fakeProtocol) Name() string { return f.name }
func (f *fakeProtocol) Execute(ctx context.Context, config map[string]interface{}) (map[string]interface{}, error) {
	return map[string]interface{}{"result": "ok"}, nil
}

type fakeAuthProvider struct {
	name     string
	authType string
}

func (f *fakeAuthProvider) Name() string { return f.name }
func (f *fakeAuthProvider) Type() string { return f.authType }
func (f *fakeAuthProvider) Apply(headers map[string]string, config map[string]string) (map[string]string, error) {
	headers["Authorization"] = "fake"
	return headers, nil
}

type fakeFormatter struct {
	name string
}

func (f *fakeFormatter) Name() string { return f.name }
func (f *fakeFormatter) Format(data map[string]interface{}) ([]byte, error) {
	return []byte("formatted"), nil
}

// --- Tests ---

func TestRegisterAndGetProtocol(t *testing.T) {
	resetRegistry()

	p := &fakeProtocol{name: "grpc"}
	RegisterProtocol(p)

	got, ok := GetProtocol("grpc")
	if !ok {
		t.Fatal("expected protocol to be found")
	}
	if got.Name() != "grpc" {
		t.Errorf("expected name %q, got %q", "grpc", got.Name())
	}
}

func TestGetProtocol_NotFound(t *testing.T) {
	resetRegistry()

	_, ok := GetProtocol("nonexistent")
	if ok {
		t.Fatal("expected protocol not to be found")
	}
}

func TestRegisterAndGetAuthProvider(t *testing.T) {
	resetRegistry()

	a := &fakeAuthProvider{name: "custom-oauth", authType: "oauth2"}
	RegisterAuthProvider(a)

	got, ok := GetAuthProvider("custom-oauth")
	if !ok {
		t.Fatal("expected auth provider to be found")
	}
	if got.Name() != "custom-oauth" {
		t.Errorf("expected name %q, got %q", "custom-oauth", got.Name())
	}
	if got.Type() != "oauth2" {
		t.Errorf("expected type %q, got %q", "oauth2", got.Type())
	}
}

func TestGetAuthProvider_NotFound(t *testing.T) {
	resetRegistry()

	_, ok := GetAuthProvider("nonexistent")
	if ok {
		t.Fatal("expected auth provider not to be found")
	}
}

func TestRegisterAndGetFormatter(t *testing.T) {
	resetRegistry()

	f := &fakeFormatter{name: "xml"}
	RegisterFormatter(f)

	got, ok := GetFormatter("xml")
	if !ok {
		t.Fatal("expected formatter to be found")
	}
	if got.Name() != "xml" {
		t.Errorf("expected name %q, got %q", "xml", got.Name())
	}
}

func TestGetFormatter_NotFound(t *testing.T) {
	resetRegistry()

	_, ok := GetFormatter("nonexistent")
	if ok {
		t.Fatal("expected formatter not to be found")
	}
}

func TestListProtocols(t *testing.T) {
	resetRegistry()

	RegisterProtocol(&fakeProtocol{name: "grpc"})
	RegisterProtocol(&fakeProtocol{name: "graphql"})

	names := ListProtocols()
	sort.Strings(names)
	if len(names) != 2 {
		t.Fatalf("expected 2 protocols, got %d", len(names))
	}
	if names[0] != "graphql" || names[1] != "grpc" {
		t.Errorf("expected [graphql grpc], got %v", names)
	}
}

func TestListAuthProviders(t *testing.T) {
	resetRegistry()

	RegisterAuthProvider(&fakeAuthProvider{name: "oauth-ext", authType: "oauth2"})
	RegisterAuthProvider(&fakeAuthProvider{name: "jwt-ext", authType: "jwt"})

	names := ListAuthProviders()
	sort.Strings(names)
	if len(names) != 2 {
		t.Fatalf("expected 2 auth providers, got %d", len(names))
	}
	if names[0] != "jwt-ext" || names[1] != "oauth-ext" {
		t.Errorf("expected [jwt-ext oauth-ext], got %v", names)
	}
}

func TestListFormatters(t *testing.T) {
	resetRegistry()

	RegisterFormatter(&fakeFormatter{name: "xml"})
	RegisterFormatter(&fakeFormatter{name: "csv"})

	names := ListFormatters()
	sort.Strings(names)
	if len(names) != 2 {
		t.Fatalf("expected 2 formatters, got %d", len(names))
	}
	if names[0] != "csv" || names[1] != "xml" {
		t.Errorf("expected [csv xml], got %v", names)
	}
}

func TestDuplicateRegistrationOverwrites(t *testing.T) {
	resetRegistry()

	p1 := &fakeProtocol{name: "grpc"}
	p2 := &fakeProtocol{name: "grpc"}
	RegisterProtocol(p1)
	RegisterProtocol(p2)

	got, ok := GetProtocol("grpc")
	if !ok {
		t.Fatal("expected protocol to be found")
	}
	// The second registration should overwrite the first.
	if got != p2 {
		t.Error("expected duplicate registration to overwrite the previous one")
	}

	// List should still only contain one entry.
	names := ListProtocols()
	if len(names) != 1 {
		t.Errorf("expected 1 protocol after duplicate registration, got %d", len(names))
	}
}

func TestDuplicateAuthProviderOverwrites(t *testing.T) {
	resetRegistry()

	a1 := &fakeAuthProvider{name: "custom", authType: "v1"}
	a2 := &fakeAuthProvider{name: "custom", authType: "v2"}
	RegisterAuthProvider(a1)
	RegisterAuthProvider(a2)

	got, ok := GetAuthProvider("custom")
	if !ok {
		t.Fatal("expected auth provider to be found")
	}
	if got.Type() != "v2" {
		t.Errorf("expected overwritten type %q, got %q", "v2", got.Type())
	}
}

func TestDuplicateFormatterOverwrites(t *testing.T) {
	resetRegistry()

	f1 := &fakeFormatter{name: "xml"}
	f2 := &fakeFormatter{name: "xml"}
	RegisterFormatter(f1)
	RegisterFormatter(f2)

	got, ok := GetFormatter("xml")
	if !ok {
		t.Fatal("expected formatter to be found")
	}
	if got != f2 {
		t.Error("expected duplicate registration to overwrite the previous one")
	}
}

func TestListEmpty(t *testing.T) {
	resetRegistry()

	if names := ListProtocols(); len(names) != 0 {
		t.Errorf("expected empty list, got %v", names)
	}
	if names := ListAuthProviders(); len(names) != 0 {
		t.Errorf("expected empty list, got %v", names)
	}
	if names := ListFormatters(); len(names) != 0 {
		t.Errorf("expected empty list, got %v", names)
	}
}

func TestProtocolExecute(t *testing.T) {
	resetRegistry()

	p := &fakeProtocol{name: "test"}
	RegisterProtocol(p)

	got, ok := GetProtocol("test")
	if !ok {
		t.Fatal("expected protocol to be found")
	}

	result, err := got.Execute(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result["result"] != "ok" {
		t.Errorf("unexpected result: %v", result)
	}
}

func TestAuthProviderApply(t *testing.T) {
	resetRegistry()

	a := &fakeAuthProvider{name: "test-auth", authType: "custom"}
	RegisterAuthProvider(a)

	got, ok := GetAuthProvider("test-auth")
	if !ok {
		t.Fatal("expected auth provider to be found")
	}

	headers, err := got.Apply(map[string]string{}, map[string]string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if headers["Authorization"] != "fake" {
		t.Errorf("expected Authorization header to be set")
	}
}

func TestFormatterFormat(t *testing.T) {
	resetRegistry()

	f := &fakeFormatter{name: "test-fmt"}
	RegisterFormatter(f)

	got, ok := GetFormatter("test-fmt")
	if !ok {
		t.Fatal("expected formatter to be found")
	}

	data, err := got.Format(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(data) != "formatted" {
		t.Errorf("expected %q, got %q", "formatted", string(data))
	}
}
