package apply

import (
	"errors"
	"testing"

	er "github.com/inskribe/schemer/internal/errschemer"
)

func TestParseApplyCommand(t *testing.T) {
	testCases := []struct {
		name     string
		expected any
		request  applyCommandArgs
	}{
		{
			name:     "Only Key",
			expected: "0002",
			request:  applyCommandArgs{connKey: "test"},
		},
		{
			name:     "Only Value",
			expected: nil,
			request:  applyCommandArgs{connString: "test"},
		},
		{
			name:     "Empty Key Value",
			expected: "0001",
			request:  applyCommandArgs{},
		},
		{
			name:     "Only From",
			expected: nil,
			request:  applyCommandArgs{fromTag: "000", connString: "test"},
		},
		{
			name:     "Only To",
			expected: nil,
			request:  applyCommandArgs{toTag: "000", connString: "test"},
		},
		{
			name:     "From To",
			expected: nil,
			request:  applyCommandArgs{fromTag: "000", toTag: "001", connString: "test"},
		},
		{
			name:     "Cherry Pick",
			expected: nil,
			request:  applyCommandArgs{cherryPickedVersions: []string{"001"}, connString: "test"},
		},
		{
			name:     "Cherry Pick and To",
			expected: "0003",
			request:  applyCommandArgs{cherryPickedVersions: []string{"000"}, toTag: "001", connString: "test"},
		},
		{
			name:     "Cherry Pick and From",
			expected: "0003",
			request:  applyCommandArgs{cherryPickedVersions: []string{"000"}, fromTag: "001", connString: "test"},
		},
		{
			name:     "Cherry Pick and From and To",
			expected: "0003",
			request:  applyCommandArgs{cherryPickedVersions: []string{"000"}, fromTag: "001", toTag: "003", connString: "test"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			applyRequest = tc.request

			err := parseApplyCommand()
			if err == nil {
				if err == tc.expected {
					return
				}
				t.Fatalf("expected %v recieved nil", tc.expected)
			}

			var actual *er.SchemerErr
			if errors.As(err, &actual) {
				if actual.Code != tc.expected {
					t.Fatalf("expected %v recieved %v", tc.expected, actual)
				}
				return
			}

			t.Fatalf("recieved unexpected error: %v", err)
		})
	}
}
