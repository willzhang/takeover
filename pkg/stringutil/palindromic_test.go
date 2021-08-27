/*
 * Copyright 2020 IBM Corporation
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package stringutil

import "testing"

func TestIsPalindromic(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{"", true},
		{"hello", false},
		{"racecar", true},
	}
	for _, c := range cases {
		got := IsPalindromic(c.in)
		if got != c.want {
			t.Errorf("IsPalindromic(%s) == %t, want %t", c.in, got, c.want)
		}
	}
}
