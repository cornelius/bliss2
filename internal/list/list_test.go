package list

import (
	"reflect"
	"testing"
)

func TestParse_empty(t *testing.T) {
	l, err := Parse("")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(l.Sections) != 1 {
		t.Errorf("sections = %d, want 1", len(l.Sections))
	}
	if len(l.Sections[0].Items) != 0 {
		t.Errorf("items = %v, want empty", l.Sections[0].Items)
	}
}

func TestParse_simpleList(t *testing.T) {
	content := "aaa-111\nbbb-222\nccc-333\n"
	l, err := Parse(content)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(l.Sections) != 1 {
		t.Fatalf("sections = %d, want 1", len(l.Sections))
	}
	want := []string{"aaa-111", "bbb-222", "ccc-333"}
	if !reflect.DeepEqual(l.Sections[0].Items, want) {
		t.Errorf("items = %v, want %v", l.Sections[0].Items, want)
	}
}

func TestParse_withSections(t *testing.T) {
	content := "aaa-111\n---\nbbb-222\n--- urgent\nccc-333\n"
	l, err := Parse(content)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(l.Sections) != 3 {
		t.Fatalf("sections = %d, want 3", len(l.Sections))
	}
	if l.Sections[0].Name != "" {
		t.Errorf("section[0].Name = %q, want empty", l.Sections[0].Name)
	}
	if !reflect.DeepEqual(l.Sections[0].Items, []string{"aaa-111"}) {
		t.Errorf("section[0].Items = %v", l.Sections[0].Items)
	}
	if l.Sections[1].Name != "" {
		t.Errorf("section[1].Name = %q, want empty", l.Sections[1].Name)
	}
	if !reflect.DeepEqual(l.Sections[1].Items, []string{"bbb-222"}) {
		t.Errorf("section[1].Items = %v", l.Sections[1].Items)
	}
	if l.Sections[2].Name != "urgent" {
		t.Errorf("section[2].Name = %q, want %q", l.Sections[2].Name, "urgent")
	}
	if !reflect.DeepEqual(l.Sections[2].Items, []string{"ccc-333"}) {
		t.Errorf("section[2].Items = %v", l.Sections[2].Items)
	}
}

func TestAdd_append(t *testing.T) {
	l := List{Sections: []Section{{Items: []string{"aaa-111"}}}}
	Add(&l, "bbb-222", false)
	want := []string{"aaa-111", "bbb-222"}
	if !reflect.DeepEqual(l.Sections[0].Items, want) {
		t.Errorf("items = %v, want %v", l.Sections[0].Items, want)
	}
}

func TestAdd_urgent(t *testing.T) {
	l := List{Sections: []Section{{Items: []string{"aaa-111", "bbb-222"}}}}
	Add(&l, "ccc-333", true)
	want := []string{"ccc-333", "aaa-111", "bbb-222"}
	if !reflect.DeepEqual(l.Sections[0].Items, want) {
		t.Errorf("items = %v, want %v", l.Sections[0].Items, want)
	}
}

func TestRemove(t *testing.T) {
	l := List{Sections: []Section{
		{Items: []string{"aaa-111", "bbb-222"}},
		{Items: []string{"bbb-222", "ccc-333"}},
	}}
	Remove(&l, "bbb-222")
	if !reflect.DeepEqual(l.Sections[0].Items, []string{"aaa-111"}) {
		t.Errorf("section[0].Items = %v", l.Sections[0].Items)
	}
	if !reflect.DeepEqual(l.Sections[1].Items, []string{"ccc-333"}) {
		t.Errorf("section[1].Items = %v", l.Sections[1].Items)
	}
}

func TestContains(t *testing.T) {
	l := List{Sections: []Section{
		{Items: []string{"aaa-111", "bbb-222"}},
		{Items: []string{"ccc-333"}},
	}}
	if !Contains(l, "aaa-111") {
		t.Error("expected Contains to return true for aaa-111")
	}
	if !Contains(l, "ccc-333") {
		t.Error("expected Contains to return true for ccc-333")
	}
	if Contains(l, "ddd-444") {
		t.Error("expected Contains to return false for ddd-444")
	}
}

func TestAllUUIDs(t *testing.T) {
	l := List{Sections: []Section{
		{Items: []string{"aaa-111", "bbb-222"}},
		{Items: []string{"ccc-333"}},
		{Items: []string{"ddd-444", "eee-555"}},
	}}
	want := []string{"aaa-111", "bbb-222", "ccc-333", "ddd-444", "eee-555"}
	got := AllUUIDs(l)
	if !reflect.DeepEqual(got, want) {
		t.Errorf("AllUUIDs = %v, want %v", got, want)
	}
}
