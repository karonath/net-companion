package configdiff

import "testing"

func ops(lines []DiffLine) string {
	s := ""
	for _, l := range lines {
		switch l.Op {
		case "same":
			s += "="
		case "add":
			s += "+"
		case "del":
			s += "-"
		}
	}
	return s
}

func TestDiffIdentical(t *testing.T) {
	lines := Diff("a\nb\nc", "a\nb\nc")
	if ops(lines) != "===" {
		t.Fatalf("ops = %q, want ===", ops(lines))
	}
}

func TestDiffAddedLine(t *testing.T) {
	lines := Diff("a\nc", "a\nb\nc")
	if ops(lines) != "=+=" {
		t.Fatalf("ops = %q, want =+=", ops(lines))
	}
	if lines[1].Op != "add" || lines[1].Text != "b" {
		t.Fatalf("ligne ajoutée = %+v", lines[1])
	}
}

func TestDiffRemovedLine(t *testing.T) {
	lines := Diff("a\nb\nc", "a\nc")
	if ops(lines) != "=-=" {
		t.Fatalf("ops = %q, want =-=", ops(lines))
	}
	if lines[1].Op != "del" || lines[1].Text != "b" {
		t.Fatalf("ligne supprimée = %+v", lines[1])
	}
}

func TestDiffMixed(t *testing.T) {
	lines := Diff("hostname OLD\nip route 1\ninterface Gi0", "hostname NEW\nip route 1\ninterface Gi0")
	// première ligne changée = del + add ; les deux suivantes identiques
	if ops(lines) != "-+==" {
		t.Fatalf("ops = %q, want -+==", ops(lines))
	}
}
