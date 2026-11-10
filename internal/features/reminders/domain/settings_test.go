package domain

import "testing"

func TestSettingsValidation(t *testing.T) {
	for _, tc := range []struct {
		v     []int
		valid bool
	}{
		{[]int{60, 0}, true}, {[]int{60, 30, 15, 5, 0}, true}, {[]int{}, true}, {[]int{0, 1, 2, 3, 4, 5}, false}, {[]int{-1}, false}, {[]int{15, 15}, false}, {[]int{525601}, false},
	} {
		in := UpdateReminderSettingsInput{TaskOffsets: &tc.v}
		if (in.Validate() == nil) != tc.valid {
			t.Fatalf("invalid validation for %v", tc.v)
		}
	}
}
