package auth

import "testing"

func TestValidPassword(t *testing.T) {
	cases := []struct {
		password string
		want     bool
	}{
		{"", false},
		{"short1a", false},                        // < 12
		{"onlylettershere", false},                // no digit
		{"123456789012345", false},                // no letter
		{"goodpassword42x", true},                 // ok
		{"Пароль1234567", true},                   // unicode letters count
		{string(make([]byte, 130)) + "a1", false}, // > 128
	}
	for _, c := range cases {
		if got := validPassword(c.password); got != c.want {
			t.Errorf("validPassword(%q) = %v, want %v", c.password, got, c.want)
		}
	}
}

func TestUsernamePattern(t *testing.T) {
	valid := []string{"abc", "user_42", "ABCdef123", "a_b_c_d_e_f_g_h_i_j_k_l_m_n_o_pq"}
	invalid := []string{"", "ab", "with space", "юникод", "dash-name", "dot.name", "waaaaaaaaaaaaaaaaaaaaaaaaaaaaytoolong"}

	for _, u := range valid {
		if !usernamePattern.MatchString(u) {
			t.Errorf("expected %q to be valid", u)
		}
	}
	for _, u := range invalid {
		if usernamePattern.MatchString(u) {
			t.Errorf("expected %q to be invalid", u)
		}
	}
}
