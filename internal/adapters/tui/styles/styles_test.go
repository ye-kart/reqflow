package styles

import "testing"

func TestTabBarStyleIsNotEmpty(t *testing.T) {
	s := TabBar()
	rendered := s.Render("Request")
	if rendered == "" {
		t.Error("expected non-empty tab bar render")
	}
}

func TestActiveTabStyleIsNotEmpty(t *testing.T) {
	s := ActiveTab()
	rendered := s.Render("Request")
	if rendered == "" {
		t.Error("expected non-empty active tab render")
	}
}

func TestInactiveTabStyleIsNotEmpty(t *testing.T) {
	s := InactiveTab()
	rendered := s.Render("Request")
	if rendered == "" {
		t.Error("expected non-empty inactive tab render")
	}
}

func TestStatusBarStyleIsNotEmpty(t *testing.T) {
	s := StatusBar()
	rendered := s.Render("Ready")
	if rendered == "" {
		t.Error("expected non-empty status bar render")
	}
}

func TestStatusCodeColors(t *testing.T) {
	tests := []struct {
		code int
		name string
	}{
		{200, "2xx success"},
		{301, "3xx redirect"},
		{404, "4xx client error"},
		{500, "5xx server error"},
		{0, "unknown status"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := StatusCode(tt.code)
			rendered := s.Render("200 OK")
			if rendered == "" {
				t.Errorf("expected non-empty status code render for code %d", tt.code)
			}
		})
	}
}

func TestMethodColorReturnsNonEmpty(t *testing.T) {
	methods := []string{"GET", "POST", "PUT", "PATCH", "DELETE", "UNKNOWN"}
	for _, m := range methods {
		t.Run(m, func(t *testing.T) {
			s := MethodColor(m)
			rendered := s.Render(m)
			if rendered == "" {
				t.Errorf("expected non-empty method color render for %s", m)
			}
		})
	}
}
