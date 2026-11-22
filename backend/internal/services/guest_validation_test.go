package services

import "testing"

func TestValidateContact(t *testing.T) {
	s := func(v string) *string { return &v }
	for _, tc := range []struct {
		phone, email *string
		errs         int
	}{
		{nil, nil, 0},
		{s("090-1234-5678"), s("a.b+c@example.test"), 0},
		{s("+81 (90) 1234 5678"), nil, 0},
		{s("０９０ー１２３４"), nil, 0},
		{s("1"), nil, 0},
		{s("abc"), nil, 1},
		{s("+"), nil, 1},
		{s("090;DROP"), nil, 1},
		{nil, s("plain"), 1},
		{nil, s("a@b"), 1},
		{nil, s("a b@example.test"), 1},
		{nil, s("Name <a@example.test>"), 1},
		{nil, s("a@example.test,b@example.test"), 1},
		{s("x"), s("y"), 2},
	} {
		if got := validateContact("body", tc.phone, tc.email); len(got) != tc.errs {
			t.Errorf("phone=%q email=%q: %d errors %+v, want %d", deref(tc.phone), deref(tc.email), len(got), got, tc.errs)
		}
	}
}

func TestGuestEdits(t *testing.T) {
	s := func(v string) *string { return &v }
	updates := map[string]any{}
	changed, errs := guestEdits("body", s("  New Name "), nil, s(""), s(" 090 "), nil, updates)
	if len(errs) != 0 {
		t.Fatalf("errs %+v", errs)
	}
	if len(changed) != 3 || updates["full_name"] != "New Name" || updates["name_kana"].(*string) != nil || *updates["phone"].(*string) != "090" {
		t.Fatalf("changed=%v updates=%v", changed, updates)
	}
	_, errs = guestEdits("body.guests[0]", s(" "), s(""), nil, nil, s("bad"), map[string]any{})
	if len(errs) != 3 || errs[0].Location != "body.guests[0].full_name" || errs[2].Location != "body.guests[0].email" {
		t.Fatalf("errs %+v", errs)
	}
}

// Owner decision 2026-11-22: phone and email for the representative only;
// name_kana stays optional.
func TestValidateGuestRepresentativeContact(t *testing.T) {
	s := func(v string) *string { return &v }
	locs := func(g guestFields) []string {
		var out []string
		for _, e := range validateGuest("g", &g) {
			out = append(out, e.Location)
		}
		return out
	}
	base := guestFields{FullName: "Test Guest", Address: "Test Town", HasJapanAddress: true}

	if got := locs(base); len(got) != 0 {
		t.Fatalf("companion without kana or contact: %v", got)
	}
	rep := base
	rep.IsRepresentative = true
	rep.Phone = s(" ")
	if got := locs(rep); len(got) != 2 || got[0] != "g.phone" || got[1] != "g.email" {
		t.Fatalf("representative without contact: %v", got)
	}
	rep.Phone, rep.Email = s("000-0000-0001"), s("rep@example.test")
	if got := locs(rep); len(got) != 0 {
		t.Fatalf("representative with contact: %v", got)
	}
}
