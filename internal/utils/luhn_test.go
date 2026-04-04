package utils

import "testing"

func TestCheckLuhn(t *testing.T) {
	tests := []struct {
		name     string
		number   string
		expected bool
	}{
		{
			name:     "valid_card",
			number:   "4532015112830366",
			expected: true,
		},
		{
			name:     "invalid_card",
			number:   "4532015112830367",
			expected: false,
		},
		{
			name:     "empty_string",
			number:   "",
			expected: false,
		},
		{
			name:     "single_valid",
			number:   "0",
			expected: true,
		},
		{
			name:     "single_invalid",
			number:   "1",
			expected: false,
		},
		{
			name:     "non-digit_characters",
			number:   "1234x",
			expected: false,
		},
		{
			name:     "alphabetic_input",
			number:   "abcd",
			expected: false,
		},
		{
			name:     "special_characters",
			number:   "1234!",
			expected: false,
		},
		{
			name:     "whitespace",
			number:   "1234 5678",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CheckLuhn(tt.number)
			if result != tt.expected {
				t.Errorf("CheckLuhn(%s) = %v; expected %v", tt.number, result, tt.expected)
			}
		})
	}
}
